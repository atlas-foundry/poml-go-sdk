package poml

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Strict variant: only runs on examples that already satisfy required meta/role/task and parses with validation on.
func TestLosslessRoundTripExamplesStrict(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("testdata", "examples", "*.poml"))
	if err != nil {
		t.Fatalf("glob examples: %v", err)
	}
	parseOpts := ParseOptions{PreserveWhitespace: true, Validate: true, Extended: ExtendedOff}
	encOpts := EncodeOptions{IncludeHeader: false, PreserveOrder: true, PreserveWS: true}

	for _, path := range files {
		path := path
		t.Run(filepath.Base(path), func(t *testing.T) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read: %v", err)
			}
			doc, err := ParseReaderWithOptions(strings.NewReader(string(raw)), parseOpts)
			if err != nil {
				t.Skipf("skip strict: %v", err)
			}
			first, err := encodeToString(doc, encOpts)
			if err != nil {
				t.Fatalf("encode1: %v", err)
			}
			doc2, err := ParseReaderWithOptions(strings.NewReader(first), parseOpts)
			if err != nil {
				t.Fatalf("parse2: %v", err)
			}
			second, err := encodeToString(doc2, encOpts)
			if err != nil {
				t.Fatalf("encode2: %v", err)
			}
			if first != second {
				t.Fatalf("strict round-trip mismatch\n--- first ---\n%s\n--- second ---\n%s", first, second)
			}
		})
	}
}
