package poml

import (
	"os"
	"strings"
	"testing"
)

func TestConvertMessageDictImage(t *testing.T) {
	tmp := t.TempDir() + "/tiny.png"
	if err := os.WriteFile(tmp, []byte{0x89, 0x50, 0x4e, 0x47}, 0o644); err != nil {
		t.Fatalf("write tmp image: %v", err)
	}
	src := `<poml><human-msg>Hello</human-msg><img src="` + tmp + `" alt="tiny" syntax="image/png"/></poml>`
	doc, err := ParseString(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	out, err := Convert(doc, FormatMessageDict, ConvertOptions{BaseDir: ""})
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	msgs := out.([]messageDict)
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
	img, ok := msgs[1].Content.(map[string]any)
	if !ok {
		t.Fatalf("expected image map")
	}
	if img["type"] != "image/png" || img["alt"] != "tiny" {
		t.Fatalf("image metadata mismatch: %+v", img)
	}
	if b64, ok := img["base64"].(string); !ok || b64 == "" {
		t.Fatalf("image base64 missing")
	}
}

func TestConvertOpenAIChatTooling(t *testing.T) {
	src := `<poml>
  <human-msg>Hello</human-msg>
  <tool-definition name="calc">{"type":"object"}</tool-definition>
  <tool-request id="call_1" name="calc" parameters="{{ { x: 1 } }}"/>
  <tool-response id="call_1" name="calc">2</tool-response>
  <output-schema>{"type":"object"}</output-schema>
  <runtime temperature="0.5" max-tokens="10"/>
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
	if len(msgs) < 3 {
		t.Fatalf("expected messages present, got %d", len(msgs))
	}
	if rf, ok := out["response_format"].(map[string]any); !ok || rf["type"] != "json_schema" {
		t.Fatalf("response_format missing: %+v", out["response_format"])
	}
	if temp, ok := out["temperature"]; !ok || temp != "0.5" && temp != 0.5 {
		t.Fatalf("runtime temperature missing: %v", out["temperature"])
	}
	if tools, ok := out["tools"].([]any); !ok || len(tools) == 0 {
		t.Fatalf("tools missing: %+v", out["tools"])
	}
	// Ensure tool call wiring present.
	foundToolCall := false
	for _, m := range msgs {
		if tc, ok := m["tool_calls"]; ok {
			if arr, ok := tc.([]any); ok && len(arr) > 0 {
				foundToolCall = true
				break
			}
		}
	}
	if !foundToolCall {
		t.Fatalf("tool_calls not found in messages")
	}
}

func TestConvertLangChain(t *testing.T) {
	src := `<poml>
  <assistant-msg>Assistant says hi</assistant-msg>
  <tool-response id="r1" name="calc">4</tool-response>
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
	if msgs[0]["type"] != "ai" || !strings.Contains(msgs[0]["data"].(map[string]any)["content"].(string), "Assistant") {
		t.Fatalf("unexpected langchain ai message: %+v", msgs[0])
	}
	if msgs[1]["type"] != "tool" {
		t.Fatalf("expected tool message, got %+v", msgs[1])
	}
}
