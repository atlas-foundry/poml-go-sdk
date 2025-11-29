package mcp

import (
	"github.com/atlas-foundry/poml-go-sdk/poml"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

// InMemoryTracerProvider returns a deterministic in-memory tracer provider for testing/demo.
func InMemoryTracerProvider(seed string) trace.TracerProvider {
	exp := tracetest.NewInMemoryExporter()
	return sdktrace.NewTracerProvider(
		sdktrace.WithIDGenerator(poml.SeededIDGenerator(seed)),
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(resource.Empty()),
	)
}
