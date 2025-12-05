// Example: Format Comparison
//
// This example shows the same POML document converted to all available formats,
// demonstrating the differences between each output structure.
//
// Run: go run main.go
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/atlas-foundry/poml-go-sdk/poml"
)

func main() {
	// A rich POML document with multiple features
	input := `<poml>
  <meta>
    <id>code-reviewer</id>
    <version>2.0.0</version>
    <owner>dev-team</owner>
  </meta>

  <role>You are a senior code reviewer with expertise in Go, Python, and TypeScript.</role>

  <task>Review the provided code for bugs, performance issues, and best practices.</task>
  <task>Suggest improvements with clear explanations.</task>

  <hint>Focus on readability and maintainability over micro-optimizations.</hint>
  <hint>Be constructive and educational in your feedback.</hint>

  <example>
    <human-msg>Review this function:
def add(a, b):
    return a + b</human-msg>
    <assistant-msg>The function is simple and correct. Consider adding type hints:
def add(a: int, b: int) -> int:
    return a + b</assistant-msg>
  </example>

  <runtime temperature="0.3" max-tokens="2048"/>

  <human-msg>Please review this code:
func fetch(url string) {
    resp, _ := http.Get(url)
    body, _ := ioutil.ReadAll(resp.Body)
    fmt.Println(string(body))
}</human-msg>
</poml>`

	doc, err := poml.ParseString(input)
	if err != nil {
		log.Fatalf("Parse error: %v", err)
	}

	if err := doc.Validate(); err != nil {
		log.Fatalf("Validation error: %v", err)
	}

	// All available formats
	allFormats := []struct {
		format poml.Format
		desc   string
	}{
		{poml.FormatMessageDict, "Simple speaker/content tuples - lightweight, universal"},
		{poml.FormatDict, "Full dict with messages, tools, schema, runtime"},
		{poml.FormatOpenAIChat, "OpenAI API format - ready for chat completions"},
		{poml.FormatLangChain, "LangChain format - for LangChain integrations"},
		{poml.FormatPydantic, "Pydantic-compatible - matches Python SDK output"},
	}

	fmt.Println("=" + strings.Repeat("=", 79))
	fmt.Println("FORMAT COMPARISON: Same POML, Different Outputs")
	fmt.Println("=" + strings.Repeat("=", 79))
	fmt.Printf("\nDocument: %s v%s\n", doc.Meta.ID, doc.Meta.Version)
	fmt.Printf("Elements: %d | Tasks: %d | Hints: %d\n\n", len(doc.Elements), len(doc.Tasks), len(doc.Hints))

	for _, f := range allFormats {
		fmt.Println("-" + strings.Repeat("-", 79))
		fmt.Printf("FORMAT: %s\n", f.format)
		fmt.Printf("USE CASE: %s\n", f.desc)
		fmt.Println("-" + strings.Repeat("-", 79))

		output, err := poml.Convert(doc, f.format, poml.ConvertOptions{})
		if err != nil {
			log.Printf("Error: %v\n\n", err)
			continue
		}

		jsonBytes, _ := json.MarshalIndent(output, "", "  ")
		// Truncate for readability
		outputStr := string(jsonBytes)
		if len(outputStr) > 2000 {
			outputStr = outputStr[:2000] + "\n  ... (truncated)"
		}
		fmt.Println(outputStr)
		fmt.Println()
	}

	// Summary table
	fmt.Println("=" + strings.Repeat("=", 79))
	fmt.Println("QUICK REFERENCE")
	fmt.Println("=" + strings.Repeat("=", 79))
	fmt.Print(`
| Format       | Structure              | Best For                    |
|--------------|------------------------|-----------------------------|
| message_dict | [{speaker, content}]   | Simple message passing      |
| dict         | {messages, tools, ...} | Full feature access         |
| openai_chat  | OpenAI API structure   | Direct OpenAI API calls     |
| langchain    | LangChain messages     | LangChain/LangGraph apps    |
| pydantic     | Python-compatible dict | Cross-SDK compatibility     |
| scene        | Visual diagram         | Diagram/flowchart rendering |
| scenejson    | Scene as JSON          | Serialized scene output     |
`)
}
