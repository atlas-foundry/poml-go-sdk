package poml

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

// ElementType enumerates the top-level nodes allowed under <poml>.
type ElementType string

const (
	ElementMeta           ElementType = "meta"
	ElementRole           ElementType = "role"
	ElementTask           ElementType = "task"
	ElementInput          ElementType = "input"
	ElementDocument       ElementType = "document"
	ElementStyle          ElementType = "style"
	ElementHumanMsg       ElementType = "human_msg"
	ElementAssistantMsg   ElementType = "assistant_msg"
	ElementSystemMsg      ElementType = "system_msg"
	ElementToolDefinition ElementType = "tool_definition"
	ElementToolRequest    ElementType = "tool_request"
	ElementToolResponse   ElementType = "tool_response"
	ElementToolResult     ElementType = "tool_result"
	ElementToolError      ElementType = "tool_error"
	ElementOutputSchema   ElementType = "output_schema"
	ElementOutputFormat   ElementType = "output_format"
	ElementAudio          ElementType = "audio"
	ElementVideo          ElementType = "video"
	ElementHint           ElementType = "hint"
	ElementExample        ElementType = "example"
	ElementContentPart    ElementType = "content_part"
	ElementObject         ElementType = "object"
	ElementRuntime        ElementType = "runtime"
	ElementImage          ElementType = "image"
	ElementDiagram        ElementType = "diagram"
	ElementUnknown        ElementType = "unknown"
)

// Element tracks an entry's type and its position in the backing slices on Document.
// Index is -1 for singleton fields (meta/role) or unknown elements.
type Element struct {
	Type     ElementType
	Index    int
	Name     string // filled for unknown elements
	RawXML   string // raw XML for unknown elements to preserve round-trips
	Comment  string // optional leading comment retained when available
	ID       string // stable element ID for mutation/lookups
	Parent   string // parent element ID (root for top-level)
	Leading  string // whitespace/comments preceding this element
	Trailing string // whitespace/comments following this element (before next element/end)
}

// Document represents a POML file.
// Elements preserves encountered order for role/task/input/document/style nodes.
type Document struct {
	Meta         Meta     `xml:"meta"`
	Role         Block    `xml:"role"`
	Tasks        []Block  `xml:"task"`
	Inputs       []Input  `xml:"input"`
	Documents    []DocRef `xml:"document"`
	Styles       []Style  `xml:"style"`
	OutFormats   []OutputFormat
	Hints        []Hint
	Examples     []Example
	ContentParts []ContentPart
	Objects      []ObjectTag
	Audios       []Media
	Videos       []Media
	Messages     []Message
	ToolDefs     []ToolDefinition
	ToolReqs     []ToolRequest
	ToolResps    []ToolResponse
	ToolResults  []ToolResult
	ToolErrors   []ToolError
	Runtimes     []Runtime
	Schema       OutputSchema
	Images       []Image
	Diagrams     []Diagram
	Elements     []Element
	rawPrefix    string // leading text before root (e.g., XML decl); kept for future extension

	nextID int // internal counter for element IDs
}

// Meta captures the id/version/owner fields under <meta>.
type Meta struct {
	ID      string `xml:"id"`
	Version string `xml:"version"`
	Owner   string `xml:"owner"`
}

// Block holds free-form body content for task/role/style sections.
type Block struct {
	Body  string     `xml:",innerxml"`
	Attrs []xml.Attr `xml:",any,attr"`
}

// Input represents a named input block.
type Input struct {
	Name     string     `xml:"name,attr"`
	Required bool       `xml:"required,attr"`
	Body     string     `xml:",innerxml"`
	Attrs    []xml.Attr `xml:",any,attr"`
}

// DocRef links to an external source document.
type DocRef struct {
	Src   string     `xml:"src,attr"`
	Attrs []xml.Attr `xml:",any,attr"`
}

// Style represents an <style><output format=...> block.
type Style struct {
	Outputs []Output   `xml:"output"`
	Attrs   []xml.Attr `xml:",any,attr"`
}

// OutputFormat is a simplified format hint (<output-format>...</output-format>).
type OutputFormat struct {
	Body  string     `xml:",innerxml"`
	Attrs []xml.Attr `xml:",any,attr"`
}

// Hint represents a <hint> block that wraps supporting context.
type Hint struct {
	Body  string     `xml:",innerxml"`
	Attrs []xml.Attr `xml:",any,attr"`
}

// Example represents an <example> block.
type Example struct {
	Body  string     `xml:",innerxml"`
	Attrs []xml.Attr `xml:",any,attr"`
}

// ContentPart represents a captioned content part (<cp>).
type ContentPart struct {
	Body  string     `xml:",innerxml"`
	Attrs []xml.Attr `xml:",any,attr"`
}

// ObjectTag represents an <object> wrapper for data payloads.
type ObjectTag struct {
	Data   string     `xml:"data,attr"`
	Syntax string     `xml:"syntax,attr"`
	Body   string     `xml:",innerxml"`
	Attrs  []xml.Attr `xml:",any,attr"`
}

// Image represents an <img> block (often used for multimedia).
type Image struct {
	Src    string     `xml:"src,attr"`
	Alt    string     `xml:"alt,attr"`
	Syntax string     `xml:"syntax,attr"`
	Body   string     `xml:",innerxml"`
	Attrs  []xml.Attr `xml:",any,attr"`
}

// Message represents <human-msg>, <assistant-msg>, or <system-msg>.
type Message struct {
	Role  string     `xml:"-"`
	Body  string     `xml:",innerxml"`
	Attrs []xml.Attr `xml:",any,attr"`
}

// ToolDefinition describes a tool/function exposed to the model.
type ToolDefinition struct {
	Name        string     `xml:"name,attr"`
	Description string     `xml:"description,attr"`
	Body        string     `xml:",innerxml"`
	Attrs       []xml.Attr `xml:",any,attr"`
}

// ToolRequest captures a tool call issued by the model.
type ToolRequest struct {
	ID         string     `xml:"id,attr"`
	Name       string     `xml:"name,attr"`
	Parameters string     `xml:"parameters,attr"`
	Attrs      []xml.Attr `xml:",any,attr"`
}

// ToolResponse captures a tool response.
type ToolResponse struct {
	ID    string     `xml:"id,attr"`
	Name  string     `xml:"name,attr"`
	Body  string     `xml:",innerxml"`
	Attrs []xml.Attr `xml:",any,attr"`
}

// ToolResult captures a tool call result (success).
type ToolResult struct {
	ID    string     `xml:"id,attr"`
	Name  string     `xml:"name,attr"`
	Body  string     `xml:",innerxml"`
	Attrs []xml.Attr `xml:",any,attr"`
}

// ToolError captures an error from a tool call.
type ToolError struct {
	ID    string     `xml:"id,attr"`
	Name  string     `xml:"name,attr"`
	Body  string     `xml:",innerxml"`
	Attrs []xml.Attr `xml:",any,attr"`
}

// OutputSchema represents a JSON schema block.
type OutputSchema struct {
	Body  string     `xml:",innerxml"`
	Attrs []xml.Attr `xml:",any,attr"`
}

// Runtime captures model/runtime hints.
type Runtime struct {
	Attrs []xml.Attr `xml:",any,attr"`
}

// Output holds a single output format entry.
type Output struct {
	Format string     `xml:"format,attr"`
	Body   string     `xml:",innerxml"`
	Attrs  []xml.Attr `xml:",any,attr"`
}

// Media represents audio/video payloads.
type Media struct {
	Src    string     `xml:"src,attr"`
	Alt    string     `xml:"alt,attr"`
	Syntax string     `xml:"syntax,attr"`
	Body   string     `xml:",innerxml"`
	Attrs  []xml.Attr `xml:",any,attr"`
}

// EncodeOptions controls XML serialization.
type EncodeOptions struct {
	Indent        string // indentation used for Encode/EncodeWithOptions; default "  "
	IncludeHeader bool   // emit xml.Header when true
	PreserveOrder bool   // when true and Elements populated, emit in original order
	PreserveWS    bool   // when true, emit preserved Leading/Trailing whitespace/comments
	Compact       bool   // when true, disable indentation
}

// ParseOptions controls parsing fidelity.
type ParseOptions struct {
	// PreserveWhitespace retains leading/trailing whitespace/comments between elements.
	// When false, Leading/Trailing fields on Elements remain empty to reduce memory.
	PreserveWhitespace bool
	// Validate runs structural validation (meta/role/task, diagrams, etc.) after parsing.
	// When false, parsing succeeds even if required fields are missing.
	Validate bool
}

var defaultParseOptions = ParseOptions{PreserveWhitespace: true}
var strictParseOptions = ParseOptions{PreserveWhitespace: true, Validate: true}
var fastParseOptions = ParseOptions{PreserveWhitespace: false}

type ErrorType string

const (
	ErrInvalidSchema ErrorType = "invalid_schema"
	ErrDecode        ErrorType = "decode_error"
	ErrValidate      ErrorType = "validation_error"
)

// POMLError wraps decoding/validation issues with context and type.
type POMLError struct {
	Type    ErrorType
	Message string
	Err     error
}

// ValidationDetail provides structured validation info.
type ValidationDetail struct {
	Field   string
	Element ElementType
	Message string
}

// ValidationError groups structural problems.
type ValidationError struct {
	Issues  []string
	Details []ValidationDetail
}

func (e *POMLError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *POMLError) Unwrap() error { return e.Err }

func (v *ValidationError) Error() string {
	return "poml validation failed: " + strings.Join(v.Issues, "; ")
}

// ParseString decodes a POML document from a string.
func ParseString(body string) (Document, error) {
	return parseWithOptions(strings.NewReader(body), defaultParseOptions)
}

// ParseStringFast decodes a POML document without whitespace/comment preservation for speed/memory.
func ParseStringFast(body string) (Document, error) {
	return parseWithOptions(strings.NewReader(body), fastParseOptions)
}

// ParseFile decodes a POML document from the given file path.
func ParseFile(path string) (Document, error) {
	f, err := os.Open(path)
	if err != nil {
		return Document{}, err
	}
	defer f.Close()
	return parseWithOptions(f, defaultParseOptions)
}

// ParseFileFast decodes a POML file without whitespace/comment preservation.
func ParseFileFast(path string) (Document, error) {
	f, err := os.Open(path)
	if err != nil {
		return Document{}, err
	}
	defer f.Close()
	return parseWithOptions(f, fastParseOptions)
}

// ParseReader decodes a POML document from an io.Reader.
func ParseReader(r io.Reader) (Document, error) {
	return parseWithOptions(r, defaultParseOptions)
}

// ParseReaderFast decodes a POML document from an io.Reader without whitespace/comment preservation.
func ParseReaderFast(r io.Reader) (Document, error) {
	return parseWithOptions(r, fastParseOptions)
}

// ParseReaderWithOptions decodes a POML document with fidelity controls.
func ParseReaderWithOptions(r io.Reader, opts ParseOptions) (Document, error) {
	return parseWithOptions(r, opts)
}

// ParseStringStrict decodes a POML document with validation enabled.
func ParseStringStrict(body string) (Document, error) {
	return parseWithOptions(strings.NewReader(body), strictParseOptions)
}

// ParseFileStrict decodes a POML file with validation enabled.
func ParseFileStrict(path string) (Document, error) {
	f, err := os.Open(path)
	if err != nil {
		return Document{}, err
	}
	defer f.Close()
	return parseWithOptions(f, strictParseOptions)
}

// ParseReaderStrict decodes a POML document from a reader with validation enabled.
func ParseReaderStrict(r io.Reader) (Document, error) {
	return parseWithOptions(r, strictParseOptions)
}

// Encode writes the POML document back to XML.
func (d Document) Encode(w io.Writer) error {
	return d.EncodeWithOptions(w, EncodeOptions{
		Indent:        "  ",
		IncludeHeader: true,
		PreserveOrder: true,
		PreserveWS:    false,
	})
}

// EncodeWithOptions writes a POML document with configurable formatting.
func (d Document) EncodeWithOptions(w io.Writer, opts EncodeOptions) error {
	enc := xml.NewEncoder(w)
	if opts.Compact {
		enc.Indent("", "")
	} else if opts.Indent != "" {
		enc.Indent("", opts.Indent)
	}
	if opts.IncludeHeader {
		if _, err := w.Write([]byte(xml.Header)); err != nil {
			return err
		}
	}
	if err := encodeDocument(enc, w, d, opts); err != nil {
		return err
	}
	return enc.Flush()
}

// WalkInputs applies fn to each input block.
func (d *Document) WalkInputs(fn func(*Input)) {
	if fn == nil {
		return
	}
	for i := range d.Inputs {
		fn(&d.Inputs[i])
	}
}

// RoleText returns the role text with surrounding whitespace trimmed.
func (d Document) RoleText() string {
	return strings.TrimSpace(d.Role.Body)
}

// TaskBodies returns all task bodies trimmed.
func (d Document) TaskBodies() []string {
	out := make([]string, 0, len(d.Tasks))
	for _, t := range d.Tasks {
		body := strings.TrimSpace(t.Body)
		if body != "" {
			out = append(out, body)
		}
	}
	return out
}

// DumpFile writes the document to path atomically using Encode options.
func (d Document) DumpFile(path string, opts EncodeOptions) error {
	tmp := path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := d.EncodeWithOptions(f, opts); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// AddRole sets the role body and appends to ordering metadata.
func (d *Document) AddRole(body string) {
	d.Role = Block{Body: body}
	d.Elements = append(d.Elements, d.newElement(ElementRole, -1, ""))
}

// AddTask appends a task and returns its index.
func (d *Document) AddTask(body string) int {
	d.Tasks = append(d.Tasks, Block{Body: body})
	idx := len(d.Tasks) - 1
	d.Elements = append(d.Elements, d.newElement(ElementTask, idx, ""))
	return idx
}

// AddInput appends an input element (with its required flag and body).
func (d *Document) AddInput(name string, required bool, body string) int {
	d.Inputs = append(d.Inputs, Input{Name: name, Required: required, Body: body})
	idx := len(d.Inputs) - 1
	d.Elements = append(d.Elements, d.newElement(ElementInput, idx, ""))
	return idx
}

// AddDocument appends a document reference.
func (d *Document) AddDocument(src string) int {
	d.Documents = append(d.Documents, DocRef{Src: src})
	idx := len(d.Documents) - 1
	d.Elements = append(d.Elements, d.newElement(ElementDocument, idx, ""))
	return idx
}

// AddStyle appends a style block.
func (d *Document) AddStyle(outputs ...Output) int {
	d.Styles = append(d.Styles, Style{Outputs: outputs})
	idx := len(d.Styles) - 1
	d.Elements = append(d.Elements, d.newElement(ElementStyle, idx, ""))
	return idx
}

// AddMessage appends a message node with the given role/body.
func (d *Document) AddMessage(role string, body string, attrs ...xml.Attr) int {
	msg := Message{Role: role, Body: body, Attrs: attrs}
	d.Messages = append(d.Messages, msg)
	elType := ElementHumanMsg
	switch role {
	case "assistant":
		elType = ElementAssistantMsg
	case "system":
		elType = ElementSystemMsg
	}
	idx := len(d.Messages) - 1
	d.Elements = append(d.Elements, d.newElement(elType, idx, ""))
	return idx
}

// AddToolDefinition appends a tool-definition.
func (d *Document) AddToolDefinition(name, description string, attrs ...xml.Attr) int {
	td := ToolDefinition{Name: name, Body: description, Attrs: attrs}
	d.ToolDefs = append(d.ToolDefs, td)
	idx := len(d.ToolDefs) - 1
	d.Elements = append(d.Elements, d.newElement(ElementToolDefinition, idx, ""))
	return idx
}

// AddToolRequest appends a tool-request entry.
func (d *Document) AddToolRequest(id, name, params string, attrs ...xml.Attr) int {
	tr := ToolRequest{ID: id, Name: name, Parameters: params, Attrs: attrs}
	d.ToolReqs = append(d.ToolReqs, tr)
	idx := len(d.ToolReqs) - 1
	d.Elements = append(d.Elements, d.newElement(ElementToolRequest, idx, ""))
	return idx
}

// AddToolResponse appends a tool-response entry.
func (d *Document) AddToolResponse(id, name, body string, attrs ...xml.Attr) int {
	tr := ToolResponse{ID: id, Name: name, Body: body, Attrs: attrs}
	d.ToolResps = append(d.ToolResps, tr)
	idx := len(d.ToolResps) - 1
	d.Elements = append(d.Elements, d.newElement(ElementToolResponse, idx, ""))
	return idx
}

// AddToolResult appends a tool-result entry.
func (d *Document) AddToolResult(id, name, body string, attrs ...xml.Attr) int {
	tr := ToolResult{ID: id, Name: name, Body: body, Attrs: attrs}
	d.ToolResults = append(d.ToolResults, tr)
	idx := len(d.ToolResults) - 1
	d.Elements = append(d.Elements, d.newElement(ElementToolResult, idx, ""))
	return idx
}

// AddToolError appends a tool-error entry.
func (d *Document) AddToolError(id, name, body string, attrs ...xml.Attr) int {
	te := ToolError{ID: id, Name: name, Body: body, Attrs: attrs}
	d.ToolErrors = append(d.ToolErrors, te)
	idx := len(d.ToolErrors) - 1
	d.Elements = append(d.Elements, d.newElement(ElementToolError, idx, ""))
	return idx
}

// AddOutputSchema sets the output schema and records ordering.
func (d *Document) AddOutputSchema(body string, attrs ...xml.Attr) {
	d.Schema = OutputSchema{Body: body, Attrs: attrs}
	// remove prior schema entries to avoid duplicates in Elements
	var filtered []Element
	for _, el := range d.Elements {
		if el.Type != ElementOutputSchema {
			filtered = append(filtered, el)
		}
	}
	d.Elements = filtered
	d.Elements = append(d.Elements, d.newElement(ElementOutputSchema, -1, ""))
}

// AddRuntime appends a runtime entry with attributes.
func (d *Document) AddRuntime(attrs ...xml.Attr) int {
	rt := Runtime{Attrs: attrs}
	d.Runtimes = append(d.Runtimes, rt)
	idx := len(d.Runtimes) - 1
	d.Elements = append(d.Elements, d.newElement(ElementRuntime, idx, ""))
	return idx
}

// AddImage appends an image node.
func (d *Document) AddImage(img Image) int {
	d.Images = append(d.Images, img)
	idx := len(d.Images) - 1
	d.Elements = append(d.Elements, d.newElement(ElementImage, idx, ""))
	return idx
}

// Validate ensures required metadata exists and inputs are well-formed.
func (d Document) Validate() error {
	var issues []string
	var details []ValidationDetail
	metaCount, roleCount, taskCount := 0, 0, len(d.Tasks)
	if len(d.Elements) > 0 {
		metaCount, roleCount, taskCount = 0, 0, 0
		for _, el := range d.Elements {
			switch el.Type {
			case ElementMeta:
				metaCount++
			case ElementRole:
				roleCount++
			case ElementTask:
				taskCount++
			}
		}
	}
	if metaCount == 0 && (d.Meta != Meta{}) {
		metaCount = 1
	}
	if roleCount == 0 && strings.TrimSpace(d.Role.Body) != "" {
		roleCount = 1
	}

	if metaCount == 0 {
		issues = append(issues, "meta section is required")
		details = append(details, ValidationDetail{Element: ElementMeta, Message: "missing meta"})
	}
	if roleCount == 0 {
		issues = append(issues, "role section is required")
		details = append(details, ValidationDetail{Element: ElementRole, Message: "missing role"})
	}
	if taskCount == 0 {
		issues = append(issues, "at least one task is required")
		details = append(details, ValidationDetail{Element: ElementTask, Message: "missing task"})
	}
	if metaCount > 1 {
		issues = append(issues, "only one meta section is allowed")
		details = append(details, ValidationDetail{Element: ElementMeta, Message: "duplicate meta"})
	}
	if roleCount > 1 {
		issues = append(issues, "only one role section is allowed")
		details = append(details, ValidationDetail{Element: ElementRole, Message: "duplicate role"})
	}
	if strings.TrimSpace(d.Meta.ID) == "" {
		issues = append(issues, "meta.id is required")
		details = append(details, ValidationDetail{Element: ElementMeta, Field: "id", Message: "missing id"})
	}
	if strings.TrimSpace(d.Meta.Version) == "" {
		issues = append(issues, "meta.version is required")
		details = append(details, ValidationDetail{Element: ElementMeta, Field: "version", Message: "missing version"})
	}
	if strings.TrimSpace(d.Meta.Owner) == "" {
		issues = append(issues, "meta.owner is required")
		details = append(details, ValidationDetail{Element: ElementMeta, Field: "owner", Message: "missing owner"})
	}
	nameSeen := make(map[string]struct{})
	inputIndex := 0
	for _, in := range d.Inputs {
		if strings.TrimSpace(in.Name) == "" {
			issues = append(issues, "input.name is required")
			details = append(details, ValidationDetail{Element: ElementInput, Field: "name", Message: "missing name"})
		}
		if _, ok := nameSeen[in.Name]; ok && in.Name != "" {
			issues = append(issues, fmt.Sprintf("duplicate input name %q", in.Name))
			details = append(details, ValidationDetail{Element: ElementInput, Field: "name", Message: "duplicate name " + in.Name})
		}
		nameSeen[in.Name] = struct{}{}
		if strings.TrimSpace(in.Name) == "" {
			details = append(details, ValidationDetail{Element: ElementInput, Field: "name", Message: fmt.Sprintf("input %d missing name", inputIndex)})
		}
		inputIndex++
	}
	for _, doc := range d.Documents {
		if strings.TrimSpace(doc.Src) == "" {
			issues = append(issues, "document src is required")
			details = append(details, ValidationDetail{Element: ElementDocument, Field: "src", Message: "missing src"})
		}
	}
	for _, st := range d.Styles {
		for _, out := range st.Outputs {
			if strings.TrimSpace(out.Format) == "" {
				issues = append(issues, "style output format is required")
				details = append(details, ValidationDetail{Element: ElementStyle, Field: "format", Message: "missing format"})
			}
		}
	}
	toolNames := make(map[string]struct{})
	for _, td := range d.ToolDefs {
		name := strings.TrimSpace(td.Name)
		if name == "" {
			issues = append(issues, "tool-definition name is required")
			details = append(details, ValidationDetail{Element: ElementToolDefinition, Field: "name", Message: "missing name"})
		}
		if name != "" {
			if _, ok := toolNames[name]; ok {
				issues = append(issues, fmt.Sprintf("duplicate tool-definition name %q", name))
				details = append(details, ValidationDetail{Element: ElementToolDefinition, Field: "name", Message: "duplicate name " + name})
			}
			toolNames[name] = struct{}{}
		}
	}
	toolReqs := make(map[string]string)
	for i, tr := range d.ToolReqs {
		id := strings.TrimSpace(tr.ID)
		name := strings.TrimSpace(tr.Name)
		if id == "" {
			issues = append(issues, "tool-request id is required")
			details = append(details, ValidationDetail{Element: ElementToolRequest, Field: "id", Message: "missing id"})
		}
		if name == "" {
			issues = append(issues, "tool-request name is required")
			details = append(details, ValidationDetail{Element: ElementToolRequest, Field: "name", Message: "missing name"})
		}
		if name != "" {
			if _, ok := toolNames[name]; !ok {
				issues = append(issues, fmt.Sprintf("tool-request %q references unknown tool-definition %q", labelOrIndex(id, i), name))
				details = append(details, ValidationDetail{Element: ElementToolRequest, Field: "name", Message: "unknown tool-definition " + name})
			}
		}
		if id != "" {
			if existing, ok := toolReqs[id]; ok {
				issues = append(issues, fmt.Sprintf("duplicate tool-request id %q", id))
				details = append(details, ValidationDetail{Element: ElementToolRequest, Field: "id", Message: "duplicate id " + id + " (also used by " + existing + ")"})
			} else {
				toolReqs[id] = name
			}
		}
	}
	for i, tr := range d.ToolResps {
		id := strings.TrimSpace(tr.ID)
		name := strings.TrimSpace(tr.Name)
		if id == "" {
			issues = append(issues, "tool-response id is required")
			details = append(details, ValidationDetail{Element: ElementToolResponse, Field: "id", Message: "missing id"})
		}
		if name == "" {
			issues = append(issues, "tool-response name is required")
			details = append(details, ValidationDetail{Element: ElementToolResponse, Field: "name", Message: "missing name"})
		}
		validateToolReference("tool-response", i, id, name, toolNames, toolReqs, ElementToolResponse, &issues, &details)
	}
	for i, tr := range d.ToolResults {
		id := strings.TrimSpace(tr.ID)
		name := strings.TrimSpace(tr.Name)
		if id == "" {
			issues = append(issues, "tool-result id is required")
			details = append(details, ValidationDetail{Element: ElementToolResult, Field: "id", Message: "missing id"})
		}
		if name == "" {
			issues = append(issues, "tool-result name is required")
			details = append(details, ValidationDetail{Element: ElementToolResult, Field: "name", Message: "missing name"})
		}
		validateToolReference("tool-result", i, id, name, toolNames, toolReqs, ElementToolResult, &issues, &details)
	}
	for i, tr := range d.ToolErrors {
		id := strings.TrimSpace(tr.ID)
		name := strings.TrimSpace(tr.Name)
		if id == "" {
			issues = append(issues, "tool-error id is required")
			details = append(details, ValidationDetail{Element: ElementToolError, Field: "id", Message: "missing id"})
		}
		if name == "" {
			issues = append(issues, "tool-error name is required")
			details = append(details, ValidationDetail{Element: ElementToolError, Field: "name", Message: "missing name"})
		}
		validateToolReference("tool-error", i, id, name, toolNames, toolReqs, ElementToolError, &issues, &details)
	}
	if d.hasSchema() && strings.TrimSpace(d.Schema.Body) == "" && len(d.Schema.Attrs) == 0 {
		issues = append(issues, "output-schema requires body or attributes")
		details = append(details, ValidationDetail{Element: ElementOutputSchema, Message: "missing schema content"})
	}
	for _, img := range d.Images {
		if strings.TrimSpace(img.Src) == "" && strings.TrimSpace(img.Body) == "" {
			issues = append(issues, "img requires src or inline body")
			details = append(details, ValidationDetail{Element: ElementImage, Field: "src", Message: "missing src/body"})
		}
	}
	for i, dg := range d.Diagrams {
		if err := ValidateDiagram(dg); err != nil {
			var ve *ValidationError
			if errors.As(err, &ve) {
				for _, issue := range ve.Issues {
					issues = append(issues, fmt.Sprintf("diagram[%d]: %s", i, issue))
				}
				for _, det := range ve.Details {
					if det.Element == "" {
						det.Element = ElementDiagram
					}
					if det.Message == "" && len(ve.Issues) > 0 {
						det.Message = ve.Issues[0]
					}
					details = append(details, det)
				}
			} else {
				issues = append(issues, fmt.Sprintf("diagram[%d]: %v", i, err))
				details = append(details, ValidationDetail{Element: ElementDiagram, Message: err.Error()})
			}
		}
	}
	for i, h := range d.Hints {
		if strings.TrimSpace(h.Body) == "" {
			issues = append(issues, fmt.Sprintf("hint[%d] requires body content", i))
			details = append(details, ValidationDetail{Element: ElementHint, Message: "missing body"})
		}
	}
	for i, ex := range d.Examples {
		if strings.TrimSpace(ex.Body) == "" {
			issues = append(issues, fmt.Sprintf("example[%d] requires body content", i))
			details = append(details, ValidationDetail{Element: ElementExample, Message: "missing body"})
		}
	}
	for i, cp := range d.ContentParts {
		if strings.TrimSpace(cp.Body) == "" {
			issues = append(issues, fmt.Sprintf("cp[%d] requires body content", i))
			details = append(details, ValidationDetail{Element: ElementContentPart, Message: "missing body"})
		}
	}
	for i, obj := range d.Objects {
		if strings.TrimSpace(obj.Data) == "" && strings.TrimSpace(obj.Body) == "" {
			issues = append(issues, fmt.Sprintf("object[%d] requires data or body", i))
			details = append(details, ValidationDetail{Element: ElementObject, Message: "missing data/body"})
		}
	}
	if len(issues) == 0 {
		return nil
	}
	return &POMLError{
		Type:    ErrValidate,
		Message: "validation failed",
		Err: &ValidationError{
			Issues:  issues,
			Details: details,
		},
	}
}

func labelOrIndex(id string, idx int) string {
	if strings.TrimSpace(id) != "" {
		return id
	}
	return fmt.Sprintf("#%d", idx)
}

func validateToolReference(kind string, idx int, id string, name string, toolNames map[string]struct{}, toolReqs map[string]string, element ElementType, issues *[]string, details *[]ValidationDetail) {
	if name != "" {
		if _, ok := toolNames[name]; !ok {
			*issues = append(*issues, fmt.Sprintf("%s %q references unknown tool-definition %q", kind, labelOrIndex(id, idx), name))
			*details = append(*details, ValidationDetail{Element: element, Field: "name", Message: "unknown tool-definition " + name})
		}
	}
	if id == "" {
		return
	}
	reqName, ok := toolReqs[id]
	if !ok {
		*issues = append(*issues, fmt.Sprintf("%s id %q does not match a tool-request", kind, id))
		*details = append(*details, ValidationDetail{Element: element, Field: "id", Message: "missing tool-request for id " + id})
		return
	}
	if name != "" && reqName != "" && name != reqName {
		*issues = append(*issues, fmt.Sprintf("%s id %q uses tool %q but request used %q", kind, id, name, reqName))
		*details = append(*details, ValidationDetail{Element: element, Field: "name", Message: "mismatched tool for id " + id})
	}
}

// Walk iterates over elements in preserved order (if available) and invokes fn.
func (d Document) Walk(fn func(Element, ElementPayload) error) error {
	if fn == nil {
		return nil
	}
	elems := d.resolveOrder()
	for _, el := range elems {
		payload := d.payloadFor(el)
		if err := fn(el, payload); err != nil {
			return err
		}
	}
	return nil
}

// ElementByID returns the element by stable ID plus its payload.
func (d Document) ElementByID(id string) (Element, ElementPayload, bool) {
	for _, el := range d.resolveOrder() {
		if el.ID == id {
			return el, d.payloadFor(el), true
		}
	}
	return Element{}, ElementPayload{}, false
}

// Mutate walks elements and allows controlled insert/replace/remove via Mutator.
func (d *Document) Mutate(fn func(Element, ElementPayload, *Mutator) error) error {
	if fn == nil {
		return nil
	}
	m := &Mutator{doc: d}
	// Iterate over a snapshot so removals won't skip elements; new inserts are not visited in the same pass.
	snapshot := append([]Element(nil), d.resolveOrder()...)
	for _, el := range snapshot {
		payload := d.payloadFor(el)
		if err := fn(el, payload, m); err != nil {
			return err
		}
		if m.modified {
			d.reindex()
			m.modified = false
		}
	}
	return nil
}

// ElementPayload resolves the concrete node for an Element.
type ElementPayload struct {
	Meta         *Meta
	Role         *Block
	Task         *Block
	Input        *Input
	DocRef       *DocRef
	Style        *Style
	Audio        *Media
	Video        *Media
	OutputFormat *OutputFormat
	Hint         *Hint
	Example      *Example
	ContentPart  *ContentPart
	Object       *ObjectTag
	Image        *Image
	Message      *Message
	ToolDef      *ToolDefinition
	ToolReq      *ToolRequest
	ToolResp     *ToolResponse
	ToolResult   *ToolResult
	ToolError    *ToolError
	Schema       *OutputSchema
	Runtime      *Runtime
	Diagram      *Diagram
	Raw          string
}

// Mutator wraps mutation helpers for a Document walk.
type Mutator struct {
	doc      *Document
	modified bool
}

// MarkModified flags that the caller changed the document directly via payload.
func (m *Mutator) MarkModified() {
	m.modified = true
}

// ReplaceBody updates the textual body of role/task/input/style nodes.
func (m *Mutator) ReplaceBody(el Element, body string) {
	d := m.doc
	switch el.Type {
	case ElementRole:
		d.Role.Body = body
	case ElementTask:
		if el.Index >= 0 && el.Index < len(d.Tasks) {
			d.Tasks[el.Index].Body = body
		}
	case ElementInput:
		if el.Index >= 0 && el.Index < len(d.Inputs) {
			d.Inputs[el.Index].Body = body
		}
	case ElementStyle:
		if el.Index >= 0 && el.Index < len(d.Styles) && len(d.Styles[el.Index].Outputs) > 0 {
			// Update first output body; callers can MarkModified for more complex changes.
			d.Styles[el.Index].Outputs[0].Body = body
		}
	case ElementOutputFormat:
		if el.Index >= 0 && el.Index < len(d.OutFormats) {
			d.OutFormats[el.Index].Body = body
		}
	case ElementHumanMsg, ElementAssistantMsg, ElementSystemMsg:
		if el.Index >= 0 && el.Index < len(d.Messages) {
			d.Messages[el.Index].Body = body
		}
	case ElementToolResponse:
		if el.Index >= 0 && el.Index < len(d.ToolResps) {
			d.ToolResps[el.Index].Body = body
		}
	case ElementOutputSchema:
		d.Schema.Body = body
	case ElementImage:
		if el.Index >= 0 && el.Index < len(d.Images) {
			d.Images[el.Index].Body = body
		}
	}
	m.modified = true
}

// Remove deletes the given element and its backing slice entry (where applicable).
func (m *Mutator) Remove(el Element) {
	d := m.doc
	switch el.Type {
	case ElementTask:
		if el.Index >= 0 && el.Index < len(d.Tasks) {
			d.Tasks = append(d.Tasks[:el.Index], d.Tasks[el.Index+1:]...)
		}
	case ElementInput:
		if el.Index >= 0 && el.Index < len(d.Inputs) {
			d.Inputs = append(d.Inputs[:el.Index], d.Inputs[el.Index+1:]...)
		}
	case ElementDocument:
		if el.Index >= 0 && el.Index < len(d.Documents) {
			d.Documents = append(d.Documents[:el.Index], d.Documents[el.Index+1:]...)
		}
	case ElementStyle:
		if el.Index >= 0 && el.Index < len(d.Styles) {
			d.Styles = append(d.Styles[:el.Index], d.Styles[el.Index+1:]...)
		}
	case ElementOutputFormat:
		if el.Index >= 0 && el.Index < len(d.OutFormats) {
			d.OutFormats = append(d.OutFormats[:el.Index], d.OutFormats[el.Index+1:]...)
		}
	case ElementRole:
		d.Role = Block{}
	case ElementMeta:
		d.Meta = Meta{}
	case ElementHumanMsg, ElementAssistantMsg, ElementSystemMsg:
		if el.Index >= 0 && el.Index < len(d.Messages) {
			d.Messages = append(d.Messages[:el.Index], d.Messages[el.Index+1:]...)
		}
	case ElementToolDefinition:
		if el.Index >= 0 && el.Index < len(d.ToolDefs) {
			d.ToolDefs = append(d.ToolDefs[:el.Index], d.ToolDefs[el.Index+1:]...)
		}
	case ElementToolRequest:
		if el.Index >= 0 && el.Index < len(d.ToolReqs) {
			d.ToolReqs = append(d.ToolReqs[:el.Index], d.ToolReqs[el.Index+1:]...)
		}
	case ElementToolResponse:
		if el.Index >= 0 && el.Index < len(d.ToolResps) {
			d.ToolResps = append(d.ToolResps[:el.Index], d.ToolResps[el.Index+1:]...)
		}
	case ElementOutputSchema:
		d.Schema = OutputSchema{}
	case ElementRuntime:
		if el.Index >= 0 && el.Index < len(d.Runtimes) {
			d.Runtimes = append(d.Runtimes[:el.Index], d.Runtimes[el.Index+1:]...)
		}
	case ElementImage:
		if el.Index >= 0 && el.Index < len(d.Images) {
			d.Images = append(d.Images[:el.Index], d.Images[el.Index+1:]...)
		}
	}
	for i, e := range d.Elements {
		if e.ID == el.ID {
			d.Elements = append(d.Elements[:i], d.Elements[i+1:]...)
			break
		}
	}
	m.modified = true
}

// InsertTaskAfter inserts a task after the given element and returns the new element ID.
func (m *Mutator) InsertTaskAfter(after Element, body string) Element {
	d := m.doc
	d.Tasks = append(d.Tasks, Block{Body: body})
	newEl := d.newElement(ElementTask, len(d.Tasks)-1, "")
	d.insertElement(after, newEl)
	return newEl
}

// InsertInputAfter inserts an input after the given element.
func (m *Mutator) InsertInputAfter(after Element, in Input) Element {
	d := m.doc
	d.Inputs = append(d.Inputs, in)
	newEl := d.newElement(ElementInput, len(d.Inputs)-1, "")
	d.insertElement(after, newEl)
	return newEl
}

// InsertDocumentAfter inserts a document reference after the given element.
func (m *Mutator) InsertDocumentAfter(after Element, src string) Element {
	d := m.doc
	d.Documents = append(d.Documents, DocRef{Src: src})
	newEl := d.newElement(ElementDocument, len(d.Documents)-1, "")
	d.insertElement(after, newEl)
	return newEl
}

// InsertStyleAfter inserts a style after the given element.
func (m *Mutator) InsertStyleAfter(after Element, st Style) Element {
	d := m.doc
	d.Styles = append(d.Styles, st)
	newEl := d.newElement(ElementStyle, len(d.Styles)-1, "")
	d.insertElement(after, newEl)
	return newEl
}

// InsertBefore inserts newEl before the given element.
func (m *Mutator) InsertBefore(before Element, newEl Element) {
	d := m.doc
	pos := len(d.Elements)
	for i, e := range d.Elements {
		if e.ID == before.ID {
			pos = i
			break
		}
	}
	if newEl.ID == "" {
		newEl.ID = d.freshID()
	}
	d.Elements = append(d.Elements[:pos], append([]Element{newEl}, d.Elements[pos:]...)...)
	d.reindex()
	m.modified = true
}

func (d *Document) insertElement(after Element, newEl Element) {
	pos := len(d.Elements)
	for i, e := range d.Elements {
		if e.ID == after.ID {
			pos = i + 1
			break
		}
	}
	if newEl.ID == "" {
		newEl.ID = d.freshID()
	}
	if newEl.Parent == "" {
		newEl.Parent = after.Parent
	}
	d.Elements = append(d.Elements[:pos], append([]Element{newEl}, d.Elements[pos:]...)...)
	d.reindex()
}

func parseWithOptions(r io.Reader, opts ParseOptions) (Document, error) {
	dec := xml.NewDecoder(r)
	dec.Strict = true

	for {
		tok, err := dec.Token()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return Document{}, fmt.Errorf("parse poml: unexpected EOF (missing <poml> root?)")
			}
			return Document{}, wrapXMLError(err, "parse poml")
		}
		start, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		if start.Name.Local != "poml" {
			return Document{}, &POMLError{
				Type:    ErrDecode,
				Message: fmt.Sprintf("parse poml: expected <poml> root, got <%s>", start.Name.Local),
			}
		}
		doc, err := decodePoml(dec, opts)
		if err != nil {
			return Document{}, err
		}
		if opts.Validate {
			if err := doc.Validate(); err != nil {
				return Document{}, err
			}
		}
		return doc, nil
	}
}

func decodePoml(dec *xml.Decoder, opts ParseOptions) (Document, error) {
	var doc Document
	doc.nextID = 1
	var lastElement *Element
	pending := ""
	preserveWS := opts.PreserveWhitespace
	for {
		tok, err := dec.Token()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return doc, fmt.Errorf("parse poml: unexpected EOF before </poml>")
			}
			return doc, wrapXMLError(err, "parse poml")
		}
		switch t := tok.(type) {
		case xml.CharData:
			if preserveWS {
				pending += string(t)
			}
		case xml.Comment:
			if preserveWS {
				pending += renderToken(t)
			}
		case xml.StartElement:
			leading := pending
			pending = ""
			switch t.Name.Local {
			case "meta":
				var m Meta
				if err := dec.DecodeElement(&m, &t); err != nil {
					return doc, wrapXMLError(err, "<meta>")
				}
				doc.Meta = m
				el := doc.newElement(ElementMeta, -1, "")
				if preserveWS {
					el.Leading = leading
				}
				doc.Elements = append(doc.Elements, el)
			case "role":
				var b Block
				if err := dec.DecodeElement(&b, &t); err != nil {
					return doc, wrapXMLError(err, "<role>")
				}
				doc.Role = b
				el := doc.newElement(ElementRole, -1, "")
				if preserveWS {
					el.Leading = leading
				}
				doc.Elements = append(doc.Elements, el)
			case "task":
				var b Block
				if err := dec.DecodeElement(&b, &t); err != nil {
					return doc, wrapXMLError(err, "<task>")
				}
				doc.Tasks = append(doc.Tasks, b)
				el := doc.newElement(ElementTask, len(doc.Tasks)-1, "")
				if preserveWS {
					el.Leading = leading
				}
				doc.Elements = append(doc.Elements, el)
			case "input":
				var in Input
				if err := dec.DecodeElement(&in, &t); err != nil {
					return doc, wrapXMLError(err, "<input>")
				}
				doc.Inputs = append(doc.Inputs, in)
				el := doc.newElement(ElementInput, len(doc.Inputs)-1, "")
				if preserveWS {
					el.Leading = leading
				}
				doc.Elements = append(doc.Elements, el)
			case "document", "Document":
				var dr DocRef
				if err := dec.DecodeElement(&dr, &t); err != nil {
					return doc, wrapXMLError(err, "<document>")
				}
				doc.Documents = append(doc.Documents, dr)
				el := doc.newElement(ElementDocument, len(doc.Documents)-1, t.Name.Local)
				if preserveWS {
					el.Leading = leading
				}
				doc.Elements = append(doc.Elements, el)
			case "style":
				var st Style
				if err := dec.DecodeElement(&st, &t); err != nil {
					return doc, wrapXMLError(err, "<style>")
				}
				doc.Styles = append(doc.Styles, st)
				el := doc.newElement(ElementStyle, len(doc.Styles)-1, "")
				if preserveWS {
					el.Leading = leading
				}
				doc.Elements = append(doc.Elements, el)
			case "hint":
				var h Hint
				if err := dec.DecodeElement(&h, &t); err != nil {
					return doc, wrapXMLError(err, "<hint>")
				}
				doc.Hints = append(doc.Hints, h)
				el := doc.newElement(ElementHint, len(doc.Hints)-1, "")
				if preserveWS {
					el.Leading = leading
				}
				doc.Elements = append(doc.Elements, el)
			case "example":
				var ex Example
				if err := dec.DecodeElement(&ex, &t); err != nil {
					return doc, wrapXMLError(err, "<example>")
				}
				doc.Examples = append(doc.Examples, ex)
				el := doc.newElement(ElementExample, len(doc.Examples)-1, "")
				if preserveWS {
					el.Leading = leading
				}
				doc.Elements = append(doc.Elements, el)
			case "cp":
				var cp ContentPart
				if err := dec.DecodeElement(&cp, &t); err != nil {
					return doc, wrapXMLError(err, "<cp>")
				}
				doc.ContentParts = append(doc.ContentParts, cp)
				el := doc.newElement(ElementContentPart, len(doc.ContentParts)-1, "")
				if preserveWS {
					el.Leading = leading
				}
				doc.Elements = append(doc.Elements, el)
			case "human-msg", "assistant-msg", "system-msg", "ai-msg":
				var msg Message
				if err := dec.DecodeElement(&msg, &t); err != nil {
					return doc, wrapXMLError(err, "<msg>")
				}
				msg.Role = strings.TrimSuffix(t.Name.Local, "-msg")
				if t.Name.Local == "ai-msg" {
					msg.Role = "assistant"
				}
				doc.Messages = append(doc.Messages, msg)
				elType := ElementHumanMsg
				switch msg.Role {
				case "assistant":
					elType = ElementAssistantMsg
				case "system":
					elType = ElementSystemMsg
				}
				el := doc.newElement(elType, len(doc.Messages)-1, "")
				if preserveWS {
					el.Leading = leading
				}
				doc.Elements = append(doc.Elements, el)
			case "tool-definition", "tool":
				var td ToolDefinition
				if err := dec.DecodeElement(&td, &t); err != nil {
					return doc, wrapXMLError(err, "<tool-definition>")
				}
				doc.ToolDefs = append(doc.ToolDefs, td)
				el := doc.newElement(ElementToolDefinition, len(doc.ToolDefs)-1, t.Name.Local)
				if preserveWS {
					el.Leading = leading
				}
				doc.Elements = append(doc.Elements, el)
			case "tool-request":
				var tr ToolRequest
				if err := dec.DecodeElement(&tr, &t); err != nil {
					return doc, wrapXMLError(err, "<tool-request>")
				}
				doc.ToolReqs = append(doc.ToolReqs, tr)
				el := doc.newElement(ElementToolRequest, len(doc.ToolReqs)-1, "")
				if preserveWS {
					el.Leading = leading
				}
				doc.Elements = append(doc.Elements, el)
			case "tool-response":
				var tr ToolResponse
				if err := dec.DecodeElement(&tr, &t); err != nil {
					return doc, wrapXMLError(err, "<tool-response>")
				}
				doc.ToolResps = append(doc.ToolResps, tr)
				el := doc.newElement(ElementToolResponse, len(doc.ToolResps)-1, "")
				if preserveWS {
					el.Leading = leading
				}
				doc.Elements = append(doc.Elements, el)
			case "tool-result":
				var tr ToolResult
				if err := dec.DecodeElement(&tr, &t); err != nil {
					return doc, wrapXMLError(err, "<tool-result>")
				}
				doc.ToolResults = append(doc.ToolResults, tr)
				el := doc.newElement(ElementToolResult, len(doc.ToolResults)-1, "")
				if preserveWS {
					el.Leading = leading
				}
				doc.Elements = append(doc.Elements, el)
			case "tool-error":
				var te ToolError
				if err := dec.DecodeElement(&te, &t); err != nil {
					return doc, wrapXMLError(err, "<tool-error>")
				}
				doc.ToolErrors = append(doc.ToolErrors, te)
				el := doc.newElement(ElementToolError, len(doc.ToolErrors)-1, "")
				if preserveWS {
					el.Leading = leading
				}
				doc.Elements = append(doc.Elements, el)
			case "output-schema":
				var os OutputSchema
				if err := dec.DecodeElement(&os, &t); err != nil {
					return doc, wrapXMLError(err, "<output-schema>")
				}
				doc.Schema = os
				el := doc.newElement(ElementOutputSchema, -1, "")
				if preserveWS {
					el.Leading = leading
				}
				doc.Elements = append(doc.Elements, el)
			case "output-format":
				var of OutputFormat
				if err := dec.DecodeElement(&of, &t); err != nil {
					return doc, wrapXMLError(err, "<output-format>")
				}
				doc.OutFormats = append(doc.OutFormats, of)
				el := doc.newElement(ElementOutputFormat, len(doc.OutFormats)-1, "")
				if preserveWS {
					el.Leading = leading
				}
				doc.Elements = append(doc.Elements, el)
			case "runtime":
				var rt Runtime
				if err := dec.DecodeElement(&rt, &t); err != nil {
					return doc, wrapXMLError(err, "<runtime>")
				}
				doc.Runtimes = append(doc.Runtimes, rt)
				el := doc.newElement(ElementRuntime, len(doc.Runtimes)-1, "")
				if preserveWS {
					el.Leading = leading
				}
				doc.Elements = append(doc.Elements, el)
			case "img":
				var im Image
				if err := dec.DecodeElement(&im, &t); err != nil {
					return doc, wrapXMLError(err, "<img>")
				}
				doc.Images = append(doc.Images, im)
				el := doc.newElement(ElementImage, len(doc.Images)-1, "")
				if preserveWS {
					el.Leading = leading
				}
				doc.Elements = append(doc.Elements, el)
			case "audio":
				var au Media
				if err := dec.DecodeElement(&au, &t); err != nil {
					return doc, wrapXMLError(err, "<audio>")
				}
				doc.Audios = append(doc.Audios, au)
				el := doc.newElement(ElementAudio, len(doc.Audios)-1, "")
				if preserveWS {
					el.Leading = leading
				}
				doc.Elements = append(doc.Elements, el)
			case "video":
				var vd Media
				if err := dec.DecodeElement(&vd, &t); err != nil {
					return doc, wrapXMLError(err, "<video>")
				}
				doc.Videos = append(doc.Videos, vd)
				el := doc.newElement(ElementVideo, len(doc.Videos)-1, "")
				if preserveWS {
					el.Leading = leading
				}
				doc.Elements = append(doc.Elements, el)
			case "object", "Object":
				var obj ObjectTag
				if err := dec.DecodeElement(&obj, &t); err != nil {
					return doc, wrapXMLError(err, "<object>")
				}
				doc.Objects = append(doc.Objects, obj)
				el := doc.newElement(ElementObject, len(doc.Objects)-1, t.Name.Local)
				if preserveWS {
					el.Leading = leading
				}
				doc.Elements = append(doc.Elements, el)
			case "diagram":
				var dg Diagram
				if err := dec.DecodeElement(&dg, &t); err != nil {
					return doc, wrapXMLError(err, "<diagram>")
				}
				doc.Diagrams = append(doc.Diagrams, dg)
				el := doc.newElement(ElementDiagram, len(doc.Diagrams)-1, "")
				if preserveWS {
					el.Leading = leading
				}
				doc.Elements = append(doc.Elements, el)
			default:
				// Preserve unknown elements as raw where possible.
				raw, err := consumeRaw(dec, t)
				if err != nil {
					return doc, wrapXMLError(err, fmt.Sprintf("<%s>", t.Name.Local))
				}
				el := doc.newElement(ElementUnknown, -1, t.Name.Local, raw)
				if preserveWS {
					el.Leading = leading
				}
				doc.Elements = append(doc.Elements, el)
			}
			if preserveWS && lastElement != nil && pending != "" {
				lastElement.Trailing = pending
			}
			lastElement = &doc.Elements[len(doc.Elements)-1]
			pending = ""
		case xml.EndElement:
			if t.Name.Local == "poml" {
				if preserveWS && lastElement != nil && pending != "" {
					lastElement.Trailing = pending
				}
				return doc, nil
			}
		}
	}
}

// consumeRaw reads the current element (start already consumed) and returns the raw XML string.
func consumeRaw(dec *xml.Decoder, start xml.StartElement) (string, error) {
	var buf bytes.Buffer
	enc := xml.NewEncoder(&buf)
	if err := enc.EncodeToken(start); err != nil {
		return "", err
	}
	depth := 1
	for depth > 0 {
		tok, err := dec.Token()
		if err != nil {
			return "", err
		}
		switch tok.(type) {
		case xml.StartElement:
			depth++
		case xml.EndElement:
			depth--
		}
		if err := enc.EncodeToken(tok); err != nil {
			return "", err
		}
	}
	if err := enc.Flush(); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// encodeDocument writes a poml root element with ordered children.
func encodeDocument(enc *xml.Encoder, out io.Writer, doc Document, opts EncodeOptions) error {
	start := xml.StartElement{Name: xml.Name{Local: "poml"}}
	if err := enc.EncodeToken(start); err != nil {
		return err
	}
	for _, el := range doc.resolveOrderWithFallback(opts.PreserveOrder) {
		if err := encodeElement(enc, out, doc, el, opts); err != nil {
			return err
		}
	}
	return enc.EncodeToken(start.End())
}

func encodeElement(enc *xml.Encoder, out io.Writer, doc Document, el Element, opts EncodeOptions) error {
	if opts.PreserveWS && el.Leading != "" {
		if err := enc.Flush(); err != nil {
			return err
		}
		if _, err := io.WriteString(out, el.Leading); err != nil {
			return err
		}
	}
	var err error
	switch el.Type {
	case ElementMeta:
		err = enc.EncodeElement(doc.Meta, xml.StartElement{Name: xml.Name{Local: "meta"}})
	case ElementRole:
		err = enc.EncodeElement(doc.Role, xml.StartElement{Name: xml.Name{Local: "role"}})
	case ElementTask:
		if el.Index < 0 || el.Index >= len(doc.Tasks) {
			return fmt.Errorf("encode task: index %d out of range", el.Index)
		}
		err = enc.EncodeElement(doc.Tasks[el.Index], xml.StartElement{Name: xml.Name{Local: "task"}})
	case ElementInput:
		if el.Index < 0 || el.Index >= len(doc.Inputs) {
			return fmt.Errorf("encode input: index %d out of range", el.Index)
		}
		err = enc.EncodeElement(doc.Inputs[el.Index], xml.StartElement{Name: xml.Name{Local: "input"}})
	case ElementDocument:
		if el.Index < 0 || el.Index >= len(doc.Documents) {
			return fmt.Errorf("encode document: index %d out of range", el.Index)
		}
		tag := "document"
		if el.Name == "Document" {
			tag = el.Name
		}
		err = enc.EncodeElement(doc.Documents[el.Index], xml.StartElement{Name: xml.Name{Local: tag}})
	case ElementStyle:
		if el.Index < 0 || el.Index >= len(doc.Styles) {
			return fmt.Errorf("encode style: index %d out of range", el.Index)
		}
		err = enc.EncodeElement(doc.Styles[el.Index], xml.StartElement{Name: xml.Name{Local: "style"}})
	case ElementHint:
		if el.Index < 0 || el.Index >= len(doc.Hints) {
			return fmt.Errorf("encode hint: index %d out of range", el.Index)
		}
		err = enc.EncodeElement(doc.Hints[el.Index], xml.StartElement{Name: xml.Name{Local: "hint"}})
	case ElementExample:
		if el.Index < 0 || el.Index >= len(doc.Examples) {
			return fmt.Errorf("encode example: index %d out of range", el.Index)
		}
		err = enc.EncodeElement(doc.Examples[el.Index], xml.StartElement{Name: xml.Name{Local: "example"}})
	case ElementContentPart:
		if el.Index < 0 || el.Index >= len(doc.ContentParts) {
			return fmt.Errorf("encode cp: index %d out of range", el.Index)
		}
		err = enc.EncodeElement(doc.ContentParts[el.Index], xml.StartElement{Name: xml.Name{Local: "cp"}})
	case ElementHumanMsg, ElementAssistantMsg, ElementSystemMsg:
		if el.Index < 0 || el.Index >= len(doc.Messages) {
			return fmt.Errorf("encode message: index %d out of range", el.Index)
		}
		tag := "human-msg"
		switch el.Type {
		case ElementAssistantMsg:
			tag = "assistant-msg"
		case ElementSystemMsg:
			tag = "system-msg"
		}
		err = enc.EncodeElement(doc.Messages[el.Index], xml.StartElement{Name: xml.Name{Local: tag}})
	case ElementToolDefinition:
		if el.Index < 0 || el.Index >= len(doc.ToolDefs) {
			return fmt.Errorf("encode tool definition: index %d out of range", el.Index)
		}
		tag := "tool-definition"
		if el.Name == "tool" {
			tag = el.Name
		}
		err = enc.EncodeElement(doc.ToolDefs[el.Index], xml.StartElement{Name: xml.Name{Local: tag}})
	case ElementToolRequest:
		if el.Index < 0 || el.Index >= len(doc.ToolReqs) {
			return fmt.Errorf("encode tool request: index %d out of range", el.Index)
		}
		err = enc.EncodeElement(doc.ToolReqs[el.Index], xml.StartElement{Name: xml.Name{Local: "tool-request"}})
	case ElementToolResponse:
		if el.Index < 0 || el.Index >= len(doc.ToolResps) {
			return fmt.Errorf("encode tool response: index %d out of range", el.Index)
		}
		err = enc.EncodeElement(doc.ToolResps[el.Index], xml.StartElement{Name: xml.Name{Local: "tool-response"}})
	case ElementToolResult:
		if el.Index < 0 || el.Index >= len(doc.ToolResults) {
			return fmt.Errorf("encode tool result: index %d out of range", el.Index)
		}
		err = enc.EncodeElement(doc.ToolResults[el.Index], xml.StartElement{Name: xml.Name{Local: "tool-result"}})
	case ElementToolError:
		if el.Index < 0 || el.Index >= len(doc.ToolErrors) {
			return fmt.Errorf("encode tool error: index %d out of range", el.Index)
		}
		err = enc.EncodeElement(doc.ToolErrors[el.Index], xml.StartElement{Name: xml.Name{Local: "tool-error"}})
	case ElementAudio:
		if el.Index < 0 || el.Index >= len(doc.Audios) {
			return fmt.Errorf("encode audio: index %d out of range", el.Index)
		}
		err = enc.EncodeElement(doc.Audios[el.Index], xml.StartElement{Name: xml.Name{Local: "audio"}})
	case ElementVideo:
		if el.Index < 0 || el.Index >= len(doc.Videos) {
			return fmt.Errorf("encode video: index %d out of range", el.Index)
		}
		err = enc.EncodeElement(doc.Videos[el.Index], xml.StartElement{Name: xml.Name{Local: "video"}})
	case ElementOutputSchema:
		err = enc.EncodeElement(doc.Schema, xml.StartElement{Name: xml.Name{Local: "output-schema"}})
	case ElementOutputFormat:
		if el.Index < 0 || el.Index >= len(doc.OutFormats) {
			return fmt.Errorf("encode output-format: index %d out of range", el.Index)
		}
		err = enc.EncodeElement(doc.OutFormats[el.Index], xml.StartElement{Name: xml.Name{Local: "output-format"}})
	case ElementRuntime:
		if el.Index < 0 || el.Index >= len(doc.Runtimes) {
			return fmt.Errorf("encode runtime: index %d out of range", el.Index)
		}
		err = enc.EncodeElement(doc.Runtimes[el.Index], xml.StartElement{Name: xml.Name{Local: "runtime"}})
	case ElementImage:
		if el.Index < 0 || el.Index >= len(doc.Images) {
			return fmt.Errorf("encode image: index %d out of range", el.Index)
		}
		err = enc.EncodeElement(doc.Images[el.Index], xml.StartElement{Name: xml.Name{Local: "img"}})
	case ElementObject:
		if el.Index < 0 || el.Index >= len(doc.Objects) {
			return fmt.Errorf("encode object: index %d out of range", el.Index)
		}
		tag := "object"
		if el.Name == "Object" {
			tag = el.Name
		}
		err = enc.EncodeElement(doc.Objects[el.Index], xml.StartElement{Name: xml.Name{Local: tag}})
	case ElementDiagram:
		if el.Index < 0 || el.Index >= len(doc.Diagrams) {
			return fmt.Errorf("encode diagram: index %d out of range", el.Index)
		}
		err = enc.EncodeElement(doc.Diagrams[el.Index], xml.StartElement{Name: xml.Name{Local: "diagram"}})
	case ElementUnknown:
		if el.RawXML == "" {
			return nil
		}
		if err = enc.Flush(); err == nil {
			_, err = io.WriteString(out, el.RawXML)
		}
	default:
	}
	if err != nil {
		return err
	}
	if opts.PreserveWS && el.Trailing != "" {
		if err := enc.Flush(); err != nil {
			return err
		}
		if _, err := io.WriteString(out, el.Trailing); err != nil {
			return err
		}
	}
	return nil
}

// resolveOrderWithFallback returns the preferred element ordering.
func (d *Document) resolveOrderWithFallback(preserve bool) []Element {
	if preserve && len(d.Elements) > 0 {
		return d.Elements
	}
	return d.defaultElements()
}

// resolveOrder returns Elements with default ordering if none are recorded.
func (d *Document) resolveOrder() []Element {
	return d.resolveOrderWithFallback(true)
}

// defaultElements builds a canonical ordering of known fields.
func (d *Document) defaultElements() []Element {
	var out []Element
	if (d.Meta != Meta{}) {
		out = append(out, d.newElement(ElementMeta, -1, ""))
	}
	if d.Role.Body != "" {
		out = append(out, d.newElement(ElementRole, -1, ""))
	}
	for i := range d.Tasks {
		out = append(out, d.newElement(ElementTask, i, ""))
	}
	for i := range d.Inputs {
		out = append(out, d.newElement(ElementInput, i, ""))
	}
	for i := range d.Documents {
		out = append(out, d.newElement(ElementDocument, i, ""))
	}
	for i := range d.Styles {
		out = append(out, d.newElement(ElementStyle, i, ""))
	}
	for i := range d.Hints {
		out = append(out, d.newElement(ElementHint, i, ""))
	}
	for i := range d.Examples {
		out = append(out, d.newElement(ElementExample, i, ""))
	}
	for i := range d.ContentParts {
		out = append(out, d.newElement(ElementContentPart, i, ""))
	}
	for i := range d.OutFormats {
		out = append(out, d.newElement(ElementOutputFormat, i, ""))
	}
	for i := range d.Messages {
		// Preserve role-specific element types.
		elType := ElementHumanMsg
		switch d.Messages[i].Role {
		case "assistant":
			elType = ElementAssistantMsg
		case "system":
			elType = ElementSystemMsg
		}
		out = append(out, d.newElement(elType, i, ""))
	}
	for i := range d.ToolDefs {
		out = append(out, d.newElement(ElementToolDefinition, i, ""))
	}
	for i := range d.ToolReqs {
		out = append(out, d.newElement(ElementToolRequest, i, ""))
	}
	for i := range d.ToolResps {
		out = append(out, d.newElement(ElementToolResponse, i, ""))
	}
	for i := range d.ToolResults {
		out = append(out, d.newElement(ElementToolResult, i, ""))
	}
	for i := range d.ToolErrors {
		out = append(out, d.newElement(ElementToolError, i, ""))
	}
	if d.hasSchema() {
		out = append(out, d.newElement(ElementOutputSchema, -1, ""))
	}
	for i := range d.Runtimes {
		out = append(out, d.newElement(ElementRuntime, i, ""))
	}
	for i := range d.Audios {
		out = append(out, d.newElement(ElementAudio, i, ""))
	}
	for i := range d.Videos {
		out = append(out, d.newElement(ElementVideo, i, ""))
	}
	for i := range d.Objects {
		out = append(out, d.newElement(ElementObject, i, ""))
	}
	for i := range d.Images {
		out = append(out, d.newElement(ElementImage, i, ""))
	}
	for i := range d.Diagrams {
		out = append(out, d.newElement(ElementDiagram, i, ""))
	}
	return out
}

func (d Document) hasSchema() bool {
	return d.Schema.Body != "" || len(d.Schema.Attrs) > 0
}

// payloadFor resolves concrete pointers for an element.
func (d Document) payloadFor(el Element) ElementPayload {
	switch el.Type {
	case ElementMeta:
		return ElementPayload{Meta: &d.Meta}
	case ElementRole:
		return ElementPayload{Role: &d.Role}
	case ElementTask:
		if el.Index >= 0 && el.Index < len(d.Tasks) {
			return ElementPayload{Task: &d.Tasks[el.Index]}
		}
	case ElementInput:
		if el.Index >= 0 && el.Index < len(d.Inputs) {
			return ElementPayload{Input: &d.Inputs[el.Index]}
		}
	case ElementDocument:
		if el.Index >= 0 && el.Index < len(d.Documents) {
			return ElementPayload{DocRef: &d.Documents[el.Index]}
		}
	case ElementStyle:
		if el.Index >= 0 && el.Index < len(d.Styles) {
			return ElementPayload{Style: &d.Styles[el.Index]}
		}
	case ElementAudio:
		if el.Index >= 0 && el.Index < len(d.Audios) {
			return ElementPayload{Audio: &d.Audios[el.Index]}
		}
	case ElementVideo:
		if el.Index >= 0 && el.Index < len(d.Videos) {
			return ElementPayload{Video: &d.Videos[el.Index]}
		}
	case ElementHint:
		if el.Index >= 0 && el.Index < len(d.Hints) {
			return ElementPayload{Hint: &d.Hints[el.Index]}
		}
	case ElementExample:
		if el.Index >= 0 && el.Index < len(d.Examples) {
			return ElementPayload{Example: &d.Examples[el.Index]}
		}
	case ElementContentPart:
		if el.Index >= 0 && el.Index < len(d.ContentParts) {
			return ElementPayload{ContentPart: &d.ContentParts[el.Index]}
		}
	case ElementOutputFormat:
		if el.Index >= 0 && el.Index < len(d.OutFormats) {
			return ElementPayload{OutputFormat: &d.OutFormats[el.Index]}
		}
	case ElementObject:
		if el.Index >= 0 && el.Index < len(d.Objects) {
			return ElementPayload{Object: &d.Objects[el.Index]}
		}
	case ElementImage:
		if el.Index >= 0 && el.Index < len(d.Images) {
			return ElementPayload{Image: &d.Images[el.Index]}
		}
	case ElementHumanMsg, ElementAssistantMsg, ElementSystemMsg:
		if el.Index >= 0 && el.Index < len(d.Messages) {
			return ElementPayload{Message: &d.Messages[el.Index]}
		}
	case ElementToolDefinition:
		if el.Index >= 0 && el.Index < len(d.ToolDefs) {
			return ElementPayload{ToolDef: &d.ToolDefs[el.Index]}
		}
	case ElementToolRequest:
		if el.Index >= 0 && el.Index < len(d.ToolReqs) {
			return ElementPayload{ToolReq: &d.ToolReqs[el.Index]}
		}
	case ElementToolResponse:
		if el.Index >= 0 && el.Index < len(d.ToolResps) {
			return ElementPayload{ToolResp: &d.ToolResps[el.Index]}
		}
	case ElementToolResult:
		if el.Index >= 0 && el.Index < len(d.ToolResults) {
			return ElementPayload{ToolResult: &d.ToolResults[el.Index]}
		}
	case ElementToolError:
		if el.Index >= 0 && el.Index < len(d.ToolErrors) {
			return ElementPayload{ToolError: &d.ToolErrors[el.Index]}
		}
	case ElementOutputSchema:
		if d.hasSchema() {
			return ElementPayload{Schema: &d.Schema}
		}
	case ElementRuntime:
		if el.Index >= 0 && el.Index < len(d.Runtimes) {
			return ElementPayload{Runtime: &d.Runtimes[el.Index]}
		}
	case ElementDiagram:
		if el.Index >= 0 && el.Index < len(d.Diagrams) {
			return ElementPayload{Diagram: &d.Diagrams[el.Index]}
		}
	case ElementUnknown:
		return ElementPayload{Raw: el.RawXML}
	}
	return ElementPayload{}
}

func wrapXMLError(err error, context string) error {
	var se *xml.SyntaxError
	if errors.As(err, &se) {
		return &POMLError{Type: ErrDecode, Message: fmt.Sprintf("%s (line %d)", context, se.Line), Err: err}
	}
	var ue *xml.UnmarshalError
	if errors.As(err, &ue) {
		return &POMLError{Type: ErrDecode, Message: context, Err: err}
	}
	return &POMLError{Type: ErrDecode, Message: context, Err: err}
}

func (d *Document) newElement(t ElementType, idx int, name string, raw ...string) Element {
	if d.nextID == 0 {
		d.nextID = 1
	}
	el := Element{
		Type:   t,
		Index:  idx,
		Name:   name,
		ID:     d.freshID(),
		Parent: rootParentID,
	}
	if len(raw) > 0 {
		el.RawXML = raw[0]
	}
	return el
}

func (d *Document) freshID() string {
	id := fmt.Sprintf("el-%d", d.nextID)
	d.nextID++
	return id
}

const rootParentID = "root"

func renderToken(tok xml.Token) string {
	var buf bytes.Buffer
	enc := xml.NewEncoder(&buf)
	_ = enc.EncodeToken(tok)
	_ = enc.Flush()
	return buf.String()
}

// reindex updates element indices to match current slice state after mutations.
func (d *Document) reindex() {
	taskIdx, inputIdx, docIdx, styleIdx, hintIdx, exIdx, cpIdx, outFmtIdx := 0, 0, 0, 0, 0, 0, 0, 0
	msgIdx, toolDefIdx, toolReqIdx, toolRespIdx, toolResultIdx, toolErrorIdx, runtimeIdx, audioIdx, videoIdx, objIdx, imageIdx, diagramIdx := 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0
	for i := range d.Elements {
		switch d.Elements[i].Type {
		case ElementTask:
			d.Elements[i].Index = taskIdx
			taskIdx++
		case ElementInput:
			d.Elements[i].Index = inputIdx
			inputIdx++
		case ElementDocument:
			d.Elements[i].Index = docIdx
			docIdx++
		case ElementStyle:
			d.Elements[i].Index = styleIdx
			styleIdx++
		case ElementHint:
			d.Elements[i].Index = hintIdx
			hintIdx++
		case ElementExample:
			d.Elements[i].Index = exIdx
			exIdx++
		case ElementContentPart:
			d.Elements[i].Index = cpIdx
			cpIdx++
		case ElementOutputFormat:
			d.Elements[i].Index = outFmtIdx
			outFmtIdx++
		case ElementHumanMsg, ElementAssistantMsg, ElementSystemMsg:
			d.Elements[i].Index = msgIdx
			msgIdx++
		case ElementToolDefinition:
			d.Elements[i].Index = toolDefIdx
			toolDefIdx++
		case ElementToolRequest:
			d.Elements[i].Index = toolReqIdx
			toolReqIdx++
		case ElementToolResponse:
			d.Elements[i].Index = toolRespIdx
			toolRespIdx++
		case ElementToolResult:
			d.Elements[i].Index = toolResultIdx
			toolResultIdx++
		case ElementToolError:
			d.Elements[i].Index = toolErrorIdx
			toolErrorIdx++
		case ElementRuntime:
			d.Elements[i].Index = runtimeIdx
			runtimeIdx++
		case ElementAudio:
			d.Elements[i].Index = audioIdx
			audioIdx++
		case ElementVideo:
			d.Elements[i].Index = videoIdx
			videoIdx++
		case ElementObject:
			d.Elements[i].Index = objIdx
			objIdx++
		case ElementImage:
			d.Elements[i].Index = imageIdx
			imageIdx++
		case ElementDiagram:
			d.Elements[i].Index = diagramIdx
			diagramIdx++
		}
	}
}
