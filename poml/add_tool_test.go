package poml

import "testing"

func TestAddToolEntries(t *testing.T) {
	var doc Document
	doc.AddToolRequest("id1", "req", "{}")
	doc.AddToolResponse("id2", "resp", "body")
	doc.AddToolResult("id3", "res", "data")
	doc.AddToolError("id4", "err", "boom")
	doc.AddRuntime()

	if len(doc.ToolReqs) != 1 || len(doc.ToolResps) != 1 || len(doc.ToolResults) != 1 || len(doc.ToolErrors) != 1 {
		t.Fatalf("unexpected tool lengths")
	}
	if len(doc.Runtimes) != 1 {
		t.Fatalf("unexpected runtime length")
	}
	if len(doc.Elements) != 5 {
		t.Fatalf("expected 5 elements, got %d", len(doc.Elements))
	}
}
