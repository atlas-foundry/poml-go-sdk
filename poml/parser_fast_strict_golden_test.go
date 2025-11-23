package poml

import (
	"strings"
	"testing"
)

// Ensure strict parsing rejects the same inputs that fast parsing might allow,
// and that valid docs stay stable across modes.
func TestParserFastVsStrictDifferentials(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		invalid bool // invalid in strict mode
	}{
		{
			name:    "missing-meta",
			input:   `<poml><role>r</role><task>t</task></poml>`,
			invalid: true,
		},
		{
			name:    "missing-task",
			input:   `<poml><meta><id>x</id><version>1</version><owner>o</owner></meta><role>r</role></poml>`,
			invalid: true,
		},
		{
			name:    "valid-minimal",
			input:   `<poml><meta><id>x</id><version>1</version><owner>o</owner></meta><role>r</role><task>t</task></poml>`,
			invalid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Fast parse should never panic on these inputs.
			if _, err := ParseStringFast(tt.input); err != nil {
				t.Fatalf("fast parse failed: %v", err)
			}
			_, err := ParseStringStrict(tt.input)
			if tt.invalid {
				if err == nil {
					t.Fatalf("expected strict parse to fail for %s", tt.name)
				}
				return
			}
			if err != nil {
				t.Fatalf("strict parse failed for valid input: %v", err)
			}
			// Round-trip stable encoding for valid docs.
			doc, _ := ParseStringStrict(tt.input)
			var buf strings.Builder
			if err := doc.EncodeWithOptions(&buf, EncodeOptions{IncludeHeader: false, PreserveOrder: true, PreserveWS: true}); err != nil {
				t.Fatalf("encode: %v", err)
			}
			if _, err := ParseStringStrict(buf.String()); err != nil {
				t.Fatalf("round-trip strict parse failed: %v", err)
			}
		})
	}
}
