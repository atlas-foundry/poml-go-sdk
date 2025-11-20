package poml

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

const sample = `<poml>
  <meta>
    <id>sample.demo</id>
    <version>1.0.0</version>
    <owner>tester</owner>
  </meta>
  <role>Demo role text</role>
  <task>
    First task body
  </task>
  <task><![CDATA[
    Second task with CDATA and <tags/>
  ]]></task>
  <input name="status" required="true">
    Provide status details
  </input>
  <input name="note" required="false"><![CDATA[Optional note]]></input>
  <document src="file://docs/example.org" />
  <style>
    <output format="markdown"><![CDATA[# Title]]></output>
  </style>
</poml>`

func TestParseSample(t *testing.T) {
	doc, err := ParseString(sample)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if doc.Meta.ID != "sample.demo" || doc.Meta.Owner != "tester" {
		t.Fatalf("meta mismatch: %+v", doc.Meta)
	}
	if doc.RoleText() != "Demo role text" {
		t.Fatalf("role text mismatch: %q", doc.RoleText())
	}
	tasks := doc.TaskBodies()
	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(tasks))
	}
	if !strings.Contains(tasks[1], "CDATA") {
		t.Fatalf("second task not preserved: %q", tasks[1])
	}
	if len(doc.Inputs) != 2 || doc.Inputs[0].Name != "status" || !doc.Inputs[0].Required {
		t.Fatalf("inputs mismatch: %+v", doc.Inputs)
	}
	if len(doc.Documents) != 1 || doc.Documents[0].Src == "" {
		t.Fatalf("doc refs missing")
	}
	if len(doc.Styles) != 1 || len(doc.Styles[0].Outputs) != 1 || doc.Styles[0].Outputs[0].Format != "markdown" {
		t.Fatalf("styles mismatch: %+v", doc.Styles)
	}
}

func TestRoundTripEncode(t *testing.T) {
	doc, err := ParseString(sample)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	var buf bytes.Buffer
	if err := doc.Encode(&buf); err != nil {
		t.Fatalf("encode: %v", err)
	}
	again, err := ParseString(buf.String())
	if err != nil {
		t.Fatalf("parse roundtrip: %v", err)
	}
	if again.Meta.ID != doc.Meta.ID || again.RoleText() != doc.RoleText() || len(again.Tasks) != len(doc.Tasks) {
		t.Fatalf("roundtrip mismatch: original %+v, again %+v", doc, again)
	}
}

func TestMalformedReportsError(t *testing.T) {
	// missing closing tag, malformed attribute
	bad := `<poml><meta><id>bad</id></meta><input name="x" required nope></input></poml>`
	if _, err := ParseString(bad); err == nil {
		t.Fatalf("expected parse error for malformed input")
	} else if !strings.Contains(err.Error(), "line") {
		t.Fatalf("error should include location information, got: %v", err)
	}
}

func TestLargeDocumentParses(t *testing.T) {
	var b strings.Builder
	b.WriteString(`<poml><meta><id>large</id></meta>`)
	for i := 0; i < 1000; i++ {
		fmt.Fprintf(&b, "<task>Task %d body</task>", i)
	}
	b.WriteString(`</poml>`)
	doc, err := ParseString(b.String())
	if err != nil {
		t.Fatalf("parse large: %v", err)
	}
	if len(doc.Tasks) != 1000 {
		t.Fatalf("expected 1000 tasks, got %d", len(doc.Tasks))
	}
}
