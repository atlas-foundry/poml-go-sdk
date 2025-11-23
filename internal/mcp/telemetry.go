package mcp

import (
	"context"
	"log"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
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

// OTLPHTTPTracerProvider creates a tracer provider that exports spans via OTLP/HTTP to endpoint.
func OTLPHTTPTracerProvider(endpoint string, insecure bool) trace.TracerProvider {
	ctx := context.Background()
	opts := []otlptracehttp.Option{otlptracehttp.WithEndpoint(endpoint)}
	if insecure {
		opts = append(opts, otlptracehttp.WithInsecure())
	}
	exp, err := otlptracehttp.New(ctx, opts...)
	if err != nil {
		log.Printf("failed to create otlphttp exporter: %v", err)
		return NoopTracerProvider()
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(resource.Default()),
	)
	return tp
}

// OTLPGRPCTracerProvider creates a tracer provider that exports spans via OTLP/gRPC to endpoint.
func OTLPGRPCTracerProvider(endpoint string, insecure bool) trace.TracerProvider {
	ctx := context.Background()
	opts := []otlptracegrpc.Option{otlptracegrpc.WithEndpoint(endpoint)}
	if insecure {
		opts = append(opts, otlptracegrpc.WithInsecure())
	}
	exp, err := otlptracegrpc.New(ctx, opts...)
	if err != nil {
		log.Printf("failed to create otlpgrpc exporter: %v", err)
		return NoopTracerProvider()
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(resource.Default()),
	)
	return tp
}
