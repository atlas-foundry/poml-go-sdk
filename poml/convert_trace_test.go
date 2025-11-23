package poml

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestConvertWithTrace(t *testing.T) {
	doc := Document{
		Meta:  Meta{ID: "a", Version: "1", Owner: "o"},
		Role:  Block{Body: "r"},
		Tasks: []Block{{Body: "t"}},
		Messages: []Message{
			{Role: "human", Body: "hi"},
		},
		Elements: []Element{
			{Type: ElementRole, Index: 0},
			{Type: ElementTask, Index: 0},
			{Type: ElementHumanMsg, Index: 0},
		},
	}
	opts := ConvertOptions{
		Trace: TraceOptions{
			TracerProvider: noop.NewTracerProvider(),
			Attributes:     []attribute.KeyValue{attribute.String("a", "b")},
		},
	}
	if _, err := ConvertWithTrace(context.Background(), doc, FormatMessageDict, opts); err != nil {
		t.Fatalf("ConvertWithTrace: %v", err)
	}
}
