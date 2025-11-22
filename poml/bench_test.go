package poml

import (
	"bytes"
	"testing"
)

const benchDoc = `<poml>
  <meta><id>bench.demo</id><version>1.0</version><owner>tester</owner></meta>
  <role>Bench role</role>
  <task>Do a thing</task>
  <human-msg>Hello</human-msg>
  <assistant-msg>Hi</assistant-msg>
  <tool-definition name="calc">{"type":"object"}</tool-definition>
  <tool-request id="call_1" name="calc" parameters="{{ { x: 1 } }}"/>
  <tool-response id="call_1" name="calc">2</tool-response>
  <output-schema>{"type":"object"}</output-schema>
  <runtime temperature="0.3"/>
</poml>`

func BenchmarkParseString(b *testing.B) {
	for i := 0; i < b.N; i++ {
		if _, err := ParseString(benchDoc); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkEncodeDocument(b *testing.B) {
	doc, err := ParseString(benchDoc)
	if err != nil {
		b.Fatalf("parse: %v", err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		if err := doc.Encode(&buf); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkConvertOpenAIChat(b *testing.B) {
	doc, err := ParseString(benchDoc)
	if err != nil {
		b.Fatalf("parse: %v", err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := Convert(doc, FormatOpenAIChat, ConvertOptions{}); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDiagramToScene(b *testing.B) {
	doc, err := ParseString(diagramSample)
	if err != nil {
		b.Fatalf("parse diagram: %v", err)
	}
	if len(doc.Diagrams) == 0 {
		b.Fatalf("diagram missing")
	}
	diag := doc.Diagrams[0]
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := DiagramToScene(diag); err != nil {
			b.Fatal(err)
		}
	}
}
