package poml

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
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
		{"persona_message_dict", FormatMessageDict, filepath.Join("testdata", "examples", "parity_persona.message_dict.json"), ConvertOptions{}},
		{"persona_dict", FormatDict, filepath.Join("testdata", "examples", "parity_persona.dict.json"), ConvertOptions{}},
		{"persona_openai_chat", FormatOpenAIChat, filepath.Join("testdata", "examples", "parity_persona.openai_chat.json"), ConvertOptions{}},
		{"persona_langchain", FormatLangChain, filepath.Join("testdata", "examples", "parity_persona.langchain.json"), ConvertOptions{}},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			fixturePath := filepath.Join("testdata", "examples", "parity_basic.poml")
			if strings.HasPrefix(tc.name, "persona_") {
				fixturePath = filepath.Join("testdata", "examples", "parity_persona.poml")
			}
			doc, err := ParseFile(fixturePath)
			if err != nil {
				t.Fatalf("parse fixture: %v", err)
			}
			if err := doc.Validate(); err != nil {
				t.Fatalf("validate fixture: %v", err)
			}

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
		filepath.Join("testdata", "examples", "ts_reference_basic.poml"),
	}
	for _, p := range paths {
		p := p
		t.Run(filepath.Base(p), func(t *testing.T) {
			doc, err := ParseFile(p)
			if err != nil {
				t.Fatalf("parse %s: %v", p, err)
			}
			if filepath.Base(p) == "ts_reference_basic.poml" {
				if err := doc.Validate(); err != nil {
					t.Fatalf("validate %s: %v", p, err)
				}
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

func TestConvertTsReferenceExample(t *testing.T) {
	fixture := filepath.Join("testdata", "examples", "ts_reference_basic.poml")
	doc, err := ParseFile(fixture)
	if err != nil {
		t.Fatalf("parse fixture: %v", err)
	}
	if err := doc.Validate(); err != nil {
		t.Fatalf("validate fixture: %v", err)
	}

	for _, format := range []Format{FormatMessageDict, FormatDict, FormatOpenAIChat, FormatLangChain} {
		format := format
		t.Run(string(format), func(t *testing.T) {
			if _, err := Convert(doc, format, ConvertOptions{}); err != nil {
				t.Fatalf("convert %s: %v", format, err)
			}
		})
	}
}

func TestConverterParityExtendedMixed(t *testing.T) {
	fixture := filepath.Join("testdata", "examples", "parity_extended_mixed.poml")
	doc, err := ParseFile(fixture)
	if err != nil {
		t.Fatalf("parse fixture: %v", err)
	}
	f := false
	if err := doc.ValidateWithOptions(ValidateOptions{Extended: ExtendedStrict, RejectUnknownAttrs: &f}); err != nil {
		t.Fatalf("validate fixture: %v", err)
	}

	cases := []struct {
		name     string
		format   Format
		expected string
	}{
		{"message_dict", FormatMessageDict, filepath.Join("testdata", "examples", "parity_extended_mixed.message_dict.json")},
		{"openai_chat", FormatOpenAIChat, filepath.Join("testdata", "examples", "parity_extended_mixed.openai_chat.json")},
		{"langchain", FormatLangChain, filepath.Join("testdata", "examples", "parity_extended_mixed.langchain.json")},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			out, err := Convert(doc, tc.format, ConvertOptions{Extended: ExtendedStrict})
			if err != nil {
				t.Fatalf("convert %s: %v", tc.name, err)
			}
			assertJSONEqual(t, out, tc.expected)
		})
	}
}

func TestConverterParityExtendedMedia(t *testing.T) {
	fixture := filepath.Join("testdata", "examples", "parity_extended_media.poml")
	doc, err := ParseFile(fixture)
	if err != nil {
		t.Fatalf("parse fixture: %v", err)
	}
	if err := doc.ValidateWithOptions(ValidateOptions{Extended: ExtendedStrict}); err != nil {
		t.Fatalf("validate fixture: %v", err)
	}

	cases := []struct {
		name     string
		format   Format
		expected string
		opts     ConvertOptions
	}{
		{"message_dict", FormatMessageDict, filepath.Join("testdata", "examples", "parity_extended_media.message_dict.json"), ConvertOptions{Extended: ExtendedStrict}},
		{"openai_chat", FormatOpenAIChat, filepath.Join("testdata", "examples", "parity_extended_media.openai_chat.json"), ConvertOptions{Extended: ExtendedStrict}},
		{"langchain", FormatLangChain, filepath.Join("testdata", "examples", "parity_extended_media.langchain.json"), ConvertOptions{Extended: ExtendedStrict}},
		{"message_dict_off", FormatMessageDict, filepath.Join("testdata", "examples", "parity_extended_media.off.message_dict.json"), ConvertOptions{Extended: ExtendedOff}},
		{"openai_chat_off", FormatOpenAIChat, filepath.Join("testdata", "examples", "parity_extended_media.off.openai_chat.json"), ConvertOptions{Extended: ExtendedOff}},
		{"langchain_off", FormatLangChain, filepath.Join("testdata", "examples", "parity_extended_media.off.langchain.json"), ConvertOptions{Extended: ExtendedOff}},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			out, err := Convert(doc, tc.format, tc.opts)
			if err != nil {
				t.Fatalf("convert %s: %v", tc.name, err)
			}
			assertJSONEqual(t, out, tc.expected)
		})
	}
}

func TestConverterParityExtendedTextEscape(t *testing.T) {
	fixture := filepath.Join("testdata", "examples", "parity_extended_textescape.poml")
	doc, err := ParseFile(fixture)
	if err != nil {
		t.Fatalf("parse fixture: %v", err)
	}
	if err := doc.ValidateWithOptions(ValidateOptions{Extended: ExtendedStrict}); err != nil {
		t.Fatalf("validate fixture: %v", err)
	}
	cases := []struct {
		name     string
		format   Format
		expected string
	}{
		{"message_dict", FormatMessageDict, filepath.Join("testdata", "examples", "parity_extended_textescape.message_dict.json")},
		{"openai_chat", FormatOpenAIChat, filepath.Join("testdata", "examples", "parity_extended_textescape.openai_chat.json")},
		{"langchain", FormatLangChain, filepath.Join("testdata", "examples", "parity_extended_textescape.langchain.json")},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			out, err := Convert(doc, tc.format, ConvertOptions{Extended: ExtendedStrict})
			if err != nil {
				t.Fatalf("convert %s: %v", tc.name, err)
			}
			assertJSONEqual(t, out, tc.expected)
		})
	}
}

func TestConverterParityCoreFull(t *testing.T) {
	fixture := filepath.Join("testdata", "examples", "core_full.poml")
	doc, err := ParseFile(fixture)
	if err != nil {
		t.Fatalf("parse fixture: %v", err)
	}
	if err := doc.Validate(); err != nil {
		t.Fatalf("validate fixture: %v", err)
	}
	cases := []struct {
		name     string
		format   Format
		expected string
	}{
		{"message_dict", FormatMessageDict, filepath.Join("testdata", "examples", "core_full.message_dict.json")},
		{"openai_chat", FormatOpenAIChat, filepath.Join("testdata", "examples", "core_full.openai_chat.json")},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			out, err := Convert(doc, tc.format, ConvertOptions{})
			if err != nil {
				t.Fatalf("convert %s: %v", tc.name, err)
			}
			assertJSONEqual(t, out, tc.expected)
		})
	}
}

func TestConverterParityMetaAttrs(t *testing.T) {
	fixture := filepath.Join("testdata", "examples", "meta_attrs.poml")
	doc, err := ParseFile(fixture)
	if err != nil {
		t.Fatalf("parse fixture: %v", err)
	}
	if err := doc.Validate(); err != nil {
		t.Fatalf("validate fixture: %v", err)
	}
	cases := []struct {
		name     string
		format   Format
		expected string
	}{
		{"message_dict", FormatMessageDict, filepath.Join("testdata", "examples", "meta_attrs.message_dict.json")},
		{"dict", FormatDict, filepath.Join("testdata", "examples", "meta_attrs.dict.json")},
		{"openai_chat", FormatOpenAIChat, filepath.Join("testdata", "examples", "meta_attrs.openai_chat.json")},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			out, err := Convert(doc, tc.format, ConvertOptions{})
			if err != nil {
				t.Fatalf("convert %s: %v", tc.name, err)
			}
			assertJSONEqual(t, out, tc.expected)
		})
	}
}

func TestConverterParityExtendedAttrs(t *testing.T) {
	fixture := filepath.Join("testdata", "examples", "parity_extended_attrs.poml")
	doc, err := ParseFile(fixture)
	if err != nil {
		t.Fatalf("parse fixture: %v", err)
	}
	if err := doc.ValidateWithOptions(ValidateOptions{Extended: ExtendedStrict}); err != nil {
		t.Fatalf("validate fixture: %v", err)
	}
	cases := []struct {
		name     string
		format   Format
		expected string
		opts     ConvertOptions
	}{
		{"message_dict", FormatMessageDict, filepath.Join("testdata", "examples", "parity_extended_attrs.message_dict.json"), ConvertOptions{Extended: ExtendedStrict}},
		{"openai_chat", FormatOpenAIChat, filepath.Join("testdata", "examples", "parity_extended_attrs.openai_chat.json"), ConvertOptions{Extended: ExtendedStrict}},
		{"langchain", FormatLangChain, filepath.Join("testdata", "examples", "parity_extended_attrs.langchain.json"), ConvertOptions{Extended: ExtendedStrict}},
		{"message_dict_off", FormatMessageDict, filepath.Join("testdata", "examples", "parity_extended_attrs.off.message_dict.json"), ConvertOptions{Extended: ExtendedOff}},
		{"openai_chat_off", FormatOpenAIChat, filepath.Join("testdata", "examples", "parity_extended_attrs.off.openai_chat.json"), ConvertOptions{Extended: ExtendedOff}},
		{"langchain_off", FormatLangChain, filepath.Join("testdata", "examples", "parity_extended_attrs.off.langchain.json"), ConvertOptions{Extended: ExtendedOff}},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			out, err := Convert(doc, tc.format, tc.opts)
			if err != nil {
				t.Fatalf("convert %s: %v", tc.name, err)
			}
			assertJSONEqual(t, out, tc.expected)
		})
	}
}

func TestConverterParityExtendedData(t *testing.T) {
	fixture := filepath.Join("testdata", "examples", "extended_data_block.poml")
	doc, err := ParseFile(fixture)
	if err != nil {
		t.Fatalf("parse fixture: %v", err)
	}
	if err := doc.ValidateWithOptions(ValidateOptions{Extended: ExtendedStrict}); err != nil {
		t.Fatalf("validate fixture: %v", err)
	}
	cases := []struct {
		name     string
		format   Format
		expected string
		opts     ConvertOptions
	}{
		{"message_dict", FormatMessageDict, filepath.Join("testdata", "examples", "extended_data_block.message_dict.json"), ConvertOptions{Extended: ExtendedStrict}},
		{"openai_chat", FormatOpenAIChat, filepath.Join("testdata", "examples", "extended_data_block.openai_chat.json"), ConvertOptions{Extended: ExtendedStrict}},
		{"langchain", FormatLangChain, filepath.Join("testdata", "examples", "extended_data_block.langchain.json"), ConvertOptions{Extended: ExtendedStrict}},
		{"message_dict_off", FormatMessageDict, filepath.Join("testdata", "examples", "extended_data_block.off.message_dict.json"), ConvertOptions{Extended: ExtendedOff}},
		{"openai_chat_off", FormatOpenAIChat, filepath.Join("testdata", "examples", "extended_data_block.off.openai_chat.json"), ConvertOptions{Extended: ExtendedOff}},
		{"langchain_off", FormatLangChain, filepath.Join("testdata", "examples", "extended_data_block.off.langchain.json"), ConvertOptions{Extended: ExtendedOff}},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			out, err := Convert(doc, tc.format, tc.opts)
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

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}

func prettyJSON(t *testing.T, v any) string {
	t.Helper()
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatalf("pretty json: %v", err)
	}
	return string(raw)
}
