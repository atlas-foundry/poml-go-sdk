// Example: Tool Definitions and Tool Calls
//
// This example demonstrates how POML handles:
// 1. Tool definitions with JSON schemas
// 2. Tool call requests from the assistant
// 3. Tool responses from the system
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
	// POML with tool definitions and a tool call/response flow
	input := `<poml>
  <meta>
    <id>weather-assistant</id>
    <version>1.0.0</version>
    <owner>examples</owner>
  </meta>

  <role>You are a helpful weather assistant that can look up current weather conditions.</role>

  <task>Help users get weather information for any location.</task>

  <!-- Tool definition with JSON schema -->
  <tool-definition name="get_weather" description="Get current weather for a location">
    <schema>
      {
        "type": "object",
        "properties": {
          "location": {
            "type": "string",
            "description": "City name or coordinates"
          },
          "units": {
            "type": "string",
            "enum": ["celsius", "fahrenheit"],
            "default": "celsius"
          }
        },
        "required": ["location"]
      }
    </schema>
  </tool-definition>

  <tool-definition name="get_forecast" description="Get 5-day weather forecast">
    <schema>
      {
        "type": "object",
        "properties": {
          "location": {"type": "string"},
          "days": {"type": "integer", "minimum": 1, "maximum": 7}
        },
        "required": ["location"]
      }
    </schema>
  </tool-definition>

  <!-- Conversation with tool usage -->
  <human-msg>What's the weather like in Tokyo?</human-msg>

  <!-- Assistant requests a tool call -->
  <assistant-msg>
    <tool-call id="call_123" name="get_weather">
      {"location": "Tokyo", "units": "celsius"}
    </tool-call>
  </assistant-msg>

  <!-- Tool response -->
  <tool-response call-id="call_123">
    {"temperature": 22, "condition": "Partly Cloudy", "humidity": 65}
  </tool-response>

  <!-- Assistant uses the tool result -->
  <assistant-msg>The current weather in Tokyo is 22°C and partly cloudy with 65% humidity.</assistant-msg>

  <!-- Another user question -->
  <human-msg>Will it rain this week?</human-msg>
</poml>`

	doc, err := poml.ParseString(input)
	if err != nil {
		log.Fatalf("Parse error: %v", err)
	}

	if err := doc.Validate(); err != nil {
		log.Fatalf("Validation error: %v", err)
	}

	fmt.Println("=== Tool Definitions ===")
	fmt.Printf("Document has %d tool definitions\n\n", len(doc.ToolDefs))

	for _, tool := range doc.ToolDefs {
		fmt.Printf("Tool: %s\n", tool.Name)
		fmt.Printf("  Description: %s\n", tool.Description)
		if tool.Body != "" {
			fmt.Printf("  Body: %s\n", truncateBody(tool.Body, 100))
		}
		fmt.Println()
	}

	// Show OpenAI format (most common for tool calling)
	fmt.Println("=== OpenAI Chat Format (for API calls) ===")
	openaiOutput, err := poml.Convert(doc, poml.FormatOpenAIChat, poml.ConvertOptions{})
	if err != nil {
		log.Fatalf("Convert error: %v", err)
	}

	jsonBytes, _ := json.MarshalIndent(openaiOutput, "", "  ")
	fmt.Println(string(jsonBytes))

	// Show dict format to see tools array
	fmt.Println("\n=== Dict Format (full structure) ===")
	dictOutput, err := poml.Convert(doc, poml.FormatDict, poml.ConvertOptions{})
	if err != nil {
		log.Fatalf("Convert error: %v", err)
	}

	// Just show tools and first few messages
	if dictMap, ok := dictOutput.(map[string]any); ok {
		if tools, ok := dictMap["tools"]; ok {
			fmt.Println("Tools:")
			toolsJSON, _ := json.MarshalIndent(tools, "", "  ")
			fmt.Println(string(toolsJSON))
		}
	}
}

func truncateBody(s string, maxLen int) string {
	s = strings.TrimSpace(s)
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
