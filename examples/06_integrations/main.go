// Example: Integration with Observability Platforms
//
// This example demonstrates how to integrate POML with:
// 1. MLflow for experiment tracking
// 2. AgentOps for agent observability
// 3. Weave for W&B artifact tracking
// 4. OpenTelemetry for distributed tracing
//
// Run: go run main.go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/atlas-foundry/poml-go-sdk/integrations"
	"github.com/atlas-foundry/poml-go-sdk/poml"
)

func main() {
	ctx := context.Background()

	// Parse a POML document
	input := `<poml>
  <meta>
    <id>tracked-prompt</id>
    <version>1.2.0</version>
    <owner>ml-team</owner>
  </meta>

  <role>You are an AI assistant for data analysis.</role>
  <task>Analyze the provided dataset and generate insights.</task>
  <human-msg>Analyze Q4 sales data.</human-msg>
</poml>`

	// Measure parse time
	parseStart := time.Now()
	doc, err := poml.ParseString(input)
	if err != nil {
		log.Fatalf("Parse error: %v", err)
	}
	parseDuration := time.Since(parseStart)

	// Measure validation time
	validateStart := time.Now()
	if err := doc.Validate(); err != nil {
		log.Fatalf("Validation error: %v", err)
	}
	validateDuration := time.Since(validateStart)

	// Measure conversion time
	convertStart := time.Now()
	output, err := poml.Convert(doc, poml.FormatOpenAIChat, poml.ConvertOptions{})
	if err != nil {
		log.Fatalf("Convert error: %v", err)
	}
	convertDuration := time.Since(convertStart)

	fmt.Println("=== POML Operations ===")
	fmt.Printf("Document: %s v%s\n", doc.Meta.ID, doc.Meta.Version)
	fmt.Printf("Parse:    %v\n", parseDuration)
	fmt.Printf("Validate: %v\n", validateDuration)
	fmt.Printf("Convert:  %v\n", convertDuration)

	// Create POMLCall records for logging
	calls := []integrations.POMLCall{
		{
			Operation:       "parse",
			DocumentID:      doc.Meta.ID,
			DocumentVersion: doc.Meta.Version,
			Input:           input,
			Duration:        parseDuration,
			Timestamp:       parseStart,
			TraceID:         "trace-abc123",
			SpanID:          "span-001",
		},
		{
			Operation:       "validate",
			DocumentID:      doc.Meta.ID,
			DocumentVersion: doc.Meta.Version,
			Duration:        validateDuration,
			Timestamp:       validateStart,
			TraceID:         "trace-abc123",
			SpanID:          "span-002",
		},
		{
			Operation:       "convert",
			Format:          "openai_chat",
			DocumentID:      doc.Meta.ID,
			DocumentVersion: doc.Meta.Version,
			Output:          output,
			Duration:        convertDuration,
			Timestamp:       convertStart,
			TraceID:         "trace-abc123",
			SpanID:          "span-003",
		},
	}

	fmt.Println("\n=== POMLCall Records (for observability) ===")
	for _, call := range calls {
		callJSON, _ := json.MarshalIndent(map[string]any{
			"operation":        call.Operation,
			"format":           call.Format,
			"document_id":      call.DocumentID,
			"document_version": call.DocumentVersion,
			"duration_ms":      call.Duration.Milliseconds(),
			"trace_id":         call.TraceID,
			"span_id":          call.SpanID,
		}, "", "  ")
		fmt.Println(string(callJSON))
	}

	// Demonstrate MultiLogger pattern
	fmt.Println("\n=== Integration Patterns ===")
	fmt.Print(`
// MLflow Integration
mlflowLogger, _ := mlflow.New(mlflow.Config{
    TrackingURI:    "http://localhost:5000",
    ExperimentName: "poml-prompts",
})

// AgentOps Integration
agentopsLogger, _ := agentops.New(agentops.Config{
    APIKey:    os.Getenv("AGENTOPS_API_KEY"),
    ProjectID: "my-project",
})

// Weave Integration
weaveLogger, _ := weave.New(weave.Config{
    APIKey:  os.Getenv("WANDB_API_KEY"),
    Project: "poml-tracking",
    Entity:  "my-team",
})

// Combine all loggers
multi := integrations.NewMultiLogger(mlflowLogger, agentopsLogger, weaveLogger)
defer multi.Close()

// Log operations to all platforms
for _, call := range calls {
    multi.LogCall(ctx, call)
}
`)

	// Demonstrate OpenTelemetry tracing
	fmt.Println("=== OpenTelemetry Tracing ===")
	fmt.Print(`
// With OpenTelemetry tracing
output, err := poml.ConvertWithTrace(ctx, doc, poml.FormatOpenAIChat, poml.ConvertOptions{
    Trace: poml.TraceOptions{
        Tracer: otel.Tracer("poml"),
    },
})

// This creates spans:
// - poml.convert (root)
//   - poml.convert.openai_chat (format-specific)
//   - poml.template.expand (if templates used)
//     - poml.template.let
//     - poml.template.interpolate
`)

	_ = ctx // Used in examples above
}
