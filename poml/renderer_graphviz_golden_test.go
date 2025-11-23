package poml

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestRenderGraphvizGolden(t *testing.T) {
	scenePath := filepath.Join("testdata", "golden", "scene_deckgl.json")
	body, err := os.ReadFile(scenePath)
	if err != nil {
		t.Fatalf("read scene: %v", err)
	}
	var scene Scene
	if err := json.Unmarshal(body, &scene); err != nil {
		t.Fatalf("unmarshal scene: %v", err)
	}
	out, err := (GraphvizRenderer{}).Render(scene)
	if err != nil {
		t.Fatalf("render graphviz: %v", err)
	}

	goldenPath := filepath.Join("testdata", "golden", "scene_graphviz.dot")
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.WriteFile(goldenPath, out, 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
	}
	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}

	gotNorm := normalizeDOT(string(out))
	wantNorm := normalizeDOT(string(want))
	if gotNorm != wantNorm {
		t.Fatalf("graphviz golden mismatch\n--- got ---\n%s\n--- want ---\n%s", gotNorm, wantNorm)
	}
}
