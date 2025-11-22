package poml

import (
	"bytes"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConvertMessageDictImage(t *testing.T) {
	base := t.TempDir()
	tmp := filepath.Join(base, "tiny.png")
	if err := os.WriteFile(tmp, []byte{0x89, 0x50, 0x4e, 0x47}, 0o644); err != nil {
		t.Fatalf("write tmp image: %v", err)
	}
	src := `<poml><human-msg>Hello</human-msg><img src="tiny.png" alt="tiny" syntax="image/png"/></poml>`
	doc, err := ParseString(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	out, err := Convert(doc, FormatMessageDict, ConvertOptions{BaseDir: base})
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

func TestConvertDictToolDefinitionParameters(t *testing.T) {
	src := `<poml><tool-definition name="calc"><![CDATA[{"type":"object","properties":{"x":{"type":"number"}}}]]></tool-definition></poml>`
	doc, err := ParseString(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	outAny, err := Convert(doc, FormatDict, ConvertOptions{})
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	out := outAny.(dictOutput)
	if len(out.Tools) != 1 {
		t.Fatalf("expected one tool, got %d", len(out.Tools))
	}
	tool := out.Tools[0].(map[string]any)
	params, ok := tool["parameters"].(map[string]any)
	if !ok || params["type"] != "object" {
		t.Fatalf("expected parsed parameters map, got %+v", tool["parameters"])
	}
}

func TestImageHelpersBuildDataURI(t *testing.T) {
	img := ImageFromBytes([]byte{0x01, 0x02}, "image/png", "tiny")
	if !strings.HasPrefix(img.Src, "data:image/png;base64,") {
		t.Fatalf("expected data uri src, got %s", img.Src)
	}
	if img.Syntax != "image/png" || img.Alt != "tiny" {
		t.Fatalf("image metadata mismatch: %+v", img)
	}
}

func TestConvertBaseDirAndNotImplemented(t *testing.T) {
	tmpDir := t.TempDir()
	rel := "pic.bin"
	path := filepath.Join(tmpDir, rel)
	if err := os.WriteFile(path, []byte{0x01, 0x02, 0x03}, 0o644); err != nil {
		t.Fatalf("write image: %v", err)
	}
	src := `<poml><img src="` + rel + `" alt="pic" syntax="image/custom"/></poml>`
	doc, err := ParseString(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	part, err := buildImagePart(doc.Images[0], ConvertOptions{BaseDir: tmpDir})
	if err != nil {
		t.Fatalf("build image part: %v", err)
	}
	if part["type"] != "image/custom" || part["alt"] != "pic" {
		t.Fatalf("metadata mismatch: %+v", part)
	}
	if data := part["base64"].(string); data == "" {
		t.Fatalf("expected base64 data")
	}
	if _, err := Convert(doc, Format("bogus"), ConvertOptions{}); !errors.Is(err, ErrNotImplemented) {
		t.Fatalf("expected ErrNotImplemented, got %v", err)
	}
}

func TestConvertImageBodyFallback(t *testing.T) {
	src := `<poml><img alt="inline">body-bytes</img></poml>`
	doc, err := ParseString(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	part, err := buildImagePart(doc.Images[0], ConvertOptions{})
	if err != nil {
		t.Fatalf("build image part: %v", err)
	}
	if part["type"] != "image/png" { // default guess
		t.Fatalf("expected default mime, got %v", part["type"])
	}
	if part["base64"] == "" {
		t.Fatalf("expected base64 from body")
	}
}

func TestJSONHelpersAndImageFromFile(t *testing.T) {
	body := `{"a":1}`
	if val, ok := parseJSONStrict(body); !ok {
		t.Fatalf("expected strict parse ok")
	} else if m, ok := val.(map[string]any); !ok || m["a"] != float64(1) {
		t.Fatalf("unexpected strict result: %+v", val)
	}
	if _, ok := parseJSONStrict("{bad"); ok {
		t.Fatalf("expected strict parse failure")
	}
	attrs := []xml.Attr{{Name: xml.Name{Local: "x"}, Value: "1"}, {Name: xml.Name{Local: "y"}, Value: "2"}}
	m := attrsToMap(attrs)
	if m["x"] != "1" || m["y"] != "2" {
		t.Fatalf("attrsToMap mismatch: %+v", m)
	}
	tmp := t.TempDir() + "/pic.gif"
	if err := os.WriteFile(tmp, []byte{0x47, 0x49, 0x46}, 0o644); err != nil {
		t.Fatalf("write gif: %v", err)
	}
	img, err := ImageFromFile(tmp, "", "gifpic")
	if err != nil {
		t.Fatalf("image from file: %v", err)
	}
	if img.Syntax != "image/gif" {
		t.Fatalf("expected gif mime, got %s", img.Syntax)
	}
}

func TestConvertStringHelper(t *testing.T) {
	src := `<poml><assistant-msg>hi</assistant-msg></poml>`
	out, err := ConvertString(src, FormatDict, ConvertOptions{})
	if err != nil {
		t.Fatalf("ConvertString: %v", err)
	}
	if _, ok := out.(dictOutput); !ok {
		t.Fatalf("unexpected type from ConvertString")
	}
}

func TestConvertLangChainWithToolCallAndImage(t *testing.T) {
	src := `<poml>
  <human-msg>Hello</human-msg>
  <tool-request id="c1" name="calc" parameters="{{ { x: 2 } }}"/>
  <tool-response id="c1" name="calc">4</tool-response>
  <tool-result id="c1" name="calc">4</tool-result>
  <tool-error id="c1" name="calc">boom</tool-error>
  <img alt="inline">pixels</img>
  <output-schema>{"type":"object"}</output-schema>
  <runtime max-tokens="5"/>
  <tool-definition name="calc">{"type":"object"}</tool-definition>
</poml>`
	doc, err := ParseString(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	outAny, err := Convert(doc, FormatLangChain, ConvertOptions{})
	if err != nil {
		t.Fatalf("convert langchain: %v", err)
	}
	out := outAny.(map[string]any)
	msgs := out["messages"].([]map[string]any)
	if len(msgs) < 3 {
		t.Fatalf("expected messages including tool call and image")
	}
	foundCall := false
	foundImage := false
	for _, m := range msgs {
		if _, ok := m["data"].(map[string]any)["tool_calls"]; ok {
			foundCall = true
		}
		if content, ok := m["data"].(map[string]any)["content"]; ok {
			if arr, ok := content.([]any); ok && len(arr) > 0 {
				foundImage = true
			}
		}
	}
	if !foundCall || !foundImage {
		t.Fatalf("expected tool call and image messages, got %+v", msgs)
	}
	if out["schema"] == nil || out["runtime"] == nil || out["tools"] == nil {
		t.Fatalf("expected schema/runtime/tools in output")
	}
	var foundResult, foundError bool
	for _, m := range msgs {
		if data, ok := m["data"].(map[string]any); ok {
			if data["result"] == true {
				foundResult = true
			}
			if data["error"] == true {
				foundError = true
			}
		}
	}
	if !foundResult || !foundError {
		t.Fatalf("expected tool result/error markers")
	}
}

func TestOpenAIChatArgumentsAndRuntimeMerge(t *testing.T) {
	src := `<poml>
  <human-msg>Hello</human-msg>
  <tool-request id="c1" name="calc" parameters="{{ { x: 3 } }}"/>
  <tool-response id="c1" name="calc">6</tool-response>
  <tool-result id="c1" name="calc">6</tool-result>
  <tool-error id="c1" name="calc">oops</tool-error>
  <runtime temperature="0.2"/>
  <runtime max-tokens="5"/>
  <tool-definition name="calc">{"type":"object"}</tool-definition>
  <output-schema>{"type":"object"}</output-schema>
</poml>`
	doc, err := ParseString(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	outAny, err := Convert(doc, FormatOpenAIChat, ConvertOptions{})
	if err != nil {
		t.Fatalf("convert openai: %v", err)
	}
	out := outAny.(map[string]any)
	msgs := out["messages"].([]map[string]any)
	foundArgs := false
	for _, m := range msgs {
		if tc, ok := m["tool_calls"]; ok {
			arr := tc.([]any)
			f := arr[0].(map[string]any)
			args := f["function"].(map[string]any)["arguments"].(string)
			if strings.Contains(args, "x") {
				foundArgs = true
			}
		}
	}
	if !foundArgs {
		t.Fatalf("expected normalized tool arguments in messages")
	}
	if out["temperature"] != "0.2" && out["temperature"] != 0.2 {
		t.Fatalf("expected merged runtime temperature, got %v", out["temperature"])
	}
	if out["max_tokens"] != "5" && out["max_tokens"] != 5 {
		t.Fatalf("expected merged runtime max_tokens, got %v", out["max_tokens"])
	}
	if out["tools"] == nil {
		t.Fatalf("expected tools in openai output")
	}
	if out["response_format"] == nil {
		t.Fatalf("expected response_format present")
	}
	var foundResult, foundError bool
	for _, m := range msgs {
		if m["type"] == "error" {
			foundError = true
		}
		if m["type"] == "result" {
			foundResult = true
		}
	}
	if !foundResult || !foundError {
		t.Fatalf("expected tool result/error messages")
	}
}

func TestConvertHintAndObjectContent(t *testing.T) {
	src := `<poml>
  <hint caption="background">See this</hint>
  <cp caption="Doc"><object data="{{ foo }}" syntax="xml"/></cp>
  <audio src="data:audio/mpeg;base64,QQ==" alt="clip"/>
  <video src="data:video/mp4;base64,QQ==" alt="vid"/>
</poml>`
	doc, err := ParseString(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	// message_dict should include hint/cp/audio/video
	msgAny, err := Convert(doc, FormatMessageDict, ConvertOptions{})
	if err != nil {
		t.Fatalf("convert message dict: %v", err)
	}
	msgs := msgAny.([]messageDict)
	if len(msgs) != 4 {
		t.Fatalf("expected 4 messages for hint/cp/audio/video, got %d", len(msgs))
	}
	// openai should emit user content for object data
	openAny, err := Convert(doc, FormatOpenAIChat, ConvertOptions{})
	if err != nil {
		t.Fatalf("convert openai: %v", err)
	}
	open := openAny.(map[string]any)
	if len(open["messages"].([]map[string]any)) != 4 {
		t.Fatalf("expected 2 messages in openai output")
	}
}

func TestRuntimeNormalizedAcrossConverters(t *testing.T) {
	src := `<poml>
  <runtime temperature="0.2" maxTokens="5"/>
  <runtime max-tokens="7" extra="x"/>
</poml>`
	doc, err := ParseString(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	dictAny, err := Convert(doc, FormatDict, ConvertOptions{})
	if err != nil {
		t.Fatalf("dict convert: %v", err)
	}
	dictOut := dictAny.(dictOutput)
	assertRuntimeValue(t, dictOut.Runtime, "temperature", "0.2")
	assertRuntimeValue(t, dictOut.Runtime, "max_tokens", "7")
	assertRuntimeValue(t, dictOut.Runtime, "extra", "x")

	openAIAny, err := Convert(doc, FormatOpenAIChat, ConvertOptions{})
	if err != nil {
		t.Fatalf("openai convert: %v", err)
	}
	openAI := openAIAny.(map[string]any)
	assertRuntimeValue(t, openAI, "temperature", "0.2")
	assertRuntimeValue(t, openAI, "max_tokens", "7")
	assertRuntimeValue(t, openAI, "extra", "x")

	langAny, err := Convert(doc, FormatLangChain, ConvertOptions{})
	if err != nil {
		t.Fatalf("langchain convert: %v", err)
	}
	lang := langAny.(map[string]any)
	rt, ok := lang["runtime"].(map[string]any)
	if !ok {
		t.Fatalf("expected runtime map in langchain output, got %+v", lang["runtime"])
	}
	assertRuntimeValue(t, rt, "temperature", "0.2")
	assertRuntimeValue(t, rt, "max_tokens", "7")
	assertRuntimeValue(t, rt, "extra", "x")
}

func assertRuntimeValue(t *testing.T, m map[string]any, key string, want string) {
	t.Helper()
	got, ok := m[key]
	if !ok {
		t.Fatalf("missing runtime key %q", key)
	}
	if fmt.Sprint(got) != want {
		t.Fatalf("runtime %s mismatch: got %v want %s", key, got, want)
	}
}

func TestBuildImagePartBaseDirAndLimits(t *testing.T) {
	base := t.TempDir()
	inside := filepath.Join(base, "pic.bin")
	if err := os.WriteFile(inside, []byte{0x01, 0x02}, 0o644); err != nil {
		t.Fatalf("write inside: %v", err)
	}
	img := Image{Src: "pic.bin", Syntax: "image/custom"}
	part, err := buildImagePart(img, ConvertOptions{BaseDir: base, MaxImageBytes: 10})
	if err != nil {
		t.Fatalf("build image part within basedir: %v", err)
	}
	if part["type"] != "image/custom" {
		t.Fatalf("expected syntax passthrough")
	}

	// Escape attempt should fail.
	imgEscape := Image{Src: "../escape.bin"}
	if _, err := buildImagePart(imgEscape, ConvertOptions{BaseDir: base}); err == nil {
		t.Fatalf("expected escape attempt to fail")
	}

	// Symlink escape should be rejected, but symlink within base should pass.
	outside := filepath.Join(filepath.Dir(base), "outside.bin")
	if err := os.WriteFile(outside, []byte{0x03}, 0o644); err != nil {
		t.Fatalf("write outside: %v", err)
	}
	escapeLink := filepath.Join(base, "escape-link.bin")
	if err := os.Symlink(outside, escapeLink); err == nil {
		if _, err := buildImagePart(Image{Src: "escape-link.bin"}, ConvertOptions{BaseDir: base}); err == nil {
			t.Fatalf("expected symlink escape to be blocked")
		}
	} else {
		t.Logf("symlinks unsupported: %v", err)
	}
	insideLink := filepath.Join(base, "inside-link.bin")
	if err := os.Symlink(inside, insideLink); err == nil {
		if _, err := buildImagePart(Image{Src: "inside-link.bin", Syntax: "image/custom"}, ConvertOptions{BaseDir: base, MaxImageBytes: 10}); err != nil {
			t.Fatalf("expected symlink within base to work: %v", err)
		}
	} else {
		t.Logf("symlinks unsupported: %v", err)
	}

	// Absolute path blocked unless allowed.
	if _, err := buildImagePart(Image{Src: inside}, ConvertOptions{}); err == nil {
		t.Fatalf("expected absolute read to be blocked without AllowAbsImagePaths")
	}
	if _, err := buildImagePart(Image{Src: inside}, ConvertOptions{AllowAbsImagePaths: true, MaxImageBytes: 10}); err != nil {
		t.Fatalf("expected absolute read when allowed, got %v", err)
	}

	// Size cap enforced.
	if _, err := buildImagePart(Image{Src: inside}, ConvertOptions{BaseDir: base, MaxImageBytes: 1}); err == nil {
		t.Fatalf("expected size cap error")
	}

	// Data URI still allowed without BaseDir.
	if _, err := buildImagePart(Image{Src: "data:image/png;base64,AA==", Syntax: "image/png"}, ConvertOptions{}); err != nil {
		t.Fatalf("data uri should pass: %v", err)
	}
}

func TestImageDefaultSizeLimit(t *testing.T) {
	if defaultMaxImageBytes != 10<<20 {
		t.Fatalf("unexpected defaultMaxImageBytes: %d", defaultMaxImageBytes)
	}
	base := t.TempDir()
	bigPath := filepath.Join(base, "big.bin")
	over := defaultMaxImageBytes + 1
	if err := os.WriteFile(bigPath, bytes.Repeat([]byte{0x01}, int(over)), 0o644); err != nil {
		t.Fatalf("create big: %v", err)
	}
	if _, err := buildImagePart(Image{Src: "big.bin"}, ConvertOptions{BaseDir: base}); err == nil {
		t.Fatalf("expected default max %d to reject large file", defaultMaxImageBytes)
	}

	payload := base64.StdEncoding.EncodeToString([]byte{0x01, 0x02, 0x03, 0x04})
	dataURI := "data:image/png;base64," + payload
	if _, err := buildImagePart(Image{Src: dataURI}, ConvertOptions{MaxImageBytes: 3}); err != nil {
		t.Fatalf("data uri should pass without size enforcement: %v", err)
	}

	if _, err := buildImagePart(Image{Src: "big.bin"}, ConvertOptions{BaseDir: base, MaxImageBytes: over}); err != nil {
		t.Fatalf("expected raised max to allow large file: %v", err)
	}

	if _, err := buildImagePart(Image{Src: "big.bin"}, ConvertOptions{BaseDir: base, MaxImageBytes: -1}); err != nil {
		t.Fatalf("expected unlimited max to allow large file: %v", err)
	}
}
