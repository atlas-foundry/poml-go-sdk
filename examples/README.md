# POML Go SDK Examples

This directory contains runnable examples demonstrating the POML Go SDK's capabilities.

## Examples

| Directory | Description |
|-----------|-------------|
| [01_basic_conversion](./01_basic_conversion) | Parse, validate, and convert POML to different formats |
| [02_format_comparison](./02_format_comparison) | Side-by-side comparison of all output formats |
| [03_tool_calls](./03_tool_calls) | Tool definitions, tool calls, and tool responses |
| [04_multimedia](./04_multimedia) | Images, audio, and video handling |
| [05_templates](./05_templates) | Variables, conditionals, loops, and includes |
| [06_integrations](./06_integrations) | MLflow, AgentOps, Weave, and OpenTelemetry integration |

## Running Examples

Each example is a standalone Go program. From the repository root:

```bash
# Run any example
go run ./examples/01_basic_conversion

# Or from the example directory
cd examples/01_basic_conversion
go run main.go
```

## Quick Start

### Parse and Convert

```go
import "github.com/atlas-foundry/poml-go-sdk/poml"

// Parse POML
doc, err := poml.ParseString(`<poml>
  <meta><id>demo</id><version>1.0</version><owner>you</owner></meta>
  <role>You are helpful.</role>
  <task>Answer questions.</task>
  <human-msg>Hello!</human-msg>
</poml>`)

// Validate
err = doc.Validate()

// Convert to OpenAI format
output, err := poml.Convert(doc, poml.FormatOpenAIChat, poml.ConvertOptions{})
```

### Available Formats

| Format | Constant | Use Case |
|--------|----------|----------|
| Message Dict | `poml.FormatMessageDict` | Simple `[{speaker, content}]` tuples |
| Dict | `poml.FormatDict` | Full structure with messages, tools, schema |
| OpenAI Chat | `poml.FormatOpenAIChat` | Direct OpenAI API integration |
| LangChain | `poml.FormatLangChain` | LangChain/LangGraph applications |
| Pydantic | `poml.FormatPydantic` | Python SDK compatibility |
| Scene | `poml.FormatScene` | Diagram visualization |
| SceneJSON | `poml.FormatSceneJSON` | Serialized scene output |

## Example Outputs

### Message Dict Format
```json
[
  {"speaker": "system", "content": "You are helpful."},
  {"speaker": "human", "content": "Hello!"}
]
```

### OpenAI Chat Format
```json
{
  "messages": [
    {"role": "system", "content": "You are helpful.\n\n# Task\nAnswer questions."},
    {"role": "user", "content": "Hello!"}
  ]
}
```

### LangChain Format
```json
{
  "messages": [
    {"type": "system", "data": {"content": "You are helpful.\n\n# Task\nAnswer questions."}},
    {"type": "human", "data": {"content": "Hello!"}}
  ]
}
```

## Features Demonstrated

### Tool Definitions
```xml
<tool-definition name="search" description="Search the web">
  <schema>{"type": "object", "properties": {"query": {"type": "string"}}}</schema>
</tool-definition>
```

### Tool Calls
```xml
<assistant-msg>
  <tool-call id="call_1" name="search">{"query": "weather"}</tool-call>
</assistant-msg>
<tool-response call-id="call_1">{"result": "Sunny, 72°F"}</tool-response>
```

### Templates
```xml
<let name="language" value="Go"/>
<role>You are a {{language}} expert.</role>

<for each="topic" in="['concurrency', 'interfaces']">
  <hint>Cover {{topic}}</hint>
</for>
```

### Multimedia
```xml
<image src="diagram.png" alt="Architecture diagram"/>
<audio src="recording.mp3" alt="Voice memo"/>
<video src="demo.mp4" alt="Demo video"/>
```

## Next Steps

- See [docs/converters.md](../docs/converters.md) for detailed format documentation
- Check [integrations/README.md](../integrations/README.md) for observability setup
- Run `go test ./poml/...` to see comprehensive test coverage
