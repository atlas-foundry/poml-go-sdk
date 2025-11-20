package poml

import (
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Format enumerates output conversion targets inspired by the Python SDK.
type Format string

const (
	FormatMessageDict Format = "message_dict"
	FormatDict        Format = "dict"
	FormatOpenAIChat  Format = "openai_chat"
	FormatLangChain   Format = "langchain"
	FormatPydantic    Format = "pydantic"
)

// ConvertOptions holds knobs for conversion (context, runtime flags, etc.).
// This will expand when format support is implemented.
type ConvertOptions struct {
	// Context may include auxiliary data (e.g., file paths) needed for conversions.
	Context map[string]any
	// BaseDir is used to resolve relative asset paths (e.g., <img src>).
	BaseDir string
}

// ErrNotImplemented signals that a conversion target is not yet supported.
var ErrNotImplemented = errors.New("conversion not implemented")

// Convert transforms a parsed Document into the requested format.
func Convert(doc Document, format Format, opts ConvertOptions) (any, error) {
	switch format {
	case FormatMessageDict:
		return convertMessageDict(doc, opts)
	case FormatDict, FormatPydantic:
		return convertDict(doc, opts)
	case FormatOpenAIChat:
		return convertOpenAIChat(doc, opts)
	case FormatLangChain:
		return convertLangChain(doc, opts)
	default:
		return nil, ErrNotImplemented
	}
}

// ConvertString parses a POML string and converts it in one step.
func ConvertString(body string, format Format, opts ConvertOptions) (any, error) {
	doc, err := ParseString(body)
	if err != nil {
		return nil, err
	}
	return Convert(doc, format, opts)
}

type messageDict struct {
	Speaker string `json:"speaker"`
	Content any    `json:"content"`
}

func convertMessageDict(doc Document, opts ConvertOptions) ([]messageDict, error) {
	var msgs []messageDict
	for _, el := range doc.resolveOrder() {
		switch el.Type {
		case ElementHumanMsg, ElementAssistantMsg, ElementSystemMsg:
			payload := doc.Messages[el.Index]
			content := strings.TrimSpace(payload.Body)
			msgs = append(msgs, messageDict{Speaker: roleToSpeaker(payload.Role), Content: content})
		case ElementToolResponse:
			payload := doc.ToolResps[el.Index]
			msgs = append(msgs, messageDict{Speaker: "tool", Content: strings.TrimSpace(payload.Body)})
		case ElementImage:
			im := doc.Images[el.Index]
			part, err := buildImagePart(im, opts)
			if err != nil {
				return nil, err
			}
			msgs = append(msgs, messageDict{Speaker: "human", Content: part})
		}
	}
	return msgs, nil
}

type dictOutput struct {
	Messages []messageDict `json:"messages"`
	Schema   any           `json:"schema,omitempty"`
	Tools    []any         `json:"tools,omitempty"`
	Runtime  map[string]any `json:"runtime,omitempty"`
}

func convertDict(doc Document, opts ConvertOptions) (dictOutput, error) {
	msgs, err := convertMessageDict(doc, opts)
	if err != nil {
		return dictOutput{}, err
	}
	out := dictOutput{Messages: msgs}
	if doc.hasSchema() {
		out.Schema = parseJSONFallback(doc.Schema.Body)
	}
	if len(doc.ToolDefs) > 0 {
		for _, td := range doc.ToolDefs {
			tool := map[string]any{
				"type": "function",
				"name": td.Name,
			}
			if td.Description != "" {
				tool["description"] = strings.TrimSpace(td.Description)
			}
			if td.Description != "" {
				tool["parameters"] = parseJSONFallback(td.Description)
			}
			if len(td.Attrs) > 0 {
				tool["attrs"] = attrsToMap(td.Attrs)
			}
			out.Tools = append(out.Tools, tool)
		}
	}
	if len(doc.Runtimes) > 0 {
		rt := make(map[string]any)
		for _, attr := range doc.Runtimes[0].Attrs {
			key := strings.ReplaceAll(attr.Name.Local, "-", "_")
			rt[key] = attr.Value
		}
		out.Runtime = rt
	}
	return out, nil
}

func convertOpenAIChat(doc Document, opts ConvertOptions) (map[string]any, error) {
	result := map[string]any{}
	var messages []map[string]any
	for _, el := range doc.resolveOrder() {
		switch el.Type {
		case ElementHumanMsg, ElementAssistantMsg, ElementSystemMsg:
			payload := doc.Messages[el.Index]
			role := roleToOpenAI(payload.Role)
			content := strings.TrimSpace(payload.Body)
			messages = append(messages, map[string]any{
				"role":    role,
				"content": content,
			})
		case ElementToolRequest:
			tr := doc.ToolReqs[el.Index]
			toolCall := map[string]any{
				"id":   tr.ID,
				"type": "function",
				"function": map[string]any{
					"name": tr.Name,
					"arguments": strings.TrimSpace(strings.TrimPrefix(strings.TrimSuffix(tr.Parameters, "}}"), "{{")),
				},
			}
			messages = append(messages, map[string]any{
				"role":       "assistant",
				"tool_calls": []any{toolCall},
			})
		case ElementToolResponse:
			resp := doc.ToolResps[el.Index]
			messages = append(messages, map[string]any{
				"role":        "tool",
				"content":     strings.TrimSpace(resp.Body),
				"tool_call_id": resp.ID,
				"name":        resp.Name,
			})
		case ElementImage:
			im := doc.Images[el.Index]
			imgPart, err := buildImagePart(im, opts)
			if err != nil {
				return nil, err
			}
			messages = append(messages, map[string]any{
				"role": "user",
				"content": []any{
					map[string]any{"type": "text", "text": im.Alt},
					map[string]any{"type": "image_url", "image_url": map[string]any{"url": "data:" + imgPart["type"].(string) + ";base64," + imgPart["base64"].(string)}},
				},
			})
		}
	}
	result["messages"] = messages
	if doc.hasSchema() {
		result["response_format"] = map[string]any{
			"type": "json_schema",
			"json_schema": map[string]any{
				"name":   "schema",
				"schema": parseJSONFallback(doc.Schema.Body),
				"strict": true,
			},
		}
	}
	if len(doc.Runtimes) > 0 {
		for _, attr := range doc.Runtimes[0].Attrs {
			key := strings.ReplaceAll(attr.Name.Local, "-", "_")
			result[key] = attr.Value
		}
	}
	if len(doc.ToolDefs) > 0 {
		var tools []any
		for _, td := range doc.ToolDefs {
			tools = append(tools, map[string]any{
				"type": "function",
				"function": map[string]any{
					"name": td.Name,
					"description": strings.TrimSpace(td.Description),
					"parameters": parseJSONFallback(td.Description),
				},
			})
		}
		result["tools"] = tools
	}
	return result, nil
}

func convertLangChain(doc Document, opts ConvertOptions) (map[string]any, error) {
	var messages []map[string]any
	for _, el := range doc.resolveOrder() {
		switch el.Type {
		case ElementHumanMsg, ElementAssistantMsg, ElementSystemMsg:
			msg := doc.Messages[el.Index]
			messages = append(messages, map[string]any{
				"type": roleToLangChain(msg.Role),
				"data": map[string]any{"content": strings.TrimSpace(msg.Body)},
			})
		case ElementToolRequest:
			tr := doc.ToolReqs[el.Index]
			messages = append(messages, map[string]any{
				"type": "ai",
				"data": map[string]any{
					"tool_calls": []any{map[string]any{
						"id":   tr.ID,
						"name": tr.Name,
						"args": parseJSONFallback(strings.Trim(tr.Parameters, "{} ")),
					}},
				},
			})
		case ElementToolResponse:
			resp := doc.ToolResps[el.Index]
			messages = append(messages, map[string]any{
				"type": "tool",
				"data": map[string]any{
					"content":     strings.TrimSpace(resp.Body),
					"tool_call_id": resp.ID,
					"name":        resp.Name,
				},
			})
		case ElementImage:
			im := doc.Images[el.Index]
			part, err := buildImagePart(im, opts)
			if err != nil {
				return nil, err
			}
			messages = append(messages, map[string]any{
				"type": "human",
				"data": map[string]any{
					"content": []any{
						map[string]any{"type": "image", "source_type": "base64", "mime_type": part["type"], "data": part["base64"]},
					},
				},
			})
		}
	}
	out := map[string]any{"messages": messages}
	if doc.hasSchema() {
		out["schema"] = parseJSONFallback(doc.Schema.Body)
	}
	if len(doc.ToolDefs) > 0 {
		var tools []any
		for _, td := range doc.ToolDefs {
			tools = append(tools, map[string]any{
				"type": "function",
				"name": td.Name,
				"description": strings.TrimSpace(td.Description),
				"parameters": parseJSONFallback(td.Description),
			})
		}
		out["tools"] = tools
	}
	if len(doc.Runtimes) > 0 {
		rt := make(map[string]any)
		for _, attr := range doc.Runtimes[0].Attrs {
			rt[attr.Name.Local] = attr.Value
		}
		out["runtime"] = rt
	}
	return out, nil
}

func buildImagePart(im Image, opts ConvertOptions) (map[string]any, error) {
	var data string
	switch {
	case strings.HasPrefix(im.Src, "data:"):
		parts := strings.SplitN(im.Src, ",", 2)
		if len(parts) == 2 {
			data = parts[1]
		}
	case im.Src != "":
		src := im.Src
		if opts.BaseDir != "" && !filepath.IsAbs(src) {
			src = filepath.Join(opts.BaseDir, src)
		}
		bytes, err := os.ReadFile(src)
		if err != nil {
			return nil, fmt.Errorf("read image %s: %w", src, err)
		}
		data = base64.StdEncoding.EncodeToString(bytes)
	case im.Body != "":
		data = base64.StdEncoding.EncodeToString([]byte(im.Body))
	}
	mime := im.Syntax
	if mime == "" {
		mime = guessMime(im.Src)
	}
	if mime == "" {
		mime = "image/png"
	}
	return map[string]any{
		"type":   mime,
		"alt":    im.Alt,
		"base64": data,
	}, nil
}

func guessMime(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	}
	return ""
}

func parseJSONFallback(body string) any {
	var out any
	if err := json.Unmarshal([]byte(strings.TrimSpace(body)), &out); err != nil {
		return strings.TrimSpace(body)
	}
	return out
}

func attrsToMap(attrs []xml.Attr) map[string]string {
	res := make(map[string]string)
	for _, a := range attrs {
		res[a.Name.Local] = a.Value
	}
	return res
}

func roleToSpeaker(role string) string {
	switch role {
	case "assistant":
		return "assistant"
	case "system":
		return "system"
	default:
		return "human"
	}
}

func roleToOpenAI(role string) string {
	switch role {
	case "assistant":
		return "assistant"
	case "system":
		return "system"
	default:
		return "user"
	}
}

func roleToLangChain(role string) string {
	switch role {
	case "assistant":
		return "ai"
	case "system":
		return "system"
	default:
		return "human"
	}
}
