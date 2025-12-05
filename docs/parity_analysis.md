# POML Go SDK Parity Analysis

Comparison against [Microsoft POML](https://github.com/microsoft/poml) TypeScript and Python SDKs.

**Spec Version: 0.0.8** (Microsoft POML specification)
**SDK Version: 0.2.0** (Go SDK release)

**Status: FULL PARITY ACHIEVED**

## Element Types

### Core Elements - IMPLEMENTED

| Element | Go SDK | MS TS/Python | Notes |
|---------|--------|--------------|-------|
| `<meta>` | ✅ | ✅ | id, version, owner, charLimit, tokenLimit, priority |
| `<role>` | ✅ | ✅ | |
| `<task>` | ✅ | ✅ | |
| `<input>` | ✅ | ✅ | |
| `<document>` | ✅ | ✅ | src attribute |
| `<style>` | ✅ | ✅ | |
| `<hint>` | ✅ | ✅ | caption attribute |
| `<example>` | ✅ | ✅ | |
| `<output-format>` | ✅ | ✅ | |
| `<persona>` | ✅ | ✅ | |

### Message Types - IMPLEMENTED

| Element | Go SDK | MS TS/Python | Notes |
|---------|--------|--------------|-------|
| `<human-msg>` | ✅ | ✅ | |
| `<assistant-msg>` / `<ai-msg>` | ✅ | ✅ | |
| `<system-msg>` | ✅ | ✅ | |

### Media Elements - IMPLEMENTED

| Element | Go SDK | MS TS/Python | Notes |
|---------|--------|--------------|-------|
| `<image>` / `<img>` | ✅ | ✅ | src, alt, base64 |
| `<audio>` | ✅ | ✅ | src, base64, type |
| `<video>` | ✅ | ✅ | |
| `<figure>` (extended) | ✅ | ✅ | src, alt, syntax, width, height |

### Tool/Function Elements - IMPLEMENTED

| Element | Go SDK | MS TS/Python | Notes |
|---------|--------|--------------|-------|
| `<tool-definition>` / `<tool>` | ✅ | ✅ | name, description |
| `<tool-request>` | ✅ | ✅ | |
| `<tool-response>` | ✅ | ✅ | |
| `<tool-result>` | ✅ | ✅ | |
| `<tool-error>` | ✅ | ✅ | |

### Schema/Runtime Elements - IMPLEMENTED

| Element | Go SDK | MS TS/Python | Notes |
|---------|--------|--------------|-------|
| `<output-schema>` | ✅ | ✅ | JSON schema support |
| `<runtime>` | ✅ | ✅ | model, temperature, etc. |

### Extended Elements - IMPLEMENTED

| Element | Go SDK | MS TS/Python | Notes |
|---------|--------|--------------|-------|
| `<op>` | ✅ | ✅ | name, kind, args |
| `<object>` | ✅ | ✅ | syntax, data |
| `<text>` | ✅ | ✅ | escape attribute |
| `<data>` | ✅ | ✅ | syntax attribute |
| `<diagram>` | ✅ | ✅ | |
| `<content-part>` | ✅ | ✅ | |

### Rich Text Elements - IMPLEMENTED (Phase 2)

| Element | Go SDK | MS TS/Python | Notes |
|---------|--------|--------------|-------|
| `<b>` (bold) | ✅ | ✅ | As ContentPart |
| `<i>` (italic) | ✅ | ✅ | As ContentPart |
| `<u>` (underline) | ✅ | ✅ | As ContentPart |
| `<s>` (strikethrough) | ✅ | ✅ | As ContentPart |
| `<code>` | ✅ | ✅ | CodeBlock with lang, inline |
| `<h>` (header) | ✅ | ✅ | Header with level |
| `<p>` (paragraph) | ✅ | ✅ | Paragraph struct |
| `<span>` (inline) | ✅ | ✅ | As ContentPart |
| `<list>` / `<item>` | ✅ | ✅ | List with style, ListItem |
| `<br>` (newline) | ✅ | ✅ | Newline with Count |
| `<section>` | ✅ | ✅ | Section with title |

### Data Components - IMPLEMENTED (Phase 3)

| Element | Go SDK | MS TS/Python | Notes |
|---------|--------|--------------|-------|
| `<table>` | ✅ | ✅ | src, parser, maxRecords |
| `<folder>` | ✅ | ✅ | src, filter, maxDepth, showContent |
| `<webpage>` | ✅ | ✅ | url, selector, extractText |
| `<conversation>` | ✅ | ✅ | Multi-turn display |

### Template Elements - IMPLEMENTED (Phase 4)

| Element | Go SDK | MS TS/Python | Notes |
|---------|--------|--------------|-------|
| `<let>` | ✅ | ✅ | name, value, src attributes |
| `<include>` | ✅ | ✅ | src, if, for attributes |

## Output Formats

| Format | Go SDK | MS TS/Python | Notes |
|--------|--------|--------------|-------|
| `message_dict` | ✅ | ✅ | |
| `dict` | ✅ | ✅ | |
| `openai_chat` | ✅ | ✅ | |
| `langchain` | ✅ | ✅ | |
| `pydantic` | ✅ | ✅ | |
| `scene` / `scenejson` | ✅ | ❌ | Go SDK extra |

## Features

### All Features - IMPLEMENTED

| Feature | Go SDK | MS TS/Python | Notes |
|---------|--------|--------------|-------|
| Parse/Validate/Encode | ✅ | ✅ | Lossless round-trip |
| Extended mode (strict/lenient/off) | ✅ | ✅ | |
| MIME allowlists | ✅ | ✅ | Custom + defaults |
| Op-kind allowlists | ✅ | ✅ | Custom + defaults |
| Path containment (BaseDir) | ✅ | ✅ | Symlink/UNC escape protection |
| Size limits (MaxImageBytes) | ✅ | ✅ | |
| Tracing (OpenTelemetry) | ✅ | ✅ | Seeded deterministic spans |
| Tool call validation | ✅ | ✅ | References checked |
| **Template Engine** | ✅ | ✅ | `{{ }}` variables, expressions |
| **Stylesheet** | ✅ | ✅ | CSS-like rules in `poml/stylesheet/` |
| **File includes** | ✅ | ✅ | `<include>` parsing |
| **Table component** | ✅ | ✅ | `<table>` with src, parser |
| **Folder component** | ✅ | ✅ | `<folder>` directory trees |
| **Webpage component** | ✅ | ✅ | `<webpage>` with url, selector |
| **Conversation component** | ✅ | ✅ | `<conversation>` multi-turn |
| **Token limits** | ✅ | ✅ | charLimit, tokenLimit, priority |
| **Version constraints** | ✅ | ✅ | minVersion/maxVersion via ValidateOptions |
| **Component filtering** | ✅ | ✅ | `+component,-component` syntax |

### Microsoft-Only (Integrations)

| Feature | Go SDK | MS TS/Python | Notes |
|---------|--------|--------------|-------|
| VS Code extension | ❌ | ✅ | Syntax highlighting, previews |
| MLflow integration | ❌ | ✅ | `log_poml_call()` |
| AgentOps integration | ❌ | ✅ | `log_poml_call()` |
| Weave integration | ❌ | ✅ | `log_poml_call()` |

## Implementation Details

### New Packages (v0.2.0)

| Package | Description |
|---------|-------------|
| `poml/template` | Expression parser, interpolation, variable context |
| `poml/stylesheet` | CSS-like rule parsing and selector matching |
| `poml/token` | Token counting, limit enforcement, priority ranking |

### New Files

```
poml/
├── version_constraint.go      # Version parsing (semver) and validation
├── version_constraint_test.go
├── component_filter.go        # Component enable/disable (+x,-y syntax)
├── component_filter_test.go
├── template/
│   ├── context.go            # Variable context with parent scoping
│   ├── context_test.go
│   ├── expr.go               # Expression parser (recursive descent)
│   ├── expr_test.go
│   ├── funcs.go              # Built-in functions (len, upper, lower, etc.)
│   ├── interpolate.go        # {{ }} detection and substitution
│   └── expand.go             # Template expansion utilities
├── stylesheet/
│   ├── stylesheet.go         # Rule parsing, selector matching, Apply
│   └── stylesheet_test.go
└── token/
    ├── count.go              # CountChars, EstimateTokens, Truncate
    └── count_test.go
```

### Updated Structs

**Meta** - Added fields:
```go
CharLimit  int64  `xml:"charLimit,attr,omitempty"`
TokenLimit int64  `xml:"tokenLimit,attr,omitempty"`
Priority   int    `xml:"priority,attr,omitempty"`
```

**ValidateOptions** - Added:
```go
EnforceVersions bool  // Validate minVersion/maxVersion constraints
```

**New Element Types**:
- `ElementHeader`, `ElementParagraph`, `ElementSection`
- `ElementList`, `ElementCode`, `ElementNewline`
- `ElementTable`, `ElementFolder`, `ElementWebpage`, `ElementConversation`
- `ElementLet`, `ElementInclude`, `ElementIf`, `ElementFor`

## Summary

### Go SDK v0.2.0 Capabilities

- **Complete Microsoft POML v0.0.8 spec implementation**
- Full parsing/encoding for all element types
- Template engine with expression evaluation
- Stylesheet system with CSS-like selectors
- Token management with priority-based truncation
- Version constraint enforcement
- Component filtering

### Go SDK Extras (Not in Microsoft SDK)

- Scene/diagram rendering (graphviz, deck.gl)
- MCP server integration
- Plugin architecture

### Not Implemented (Integration-specific)

These are third-party integrations specific to the Python ecosystem:
- VS Code extension
- MLflow/AgentOps/Weave integrations

---

## Verification

All implementations verified via:
1. Unit tests for each new package
2. Parser tests for new element types
3. Encoder tests for round-trip
4. Compliance fixtures for output format parity
