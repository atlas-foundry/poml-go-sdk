package poml

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestRenderDeckGLGolden(t *testing.T) {
	scenePath := filepath.Join("testdata", "golden", "scene_deckgl.json")
	body, err := os.ReadFile(scenePath)
	if err != nil {
		t.Fatalf("read scene: %v", err)
	}
	var scene Scene
	if err := json.Unmarshal(body, &scene); err != nil {
		t.Fatalf("unmarshal scene: %v", err)
	}
	out, err := (DeckGLRenderer{}).Render(scene)
	if err != nil {
		t.Fatalf("render deckgl: %v", err)
	}

	goldenPath := filepath.Join("testdata", "golden", "scene_deckgl.json")
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.WriteFile(goldenPath, out, 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
	}
	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}

	var gotObj any
	var wantObj any
	if err := json.Unmarshal(out, &gotObj); err != nil {
		t.Fatalf("unmarshal got: %v", err)
	}
	if err := json.Unmarshal(want, &wantObj); err != nil {
		t.Fatalf("unmarshal want: %v", err)
	}
	if !reflect.DeepEqual(gotObj, wantObj) {
		t.Fatalf("deckgl golden mismatch\n--- got ---\n%s\n--- want ---\n%s", string(out), string(want))
	}
}
