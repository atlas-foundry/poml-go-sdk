package main

import (
	"os"
	"testing"

	"github.com/atlas-foundry/poml-go-sdk/poml"
	"strings"
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
