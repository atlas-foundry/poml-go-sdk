# POML Converter Formats Guide

This guide provides detailed documentation on each output format available in the POML Go SDK.

## Overview

The POML SDK converts parsed documents into various output formats suitable for different LLM APIs and frameworks. Each format has specific characteristics and use cases.

```go
output, err := poml.Convert(doc, format, poml.ConvertOptions{})
```

## Format Comparison

| Format | Output Type | Messages | Tools | Runtime | Schema | Media |
|--------|-------------|----------|-------|---------|--------|-------|
| `message_dict` | `[]messageDict` | ✅ | ❌ | ❌ | ❌ | Inline |
| `dict` | `map[string]any` | ✅ | ✅ | ✅ | ✅ | Inline |
| `openai_chat` | `map[string]any` | ✅ | ✅ | ✅ | ✅ | `image_url` |
| `langchain` | `map[string]any` | ✅ | ✅ | ✅ | ✅ | Nested |
| `pydantic` | `map[string]any` | ✅ | ✅ | ✅ | ✅ | Separate |
| `scene` | `Scene` | N/A | N/A | N/A | N/A | N/A |
| `scenejson` | `string` | N/A | N/A | N/A | N/A | N/A |

## Format Details

### Message Dict (`message_dict`)

The simplest format—a list of speaker/content tuples.

**Use Case:** Simple message passing, custom integrations, debugging.

**Output Structure:**
```json
[
  {"speaker": "system", "content": "You are a helpful assistant."},
  {"speaker": "human", "content": "Hello!"},
  {"speaker": "ai", "content": "Hi there!"}
]
```

**Go Usage:**
```go
output, _ := poml.Convert(doc, poml.FormatMessageDict, poml.ConvertOptions{})
messages := output.([]poml.MessageDict)
for _, msg := range messages {
    fmt.Printf("%s: %s\n", msg.Speaker, msg.Content)
}
```

**Speaker Mapping:**
| POML Element | Speaker |
|--------------|---------|
| `<role>`, `<system-msg>` | `"system"` |
| `<human-msg>` | `"human"` |
| `<assistant-msg>` | `"ai"` |
| `<tool-response>` | `"tool"` |

---

### Dict (`dict`)

Full-featured dictionary with all POML components.

**Use Case:** Access to complete document structure including tools, runtime, and schema.

**Output Structure:**
```json
{
  "messages": [
    {"speaker": "system", "content": "..."},
    {"speaker": "human", "content": "..."}
  ],
  "tools": [
    {
      "name": "search",
      "description": "Search the web",
      "schema": {"type": "object", "properties": {...}}
    }
  ],
  "runtime": {
    "temperature": 0.7,
    "max_tokens": 1024
  },
  "schema": {
    "type": "object",
    "properties": {...}
  }
}
```

**Go Usage:**
```go
output, _ := poml.Convert(doc, poml.FormatDict, poml.ConvertOptions{})
dict := output.(map[string]any)
messages := dict["messages"]
tools := dict["tools"]
```

---

### OpenAI Chat (`openai_chat`)

Format matching the OpenAI Chat Completions API.

**Use Case:** Direct integration with OpenAI API, GPT models.

**Output Structure:**
```json
{
  "messages": [
    {
      "role": "system",
      "content": "You are helpful.\n\n# Task\nAnswer questions."
    },
    {
      "role": "user",
      "content": [
        {"type": "text", "text": "What's in this image?"},
        {"type": "image_url", "image_url": {"url": "data:image/png;base64,..."}}
      ]
    },
    {
      "role": "assistant",
      "content": null,
      "tool_calls": [
        {
          "id": "call_123",
          "type": "function",
          "function": {"name": "search", "arguments": "{\"query\": \"weather\"}"}
        }
      ]
    },
    {
      "role": "tool",
      "tool_call_id": "call_123",
      "content": "{\"result\": \"Sunny\"}"
    }
  ],
  "tools": [
    {
      "type": "function",
      "function": {
        "name": "search",
        "description": "Search the web",
        "parameters": {"type": "object", "properties": {...}}
      }
    }
  ],
  "temperature": 0.7,
  "max_tokens": 1024,
  "response_format": {
    "type": "json_schema",
    "json_schema": {"name": "response", "schema": {...}}
  }
}
```

**Go Usage:**
```go
output, _ := poml.Convert(doc, poml.FormatOpenAIChat, poml.ConvertOptions{
    BaseDir: ".",  // For resolving image paths
})

// Ready to send to OpenAI API
jsonBytes, _ := json.Marshal(output)
```

**Key Features:**
- Role mapping: `system`, `user`, `assistant`, `tool`
- Images as `image_url` content parts
- Tool calls with `function` wrapper
- Runtime params at top level
- Schema wrapped in `json_schema`

---

### LangChain (`langchain`)

Format compatible with LangChain message types.

**Use Case:** LangChain applications, LangGraph agents.

**Output Structure:**
```json
{
  "messages": [
    {
      "type": "system",
      "data": {"content": "You are helpful."}
    },
    {
      "type": "human",
      "data": {"content": "Hello!"}
    },
    {
      "type": "ai",
      "data": {
        "content": "",
        "tool_calls": [
          {"name": "search", "args": {"query": "weather"}, "id": "call_1"}
        ]
      }
    },
    {
      "type": "tool",
      "data": {
        "content": "{\"result\": \"Sunny\"}",
        "tool_call_id": "call_1"
      }
    }
  ],
  "tools": [...],
  "runtime": {...}
}
```

**Go Usage:**
```go
output, _ := poml.Convert(doc, poml.FormatLangChain, poml.ConvertOptions{})
lcMessages := output.(map[string]any)["messages"]
```

**Key Features:**
- Message type in `type` field
- Content nested under `data`
- Tool calls with `args` (not `arguments`)
- Compatible with `langchain-core` message types

---

### Pydantic (`pydantic`)

Python SDK compatible format with separate media collection.

**Use Case:** Cross-SDK compatibility, Python interop.

**Output Structure:**
```json
{
  "messages": [...],
  "tools": [...],
  "runtime": {...},
  "schema": {...},
  "media": [
    {"type": "image", "src": "...", "alt": "..."},
    {"type": "audio", "src": "...", "alt": "..."}
  ]
}
```

**Go Usage:**
```go
output, _ := poml.Convert(doc, poml.FormatPydantic, poml.ConvertOptions{})
pydantic := output.(map[string]any)
media := pydantic["media"]  // Collected media elements
```

---

### Scene (`scene`) and SceneJSON (`scenejson`)

Diagram visualization formats for `<diagram>` elements.

**Use Case:** Rendering flowcharts, architecture diagrams.

**Output:** `Scene` struct or JSON string representation.

---

## Convert Options

```go
type ConvertOptions struct {
    // Extended mode for non-standard elements
    Extended ExtendedMode  // ExtendedOff, ExtendedOn, ExtendedLenient

    // Base directory for resolving relative paths
    BaseDir string

    // Template expansion
    ExpandTemplates bool  // default: true

    // Runtime variable values
    Variables map[string]string

    // Tracing options
    Trace TraceOptions
}
```

### Extended Mode

Controls handling of non-standard POML elements:

| Mode | Behavior |
|------|----------|
| `ExtendedOff` | Only standard elements, strict validation |
| `ExtendedOn` | Extended elements allowed (ops, objects, figures) |
| `ExtendedLenient` | Extended + unknown elements preserved |

```go
// Allow extended elements
output, _ := poml.Convert(doc, poml.FormatOpenAIChat, poml.ConvertOptions{
    Extended: poml.ExtendedLenient,
})
```

### Runtime Variables

Substitute values at conversion time:

```go
// POML: <let name="topic" runtime="true"/>
output, _ := poml.Convert(doc, poml.FormatOpenAIChat, poml.ConvertOptions{
    Variables: map[string]string{
        "topic": "machine learning",
    },
})
```

---

## Element Mapping

### Messages

| POML Element | message_dict | openai_chat | langchain |
|--------------|--------------|-------------|-----------|
| `<role>` | speaker: system | role: system | type: system |
| `<system-msg>` | speaker: system | role: system | type: system |
| `<human-msg>` | speaker: human | role: user | type: human |
| `<assistant-msg>` | speaker: ai | role: assistant | type: ai |
| `<tool-response>` | speaker: tool | role: tool | type: tool |

### Content Assembly

The system message is assembled from multiple elements:

```
[Role content]

# Task
[Task 1]
[Task 2]

# Hints
[Hint 1]
[Hint 2]

# Examples
[Example content]
```

### Tool Calls

```xml
<assistant-msg>
  <tool-call id="call_1" name="func">{"arg": "value"}</tool-call>
</assistant-msg>
```

**OpenAI format:**
```json
{
  "role": "assistant",
  "content": null,
  "tool_calls": [{
    "id": "call_1",
    "type": "function",
    "function": {"name": "func", "arguments": "{\"arg\": \"value\"}"}
  }]
}
```

**LangChain format:**
```json
{
  "type": "ai",
  "data": {
    "content": "",
    "tool_calls": [{"name": "func", "args": {"arg": "value"}, "id": "call_1"}]
  }
}
```

---

## Best Practices

1. **Choose the right format:**
   - OpenAI API → `openai_chat`
   - LangChain apps → `langchain`
   - Simple needs → `message_dict`
   - Full control → `dict`

2. **Handle images properly:**
   - Set `BaseDir` for relative paths
   - Images are base64-encoded automatically
   - Use data URIs for inline images

3. **Use Extended mode wisely:**
   - `ExtendedOff` for strict POML compliance
   - `ExtendedLenient` for flexible parsing

4. **Template expansion:**
   - Enabled by default
   - Use `Variables` for runtime substitution
   - Disable with `ExpandTemplates: false` for debugging
