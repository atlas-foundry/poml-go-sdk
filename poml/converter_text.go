package poml

import (
	"bytes"
	"strings"

	goorg "github.com/niklasfasching/go-org/org"
	"github.com/yuin/goldmark"
	mdast "github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	mdtext "github.com/yuin/goldmark/text"
)

// TextFormat enumerates text-based converter targets.
type TextFormat string

const (
	FormatMarkdown TextFormat = "markdown"
	FormatOrg      TextFormat = "org"
)

// ConvertTextToPOML parses a text document (markdown/org) to a minimal POML Document.
// Headings are mapped to tasks (after the first, which seeds role).
func ConvertTextToPOML(body string, format TextFormat) (Document, error) {
	switch format {
	case FormatMarkdown:
		return convertMarkdownToPOML(body)
	case FormatOrg:
		return convertOrgToPOML(body)
	default:
		return Document{}, ErrNotImplemented
	}
}

// ConvertPOMLToText renders a POML Document to text (markdown/org).
func ConvertPOMLToText(doc Document, format TextFormat) (string, error) {
	switch format {
	case FormatMarkdown:
		return renderMarkdown(doc), nil
	case FormatOrg:
		return renderOrg(doc), nil
	default:
		return "", ErrNotImplemented
	}
}

func convertMarkdownToPOML(body string) (Document, error) {
	md := goldmark.New(
		goldmark.WithExtensions(extension.Table, extension.Strikethrough, extension.Linkify),
		goldmark.WithParserOptions(parser.WithAutoHeadingID()),
	)
	src := []byte(body)
	reader := mdtext.NewReader(src)
	root := md.Parser().Parse(reader)
	doc := Document{Meta: Meta{ID: "converted.markdown", Version: "0.0.0", Owner: "converter"}}

	var tasks []string
	var role string
	mdast.Walk(root, func(n mdast.Node, entering bool) (mdast.WalkStatus, error) {
		switch node := n.(type) {
		case *mdast.Heading:
			if entering {
				text := extractText(node, src)
				if text != "" {
					if role == "" {
						role = text
					} else {
						tasks = append(tasks, text)
					}
				}
			}
		case *mdast.Paragraph:
			if entering {
				text := extractText(node, src)
				if text != "" {
					tasks = append(tasks, text)
				}
			}
		}
		return mdast.WalkContinue, nil
	})

	if role != "" {
		doc.Role = Block{Body: role}
	} else {
		doc.Role = Block{Body: "Converted markdown"}
	}
	for _, t := range tasks {
		doc.AddTask(t)
	}
	return doc, nil
}

func convertOrgToPOML(body string) (Document, error) {
	o := goorg.New().Parse(strings.NewReader(body), "")
	out, err := o.Write(goorg.NewOrgWriter())
	if err != nil {
		return Document{}, err
	}
	// Simple heuristic: first line as role, rest as tasks paragraphs.
	lines := strings.Split(strings.TrimSpace(out), "\n")
	doc := Document{Meta: Meta{ID: "converted.org", Version: "0.0.0", Owner: "converter"}}
	if len(lines) > 0 {
		doc.Role = Block{Body: strings.TrimSpace(lines[0])}
		for _, line := range lines[1:] {
			if strings.TrimSpace(line) != "" {
				doc.AddTask(strings.TrimSpace(line))
			}
		}
	}
	return doc, nil
}

func renderMarkdown(doc Document) string {
	var b strings.Builder
	if r := strings.TrimSpace(doc.Role.Body); r != "" {
		b.WriteString("# ")
		b.WriteString(r)
		b.WriteString("\n\n")
	}
	for _, t := range doc.Tasks {
		if tb := strings.TrimSpace(t.Body); tb != "" {
			b.WriteString("## Task\n\n")
			b.WriteString(tb)
			b.WriteString("\n\n")
		}
	}
	for _, in := range doc.Inputs {
		b.WriteString("- Input ")
		b.WriteString(in.Name)
		if in.Required {
			b.WriteString(" (required)")
		}
		if b.Len() > 0 {
			b.WriteString(": ")
		}
		b.WriteString(strings.TrimSpace(in.Body))
		b.WriteString("\n")
	}
	return strings.TrimSpace(b.String())
}

func renderOrg(doc Document) string {
	var b strings.Builder
	if r := strings.TrimSpace(doc.Role.Body); r != "" {
		b.WriteString("* ")
		b.WriteString(r)
		b.WriteString("\n\n")
	}
	for _, t := range doc.Tasks {
		if tb := strings.TrimSpace(t.Body); tb != "" {
			b.WriteString("** Task\n\n")
			b.WriteString(tb)
			b.WriteString("\n\n")
		}
	}
	for _, in := range doc.Inputs {
		b.WriteString("- Input ")
		b.WriteString(in.Name)
		if in.Required {
			b.WriteString(" [required]")
		}
		if b.Len() > 0 {
			b.WriteString(": ")
		}
		b.WriteString(strings.TrimSpace(in.Body))
		b.WriteString("\n")
	}
	return strings.TrimSpace(b.String())
}

func extractText(n mdast.Node, src []byte) string {
	var b bytes.Buffer
	mdast.Walk(n, func(nn mdast.Node, entering bool) (mdast.WalkStatus, error) {
		if !entering {
			return mdast.WalkContinue, nil
		}
		if tn, ok := nn.(*mdast.Text); ok {
			b.Write(tn.Segment.Value(src))
		}
		return mdast.WalkContinue, nil
	})
	return strings.TrimSpace(b.String())
}
