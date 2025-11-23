package poml

import (
	"context"
	"path/filepath"
	"testing"

	"go.opentelemetry.io/otel/trace/noop"
)

func TestValidateWithTrace(t *testing.T) {
	doc, err := ParseString(minimalPOML)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if err := doc.ValidateWithTrace(context.Background(), TraceOptions{TracerProvider: noop.NewTracerProvider()}); err != nil {
		t.Fatalf("ValidateWithTrace: %v", err)
	}
}

func TestDumpFileAtomicWrite(t *testing.T) {
	doc, err := ParseString(minimalPOML)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	tmp := filepath.Join(t.TempDir(), "out.poml")
	if err := doc.DumpFile(tmp, EncodeOptions{}); err != nil {
		t.Fatalf("DumpFile: %v", err)
	}
}

func TestGuessMediaMime(t *testing.T) {
	cases := map[string]string{
		"clip.mp3":  "audio/mpeg",
		"video.mp4": "video/mp4",
		"other.bin": "application/octet-stream",
	}
	for path, expect := range cases {
		if got := guessMediaMime(path); got != expect {
			t.Fatalf("guessMediaMime(%s) = %s, want %s", path, got, expect)
		}
	}
}

func TestMutatorRemoveBranches(t *testing.T) {
	doc := Document{
		Tasks:      []Block{{Body: "a"}, {Body: "b"}},
		Inputs:     []Input{{Body: "in"}},
		Documents:  []DocRef{{Src: "doc"}},
		Styles:     []Style{{Outputs: []Output{{Body: "style"}}}},
		OutFormats: []OutputFormat{{Body: "fmt"}},
		Messages:   []Message{{Body: "m1"}, {Body: "m2"}, {Body: "m3"}},
		ToolResps:  []ToolResponse{{Body: "resp"}},
		Schema:     OutputSchema{Body: "schema"},
		Images:     []Image{{Body: "img"}},
	}
	m := Mutator{doc: &doc}
	m.Remove(Element{Type: ElementTask, Index: 0})
	m.Remove(Element{Type: ElementInput, Index: 0})
	m.Remove(Element{Type: ElementDocument, Index: 0})
	m.Remove(Element{Type: ElementStyle, Index: 0})
	m.Remove(Element{Type: ElementOutputFormat, Index: 0})
	m.Remove(Element{Type: ElementHumanMsg, Index: 0})
	m.Remove(Element{Type: ElementToolResponse, Index: 0})
	m.Remove(Element{Type: ElementOutputSchema, Index: 0})
	m.Remove(Element{Type: ElementImage, Index: 0})

	if len(doc.Tasks) != 1 || len(doc.Inputs) != 0 || len(doc.Documents) != 0 || len(doc.Styles) != 0 || len(doc.OutFormats) != 0 || len(doc.Messages) != 2 || len(doc.ToolResps) != 0 || doc.Schema.Body != "" || len(doc.Images) != 0 {
		t.Fatalf("mutator remove branches did not update slices")
	}
}
