package poml

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"html"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/atlas-foundry/poml-go-sdk/poml/stylesheet"
	"github.com/atlas-foundry/poml-go-sdk/poml/template"
	"github.com/atlas-foundry/poml-go-sdk/poml/token"
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

	// TemplateVars provides variables for template interpolation ({{ var }}).
	// When non-nil, template expansion is enabled.
	TemplateVars map[string]any
	// ExpandTemplates enables processing of <let>, if/for attributes, and {{ }} interpolation.
	ExpandTemplates bool
	// ApplyStylesheet enables stylesheet application from <style> elements.
	ApplyStylesheet bool
	// EnforceLimits enables charLimit/tokenLimit truncation based on priority.
	EnforceLimits bool
	// MaxIncludeDepth limits recursive <include> depth (default 10).
	MaxIncludeDepth int
	// Components specifies which components to include/exclude.
	// Format: "component1,+component2,-component3"
	// Use "-component" to disable, "component" or "+component" to enable.
	Components string
	// EnforceVersions validates minVersion/maxVersion constraints during conversion.
	// If enabled and version constraints are violated, conversion returns an error.
	EnforceVersions bool
}

const defaultMaxImageBytes int64 = 10 << 20 // 10MB safeguard
const defaultMaxMediaBytes int64 = 10 << 20 // 10MB safeguard for audio/video

// ErrNotImplemented signals that a conversion target is not yet supported.
var ErrNotImplemented = errors.New("conversion not implemented")

// Convert transforms a parsed Document into the requested format.
func Convert(doc Document, format Format, opts ConvertOptions) (any, error) {
	// Check version constraints if enforcement is enabled
	if opts.EnforceVersions {
		if err := CheckVersionConstraint(SpecVersion, doc.Meta.MinVersion, doc.Meta.MaxVersion); err != nil {
			return nil, fmt.Errorf("version constraint: %w", err)
		}
	}

	// Preprocess: expand templates, apply stylesheet, enforce limits
	expanded, err := expandDocument(doc, opts)
	if err != nil {
		return nil, fmt.Errorf("template expansion: %w", err)
	}
	doc = expanded

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
		attribute.String("poml.meta.version", strings.TrimSpace(doc.Meta.Version)),
		attribute.String("poml.meta.owner", strings.TrimSpace(doc.Meta.Owner)),
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
		"type":      "format",
		"tag":       name,
		"content":   strings.TrimSpace(body),
		"raw":       body,
		"syntax":    "poml",
		"semantics": "inline",
	}
}

// extractAttr attempts to grab a simple attribute from raw XML (best-effort for unknown tags).
func extractAttr(raw string, name string) string {
	pat := name + "=\""
	idx := strings.Index(raw, pat)
	if idx == -1 {
		return ""
	}
	rest := raw[idx+len(pat):]
	if end := strings.Index(rest, "\""); end != -1 {
		return rest[:end]
	}
	return ""
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
			if opts.Extended == ExtendedOff {
				continue
			}
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
		case ElementTable:
			tbl := doc.Tables[el.Index]
			content := formatTableContent(tbl)
			if content != "" {
				msgs = append(msgs, messageDict{Speaker: "human", Content: content})
			}
		case ElementFolder:
			folder := doc.Folders[el.Index]
			content := formatFolderContent(folder, opts.BaseDir)
			if content != "" {
				msgs = append(msgs, messageDict{Speaker: "human", Content: content})
			}
		case ElementWebpage:
			// Webpage content is not fetched at convert time for security
			// Just include the URL reference
			wp := doc.Webpages[el.Index]
			msgs = append(msgs, messageDict{
				Speaker: "human",
				Content: map[string]any{
					"type":     "webpage",
					"url":      wp.URL,
					"selector": wp.Selector,
				},
			})
		case ElementConversation:
			conv := doc.Conversations[el.Index]
			content := formatConversationContent(conv)
			if content != "" {
				msgs = append(msgs, messageDict{Speaker: "human", Content: content})
			}
		case ElementHeader:
			if el.Index >= 0 && el.Index < len(doc.Headers) {
				h := doc.Headers[el.Index]
				content := formatHeaderContent(h)
				if content != "" {
					msgs = append(msgs, messageDict{Speaker: "human", Content: content})
				}
			}
		case ElementParagraph:
			if el.Index >= 0 && el.Index < len(doc.Paragraphs) {
				p := doc.Paragraphs[el.Index]
				content := strings.TrimSpace(p.Content)
				if content != "" {
					msgs = append(msgs, messageDict{Speaker: "human", Content: content})
				}
			}
		case ElementSection:
			if el.Index >= 0 && el.Index < len(doc.Sections) {
				sec := doc.Sections[el.Index]
				content := formatSectionContent(sec)
				if content != "" {
					msgs = append(msgs, messageDict{Speaker: "human", Content: content})
				}
			}
		case ElementList:
			if el.Index >= 0 && el.Index < len(doc.Lists) {
				list := doc.Lists[el.Index]
				content := formatListContent(list)
				if content != "" {
					msgs = append(msgs, messageDict{Speaker: "human", Content: content})
				}
			}
		case ElementCode:
			if el.Index >= 0 && el.Index < len(doc.CodeBlocks) {
				code := doc.CodeBlocks[el.Index]
				content := formatCodeBlockContent(code)
				if content != "" {
					msgs = append(msgs, messageDict{Speaker: "human", Content: content})
				}
			}
		case ElementNewline:
			if el.Index >= 0 && el.Index < len(doc.Newlines) {
				nl := doc.Newlines[el.Index]
				count := nl.Count
				if count <= 0 {
					count = 1
				}
				content := strings.Repeat("\n", count)
				msgs = append(msgs, messageDict{Speaker: "human", Content: content})
			}
		case ElementUnknown:
			if opts.Extended != ExtendedOff {
				if strings.EqualFold(el.Name, "data") {
					body := strings.TrimSpace(doc.elementBody(el))
					msgs = append(msgs, messageDict{
						Speaker: "human",
						Content: map[string]any{
							"type":   "data",
							"syntax": extractAttr(el.RawXML, "syntax"),
							"body":   body,
						},
					})
				} else {
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
	}
	if msgs == nil {
		msgs = []messageDict{}
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
	if doc.Meta.MinVersion != "" || doc.Meta.MaxVersion != "" || doc.Meta.Components != "" || doc.Meta.Stylesheet != "" {
		if out.Runtime == nil {
			out.Runtime = map[string]any{}
		}
		if doc.Meta.MinVersion != "" {
			out.Runtime["min_version"] = doc.Meta.MinVersion
		}
		if doc.Meta.MaxVersion != "" {
			out.Runtime["max_version"] = doc.Meta.MaxVersion
		}
		if doc.Meta.Components != "" {
			out.Runtime["components"] = doc.Meta.Components
		}
		if doc.Meta.Stylesheet != "" {
			out.Runtime["stylesheet"] = doc.Meta.Stylesheet
		}
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
			if opts.Extended == ExtendedOff {
				continue
			}
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
				if strings.EqualFold(el.Name, "data") {
					body := strings.TrimSpace(doc.elementBody(el))
					messages = append(messages, map[string]any{
						"role":    "user",
						"content": body,
					})
				} else {
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
	if messages == nil {
		messages = []map[string]any{}
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
	if doc.Meta.MinVersion != "" {
		result["min_version"] = doc.Meta.MinVersion
	}
	if doc.Meta.MaxVersion != "" {
		result["max_version"] = doc.Meta.MaxVersion
	}
	if doc.Meta.Components != "" {
		result["components"] = doc.Meta.Components
	}
	if doc.Meta.Stylesheet != "" {
		result["stylesheet"] = doc.Meta.Stylesheet
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
			if opts.Extended == ExtendedOff {
				continue
			}
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
				if strings.EqualFold(el.Name, "data") {
					body := strings.TrimSpace(doc.elementBody(el))
					messages = append(messages, map[string]any{
						"type": "human",
						"data": map[string]any{
							"content": body,
						},
					})
				} else {
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
	}
	if messages == nil {
		messages = []map[string]any{}
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
	case ElementUnknown:
		if strings.TrimSpace(el.RawXML) != "" {
			raw := el.RawXML
			if i := strings.Index(raw, ">"); i != -1 {
				if j := strings.LastIndex(raw, "<"); j != -1 && j > i {
					body := raw[i+1 : j]
					return html.UnescapeString(stripCDATA(body))
				}
			}
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

// expandDocument processes template expansion, stylesheet application, and limit enforcement.
func expandDocument(doc Document, opts ConvertOptions) (Document, error) {
	// Step 1: Process <let> bindings and build template context
	ctx := template.NewContext(nil)

	// Add user-provided variables
	for k, v := range opts.TemplateVars {
		ctx.Set(k, v)
	}

	// Process <let> elements to populate context
	for _, let := range doc.LetBindings {
		if let.Value != "" {
			// Expression or literal value
			ctx.Set(let.Name, let.Value)
		} else if let.Src != "" && opts.BaseDir != "" {
			// Load from file
			path := filepath.Join(opts.BaseDir, let.Src)
			data, err := os.ReadFile(path)
			if err != nil {
				return doc, fmt.Errorf("let %q: load src %q: %w", let.Name, let.Src, err)
			}
			// Try to parse as JSON, otherwise use as string
			var jsonVal any
			if err := json.Unmarshal(data, &jsonVal); err == nil {
				ctx.Set(let.Name, jsonVal)
			} else {
				ctx.Set(let.Name, string(data))
			}
		} else if let.Body != "" {
			// Inline body - try JSON first
			var jsonVal any
			if err := json.Unmarshal([]byte(let.Body), &jsonVal); err == nil {
				ctx.Set(let.Name, jsonVal)
			} else {
				ctx.Set(let.Name, let.Body)
			}
		}
	}

	// Step 2: Process if/for conditions on elements
	doc = processConditionalsAndLoops(doc, ctx)

	// Step 3: Process includes (if enabled and base dir set)
	if opts.BaseDir != "" {
		maxDepth := opts.MaxIncludeDepth
		if maxDepth <= 0 {
			maxDepth = 10 // Default max include depth
		}
		var err error
		doc, err = processIncludes(doc, opts.BaseDir, maxDepth, 0, ctx)
		if err != nil {
			return doc, fmt.Errorf("process includes: %w", err)
		}
	}

	// Step 4: Template interpolation (if enabled)
	if opts.ExpandTemplates || opts.TemplateVars != nil {
		doc = interpolateDocument(doc, ctx)
	}

	// Step 5: Apply stylesheet (if enabled and present)
	if opts.ApplyStylesheet && len(doc.Styles) > 0 {
		// Combine all style blocks
		for _, style := range doc.Styles {
			for _, out := range style.Outputs {
				if ss, err := stylesheet.Parse(out.Body); err == nil && ss != nil {
					doc = applyStylesheetToDocument(doc, ss)
				}
			}
		}
	}

	// Step 6: Enforce limits (if enabled)
	if opts.EnforceLimits {
		doc = enforceLimitsOnDocument(doc)
	}

	// Step 7: Apply component filtering (if specified)
	if opts.Components != "" {
		filtered := FilterDocumentByComponents(&doc, opts.Components)
		doc = *filtered
	}

	return doc, nil
}

// processConditionalsAndLoops evaluates if/for attributes on elements.
// Elements with false conditions are removed; elements with loops are expanded.
func processConditionalsAndLoops(doc Document, ctx *template.Context) Document {
	// Filter includes by condition and expand loops
	var filteredIncludes []Include
	for _, inc := range doc.Includes {
		// Check if condition (if present)
		if inc.Condition != "" {
			match, err := template.EvalCondition(inc.Condition, ctx)
			if err != nil || !match {
				continue // Skip this include
			}
		}

		// Check for loop (if present)
		if inc.Loop != "" {
			varName, listExpr, ok := template.ParseForAttribute(inc.Loop)
			if ok {
				items, err := template.EvalLoop(listExpr, ctx)
				if err == nil && len(items) > 0 {
					// Expand the loop - create one include per item
					for _, item := range items {
						newInc := inc
						// Set the loop variable in context for interpolation
						ctx.Set(varName, item)
						filteredIncludes = append(filteredIncludes, newInc)
					}
					continue
				}
			}
		}

		filteredIncludes = append(filteredIncludes, inc)
	}
	doc.Includes = filteredIncludes

	// Filter elements by checking if/for on their underlying items' Attrs
	var filteredElements []Element
	for _, el := range doc.Elements {
		attrs := getElementAttrs(doc, el)
		ifVal := findAttr(attrs, "if")
		forVal := findAttr(attrs, "for")

		// Check if condition
		if ifVal != "" {
			match, err := template.EvalCondition(ifVal, ctx)
			if err != nil || !match {
				continue // Skip this element
			}
		}

		// Check for loop
		if forVal != "" {
			varName, listExpr, ok := template.ParseForAttribute(forVal)
			if ok {
				items, err := template.EvalLoop(listExpr, ctx)
				if err == nil && len(items) > 0 {
					// Expand the loop
					for _, item := range items {
						ctx.Set(varName, item)
						filteredElements = append(filteredElements, el)
					}
					continue
				}
			}
		}

		filteredElements = append(filteredElements, el)
	}
	doc.Elements = filteredElements

	return doc
}

// processIncludes loads and merges included POML files into the document.
func processIncludes(doc Document, baseDir string, maxDepth, currentDepth int, ctx *template.Context) (Document, error) {
	if currentDepth >= maxDepth {
		return doc, fmt.Errorf("maximum include depth (%d) exceeded", maxDepth)
	}

	if len(doc.Includes) == 0 {
		return doc, nil
	}

	// Process each include
	for _, inc := range doc.Includes {
		if inc.Src == "" {
			continue
		}

		// Interpolate src path if it contains template expressions
		srcPath := inc.Src
		if strings.Contains(srcPath, "{{") {
			if result, err := template.Interpolate(srcPath, ctx); err == nil {
				srcPath = result
			}
		}

		// Resolve path relative to baseDir
		fullPath := srcPath
		if !filepath.IsAbs(srcPath) {
			fullPath = filepath.Join(baseDir, srcPath)
		}

		// Read and parse the included file
		data, err := os.ReadFile(fullPath)
		if err != nil {
			return doc, fmt.Errorf("include %q: %w", inc.Src, err)
		}

		// Parse the included document
		incDir := filepath.Dir(fullPath)
		incDoc, err := ParseString(string(data))
		if err != nil {
			return doc, fmt.Errorf("parse include %q: %w", inc.Src, err)
		}
		_ = incDir // Used for recursive includes

		// Recursively process includes in the included document
		incDoc, err = processIncludes(incDoc, incDir, maxDepth, currentDepth+1, ctx)
		if err != nil {
			return doc, fmt.Errorf("include %q: %w", inc.Src, err)
		}

		// Merge content from included document into main document
		doc = mergeIncludedDocument(doc, incDoc)
	}

	// Clear includes since they've been processed
	doc.Includes = nil

	return doc, nil
}

// mergeIncludedDocument merges content from an included document into the main document.
func mergeIncludedDocument(doc, inc Document) Document {
	// Merge inputs
	doc.Inputs = append(doc.Inputs, inc.Inputs...)

	// Merge examples
	doc.Examples = append(doc.Examples, inc.Examples...)

	// Merge hints
	doc.Hints = append(doc.Hints, inc.Hints...)

	// Merge output formats
	doc.OutFormats = append(doc.OutFormats, inc.OutFormats...)

	// Merge images
	doc.Images = append(doc.Images, inc.Images...)

	// Merge audios
	doc.Audios = append(doc.Audios, inc.Audios...)

	// Merge videos
	doc.Videos = append(doc.Videos, inc.Videos...)

	// Merge diagrams
	doc.Diagrams = append(doc.Diagrams, inc.Diagrams...)

	// Merge tables
	doc.Tables = append(doc.Tables, inc.Tables...)

	// Merge folders
	doc.Folders = append(doc.Folders, inc.Folders...)

	// Merge webpages
	doc.Webpages = append(doc.Webpages, inc.Webpages...)

	// Merge conversations
	doc.Conversations = append(doc.Conversations, inc.Conversations...)

	// Merge let bindings
	doc.LetBindings = append(doc.LetBindings, inc.LetBindings...)

	// Merge styles
	doc.Styles = append(doc.Styles, inc.Styles...)

	// Merge messages
	doc.Messages = append(doc.Messages, inc.Messages...)

	// If main doc lacks role/persona, use included doc's
	if doc.Role.Body == "" && inc.Role.Body != "" {
		doc.Role = inc.Role
	}
	if doc.Persona.Body == "" && inc.Persona.Body != "" {
		doc.Persona = inc.Persona
	}

	// Merge tasks
	doc.Tasks = append(doc.Tasks, inc.Tasks...)

	return doc
}

// getElementAttrs returns the Attrs slice for an element based on its type.
func getElementAttrs(doc Document, el Element) []xml.Attr {
	switch el.Type {
	case ElementInput:
		if el.Index >= 0 && el.Index < len(doc.Inputs) {
			return doc.Inputs[el.Index].Attrs
		}
	case ElementExample:
		if el.Index >= 0 && el.Index < len(doc.Examples) {
			return doc.Examples[el.Index].Attrs
		}
	case ElementHint:
		if el.Index >= 0 && el.Index < len(doc.Hints) {
			return doc.Hints[el.Index].Attrs
		}
	case ElementOutputFormat:
		if el.Index >= 0 && el.Index < len(doc.OutFormats) {
			return doc.OutFormats[el.Index].Attrs
		}
	case ElementImage:
		if el.Index >= 0 && el.Index < len(doc.Images) {
			return doc.Images[el.Index].Attrs
		}
	case ElementAudio:
		if el.Index >= 0 && el.Index < len(doc.Audios) {
			return doc.Audios[el.Index].Attrs
		}
	case ElementVideo:
		if el.Index >= 0 && el.Index < len(doc.Videos) {
			return doc.Videos[el.Index].Attrs
		}
	case ElementDiagram:
		if el.Index >= 0 && el.Index < len(doc.Diagrams) {
			return doc.Diagrams[el.Index].Attrs
		}
	case ElementTable:
		if el.Index >= 0 && el.Index < len(doc.Tables) {
			return doc.Tables[el.Index].Attrs
		}
	case ElementFolder:
		if el.Index >= 0 && el.Index < len(doc.Folders) {
			return doc.Folders[el.Index].Attrs
		}
	case ElementWebpage:
		if el.Index >= 0 && el.Index < len(doc.Webpages) {
			return doc.Webpages[el.Index].Attrs
		}
	case ElementConversation:
		if el.Index >= 0 && el.Index < len(doc.Conversations) {
			return doc.Conversations[el.Index].Attrs
		}
	case ElementInclude:
		if el.Index >= 0 && el.Index < len(doc.Includes) {
			return doc.Includes[el.Index].Attrs
		}
	case ElementHeader:
		if el.Index >= 0 && el.Index < len(doc.Headers) {
			return doc.Headers[el.Index].Attrs
		}
	case ElementParagraph:
		if el.Index >= 0 && el.Index < len(doc.Paragraphs) {
			return doc.Paragraphs[el.Index].Attrs
		}
	case ElementSection:
		if el.Index >= 0 && el.Index < len(doc.Sections) {
			return doc.Sections[el.Index].Attrs
		}
	case ElementList:
		if el.Index >= 0 && el.Index < len(doc.Lists) {
			return doc.Lists[el.Index].Attrs
		}
	case ElementCode:
		if el.Index >= 0 && el.Index < len(doc.CodeBlocks) {
			return doc.CodeBlocks[el.Index].Attrs
		}
	case ElementNewline:
		if el.Index >= 0 && el.Index < len(doc.Newlines) {
			return doc.Newlines[el.Index].Attrs
		}
	}
	return nil
}

// findAttr looks up an attribute by local name.
func findAttr(attrs []xml.Attr, name string) string {
	for _, a := range attrs {
		if a.Name.Local == name {
			return a.Value
		}
	}
	return ""
}

// interpolateDocument applies {{ }} template substitution to text fields.
func interpolateDocument(doc Document, ctx *template.Context) Document {
	// Interpolate role
	if doc.Role.Body != "" {
		if result, err := template.Interpolate(doc.Role.Body, ctx); err == nil {
			doc.Role.Body = result
		}
	}

	// Interpolate persona
	if doc.Persona.Body != "" {
		if result, err := template.Interpolate(doc.Persona.Body, ctx); err == nil {
			doc.Persona.Body = result
		}
	}

	// Interpolate tasks
	for i := range doc.Tasks {
		if doc.Tasks[i].Body != "" {
			if result, err := template.Interpolate(doc.Tasks[i].Body, ctx); err == nil {
				doc.Tasks[i].Body = result
			}
		}
	}

	// Interpolate inputs
	for i := range doc.Inputs {
		if doc.Inputs[i].Body != "" {
			if result, err := template.Interpolate(doc.Inputs[i].Body, ctx); err == nil {
				doc.Inputs[i].Body = result
			}
		}
	}

	// Interpolate hints
	for i := range doc.Hints {
		if doc.Hints[i].Body != "" {
			if result, err := template.Interpolate(doc.Hints[i].Body, ctx); err == nil {
				doc.Hints[i].Body = result
			}
		}
	}

	// Interpolate examples
	for i := range doc.Examples {
		if doc.Examples[i].Body != "" {
			if result, err := template.Interpolate(doc.Examples[i].Body, ctx); err == nil {
				doc.Examples[i].Body = result
			}
		}
	}

	// Interpolate messages
	for i := range doc.Messages {
		if doc.Messages[i].Body != "" {
			if result, err := template.Interpolate(doc.Messages[i].Body, ctx); err == nil {
				doc.Messages[i].Body = result
			}
		}
	}

	return doc
}

// applyStylesheetToDocument applies CSS-like rules to elements.
func applyStylesheetToDocument(doc Document, ss *stylesheet.Stylesheet) Document {
	// Helper to apply all properties from stylesheet to attrs
	applyProps := func(attrs []xml.Attr, tagName string) []xml.Attr {
		className := getAttr(attrs, "class")
		props := ss.Apply(tagName, className)
		for name, value := range props {
			attrs = setAttr(attrs, name, value)
		}
		return attrs
	}

	// Apply to hints
	for i := range doc.Hints {
		doc.Hints[i].Attrs = applyProps(doc.Hints[i].Attrs, "hint")
	}

	// Apply to examples
	for i := range doc.Examples {
		doc.Examples[i].Attrs = applyProps(doc.Examples[i].Attrs, "example")
	}

	// Apply to documents
	for i := range doc.Documents {
		doc.Documents[i].Attrs = applyProps(doc.Documents[i].Attrs, "document")
	}

	// Apply to inputs
	for i := range doc.Inputs {
		doc.Inputs[i].Attrs = applyProps(doc.Inputs[i].Attrs, "input")
	}

	// Apply to output formats
	for i := range doc.OutFormats {
		doc.OutFormats[i].Attrs = applyProps(doc.OutFormats[i].Attrs, "output-format")
	}

	// Apply to images
	for i := range doc.Images {
		doc.Images[i].Attrs = applyProps(doc.Images[i].Attrs, "image")
	}

	// Apply to audio
	for i := range doc.Audios {
		doc.Audios[i].Attrs = applyProps(doc.Audios[i].Attrs, "audio")
	}

	// Apply to video
	for i := range doc.Videos {
		doc.Videos[i].Attrs = applyProps(doc.Videos[i].Attrs, "video")
	}

	// Apply to diagrams
	for i := range doc.Diagrams {
		doc.Diagrams[i].Attrs = applyProps(doc.Diagrams[i].Attrs, "diagram")
	}

	// Apply to tables
	for i := range doc.Tables {
		doc.Tables[i].Attrs = applyProps(doc.Tables[i].Attrs, "table")
	}

	// Apply to folders
	for i := range doc.Folders {
		doc.Folders[i].Attrs = applyProps(doc.Folders[i].Attrs, "folder")
	}

	// Apply to webpages
	for i := range doc.Webpages {
		doc.Webpages[i].Attrs = applyProps(doc.Webpages[i].Attrs, "webpage")
	}

	// Apply to conversations
	for i := range doc.Conversations {
		doc.Conversations[i].Attrs = applyProps(doc.Conversations[i].Attrs, "conversation")
	}

	// Apply to code blocks
	for i := range doc.CodeBlocks {
		doc.CodeBlocks[i].Attrs = applyProps(doc.CodeBlocks[i].Attrs, "code")
	}

	// Apply to headers
	for i := range doc.Headers {
		doc.Headers[i].Attrs = applyProps(doc.Headers[i].Attrs, "h")
	}

	// Apply to paragraphs
	for i := range doc.Paragraphs {
		doc.Paragraphs[i].Attrs = applyProps(doc.Paragraphs[i].Attrs, "p")
	}

	// Apply to sections
	for i := range doc.Sections {
		doc.Sections[i].Attrs = applyProps(doc.Sections[i].Attrs, "section")
	}

	// Apply to lists
	for i := range doc.Lists {
		doc.Lists[i].Attrs = applyProps(doc.Lists[i].Attrs, "list")
	}

	return doc
}

// getAttr retrieves an attribute value from a slice of xml.Attr.
func getAttr(attrs []xml.Attr, name string) string {
	for _, a := range attrs {
		if a.Name.Local == name {
			return a.Value
		}
	}
	return ""
}

// setAttr sets or adds an attribute in a slice of xml.Attr.
func setAttr(attrs []xml.Attr, name, value string) []xml.Attr {
	for i, a := range attrs {
		if a.Name.Local == name {
			attrs[i].Value = value
			return attrs
		}
	}
	return append(attrs, xml.Attr{Name: xml.Name{Local: name}, Value: value})
}

// enforceLimitsOnDocument applies charLimit/tokenLimit with priority-based truncation.
func enforceLimitsOnDocument(doc Document) Document {
	charLimit := doc.Meta.CharLimit
	tokenLimit := doc.Meta.TokenLimit

	if charLimit <= 0 && tokenLimit <= 0 {
		return doc
	}

	// Get total task content
	var taskContent string
	for _, t := range doc.Tasks {
		taskContent += t.Body
	}

	// Collect content with priorities for potential truncation
	// For now, apply simple truncation to task and role if needed
	if charLimit > 0 {
		totalChars := int64(len(doc.Role.Body) + len(taskContent))
		if totalChars > charLimit {
			// Truncate tasks first (lower priority typically)
			remaining := charLimit - int64(len(doc.Role.Body))
			if remaining > 0 && len(doc.Tasks) > 0 && int64(len(doc.Tasks[0].Body)) > remaining {
				doc.Tasks[0].Body = token.TruncateToCharLimit(doc.Tasks[0].Body, remaining)
			}
		}
	}

	if tokenLimit > 0 {
		// Use token-based truncation
		totalTokens := token.EstimateTokens(doc.Role.Body + taskContent)
		if totalTokens > tokenLimit {
			// Estimate how many chars we can keep
			charBudget := tokenLimit * 4 // rough estimate
			remaining := charBudget - int64(len(doc.Role.Body))
			if remaining > 0 && len(doc.Tasks) > 0 && int64(len(doc.Tasks[0].Body)) > remaining {
				doc.Tasks[0].Body = token.TruncateToCharLimit(doc.Tasks[0].Body, remaining)
			}
		}
	}

	return doc
}

// formatTableContent formats a Table as markdown-like text.
func formatTableContent(tbl Table) string {
	if len(tbl.Records) == 0 {
		return ""
	}

	var sb strings.Builder

	// Extract column headers from Columns or Records keys
	var headers []string
	if len(tbl.Columns) > 0 {
		for _, col := range tbl.Columns {
			if col.Header != "" {
				headers = append(headers, col.Header)
			} else {
				headers = append(headers, col.Field)
			}
		}
	} else if len(tbl.Records) > 0 {
		// Extract keys from first record
		for k := range tbl.Records[0] {
			headers = append(headers, k)
		}
	}

	// Write headers
	if len(headers) > 0 {
		sb.WriteString("| ")
		for i, h := range headers {
			if i > 0 {
				sb.WriteString(" | ")
			}
			sb.WriteString(h)
		}
		sb.WriteString(" |\n")

		// Separator
		sb.WriteString("|")
		for range headers {
			sb.WriteString(" --- |")
		}
		sb.WriteString("\n")
	}

	// Determine field order for values
	var fields []string
	if len(tbl.Columns) > 0 {
		for _, col := range tbl.Columns {
			fields = append(fields, col.Field)
		}
	} else {
		fields = headers
	}

	// Write rows
	for _, rec := range tbl.Records {
		sb.WriteString("| ")
		for i, field := range fields {
			if i > 0 {
				sb.WriteString(" | ")
			}
			if val, ok := rec[field]; ok {
				sb.WriteString(fmt.Sprintf("%v", val))
			}
		}
		sb.WriteString(" |\n")
	}

	return sb.String()
}

// formatFolderContent formats a Folder as a tree structure.
func formatFolderContent(folder Folder, baseDir string) string {
	var sb strings.Builder

	if folder.Src != "" {
		sb.WriteString("Folder: ")
		sb.WriteString(folder.Src)
		sb.WriteString("\n")
	}

	// List entries
	for _, entry := range folder.Entries {
		if entry.IsDir {
			sb.WriteString("  [dir] ")
		} else {
			sb.WriteString("  - ")
		}
		sb.WriteString(entry.Path)
		sb.WriteString("\n")

		if entry.Content != "" && folder.ShowContent {
			sb.WriteString("    ```\n")
			sb.WriteString("    ")
			sb.WriteString(entry.Content)
			sb.WriteString("\n    ```\n")
		}
	}

	return sb.String()
}

// formatConversationContent formats a Conversation as readable text.
func formatConversationContent(conv Conversation) string {
	var sb strings.Builder

	for _, turn := range conv.Turns {
		sb.WriteString("[")
		sb.WriteString(turn.Speaker)
		sb.WriteString("]: ")
		sb.WriteString(turn.Content)
		sb.WriteString("\n\n")
	}

	return strings.TrimSpace(sb.String())
}

// formatHeaderContent formats a Header as markdown heading.
func formatHeaderContent(h Header) string {
	level := h.Level
	if level < 1 {
		level = 1
	}
	if level > 6 {
		level = 6
	}
	prefix := strings.Repeat("#", level)
	content := strings.TrimSpace(h.Content)
	if content == "" {
		return ""
	}
	return prefix + " " + content
}

// formatSectionContent formats a Section with optional title.
func formatSectionContent(sec Section) string {
	var sb strings.Builder
	if sec.Title != "" {
		sb.WriteString("## ")
		sb.WriteString(sec.Title)
		sb.WriteString("\n\n")
	}
	content := strings.TrimSpace(sec.Content)
	if content != "" {
		sb.WriteString(content)
	}
	return strings.TrimSpace(sb.String())
}

// formatListContent formats a List as markdown list.
func formatListContent(list List) string {
	if len(list.Items) == 0 {
		return ""
	}

	var sb strings.Builder
	for i, item := range list.Items {
		content := strings.TrimSpace(item.Content)
		if content == "" {
			continue
		}

		switch list.Style {
		case "decimal":
			sb.WriteString(fmt.Sprintf("%d. ", i+1))
		case "latin":
			if i < 26 {
				sb.WriteString(string(rune('a'+i)) + ". ")
			} else {
				sb.WriteString(fmt.Sprintf("%d. ", i+1))
			}
		default: // star, dash, plus, or empty
			var marker string
			switch list.Style {
			case "star":
				marker = "*"
			case "plus":
				marker = "+"
			default:
				marker = "-"
			}
			sb.WriteString(marker + " ")
		}
		sb.WriteString(content)
		sb.WriteString("\n")
	}
	return strings.TrimSpace(sb.String())
}

// formatCodeBlockContent formats a CodeBlock as markdown fenced code.
func formatCodeBlockContent(code CodeBlock) string {
	content := strings.TrimSpace(code.Content)
	if content == "" {
		return ""
	}

	if code.Inline {
		return "`" + content + "`"
	}

	var sb strings.Builder
	sb.WriteString("```")
	if code.Lang != "" {
		sb.WriteString(code.Lang)
	}
	sb.WriteString("\n")
	sb.WriteString(content)
	sb.WriteString("\n```")
	return sb.String()
}
