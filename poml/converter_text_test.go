package poml

import (
	"errors"
	"strings"
	"testing"
)

func TestConvertMarkdownToPOMLAndBack(t *testing.T) {
	doc, err := ConvertTextToPOML("# title\n\nbody", FormatMarkdown)
	if err != nil {
		t.Fatalf("convert markdown: %v", err)
	}
	if err := doc.Validate(); err != nil {
		t.Fatalf("validate converted doc: %v", err)
	}
	out, err := ConvertPOMLToText(doc, FormatMarkdown)
	if err != nil {
		t.Fatalf("convert back: %v", err)
	}
	if out == "" {
		t.Fatalf("expected non-empty markdown output")
	}
}

func TestConvertOrgToPOML(t *testing.T) {
	_, err := ConvertTextToPOML("* heading\ncontent", FormatOrg)
	if err != nil {
		t.Fatalf("convert org: %v", err)
	}
}

func TestConvertPOMLToOrgAndNotImplemented(t *testing.T) {
	doc := Document{
		Role: Block{Body: "Role text"},
		Tasks: []Block{
			{Body: "Task one"},
		},
		Inputs: []Input{{Name: "input1", Required: true, Body: "value"}},
	}
	out, err := ConvertPOMLToText(doc, FormatOrg)
	if err != nil {
		t.Fatalf("convert org: %v", err)
	}
	if !strings.Contains(out, "* Role text") || !strings.Contains(out, "Task one") {
		t.Fatalf("unexpected org output: %s", out)
	}
	if _, err := ConvertPOMLToText(doc, TextFormat("bogus")); !errors.Is(err, ErrNotImplemented) {
		t.Fatalf("expected ErrNotImplemented for bad format, got %v", err)
	}
}
