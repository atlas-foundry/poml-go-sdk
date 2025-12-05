package poml

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

// TestParityFixturesParse verifies all parity fixtures parse without error.
func TestParityFixturesParse(t *testing.T) {
	fixtures, err := filepath.Glob(filepath.Join("testdata", "parity", "*.poml"))
	if err != nil {
		t.Fatalf("glob: %v", err)
	}

	if len(fixtures) == 0 {
		t.Fatal("no parity fixtures found")
	}

	for _, path := range fixtures {
		name := filepath.Base(path)
		t.Run(name, func(t *testing.T) {
			doc, err := ParseFile(path)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}

			// Basic sanity checks
			if doc.Meta.Version == "" {
				t.Error("missing version")
			}
			if doc.Role.Body == "" {
				t.Error("missing role")
			}
		})
	}
}

// TestParityFixturesConvert verifies all parity fixtures convert to all formats.
func TestParityFixturesConvert(t *testing.T) {
	fixtures, err := filepath.Glob(filepath.Join("testdata", "parity", "*.poml"))
	if err != nil {
		t.Fatalf("glob: %v", err)
	}

	formats := []Format{
		FormatMessageDict,
		FormatDict,
		FormatOpenAIChat,
		FormatLangChain,
		FormatPydantic,
	}

	for _, path := range fixtures {
		name := filepath.Base(path)
		t.Run(name, func(t *testing.T) {
			doc, err := ParseFile(path)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}

			for _, format := range formats {
				t.Run(string(format), func(t *testing.T) {
					_, err := Convert(doc, format, ConvertOptions{})
					if err != nil {
						t.Errorf("convert to %s: %v", format, err)
					}
				})
			}
		})
	}
}

// TestRichTextFixtures verifies rich text elements convert correctly.
func TestRichTextFixtures(t *testing.T) {
	cases := []struct {
		file     string
		expected []string
	}{
		{
			file:     "richtext_blocks.poml",
			expected: []string{"Main Heading", "paragraph", "Subheading", "Named Section"},
		},
		{
			file:     "richtext_lists.poml",
			expected: []string{"First", "Second", "Third"},
		},
		{
			file:     "richtext_code.poml",
			expected: []string{"```go", "```python", "fmt.Println"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.file, func(t *testing.T) {
			path := filepath.Join("testdata", "parity", tc.file)
			doc, err := ParseFile(path)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}

			result, err := Convert(doc, FormatDict, ConvertOptions{})
			if err != nil {
				t.Fatalf("convert: %v", err)
			}

			resultStr := formatResult(result)
			for _, exp := range tc.expected {
				if !strings.Contains(resultStr, exp) {
					t.Errorf("expected %q in output, got: %s", exp, resultStr)
				}
			}
		})
	}
}

// TestTemplateFixtures verifies template features work.
func TestTemplateFixtures(t *testing.T) {
	t.Run("template_let", func(t *testing.T) {
		path := filepath.Join("testdata", "parity", "template_let.poml")
		doc, err := ParseFile(path)
		if err != nil {
			t.Fatalf("parse: %v", err)
		}

		// Verify let bindings were parsed
		if len(doc.LetBindings) != 2 {
			t.Errorf("expected 2 let bindings, got %d", len(doc.LetBindings))
		}

		result, err := Convert(doc, FormatDict, ConvertOptions{
			ExpandTemplates: true,
		})
		if err != nil {
			t.Fatalf("convert: %v", err)
		}

		resultStr := formatResult(result)
		if !strings.Contains(resultStr, "Hello") || !strings.Contains(resultStr, "World") {
			t.Errorf("template not interpolated: %s", resultStr)
		}
	})

	t.Run("template_variables", func(t *testing.T) {
		path := filepath.Join("testdata", "parity", "template_variables.poml")
		doc, err := ParseFile(path)
		if err != nil {
			t.Fatalf("parse: %v", err)
		}

		result, err := Convert(doc, FormatDict, ConvertOptions{
			TemplateVars: map[string]any{
				"name":    "Alice",
				"count":   5,
				"balance": "100.00",
			},
		})
		if err != nil {
			t.Fatalf("convert: %v", err)
		}

		resultStr := formatResult(result)
		if !strings.Contains(resultStr, "Alice") {
			t.Errorf("name not interpolated: %s", resultStr)
		}
		if !strings.Contains(resultStr, "5") {
			t.Errorf("count not interpolated: %s", resultStr)
		}
	})
}

// TestTemplateCrossFormat verifies templates work across all output formats.
func TestTemplateCrossFormat(t *testing.T) {
	input := `<poml>
<meta><version>0.0.8</version></meta>
<role>Test cross-format templates</role>
<task>Test</task>
<hint>Hello, {{ name }}!</hint>
</poml>`

	doc, err := ParseString(input)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	formats := []Format{
		FormatMessageDict,
		FormatDict,
		FormatOpenAIChat,
		FormatLangChain,
		FormatPydantic,
	}

	opts := ConvertOptions{
		ExpandTemplates: true,
		TemplateVars: map[string]any{
			"name": "CrossFormat",
		},
	}

	for _, format := range formats {
		t.Run(string(format), func(t *testing.T) {
			result, err := Convert(doc, format, opts)
			if err != nil {
				t.Fatalf("convert: %v", err)
			}

			resultStr := formatResult(result)
			if !strings.Contains(resultStr, "CrossFormat") {
				t.Errorf("template not interpolated in %s format: %s", format, resultStr)
			}
			// Should not contain the raw template syntax
			if strings.Contains(resultStr, "{{") {
				t.Errorf("raw template syntax found in %s format: %s", format, resultStr)
			}
		})
	}
}

// TestLetBindingsCrossFormat verifies let bindings work across formats.
func TestLetBindingsCrossFormat(t *testing.T) {
	input := `<poml>
<meta><version>0.0.8</version></meta>
<role>Test let bindings</role>
<let name="greeting" value="Greetings"/>
<let name="target" value="Universe"/>
<task>Test</task>
<hint>{{ greeting }}, {{ target }}!</hint>
</poml>`

	doc, err := ParseString(input)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	formats := []Format{
		FormatMessageDict,
		FormatDict,
		FormatOpenAIChat,
		FormatLangChain,
		FormatPydantic,
	}

	for _, format := range formats {
		t.Run(string(format), func(t *testing.T) {
			result, err := Convert(doc, format, ConvertOptions{
				ExpandTemplates: true,
			})
			if err != nil {
				t.Fatalf("convert: %v", err)
			}

			resultStr := formatResult(result)
			if !strings.Contains(resultStr, "Greetings") {
				t.Errorf("let binding 'greeting' not interpolated in %s: %s", format, resultStr)
			}
			if !strings.Contains(resultStr, "Universe") {
				t.Errorf("let binding 'target' not interpolated in %s: %s", format, resultStr)
			}
		})
	}
}

// TestVersionConstraintFixtures verifies version constraint handling.
func TestVersionConstraintFixtures(t *testing.T) {
	t.Run("version_valid", func(t *testing.T) {
		path := filepath.Join("testdata", "parity", "version_valid.poml")
		doc, err := ParseFile(path)
		if err != nil {
			t.Fatalf("parse: %v", err)
		}

		// Should have version constraints
		if doc.Meta.MinVersion == "" {
			t.Error("missing minVersion")
		}
		if doc.Meta.MaxVersion == "" {
			t.Error("missing maxVersion")
		}

		// Should convert without error (version 0.0.8 is within 0.0.1-1.0.0)
		_, err = Convert(doc, FormatMessageDict, ConvertOptions{})
		if err != nil {
			t.Errorf("convert: %v", err)
		}
	})

	t.Run("version_invalid_min", func(t *testing.T) {
		path := filepath.Join("testdata", "parity", "version_invalid_min.poml")
		doc, err := ParseFile(path)
		if err != nil {
			t.Fatalf("parse: %v", err)
		}

		// minVersion should be set to something higher than current spec version
		if doc.Meta.MinVersion == "" {
			t.Error("missing minVersion in invalid fixture")
		}
	})

	t.Run("version_invalid_max", func(t *testing.T) {
		path := filepath.Join("testdata", "parity", "version_invalid_max.poml")
		doc, err := ParseFile(path)
		if err != nil {
			t.Fatalf("parse: %v", err)
		}

		// maxVersion should be set to something lower than current spec version
		if doc.Meta.MaxVersion == "" {
			t.Error("missing maxVersion in invalid fixture")
		}
	})
}

// TestPydanticMediaCollection verifies that Pydantic format collects media.
func TestPydanticMediaCollection(t *testing.T) {
	input := `<poml>
<meta><version>0.0.8</version></meta>
<role>Test Pydantic media</role>
<task>Test media collection</task>
<img src="data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==" alt="test"/>
</poml>`

	doc, err := ParseString(input)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	result, err := Convert(doc, FormatPydantic, ConvertOptions{})
	if err != nil {
		t.Fatalf("convert: %v", err)
	}

	resultStr := formatResult(result)

	// Pydantic format should include media collection
	if !strings.Contains(resultStr, "media") {
		t.Logf("Pydantic output: %s", resultStr)
	}
	// Should contain the base64 image data
	if !strings.Contains(resultStr, "base64") {
		t.Errorf("Pydantic output should contain base64 image data")
	}
}

// TestStylesheetComponentInteraction verifies stylesheet and component filtering work together.
func TestStylesheetComponentInteraction(t *testing.T) {
	t.Run("filter_with_stylesheet", func(t *testing.T) {
		input := `<poml>
<meta><version>0.0.8</version></meta>
<role>Test interaction</role>
<task>Test stylesheet + filter</task>
<style>
	<output format="json">{"hint": {"syntax": "markdown"}}</output>
</style>
<hint class="important">Keep this hint</hint>
<img src="data:image/png;base64,iVBORw0KGgo=" alt="filter me"/>
</poml>`

		doc, err := ParseString(input)
		if err != nil {
			t.Fatalf("parse: %v", err)
		}

		// Convert with both stylesheet and component filtering
		result, err := Convert(doc, FormatDict, ConvertOptions{
			ApplyStylesheet: true,
			Components:      "-image", // Filter out images
		})
		if err != nil {
			t.Fatalf("convert: %v", err)
		}

		resultStr := formatResult(result)

		// Hint should be present
		if !strings.Contains(resultStr, "Keep this hint") {
			t.Errorf("hint should be present: %s", resultStr)
		}
		// Image should be filtered out
		if strings.Contains(resultStr, "iVBORw0KGgo") {
			t.Errorf("image should be filtered out: %s", resultStr)
		}
	})

	t.Run("multiple_filters", func(t *testing.T) {
		input := `<poml>
<meta><version>0.0.8</version></meta>
<role>Test multiple filters</role>
<task>Test</task>
<hint>Visible hint</hint>
<example>Visible example</example>
<img src="data:image/png;base64,hidden" alt="hidden"/>
</poml>`

		doc, err := ParseString(input)
		if err != nil {
			t.Fatalf("parse: %v", err)
		}

		// Include hints and examples, exclude images
		result, err := Convert(doc, FormatDict, ConvertOptions{
			Components: "+hint,+example,-image",
		})
		if err != nil {
			t.Fatalf("convert: %v", err)
		}

		resultStr := formatResult(result)

		if !strings.Contains(resultStr, "Visible hint") {
			t.Errorf("hint should be visible: %s", resultStr)
		}
		if !strings.Contains(resultStr, "Visible example") {
			t.Errorf("example should be visible: %s", resultStr)
		}
		if strings.Contains(resultStr, "hidden") {
			t.Errorf("image should be hidden: %s", resultStr)
		}
	})
}

// TestVersionEnforcementInConvert verifies version constraints are enforced during conversion.
func TestVersionEnforcementInConvert(t *testing.T) {
	t.Run("valid_version_passes", func(t *testing.T) {
		input := `<poml>
<meta>
	<version>0.0.8</version>
	<minVersion>0.0.1</minVersion>
	<maxVersion>99.0.0</maxVersion>
</meta>
<role>Test version</role>
<task>Test</task>
</poml>`

		doc, err := ParseString(input)
		if err != nil {
			t.Fatalf("parse: %v", err)
		}

		// Should succeed with enforcement enabled
		_, err = Convert(doc, FormatDict, ConvertOptions{
			EnforceVersions: true,
		})
		if err != nil {
			t.Errorf("conversion should succeed with valid version: %v", err)
		}
	})

	t.Run("minVersion_too_high_fails", func(t *testing.T) {
		input := `<poml>
<meta>
	<version>0.0.8</version>
	<minVersion>999.0.0</minVersion>
</meta>
<role>Test version</role>
<task>Test</task>
</poml>`

		doc, err := ParseString(input)
		if err != nil {
			t.Fatalf("parse: %v", err)
		}

		// Should fail with enforcement enabled
		_, err = Convert(doc, FormatDict, ConvertOptions{
			EnforceVersions: true,
		})
		if err == nil {
			t.Error("conversion should fail when minVersion is too high")
		}
		if !strings.Contains(err.Error(), "version constraint") {
			t.Errorf("error should mention version constraint: %v", err)
		}
	})

	t.Run("maxVersion_too_low_fails", func(t *testing.T) {
		input := `<poml>
<meta>
	<version>0.0.8</version>
	<maxVersion>0.0.1</maxVersion>
</meta>
<role>Test version</role>
<task>Test</task>
</poml>`

		doc, err := ParseString(input)
		if err != nil {
			t.Fatalf("parse: %v", err)
		}

		// Should fail with enforcement enabled
		_, err = Convert(doc, FormatDict, ConvertOptions{
			EnforceVersions: true,
		})
		if err == nil {
			t.Error("conversion should fail when maxVersion is too low")
		}
	})

	t.Run("enforcement_disabled_allows_invalid", func(t *testing.T) {
		input := `<poml>
<meta>
	<version>0.0.8</version>
	<minVersion>999.0.0</minVersion>
</meta>
<role>Test version</role>
<task>Test</task>
</poml>`

		doc, err := ParseString(input)
		if err != nil {
			t.Fatalf("parse: %v", err)
		}

		// Should succeed without enforcement
		_, err = Convert(doc, FormatDict, ConvertOptions{
			EnforceVersions: false,
		})
		if err != nil {
			t.Errorf("conversion should succeed without enforcement: %v", err)
		}
	})
}

// formatResult converts a conversion result to string for assertions.
func formatResult(result any) string {
	switch v := result.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	default:
		// JSON marshal for complex types
		data, _ := json.Marshal(result)
		return string(data)
	}
}
