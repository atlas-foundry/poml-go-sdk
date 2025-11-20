package poml

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strings"
)

// Document represents a POML file. Unknown elements are ignored but preserved in raw form when round-tripping.
type Document struct {
	Meta      Meta     `xml:"meta"`
	Role      string   `xml:"role"`
	Tasks     []Block  `xml:"task"`
	Inputs    []Input  `xml:"input"`
	Documents []DocRef `xml:"document"`
	Styles    []Style  `xml:"style"`
	rawPrefix string   // leading text before root (e.g., XML decl); kept for future extension
}

// Meta captures the id/version/owner fields under <meta>.
type Meta struct {
	ID      string `xml:"id"`
	Version string `xml:"version"`
	Owner   string `xml:"owner"`
}

// Block holds free-form body content for task/role/style sections.
type Block struct {
	Body string `xml:",innerxml"`
}

// Input represents a named input block.
type Input struct {
	Name     string `xml:"name,attr"`
	Required bool   `xml:"required,attr"`
	Body     string `xml:",innerxml"`
}

// DocRef links to an external source document.
type DocRef struct {
	Src string `xml:"src,attr"`
}

// Style represents an <style><output format=...> block.
type Style struct {
	Outputs []Output `xml:"output"`
}

// Output holds a single output format entry.
type Output struct {
	Format string `xml:"format,attr"`
	Body   string `xml:",innerxml"`
}

// ParseString decodes a POML document from a string.
func ParseString(body string) (Document, error) {
	return parse(strings.NewReader(body))
}

// ParseFile decodes a POML document from the given file path.
func ParseFile(path string) (Document, error) {
	f, err := os.Open(path)
	if err != nil {
		return Document{}, err
	}
	defer f.Close()
	return parse(f)
}

// Encode writes the POML document back to XML.
func (d Document) Encode(w io.Writer) error {
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	if _, err := w.Write([]byte(xml.Header)); err != nil {
		return err
	}
	return enc.Encode(d)
}

func parse(r io.Reader) (Document, error) {
	dec := xml.NewDecoder(r)
	dec.Strict = true
	var doc Document
	if err := dec.Decode(&doc); err != nil {
		return Document{}, fmt.Errorf("parse poml: %w", err)
	}
	return doc, nil
}

// WalkInputs applies fn to each input block.
func (d *Document) WalkInputs(fn func(*Input)) {
	for i := range d.Inputs {
		fn(&d.Inputs[i])
	}
}

// RoleText returns the role text with surrounding whitespace trimmed.
func (d Document) RoleText() string {
	return strings.TrimSpace(d.Role)
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
