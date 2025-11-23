package poml

import (
	"encoding/xml"
	"strings"
	"testing"
)

func TestBuilderCreatesToolingAndSchema(t *testing.T) {
	b := NewBuilder().
		Meta("builder.demo", "1.0.0", "me").
		Role("role").
		Task("t").
		Human("hi").
		Assistant("calling tool").
		ToolDefinition("search", "Search for things", map[string]any{"type": "object", "properties": map[string]any{"query": map[string]any{"type": "string"}}}).
		ToolRequest("call_1", "search", map[string]any{"query": "python"}, xml.Attr{Name: xml.Name{Local: "source"}, Value: "test"}).
		ToolResponse("call_1", "search", "result").
		OutputSchema(map[string]any{"type": "object", "properties": map[string]any{"answer": map[string]any{"type": "string"}}}).
		Runtime(map[string]any{"temperature": 0.2})

	doc := b.Build()
	if err := doc.Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}

	outAny, err := Convert(doc, FormatDict, ConvertOptions{})
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	out := outAny.(dictOutput)
	if len(out.Tools) != 1 {
		t.Fatalf("expected one tool, got %d", len(out.Tools))
	}
	tool := out.Tools[0].(map[string]any)
	if tool["description"] != "Search for things" {
		t.Fatalf("description mismatch: %+v", tool)
	}
	params, ok := tool["parameters"].(map[string]any)
	if !ok || params["type"] != "object" {
		t.Fatalf("parameters not parsed: %+v", tool["parameters"])
	}
	if out.Schema == nil {
		t.Fatalf("schema missing in output")
	}
	rt := out.Runtime
	if rt["temperature"] != 0.2 && rt["temperature"] != "0.2" {
		t.Fatalf("runtime mismatch: %+v", rt)
	}
}

func TestBuilderAddsMediaAndObjects(t *testing.T) {
	diagram := Diagram{
		ID: "d1",
		Graph: DiagramGraph{
			Nodes: []DiagramNode{{ID: "n1", X: "0", Y: "0"}},
		},
	}
	doc := NewBuilder().
		Meta("builder.media", "1.0.0", "me").
		Image(Image{Alt: "img", Src: "http://example.com/image.png"}).
		Audio(Media{Src: "sound.wav", Alt: "sound"}).
		Video(Media{Src: "video.mp4", Alt: "clip"}).
		Hint("remember this").
		Example("example text").
		ContentPart("content body").
		Object("{}", "json", "<raw/>").
		Diagram(diagram).
		ToolResult("id-1", "convert", "ok").
		ToolError("id-2", "convert", "failed").
		Build()

	if len(doc.Images) != 1 || len(doc.Audios) != 1 || len(doc.Videos) != 1 {
		t.Fatalf("media not recorded: imgs=%d audios=%d videos=%d", len(doc.Images), len(doc.Audios), len(doc.Videos))
	}
	if len(doc.Hints) != 1 || len(doc.Examples) != 1 || len(doc.ContentParts) != 1 {
		t.Fatalf("content parts missing: hints=%d examples=%d cp=%d", len(doc.Hints), len(doc.Examples), len(doc.ContentParts))
	}
	if len(doc.Objects) != 1 || len(doc.Diagrams) != 1 {
		t.Fatalf("objects or diagrams missing: objs=%d diags=%d", len(doc.Objects), len(doc.Diagrams))
	}
	if len(doc.ToolResults) != 1 || len(doc.ToolErrors) != 1 {
		t.Fatalf("tool results/errors missing: results=%d errors=%d", len(doc.ToolResults), len(doc.ToolErrors))
	}
	if got := doc.Elements[len(doc.Elements)-1].Type; got != ElementToolError {
		t.Fatalf("expected last element to be tool-error, got %v", got)
	}
}

func TestBuilderRecordsOrderingForInputsAndRaw(t *testing.T) {
	doc := NewBuilder().
		Meta("builder.order", "1.0.0", "me").
		Input("name", true, "value").
		DocumentRef("doc.xml").
		Style(Output{Format: "text"}).
		OutputFormat("markdown").
		System("sys msg").
		Raw("<extra/>").
		Build()

	if len(doc.Elements) != 7 { // meta + 5 calls above + output-format
		t.Fatalf("unexpected elements len: %d", len(doc.Elements))
	}
	last := doc.Elements[len(doc.Elements)-1]
	if last.Type != ElementUnknown || strings.TrimSpace(last.RawXML) != "<extra/>" {
		t.Fatalf("raw element not preserved: %+v", last)
	}
	if len(doc.Inputs) != 1 || doc.Inputs[0].Name != "name" || !doc.Inputs[0].Required {
		t.Fatalf("input not recorded: %+v", doc.Inputs)
	}
	if len(doc.Documents) != 1 || doc.Documents[0].Src != "doc.xml" {
		t.Fatalf("document ref missing: %+v", doc.Documents)
	}
	if len(doc.Styles) != 1 || len(doc.OutFormats) != 1 || doc.Role.Body != "" {
		t.Fatalf("style/output-format not captured")
	}
}
