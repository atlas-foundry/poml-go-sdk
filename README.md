# POML Go SDK

**Structured prompts for Go, LLMs, and tools**

[![CI Status](https://img.shields.io/github/actions/workflow/status/atlas-foundry/poml-go-sdk/ci.yml?branch=main&label=ci&logo=github)](https://github.com/atlas-foundry/poml-go-sdk/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/atlas-foundry/poml-go-sdk?logo=github&color=4c1)](https://github.com/atlas-foundry/poml-go-sdk/releases)
[![Go Version](https://img.shields.io/github/go-mod/go-version/atlas-foundry/poml-go-sdk?logo=go&color=00add8)](https://img.shields.io/github/go-mod/go-version/atlas-foundry/poml-go-sdk)
[![Go Reference](https://pkg.go.dev/badge/github.com/atlas-foundry/poml-go-sdk.svg)](https://pkg.go.dev/github.com/atlas-foundry/poml-go-sdk)
[![License MIT](https://img.shields.io/github/license/atlas-foundry/poml-go-sdk?color=blue)](https://github.com/atlas-foundry/poml-go-sdk/blob/main/LICENSE)

Production-ready Go library for **Prompt Orchestration Markup Language (POML)** - parse, validate, and convert structured prompts into formats ready for OpenAI, LangChain, and other LLM frameworks.

## What is POML?

POML is like HTML for AI prompts. Instead of writing unstructured text, you define prompts with clear structure:

```xml
<poml>
  <role>You are a helpful coding assistant.</role>
  <task>Help the user write clean Go code.</task>
  <human>How do I read a file in Go?</human>
</poml>
```

This SDK parses POML files and converts them to formats that LLMs understand (like OpenAI's Chat API).

## Quick Start

### Installation

```bash
go get github.com/atlas-foundry/poml-go-sdk
```

### Your First POML Program

**1. Create `hello.poml`:**

```xml
<poml>
  <role>You are a friendly assistant.</role>
  <task>Greet the user warmly.</task>
  <human>Hello!</human>
</poml>
```

**2. Create `main.go`:**

```go
package main

import (
    "encoding/json"
    "fmt"
    "log"

    "github.com/atlas-foundry/poml-go-sdk/poml"
)

func main() {
    // Parse the POML file
    doc, err := poml.ParseFile("hello.poml")
    if err != nil {
        log.Fatal(err)
    }

    // Convert to OpenAI Chat format
    result, err := poml.Convert(doc, poml.FormatOpenAIChat, poml.ConvertOptions{})
    if err != nil {
        log.Fatal(err)
    }

    // Print the result
    output, _ := json.MarshalIndent(result, "", "  ")
    fmt.Println(string(output))
}
```

**3. Run it:**

```bash
go run main.go
```

You'll see JSON output ready to send to the OpenAI API!

## Core Concepts

### The Flow

```
POML File  -->  Parse  -->  Document  -->  Convert  -->  LLM-ready JSON
```

### Key Tags

| Tag | Purpose | Example |
|-----|---------|---------|
| `<role>` | System message / AI persona | `<role>You are a helpful assistant.</role>` |
| `<task>` | Instructions for the AI | `<task>Summarize the text below.</task>` |
| `<human>` | User message | `<human>What's 2+2?</human>` |
| `<assistant>` | AI response (for few-shot examples) | `<assistant>4</assistant>` |
| `<input>` | Data/context to process | `<input>Some text to analyze...</input>` |

### Output Formats

| Format | Use Case |
|--------|----------|
| `FormatOpenAIChat` | OpenAI Chat Completions API |
| `FormatLangChain` | LangChain tool calls |
| `FormatMessageDict` | Generic JSON dictionary |
| `FormatDict` | Python-style dict (debugging) |

## Examples

### Building Documents Programmatically

```go
doc := &poml.Document{}
doc.AddRole("You are a code reviewer.")
doc.AddTask("Review the following code for bugs.")
doc.AddMessage("human", "func add(a, b int) { return a - b }")

result, _ := poml.Convert(doc, poml.FormatOpenAIChat, poml.ConvertOptions{})
```

### Working with Tools

```xml
<poml>
  <role>You can search the web.</role>
  <task>Answer questions using web search.</task>
  <tool-def name="search">
    <description>Search the web</description>
    <param name="query" type="string" required="true">Search query</param>
  </tool-def>
  <human>What's the weather in Tokyo?</human>
</poml>
```

### Multimedia Content

```xml
<poml>
  <role>You are an image analyst.</role>
  <task>Describe what you see.</task>
  <human>
    What's in this image?
    <img src="photo.jpg" alt="A photo to analyze"/>
  </human>
</poml>
```

## CLI Tool

The SDK includes a CLI for quick testing:

```bash
# Start the MCP server
go run ./cmd/poml mcp --file hello.poml --addr :7777

# Then visit in your browser:
# http://localhost:7777/inspect  - Document summary
# http://localhost:7777/ast      - Parsed structure
# http://localhost:7777/validate - Validation results
# http://localhost:7777/convert?format=openai_chat - Converted output
```

## Runnable Examples

Check out the `examples/` directory for complete working examples:

| Example | Description |
|---------|-------------|
| [01_basic_conversion](examples/01_basic_conversion) | Parse and convert to multiple formats |
| [02_format_comparison](examples/02_format_comparison) | Compare all output formats |
| [03_tool_calls](examples/03_tool_calls) | Tool definitions and responses |
| [04_multimedia](examples/04_multimedia) | Images, audio, and video |
| [05_templates](examples/05_templates) | Variables, conditionals, loops |
| [06_integrations](examples/06_integrations) | MLflow, OpenTelemetry integrations |

## Architecture

```
poml/
  parser.go      # Reads POML files into Document structs
  validator.go   # Validates documents against the spec
  renderer.go    # Converts Documents back to POML text
  converter.go   # Transforms Documents to JSON formats
  testdata/      # Example POML files and test fixtures
```

The parser is **lossless** - it preserves whitespace, comments, and unknown tags so you can round-trip POML files without data loss.

## Test Coverage

This SDK maintains **80%+ test coverage** enforced by CI. Run tests with:

```bash
go test ./...
```

## Documentation

- [POML Official Spec](https://microsoft.github.io/poml) - Language reference
- [Go Package Docs](https://pkg.go.dev/github.com/atlas-foundry/poml-go-sdk) - API reference
- [Converter Guide](docs/converters.md) - Output format details
- [Integration Guide](integrations/README.md) - Observability plugins

## Contributing

We welcome contributions from everyone, whether you're:

- New to Go
- New to POML
- New to open source

**How to contribute:**

1. Fork the repo
2. Create a branch (`git checkout -b my-feature`)
3. Make your changes
4. Run tests (`go test ./...`)
5. Submit a PR

### Commit Message Format

```
feat: add new feature
fix: fix a bug
docs: update documentation
test: add tests
chore: maintenance tasks
```

## License

MIT - see [LICENSE](LICENSE) for details.

---

**Questions?** Open an issue or start a discussion. We're here to help!
