package mcp

import (
	"context"
	"hash/fnv"
	"math/rand"
	"sync"

	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

// seededIDGenerator yields deterministic IDs for repeatable traces.
type seededIDGenerator struct {
	mu  sync.Mutex
	rng *rand.Rand
}

func newSeededIDGenerator(seed string) sdktrace.IDGenerator {
	hasher := fnv.New64a()
	_, _ = hasher.Write([]byte(seed))
	sum := hasher.Sum64()
	if sum == 0 {
		sum = 1 // zero seed makes math/rand choke
	}
	return &seededIDGenerator{rng: rand.New(rand.NewSource(int64(sum)))}
}

func (g *seededIDGenerator) NewIDs(ctx context.Context) (trace.TraceID, trace.SpanID) {
	g.mu.Lock()
	defer g.mu.Unlock()
	var tid trace.TraceID
	var sid trace.SpanID
	_, _ = g.rng.Read(tid[:])
	_, _ = g.rng.Read(sid[:])
	return tid, sid
}

func (g *seededIDGenerator) NewSpanID(ctx context.Context, traceID trace.TraceID) trace.SpanID {
	g.mu.Lock()
	defer g.mu.Unlock()
	var sid trace.SpanID
	_, _ = g.rng.Read(sid[:])
	return sid
}

// InMemoryTracerProvider returns a deterministic in-memory tracer provider for testing/demo.
func InMemoryTracerProvider(seed string) trace.TracerProvider {
	exp := tracetest.NewInMemoryExporter()
	return sdktrace.NewTracerProvider(
		sdktrace.WithIDGenerator(newSeededIDGenerator(seed)),
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(resource.Empty()),
	)
}
