package poml

import (
	"context"
	"encoding/json"
	"hash/fnv"
	"math/rand"
	"os"
	"sort"
	"sync"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

// SeededIDGenerator produces deterministic trace/span IDs for reproducible fixtures.
func SeededIDGenerator(seed string) sdktrace.IDGenerator {
	hasher := fnv.New64a()
	_, _ = hasher.Write([]byte(seed))
	sum := hasher.Sum64()
	if sum == 0 {
		sum = 1 // zero seed makes math/rand choke
	}
	return &seededIDGenerator{rng: rand.New(rand.NewSource(int64(sum)))}
}

type seededIDGenerator struct {
	mu  sync.Mutex
	rng *rand.Rand
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

func (g *seededIDGenerator) NewSpanID(ctx context.Context, _ trace.TraceID) trace.SpanID {
	g.mu.Lock()
	defer g.mu.Unlock()
	var sid trace.SpanID
	_, _ = g.rng.Read(sid[:])
	return sid
}

// TraceRecorder wraps a deterministic tracer provider and exporter for capturing fixtures.
type TraceRecorder struct {
	Provider *sdktrace.TracerProvider
	exporter *tracetest.InMemoryExporter
}

// NewTraceRecorder seeds a deterministic provider + exporter pair for tests and demos.
func NewTraceRecorder(seed string) TraceRecorder {
	exp := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithIDGenerator(SeededIDGenerator(seed)),
		sdktrace.WithSpanProcessor(sdktrace.NewSimpleSpanProcessor(exp)),
		sdktrace.WithResource(resource.Empty()),
	)
	return TraceRecorder{Provider: tp, exporter: exp}
}

// SpanDump is a JSON-friendly view of a recorded span.
type SpanDump struct {
	Name         string     `json:"name"`
	TraceID      string     `json:"trace_id"`
	SpanID       string     `json:"span_id"`
	ParentSpanID string     `json:"parent_span_id,omitempty"`
	Attributes   []SpanAttr `json:"attributes,omitempty"`
}

// SpanAttr retains attribute ordering for deterministic goldens.
type SpanAttr struct {
	Key   string      `json:"key"`
	Value interface{} `json:"value"`
}

// Dump returns all recorded spans in a deterministic order.
func (r TraceRecorder) Dump() []SpanDump {
	spans := r.exporter.GetSpans()
	sort.Slice(spans, func(i, j int) bool {
		if spans[i].StartTime.Equal(spans[j].StartTime) {
			return spans[i].Name < spans[j].Name
		}
		return spans[i].StartTime.Before(spans[j].StartTime)
	})
	out := make([]SpanDump, 0, len(spans))
	for _, sp := range spans {
		out = append(out, toSpanDump(sp))
	}
	return out
}

// DumpToFile writes a pretty-printed JSON file of recorded spans.
func (r TraceRecorder) DumpToFile(path string) error {
	data, err := json.MarshalIndent(r.Dump(), "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func toSpanDump(sp tracetest.SpanStub) SpanDump {
	parent := ""
	if sp.Parent.IsValid() {
		parent = sp.Parent.SpanID().String()
	}
	attrs := sp.Attributes
	sort.Slice(attrs, func(i, j int) bool { return attrs[i].Key < attrs[j].Key })
	outAttrs := make([]SpanAttr, 0, len(attrs))
	for _, kv := range attrs {
		outAttrs = append(outAttrs, SpanAttr{
			Key:   string(kv.Key),
			Value: attrValue(kv),
		})
	}
	return SpanDump{
		Name:         sp.Name,
		TraceID:      sp.SpanContext.TraceID().String(),
		SpanID:       sp.SpanContext.SpanID().String(),
		ParentSpanID: parent,
		Attributes:   outAttrs,
	}
}

func attrValue(kv attribute.KeyValue) interface{} {
	switch kv.Value.Type() {
	case attribute.BOOL:
		return kv.Value.AsBool()
	case attribute.INT64:
		return float64(kv.Value.AsInt64())
	case attribute.FLOAT64:
		return kv.Value.AsFloat64()
	default:
		return kv.Value.AsString()
	}
}
