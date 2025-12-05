// Example: Basic POML Parsing and Conversion
//
// This example demonstrates the fundamental workflow:
// 1. Parse a POML document
// 2. Validate it
// 3. Convert to different output formats
//
// Run: go run main.go
package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/atlas-foundry/poml-go-sdk/poml"
)

func main() {
	// Example POML document
	input := `<poml>
  <meta>
    <id>greeting-assistant</id>
    <version>1.0.0</version>
    <owner>examples</owner>
  </meta>

  <role>You are a friendly greeting assistant.</role>

  <task>Generate a warm, personalized greeting for the user.</task>

  <hint>Keep greetings concise but warm. Use the user's name if provided.</hint>

  <example>
    <human-msg>Hi, I'm Alice!</human-msg>
    <assistant-msg>Hello Alice! It's wonderful to meet you. How can I brighten your day?</assistant-msg>
  </example>

  <human-msg>My name is Bob.</human-msg>
</poml>`

	// Step 1: Parse the POML document
	fmt.Println("=== Step 1: Parsing POML ===")
	doc, err := poml.ParseString(input)
	if err != nil {
		log.Fatalf("Parse error: %v", err)
	}
	fmt.Printf("Parsed document: %s v%s\n", doc.Meta.ID, doc.Meta.Version)
	fmt.Printf("Role: %s\n", doc.Role)
	fmt.Printf("Tasks: %d, Elements: %d\n\n", len(doc.Tasks), len(doc.Elements))

	// Step 2: Validate the document
	fmt.Println("=== Step 2: Validating ===")
	if err := doc.Validate(); err != nil {
		log.Fatalf("Validation error: %v", err)
	}
	fmt.Println("Document is valid!")
	fmt.Println()

	// Step 3: Convert to different formats
	formats := []poml.Format{
		poml.FormatMessageDict,
		poml.FormatOpenAIChat,
		poml.FormatLangChain,
	}

	for _, format := range formats {
		fmt.Printf("=== Format: %s ===\n", format)
		output, err := poml.Convert(doc, format, poml.ConvertOptions{})
		if err != nil {
			log.Printf("Convert error for %s: %v", format, err)
			continue
		}

		// Pretty print the output
		jsonBytes, _ := json.MarshalIndent(output, "", "  ")
		fmt.Println(string(jsonBytes))
		fmt.Println()
	}
}
