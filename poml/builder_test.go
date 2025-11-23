package poml

import (
	"encoding/xml"
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
