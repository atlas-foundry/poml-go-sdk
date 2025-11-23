package poml

import (
	"bytes"
	"math/rand"
	"testing"
	"time"
)

// FuzzParseRoundTrip guards against panics/regressions in ParseString/EncodeWithOptions.
func FuzzParseRoundTrip(f *testing.F) {
	seeds := []string{
		"<poml></poml>",
		"<poml><meta><id>a</id><version>1</version><owner>o</owner></meta><role>r</role><task>t</task></poml>",
		"<poml><task>only task</task></poml>",
		"<poml><role><![CDATA[<xml>weird</xml>]]></role></poml>",
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, input string) {
		doc, err := ParseString(input)
		if err != nil {
			return // invalid inputs are fine; ensure no panic
		}
		var buf bytes.Buffer
		if err := doc.EncodeWithOptions(&buf, EncodeOptions{IncludeHeader: false, PreserveOrder: true, PreserveWS: true}); err != nil {
			t.Fatalf("encode: %v", err)
		}
		if _, err := ParseString(buf.String()); err != nil {
			t.Fatalf("round-trip parse failed: %v", err)
		}
	})
}

func TestParseRoundTripRandomSeeds(t *testing.T) {
	rng := rand.New(rand.NewSource(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).UnixNano()))
	for i := 0; i < 50; i++ {
		// Generate small random ASCII strings to probe parser; allow invalids.
		n := rng.Intn(64)
		b := make([]byte, n)
		for j := range b {
			b[j] = byte(rng.Intn(94) + 32) // printable ASCII
		}
		s := string(b)
		doc, err := ParseString(s)
		if err != nil {
			continue
		}
		var buf bytes.Buffer
		if err := doc.EncodeWithOptions(&buf, EncodeOptions{IncludeHeader: false, PreserveOrder: true, PreserveWS: true}); err != nil {
			t.Fatalf("encode seed %d: %v", i, err)
		}
		if _, err := ParseString(buf.String()); err != nil {
			t.Fatalf("round-trip seed %d failed: %v", i, err)
		}
	}
}
