package poml

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"strings"
	"testing"
)

const sample = `<poml>
  <meta>
    <id>sample.demo</id>
    <version>1.0.0</version>
    <owner>tester</owner>
  </meta>
  <role>Demo role text</role>
  <task>
    First task body
  </task>
  <task><![CDATA[
    Second task with CDATA and <tags/>
  ]]></task>
  <input name="status" required="true">
    Provide status details
  </input>
  <input name="note" required="false"><![CDATA[Optional note]]></input>
  <document src="file://docs/example.org" />
  <style>
    <output format="markdown"><![CDATA[# Title]]></output>
  </style>
  <extra foo="bar"><![CDATA[extra payload]]></extra>
</poml>`

func TestParseSampleAndWalkOrder(t *testing.T) {
	doc, err := ParseString(sample)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if doc.Meta.ID != "sample.demo" || doc.Meta.Owner != "tester" {
		t.Fatalf("meta mismatch: %+v", doc.Meta)
	}
	if doc.RoleText() != "Demo role text" {
		t.Fatalf("role text mismatch: %q", doc.RoleText())
	}
	tasks := doc.TaskBodies()
	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(tasks))
	}
	if !strings.Contains(tasks[1], "CDATA") {
		t.Fatalf("second task not preserved: %q", tasks[1])
	}
	if len(doc.Inputs) != 2 || doc.Inputs[0].Name != "status" || !doc.Inputs[0].Required {
		t.Fatalf("inputs mismatch: %+v", doc.Inputs)
	}
	if len(doc.Documents) != 1 || doc.Documents[0].Src == "" {
		t.Fatalf("doc refs missing")
	}
	if len(doc.Styles) != 1 || len(doc.Styles[0].Outputs) != 1 || doc.Styles[0].Outputs[0].Format != "markdown" {
		t.Fatalf("styles mismatch: %+v", doc.Styles)
	}

	var seen []ElementType
	if err := doc.Walk(func(el Element, _ ElementPayload) error {
		seen = append(seen, el.Type)
		return nil
	}); err != nil {
		t.Fatalf("walk: %v", err)
	}
	want := []ElementType{ElementMeta, ElementRole, ElementTask, ElementTask, ElementInput, ElementInput, ElementDocument, ElementStyle, ElementUnknown}
	if len(seen) != len(want) {
		t.Fatalf("walk count mismatch: got %v want %v", seen, want)
	}
	for i := range want {
		if seen[i] != want[i] {
			t.Fatalf("order mismatch at %d: got %s want %s", i, seen[i], want[i])
		}
	}
}

func TestRoundTripPreservesOrderAndUnknown(t *testing.T) {
	doc, err := ParseString(sample)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	var buf bytes.Buffer
	if err := doc.Encode(&buf); err != nil {
		t.Fatalf("encode: %v", err)
	}
	again, err := ParseString(buf.String())
	if err != nil {
		t.Fatalf("parse roundtrip: %v", err)
	}
	if again.Meta.ID != doc.Meta.ID || again.RoleText() != doc.RoleText() || len(again.Tasks) != len(doc.Tasks) {
		t.Fatalf("roundtrip mismatch: original %+v, again %+v", doc, again)
	}
	// Unknown element should remain present.
	var unknownCount int
	for _, el := range again.Elements {
		if el.Type == ElementUnknown {
			unknownCount++
		}
	}
	if unknownCount != 1 {
		t.Fatalf("expected unknown element preserved, got %d", unknownCount)
	}
}

func TestAttrsPreserved(t *testing.T) {
	src := `<poml>
  <task foo="bar">body</task>
  <input name="a" required="true" extra="y">x</input>
  <document src="file://x" other="z"/>
</poml>`
	doc, err := ParseString(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got := len(doc.Tasks[0].Attrs); got == 0 {
		t.Fatalf("task attrs not preserved")
	}
	if got := len(doc.Inputs[0].Attrs); got == 0 {
		t.Fatalf("input attrs not preserved")
	}
	if got := len(doc.Documents[0].Attrs); got == 0 {
		t.Fatalf("document attrs not preserved")
	}
	var buf bytes.Buffer
	if err := doc.Encode(&buf); err != nil {
		t.Fatalf("encode: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, `foo="bar"`) || !strings.Contains(out, `extra="y"`) {
		t.Fatalf("attributes not in output: %s", out)
	}
}

func TestPreserveWhitespaceLeading(t *testing.T) {
	src := "<poml>\n  <task>one</task>\n    \n\t<task>two</task>\n</poml>"
	doc, err := ParseString(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(doc.Elements) < 2 || !strings.Contains(doc.Elements[1].Leading, "\n") {
		t.Fatalf("leading whitespace not captured: %+v", doc.Elements)
	}
	var buf bytes.Buffer
	if err := doc.EncodeWithOptions(&buf, EncodeOptions{IncludeHeader: false, PreserveOrder: true, PreserveWS: true}); err != nil {
		t.Fatalf("encode: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "one</task>\n    \n\t<task>two") {
		t.Fatalf("whitespace not preserved, got:\n%s", out)
	}
}

func TestPreserveTrailingWhitespace(t *testing.T) {
	src := "<poml><task>one</task>\n  <!-- trailing comment -->\n</poml>"
	doc, err := ParseString(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(doc.Elements) == 0 || strings.TrimSpace(doc.Elements[len(doc.Elements)-1].Trailing) == "" {
		t.Fatalf("trailing whitespace/comments not captured: %+v", doc.Elements)
	}

	var buf bytes.Buffer
	if err := doc.EncodeWithOptions(&buf, EncodeOptions{IncludeHeader: false, PreserveOrder: true, PreserveWS: true}); err != nil {
		t.Fatalf("encode: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "</task>\n  <!-- trailing comment -->") {
		t.Fatalf("trailing whitespace/comments not preserved:\n%s", out)
	}
}

func TestParseOptionsDisableWhitespace(t *testing.T) {
	src := "<poml><task>one</task><!-- gap --><task>two</task></poml>"
	doc, err := ParseReaderWithOptions(strings.NewReader(src), ParseOptions{PreserveWhitespace: false})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	for _, el := range doc.Elements {
		if el.Leading != "" || el.Trailing != "" {
			t.Fatalf("expected whitespace fields empty when disabled: %+v", el)
		}
	}
	var buf bytes.Buffer
	if err := doc.EncodeWithOptions(&buf, EncodeOptions{IncludeHeader: false, PreserveOrder: true, PreserveWS: true, Compact: true}); err != nil {
		t.Fatalf("encode: %v", err)
	}
	if strings.Contains(buf.String(), "gap") {
		t.Fatalf("whitespace/comment should not be preserved when disabled: %s", buf.String())
	}
}

func TestParseOptionsValidate(t *testing.T) {
	src := "<poml><task>one</task></poml>"
	if _, err := ParseReaderWithOptions(strings.NewReader(src), ParseOptions{Validate: true}); err == nil {
		t.Fatalf("expected validation error when Validate=true")
	} else {
		var pe *POMLError
		if !errors.As(err, &pe) || pe.Type != ErrValidate {
			t.Fatalf("expected POMLError validation, got %v", err)
		}
	}
	doc, err := ParseReaderWithOptions(strings.NewReader(src), ParseOptions{Validate: false})
	if err != nil {
		t.Fatalf("parse without validation: %v", err)
	}
	if len(doc.Tasks) != 1 {
		t.Fatalf("expected task parsed even when validation disabled, got %d", len(doc.Tasks))
	}
}

func TestParseStrictHelpers(t *testing.T) {
	src := "<poml><task>one</task></poml>"
	if _, err := ParseStringStrict(src); err == nil {
		t.Fatalf("expected strict parse to fail validation")
	}
	valid := `<poml><meta><id>x</id><version>1</version><owner>me</owner></meta><role>r</role><task>t</task></poml>`
	if _, err := ParseStringStrict(valid); err != nil {
		t.Fatalf("strict parse should succeed: %v", err)
	}
}

func TestParseOptionsValidateWithInvalidDiagramAndUnknownTags(t *testing.T) {
	src := `<poml>
  <diagram id="bad">
    <graph>
      <node label="no id"/>
      <edge from="" to="" directed="true"/>
    </graph>
  </diagram>
  <unknown foo="bar">keep me</unknown>
</poml>`
	if _, err := ParseReaderWithOptions(strings.NewReader(src), ParseOptions{Validate: true}); err == nil {
		t.Fatalf("expected diagram validation error")
	}
	doc, err := ParseReaderWithOptions(strings.NewReader(src), ParseOptions{Validate: false})
	if err != nil {
		t.Fatalf("parse without validation: %v", err)
	}
	if len(doc.Diagrams) != 1 || len(doc.Elements) == 0 {
		t.Fatalf("expected diagram and elements")
	}
	var foundUnknown bool
	for _, el := range doc.Elements {
		if el.Type == ElementUnknown {
			foundUnknown = true
			break
		}
	}
	if !foundUnknown {
		t.Fatalf("expected unknown element preserved")
	}
}

func TestPOMLErrorUnwrap(t *testing.T) {
	inner := errors.New("root")
	err := &POMLError{Type: ErrDecode, Message: "wrap", Err: inner}
	if !errors.Is(err, inner) {
		t.Fatalf("expected unwrap to expose inner error")
	}
}

func TestWalkInputsHelper(t *testing.T) {
	doc, err := ParseString(`<poml><input name="a" required="true">x</input><input name="b" required="false">y</input></poml>`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	var seen []string
	doc.WalkInputs(func(in *Input) {
		seen = append(seen, in.Name)
	})
	if len(seen) != 2 || seen[0] != "a" || seen[1] != "b" {
		t.Fatalf("walk inputs mismatch: %v", seen)
	}
}

func TestMutatorMarkModifiedAndInsertTaskBefore(t *testing.T) {
	doc, err := ParseString("<poml><task>t1</task><task>t2</task></poml>")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	err = doc.Mutate(func(el Element, payload ElementPayload, m *Mutator) error {
		if el.Type == ElementTask && el.Index == 0 {
			payload.Task.Body = "updated"
			m.MarkModified()
			m.InsertTaskAfter(el, "inserted")
		}
		if el.Type == ElementTask && el.Index == 1 {
			// insert unknown element before to exercise InsertBefore
			m.InsertBefore(el, Element{Type: ElementUnknown, RawXML: "<extra/>"})
		}
		return nil
	})
	if err != nil {
		t.Fatalf("mutate: %v", err)
	}
	if doc.Tasks[0].Body != "updated" {
		t.Fatalf("mark modified did not persist change")
	}
	if len(doc.Tasks) != 3 {
		t.Fatalf("expected inserted task, got %d", len(doc.Tasks))
	}
	if !containsRaw(doc.Elements, "<extra/>") {
		t.Fatalf("expected inserted unknown element")
	}
}

func containsRaw(elems []Element, raw string) bool {
	for _, el := range elems {
		if el.RawXML == raw {
			return true
		}
	}
	return false
}

func TestParseReaderAndEncodeAllElements(t *testing.T) {
	directed := true
	doc := Document{
		Meta:      Meta{ID: "full.demo", Version: "1", Owner: "me"},
		Role:      Block{Body: "role"},
		Tasks:     []Block{{Body: "t1"}},
		Inputs:    []Input{{Name: "input", Required: true, Body: "body"}},
		Documents: []DocRef{{Src: "file://x"}},
		Styles:    []Style{{Outputs: []Output{{Format: "markdown", Body: "# hi"}}}},
		Messages:  []Message{{Role: "human", Body: "hi"}},
		ToolDefs:  []ToolDefinition{{Name: "calc", Body: `{"type":"object"}`, Attrs: []xml.Attr{{Name: xml.Name{Local: "x"}, Value: "1"}}}},
		ToolReqs:  []ToolRequest{{ID: "call_1", Name: "calc", Parameters: "{{ {\"x\":1} }}"}},
		ToolResps: []ToolResponse{{ID: "call_1", Name: "calc", Body: "2"}},
		Schema:    OutputSchema{Body: `{"type":"object"}`},
		Runtimes:  []Runtime{{Attrs: []xml.Attr{{Name: xml.Name{Local: "temperature"}, Value: "0.1"}}}},
		Images:    []Image{{Src: "data:image/png;base64,AA==", Alt: "a", Syntax: "image/png"}},
		Diagrams: []Diagram{{
			ID: "d1",
			Graph: DiagramGraph{
				Nodes: []DiagramNode{{ID: "n1", X: "0", Y: "0", Z: "0"}},
				Edges: []DiagramEdge{{From: "n1", To: "n1", Directed: &directed}},
			},
		}},
	}
	doc.Elements = doc.defaultElements()
	var buf bytes.Buffer
	if err := doc.Encode(&buf); err != nil {
		t.Fatalf("encode all: %v", err)
	}
	if _, err := ParseReader(strings.NewReader(buf.String())); err != nil {
		t.Fatalf("ParseReader: %v", err)
	}
}

func TestMutatorRemoveAndReplaceBodyBranches(t *testing.T) {
	doc := Document{}
	doc.Meta = Meta{ID: "r", Version: "1", Owner: "o"}
	doc.AddRole("role")
	doc.AddTask("t1")
	doc.AddInput("name", true, "body")
	doc.AddDocument("file://d")
	doc.AddStyle(Output{Format: "markdown", Body: "orig"})
	doc.AddMessage("assistant", "msg")
	doc.AddToolDefinition("calc", "{}")
	doc.AddToolRequest("id", "calc", "{{ {\"x\":1} }}")
	doc.AddToolResponse("id", "calc", "2")
	doc.AddOutputSchema(`{"type":"object"}`)
	doc.AddRuntime(xml.Attr{Name: xml.Name{Local: "temp"}, Value: "0.2"})
	doc.AddImage(ImageFromBytes([]byte{0x01}, "image/png", "a"))

	err := doc.Mutate(func(el Element, payload ElementPayload, m *Mutator) error {
		switch el.Type {
		case ElementStyle:
			m.ReplaceBody(el, "updated-style")
		case ElementToolResponse:
			m.ReplaceBody(el, "updated-tool")
		case ElementImage:
			m.ReplaceBody(el, "img-body")
		case ElementRole:
			m.Remove(el)
		case ElementRuntime:
			m.Remove(el)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("mutate: %v", err)
	}
	if doc.Styles[0].Outputs[0].Body != "updated-style" {
		t.Fatalf("replace style failed")
	}
	if doc.ToolResps[0].Body != "updated-tool" {
		t.Fatalf("replace tool response failed")
	}
	if doc.Images[0].Body != "img-body" {
		t.Fatalf("replace image failed")
	}
	if doc.Role.Body != "" {
		t.Fatalf("role should be removed")
	}
	if len(doc.Runtimes) != 0 {
		t.Fatalf("runtime should be removed")
	}
}

func TestWrapXMLError(t *testing.T) {
	syn := &xml.SyntaxError{Line: 3}
	err := wrapXMLError(syn, "ctx")
	var pe *POMLError
	if !errors.As(err, &pe) || pe.Type != ErrDecode {
		t.Fatalf("wrapXMLError should wrap syntax errors, got %v", err)
	}
}

func TestElementByIDLookup(t *testing.T) {
	doc, err := ParseString(sample)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(doc.Elements) == 0 {
		t.Fatalf("expected populated elements")
	}
	el, payload, ok := doc.ElementByID(doc.Elements[0].ID)
	if !ok {
		t.Fatalf("expected element lookup by ID to succeed")
	}
	if el.Type != ElementMeta || payload.Meta == nil || payload.Meta.ID == "" {
		t.Fatalf("unexpected payload returned: %+v %+v", el, payload.Meta)
	}
}

func TestMalformedReportsError(t *testing.T) {
	// missing closing tag, malformed attribute
	bad := `<poml><meta><id>bad</id></meta><input name="x" required nope></input></poml>`
	if _, err := ParseString(bad); err == nil {
		t.Fatalf("expected parse error for malformed input")
	} else if !strings.Contains(err.Error(), "line") {
		t.Fatalf("error should include location information, got: %v", err)
	}
}

func TestLargeDocumentParses(t *testing.T) {
	var b strings.Builder
	b.WriteString(`<poml><meta><id>large</id><version>1</version><owner>me</owner></meta>`)
	for i := 0; i < 1000; i++ {
		fmt.Fprintf(&b, "<task>Task %d body</task>", i)
	}
	b.WriteString(`</poml>`)
	doc, err := ParseString(b.String())
	if err != nil {
		t.Fatalf("parse large: %v", err)
	}
	if len(doc.Tasks) != 1000 {
		t.Fatalf("expected 1000 tasks, got %d", len(doc.Tasks))
	}
}

func TestBuilderAndValidation(t *testing.T) {
	var doc Document
	doc.Meta = Meta{ID: "builder.demo", Version: "0.0.1", Owner: "tester"}
	doc.AddRole("builder role")
	doc.AddTask("t1")
	doc.AddTask("t2")
	doc.AddInput("a", true, "body")
	doc.AddInput("b", false, "")
	doc.AddDocument("file://x")
	doc.AddStyle(Output{Format: "markdown", Body: "# hi"})

	if err := doc.Validate(); err != nil {
		t.Fatalf("validate failed: %v", err)
	}

	// Duplicate input names should trigger validation.
	docDup := doc
	docDup.Inputs[1].Name = "a"
	if err := docDup.Validate(); err == nil {
		t.Fatalf("expected validation error for duplicate input name")
	}
}

func TestValidateToolEvents(t *testing.T) {
	makeDoc := func() Document {
		return Document{
			Meta:  Meta{ID: "tool.demo", Version: "1", Owner: "me"},
			Role:  Block{Body: "role"},
			Tasks: []Block{{Body: "task"}},
		}
	}
	tests := []struct {
		name     string
		prepare  func(d *Document)
		wantErr  bool
		wantText string
	}{
		{
			name: "valid tool flow",
			prepare: func(d *Document) {
				d.ToolDefs = []ToolDefinition{{Name: "calc", Body: "{}"}}
				d.ToolReqs = []ToolRequest{{ID: "call_1", Name: "calc", Parameters: "{}"}}
				d.ToolResps = []ToolResponse{{ID: "call_1", Name: "calc", Body: "ok"}}
				d.ToolResults = []ToolResult{{ID: "call_1", Name: "calc", Body: "result"}}
				d.ToolErrors = []ToolError{{ID: "call_1", Name: "calc", Body: "err"}}
			},
		},
		{
			name: "missing tool request",
			prepare: func(d *Document) {
				d.ToolDefs = []ToolDefinition{{Name: "calc", Body: "{}"}}
				d.ToolResps = []ToolResponse{{ID: "call_1", Name: "calc", Body: "ok"}}
			},
			wantErr:  true,
			wantText: "tool-response id",
		},
		{
			name: "unknown tool definition in response",
			prepare: func(d *Document) {
				d.ToolReqs = []ToolRequest{{ID: "call_1", Name: "calc", Parameters: "{}"}}
				d.ToolResps = []ToolResponse{{ID: "call_1", Name: "calc", Body: "ok"}}
			},
			wantErr:  true,
			wantText: "unknown tool-definition",
		},
		{
			name: "mismatched tool name for id",
			prepare: func(d *Document) {
				d.ToolDefs = []ToolDefinition{{Name: "calc", Body: "{}"}}
				d.ToolReqs = []ToolRequest{{ID: "call_1", Name: "calc", Parameters: "{}"}}
				d.ToolResps = []ToolResponse{{ID: "call_1", Name: "other", Body: "ok"}}
			},
			wantErr:  true,
			wantText: "uses tool",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := makeDoc()
			tt.prepare(&doc)
			err := doc.Validate()
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected validation error")
				}
				var pe *POMLError
				if !errors.As(err, &pe) || pe.Type != ErrValidate {
					t.Fatalf("expected POMLError validate, got %v", err)
				}
				if tt.wantText != "" && !strings.Contains(err.Error(), tt.wantText) {
					t.Fatalf("expected error to mention %q, got %v", tt.wantText, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected validation error: %v", err)
			}
		})
	}
}

func TestToolAliasParsesAndEncodes(t *testing.T) {
	src := `<poml><tool name="calc" description="add"><![CDATA[{"type":"object"}]]></tool></poml>`
	doc, err := ParseString(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(doc.ToolDefs) != 1 || doc.ToolDefs[0].Name != "calc" {
		t.Fatalf("tool alias not parsed: %+v", doc.ToolDefs)
	}
	var buf bytes.Buffer
	if err := doc.EncodeWithOptions(&buf, EncodeOptions{IncludeHeader: false, PreserveOrder: true, PreserveWS: true}); err != nil {
		t.Fatalf("encode: %v", err)
	}
	if !strings.Contains(buf.String(), "<tool name=\"calc\"") {
		t.Fatalf("expected tool tag preserved, got: %s", buf.String())
	}
}

func TestOutputFormatAndDocumentAlias(t *testing.T) {
	src := `<poml><output-format>plain text</output-format><Document src="file://x.doc"/></poml>`
	doc, err := ParseString(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(doc.OutFormats) != 1 || !strings.Contains(doc.OutFormats[0].Body, "plain") {
		t.Fatalf("output-format not parsed: %+v", doc.OutFormats)
	}
	if len(doc.Documents) != 1 || doc.Documents[0].Src != "file://x.doc" {
		t.Fatalf("document alias not parsed: %+v", doc.Documents)
	}
	var buf bytes.Buffer
	if err := doc.EncodeWithOptions(&buf, EncodeOptions{IncludeHeader: false, PreserveOrder: true, PreserveWS: true}); err != nil {
		t.Fatalf("encode: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "<output-format>plain text</output-format>") {
		t.Fatalf("output-format not preserved, got: %s", out)
	}
	if !strings.Contains(out, "<Document src=\"file://x.doc\"></Document>") {
		t.Fatalf("Document alias not preserved, got: %s", out)
	}
}

func TestHintExampleContentPartObjectRoundTrip(t *testing.T) {
	src := `<poml>
  <hint caption="Background"><![CDATA[<p>context</p>]]></hint>
  <example id="ex1"><input>foo</input></example>
  <cp caption="Doc"><object data="{{ foo }}" syntax="xml"/></cp>
  <audio src="data:audio/mpeg;base64,QQ==" alt="clip"/>
  <video src="data:video/mp4;base64,QQ==" alt="vid"/>
</poml>`
	doc, err := ParseString(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(doc.Hints) != 1 || len(doc.Examples) != 1 || len(doc.ContentParts) != 1 || len(doc.Audios) != 1 || len(doc.Videos) != 1 {
		t.Fatalf("expected hint/example/cp/audio/video parsed, got %+v %+v %+v %d %d", doc.Hints, doc.Examples, doc.ContentParts, len(doc.Audios), len(doc.Videos))
	}
	if !strings.Contains(doc.ContentParts[0].Body, "<object") {
		t.Fatalf("expected cp body to contain object tag: %s", doc.ContentParts[0].Body)
	}
	var buf bytes.Buffer
	if err := doc.EncodeWithOptions(&buf, EncodeOptions{IncludeHeader: false, PreserveOrder: true, PreserveWS: true}); err != nil {
		t.Fatalf("encode: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "<hint") || !strings.Contains(out, "<example") || !strings.Contains(out, "<cp") || !strings.Contains(out, "<object") {
		t.Fatalf("expected hint/example/cp/object in output, got: %s", out)
	}
}

func TestValidateObjectRequiresDataOrBody(t *testing.T) {
	doc := Document{
		Meta:  Meta{ID: "v", Version: "1", Owner: "me"},
		Role:  Block{Body: "role"},
		Tasks: []Block{{Body: "task"}},
		Objects: []ObjectTag{{
			Syntax: "xml",
		}},
	}
	if err := doc.Validate(); err == nil {
		t.Fatalf("expected validation error for missing object data/body")
	}
}

func TestValidateMetaRoleTasks(t *testing.T) {
	doc := Document{}
	if err := doc.Validate(); err == nil {
		t.Fatalf("expected validation errors for missing sections")
	}
	doc.Meta = Meta{ID: "x", Version: "1", Owner: "me"}
	doc.AddRole("r")
	doc.AddTask("t")
	if err := doc.Validate(); err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
}

func TestValidateToolingReferences(t *testing.T) {
	base := Document{
		Meta:  Meta{ID: "tools.demo", Version: "1", Owner: "me"},
		Role:  Block{Body: "role"},
		Tasks: []Block{{Body: "t"}},
		ToolDefs: []ToolDefinition{
			{Name: "calc"},
			{Name: "echo"},
		},
		ToolReqs: []ToolRequest{{ID: "call_1", Name: "calc"}},
		ToolResps: []ToolResponse{
			{ID: "call_1", Name: "calc"},
		},
		ToolResults: []ToolResult{
			{ID: "call_1", Name: "calc"},
		},
		ToolErrors: []ToolError{
			{ID: "call_1", Name: "calc"},
		},
	}

	if err := base.Validate(); err != nil {
		t.Fatalf("expected valid tooling to pass validation: %v", err)
	}

	docUnknownDef := base
	docUnknownDef.ToolReqs[0].Name = "missing"
	if err := docUnknownDef.Validate(); err == nil {
		t.Fatalf("expected validation error for unknown tool-definition")
	}

	docMissingRequest := base
	docMissingRequest.ToolResps[0].ID = "call_2"
	if err := docMissingRequest.Validate(); err == nil {
		t.Fatalf("expected validation error for missing tool-request reference")
	}

	docMismatchedTool := base
	docMismatchedTool.ToolResps[0].Name = "echo"
	if err := docMismatchedTool.Validate(); err == nil {
		t.Fatalf("expected validation error for mismatched tool name on response")
	}

	docDuplicateReq := base
	docDuplicateReq.ToolReqs = append(docDuplicateReq.ToolReqs, ToolRequest{ID: "call_1", Name: "calc"})
	if err := docDuplicateReq.Validate(); err == nil {
		t.Fatalf("expected validation error for duplicate tool-request id")
	}

	docUnknownToolError := base
	docUnknownToolError.ToolErrors[0].Name = "missing"
	if err := docUnknownToolError.Validate(); err == nil {
		t.Fatalf("expected validation error for tool-error with unknown tool-definition")
	}
}

func TestDefaultElementsAdvanceNextID(t *testing.T) {
	doc := Document{
		Meta:  Meta{ID: "seed", Version: "1", Owner: "me"},
		Tasks: []Block{{Body: "t1"}},
	}
	doc.Elements = doc.defaultElements()

	seen := make(map[string]struct{})
	for _, el := range doc.Elements {
		if el.ID == "" {
			t.Fatalf("expected element IDs populated, got %+v", el)
		}
		if _, ok := seen[el.ID]; ok {
			t.Fatalf("duplicate ID after defaultElements: %s", el.ID)
		}
		seen[el.ID] = struct{}{}
	}

	doc.AddTask("t2")

	seen = make(map[string]struct{})
	for _, el := range doc.Elements {
		if _, ok := seen[el.ID]; ok {
			t.Fatalf("duplicate ID after adding task: %s", el.ID)
		}
		seen[el.ID] = struct{}{}
	}
	if doc.nextID <= len(doc.Elements) {
		t.Fatalf("nextID not advanced; got %d with %d elements", doc.nextID, len(doc.Elements))
	}
}

func TestDumpFile(t *testing.T) {
	doc := Document{
		Meta: Meta{ID: "dump.demo", Version: "1", Owner: "me"},
		Role: Block{Body: "role"},
		Tasks: []Block{
			{Body: "t1"},
		},
		Elements: []Element{
			{Type: ElementMeta},
			{Type: ElementRole},
			{Type: ElementTask, Index: 0},
		},
	}
	tmp := t.TempDir() + "/out.poml"
	if err := doc.DumpFile(tmp, EncodeOptions{IncludeHeader: false}); err != nil {
		t.Fatalf("dump: %v", err)
	}
	loaded, err := ParseFile(tmp)
	if err != nil {
		t.Fatalf("parse dumped: %v", err)
	}
	if loaded.Meta.ID != doc.Meta.ID || loaded.RoleText() != "role" {
		t.Fatalf("dump roundtrip mismatch: %+v", loaded)
	}
}

func TestMutateReplaceRemoveInsert(t *testing.T) {
	doc, err := ParseString(sample)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	err = doc.Mutate(func(el Element, payload ElementPayload, m *Mutator) error {
		switch el.Type {
		case ElementTask:
			if el.Index == 0 {
				m.ReplaceBody(el, "Updated body")
			}
		case ElementInput:
			if payload.Input != nil && payload.Input.Name == "note" {
				m.Remove(el)
			}
		case ElementDocument:
			m.InsertInputAfter(el, Input{Name: "added", Required: false, Body: "extra"})
		}
		return nil
	})
	if err != nil {
		t.Fatalf("mutate: %v", err)
	}

	// Round-trip and verify changes.
	var buf bytes.Buffer
	if err := doc.Encode(&buf); err != nil {
		t.Fatalf("encode: %v", err)
	}
	rt, err := ParseString(buf.String())
	if err != nil {
		t.Fatalf("parse after mutate: %v", err)
	}
	if got := rt.Tasks[0].Body; !strings.Contains(got, "Updated body") {
		t.Fatalf("task not replaced: %q", got)
	}
	if len(rt.Inputs) != 2 { // original 2, removed note (-1), inserted added (+1)
		t.Fatalf("unexpected input count: %d", len(rt.Inputs))
	}
	names := []string{rt.Inputs[0].Name, rt.Inputs[1].Name}
	if !contains(names, "added") {
		t.Fatalf("inserted input missing: %v", names)
	}
}

func contains(list []string, target string) bool {
	for _, v := range list {
		if v == target {
			return true
		}
	}
	return false
}

func TestParsesMessagesAndTools(t *testing.T) {
	src := `<poml>
  <human-msg>Hello</human-msg>
  <assistant-msg>Hi</assistant-msg>
  <img src="file://foo.png" alt="pic" syntax="multimedia"><![CDATA[data]]></img>
  <tool-definition name="calc">{"type":"object"}</tool-definition>
  <tool-request id="call_1" name="calc" parameters="{{ { x: 1 } }}"/>
  <tool-response id="call_1" name="calc">2</tool-response>
  <output-schema>{"type":"object"}</output-schema>
  <runtime temperature="0.5" max-tokens="10"/>
</poml>`
	doc, err := ParseString(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(doc.Messages) != 2 || doc.Messages[0].Role != "human" || doc.Messages[1].Role != "assistant" {
		t.Fatalf("messages not captured: %+v", doc.Messages)
	}
	if len(doc.ToolDefs) != 1 || doc.ToolDefs[0].Name != "calc" {
		t.Fatalf("tool-definition missing: %+v", doc.ToolDefs)
	}
	if len(doc.ToolReqs) != 1 || doc.ToolReqs[0].ID != "call_1" {
		t.Fatalf("tool-request missing: %+v", doc.ToolReqs)
	}
	if len(doc.ToolResps) != 1 || doc.ToolResps[0].Body == "" {
		t.Fatalf("tool-response missing: %+v", doc.ToolResps)
	}
	if doc.Schema.Body == "" {
		t.Fatalf("output-schema missing")
	}
	if len(doc.Runtimes) != 1 || len(doc.Runtimes[0].Attrs) == 0 {
		t.Fatalf("runtime missing: %+v", doc.Runtimes)
	}
	if len(doc.Images) != 1 || doc.Images[0].Src == "" || doc.Images[0].Body == "" {
		t.Fatalf("image missing or incomplete: %+v", doc.Images)
	}
	var types []ElementType
	for _, el := range doc.Elements {
		types = append(types, el.Type)
	}
	want := []ElementType{
		ElementHumanMsg, ElementAssistantMsg, ElementImage, ElementToolDefinition,
		ElementToolRequest, ElementToolResponse, ElementOutputSchema, ElementRuntime,
	}
	if len(types) != len(want) {
		t.Fatalf("element count mismatch: %v", types)
	}
	for i := range want {
		if types[i] != want[i] {
			t.Fatalf("element order mismatch at %d: got %s want %s", i, types[i], want[i])
		}
	}
}

func TestBuildersForMessagesToolsSchemaRuntimeImage(t *testing.T) {
	var doc Document
	doc.Meta = Meta{ID: "builder.demo", Version: "1", Owner: "me"}
	doc.AddRole("role")
	doc.AddTask("t1")
	doc.AddMessage("assistant", "hi", xml.Attr{Name: xml.Name{Local: "tone"}, Value: "brief"})
	doc.AddToolDefinition("calc", `{"type":"object"}`)
	doc.AddToolRequest("call_1", "calc", "{{ {\"x\":1} }}")
	doc.AddToolResponse("call_1", "calc", "2")
	doc.AddOutputSchema(`{"type":"object","properties":{"x":{"type":"number"}}}`)
	doc.AddRuntime(xml.Attr{Name: xml.Name{Local: "temperature"}, Value: "0.1"})
	doc.AddImage(ImageFromBytes([]byte{0x01, 0x02}, "image/png", "tiny"))

	if err := doc.Validate(); err != nil {
		t.Fatalf("validate builders doc: %v", err)
	}
	var seen []ElementType
	for _, el := range doc.Elements {
		seen = append(seen, el.Type)
	}
	if !containsType(seen, ElementAssistantMsg) || !containsType(seen, ElementToolDefinition) || !containsType(seen, ElementOutputSchema) || !containsType(seen, ElementRuntime) || !containsType(seen, ElementImage) {
		t.Fatalf("expected new element types recorded, got %v", seen)
	}
}

func TestValidationCatchesMissingNamesAndSchema(t *testing.T) {
	var doc Document
	doc.Meta = Meta{ID: "v", Version: "1", Owner: "me"}
	doc.AddRole("role")
	doc.AddTask("task")
	doc.ToolDefs = append(doc.ToolDefs, ToolDefinition{Body: "{}"})
	doc.ToolReqs = append(doc.ToolReqs, ToolRequest{})
	doc.AddOutputSchema("")
	if err := doc.Validate(); err == nil {
		t.Fatalf("expected validation errors for missing tool names and schema")
	}
}

func containsType(list []ElementType, target ElementType) bool {
	for _, v := range list {
		if v == target {
			return true
		}
	}
	return false
}

func TestMutatorInsertDocumentAndStyle(t *testing.T) {
	doc, err := ParseString(sample)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	err = doc.Mutate(func(el Element, payload ElementPayload, m *Mutator) error {
		if el.Type == ElementDocument {
			m.InsertDocumentAfter(el, "file://extra")
			m.InsertStyleAfter(el, Style{Outputs: []Output{{Format: "text", Body: "plain"}}})
		}
		return nil
	})
	if err != nil {
		t.Fatalf("mutate: %v", err)
	}
	if len(doc.Documents) != 2 {
		t.Fatalf("expected second document inserted, got %d", len(doc.Documents))
	}
	if len(doc.Styles) != 2 {
		t.Fatalf("expected second style inserted, got %d", len(doc.Styles))
	}
	// Ensure elements are reindexed in order after inserts.
	var seenDocs, seenStyles int
	for _, el := range doc.Elements {
		if el.Type == ElementDocument {
			if el.Index == 0 || el.Index == 1 {
				seenDocs++
			}
		}
		if el.Type == ElementStyle {
			if el.Index == 0 || el.Index == 1 {
				seenStyles++
			}
		}
	}
	if seenDocs != 2 || seenStyles != 2 {
		t.Fatalf("expected reindexed elements for docs/styles, got docs=%d styles=%d", seenDocs, seenStyles)
	}
}
