package poml

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Ensures examples round-trip parse→encode→parse deterministically and remain convertible.
func TestLosslessRoundTripExamples(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("testdata", "examples", "*.poml"))
	if err != nil {
		t.Fatalf("glob examples: %v", err)
	}
	if len(files) == 0 {
		t.Fatalf("no example .poml files found")
	}
	parseOpts := ParseOptions{PreserveWhitespace: true, Validate: false, Extended: ExtendedLenient}
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
				t.Fatalf("parse: %v", err)
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
				t.Fatalf("round-trip mismatch\n--- first ---\n%s\n--- second ---\n%s", first, second)
			}

			// Ensure conversions remain possible after round-trip when required sections exist.
			if doc2.Meta.ID == "" || doc2.Role.Body == "" || len(doc2.Tasks) == 0 {
				t.Skip("missing required meta/role/task for conversion; parsed losslessly")
			}
			formats := []Format{FormatMessageDict, FormatDict, FormatOpenAIChat, FormatLangChain, FormatPydantic, FormatScene, FormatSceneJSON}
			opts := ConvertOptions{
				BaseDir:            filepath.Dir(path),
				MaxImageBytes:      1 << 20,
				MaxMediaBytes:      1 << 20,
				AllowAbsImagePaths: true,
				Extended:           ExtendedLenient,
			}
			for _, f := range formats {
				if _, err := Convert(doc2, f, opts); err != nil {
					t.Fatalf("convert %s: %v", f, err)
				}
			}
			// Strict validation should succeed when meta/role/task exist.
			if _, err := ParseReaderWithOptions(strings.NewReader(second), ParseOptions{PreserveWhitespace: true, Validate: true, Extended: ExtendedLenient}); err != nil {
				t.Fatalf("strict parse after round-trip failed: %v", err)
			}
		})
	}
}

func encodeToString(doc Document, opts EncodeOptions) (string, error) {
	var b strings.Builder
	if err := doc.EncodeWithOptions(&b, opts); err != nil {
		return "", err
	}
	return b.String(), nil
}
