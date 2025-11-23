package poml

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// TraceOptions configure OpenTelemetry spans for parsing/validation/conversion.
// If left zero-valued, tracing is skipped and existing behavior is unchanged.
type TraceOptions struct {
	// TracerProvider overrides the global provider; if nil, the global provider is used.
	TracerProvider trace.TracerProvider
	// Attributes are added to every span started via TraceOptions.
	Attributes []attribute.KeyValue
}

func (t TraceOptions) tracer() trace.Tracer {
	tp := t.TracerProvider
	if tp == nil {
		tp = otel.GetTracerProvider()
	}
	return tp.Tracer("github.com/atlas-foundry/poml-go-sdk")
}

func (t TraceOptions) skip() bool {
	return t.TracerProvider == nil && len(t.Attributes) == 0
}

func (t TraceOptions) start(ctx context.Context, name string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	all := make([]attribute.KeyValue, 0, len(t.Attributes)+len(attrs))
	all = append(all, t.Attributes...)
	all = append(all, attrs...)
	return t.tracer().Start(ctx, name, trace.WithAttributes(all...))
}
