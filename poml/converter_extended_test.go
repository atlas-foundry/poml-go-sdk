package poml

import (
	"fmt"
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

func TestConvertExtendedOpAndFigure(t *testing.T) {
	src := `<poml><meta><id>x</id><version>1</version><owner>o</owner></meta><role>r</role><task>t</task><op name="demo" kind="custom" args='{"n":1}'>do it</op><figure src="data:image/svg+xml;base64,PHN2Zz48L3N2Zz4=" alt="svg" syntax="image/svg+xml"/></poml>`
	doc, err := ParseReaderWithOptions(strings.NewReader(src), ParseOptions{PreserveWhitespace: true, Validate: true, Extended: ExtendedStrict})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	// message_dict should surface the op and figure payloads.
	outAny, err := Convert(doc, FormatMessageDict, ConvertOptions{Extended: ExtendedStrict})
	if err != nil {
		t.Fatalf("convert message_dict: %v", err)
	}
	msgs, ok := outAny.([]messageDict)
	if !ok {
		t.Fatalf("unexpected message_dict type %T", outAny)
	}
	if len(msgs) != 2 {
		t.Fatalf("expected two extended messages, got %d", len(msgs))
	}
	opPayload, ok := msgs[0].Content.(map[string]any)
	if !ok || opPayload["type"] != "op" || opPayload["name"] != "demo" {
		t.Fatalf("unexpected op payload: %+v", msgs[0])
	}

	// openai_chat should embed extended pieces as user messages.
	openaiAny, err := Convert(doc, FormatOpenAIChat, ConvertOptions{Extended: ExtendedStrict})
	if err != nil {
		t.Fatalf("convert openai: %v", err)
	}
	openai, ok := openaiAny.(map[string]any)
	if !ok {
		t.Fatalf("unexpected openai type %T", openaiAny)
	}
	openaiMsgs, ok := openai["messages"].([]map[string]any)
	if !ok {
		t.Fatalf("unexpected openai messages type %T", openai["messages"])
	}
	if len(openaiMsgs) != 2 {
		t.Fatalf("expected two openai messages, got %d", len(openaiMsgs))
	}
	firstContent, _ := openaiMsgs[0]["content"].([]any)
	if len(firstContent) == 0 || !strings.Contains(fmt.Sprint(firstContent[0]), "[op:demo]") {
		t.Fatalf("expected op marker in first message: %+v", firstContent)
	}
	secondContent, _ := openaiMsgs[1]["content"].([]any)
	foundImage := false
	for _, c := range secondContent {
		if m, ok := c.(map[string]any); ok && m["type"] == "image_url" && strings.Contains(fmt.Sprint(m["image_url"]), "image/svg+xml") {
			foundImage = true
		}
	}
	if !foundImage {
		t.Fatalf("expected figure/image_url in second message: %+v", secondContent)
	}
}
