# POML Integrations

This package provides ecosystem bridge plugins for POML observability and tooling. These integrations match the Python SDK's ecosystem bridges, providing consistent functionality across both SDKs.

## Available Integrations

| Integration | Description | Status |
|-------------|-------------|--------|
| [MLflow](./mlflow) | Experiment tracking and model management | Production |
| [AgentOps](./agentops) | AI agent observability and monitoring | Production |
| [Weave](./weave) | W&B ML artifact versioning and lineage | Production |
| [LSP](./lsp) | Language Server Protocol for IDE integration | Production |

## Quick Start

All observability integrations implement the `POMLLogger` interface:

```go
type POMLLogger interface {
    LogCall(ctx context.Context, call POMLCall) error
    Flush(ctx context.Context) error
    Close() error
}
```

### MLflow

```go
import "github.com/atlas-foundry/poml-go-sdk/integrations/mlflow"

logger, err := mlflow.New(mlflow.Config{
    TrackingURI:    "http://localhost:5000",
    ExperimentName: "poml-prompts",
})
if err != nil {
    log.Fatal(err)
}
defer logger.Close()

// Log a POML operation
err = logger.LogCall(ctx, integrations.POMLCall{
    Operation:  "convert",
    Format:     "openai_chat",
    DocumentID: "my-prompt",
    Duration:   50 * time.Millisecond,
    Timestamp:  time.Now(),
})
```

### AgentOps

```go
import "github.com/atlas-foundry/poml-go-sdk/integrations/agentops"

logger, err := agentops.New(agentops.Config{
    APIKey:    "your-api-key",
    ProjectID: "your-project",
    AgentID:   "poml-agent",
})
if err != nil {
    log.Fatal(err)
}
defer logger.Close()

// A session is automatically created
fmt.Println("Session ID:", logger.SessionID())

err = logger.LogCall(ctx, integrations.POMLCall{
    Operation: "convert",
    Format:    "openai_chat",
    Timestamp: time.Now(),
})
```

### Weave (W&B)

```go
import "github.com/atlas-foundry/poml-go-sdk/integrations/weave"

logger, err := weave.New(weave.Config{
    APIKey:  "your-wandb-api-key",
    Project: "my-project",
    Entity:  "my-team",
})
if err != nil {
    log.Fatal(err)
}
defer logger.Close()

err = logger.LogCall(ctx, integrations.POMLCall{
    Operation:  "convert",
    Format:     "langchain",
    DocumentID: "prompt-v1",
    Duration:   100 * time.Millisecond,
    Timestamp:  time.Now(),
})
```

### Language Server Protocol (LSP)

```go
import "github.com/atlas-foundry/poml-go-sdk/integrations/lsp"

server := lsp.New()

// Serve over stdio (for VS Code, Neovim, etc.)
err := server.ServeStdio(ctx)

// Or serve over TCP
err := server.ServeTCP(ctx, ":6060")
```

## Common Configuration

All observability integrations share common configuration options:

```go
type Config struct {
    // Enabled controls whether the integration is active
    Enabled bool

    // MaxInputSize is the maximum input size to log (bytes). 0 = no limit
    MaxInputSize int

    // MaxOutputSize is the maximum output size to log (bytes). 0 = no limit
    MaxOutputSize int

    // AsyncBatchSize is the number of calls to batch before sending. 0 = sync
    AsyncBatchSize int

    // FlushInterval is how often to flush batched calls. 0 = flush immediately
    FlushInterval time.Duration
}
```

Default values:
- `Enabled`: true
- `MaxInputSize`: 64KB
- `MaxOutputSize`: 64KB
- `AsyncBatchSize`: 10
- `FlushInterval`: 5 seconds

## POMLCall Structure

The `POMLCall` type captures all relevant information about a POML operation:

```go
type POMLCall struct {
    TraceID         string         // OpenTelemetry trace ID
    SpanID          string         // OpenTelemetry span ID
    Operation       string         // parse, validate, convert
    Format          string         // Output format for convert
    DocumentID      string         // From meta.id
    DocumentVersion string         // From meta.version
    Input           string         // Raw POML input (may be truncated)
    Output          any            // Conversion output (may be truncated)
    Duration        time.Duration  // Operation duration
    Error           string         // Error message if failed
    Timestamp       time.Time      // When the operation occurred
    Metadata        map[string]any // Additional key-value pairs
}
```

## MultiLogger

Use `MultiLogger` to log to multiple backends simultaneously:

```go
mlflowLogger, _ := mlflow.New(mlflow.Config{...})
agentopsLogger, _ := agentops.New(agentops.Config{...})
weaveLogger, _ := weave.New(weave.Config{...})

multi := integrations.NewMultiLogger(mlflowLogger, agentopsLogger, weaveLogger)
defer multi.Close()

// All backends receive the call
err := multi.LogCall(ctx, integrations.POMLCall{
    Operation: "convert",
    Format:    "openai_chat",
    Timestamp: time.Now(),
})
```

## Integration with OpenTelemetry

The integrations work seamlessly with OpenTelemetry tracing:

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/trace"
)

// Get trace context from current span
span := trace.SpanFromContext(ctx)
spanCtx := span.SpanContext()

call := integrations.POMLCall{
    TraceID:   spanCtx.TraceID().String(),
    SpanID:    spanCtx.SpanID().String(),
    Operation: "convert",
    Format:    "openai_chat",
    Timestamp: time.Now(),
}
```

## LSP Features

The Language Server Protocol integration provides:

- **Diagnostics**: Real-time syntax and validation errors
- **Hover**: Element documentation on hover
- **Completion**: Auto-completion for POML elements
- **Document Symbols**: Outline view of document structure
- **Code Actions**: Quick fixes for common issues

### Editor Setup

#### VS Code

Add to your `settings.json`:

```json
{
    "poml.server.path": "poml",
    "poml.server.args": ["lsp", "--stdio"]
}
```

#### Neovim (with nvim-lspconfig)

```lua
require('lspconfig').poml.setup{
    cmd = { "poml", "lsp", "--stdio" },
    filetypes = { "poml", "xml" },
}
```

## Testing

Each integration includes comprehensive tests using mock HTTP servers:

```bash
# Run all integration tests
go test ./integrations/...

# Run specific integration tests
go test ./integrations/mlflow -v
go test ./integrations/agentops -v
go test ./integrations/weave -v
go test ./integrations/lsp -v
```

## Error Handling

All integrations follow consistent error handling:

1. Configuration validation errors are returned from `New()`
2. Logging errors are returned from `LogCall()`
3. `Close()` is idempotent and safe to call multiple times
4. `LogCall()` after `Close()` returns an error

## Performance Considerations

- Use async batching for high-volume logging
- Configure appropriate `MaxInputSize` and `MaxOutputSize` to limit payload sizes
- Use `FlushInterval` to control network overhead
- The LSP server uses concurrent message handling for responsiveness
