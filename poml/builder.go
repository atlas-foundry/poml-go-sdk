package poml

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
)

// Builder provides a fluent API for constructing a Document in code (similar to the Python Prompt builder).
type Builder struct {
	doc Document
}

// NewBuilder creates an empty builder.
func NewBuilder() *Builder {
	return &Builder{}
}

// Build returns the assembled Document.
func (b *Builder) Build() Document {
	return b.doc
}

// Meta sets the required meta section.
func (b *Builder) Meta(id, version, owner string) *Builder {
	b.doc.Meta = Meta{ID: id, Version: version, Owner: owner}
	b.doc.Elements = append(b.doc.Elements, b.doc.newElement(ElementMeta, -1, ""))
	return b
}

// Role sets the role block.
func (b *Builder) Role(body string) *Builder {
	b.doc.Role = Block{Body: body}
	b.doc.Elements = append(b.doc.Elements, b.doc.newElement(ElementRole, -1, ""))
	return b
}

// Task appends a task block.
func (b *Builder) Task(body string, attrs ...xml.Attr) *Builder {
	b.doc.Tasks = append(b.doc.Tasks, Block{Body: body, Attrs: attrs})
	b.doc.Elements = append(b.doc.Elements, b.doc.newElement(ElementTask, len(b.doc.Tasks)-1, ""))
	return b
}

// Input appends an input block.
func (b *Builder) Input(name string, required bool, body string, attrs ...xml.Attr) *Builder {
	b.doc.Inputs = append(b.doc.Inputs, Input{Name: name, Required: required, Body: body, Attrs: attrs})
	b.doc.Elements = append(b.doc.Elements, b.doc.newElement(ElementInput, len(b.doc.Inputs)-1, ""))
	return b
}

// DocumentRef appends a document reference.
func (b *Builder) DocumentRef(src string, attrs ...xml.Attr) *Builder {
	b.doc.Documents = append(b.doc.Documents, DocRef{Src: src, Attrs: attrs})
	b.doc.Elements = append(b.doc.Elements, b.doc.newElement(ElementDocument, len(b.doc.Documents)-1, "document"))
	return b
}

// Style appends a style block.
func (b *Builder) Style(outputs ...Output) *Builder {
	b.doc.Styles = append(b.doc.Styles, Style{Outputs: outputs})
	b.doc.Elements = append(b.doc.Elements, b.doc.newElement(ElementStyle, len(b.doc.Styles)-1, ""))
	return b
}

// OutputFormat appends an output-format block.
func (b *Builder) OutputFormat(body string, attrs ...xml.Attr) *Builder {
	b.doc.OutFormats = append(b.doc.OutFormats, OutputFormat{Body: body, Attrs: attrs})
	b.doc.Elements = append(b.doc.Elements, b.doc.newElement(ElementOutputFormat, len(b.doc.OutFormats)-1, ""))
	return b
}

// Human appends a human message.
func (b *Builder) Human(body string, attrs ...xml.Attr) *Builder {
	b.doc.Messages = append(b.doc.Messages, Message{Role: "human", Body: body, Attrs: attrs})
	b.doc.Elements = append(b.doc.Elements, b.doc.newElement(ElementHumanMsg, len(b.doc.Messages)-1, ""))
	return b
}

// Assistant appends an assistant message.
func (b *Builder) Assistant(body string, attrs ...xml.Attr) *Builder {
	b.doc.Messages = append(b.doc.Messages, Message{Role: "assistant", Body: body, Attrs: attrs})
	b.doc.Elements = append(b.doc.Elements, b.doc.newElement(ElementAssistantMsg, len(b.doc.Messages)-1, ""))
	return b
}

// System appends a system message.
func (b *Builder) System(body string, attrs ...xml.Attr) *Builder {
	b.doc.Messages = append(b.doc.Messages, Message{Role: "system", Body: body, Attrs: attrs})
	b.doc.Elements = append(b.doc.Elements, b.doc.newElement(ElementSystemMsg, len(b.doc.Messages)-1, ""))
	return b
}

// ToolDefinition appends a tool-definition with optional description attribute and parameters body.
func (b *Builder) ToolDefinition(name, description string, parameters any, attrs ...xml.Attr) *Builder {
	body := marshalAny(parameters)
	b.doc.ToolDefs = append(b.doc.ToolDefs, ToolDefinition{Name: name, Description: description, Body: body, Attrs: attrs})
	b.doc.Elements = append(b.doc.Elements, b.doc.newElement(ElementToolDefinition, len(b.doc.ToolDefs)-1, ""))
	return b
}

// ToolRequest appends a tool-request.
func (b *Builder) ToolRequest(id, name string, parameters any, attrs ...xml.Attr) *Builder {
	params := marshalAny(parameters)
	b.doc.ToolReqs = append(b.doc.ToolReqs, ToolRequest{ID: id, Name: name, Parameters: params, Attrs: attrs})
	b.doc.Elements = append(b.doc.Elements, b.doc.newElement(ElementToolRequest, len(b.doc.ToolReqs)-1, ""))
	return b
}

// ToolResponse appends a tool-response.
func (b *Builder) ToolResponse(id, name, body string, attrs ...xml.Attr) *Builder {
	b.doc.ToolResps = append(b.doc.ToolResps, ToolResponse{ID: id, Name: name, Body: body, Attrs: attrs})
	b.doc.Elements = append(b.doc.Elements, b.doc.newElement(ElementToolResponse, len(b.doc.ToolResps)-1, ""))
	return b
}

// ToolResult appends a tool-result.
func (b *Builder) ToolResult(id, name, body string, attrs ...xml.Attr) *Builder {
	b.doc.ToolResults = append(b.doc.ToolResults, ToolResult{ID: id, Name: name, Body: body, Attrs: attrs})
	b.doc.Elements = append(b.doc.Elements, b.doc.newElement(ElementToolResult, len(b.doc.ToolResults)-1, ""))
	return b
}

// ToolError appends a tool-error.
func (b *Builder) ToolError(id, name, body string, attrs ...xml.Attr) *Builder {
	b.doc.ToolErrors = append(b.doc.ToolErrors, ToolError{ID: id, Name: name, Body: body, Attrs: attrs})
	b.doc.Elements = append(b.doc.Elements, b.doc.newElement(ElementToolError, len(b.doc.ToolErrors)-1, ""))
	return b
}

// OutputSchema sets the output-schema.
func (b *Builder) OutputSchema(schema any, attrs ...xml.Attr) *Builder {
	body := marshalAny(schema)
	b.doc.Schema = OutputSchema{Body: body, Attrs: attrs}
	// remove prior schema element if present
	var filtered []Element
	for _, el := range b.doc.Elements {
		if el.Type != ElementOutputSchema {
			filtered = append(filtered, el)
		}
	}
	b.doc.Elements = append(filtered, b.doc.newElement(ElementOutputSchema, -1, ""))
	return b
}

// Runtime appends a runtime entry from a map of attributes.
func (b *Builder) Runtime(attrs map[string]any) *Builder {
	var xmlAttrs []xml.Attr
	for k, v := range attrs {
		xmlAttrs = append(xmlAttrs, xml.Attr{Name: xml.Name{Local: k}, Value: fmt.Sprint(v)})
	}
	b.doc.Runtimes = append(b.doc.Runtimes, Runtime{Attrs: xmlAttrs})
	b.doc.Elements = append(b.doc.Elements, b.doc.newElement(ElementRuntime, len(b.doc.Runtimes)-1, ""))
	return b
}

// Image appends an image element.
func (b *Builder) Image(img Image) *Builder {
	b.doc.Images = append(b.doc.Images, img)
	b.doc.Elements = append(b.doc.Elements, b.doc.newElement(ElementImage, len(b.doc.Images)-1, ""))
	return b
}

// Audio appends an audio element.
func (b *Builder) Audio(media Media) *Builder {
	b.doc.Audios = append(b.doc.Audios, media)
	b.doc.Elements = append(b.doc.Elements, b.doc.newElement(ElementAudio, len(b.doc.Audios)-1, ""))
	return b
}

// Video appends a video element.
func (b *Builder) Video(media Media) *Builder {
	b.doc.Videos = append(b.doc.Videos, media)
	b.doc.Elements = append(b.doc.Elements, b.doc.newElement(ElementVideo, len(b.doc.Videos)-1, ""))
	return b
}

// Hint appends a hint block.
func (b *Builder) Hint(body string, attrs ...xml.Attr) *Builder {
	b.doc.Hints = append(b.doc.Hints, Hint{Body: body, Attrs: attrs})
	b.doc.Elements = append(b.doc.Elements, b.doc.newElement(ElementHint, len(b.doc.Hints)-1, ""))
	return b
}

// Example appends an example block.
func (b *Builder) Example(body string, attrs ...xml.Attr) *Builder {
	b.doc.Examples = append(b.doc.Examples, Example{Body: body, Attrs: attrs})
	b.doc.Elements = append(b.doc.Elements, b.doc.newElement(ElementExample, len(b.doc.Examples)-1, ""))
	return b
}

// ContentPart appends a content part (<cp>).
func (b *Builder) ContentPart(body string, attrs ...xml.Attr) *Builder {
	b.doc.ContentParts = append(b.doc.ContentParts, ContentPart{Body: body, Attrs: attrs})
	b.doc.Elements = append(b.doc.Elements, b.doc.newElement(ElementContentPart, len(b.doc.ContentParts)-1, ""))
	return b
}

// Object appends an object wrapper.
func (b *Builder) Object(data, syntax, body string, attrs ...xml.Attr) *Builder {
	b.doc.Objects = append(b.doc.Objects, ObjectTag{Data: data, Syntax: syntax, Body: body, Attrs: attrs})
	b.doc.Elements = append(b.doc.Elements, b.doc.newElement(ElementObject, len(b.doc.Objects)-1, ""))
	return b
}

// Diagram appends a diagram element.
func (b *Builder) Diagram(d Diagram) *Builder {
	b.doc.Diagrams = append(b.doc.Diagrams, d)
	b.doc.Elements = append(b.doc.Elements, b.doc.newElement(ElementDiagram, len(b.doc.Diagrams)-1, ""))
	return b
}

// Raw inserts an unknown element using raw XML to keep extension tags in order.
func (b *Builder) Raw(rawXML string) *Builder {
	b.doc.Elements = append(b.doc.Elements, b.doc.newElement(ElementUnknown, -1, "", rawXML))
	return b
}

func marshalAny(v any) string {
	switch val := v.(type) {
	case nil:
		return ""
	case string:
		return val
	default:
		bs, err := json.Marshal(val)
		if err != nil {
			return ""
		}
		return string(bs)
	}
}
