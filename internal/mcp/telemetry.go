package mcp

import (
	"log"

	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

// NoopTracerProvider returns a tracer provider that drops spans.
func NoopTracerProvider() trace.TracerProvider {
	return noop.NewTracerProvider()
}

// StdoutTracerProvider creates a tracer provider that exports spans to stdout.
func StdoutTracerProvider() trace.TracerProvider {
	exp, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		log.Printf("failed to create stdout exporter: %v", err)
		return NoopTracerProvider()
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(resource.Default()),
	)
	return tp
}
