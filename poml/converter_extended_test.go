package poml

import (
	"strings"
	"testing"
)

func TestConvertOpenAIChatWithExtendedUnknown(t *testing.T) {
	src := `<poml><meta><id>x</id><version>1</version><owner>o</owner></meta><role>r</role><task>t</task><unknown-tag foo="bar">hello</unknown-tag></poml>`
	doc, err := ParseReaderWithOptions(strings.NewReader(src), ParseOptions{PreserveWhitespace: true, Validate: false, Extended: ExtendedLenient})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	outAny, err := Convert(doc, FormatOpenAIChat, ConvertOptions{Extended: ExtendedLenient})
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	out, ok := outAny.(map[string]any)
	if !ok {
		t.Fatalf("unexpected type %T", outAny)
	}
	msgs, ok := out["messages"].([]map[string]any)
	if !ok {
		t.Fatalf("unexpected messages type %T", out["messages"])
	}
	found := false
	for _, m := range msgs {
		if m["role"] == "user" {
			if content, ok := m["content"].([]any); ok && len(content) > 0 {
				if c0, ok := content[0].(map[string]any); ok {
					if c0["type"] == "text" && strings.Contains(c0["text"].(string), "[unknown:unknown-tag]") {
						found = true
					}
				}
			}
		}
	}
	if !found {
		t.Fatalf("expected unknown tag to appear in messages: %+v", msgs)
	}
}
