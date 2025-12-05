package main

import (
	"os"
	"strings"
	"testing"

	"github.com/atlas-foundry/poml-go-sdk/poml"
)

func TestExtractListFromConfig(t *testing.T) {
	cfgPath := "../poml/testdata/examples/validation_config.poml"
	body, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Skip("missing config fixture")
	}
	doc, err := poml.ParseReaderWithOptions(strings.NewReader(string(body)), poml.ParseOptions{PreserveWhitespace: true})
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	mimes := extractListFromConfig(doc, "allowed-mime")
	kinds := extractListFromConfig(doc, "allowed-op-kinds")
	if len(mimes) == 0 || len(kinds) == 0 {
		t.Fatalf("expected lists from config, got mimes=%v kinds=%v", mimes, kinds)
	}
	if mimes[0] != "image/tiff" {
		t.Fatalf("unexpected mime list: %v", mimes)
	}
	if kinds[0] != "custom-kind" {
		t.Fatalf("unexpected kinds list: %v", kinds)
	}
}

func TestExtractListFromConfigInline(t *testing.T) {
	// Test with inline POML that has object tags
	doc, err := poml.ParseString(`<poml>
		<meta><id>cfg</id><version>1</version><owner>test</owner></meta>
		<role>r</role>
		<task>t</task>
		<object name="allowed-mime">["image/webp", "video/mp4"]</object>
		<object name="allowed-op-kinds">["custom", "special"]</object>
	</poml>`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	mimes := extractListFromConfig(doc, "allowed-mime")
	if len(mimes) != 2 {
		t.Fatalf("expected 2 mimes, got %v", mimes)
	}
	if mimes[0] != "image/webp" || mimes[1] != "video/mp4" {
		t.Fatalf("unexpected mimes: %v", mimes)
	}

	kinds := extractListFromConfig(doc, "allowed-op-kinds")
	if len(kinds) != 2 {
		t.Fatalf("expected 2 kinds, got %v", kinds)
	}
}

func TestExtractListFromConfigEmpty(t *testing.T) {
	doc, _ := poml.ParseString(`<poml><meta><id>a</id><version>1</version><owner>o</owner></meta><role>r</role><task>t</task></poml>`)
	result := extractListFromConfig(doc, "nonexistent")
	if len(result) != 0 {
		t.Fatalf("expected empty result for nonexistent key, got %v", result)
	}
}

func TestExtractListFromConfigInvalidJSON(t *testing.T) {
	doc, err := poml.ParseString(`<poml>
		<meta><id>cfg</id><version>1</version><owner>test</owner></meta>
		<role>r</role>
		<task>t</task>
		<object name="test-key">not valid json</object>
	</poml>`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	result := extractListFromConfig(doc, "test-key")
	if len(result) != 0 {
		t.Fatalf("expected empty result for invalid json, got %v", result)
	}
}

func TestExtractListFromConfigEmptyBody(t *testing.T) {
	doc, err := poml.ParseString(`<poml>
		<meta><id>cfg</id><version>1</version><owner>test</owner></meta>
		<role>r</role>
		<task>t</task>
		<object name="empty-key"></object>
	</poml>`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	result := extractListFromConfig(doc, "empty-key")
	if len(result) != 0 {
		t.Fatalf("expected empty result for empty body, got %v", result)
	}
}

func TestExtractListFromConfigWithDataAttr(t *testing.T) {
	doc, err := poml.ParseString(`<poml>
		<meta><id>cfg</id><version>1</version><owner>test</owner></meta>
		<role>r</role>
		<task>t</task>
		<object name="data-key" data='["a", "b"]'></object>
	</poml>`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	result := extractListFromConfig(doc, "data-key")
	if len(result) != 2 {
		t.Fatalf("expected 2 items from data attr, got %v", result)
	}
}
