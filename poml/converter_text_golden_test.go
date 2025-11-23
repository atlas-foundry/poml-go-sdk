package poml

import "testing"

func TestConvertMarkdownToPOMLGolden(t *testing.T) {
	md := `# role

task one

## task two

Paragraph text.`

	doc, err := convertMarkdownToPOML(md)
	if err != nil {
		t.Fatalf("convert markdown: %v", err)
	}
	if got := doc.Role.Body; got != "role" {
		t.Fatalf("role mismatch: %q", got)
	}
	if len(doc.Tasks) != 3 {
		t.Fatalf("expected 3 tasks (two headings + paragraph), got %d", len(doc.Tasks))
	}
	if doc.Tasks[0].Body != "task one" || doc.Tasks[1].Body != "task two" {
		t.Fatalf("tasks mismatch: %#v", doc.Tasks)
	}
}

func TestConvertOrgToPOMLGolden(t *testing.T) {
	org := `* role
** task one
** task two`

	doc, err := convertOrgToPOML(org)
	if err != nil {
		t.Fatalf("convert org: %v", err)
	}
	if got := doc.Role.Body; got != "* role" { // go-org writer preserves heading markers
		t.Fatalf("role mismatch: %q", got)
	}
	if len(doc.Tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(doc.Tasks))
	}
	if doc.Tasks[0].Body != "** task one" || doc.Tasks[1].Body != "** task two" {
		t.Fatalf("tasks mismatch: %#v", doc.Tasks)
	}
}
