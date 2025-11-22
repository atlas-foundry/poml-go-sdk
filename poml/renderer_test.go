package poml

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDeckGLRendererJSON(t *testing.T) {
	doc, err := ParseString(`<poml><diagram id="d"><graph><node id="n" x="0" y="0" z="0"/></graph></diagram></poml>`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	scene, err := DiagramToScene(doc.Diagrams[0])
	if err != nil {
		t.Fatalf("scene: %v", err)
	}
	out, err := (DeckGLRenderer{}).Render(scene)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(string(out), `"id": "d"`) {
		t.Fatalf("expected scene id in deck.gl json: %s", string(out))
	}
}

func TestGraphvizRendererDOT(t *testing.T) {
	pomlPath := filepath.Join("testdata", "diagrams", "chain_sample.poml")
	body, err := os.ReadFile(pomlPath)
	if err != nil {
		t.Fatalf("read poml: %v", err)
	}
	doc, err := ParseString(string(body))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	scene, err := DiagramToScene(doc.Diagrams[0])
	if err != nil {
		t.Fatalf("scene: %v", err)
	}
	dot, err := (GraphvizRenderer{}).Render(scene)
	if err != nil {
		t.Fatalf("render dot: %v", err)
	}
	expectedPath := filepath.Join("testdata", "diagrams", "chain_sample.dot")
	want, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("read expected dot: %v", err)
	}
	if strings.TrimSpace(string(dot)) != strings.TrimSpace(string(want)) {
		t.Fatalf("dot mismatch.\n got:\n%s\nwant:\n%s", string(dot), string(want))
	}
}
