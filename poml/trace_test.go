package poml

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel/attribute"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func newTracer(t *testing.T) (*tracetest.SpanRecorder, TraceOptions) {
	t.Helper()
	rec := tracetest.NewSpanRecorder()
	tp := tracesdk.NewTracerProvider(tracesdk.WithSpanProcessor(rec))
	return rec, TraceOptions{TracerProvider: tp}
}

func TestConvertWithTrace(t *testing.T) {
	rec, traceOpts := newTracer(t)
	builder := NewBuilder()
	builder.Meta("trace.demo", "1.0.0", "trace-owner")
	builder.Role("test")
	builder.Human("hello")
	doc := builder.Build()

	_, err := ConvertWithTrace(context.Background(), doc, FormatDict, ConvertOptions{Trace: traceOpts})
	if err != nil {
		t.Fatalf("ConvertWithTrace returned error: %v", err)
	}

	spans := rec.Ended()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}
	span := spans[0]
	if span.Name() != "poml.convert" {
		t.Fatalf("unexpected span name: %s", span.Name())
	}
	if got := findAttr(span.Attributes(), "poml.format"); got != string(FormatDict) {
		t.Fatalf("expected format attr %s, got %v", FormatDict, got)
	}
	if got := findAttr(span.Attributes(), "poml.meta.id"); got != "trace.demo" {
		t.Fatalf("expected meta id trace.demo, got %v", got)
	}
}

func TestParseWithTrace(t *testing.T) {
	rec, traceOpts := newTracer(t)
	src := `<poml><meta><id>x</id><version>1</version><owner>o</owner></meta><role>r</role><task>t</task></poml>`
	doc, err := ParseStringWithTrace(context.Background(), src, traceOpts)
	if err != nil {
		t.Fatalf("ParseStringWithTrace error: %v", err)
	}
	if doc.Meta.ID != "x" {
		t.Fatalf("expected meta.id x, got %s", doc.Meta.ID)
	}
	spans := rec.Ended()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}
	if spans[0].Name() != "poml.parse" {
		t.Fatalf("unexpected span name: %s", spans[0].Name())
	}
}

func TestValidateWithTrace(t *testing.T) {
	rec, traceOpts := newTracer(t)
	builder := NewBuilder()
	builder.Meta("val.demo", "1.0.0", "owner")
	builder.Role("r")
	builder.Task("t")
	doc := builder.Build()

	if err := doc.ValidateWithTrace(context.Background(), traceOpts); err != nil {
		t.Fatalf("ValidateWithTrace error: %v", err)
	}
	spans := rec.Ended()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}
	span := spans[0]
	if span.Name() != "poml.validate" {
		t.Fatalf("unexpected span name: %s", span.Name())
	}
	if got := findAttr(span.Attributes(), "poml.meta.id"); got != "val.demo" {
		t.Fatalf("expected meta id val.demo, got %v", got)
	}
}

func TestTraceOptionsAttributes(t *testing.T) {
	rec := tracetest.NewSpanRecorder()
	tp := tracesdk.NewTracerProvider(tracesdk.WithSpanProcessor(rec))
	traceOpts := TraceOptions{
		TracerProvider: tp,
		Attributes: []attribute.KeyValue{
			attribute.String("common", "attr"),
		},
	}
	builder := NewBuilder()
	builder.Meta("common.demo", "1.0.0", "owner")
	builder.Role("r")
	builder.Task("t")
	doc := builder.Build()

	if err := doc.ValidateWithTrace(context.Background(), traceOpts); err != nil {
		t.Fatalf("ValidateWithTrace error: %v", err)
	}
	if len(rec.Ended()) != 1 {
		t.Fatalf("expected 1 span, got %d", len(rec.Ended()))
	}
	if got := findAttr(rec.Ended()[0].Attributes(), "common"); got != "attr" {
		t.Fatalf("expected common attr attr, got %v", got)
	}
}

func findAttr(attrs []attribute.KeyValue, key string) string {
	for _, kv := range attrs {
		if string(kv.Key) == key {
			return kv.Value.AsString()
		}
	}
	return ""
}
