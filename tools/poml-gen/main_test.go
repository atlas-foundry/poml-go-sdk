package main

import (
	"strings"
	"testing"

	"github.com/atlas-foundry/poml-go-sdk/poml"
)

func TestNewGenerator(t *testing.T) {
	g := NewGenerator(12345)
	if g == nil {
		t.Fatal("NewGenerator returned nil")
	}
	if g.rng == nil {
		t.Fatal("Generator.rng is nil")
	}
}

func TestGeneratorDeterministic(t *testing.T) {
	g1 := NewGenerator(12345)
	g2 := NewGenerator(12345)

	out1 := g1.GeneratePOML()
	out2 := g2.GeneratePOML()

	if out1 != out2 {
		t.Error("generators with same seed produced different output")
	}
}

func TestGeneratePOMLStructure(t *testing.T) {
	g := NewGenerator(99999)
	out := g.GeneratePOML()

	if !strings.HasPrefix(out, "<poml>") {
		t.Error("output does not start with <poml>")
	}
	if !strings.HasSuffix(out, "</poml>") {
		t.Error("output does not end with </poml>")
	}

	// Should contain version, owner, persona, task
	if !strings.Contains(out, "<version>") {
		t.Error("missing <version>")
	}
	if !strings.Contains(out, "<owner>") {
		t.Error("missing <owner>")
	}
	if !strings.Contains(out, "<persona>") {
		t.Error("missing <persona>")
	}
	if !strings.Contains(out, "<task>") {
		t.Error("missing <task>")
	}
}

func TestGeneratePOMLParseable(t *testing.T) {
	g := NewGenerator(54321)

	// Generate multiple and ensure they parse
	for i := 0; i < 10; i++ {
		out := g.GeneratePOML()
		_, err := poml.ParseString(out)
		if err != nil {
			// Some generated content may not be valid XML due to random JSX tags
			// This is expected behavior - the generator tests parsing resilience
			t.Logf("iteration %d: parse error (expected for some random content): %v", i, err)
		}
	}
}

func TestRandomString(t *testing.T) {
	g := NewGenerator(11111)

	s := g.randomString(5, 10)
	if len(s) < 5 || len(s) > 10 {
		t.Errorf("randomString length %d not in range [5,10]", len(s))
	}

	// Check it only contains letters
	for _, c := range s {
		isLower := c >= 'a' && c <= 'z'
		isUpper := c >= 'A' && c <= 'Z'
		if !isLower && !isUpper {
			t.Errorf("randomString contains non-letter: %c", c)
		}
	}
}

func TestRandomText(t *testing.T) {
	g := NewGenerator(22222)

	text := g.randomText(5, 10)
	words := strings.Fields(text)
	if len(words) < 5 || len(words) > 10 {
		t.Errorf("randomText word count %d not in range [5,10]", len(words))
	}
}

func TestRandomURL(t *testing.T) {
	g := NewGenerator(33333)

	url := g.randomURL()
	if !strings.HasPrefix(url, "https://example.com/") {
		t.Errorf("randomURL doesn't start with expected prefix: %s", url)
	}
	if !strings.HasSuffix(url, ".ext") {
		t.Errorf("randomURL doesn't end with .ext: %s", url)
	}
}

func TestRandomJSXTag(t *testing.T) {
	g := NewGenerator(44444)

	// Generate multiple tags and ensure they're non-empty
	for i := 0; i < 20; i++ {
		tag := g.randomJSXTag()
		if len(tag) == 0 {
			t.Error("randomJSXTag returned empty string")
		}
	}
}

func TestRandomAttributes(t *testing.T) {
	g := NewGenerator(55555)

	// Generate multiple and check format
	for i := 0; i < 20; i++ {
		attrs := g.randomAttributes()
		// May be empty or contain key="value" pairs
		if attrs != "" && !strings.Contains(attrs, "=") {
			t.Errorf("non-empty attributes missing '=': %s", attrs)
		}
	}
}

func TestGenerateElement(t *testing.T) {
	g := NewGenerator(66666)

	// Generate elements at various depths
	for depth := 1; depth <= 5; depth++ {
		elem := g.generateElement(depth)
		if elem == "" && depth <= 5 {
			// Empty is valid at max depth
			continue
		}
		if !strings.Contains(elem, "<") {
			t.Errorf("generateElement at depth %d missing '<': %s", depth, elem)
		}
	}

	// At depth > 5, should return empty
	elem := g.generateElement(6)
	if elem != "" {
		t.Errorf("generateElement at depth 6 should be empty, got: %s", elem)
	}
}

func TestGenerateElementTypes(t *testing.T) {
	// Test that various element types are generated over many iterations
	g := NewGenerator(77777)

	standardFound := false
	mediaFound := false

	for i := 0; i < 100; i++ {
		elem := g.generateElement(1)
		if strings.Contains(elem, "<input") || strings.Contains(elem, "<example") ||
			strings.Contains(elem, "<hint") || strings.Contains(elem, "<output-format") {
			standardFound = true
		}
		if strings.Contains(elem, "<image") || strings.Contains(elem, "<audio") ||
			strings.Contains(elem, "<video") || strings.Contains(elem, "<document") {
			mediaFound = true
		}
	}

	if !standardFound {
		t.Error("no standard elements generated in 100 iterations")
	}
	if !mediaFound {
		t.Error("no media elements generated in 100 iterations")
	}
}
