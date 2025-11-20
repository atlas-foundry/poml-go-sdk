package poml

import (
	"bytes"
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
