package poml

import "testing"

// Covers ReplaceBody branches that were previously untested.
func TestMutatorReplaceBody(t *testing.T) {
	doc := Document{
		Role:       Block{Body: "r"},
		Tasks:      []Block{{Body: "t1"}},
		Inputs:     []Input{{Body: "in"}},
		Styles:     []Style{{Outputs: []Output{{Body: "style0"}}}},
		OutFormats: []OutputFormat{{Body: "fmt"}},
		Messages:   []Message{{Body: "m1"}},
		ToolResps:  []ToolResponse{{Body: "resp"}},
		Schema:     OutputSchema{Body: "schema"},
		Images:     []Image{{Body: "img"}},
		Elements: []Element{
			{Type: ElementRole},
			{Type: ElementTask, Index: 0},
			{Type: ElementInput, Index: 0},
			{Type: ElementStyle, Index: 0},
			{Type: ElementOutputFormat, Index: 0},
			{Type: ElementHumanMsg, Index: 0},
			{Type: ElementToolResponse, Index: 0},
			{Type: ElementOutputSchema},
			{Type: ElementImage, Index: 0},
		},
	}
	m := Mutator{doc: &doc}
	m.ReplaceBody(doc.Elements[0], "R")
	m.ReplaceBody(doc.Elements[1], "T")
	m.ReplaceBody(doc.Elements[2], "IN")
	m.ReplaceBody(doc.Elements[3], "STYLE")
	m.ReplaceBody(doc.Elements[4], "FMT")
	m.ReplaceBody(doc.Elements[5], "MSG")
	m.ReplaceBody(doc.Elements[6], "RESP")
	m.ReplaceBody(doc.Elements[7], "SCHEMA")
	m.ReplaceBody(doc.Elements[8], "IMG")

	if doc.Role.Body != "R" || doc.Tasks[0].Body != "T" || doc.Inputs[0].Body != "IN" {
		t.Fatalf("role/tasks/inputs not updated")
	}
	if doc.Styles[0].Outputs[0].Body != "STYLE" || doc.OutFormats[0].Body != "FMT" {
		t.Fatalf("style/outformat not updated")
	}
	if doc.Messages[0].Body != "MSG" || doc.ToolResps[0].Body != "RESP" {
		t.Fatalf("message/tool resp not updated")
	}
	if doc.Schema.Body != "SCHEMA" || doc.Images[0].Body != "IMG" {
		t.Fatalf("schema/image not updated")
	}
}

// Covers additional Remove branches (tool response already covered elsewhere).
func TestMutatorRemoveImagesAndSchema(t *testing.T) {
	doc := Document{
		Images:    []Image{{Body: "img1"}},
		Schema:    OutputSchema{Body: "schema"},
		ToolResps: []ToolResponse{{Body: "resp"}},
		Elements: []Element{
			{Type: ElementImage, Index: 0},
			{Type: ElementOutputSchema},
			{Type: ElementToolResponse, Index: 0},
		},
	}
	m := Mutator{doc: &doc}
	e0, e1, e2 := doc.Elements[0], doc.Elements[1], doc.Elements[2]
	m.Remove(e0)
	m.Remove(e1)
	m.Remove(e2)
	if len(doc.Images) != 0 {
		t.Fatalf("image not removed")
	}
	if doc.Schema.Body != "" {
		t.Fatalf("schema not cleared")
	}
	if len(doc.ToolResps) != 0 {
		t.Fatalf("tool resp not removed")
	}
}
