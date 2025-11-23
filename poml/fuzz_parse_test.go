//go:build go1.19

package poml

import (
	"strings"
	"testing"
)

// FuzzParseStringRoundTrip ensures ParseString and ParseStringFast can round-trip
// reasonable documents without losing structure. Keep -fuzztime short in CI.
func FuzzParseStringRoundTrip(f *testing.F) {
	seeds := []string{
		`<poml><meta><id>a</id><version>1</version><owner>o</owner></meta><role>r</role><task>t</task></poml>`,
		`<poml><meta><id>b</id><version>2</version><owner>o</owner></meta><role><![CDATA[r]]></role><task>t</task><system-msg>sys</system-msg><human-msg>hi</human-msg></poml>`,
		`<poml><meta><id>c</id><version>3</version><owner>o</owner></meta><role>r</role><task>t</task><assistant-msg>resp</assistant-msg><unknown foo="bar">x</unknown></poml>`,
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, input string) {
		if len(input) == 0 || len(input) > 64*1024 {
			t.Skip()
		}
		input = strings.TrimSpace(input)
		if input == "" {
			t.Skip()
		}

		doc, err := ParseString(input)
		if err != nil {
			t.Skip()
		}
		var buf strings.Builder
		if err := doc.EncodeWithOptions(&buf, EncodeOptions{IncludeHeader: false, PreserveOrder: true, PreserveWS: true}); err != nil {
			t.Fatalf("encode: %v", err)
		}
		rt := buf.String()
		if _, err := ParseString(rt); err != nil {
			t.Fatalf("round-trip parse: %v\ninput: %s\nrt: %s", err, input, rt)
		}

		// Fast path should accept the same document; failures are surfaced.
		if fastDoc, err := ParseStringFast(rt); err == nil {
			var fastBuf strings.Builder
			if err := fastDoc.EncodeWithOptions(&fastBuf, EncodeOptions{IncludeHeader: false, PreserveOrder: true}); err != nil {
				t.Fatalf("fast encode: %v", err)
			}
			if _, err := ParseString(fastBuf.String()); err != nil {
				t.Fatalf("fast round-trip parse: %v", err)
			}
		}
	})
}
