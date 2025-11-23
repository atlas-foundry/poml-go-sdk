package poml

import (
	"context"
	"strings"
	"testing"

	"go.opentelemetry.io/otel/trace/noop"
)

func TestValidateWithTraceErrorPath(t *testing.T) {
	doc := Document{
		Meta:     Meta{ID: "a", Version: "1", Owner: "o"},
		Role:     Block{Body: "r"},
		Tasks:    []Block{{Body: "t"}},
		ToolReqs: []ToolRequest{{ID: "req1", Name: "missing"}},
		Elements: []Element{
			{Type: ElementMeta},
			{Type: ElementRole},
			{Type: ElementTask, Index: 0},
			{Type: ElementToolRequest, Index: 0},
		},
	}
	err := doc.ValidateWithTrace(context.Background(), TraceOptions{TracerProvider: noop.NewTracerProvider()})
	if err == nil {
		t.Fatalf("expected validation error for missing tool definition")
	}
	if !strings.Contains(err.Error(), "unknown tool-definition") {
		t.Fatalf("unexpected error: %v", err)
	}
}
