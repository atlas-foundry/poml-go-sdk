package poml_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/atlas-foundry/poml-go-sdk/poml"
)

func TestGeneratedGoldenFiles(t *testing.T) {
	dir := "testdata/generated"
	files, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("failed to read generated directory: %v", err)
	}

	if len(files) == 0 {
		t.Skip("no generated files found, skipping test")
	}

	for _, f := range files {
		if !strings.HasSuffix(f.Name(), ".poml") {
			continue
		}

		t.Run(f.Name(), func(t *testing.T) {
			path := filepath.Join(dir, f.Name())
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("failed to read file: %v", err)
			}

			// Parse with whitespace preservation
			doc, err := poml.ParseString(string(content))
			if err != nil {
				t.Fatalf("failed to parse: %v", err)
			}

			// Render back to string with whitespace preservation
			var buf bytes.Buffer
			err = doc.EncodeWithOptions(&buf, poml.EncodeOptions{
				PreserveWS:    true,
				PreserveOrder: true,
				// We don't include header in generator, so we shouldn't here if we want exact match,
				// OR we should check if generator includes header.
				// Generator code: sb.WriteString("<poml>\n") -> No XML header.
				IncludeHeader: false, 
			})
			if err != nil {
				t.Fatalf("failed to encode: %v", err)
			}

			// Compare
			if string(content) != buf.String() {
				// If they don't match, it might be due to subtle whitespace or attribute ordering.
				// Let's try to be a bit more lenient or debug.
				// For now, strict equality.
				t.Errorf("round-trip mismatch.\nExpected:\n%s\nGot:\n%s", string(content), buf.String())
			}
		})
	}
}
