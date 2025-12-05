package mcp

import (
	"context"
	"testing"
)

func TestNoopTracerProvider(t *testing.T) {
	tp := NoopTracerProvider()
	if tp == nil {
		t.Fatal("NoopTracerProvider returned nil")
	}
	tracer := tp.Tracer("test")
	if tracer == nil {
		t.Fatal("Tracer returned nil")
	}
}

func TestStdoutTracerProvider(t *testing.T) {
	tp := StdoutTracerProvider()
	if tp == nil {
		t.Fatal("StdoutTracerProvider returned nil")
	}
	// Verify it implements the interface by calling a method
	_ = tp.Tracer("test")
}

func TestOTLPHTTPTracerProviderInsecure(t *testing.T) {
	// This will fail to connect but should still return a provider
	tp := OTLPHTTPTracerProvider("localhost:4318", true)
	if tp == nil {
		t.Fatal("OTLPHTTPTracerProvider returned nil")
	}
	// Verify it implements the interface
	_ = tp.Tracer("test")
}

func TestOTLPHTTPTracerProviderSecure(t *testing.T) {
	tp := OTLPHTTPTracerProvider("localhost:4318", false)
	if tp == nil {
		t.Fatal("OTLPHTTPTracerProvider returned nil")
	}
}

func TestOTLPGRPCTracerProviderInsecure(t *testing.T) {
	tp := OTLPGRPCTracerProvider("localhost:4317", true)
	if tp == nil {
		t.Fatal("OTLPGRPCTracerProvider returned nil")
	}
	// Verify it implements the interface
	_ = tp.Tracer("test")
}

func TestOTLPGRPCTracerProviderSecure(t *testing.T) {
	tp := OTLPGRPCTracerProvider("localhost:4317", false)
	if tp == nil {
		t.Fatal("OTLPGRPCTracerProvider returned nil")
	}
}

func TestInMemoryTracerProvider(t *testing.T) {
	tp := InMemoryTracerProvider("test-seed")
	if tp == nil {
		t.Fatal("InMemoryTracerProvider returned nil")
	}

	// Verify we can create spans
	tracer := tp.Tracer("test")
	_, span := tracer.Start(context.Background(), "test-span")
	if span == nil {
		t.Fatal("span is nil")
	}
	span.End()
}
