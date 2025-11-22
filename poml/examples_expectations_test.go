package poml

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Ensures every upstream example expectation has a matching POML fixture and non-empty content.
func TestExampleExpectationsPresent(t *testing.T) {
	expectFiles, err := filepath.Glob(filepath.Join("testdata", "examples", "expects", "*.txt"))
	if err != nil {
		t.Fatalf("glob expects: %v", err)
	}
	if len(expectFiles) == 0 {
		t.Skip("no expectation fixtures present")
	}
	for _, exp := range expectFiles {
		body, err := os.ReadFile(exp)
		if err != nil {
			t.Fatalf("read expect %s: %v", exp, err)
		}
		if len(strings.TrimSpace(string(body))) == 0 {
			t.Fatalf("expectation %s is empty", exp)
		}
		base := filepath.Base(exp)
		name := strings.TrimSuffix(base, filepath.Ext(base))
		pomlPath := filepath.Join("testdata", "examples", name+".poml")
		if _, err := os.Stat(pomlPath); err != nil {
			t.Fatalf("missing poml fixture for %s: %v", base, err)
		}
	}
}
