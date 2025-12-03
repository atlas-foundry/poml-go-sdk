package poml

import (
	"fmt"
	"os"
	"path/filepath"
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

func TestConvertExtendedTextSegments(t *testing.T) {
	src := `<poml mode="extended">before<meta><id>x</id><version>1</version><owner>o</owner></meta><role>r</role><task>t</task>after</poml>`
	doc, err := ParseReaderWithOptions(strings.NewReader(src), ParseOptions{PreserveWhitespace: true, Validate: true})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	outAny, err := Convert(doc, FormatMessageDict, ConvertOptions{Extended: ExtendedStrict})
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	msgs, ok := outAny.([]messageDict)
	if !ok {
		t.Fatalf("unexpected type %T", outAny)
	}
	if len(msgs) < 2 {
		t.Fatalf("expected text messages surfaced, got %d", len(msgs))
	}
	foundBefore := false
	foundAfter := false
	for _, m := range msgs {
		if s, ok := m.Content.(string); ok {
			if s == "before" {
				foundBefore = true
			}
			if s == "after" {
				foundAfter = true
			}
		}
	}
	if !foundBefore || !foundAfter {
		t.Fatalf("expected text segments preserved (before=%v after=%v)", foundBefore, foundAfter)
	}
}

func TestConvertExtendedRootlessTextFallback(t *testing.T) {
	body := "plain text only"
	doc, err := ParseReaderWithOptions(strings.NewReader(body), ParseOptions{PreserveWhitespace: true, Validate: false, Extended: ExtendedStrict})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	outAny, err := Convert(doc, FormatMessageDict, ConvertOptions{Extended: ExtendedStrict})
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	msgs, ok := outAny.([]messageDict)
	if !ok {
		t.Fatalf("unexpected type %T", outAny)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected single text message, got %d", len(msgs))
	}
	if s, ok := msgs[0].Content.(string); !ok || !strings.Contains(s, "plain text") {
		t.Fatalf("expected plain text content, got %+v", msgs[0].Content)
	}
}

func TestConvertExtendedTextEscapeLiteral(t *testing.T) {
	src := `<poml mode="extended"><meta><id>x</id><version>1</version><owner>o</owner></meta><text><role>literal</role></text><role>r</role><task>t</task></poml>`
	doc, err := ParseReaderWithOptions(strings.NewReader(src), ParseOptions{PreserveWhitespace: true, Validate: true})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	outAny, err := Convert(doc, FormatMessageDict, ConvertOptions{Extended: ExtendedStrict})
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	msgs, ok := outAny.([]messageDict)
	if !ok {
		t.Fatalf("unexpected type %T", outAny)
	}
	found := false
	for _, m := range msgs {
		if s, ok := m.Content.(string); ok && strings.Contains(s, "<role>literal</role>") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected literal text content in messages: %+v", msgs)
	}
}

func TestConvertExtendedMixedTextAndDataOrder(t *testing.T) {
	src := `<poml mode="extended">
  <meta><id>x</id><version>1</version><owner>me</owner></meta>
  <role>r</role><task>t</task>
  Text before data.
  <data syntax="application/json">{"a":1}</data>
  <text>wrapped text</text>
</poml>`
	doc, err := ParseReaderWithOptions(strings.NewReader(src), ParseOptions{PreserveWhitespace: true, Validate: true, Extended: ExtendedStrict})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	outAny, err := Convert(doc, FormatMessageDict, ConvertOptions{Extended: ExtendedStrict})
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	msgs := outAny.([]messageDict)
	if len(msgs) != 3 {
		t.Fatalf("expected 3 messages, got %d (%+v)", len(msgs), msgs)
	}
	if text, ok := msgs[0].Content.(string); !ok || !strings.Contains(text, "Text before data.") {
		t.Fatalf("expected leading text content, got %+v", msgs[0])
	}
	if data, ok := msgs[1].Content.(map[string]any); !ok || data["type"] != "data" || data["syntax"] != "application/json" {
		t.Fatalf("expected data payload, got %+v", msgs[1])
	}
	if text, ok := msgs[2].Content.(string); !ok || !strings.Contains(text, "wrapped text") {
		t.Fatalf("expected trailing wrapped text, got %+v", msgs[2])
	}
}

func TestConvertTextBlocksIgnoredWhenExtendedOff(t *testing.T) {
	src := `<poml><meta><id>x</id><version>1</version><owner>o</owner></meta><role>r</role><text>inline text</text><task>t</task></poml>`
	doc, err := ParseReaderWithOptions(strings.NewReader(src), ParseOptions{PreserveWhitespace: true, Validate: false})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	outAny, err := Convert(doc, FormatMessageDict, ConvertOptions{Extended: ExtendedOff})
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	msgs, ok := outAny.([]messageDict)
	if !ok {
		t.Fatalf("unexpected type %T", outAny)
	}
	for _, m := range msgs {
		if s, ok := m.Content.(string); ok && strings.Contains(s, "inline text") {
			t.Fatalf("expected text block ignored when ExtendedOff, got %+v", msgs)
		}
	}
}

func TestExtendedMediaRoundTrip(t *testing.T) {
	path := filepath.Join("testdata", "examples", "parity_extended_media.poml")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	parseOpts := ParseOptions{PreserveWhitespace: true, Validate: false, Extended: ExtendedLenient}
	doc, err := ParseReaderWithOptions(strings.NewReader(string(raw)), parseOpts)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	var buf strings.Builder
	if err := doc.EncodeWithOptions(&buf, EncodeOptions{IncludeHeader: false, PreserveOrder: true, PreserveWS: true}); err != nil {
		t.Fatalf("encode: %v", err)
	}
	doc2, err := ParseReaderWithOptions(strings.NewReader(buf.String()), parseOpts)
	if err != nil {
		t.Fatalf("parse2: %v", err)
	}
	if len(doc2.Figures) != len(doc.Figures) || len(doc2.Objects) != len(doc.Objects) {
		t.Fatalf("counts changed after round-trip: figs %d->%d objs %d->%d", len(doc.Figures), len(doc2.Figures), len(doc.Objects), len(doc2.Objects))
	}
	for i := range doc.Figures {
		if doc.Figures[i].Src != doc2.Figures[i].Src || strings.TrimSpace(doc.Figures[i].Body) != strings.TrimSpace(doc2.Figures[i].Body) {
			t.Fatalf("figure %d changed after round-trip: before=%+v after=%+v", i, doc.Figures[i], doc2.Figures[i])
		}
	}
	for i := range doc.Objects {
		if strings.TrimSpace(doc.Objects[i].Body) != strings.TrimSpace(doc2.Objects[i].Body) || doc.Objects[i].Syntax != doc2.Objects[i].Syntax {
			t.Fatalf("object %d changed after round-trip: before=%+v after=%+v", i, doc.Objects[i], doc2.Objects[i])
		}
	}
}
