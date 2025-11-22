package poml

import (
	"encoding/base64"
	"strings"
	"testing"
)

const pngData = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNkYAAAAAYAAjCB0C8AAAAASUVORK5CYII="

func TestOpenAIChatToolCallShape(t *testing.T) {
	src := `<poml>
  <human-msg>Search for Python</human-msg>
  <tool-request id="call_123" name="search" parameters="{{ { query: 'Python' } }}" />
  <tool-response id="call_123" name="search">Python is a language.</tool-response>
</poml>`
	doc, err := ParseString(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	outAny, err := Convert(doc, FormatOpenAIChat, ConvertOptions{})
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	out := outAny.(map[string]any)
	msgs := out["messages"].([]map[string]any)
	if len(msgs) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(msgs))
	}
	callMsg := msgs[1]
	if callMsg["role"] != "assistant" {
		t.Fatalf("expected assistant role for tool call")
	}
	calls := callMsg["tool_calls"].([]any)
	fn := calls[0].(map[string]any)["function"].(map[string]any)
	if fn["name"] != "search" || !strings.Contains(fn["arguments"].(string), "Python") {
		t.Fatalf("tool call function mismatch: %+v", fn)
	}
	resp := msgs[2]
	if resp["role"] != "tool" || resp["tool_call_id"] != "call_123" {
		t.Fatalf("tool response mismatch: %+v", resp)
	}
}

func TestLangChainToolCallShape(t *testing.T) {
	src := `<poml>
  <tool-request id="call_456" name="calculate" parameters="{{ { expression: '2 + 2' } }}" />
  <tool-response id="call_456" name="calculate">4</tool-response>
</poml>`
	doc, err := ParseString(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	outAny, err := Convert(doc, FormatLangChain, ConvertOptions{})
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	out := outAny.(map[string]any)
	msgs := out["messages"].([]map[string]any)
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
	call := msgs[0]["data"].(map[string]any)["tool_calls"].([]any)[0].(map[string]any)
	if call["name"] != "calculate" {
		t.Fatalf("tool call name mismatch: %+v", call)
	}
	args := call["args"].(map[string]any)
	if args["expression"] != "2 + 2" {
		t.Fatalf("tool call args mismatch: %+v", args)
	}
}

func TestDictFormatWithSchemaToolsRuntimeShape(t *testing.T) {
	src := `<poml>
  <output-schema>{"type": "object", "properties": {"answer": {"type": "string"}}, "required": ["answer"]}</output-schema>
  <tool-definition name="search" description="Search for information">
    {"type": "object", "properties": {"query": {"type": "string"}}}
  </tool-definition>
  <runtime temperature="0.5" max-tokens="150" />
  <human-msg>What is AI?</human-msg>
</poml>`
	doc, err := ParseString(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	outAny, err := Convert(doc, FormatDict, ConvertOptions{})
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	out := outAny.(dictOutput)
	if len(out.Messages) != 1 || out.Messages[0].Speaker != "human" {
		t.Fatalf("messages mismatch: %+v", out.Messages)
	}
	if out.Schema == nil {
		t.Fatalf("schema missing")
	}
	if len(out.Tools) != 1 {
		t.Fatalf("tools missing: %+v", out.Tools)
	}
	rt := out.Runtime
	if rt["temperature"] != 0.5 && rt["temperature"] != "0.5" {
		t.Fatalf("runtime temp mismatch: %+v", rt)
	}
	if rt["max_tokens"] != 150 && rt["max_tokens"] != "150" {
		t.Fatalf("runtime max_tokens mismatch: %+v", rt)
	}
}

func TestOpenAIChatRuntimeSnakeCase(t *testing.T) {
	src := `<poml>
  <runtime maxTokens="1000" topP="0.95" frequencyPenalty="0.5" presencePenalty="0.3" stop-sequences='["END","STOP"]' />
  <human-msg>Test</human-msg>
</poml>`
	doc, err := ParseString(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	outAny, err := Convert(doc, FormatOpenAIChat, ConvertOptions{})
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	out := outAny.(map[string]any)
	if out["max_tokens"] != 1000 && out["max_tokens"] != "1000" {
		t.Fatalf("max_tokens mismatch: %+v", out["max_tokens"])
	}
	if out["top_p"] != 0.95 && out["top_p"] != "0.95" {
		t.Fatalf("top_p mismatch: %+v", out["top_p"])
	}
	if out["frequency_penalty"] == nil || out["presence_penalty"] == nil {
		t.Fatalf("penalties missing")
	}
	if seq, ok := out["stop_sequences"].([]any); !ok || len(seq) != 2 {
		t.Fatalf("stop_sequences mismatch: %+v", out["stop_sequences"])
	}
}

func TestOpenAIChatWithToolsShape(t *testing.T) {
	src := `<poml>
  <tool-definition name="get_weather" description="Get weather information">
    {"type": "object", "properties": {"location": {"type": "string"}}, "required": ["location"]}
  </tool-definition>
  <human-msg>What's the weather?</human-msg>
</poml>`
	doc, err := ParseString(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	outAny, err := Convert(doc, FormatOpenAIChat, ConvertOptions{})
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	out := outAny.(map[string]any)
	tools := out["tools"].([]any)
	if len(tools) != 1 {
		t.Fatalf("tools missing: %+v", tools)
	}
	fn := tools[0].(map[string]any)["function"].(map[string]any)
	if fn["name"] != "get_weather" {
		t.Fatalf("tool name mismatch: %+v", fn)
	}
}

func TestMessageDictFormatSimple(t *testing.T) {
	doc, err := ParseString("<poml><human-msg>Hello world</human-msg></poml>")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	outAny, err := Convert(doc, FormatMessageDict, ConvertOptions{})
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	out := outAny.([]messageDict)
	if len(out) != 1 || out[0].Speaker != "human" || out[0].Content != "Hello world" {
		t.Fatalf("message_dict mismatch: %+v", out)
	}
}

func TestPydanticFormatSimple(t *testing.T) {
	doc, err := ParseString("<poml><human-msg>Hello world</human-msg></poml>")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	outAny, err := Convert(doc, FormatPydantic, ConvertOptions{})
	if err != nil {
		t.Fatalf("convert pydantic: %v", err)
	}
	out := outAny.(dictOutput)
	if len(out.Messages) != 1 || out.Messages[0].Speaker != "human" {
		t.Fatalf("pydantic messages mismatch: %+v", out.Messages)
	}
}

func TestPydanticIncludesMedia(t *testing.T) {
	src := `<poml><img src="data:image/png;base64,AA==" alt="tiny" syntax="image/png"/></poml>`
	doc, err := ParseString(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	outAny, err := Convert(doc, FormatPydantic, ConvertOptions{})
	if err != nil {
		t.Fatalf("convert pydantic: %v", err)
	}
	out := outAny.(dictOutput)
	if len(out.Media) != 1 {
		t.Fatalf("expected media array populated, got %d", len(out.Media))
	}
}

func TestImageFormatsBasics(t *testing.T) {
	raw, _ := base64.StdEncoding.DecodeString(pngData)
	img := ImageFromBytes(raw, "image/png", "tiny")
	doc := Document{}
	doc.AddImage(img)
	doc.Elements = doc.defaultElements()

	msgDict, err := convertMessageDict(doc, ConvertOptions{})
	if err != nil {
		t.Fatalf("message dict convert: %v", err)
	}
	if imgPart, ok := msgDict[0].Content.(map[string]any); !ok || !strings.HasPrefix(imgPart["base64"].(string), pngData[:28]) {
		t.Fatalf("image base64 missing: %+v", msgDict)
	}

	openai, err := convertOpenAIChat(doc, ConvertOptions{})
	if err != nil {
		t.Fatalf("openai convert: %v", err)
	}
	messages := openai["messages"].([]map[string]any)
	if len(messages) != 1 {
		t.Fatalf("expected one message for image")
	}
	parts := messages[0]["content"].([]any)
	if len(parts) != 2 {
		t.Fatalf("expected text+image parts, got %v", parts)
	}
}
