package poml

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseUpstreamExamples(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("testdata", "examples", "*.poml"))
	if err != nil {
		t.Fatalf("glob examples: %v", err)
	}
	if len(files) == 0 {
		t.Skip("no upstream example fixtures present")
	}
	for _, path := range files {
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		if _, err := ParseString(string(body)); err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
	}
}
