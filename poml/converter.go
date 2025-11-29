package poml

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"go.opentelemetry.io/otel/attribute"
)

// Format enumerates output conversion targets inspired by the Python SDK.
type Format string

const (
	FormatMessageDict Format = "message_dict"
	FormatDict        Format = "dict"
	FormatOpenAIChat  Format = "openai_chat"
	FormatLangChain   Format = "langchain"
	FormatPydantic    Format = "pydantic"
	FormatScene       Format = "scene"
	FormatSceneJSON   Format = "scenejson"
)

// ConvertOptions holds knobs for conversion (context, runtime flags, etc.).
// This will expand when format support is implemented.
type ConvertOptions struct {
	// Context may include auxiliary data (e.g., file paths) needed for conversions.
	Context map[string]any
	// BaseDir is used to resolve relative asset paths (e.g., <img src>).
	BaseDir string
	// AllowAbsImagePaths permits absolute image paths; defaults to false to avoid accidental file reads.
	AllowAbsImagePaths bool
	// MaxImageBytes caps bytes read before Base64 encoding; zero applies a default cap, negative disables the cap.
	MaxImageBytes int64
	// MaxMediaBytes caps bytes read for audio/video; zero applies a default cap, negative disables the cap.
	MaxMediaBytes int64
	// Trace configures OpenTelemetry spans for conversion; when empty, tracing is skipped.
	Trace TraceOptions
	// Extended toggles conversion of POML Extended constructs; off by default.
	Extended ExtendedMode
}

const defaultMaxImageBytes int64 = 10 << 20 // 10MB safeguard
const defaultMaxMediaBytes int64 = 10 << 20 // 10MB safeguard for audio/video

// ErrNotImplemented signals that a conversion target is not yet supported.
var ErrNotImplemented = errors.New("conversion not implemented")

// Convert transforms a parsed Document into the requested format.
func Convert(doc Document, format Format, opts ConvertOptions) (any, error) {
	switch format {
	case FormatMessageDict:
		return convertMessageDict(doc, opts)
	case FormatDict:
		return convertDict(doc, opts)
	case FormatPydantic:
		return convertPydantic(doc, opts)
	case FormatOpenAIChat:
		return convertOpenAIChat(doc, opts)
	case FormatLangChain:
		return convertLangChain(doc, opts)
	case FormatScene:
		scenes, err := diagramsToScenes(doc.Diagrams, defaultSceneExportOptions)
		if err != nil {
			return nil, err
		}
		switch len(scenes) {
		case 0:
			return doc.Scene(), nil
		case 1:
			return scenes[0], nil
		default:
			return scenes, nil
		}
	case FormatSceneJSON:
		scenes, err := diagramsToScenes(doc.Diagrams, defaultSceneExportOptions)
		if err != nil {
			return nil, err
		}
		switch len(scenes) {
		case 0:
			return encodeSceneJSON(doc.Scene())
		case 1:
			return encodeSceneJSON(scenes[0])
		default:
			return encodeScenesJSON(scenes)
		}
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

// ConvertWithTrace wraps Convert with an OpenTelemetry span. If opts.Trace is empty,
// ConvertWithTrace behaves like Convert.
func ConvertWithTrace(ctx context.Context, doc Document, format Format, opts ConvertOptions) (any, error) {
	if opts.Trace.skip() {
		return Convert(doc, format, opts)
	}
	_, span := opts.Trace.start(ctx, "poml.convert",
		attribute.String("poml.format", string(format)),
		attribute.String("poml.meta.id", strings.TrimSpace(doc.Meta.ID)),
	)
	defer span.End()
	out, err := Convert(doc, format, opts)
	if err != nil {
		span.RecordError(err)
	}
	return out, err
}

func diagramsToScenes(diagrams []Diagram, opts SceneExportOptions) ([]Scene, error) {
	if len(diagrams) == 0 {
		return nil, nil
	}
	out := make([]Scene, 0, len(diagrams))
	for _, d := range diagrams {
		scene, err := DiagramToSceneWithOptions(d, opts)
		if err != nil {
			return nil, err
		}
		out = append(out, scene)
	}
	return out, nil
}

type messageDict struct {
	Speaker string `json:"speaker"`
	Content any    `json:"content"`
}

func formatContentPart(name string, body string) map[string]any {
	return map[string]any{
		"type":    "format",
		"tag":     name,
		"content": strings.TrimSpace(body),
	}
}

func convertMessageDict(doc Document, opts ConvertOptions) ([]messageDict, error) {
	var msgs []messageDict
	for _, el := range doc.resolveOrder() {
		switch el.Type {
		case ElementHumanMsg, ElementAssistantMsg, ElementSystemMsg:
			payload := doc.Messages[el.Index]
			content := strings.TrimSpace(payload.Body)
			msgs = append(msgs, messageDict{Speaker: roleToSpeaker(payload.Role), Content: content})
		case ElementToolResult:
			payload := doc.ToolResults[el.Index]
			msgs = append(msgs, messageDict{Speaker: "tool", Content: strings.TrimSpace(payload.Body)})
		case ElementToolError:
			payload := doc.ToolErrors[el.Index]
			msgs = append(msgs, messageDict{Speaker: "tool", Content: map[string]any{"error": strings.TrimSpace(payload.Body), "name": payload.Name}})
		case ElementToolResponse:
			payload := doc.ToolResps[el.Index]
			msgs = append(msgs, messageDict{Speaker: "tool", Content: strings.TrimSpace(payload.Body)})
		case ElementHint, ElementExample:
			body := strings.TrimSpace(doc.elementBody(el))
			if body != "" {
				msgs = append(msgs, messageDict{Speaker: "human", Content: body})
			}
		case ElementContentPart:
			body := strings.TrimSpace(doc.elementBody(el))
			if body != "" {
				msgs = append(msgs, messageDict{Speaker: "human", Content: formatContentPart(el.Name, body)})
			}
		case ElementPersona:
			body := strings.TrimSpace(doc.Persona.Body)
			if body != "" {
				msgs = append(msgs, messageDict{Speaker: "human", Content: body})
			}
		case ElementObject:
			obj := doc.Objects[el.Index]
			msgs = append(msgs, messageDict{
				Speaker: "human",
				Content: map[string]any{
					"type":   "object",
					"data":   obj.Data,
					"syntax": obj.Syntax,
					"body":   strings.TrimSpace(obj.Body),
				},
			})
		case ElementImage:
			im := doc.Images[el.Index]
			part, err := buildImagePart(im, opts)
			if err != nil {
				return nil, err
			}
			msgs = append(msgs, messageDict{Speaker: "human", Content: part})
		case ElementAudio:
			au := doc.Audios[el.Index]
			part, err := buildMediaPart(au, opts)
			if err != nil {
				return nil, err
			}
			msgs = append(msgs, messageDict{Speaker: "human", Content: part})
		case ElementVideo:
			vd := doc.Videos[el.Index]
			part, err := buildMediaPart(vd, opts)
			if err != nil {
				return nil, err
			}
			msgs = append(msgs, messageDict{Speaker: "human", Content: part})
		case ElementOp:
			if opts.Extended == ExtendedOff {
				continue
			}
			op := doc.Ops[el.Index]
			name := strings.TrimSpace(op.Name)
			if name == "" {
				name = el.Name
			}
			msgs = append(msgs, messageDict{
				Speaker: "human",
				Content: map[string]any{
					"type":   "op",
					"name":   name,
					"kind":   strings.TrimSpace(op.Kind),
					"args":   parseJSONFallback(op.Args),
					"body":   strings.TrimSpace(op.Body),
					"attrs":  attrsToMap(op.Attrs),
					"tag":    el.Name,
					"source": "extended",
				},
			})
		case ElementFigure:
			if opts.Extended == ExtendedOff {
				continue
			}
			part, err := buildFigurePart(doc.Figures[el.Index], opts)
			if err != nil {
				return nil, err
			}
			msgs = append(msgs, messageDict{Speaker: "human", Content: part})
		case ElementText:
			if opts.Extended == ExtendedOff {
				continue
			}
			body := strings.TrimSpace(doc.elementBody(el))
			if body != "" {
				msgs = append(msgs, messageDict{Speaker: "human", Content: body})
			}
		case ElementUnknown:
			if opts.Extended != ExtendedOff {
				msgs = append(msgs, messageDict{
					Speaker: "human",
					Content: map[string]any{
						"type": "unknown",
						"name": el.Name,
						"raw":  strings.TrimSpace(el.RawXML),
					},
				})
			}
		}
	}
	return msgs, nil
}

type dictOutput struct {
	Messages []messageDict  `json:"messages"`
	Schema   any            `json:"schema,omitempty"`
	Tools    []any          `json:"tools,omitempty"`
	Runtime  map[string]any `json:"runtime,omitempty"`
	Media    []any          `json:"media,omitempty"`
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
			out.Tools = append(out.Tools, buildFlatToolDefinition(td))
		}
	}
	if rt := collectRuntime(doc); rt != nil {
		out.Runtime = rt
	}
	return out, nil
}

// convertPydantic aligns with Python SDK pydantic export (mirrors dict structure with consistent field names).
func convertPydantic(doc Document, opts ConvertOptions) (dictOutput, error) {
	out, err := convertDict(doc, opts)
	if err != nil {
		return dictOutput{}, err
	}
	if media := collectMedia(doc, opts); len(media) > 0 {
		out.Media = media
	}
	return out, nil
}

func collectMedia(doc Document, opts ConvertOptions) []any {
	var media []any
	for _, el := range doc.resolveOrder() {
		switch el.Type {
		case ElementImage:
			if part, err := buildImagePart(doc.Images[el.Index], opts); err == nil {
				media = append(media, part)
			}
		case ElementAudio:
			if part, err := buildMediaPart(doc.Audios[el.Index], opts); err == nil {
				media = append(media, part)
			}
		case ElementVideo:
			if part, err := buildMediaPart(doc.Videos[el.Index], opts); err == nil {
				media = append(media, part)
			}
		}
	}
	return media
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
		case ElementHint, ElementExample, ElementContentPart:
			body := strings.TrimSpace(doc.elementBody(el))
			if body != "" {
				messages = append(messages, map[string]any{
					"role":    "user",
					"content": body,
				})
			}
		case ElementPersona:
			body := strings.TrimSpace(doc.Persona.Body)
			if body != "" {
				messages = append(messages, map[string]any{
					"role":    "user",
					"content": body,
				})
			}
		case ElementObject:
			obj := doc.Objects[el.Index]
			content := strings.TrimSpace(obj.Body)
			if content == "" {
				content = strings.TrimSpace(obj.Data)
			}
			messages = append(messages, map[string]any{
				"role":    "user",
				"content": content,
			})
		case ElementToolRequest:
			tr := doc.ToolReqs[el.Index]
			toolCall := map[string]any{
				"id":   tr.ID,
				"type": "function",
				"function": map[string]any{
					"name":      tr.Name,
					"arguments": normalizeToolArgsJSON(tr.Parameters),
				},
			}
			if len(messages) > 0 {
				last := messages[len(messages)-1]
				if last["role"] == "assistant" {
					existing, ok := last["tool_calls"].([]any)
					if !ok {
						existing = nil
					}
					last["tool_calls"] = append(existing, toolCall)
					messages[len(messages)-1] = last
					continue
				}
			}
			messages = append(messages, map[string]any{
				"role":       "assistant",
				"tool_calls": []any{toolCall},
			})
		case ElementToolResponse:
			resp := doc.ToolResps[el.Index]
			messages = append(messages, map[string]any{
				"role":         "tool",
				"content":      strings.TrimSpace(resp.Body),
				"tool_call_id": resp.ID,
				"name":         resp.Name,
			})
		case ElementToolResult:
			resp := doc.ToolResults[el.Index]
			messages = append(messages, map[string]any{
				"role":         "tool",
				"content":      strings.TrimSpace(resp.Body),
				"tool_call_id": resp.ID,
				"name":         resp.Name,
				"type":         "result",
			})
		case ElementToolError:
			resp := doc.ToolErrors[el.Index]
			messages = append(messages, map[string]any{
				"role":         "tool",
				"content":      strings.TrimSpace(resp.Body),
				"tool_call_id": resp.ID,
				"name":         resp.Name,
				"type":         "error",
			})
		case ElementOp:
			if opts.Extended == ExtendedOff {
				continue
			}
			op := doc.Ops[el.Index]
			name := strings.TrimSpace(op.Name)
			if name == "" {
				name = el.Name
			}
			meta := map[string]any{}
			if op.Kind != "" {
				meta["kind"] = strings.TrimSpace(op.Kind)
			}
			if op.Args != "" {
				meta["args"] = parseJSONFallback(op.Args)
			}
			if len(attrsToMap(op.Attrs)) > 0 {
				meta["attrs"] = attrsToMap(op.Attrs)
			}
			if el.Name != "" {
				meta["tag"] = el.Name
			}
			text := strings.TrimSpace(op.Body)
			if text == "" {
				text = fmt.Sprintf("[op:%s]", name)
			} else {
				text = fmt.Sprintf("[op:%s] %s", name, text)
			}
			content := map[string]any{"type": "text", "text": text}
			if len(meta) > 0 {
				content["metadata"] = meta
			}
			messages = append(messages, map[string]any{
				"role":    "user",
				"content": []any{content},
			})
		case ElementUnknown:
			if opts.Extended != ExtendedOff {
				raw := strings.TrimSpace(el.RawXML)
				if raw == "" {
					raw = el.Name
				}
				messages = append(messages, map[string]any{
					"role": "user",
					"content": []any{map[string]any{
						"type": "text",
						"text": fmt.Sprintf("[unknown:%s] %s", el.Name, raw),
					}},
				})
			}
		case ElementText:
			if opts.Extended == ExtendedOff {
				continue
			}
			body := strings.TrimSpace(doc.elementBody(el))
			if body != "" {
				messages = append(messages, map[string]any{
					"role":    "user",
					"content": body,
				})
			}
		case ElementAudio:
			au := doc.Audios[el.Index]
			part, err := buildMediaPart(au, opts)
			if err != nil {
				return nil, err
			}
			messages = append(messages, map[string]any{
				"role": "user",
				"content": []any{
					map[string]any{"type": "input_audio", "audio": part},
				},
			})
		case ElementVideo:
			vd := doc.Videos[el.Index]
			part, err := buildMediaPart(vd, opts)
			if err != nil {
				return nil, err
			}
			messages = append(messages, map[string]any{
				"role": "user",
				"content": []any{
					map[string]any{"type": "input_video", "video": part},
				},
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
		case ElementFigure:
			if opts.Extended == ExtendedOff {
				continue
			}
			fig := doc.Figures[el.Index]
			part, err := buildFigurePart(fig, opts)
			if err != nil {
				return nil, err
			}
			url := "data:" + part["mime"].(string) + ";base64," + part["base64"].(string)
			content := []any{
				map[string]any{"type": "text", "text": strings.TrimSpace(fig.Alt)},
				map[string]any{"type": "image_url", "image_url": map[string]any{"url": url}},
			}
			if strings.TrimSpace(fig.Alt) == "" {
				content = content[1:]
			}
			messages = append(messages, map[string]any{
				"role":    "user",
				"content": content,
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
	if rt := collectRuntime(doc); rt != nil {
		for k, v := range rt {
			result[k] = v
		}
	}
	if len(doc.ToolDefs) > 0 {
		var tools []any
		for _, td := range doc.ToolDefs {
			tools = append(tools, buildOpenAIToolDefinition(td))
		}
		result["tools"] = tools
	}
	return result, nil
}

func normalizeToolArgs(raw string) string {
	body := strings.TrimSpace(raw)
	if strings.HasPrefix(body, "{{") && strings.HasSuffix(body, "}}") {
		body = strings.TrimSpace(strings.TrimPrefix(strings.TrimSuffix(body, "}}"), "{{"))
	}
	return body
}

func normalizeToolArgsJSON(raw string) string {
	body := normalizeToolArgs(raw)
	if val, ok := parseLooseJSONValue(body); ok {
		b, err := json.Marshal(val)
		if err == nil {
			return string(b)
		}
	}
	return body
}

var bareKeyRe = regexp.MustCompile(`([{\s,])([A-Za-z0-9_\-]+)\s*:`)

func parseLooseJSON(body string) any {
	if val, ok := parseLooseJSONValue(body); ok {
		return val
	}
	return body
}

func parseLooseJSONValue(body string) (any, bool) {
	body = strings.TrimSpace(body)
	if body == "" {
		return nil, false
	}
	var val any
	if err := json.Unmarshal([]byte(body), &val); err == nil {
		return val, true
	}
	// Normalize single quotes and bare keys.
	body = strings.ReplaceAll(body, `'`, `"`)
	body = bareKeyRe.ReplaceAllString(body, `$1"$2":`)
	if err := json.Unmarshal([]byte(body), &val); err == nil {
		return val, true
	}
	return nil, false
}

func normalizeRuntimeKey(key string) string {
	key = strings.ReplaceAll(key, "-", "_")
	// camelCase to snake_case (basic)
	var out []rune
	for i, r := range key {
		if i > 0 && r >= 'A' && r <= 'Z' {
			out = append(out, '_', r+('a'-'A'))
		} else {
			out = append(out, r)
		}
	}
	return string(out)
}

func parseRuntimeValue(val string) any {
	val = strings.TrimSpace(val)
	if val == "" {
		return val
	}
	var num float64
	if err := json.Unmarshal([]byte(val), &num); err == nil {
		// preserve ints when applicable
		if float64(int(num)) == num {
			return int(num)
		}
		return num
	}
	// try as array/object
	var anyVal any
	if err := json.Unmarshal([]byte(val), &anyVal); err == nil {
		return anyVal
	}
	return val
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
		case ElementHint, ElementExample, ElementContentPart:
			body := strings.TrimSpace(doc.elementBody(el))
			if body != "" {
				messages = append(messages, map[string]any{
					"type": "human",
					"data": map[string]any{"content": body},
				})
			}
		case ElementPersona:
			body := strings.TrimSpace(doc.Persona.Body)
			if body != "" {
				messages = append(messages, map[string]any{
					"type": "human",
					"data": map[string]any{"content": body},
				})
			}
		case ElementText:
			if opts.Extended == ExtendedOff {
				continue
			}
			body := strings.TrimSpace(doc.elementBody(el))
			if body != "" {
				messages = append(messages, map[string]any{
					"type": "human",
					"data": map[string]any{"content": body},
				})
			}
		case ElementOp:
			if opts.Extended == ExtendedOff {
				continue
			}
			op := doc.Ops[el.Index]
			name := strings.TrimSpace(op.Name)
			if name == "" {
				name = el.Name
			}
			payload := map[string]any{
				"type":  "op",
				"name":  name,
				"body":  strings.TrimSpace(op.Body),
				"attrs": attrsToMap(op.Attrs),
				"tag":   el.Name,
			}
			if op.Kind != "" {
				payload["kind"] = strings.TrimSpace(op.Kind)
			}
			if op.Args != "" {
				payload["args"] = parseJSONFallback(op.Args)
			}
			messages = append(messages, map[string]any{
				"type": "human",
				"data": map[string]any{
					"content": []any{payload},
				},
			})
		case ElementAudio:
			au := doc.Audios[el.Index]
			part, err := buildMediaPart(au, opts)
			if err != nil {
				return nil, err
			}
			messages = append(messages, map[string]any{
				"type": "human",
				"data": map[string]any{
					"content": []any{
						map[string]any{"type": "audio", "source_type": "base64", "mime_type": part["type"], "data": part["base64"]},
					},
				},
			})
		case ElementVideo:
			vd := doc.Videos[el.Index]
			part, err := buildMediaPart(vd, opts)
			if err != nil {
				return nil, err
			}
			messages = append(messages, map[string]any{
				"type": "human",
				"data": map[string]any{
					"content": []any{
						map[string]any{"type": "video", "source_type": "base64", "mime_type": part["type"], "data": part["base64"]},
					},
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
		case ElementFigure:
			if opts.Extended == ExtendedOff {
				continue
			}
			fig := doc.Figures[el.Index]
			part, err := buildFigurePart(fig, opts)
			if err != nil {
				return nil, err
			}
			messages = append(messages, map[string]any{
				"type": "human",
				"data": map[string]any{
					"content": []any{
						map[string]any{"type": "figure", "source_type": "base64", "mime_type": part["mime"], "data": part["base64"], "alt": fig.Alt, "tag": el.Name},
					},
				},
			})
		case ElementObject:
			obj := doc.Objects[el.Index]
			content := strings.TrimSpace(obj.Body)
			if content == "" {
				content = strings.TrimSpace(obj.Data)
			}
			messages = append(messages, map[string]any{
				"type": "human",
				"data": map[string]any{"content": content},
			})
		case ElementToolRequest:
			tr := doc.ToolReqs[el.Index]
			call := map[string]any{
				"id":   tr.ID,
				"name": tr.Name,
				"args": parseLooseJSON(normalizeToolArgs(tr.Parameters)),
			}
			if len(messages) > 0 && messages[len(messages)-1]["type"] == "ai" {
				last := messages[len(messages)-1]
				data := last["data"].(map[string]any)
				existing, _ := data["tool_calls"].([]any)
				data["tool_calls"] = append(existing, call)
				last["data"] = data
				messages[len(messages)-1] = last
			} else {
				messages = append(messages, map[string]any{
					"type": "ai",
					"data": map[string]any{
						"tool_calls": []any{call},
					},
				})
			}
		case ElementToolResponse:
			resp := doc.ToolResps[el.Index]
			messages = append(messages, map[string]any{
				"type": "tool",
				"data": map[string]any{
					"content":      strings.TrimSpace(resp.Body),
					"tool_call_id": resp.ID,
					"name":         resp.Name,
				},
			})
		case ElementToolResult:
			resp := doc.ToolResults[el.Index]
			messages = append(messages, map[string]any{
				"type": "tool",
				"data": map[string]any{
					"content":      strings.TrimSpace(resp.Body),
					"tool_call_id": resp.ID,
					"name":         resp.Name,
					"result":       true,
				},
			})
		case ElementToolError:
			resp := doc.ToolErrors[el.Index]
			messages = append(messages, map[string]any{
				"type": "tool",
				"data": map[string]any{
					"content":      strings.TrimSpace(resp.Body),
					"tool_call_id": resp.ID,
					"name":         resp.Name,
					"error":        true,
				},
			})
		case ElementUnknown:
			if opts.Extended != ExtendedOff {
				raw := strings.TrimSpace(el.RawXML)
				if raw == "" {
					raw = el.Name
				}
				messages = append(messages, map[string]any{
					"type": "human",
					"data": map[string]any{
						"content": []any{
							map[string]any{
								"type": "text",
								"text": fmt.Sprintf("[unknown:%s] %s", el.Name, raw),
							},
						},
					},
				})
			}
		}
	}
	out := map[string]any{"messages": messages}
	if doc.hasSchema() {
		out["schema"] = parseJSONFallback(doc.Schema.Body)
	}
	if len(doc.ToolDefs) > 0 {
		var tools []any
		for _, td := range doc.ToolDefs {
			tools = append(tools, buildFlatToolDefinition(td))
		}
		out["tools"] = tools
	}
	if rt := collectRuntime(doc); rt != nil {
		out["runtime"] = rt
	}
	return out, nil
}

func collectRuntime(doc Document) map[string]any {
	if len(doc.Runtimes) == 0 {
		return nil
	}
	rt := make(map[string]any)
	for _, runtime := range doc.Runtimes {
		for _, attr := range runtime.Attrs {
			key := normalizeRuntimeKey(attr.Name.Local)
			rt[key] = parseRuntimeValue(attr.Value)
		}
	}
	if len(rt) == 0 {
		return nil
	}
	return rt
}

func buildFigurePart(fig ExtendedFigure, opts ConvertOptions) (map[string]any, error) {
	limit := opts.MaxMediaBytes
	if limit == 0 {
		limit = defaultMaxMediaBytes
	}
	var data string
	checkLimit := func(raw string, label string) error {
		if limit <= 0 {
			return nil
		}
		size := int64(base64.StdEncoding.DecodedLen(len(raw)))
		return enforceByteLimit(size, limit, label)
	}
	switch {
	case strings.HasPrefix(fig.Src, "data:"):
		parts := strings.SplitN(fig.Src, ",", 2)
		if len(parts) == 2 {
			payload := parts[1]
			if err := checkLimit(payload, "data URI figure"); err != nil {
				return nil, err
			}
			data = payload
		}
	case fig.Src != "":
		src, err := resolveMediaPath(fig.Src, opts)
		if err != nil {
			return nil, err
		}
		bytes, err := readFileWithLimit(src, limit)
		if err != nil {
			return nil, fmt.Errorf("read figure %s: %w", src, err)
		}
		data = base64.StdEncoding.EncodeToString(bytes)
	case fig.Body != "":
		body := []byte(fig.Body)
		if err := enforceByteLimit(int64(len(body)), limit, "inline figure body"); err != nil {
			return nil, err
		}
		data = base64.StdEncoding.EncodeToString(body)
	}
	mime := fig.Syntax
	if mime == "" {
		mime = guessMime(fig.Src)
	}
	if mime == "" {
		mime = "application/octet-stream"
	}
	return map[string]any{
		"type":      mime,
		"mime":      mime,
		"mime_type": mime,
		"alt":       fig.Alt,
		"base64":    data,
		"source":    "base64",
		"syntax":    fig.Syntax,
		"data":      data,
	}, nil
}

func buildImagePart(im Image, opts ConvertOptions) (map[string]any, error) {
	limit := opts.MaxImageBytes
	if limit == 0 {
		limit = defaultMaxImageBytes
	}
	var data string
	checkLimit := func(raw string, label string) error {
		if limit <= 0 {
			return nil
		}
		size := int64(base64.StdEncoding.DecodedLen(len(raw)))
		return enforceByteLimit(size, limit, label)
	}
	switch {
	case strings.HasPrefix(im.Src, "data:"):
		parts := strings.SplitN(im.Src, ",", 2)
		if len(parts) == 2 {
			payload := parts[1]
			if err := checkLimit(payload, "data URI image"); err != nil {
				return nil, err
			}
			data = payload
		}
	case im.Src != "":
		src, err := resolveImagePath(im.Src, opts)
		if err != nil {
			return nil, err
		}
		bytes, err := readFileWithLimit(src, limit)
		if err != nil {
			return nil, fmt.Errorf("read image %s: %w", src, err)
		}
		data = base64.StdEncoding.EncodeToString(bytes)
	case im.Body != "":
		body := []byte(im.Body)
		if err := enforceByteLimit(int64(len(body)), limit, "inline image body"); err != nil {
			return nil, err
		}
		data = base64.StdEncoding.EncodeToString(body)
	}
	mime := im.Syntax
	if mime == "" {
		mime = guessMime(im.Src)
	}
	if mime == "" {
		mime = "image/png"
	}
	return map[string]any{
		"type":      mime,
		"mime":      mime,
		"mime_type": mime,
		"alt":       im.Alt,
		"base64":    data,
		"source":    "base64",
		"syntax":    im.Syntax,
		"data":      data,
	}, nil
}

func buildMediaPart(m Media, opts ConvertOptions) (map[string]any, error) {
	limit := opts.MaxMediaBytes
	if limit == 0 {
		limit = defaultMaxMediaBytes
	}
	var data string
	checkLimit := func(raw string, label string) error {
		if limit <= 0 {
			return nil
		}
		size := int64(base64.StdEncoding.DecodedLen(len(raw)))
		return enforceByteLimit(size, limit, label)
	}
	switch {
	case strings.HasPrefix(m.Src, "data:"):
		parts := strings.SplitN(m.Src, ",", 2)
		if len(parts) == 2 {
			payload := parts[1]
			if err := checkLimit(payload, "data URI media"); err != nil {
				return nil, err
			}
			data = payload
		}
	case m.Src != "":
		src, err := resolveMediaPath(m.Src, opts)
		if err != nil {
			return nil, err
		}
		bytes, err := readFileWithLimit(src, limit)
		if err != nil {
			return nil, fmt.Errorf("read media %s: %w", src, err)
		}
		data = base64.StdEncoding.EncodeToString(bytes)
	case m.Body != "":
		body := []byte(m.Body)
		if err := enforceByteLimit(int64(len(body)), limit, "inline media body"); err != nil {
			return nil, err
		}
		data = base64.StdEncoding.EncodeToString(body)
	}
	mime := m.Syntax
	if mime == "" {
		mime = guessMediaMime(m.Src)
	}
	return map[string]any{
		"type":      mime,
		"mime":      mime,
		"mime_type": mime,
		"alt":       m.Alt,
		"base64":    data,
		"source":    "base64",
		"syntax":    m.Syntax,
		"data":      data,
	}, nil
}

func resolveImagePath(raw string, opts ConvertOptions) (string, error) {
	cleaned := filepath.Clean(raw)
	base := strings.TrimSpace(opts.BaseDir)
	if base != "" {
		base = strings.TrimSuffix(filepath.Clean(base), string(filepath.Separator))
		resolvedBase, err := filepath.EvalSymlinks(base)
		if err != nil {
			return "", fmt.Errorf("resolve base dir %s: %w", opts.BaseDir, err)
		}
		base = resolvedBase
	}
	ensureContained := func(candidate string) error {
		if base == "" {
			return nil
		}
		rel, err := filepath.Rel(base, candidate)
		if err != nil {
			return fmt.Errorf("image path %s escapes BaseDir %s", raw, opts.BaseDir)
		}
		if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return fmt.Errorf("image path %s escapes BaseDir %s", raw, opts.BaseDir)
		}
		return nil
	}
	if filepath.IsAbs(cleaned) {
		candidate, err := filepath.EvalSymlinks(cleaned)
		if err != nil {
			return "", fmt.Errorf("resolve image path %s: %w", raw, err)
		}
		if base == "" && !opts.AllowAbsImagePaths {
			return "", fmt.Errorf("absolute image path %s disallowed without AllowAbsImagePaths", raw)
		}
		if err := ensureContained(candidate); err != nil {
			return "", err
		}
		return candidate, nil
	}
	if base == "" {
		return cleaned, nil
	}
	candidate, err := filepath.EvalSymlinks(filepath.Join(base, cleaned))
	if err != nil {
		return "", fmt.Errorf("resolve image path %s: %w", raw, err)
	}
	if err := ensureContained(candidate); err != nil {
		return "", err
	}
	return candidate, nil
}

func resolveMediaPath(raw string, opts ConvertOptions) (string, error) {
	return resolveImagePath(raw, opts)
}

func readFileWithLimit(path string, limit int64) ([]byte, error) {
	if limit <= 0 {
		return os.ReadFile(path)
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = f.Close()
	}()
	info, err := f.Stat()
	if err == nil && info.Size() > limit {
		return nil, fmt.Errorf("file %s exceeds max size %d bytes", path, limit)
	}
	r := io.LimitReader(f, limit+1)
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("file %s exceeds max size %d bytes", path, limit)
	}
	return data, nil
}

func enforceByteLimit(size int64, limit int64, label string) error {
	if limit > 0 && size > limit {
		return fmt.Errorf("%s exceeds max size %d bytes", label, limit)
	}
	return nil
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
	case ".svg":
		return "image/svg+xml"
	case ".webp":
		return "image/webp"
	case ".tif", ".tiff":
		return "image/tiff"
	case ".heic":
		return "image/heic"
	case ".heif":
		return "image/heif"
	case ".avif":
		return "image/avif"
	}
	return ""
}

func guessMediaMime(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".mp3":
		return "audio/mpeg"
	case ".wav":
		return "audio/wav"
	case ".ogg":
		return "audio/ogg"
	case ".opus":
		return "audio/ogg; codecs=opus"
	case ".flac":
		return "audio/flac"
	case ".aac":
		return "audio/aac"
	case ".m4a":
		return "audio/mp4"
	case ".mp4":
		return "video/mp4"
	case ".mov":
		return "video/quicktime"
	case ".webm":
		return "video/webm"
	case ".m4v":
		return "video/x-m4v"
	case ".mpeg", ".mpg":
		return "video/mpeg"
	case ".avi":
		return "video/x-msvideo"
	case ".mkv":
		return "video/x-matroska"
	case ".3gp":
		return "video/3gpp"
	}
	return "application/octet-stream"
}

func parseJSONFallback(body string) any {
	var out any
	if err := json.Unmarshal([]byte(strings.TrimSpace(body)), &out); err != nil {
		return strings.TrimSpace(body)
	}
	return out
}

func parseJSONStrict(body string) (any, bool) {
	var out any
	if err := json.Unmarshal([]byte(strings.TrimSpace(body)), &out); err != nil {
		return nil, false
	}
	return out, true
}

func parseJSONIfStruct(body string) (any, bool) {
	val := parseJSONFallback(body)
	switch val.(type) {
	case map[string]any, []any:
		return val, true
	default:
		return nil, false
	}
}

func stripCDATA(body string) string {
	if strings.HasPrefix(body, "<![CDATA[") && strings.HasSuffix(body, "]]>") {
		body = strings.TrimPrefix(body, "<![CDATA[")
		body = strings.TrimSuffix(body, "]]>")
	}
	return body
}

// elementBody returns the inner body for container-like tags, falling back to known fields.
func (d Document) elementBody(el Element) string {
	switch el.Type {
	case ElementHint:
		if el.Index >= 0 && el.Index < len(d.Hints) {
			return d.Hints[el.Index].Body
		}
	case ElementExample:
		if el.Index >= 0 && el.Index < len(d.Examples) {
			return d.Examples[el.Index].Body
		}
	case ElementContentPart:
		if el.Index >= 0 && el.Index < len(d.ContentParts) {
			return d.ContentParts[el.Index].Body
		}
	case ElementPersona:
		return d.Persona.Body
	case ElementRole:
		return d.Role.Body
	case ElementTask:
		if el.Index >= 0 && el.Index < len(d.Tasks) {
			return d.Tasks[el.Index].Body
		}
	case ElementText:
		if el.Index >= 0 && el.Index < len(d.Texts) {
			return d.Texts[el.Index].Body
		}
	case ElementOp:
		if el.Index >= 0 && el.Index < len(d.Ops) {
			return d.Ops[el.Index].Body
		}
	case ElementFigure:
		if el.Index >= 0 && el.Index < len(d.Figures) {
			return d.Figures[el.Index].Body
		}
	}
	return ""
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

func buildFlatToolDefinition(td ToolDefinition) map[string]any {
	desc := stripCDATA(strings.TrimSpace(td.Description))
	body := stripCDATA(strings.TrimSpace(td.Body))
	if desc == "" {
		desc = body
	}
	tool := map[string]any{
		"type": "function",
		"name": td.Name,
	}
	if desc != "" {
		tool["description"] = desc
	}
	if params, ok := parseJSONIfStruct(body); ok {
		tool["parameters"] = params
	}
	if len(td.Attrs) > 0 {
		tool["attrs"] = attrsToMap(td.Attrs)
	}
	return tool
}

func buildOpenAIToolDefinition(td ToolDefinition) map[string]any {
	desc := stripCDATA(strings.TrimSpace(td.Description))
	body := stripCDATA(strings.TrimSpace(td.Body))
	if desc == "" {
		desc = body
	}
	fn := map[string]any{
		"name": td.Name,
	}
	if desc != "" {
		fn["description"] = desc
	}
	if params, ok := parseJSONIfStruct(body); ok {
		fn["parameters"] = params
	}
	if len(td.Attrs) > 0 {
		fn["attrs"] = attrsToMap(td.Attrs)
	}
	return map[string]any{
		"type":     "function",
		"function": fn,
	}
}

// ImageFromBase64 builds an <img> node backed by a data URI.
func ImageFromBase64(data string, mime string, alt string) Image {
	if mime == "" {
		mime = "application/octet-stream"
	}
	return Image{
		Src:    "data:" + mime + ";base64," + data,
		Alt:    alt,
		Syntax: mime,
	}
}

// ImageFromBytes builds an <img> node from raw bytes.
func ImageFromBytes(raw []byte, mime string, alt string) Image {
	return ImageFromBase64(base64.StdEncoding.EncodeToString(raw), mime, alt)
}

// ImageFromFile reads a local file and builds a data URI image.
func ImageFromFile(path string, mime string, alt string) (Image, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Image{}, err
	}
	if mime == "" {
		mime = guessMime(path)
	}
	if mime == "" {
		mime = "application/octet-stream"
	}
	return ImageFromBytes(raw, mime, alt), nil
}
