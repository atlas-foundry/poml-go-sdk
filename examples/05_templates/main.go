// Example: Template Features
//
// This example demonstrates POML template capabilities:
// 1. Variables with <let> bindings
// 2. String interpolation with {{variable}}
// 3. Conditionals with <if>/<else>
// 4. Loops with <for>
// 5. Includes with <include>
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
	// POML with template features
	input := `<poml>
  <meta>
    <id>templated-prompt</id>
    <version>1.0.0</version>
    <owner>examples</owner>
  </meta>

  <!-- Variable bindings -->
  <let name="assistant_name" value="Claude"/>
  <let name="language" value="Go"/>
  <let name="expertise_level" value="senior"/>

  <role>You are {{assistant_name}}, a {{expertise_level}} {{language}} developer.</role>

  <task>Help users write idiomatic {{language}} code.</task>

  <!-- Conditional content -->
  <if cond="expertise_level == 'senior'">
    <hint>Provide advanced patterns and best practices.</hint>
    <hint>Discuss trade-offs and architectural considerations.</hint>
  </if>
  <if cond="expertise_level == 'junior'">
    <hint>Keep explanations simple and beginner-friendly.</hint>
    <hint>Provide more code examples.</hint>
  </if>

  <!-- Loop over items -->
  <hint>Key {{language}} principles to follow:</hint>
  <for each="principle" in="['simplicity', 'readability', 'composition']">
    <hint>- Focus on {{principle}}</hint>
  </for>

  <example>
    <human-msg>How do I handle errors in {{language}}?</human-msg>
    <assistant-msg>In {{language}}, errors are values. Here's the idiomatic pattern:

` + "```" + `{{language}}
result, err := doSomething()
if err != nil {
    return fmt.Errorf("operation failed: %w", err)
}
` + "```" + `</assistant-msg>
  </example>

  <human-msg>Write a function to read a file.</human-msg>
</poml>`

	fmt.Println("=== Parsing with Template Expansion ===")
	doc, err := poml.ParseString(input)
	if err != nil {
		log.Fatalf("Parse error: %v", err)
	}

	if err := doc.Validate(); err != nil {
		log.Fatalf("Validation error: %v", err)
	}

	// Show let bindings
	fmt.Println("\nLet Bindings:")
	for _, binding := range doc.LetBindings {
		fmt.Printf("  %s = %q\n", binding.Name, binding.Value)
	}

	// Convert with template expansion enabled (default)
	fmt.Println("\n=== With Template Expansion (default) ===")
	output, err := poml.Convert(doc, poml.FormatMessageDict, poml.ConvertOptions{
		ExpandTemplates: true, // This is the default
	})
	if err != nil {
		log.Fatalf("Convert error: %v", err)
	}

	jsonBytes, _ := json.MarshalIndent(output, "", "  ")
	fmt.Println(string(jsonBytes))

	// Show the expanded role
	fmt.Println("\n=== Key Observations ===")
	fmt.Println("1. Variables ({{assistant_name}}, etc.) are replaced with values")
	fmt.Println("2. Conditionals evaluate based on variable values")
	fmt.Println("3. Loops expand into multiple elements")
	fmt.Println("4. The final output is clean, with no template syntax")

	// Runtime variables example
	fmt.Println("\n=== Runtime Variables ===")
	inputWithRuntime := `<poml>
  <meta><id>runtime-demo</id><version>1.0</version><owner>demo</owner></meta>

  <let name="user_name" runtime="true"/>
  <let name="topic" runtime="true"/>

  <role>You help {{user_name}} learn about {{topic}}.</role>
  <task>Explain {{topic}} concepts clearly.</task>
  <human-msg>Tell me about {{topic}}.</human-msg>
</poml>`

	docRuntime, _ := poml.ParseString(inputWithRuntime)
	_ = docRuntime.Validate()

	// Convert with runtime variable substitution
	outputRuntime, _ := poml.Convert(docRuntime, poml.FormatMessageDict, poml.ConvertOptions{
		TemplateVars: map[string]any{
			"user_name": "Alice",
			"topic":     "concurrency in Go",
		},
	})

	runtimeJSON, _ := json.MarshalIndent(outputRuntime, "", "  ")
	fmt.Println("With runtime variables {user_name: Alice, topic: concurrency in Go}:")
	fmt.Println(string(runtimeJSON))
}
