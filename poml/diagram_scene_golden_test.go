package poml

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// Ensure diagram → scene export is deterministic and matches golden JSON.
func TestDiagramToSceneGolden(t *testing.T) {
	doc, err := ParseFile(filepath.Join("testdata", "diagrams", "grid_sample.poml"))
	if err != nil {
		t.Fatalf("parse diagram: %v", err)
	}
	scenes, err := diagramsToScenes(doc.Diagrams, defaultSceneExportOptions)
	if err != nil {
		t.Fatalf("diagramsToScenes: %v", err)
	}
	if len(scenes) != 1 {
		t.Fatalf("expected 1 scene, got %d", len(scenes))
	}
	scene := normalizeScene(scenes[0])

	got, err := json.Marshal(scene)
	if err != nil {
		t.Fatalf("marshal got: %v", err)
	}
	goldenPath := filepath.Join("testdata", "diagrams", "grid_sample_scene.json")
	wantBytes, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	var want Scene
	if err := json.Unmarshal(wantBytes, &want); err != nil {
		t.Fatalf("unmarshal golden: %v", err)
	}
	want = normalizeScene(want)
	wantNorm, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("marshal want: %v", err)
	}
	if string(got) != string(wantNorm) {
		t.Fatalf("diagram→scene mismatch\n--- got ---\n%s\n--- want ---\n%s", string(got), string(wantNorm))
	}
}

func normalizeScene(s Scene) Scene {
	sort.Slice(s.Nodes, func(i, j int) bool { return s.Nodes[i].ID < s.Nodes[j].ID })
	sort.Slice(s.Edges, func(i, j int) bool {
		if s.Edges[i].From != s.Edges[j].From {
			return s.Edges[i].From < s.Edges[j].From
		}
		return s.Edges[i].To < s.Edges[j].To
	})
	sort.Slice(s.Layers, func(i, j int) bool { return s.Layers[i].ID < s.Layers[j].ID })
	return s
}
