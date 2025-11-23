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

func TestGraphvizRendererDirectedOverride(t *testing.T) {
	scene := Scene{
		Nodes: []SceneNode{
			{ID: "A", Position: [3]float64{0, 0, 0}},
			{ID: "B", Position: [3]float64{1, 1, 0}},
		},
		Edges: []SceneEdge{
			{From: "A", To: "B", Directed: true, Style: map[string]string{"stroke": "red"}},
		},
	}
	forcedUndirected := false
	dot, err := (GraphvizRenderer{Directed: &forcedUndirected}).Render(scene)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	out := string(dot)
	if !strings.Contains(out, "\"A\" -- \"B\"") {
		t.Fatalf("expected undirected edge, got: %s", out)
	}
	if !strings.Contains(out, "color=\"red\"") {
		t.Fatalf("expected stroke color in attrs: %s", out)
	}
}

func TestRenderersGolden(t *testing.T) {
	scene := Scene{
		ID: "scene-1",
		Nodes: []SceneNode{
			{ID: "n1", Label: "Node 1", Position: [3]float64{0, 0, 0}, Style: map[string]string{"color": "blue"}},
			{ID: "n2", Label: "Node 2", Position: [3]float64{1, 1, 0}, Style: map[string]string{"shape": "hex"}},
		},
		Edges: []SceneEdge{
			{From: "n1", To: "n2", Kind: "link", Directed: true, Style: map[string]string{"stroke": "black"}},
		},
	}

	// Deck.gl JSON
	jsonOut, err := (DeckGLRenderer{}).Render(scene)
	if err != nil {
		t.Fatalf("deckgl render: %v", err)
	}
	assertJSONEqualRaw(t, jsonOut, filepath.Join("testdata", "golden", "scene_deckgl.json"))

	// Graphviz DOT
	dotOut, err := (GraphvizRenderer{}).Render(scene)
	if err != nil {
		t.Fatalf("graphviz render: %v", err)
	}
	wantDOT := readFile(t, filepath.Join("testdata", "golden", "scene_graphviz.dot"))
	if strings.TrimSpace(string(dotOut)) != strings.TrimSpace(wantDOT) {
		t.Fatalf("graphviz mismatch\n got:\n%s\nwant:\n%s", string(dotOut), wantDOT)
	}
}

func TestBuildDOTAttrsAndStyles(t *testing.T) {
	got := buildDOTAttrs(map[string]string{
		"b":     "2",
		"empty": " ",
		"a":     "1",
	})
	if got != " [a=\"1\",b=\"2\"]" {
		t.Fatalf("unexpected attrs: %s", got)
	}

	tests := []struct {
		existing string
		extra    string
		want     string
	}{
		{"", "filled", "filled"},
		{"rounded", "filled", "rounded,filled"},
		{"existing", "   ", "existing"},
	}
	for _, tt := range tests {
		if res := appendStyle(tt.existing, tt.extra); res != tt.want {
			t.Fatalf("appendStyle(%q,%q)=%q want %q", tt.existing, tt.extra, res, tt.want)
		}
	}
}

func TestBuildDOTNodeAttrs(t *testing.T) {
	node := SceneNode{
		ID:       "node-1",
		Position: [3]float64{1.2, 3.4, 0},
		Style: map[string]string{
			"shape":  "hex",
			"color":  "blue",
			"stroke": "black",
		},
	}
	attrs := buildDOTNodeAttrs(node)
	for _, want := range []string{`label="node-1"`, `shape="hexagon"`, `fillcolor="blue"`, `style="filled"`, `color="black"`, `pos="1.200,3.400!"`} {
		if !strings.Contains(attrs, want) {
			t.Fatalf("expected %s in attrs %s", want, attrs)
		}
	}
}
