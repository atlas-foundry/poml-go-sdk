package main

import (
	"bytes"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"

	"github.com/atlas-foundry/poml-go-sdk/poml"
)

var (
	numFiles = flag.Int("n", 200, "number of files to generate")
	outDir   = flag.String("out", "poml/testdata/generated", "output directory")
	seed     = flag.Int64("seed", 12345, "random seed")
)

type Generator struct {
	rng *rand.Rand
}

func NewGenerator(seed int64) *Generator {
	return &Generator{
		rng: rand.New(rand.NewSource(seed)),
	}
}

func (g *Generator) GeneratePOML() string {
	var sb strings.Builder
	sb.WriteString("<poml>\n")

	// Meta
	sb.WriteString(fmt.Sprintf("  <version>%d.%d</version>\n", g.rng.Intn(2)+1, g.rng.Intn(10)))
	sb.WriteString(fmt.Sprintf("  <owner>%s</owner>\n", g.randomString(5, 10)))

	// Persona
	sb.WriteString("  <persona>\n")
	sb.WriteString(g.randomText(20, 100))
	sb.WriteString("\n  </persona>\n")

	// Tasks (1 to 3)
	numTasks := g.rng.Intn(3) + 1
	for i := 0; i < numTasks; i++ {
		sb.WriteString("  <task>\n")
		sb.WriteString(g.randomText(10, 50))
		sb.WriteString("\n  </task>\n")
	}

	// Random Elements
	numElements := g.rng.Intn(10) + 5
	for i := 0; i < numElements; i++ {
		sb.WriteString(g.generateElement(1))
	}

	sb.WriteString("</poml>")
	return sb.String()
}

func (g *Generator) generateElement(depth int) string {
	if depth > 5 {
		return ""
	}

	indent := strings.Repeat("  ", depth+1)

	// Types of elements:
	// 0: Standard (input, output-format, example, hint)
	// 1: Context (context, user, model)
	// 2: Media (image, audio, video, document)
	// 3: JSX/Unknown

	typeChoice := g.rng.Intn(10)

	var tagName string
	var attributes string
	var content string
	var isSelfClosing bool

	switch {
	case typeChoice < 3: // Standard
		tags := []string{"input", "output-format", "example", "hint"}
		tagName = tags[g.rng.Intn(len(tags))]
		if tagName == "input" {
			attributes = fmt.Sprintf(` name="%s"`, g.randomString(3, 8))
		}
		content = g.randomText(10, 50)

	case typeChoice < 5: // Context
		tags := []string{"context", "user", "model"}
		tagName = tags[g.rng.Intn(len(tags))]
		content = g.randomText(10, 50)
		// Maybe add nested elements
		if g.rng.Float32() < 0.3 {
			content += "\n" + g.generateElement(depth+1) + indent
		}

	case typeChoice < 7: // Media
		tags := []string{"image", "audio", "video", "document"}
		tagName = tags[g.rng.Intn(len(tags))]
		attributes = fmt.Sprintf(` src="%s" alt="%s"`, g.randomURL(), g.randomString(5, 15))
		isSelfClosing = true

	default: // JSX/Unknown (30% chance)
		tagName = g.randomJSXTag()
		attributes = g.randomAttributes()
		if g.rng.Float32() < 0.5 {
			isSelfClosing = true
		} else {
			content = g.randomText(5, 30)
			if g.rng.Float32() < 0.3 {
				content += "\n" + g.generateElement(depth+1) + indent
			}
		}
	}

	var sb strings.Builder
	sb.WriteString(indent)
	sb.WriteString("<" + tagName + attributes)
	if isSelfClosing {
		sb.WriteString(" />\n")
	} else {
		sb.WriteString(">\n")
		sb.WriteString(indent + "  " + content + "\n")
		sb.WriteString(indent + "</" + tagName + ">\n")
	}
	return sb.String()
}

func (g *Generator) randomString(min, max int) string {
	length := g.rng.Intn(max-min+1) + min
	chars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	b := make([]byte, length)
	for i := range b {
		b[i] = chars[g.rng.Intn(len(chars))]
	}
	return string(b)
}

func (g *Generator) randomText(min, max int) string {
	words := []string{"lorem", "ipsum", "dolor", "sit", "amet", "consectetur", "adipiscing", "elit", "sed", "do", "eiusmod", "tempor", "incididunt", "ut", "labore", "et", "dolore", "magna", "aliqua"}
	length := g.rng.Intn(max-min+1) + min
	var sb strings.Builder
	for i := 0; i < length; i++ {
		sb.WriteString(words[g.rng.Intn(len(words))])
		if i < length-1 {
			sb.WriteString(" ")
		}
	}
	return sb.String()
}

func (g *Generator) randomURL() string {
	return "https://example.com/" + g.randomString(5, 10) + ".ext"
}

func (g *Generator) randomJSXTag() string {
	// Mix of PascalCase (React components) and camelCase or lowercase
	if g.rng.Float32() < 0.7 {
		// PascalCase
		return strings.ToUpper(g.randomString(1, 1)) + g.randomString(3, 8)
	}
	return g.randomString(4, 10)
}

func (g *Generator) randomAttributes() string {
	numAttrs := g.rng.Intn(4)
	var sb strings.Builder
	for i := 0; i < numAttrs; i++ {
		key := g.randomString(3, 8)
		val := g.randomString(3, 10)
		sb.WriteString(fmt.Sprintf(` %s="%s"`, key, val))
	}
	return sb.String()
}

func main() {
	fmt.Println("Starting generator...")
	flag.Parse()

	if err := os.MkdirAll(*outDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "failed to create output directory: %v\n", err)
		os.Exit(1)
	}

	gen := NewGenerator(*seed)

	for i := 0; i < *numFiles; i++ {
		content := gen.GeneratePOML()

		// Validate and Normalize
		doc, err := poml.ParseString(content)
		if err != nil {
			fmt.Printf("Skipping invalid generated file %d: %v\n", i, err)
			continue
		}

		// Encode back to string to get canonical format
		var buf bytes.Buffer
		err = doc.EncodeWithOptions(&buf, poml.EncodeOptions{
			PreserveWS:    true,
			PreserveOrder: true,
			IncludeHeader: false,
		})
		if err != nil {
			fmt.Printf("Failed to encode file %d: %v\n", i, err)
			continue
		}

		finalContent := buf.Bytes()

		filename := filepath.Join(*outDir, fmt.Sprintf("gen_%03d.poml", i))
		if err := os.WriteFile(filename, finalContent, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "failed to write file %s: %v\n", filename, err)
		}
	}
	fmt.Printf("Generated %d files in %s\n", *numFiles, *outDir)
}
