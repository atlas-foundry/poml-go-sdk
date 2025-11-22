package poml

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// Parity check: ensure converters produce stable shapes matching Python fixture expectations.
func TestConverterParityFixtures(t *testing.T) {
	fixture := filepath.Join("testdata", "examples", "parity_basic.poml")
	doc, err := ParseFile(fixture)
	if err != nil {
		t.Fatalf("parse fixture: %v", err)
	}
	if err := doc.Validate(); err != nil {
		t.Fatalf("validate fixture: %v", err)
	}

	tests := []struct {
		name     string
		format   Format
		expected string
		opts     ConvertOptions
	}{
		{"message_dict", FormatMessageDict, filepath.Join("testdata", "examples", "parity_basic.message_dict.json"), ConvertOptions{}},
		{"dict", FormatDict, filepath.Join("testdata", "examples", "parity_basic.dict.json"), ConvertOptions{}},
		{"openai_chat", FormatOpenAIChat, filepath.Join("testdata", "examples", "parity_basic.openai_chat.json"), ConvertOptions{}},
		{"langchain", FormatLangChain, filepath.Join("testdata", "examples", "parity_basic.langchain.json"), ConvertOptions{}},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			out, err := Convert(doc, tc.format, tc.opts)
			if err != nil {
				t.Fatalf("convert (%s): %v", tc.name, err)
			}
			assertJSONEqual(t, out, tc.expected)
		})
	}
}

func TestParseUpstreamExamplesWithExtendedTags(t *testing.T) {
	paths := []string{
		filepath.Join("testdata", "examples", "101_explain_character.poml"),
		filepath.Join("testdata", "examples", "206_expense_send_email.poml"),
	}
	for _, p := range paths {
		p := p
		t.Run(filepath.Base(p), func(t *testing.T) {
			if _, err := ParseFile(p); err != nil {
				t.Fatalf("parse %s: %v", p, err)
			}
		})
	}
}

func TestConverterParityMultimedia(t *testing.T) {
	fixture := filepath.Join("testdata", "examples", "207_multimedia.poml")
	doc, err := ParseFile(fixture)
	if err != nil {
		t.Fatalf("parse fixture: %v", err)
	}
	cases := []struct {
		name     string
		format   Format
		expected string
	}{
		{"message_dict", FormatMessageDict, filepath.Join("testdata", "examples", "parity_multimedia.message_dict.json")},
		{"openai_chat", FormatOpenAIChat, filepath.Join("testdata", "examples", "parity_multimedia.openai_chat.json")},
		{"langchain", FormatLangChain, filepath.Join("testdata", "examples", "parity_multimedia.langchain.json")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, err := Convert(doc, tc.format, ConvertOptions{})
			if err != nil {
				t.Fatalf("convert %s: %v", tc.name, err)
			}
			assertJSONEqual(t, out, tc.expected)
		})
	}
}

func assertJSONEqual(t *testing.T, actual any, expectedPath string) {
	t.Helper()
	expected := loadJSON(t, expectedPath)
	normalizedActual := canonicalizeJSON(t, actual)
	if !reflect.DeepEqual(normalizedActual, expected) {
		t.Fatalf("mismatch for %s\nexpected:\n%s\nactual:\n%s", expectedPath, prettyJSON(t, expected), prettyJSON(t, normalizedActual))
	}
}

func loadJSON(t *testing.T, path string) any {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatalf("unmarshal %s: %v", path, err)
	}
	return v
}

func canonicalizeJSON(t *testing.T, v any) any {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("remarshal: %v", err)
	}
	return out
}

func prettyJSON(t *testing.T, v any) string {
	t.Helper()
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatalf("pretty json: %v", err)
	}
	return string(raw)
}
