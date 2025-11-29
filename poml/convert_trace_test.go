package poml

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
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

func TestConvertWithTraceCapturesMeta(t *testing.T) {
	doc := Document{
		Meta:  Meta{ID: "trace.conv", Version: "1", Owner: "o"},
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
	exp := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithBatcher(exp))
	opts := ConvertOptions{
		Trace: TraceOptions{
			TracerProvider: tp,
		},
	}
	if _, err := ConvertWithTrace(context.Background(), doc, FormatMessageDict, opts); err != nil {
		t.Fatalf("ConvertWithTrace: %v", err)
	}
	tp.ForceFlush(context.Background())
	spans := exp.GetSpans()
	if len(spans) == 0 {
		t.Fatalf("expected convert span recorded")
	}
	found := false
	for _, kv := range spans[0].Attributes {
		if kv.Key == "poml.meta.id" && kv.Value.AsString() == "trace.conv" {
			found = true
		}
	}
	if !found {
		t.Fatalf("meta id not found in convert span: %+v", spans[0].Attributes)
	}
}
