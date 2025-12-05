# POML Go SDK - Microsoft Parity Implementation Plan

Complete implementation plan for adding all missing Microsoft POML features to achieve full parity with the TypeScript and Python SDKs.

## Overview

Based on analysis of [Microsoft POML v0.0.8](https://github.com/microsoft/poml), the Go SDK needs the following feature additions:

| Category | Features | Complexity |
|----------|----------|------------|
| Template Engine | `{{ }}`, `<let>`, `for`, `if`, `<include>` | High |
| Rich Text | `<b>`, `<i>`, `<u>`, `<s>`, `<code>`, `<h>`, `<p>`, `<list>`, `<br>` | Medium |
| Data Components | `<table>`, `<folder>`, `<webpage>`, `<conversation>` | Medium |
| Stylesheet | `<stylesheet>` CSS-like system | Medium |
| Token Management | `charLimit`, `tokenLimit`, `priority` | Low |
| Version Constraints | `minVersion`, `maxVersion` enforcement | Low |
| Component Filtering | `components` enable/disable | Low |

---

## Phase 1: Foundation & Low-Hanging Fruit

### 1.1 Meta Enhancements

**Files:** `poml/parser.go`

Add to Meta struct:
```go
type Meta struct {
    ID          string `xml:"id"`
    Version     string `xml:"version"`
    Owner       string `xml:"owner"`
    MinVersion  string `xml:"minVersion,attr,omitempty"`  // exists
    MaxVersion  string `xml:"maxVersion,attr,omitempty"`  // exists
    Components  string `xml:"components,attr,omitempty"`  // exists
    Stylesheet  string `xml:"stylesheet,attr,omitempty"`  // exists
    // NEW:
    CharLimit   int64  `xml:"charLimit,attr,omitempty"`
    TokenLimit  int64  `xml:"tokenLimit,attr,omitempty"`
    Priority    int    `xml:"priority,attr,omitempty"`
}
```

**Tasks:**
- [ ] Add CharLimit, TokenLimit, Priority fields to Meta
- [ ] Update parser to decode new attributes
- [ ] Update encoder to emit new attributes
- [ ] Update convertDict to include in runtime output
- [ ] Add validation for numeric values
- [ ] Add tests for meta attribute parsing

### 1.2 Version Constraint Enforcement

**Files:** `poml/version_constraint.go` (new)

```go
type Version struct {
    Major, Minor, Patch int
    Prerelease string
}

func ParseVersion(s string) (Version, error)
func (v Version) Compare(other Version) int
func CheckVersionConstraint(current, min, max string) error
```

**Tasks:**
- [ ] Create version_constraint.go
- [ ] Implement semantic version parsing
- [ ] Add ValidateOptions.EnforceVersions flag
- [ ] Integrate into ValidateWithOptions
- [ ] Add version constraint test fixtures
- [ ] Add golden error messages for version failures

### 1.3 Component Filtering

**Files:** `poml/component_filter.go` (new)

```go
func ParseComponents(spec string) (enabled, disabled []string)
func FilterDocument(doc *Document, components string) *Document
func IsElementEnabled(el ElementType, enabled, disabled []string) bool
```

**Tasks:**
- [ ] Create component_filter.go
- [ ] Parse `+component,-component` syntax
- [ ] Filter elements during conversion (not parsing)
- [ ] Add ValidateOptions.EnforceComponents flag
- [ ] Add component filtering tests

---

## Phase 2: Rich Text Elements

### 2.1 New Element Types

**Files:** `poml/parser.go`

Add ElementType constants:
```go
const (
    // ... existing ...
    ElementBold          ElementType = "bold"
    ElementItalic        ElementType = "italic"
    ElementUnderline     ElementType = "underline"
    ElementStrike        ElementType = "strike"
    ElementCode          ElementType = "code"
    ElementHeader        ElementType = "header"
    ElementParagraph     ElementType = "paragraph"
    ElementSpan          ElementType = "span"
    ElementList          ElementType = "list"
    ElementListItem      ElementType = "list_item"
    ElementNewline       ElementType = "newline"
    ElementSection       ElementType = "section"
)
```

### 2.2 Rich Text Data Types

```go
type RichText struct {
    Tag      string            // b, i, u, s, code, h, p, span
    Content  string
    Children []RichText
    Attrs    map[string]string // lang for code, level for h
}

type List struct {
    Style string     // star, dash, plus, decimal, latin
    Items []ListItem
}

type ListItem struct {
    Content  string
    Children []RichText
}
```

**Tasks:**
- [ ] Add RichText, List, ListItem structs
- [ ] Add Document.RichTexts, Document.Lists slices
- [ ] Parse `<b>`, `<i>`, `<u>`, `<s>` as RichText
- [ ] Parse `<code lang="...">` with attribute
- [ ] Parse `<h level="1-6">` with attribute
- [ ] Parse `<list style="...">` with items
- [ ] Parse `<br newLineCount="N">`
- [ ] Update converters to emit rich text
- [ ] Add rich text formatting tests

---

## Phase 3: Data Components

### 3.1 Table Component

**Files:** `poml/parser.go`, `poml/converter.go`

```go
type Table struct {
    ID             string
    Src            string            // external file path
    Parser         string            // csv, xlsx, json
    Records        []map[string]any  // inline data
    Columns        []TableColumn
    SelectedCols   []string
    SelectedRows   []int
    MaxRecords     int
}

type TableColumn struct {
    Field       string
    Header      string
    Description string
}
```

**Tasks:**
- [ ] Add Table struct and ElementTable type
- [ ] Parse `<table>` with src, parser, records, columns attributes
- [ ] Handle inline JSON records
- [ ] Handle external file loading (CSV, XLSX)
- [ ] Implement selectedColumns, selectedRecords filtering
- [ ] Add table conversion to all formats
- [ ] Add table test fixtures

### 3.2 Folder Component

```go
type Folder struct {
    ID          string
    Src         string   // directory path
    Filter      string   // glob pattern
    MaxDepth    int
    ShowContent bool
    Files       []FolderEntry
}

type FolderEntry struct {
    Path    string
    IsDir   bool
    Content string // if ShowContent=true
}
```

**Tasks:**
- [ ] Add Folder struct and ElementFolder type
- [ ] Parse `<folder>` with src, filter, maxDepth, showContent
- [ ] Implement directory traversal with BaseDir containment
- [ ] Implement glob filtering
- [ ] Add folder conversion to all formats
- [ ] Add folder security tests (escape attempts)

### 3.3 Webpage Component

```go
type Webpage struct {
    ID          string
    URL         string
    Selector    string  // CSS selector for extraction
    ExtractText bool
    Title       string
    Content     string
}
```

**Tasks:**
- [ ] Add Webpage struct and ElementWebpage type
- [ ] Parse `<webpage>` with url, selector, extractText
- [ ] NOTE: Actual fetching is conversion-time, not parse-time
- [ ] Add ConvertOptions.AllowWebFetch flag
- [ ] Implement webpage conversion (fetch + extract)
- [ ] Add webpage test fixtures (mock server)

### 3.4 Conversation Component

```go
type Conversation struct {
    ID               string
    Messages         []ConversationMessage
    SelectedMessages []int
}

type ConversationMessage struct {
    Speaker string // human, ai, system
    Content string
}
```

**Tasks:**
- [ ] Add Conversation struct and ElementConversation type
- [ ] Parse `<conversation>` with nested messages
- [ ] Implement selectedMessages filtering
- [ ] Add conversation conversion to all formats
- [ ] Add conversation test fixtures

---

## Phase 4: Template Engine

### 4.1 Expression Parser

**Files:** `poml/template/expr.go` (new package)

```go
type Expression interface {
    Eval(ctx *Context) (any, error)
}

type Context struct {
    Variables map[string]any
    Parent    *Context
}

func ParseExpression(s string) (Expression, error)
func (c *Context) Get(name string) (any, bool)
func (c *Context) Set(name, value any)
func (c *Context) Child() *Context
```

Supported expressions:
- Variable reference: `x`, `user.name`
- Literals: `"string"`, `123`, `true`, `null`
- Arithmetic: `a + b`, `a - b`, `a * b`, `a / b`
- Comparison: `a == b`, `a != b`, `a < b`, `a <= b`
- Logical: `a && b`, `a || b`, `!a`
- Ternary: `condition ? then : else`
- Array access: `arr[0]`, `obj["key"]`
- Function calls: `len(arr)`, `upper(s)`

**Tasks:**
- [ ] Create poml/template package
- [ ] Implement expression lexer
- [ ] Implement expression parser (recursive descent)
- [ ] Implement expression evaluator
- [ ] Add built-in functions (len, upper, lower, trim, etc.)
- [ ] Add expression test suite

### 4.2 Variable Interpolation

**Files:** `poml/template/interpolate.go`

```go
func Interpolate(text string, ctx *Context) (string, error)
func FindExpressions(text string) []ExpressionSpan

type ExpressionSpan struct {
    Start, End int
    Expression string
}
```

**Tasks:**
- [ ] Implement `{{ }}` detection in text
- [ ] Parse and evaluate embedded expressions
- [ ] Handle nested braces and escaping
- [ ] Add interpolation tests

### 4.3 Let Bindings

**Files:** `poml/parser.go`, `poml/template/let.go`

```go
type LetBinding struct {
    Name  string
    Value string // expression or literal
    Src   string // file import
    Body  string // inline JSON/text
}
```

Parse modes:
1. `<let name="x">literal</let>` - literal string
2. `<let name="x" value="expr" />` - expression
3. `<let name="x" src="file.json" />` - file import
4. `<let name="x">{ "json": true }</let>` - inline JSON

**Tasks:**
- [ ] Add LetBinding struct and ElementLet type
- [ ] Parse all four `<let>` syntaxes
- [ ] Implement file loading for src attribute
- [ ] Implement JSON parsing for inline bodies
- [ ] Add let binding to Context before template expansion
- [ ] Add let binding tests

### 4.4 Conditionals

**Files:** `poml/parser.go`, `poml/template/conditional.go`

```go
type Conditional struct {
    Condition string   // expression
    Then      []Element
    Else      []Element
}
```

Support `if` attribute on any element:
```xml
<p if="isVisible">Visible content</p>
<div if="count > 0">Has items</div>
```

**Tasks:**
- [ ] Parse `if` attribute on all elements
- [ ] Evaluate condition during conversion
- [ ] Skip element if condition is false
- [ ] Handle nested conditionals
- [ ] Add conditional tests

### 4.5 Loops

**Files:** `poml/parser.go`, `poml/template/loop.go`

```go
type Loop struct {
    Variable string   // iteration variable name
    ListExpr string   // expression yielding array
    Body     []Element
}

type LoopContext struct {
    Index  int
    Length int
    First  bool
    Last   bool
    Value  any
}
```

Support `for` attribute:
```xml
<item for="x in items">{{ x }}</item>
<p for="i, v in enumerate(list)">{{ i }}: {{ v }}</p>
```

**Tasks:**
- [ ] Parse `for` attribute syntax
- [ ] Evaluate list expression
- [ ] Create loop context with index/length/first/last
- [ ] Clone and expand body for each iteration
- [ ] Add loop variable to child context
- [ ] Add loop tests

### 4.6 Includes

**Files:** `poml/parser.go`, `poml/template/include.go`

```go
type Include struct {
    Src       string
    Condition string // optional if attribute
    Loop      string // optional for attribute
}

func LoadInclude(src string, baseDir string) (*Document, error)
func MergeInclude(parent, child *Document, position int) *Document
```

**Tasks:**
- [ ] Add Include struct and ElementInclude type
- [ ] Parse `<include src="...">`
- [ ] Resolve src relative to BaseDir
- [ ] Parse included file recursively
- [ ] Merge included elements at position
- [ ] Handle circular include detection
- [ ] Support if/for on includes
- [ ] Add include tests

### 4.7 Template Expansion Pipeline

**Files:** `poml/template/expand.go`

```go
func Expand(doc *Document, vars map[string]any, opts ExpandOptions) (*Document, error)

type ExpandOptions struct {
    BaseDir       string
    MaxDepth      int    // include recursion limit
    StrictMode    bool   // error on undefined variables
    AllowWebFetch bool   // for webpage components
}
```

Expansion order:
1. Process `<let>` bindings → populate Context
2. Load `<include>` files → merge into document
3. Evaluate `for` loops → expand elements
4. Evaluate `if` conditions → filter elements
5. Interpolate `{{ }}` expressions → substitute text

**Tasks:**
- [ ] Implement expansion pipeline
- [ ] Handle expansion order correctly
- [ ] Propagate context through nested scopes
- [ ] Add expansion integration tests

---

## Phase 5: Stylesheet System

### 5.1 Stylesheet Parser

**Files:** `poml/stylesheet/parse.go` (new package)

```go
type Stylesheet struct {
    Rules []Rule
}

type Rule struct {
    Selector string            // tag name or .className
    Props    map[string]string // attribute overrides
}

func ParseStylesheet(json string) (*Stylesheet, error)
```

Stylesheet format (JSON):
```json
{
  "hint": { "syntax": "markdown" },
  ".important": { "priority": "1" },
  "code": { "lang": "python" }
}
```

**Tasks:**
- [ ] Create poml/stylesheet package
- [ ] Parse stylesheet JSON
- [ ] Match selectors to elements
- [ ] Apply property overrides
- [ ] Add stylesheet tests

### 5.2 Stylesheet Application

**Files:** `poml/stylesheet/apply.go`

```go
func Apply(doc *Document, sheet *Stylesheet) *Document
func MatchSelector(el Element, selector string) bool
func ApplyProps(el *Element, props map[string]string)
```

**Tasks:**
- [ ] Implement selector matching (tag, .class)
- [ ] Apply properties as element attributes
- [ ] Handle className attribute on elements
- [ ] Integration with conversion pipeline
- [ ] Add application tests

---

## Phase 6: Token Management

### 6.1 Token Counting

**Files:** `poml/token/count.go` (new package)

```go
func CountChars(doc *Document) int64
func CountTokens(doc *Document, model string) (int64, error)
func EstimateTokens(text string) int64  // approximate without tokenizer

type TokenCounter interface {
    Count(text string) (int64, error)
}
```

**Tasks:**
- [ ] Create poml/token package
- [ ] Implement character counting
- [ ] Implement approximate token estimation (chars/4)
- [ ] Add tiktoken integration (optional, behind build tag)
- [ ] Add token counting tests

### 6.2 Limit Enforcement

**Files:** `poml/token/limit.go`

```go
func EnforceLimits(doc *Document, charLimit, tokenLimit int64) (*Document, error)
func PrioritizeElements(doc *Document) []*Element
func TruncateToLimit(elements []*Element, limit int64) []*Element
```

**Tasks:**
- [ ] Sort elements by priority attribute
- [ ] Truncate low-priority elements to fit limit
- [ ] Add ValidateOptions.EnforceLimits flag
- [ ] Integrate with conversion pipeline
- [ ] Add limit enforcement tests

---

## Phase 7: Integration & Testing

### 7.1 Parity Test Suite

**Files:** `poml/parity_microsoft_test.go`

```go
func TestMicrosoftParityTemplate(t *testing.T)
func TestMicrosoftParityRichText(t *testing.T)
func TestMicrosoftParityTable(t *testing.T)
func TestMicrosoftParityFolder(t *testing.T)
func TestMicrosoftParityStylesheet(t *testing.T)
func TestMicrosoftParityTokenLimit(t *testing.T)
```

**Tasks:**
- [ ] Create Microsoft parity test fixtures
- [ ] Compare Go SDK output with Python SDK
- [ ] Ensure format consistency (message_dict, openai_chat, langchain)
- [ ] Add negative test cases
- [ ] Gate CI on parity tests

### 7.2 Documentation

**Tasks:**
- [ ] Update README with new features
- [ ] Add template engine documentation
- [ ] Add data component documentation
- [ ] Add migration guide from core to full
- [ ] Update parity_analysis.md

### 7.3 Performance

**Tasks:**
- [ ] Benchmark template expansion
- [ ] Benchmark large table conversion
- [ ] Benchmark deep folder traversal
- [ ] Add performance regression tests

---

## File Summary

### New Files
```
poml/
├── version_constraint.go      # Version parsing and validation
├── component_filter.go        # Component enable/disable
├── template/
│   ├── expr.go               # Expression parser/evaluator
│   ├── interpolate.go        # {{ }} substitution
│   ├── let.go                # Let binding processing
│   ├── conditional.go        # If condition handling
│   ├── loop.go               # For loop expansion
│   ├── include.go            # File include loading
│   └── expand.go             # Main expansion pipeline
├── stylesheet/
│   ├── parse.go              # Stylesheet JSON parser
│   └── apply.go              # Selector matching & application
├── token/
│   ├── count.go              # Character/token counting
│   └── limit.go              # Limit enforcement
└── parity_microsoft_test.go  # Cross-SDK parity tests
```

### Modified Files
```
poml/
├── parser.go                 # New ElementTypes, structs, parsing
├── converter.go              # New element handlers, template expansion
├── builder.go                # New builder methods
├── validation.go             # New validation rules
└── *_test.go                 # New test cases
```

---

## Estimated Effort

| Phase | Description | Effort |
|-------|-------------|--------|
| 1 | Foundation (meta, version, components) | 2-3 days |
| 2 | Rich Text Elements | 3-4 days |
| 3 | Data Components | 5-7 days |
| 4 | Template Engine | 7-10 days |
| 5 | Stylesheet System | 3-4 days |
| 6 | Token Management | 2-3 days |
| 7 | Integration & Testing | 3-5 days |
| **Total** | | **25-36 days** |

---

## Dependencies

### Required
- None (pure Go implementation)

### Optional
- `tiktoken-go` - Accurate token counting (behind build tag)
- `goquery` - HTML parsing for webpage component
- `excelize` - XLSX parsing for table component

---

## Backwards Compatibility

All new features are **opt-in**:
- Template expressions only processed when `ExpandOptions` provided
- New elements parsed as `ElementUnknown` in `ExtendedOff` mode
- Existing tests continue to pass unchanged
- No breaking changes to existing API

---

## Success Criteria

1. All Microsoft POML v0.0.8 examples parse without error
2. Template expansion produces identical output to Python SDK
3. All new elements round-trip losslessly
4. ExtendedOff mode rejects new elements appropriately
5. Performance within 2x of Python SDK for equivalent operations
6. 90%+ test coverage on new code
7. CI gates on parity test suite
