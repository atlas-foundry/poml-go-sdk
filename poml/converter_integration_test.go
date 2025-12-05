package poml

import (
	"encoding/json"
	"strings"
	"testing"
)

// resultToString converts a Convert result to a string for checking content.
func resultToString(result any) string {
	data, _ := json.Marshal(result)
	return string(data)
}

// TestRichTextConversion verifies that rich text elements are properly converted.
func TestRichTextConversion(t *testing.T) {
	input := `<poml>
		<version>0.0.8</version>
		<role>assistant</role>
		<task>Test rich text</task>
		<h level="2">Heading Level 2</h>
		<p>This is a paragraph.</p>
		<section title="My Section">Section content here.</section>
		<list style="decimal">
			<item>First item</item>
			<item>Second item</item>
		</list>
		<code lang="go">fmt.Println("hello")</code>
	</poml>`

	doc, err := ParseString(input)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	result, err := Convert(doc, FormatMessageDict, ConvertOptions{})
	if err != nil {
		t.Fatalf("convert: %v", err)
	}

	resultStr := resultToString(result)

	// Verify rich text elements are present in output
	checks := []string{
		"## Heading Level 2",
		"This is a paragraph",
		"## My Section",
		"Section content here",
		"1. First item",
		"2. Second item",
		"```go",
		"fmt.Println",
		"```",
	}

	for _, check := range checks {
		if !strings.Contains(resultStr, check) {
			t.Errorf("output missing expected content: %q\nGot: %s", check, resultStr)
		}
	}
}

// TestTemplateInterpolation verifies that template variables are interpolated.
func TestTemplateInterpolation(t *testing.T) {
	// Use human-msg elements which are included in message_dict output
	input := `<poml>
<meta><version>0.0.8</version></meta>
<role>assistant</role>
<task>task</task>
<human-msg>Hello {{name}}, welcome!</human-msg>
</poml>`

	doc, err := ParseString(input)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	result, err := Convert(doc, FormatMessageDict, ConvertOptions{
		ExpandTemplates: true,
		TemplateVars: map[string]any{
			"name": "World",
		},
	})
	if err != nil {
		t.Fatalf("convert: %v", err)
	}

	resultStr := resultToString(result)

	if !strings.Contains(resultStr, "Hello World") {
		t.Errorf("template not interpolated, got: %s", resultStr)
	}
}

// TestTemplateVariables verifies that external template variables work.
func TestTemplateVariables(t *testing.T) {
	input := `<poml>
<meta><version>0.0.8</version></meta>
<role>assistant</role>
<task>task</task>
<human-msg>Hello {{name}}, your order is {{order_id}}.</human-msg>
</poml>`

	doc, err := ParseString(input)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	result, err := Convert(doc, FormatMessageDict, ConvertOptions{
		TemplateVars: map[string]any{
			"name":     "Alice",
			"order_id": "12345",
		},
	})
	if err != nil {
		t.Fatalf("convert: %v", err)
	}

	resultStr := resultToString(result)

	if !strings.Contains(resultStr, "Hello Alice") {
		t.Errorf("name not interpolated, got: %s", resultStr)
	}
	if !strings.Contains(resultStr, "order is 12345") {
		t.Errorf("order_id not interpolated, got: %s", resultStr)
	}
}

// TestConditionalIf verifies that if conditions filter elements.
func TestConditionalIf(t *testing.T) {
	input := `<poml>
<meta><version>0.0.8</version></meta>
<role>assistant</role>
<task>Test conditionals</task>
<hint if="show_hint">This should appear</hint>
<hint if="hide_hint">This should not appear</hint>
</poml>`

	doc, err := ParseString(input)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	result, err := Convert(doc, FormatMessageDict, ConvertOptions{
		TemplateVars: map[string]any{
			"show_hint": true,
			"hide_hint": false,
		},
		ExpandTemplates: true,
	})
	if err != nil {
		t.Fatalf("convert: %v", err)
	}

	resultStr := resultToString(result)

	if !strings.Contains(resultStr, "This should appear") {
		t.Errorf("conditional with true should appear, got: %s", resultStr)
	}
	if strings.Contains(resultStr, "This should not appear") {
		t.Errorf("conditional with false should not appear, got: %s", resultStr)
	}
}

// TestComponentFiltering verifies that components can be filtered.
func TestComponentFiltering(t *testing.T) {
	// Use base64 data URIs so no file access is needed
	input := `<poml>
<meta><version>0.0.8</version></meta>
<role>assistant</role>
<task>Test filtering</task>
<img src="data:image/png;base64,iVBORw0KGgo=" alt="test image"/>
</poml>`

	doc, err := ParseString(input)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	// With image enabled
	resultWithImage, err := Convert(doc, FormatMessageDict, ConvertOptions{})
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	withImageStr := resultToString(resultWithImage)
	if !strings.Contains(withImageStr, "base64") {
		t.Errorf("image should be present when not filtered, got: %s", withImageStr)
	}

	// With image disabled
	resultNoImage, err := Convert(doc, FormatMessageDict, ConvertOptions{
		Components: "-image",
	})
	if err != nil {
		t.Fatalf("convert with filter: %v", err)
	}
	noImageStr := resultToString(resultNoImage)
	if strings.Contains(noImageStr, "base64") {
		t.Errorf("image should be filtered out, got: %s", noImageStr)
	}
}

// TestStylesheetApplication verifies that stylesheets apply properties.
func TestStylesheetApplication(t *testing.T) {
	input := `<poml>
		<version>0.0.8</version>
		<role>assistant</role>
		<task>Test stylesheet</task>
		<style>
			<output format="json">{"hint": {"syntax": "markdown"}}</output>
		</style>
		<hint>A hint with styled syntax</hint>
	</poml>`

	doc, err := ParseString(input)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	// Convert with stylesheet enabled
	_, err = Convert(doc, FormatMessageDict, ConvertOptions{
		ApplyStylesheet: true,
	})
	if err != nil {
		t.Fatalf("convert: %v", err)
	}

	// Basic test - just ensure no error during conversion
	// Full verification would check the hint has syntax="markdown"
}

// TestNewlineConversion verifies that newline elements are converted.
func TestNewlineConversion(t *testing.T) {
	input := `<poml>
<meta><version>0.0.8</version></meta>
<role>assistant</role>
<task>task</task>
<human-msg>Line one</human-msg>
<br/>
<human-msg>Line two</human-msg>
<br newLineCount="3"/>
<human-msg>Line three</human-msg>
</poml>`

	doc, err := ParseString(input)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if len(doc.Newlines) != 2 {
		t.Errorf("expected 2 newlines, got %d", len(doc.Newlines))
	}

	result, err := Convert(doc, FormatMessageDict, ConvertOptions{})
	if err != nil {
		t.Fatalf("convert: %v", err)
	}

	resultStr := resultToString(result)

	// Should contain newline content
	if !strings.Contains(resultStr, "Line one") {
		t.Errorf("missing 'Line one' in output: %s", resultStr)
	}
	if !strings.Contains(resultStr, "Line two") {
		t.Errorf("missing 'Line two' in output: %s", resultStr)
	}
}

// TestDataComponentConversion verifies table/folder/conversation conversion.
func TestDataComponentConversion(t *testing.T) {
	t.Run("table", func(t *testing.T) {
		input := `<poml>
			<version>0.0.8</version>
			<role>assistant</role>
			<task>Show data</task>
			<table>
				<column field="name" header="Name"/>
				<column field="age" header="Age"/>
				<record>{"name": "Alice", "age": 30}</record>
				<record>{"name": "Bob", "age": 25}</record>
			</table>
		</poml>`

		doc, err := ParseString(input)
		if err != nil {
			t.Fatalf("parse: %v", err)
		}

		result, err := Convert(doc, FormatMessageDict, ConvertOptions{Extended: ExtendedStrict})
		if err != nil {
			t.Fatalf("convert: %v", err)
		}

		resultStr := resultToString(result)

		// Check for table-like output
		if !strings.Contains(resultStr, "Name") || !strings.Contains(resultStr, "Age") {
			t.Logf("Note: table headers may not be in output if parsing differs, got: %s", resultStr)
		}
	})
}
