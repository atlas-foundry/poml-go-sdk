package poml

import (
	"bytes"
	"path/filepath"
	"testing"
)

func TestConvertMarkdownToPOMLGolden(t *testing.T) {
	md := `# role

task one

## task two

Paragraph text.`

	doc, err := convertMarkdownToPOML(md)
	if err != nil {
		t.Fatalf("convert markdown: %v", err)
	}
	assertDocMatchesGolden(t, doc, filepath.Join("testdata", "golden", "markdown_to_poml.poml"))
}

func TestConvertOrgToPOMLGolden(t *testing.T) {
	org := `* role
** task one
** task two`

	doc, err := convertOrgToPOML(org)
	if err != nil {
		t.Fatalf("convert org: %v", err)
	}
	assertDocMatchesGolden(t, doc, filepath.Join("testdata", "golden", "org_to_poml.poml"))
}

func assertDocMatchesGolden(t *testing.T, doc Document, goldenPath string) {
	t.Helper()
	var gotBuf bytes.Buffer
	if err := doc.EncodeWithOptions(&gotBuf, EncodeOptions{IncludeHeader: false, PreserveOrder: false, PreserveWS: true, Compact: true}); err != nil {
		t.Fatalf("encode: %v", err)
	}
	goldenDoc, err := ParseFile(goldenPath)
	if err != nil {
		t.Fatalf("parse golden %s: %v", goldenPath, err)
	}
	var wantBuf bytes.Buffer
	if err := goldenDoc.EncodeWithOptions(&wantBuf, EncodeOptions{IncludeHeader: false, PreserveOrder: false, PreserveWS: true, Compact: true}); err != nil {
		t.Fatalf("encode golden: %v", err)
	}
	if gotBuf.String() != wantBuf.String() {
		t.Fatalf("golden mismatch for %s:\nwant:\n%s\n\ngot:\n%s", goldenPath, wantBuf.String(), gotBuf.String())
	}
}
