package poml

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderDOTSimple(t *testing.T) {
	assertDOTSnapshot(t, "simple.poml")
}

func TestRenderDOTStyled(t *testing.T) {
	assertDOTSnapshot(t, "styled.poml")
}

func assertDOTSnapshot(t *testing.T, name string) {
	t.Helper()
	p := filepath.Join("testdata", "dot", name)
	body, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	doc, err := ParseString(string(body))
	if err != nil {
		t.Fatalf("parse %s: %v", name, err)
	}
	sceneAny, err := Convert(doc, FormatScene, ConvertOptions{})
	if err != nil {
		t.Fatalf("convert to scene: %v", err)
	}
	scene, ok := sceneAny.(Scene)
	if !ok {
		t.Fatalf("unexpected scene type %T", sceneAny)
	}
	out, err := (GraphvizRenderer{}).Render(scene)
	if err != nil {
		t.Fatalf("render DOT: %v", err)
	}
	rendered := strings.TrimSpace(string(out))
	if rendered == "" {
		t.Fatalf("empty render output for %s", name)
	}
	goldenPath := filepath.Join("testdata", "dot", name+".golden")
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.WriteFile(goldenPath, []byte(rendered+"\n"), 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
	}
	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	got := normalizeDOT(rendered)
	expect := normalizeDOT(string(want))
	if got != expect {
		t.Fatalf("render mismatch for %s\n--- got ---\n%s\n--- want ---\n%s", name, got, expect)
	}
}

func normalizeDOT(s string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	for i, l := range lines {
		lines[i] = strings.TrimSpace(l)
	}
	return strings.Join(lines, "\n")
}
