package poml

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestTraceOptionsTracerFallback(t *testing.T) {
	to := TraceOptions{}
	if to.tracer() == nil {
		t.Fatalf("expected default tracer")
	}
}

func TestTraceOptionsTracerCustomAndStart(t *testing.T) {
	tp := noop.NewTracerProvider()
	to := TraceOptions{
		TracerProvider: tp,
		Attributes:     []attribute.KeyValue{attribute.String("k", "v")},
	}
	ctx, span := to.start(context.Background(), "test-span", attribute.String("k2", "v2"))
	if ctx == nil || span == nil {
		t.Fatalf("start returned nil span or ctx")
	}
	span.End()
}
