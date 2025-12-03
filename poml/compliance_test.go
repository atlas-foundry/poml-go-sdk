package poml

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Compliance smoke: ensure positive fixtures validate and convert under strict core/extended rules.
func TestComplianceMatrix(t *testing.T) {
	examples, err := filepath.Glob(filepath.Join("testdata", "examples", "*.poml"))
	if err != nil {
		t.Fatalf("glob examples: %v", err)
	}
	skipWords := []string{"invalid", "missing", "oversize", "off_", "off.", "bad", "fail", "unknown_attr"}
	var coreTotal, corePass, extTotal, extPass int

	for _, path := range examples {
		base := filepath.Base(path)
		lower := strings.ToLower(base)
		skip := false
		for _, word := range skipWords {
			if strings.Contains(lower, word) {
				skip = true
				break
			}
		}
		if base == "validation_config.poml" {
			skip = true
		}
		if skip {
			continue
		}

		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		lowerBody := strings.ToLower(string(body))
		isExtended := strings.Contains(lowerBody, `mode="extended"`) || strings.Contains(lowerBody, `extended="true"`)
		if strings.Contains(lowerBody, "<op") || strings.Contains(lowerBody, "<figure") || strings.Contains(lowerBody, "<object") || strings.Contains(lowerBody, "<extended-op") || strings.Contains(lowerBody, "<extended-figure") || strings.Contains(lowerBody, "<data") || strings.Contains(lowerBody, "<text") {
			isExtended = true
		}

		doc, err := ParseFile(path)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}

		validateOpts := ValidateOptions{Extended: ExtendedOff}
		if isExtended {
			validateOpts.Extended = ExtendedStrict
		}
		if strings.TrimSpace(doc.Meta.ID) == "" || strings.TrimSpace(doc.Meta.Version) == "" || strings.TrimSpace(doc.Meta.Owner) == "" || strings.TrimSpace(doc.Role.Body) == "" || len(doc.Tasks) == 0 {
			continue
		}

		if err := doc.ValidateWithOptions(validateOpts); err != nil {
			t.Fatalf("validate %s (%v): %v", base, validateOpts.Extended, err)
		}
		if _, err := Convert(doc, FormatMessageDict, ConvertOptions{Extended: validateOpts.Extended}); err != nil {
			t.Fatalf("convert %s (%v): %v", base, validateOpts.Extended, err)
		}

		if isExtended {
			extTotal++
			extPass++
		} else {
			coreTotal++
			corePass++
		}
	}

	t.Logf("Compliance matrix: core %d/%d, extended %d/%d (strict defaults)", corePass, coreTotal, extPass, extTotal)
}
