//go:build go1.19

package poml

import (
	"errors"
	"os"
	"strings"
	"testing"
)

// FuzzParseEncodeExtended exercises parse→encode→parse across Extended modes using
// a mix of structured and fuzzed inputs. It is short by default; extend -fuzztime
// locally for deeper coverage.
func FuzzParseEncodeExtended(f *testing.F) {
	seeds := []string{
		`<poml><meta><id>x</id><version>1</version><owner>o</owner></meta><role>r</role><task>t</task></poml>`,
		`<poml><meta><id>x</id><version>1</version><owner>o</owner></meta><role>r</role><task>t</task><extended-op name="demo" foo="bar">hello</extended-op></poml>`,
		`<poml><meta><id>x</id><version>1</version><owner>o</owner></meta><role><![CDATA[with cdata]]></role><task>t</task><unknown attr="1"/><image src="data:image/png;base64,AA=="/> </poml>`,
	}
	corpusFiles := []string{
		"poml/testdata/fuzz/extended/corpus/001_unknown_tag.poml",
		"poml/testdata/fuzz/extended/corpus/002_nested_unknowns.poml",
		"poml/testdata/fuzz/extended/corpus/003_media_extended.poml",
	}
	for _, path := range corpusFiles {
		if body, err := os.ReadFile(path); err == nil {
			f.Add(string(body))
		}
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, input string) {
		// Bound input size to avoid pathological payloads.
		if len(input) > 64*1024 {
			t.Skip()
		}
		// Normalize to keep XML decoder happier during fuzzing.
		input = strings.TrimSpace(input)
		if input == "" {
			t.Skip()
		}
		modes := []ExtendedMode{ExtendedOff, ExtendedLenient, ExtendedStrict}
		for _, mode := range modes {
			opts := ParseOptions{PreserveWhitespace: true, Validate: mode != ExtendedOff, Extended: mode}
			doc, err := ParseReaderWithOptions(strings.NewReader(input), opts)
			if err != nil {
				// In ExtendedOff, unknown/extended tags may fail validation; skip.
				if mode == ExtendedOff {
					continue
				}
				var pe *POMLError
				if errors.As(err, &pe) && pe.Type == ErrValidate {
					t.Skip("validation error in strict extended mode")
				}
				t.Fatalf("parse (%v): %v", mode, err)
			}
			var buf strings.Builder
			if err := doc.EncodeWithOptions(&buf, EncodeOptions{IncludeHeader: false, PreserveOrder: true, PreserveWS: true}); err != nil {
				t.Fatalf("encode (%v): %v", mode, err)
			}
			roundtrip := buf.String()
			if _, err := ParseReaderWithOptions(strings.NewReader(roundtrip), opts); err != nil {
				var pe *POMLError
				if mode != ExtendedOff && errors.As(err, &pe) && pe.Type == ErrValidate {
					t.Skip("validation error in strict extended mode after round-trip")
				}
				t.Fatalf("roundtrip parse (%v): %v\ninput: %s\nrt: %s", mode, err, input, roundtrip)
			}
		}
	})
}

// Table-driven sanity for known extended-like constructs to ensure coverage in short runs.
func TestExtendedRoundTripTable(t *testing.T) {
	tests := []struct {
		name string
		src  string
		mode ExtendedMode
	}{
		{"unknown-allowed-lenient", `<poml><meta><id>x</id><version>1</version><owner>o</owner></meta><role>r</role><task>t</task><unknown foo="bar">hi</unknown></poml>`, ExtendedLenient},
		{"unknown-allowed-strict", `<poml><meta><id>x</id><version>1</version><owner>o</owner></meta><role>r</role><task>t</task><unknown foo="bar"/><image src="data:image/png;base64,AA=="/></poml>`, ExtendedStrict},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := ParseOptions{PreserveWhitespace: true, Validate: true, Extended: tt.mode}
			doc, err := ParseReaderWithOptions(strings.NewReader(tt.src), opts)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			var buf strings.Builder
			if err := doc.EncodeWithOptions(&buf, EncodeOptions{IncludeHeader: false, PreserveOrder: true, PreserveWS: true}); err != nil {
				t.Fatalf("encode: %v", err)
			}
			if _, err := ParseReaderWithOptions(strings.NewReader(buf.String()), opts); err != nil {
				t.Fatalf("roundtrip parse: %v", err)
			}
		})
	}
}
