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
	ElementMeta     ElementType = "meta"
	ElementRole     ElementType = "role"
	ElementTask     ElementType = "task"
	ElementInput    ElementType = "input"
	ElementDocument ElementType = "document"
	ElementStyle    ElementType = "style"
	ElementUnknown  ElementType = "unknown"
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
	Meta      Meta     `xml:"meta"`
	Role      Block    `xml:"role"`
	Tasks     []Block  `xml:"task"`
	Inputs    []Input  `xml:"input"`
	Documents []DocRef `xml:"document"`
	Styles    []Style  `xml:"style"`
	Elements  []Element
	rawPrefix string // leading text before root (e.g., XML decl); kept for future extension

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

// Output holds a single output format entry.
type Output struct {
	Format string     `xml:"format,attr"`
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
}

var defaultParseOptions = ParseOptions{PreserveWhitespace: true}

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

// ParseFile decodes a POML document from the given file path.
func ParseFile(path string) (Document, error) {
	f, err := os.Open(path)
	if err != nil {
		return Document{}, err
	}
	defer f.Close()
	return parseWithOptions(f, defaultParseOptions)
}

// ParseReader decodes a POML document from an io.Reader.
func ParseReader(r io.Reader) (Document, error) {
	return parseWithOptions(r, defaultParseOptions)
}

// ParseReaderWithOptions decodes a POML document with fidelity controls.
func ParseReaderWithOptions(r io.Reader, opts ParseOptions) (Document, error) {
	return parseWithOptions(r, opts)
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
	Meta   *Meta
	Role   *Block
	Task   *Block
	Input  *Input
	DocRef *DocRef
	Style  *Style
	Raw    string
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
	case ElementRole:
		d.Role = Block{}
	case ElementMeta:
		d.Meta = Meta{}
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
		return decodePoml(dec, opts)
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
			case "document":
				var dr DocRef
				if err := dec.DecodeElement(&dr, &t); err != nil {
					return doc, wrapXMLError(err, "<document>")
				}
				doc.Documents = append(doc.Documents, dr)
				el := doc.newElement(ElementDocument, len(doc.Documents)-1, "")
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
		err = enc.EncodeElement(doc.Documents[el.Index], xml.StartElement{Name: xml.Name{Local: "document"}})
	case ElementStyle:
		if el.Index < 0 || el.Index >= len(doc.Styles) {
			return fmt.Errorf("encode style: index %d out of range", el.Index)
		}
		err = enc.EncodeElement(doc.Styles[el.Index], xml.StartElement{Name: xml.Name{Local: "style"}})
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
func (d Document) resolveOrderWithFallback(preserve bool) []Element {
	if preserve && len(d.Elements) > 0 {
		return d.Elements
	}
	return d.defaultElements()
}

// resolveOrder returns Elements with default ordering if none are recorded.
func (d Document) resolveOrder() []Element {
	return d.resolveOrderWithFallback(true)
}

// defaultElements builds a canonical ordering of known fields.
func (d Document) defaultElements() []Element {
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
	return out
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
	taskIdx, inputIdx, docIdx, styleIdx := 0, 0, 0, 0
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
		}
	}
}
