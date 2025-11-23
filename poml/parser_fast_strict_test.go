package poml

import (
	"bytes"
	"strings"
	"testing"
)

func TestParserFastVsDefaultWhitespace(t *testing.T) {
	src := "<poml><task>one</task><!-- gap --><task>two</task></poml>"

	defaultDoc, err := ParseReaderWithOptions(strings.NewReader(src), ParseOptions{PreserveWhitespace: true})
	if err != nil {
		t.Fatalf("default parse: %v", err)
	}
	fastDoc, err := ParseReaderWithOptions(strings.NewReader(src), ParseOptions{PreserveWhitespace: false})
	if err != nil {
		t.Fatalf("fast parse: %v", err)
	}
	for _, el := range fastDoc.Elements {
		if el.Leading != "" || el.Trailing != "" {
			t.Fatalf("expected whitespace stripped in fast parser: %+v", el)
		}
	}
	if len(defaultDoc.Elements) == 0 {
		t.Fatalf("expected elements in default parse")
	}
	// The comment should be captured as trailing on the first task or leading on the second.
	hasComment := strings.Contains(defaultDoc.Elements[0].Trailing, "gap")
	if len(defaultDoc.Elements) > 1 {
		hasComment = hasComment || strings.Contains(defaultDoc.Elements[1].Leading, "gap")
	}
	if !hasComment {
		t.Fatalf("expected comment captured, got elements: %+v", defaultDoc.Elements)
	}
	var buf bytes.Buffer
	if err := fastDoc.EncodeWithOptions(&buf, EncodeOptions{IncludeHeader: false, PreserveOrder: true, PreserveWS: true, Compact: true}); err != nil {
		t.Fatalf("encode fast doc: %v", err)
	}
	if strings.Contains(buf.String(), "gap") {
		t.Fatalf("fast doc should not preserve whitespace/comments, got: %s", buf.String())
	}
}

func TestParserStrictValidation(t *testing.T) {
	src := "<poml><task>only task</task></poml>"
	if _, err := ParseStringStrict(src); err == nil {
		t.Fatalf("expected strict validation failure")
	}
	valid := `<poml><meta><id>x</id><version>1</version><owner>o</owner></meta><role>r</role><task>t</task></poml>`
	if _, err := ParseStringStrict(valid); err != nil {
		t.Fatalf("strict parse should succeed: %v", err)
	}
}
