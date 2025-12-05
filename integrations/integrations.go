// Package integrations provides ecosystem bridge plugins for POML observability and tooling.
//
// This package implements integrations matching the Python SDK's ecosystem bridges:
//   - MLflow: Experiment tracking and model management
//   - AgentOps: AI agent observability and monitoring
//   - Weave: ML artifact versioning and lineage
//   - LSP: Language Server Protocol for IDE integration
//
// Each integration implements the POMLLogger interface for consistent call logging.
package integrations

import (
	"context"
	"encoding/json"
	"time"
)

// POMLCall represents a single POML operation for logging/tracking.
type POMLCall struct {
	// TraceID is the OpenTelemetry trace ID if available.
	TraceID string `json:"trace_id,omitempty"`
	// SpanID is the OpenTelemetry span ID if available.
	SpanID string `json:"span_id,omitempty"`
	// Operation is the type of operation (parse, validate, convert).
	Operation string `json:"operation"`
	// Format is the output format for convert operations.
	Format string `json:"format,omitempty"`
	// DocumentID is the POML document meta.id if available.
	DocumentID string `json:"document_id,omitempty"`
	// DocumentVersion is the POML document meta.version if available.
	DocumentVersion string `json:"document_version,omitempty"`
	// Input is the raw POML input (truncated for large documents).
	Input string `json:"input,omitempty"`
	// Output is the conversion output (truncated for large outputs).
	Output any `json:"output,omitempty"`
	// Duration is how long the operation took.
	Duration time.Duration `json:"duration_ns"`
	// Error contains error message if the operation failed.
	Error string `json:"error,omitempty"`
	// Timestamp is when the operation occurred.
	Timestamp time.Time `json:"timestamp"`
	// Metadata contains additional key-value pairs.
	Metadata map[string]any `json:"metadata,omitempty"`
}

// POMLLogger is the interface for logging POML operations to external systems.
// This matches the Python SDK's log_poml_call() functionality.
type POMLLogger interface {
	// LogCall logs a POML operation to the external system.
	LogCall(ctx context.Context, call POMLCall) error
	// Flush ensures all pending logs are sent.
	Flush(ctx context.Context) error
	// Close releases any resources held by the logger.
	Close() error
}

// Config holds common configuration for integrations.
type Config struct {
	// Enabled controls whether the integration is active.
	Enabled bool `json:"enabled"`
	// MaxInputSize is the maximum input size to log (bytes). 0 means no limit.
	MaxInputSize int `json:"max_input_size,omitempty"`
	// MaxOutputSize is the maximum output size to log (bytes). 0 means no limit.
	MaxOutputSize int `json:"max_output_size,omitempty"`
	// AsyncBatchSize is the number of calls to batch before sending. 0 means sync.
	AsyncBatchSize int `json:"async_batch_size,omitempty"`
	// FlushInterval is how often to flush batched calls. 0 means flush immediately.
	FlushInterval time.Duration `json:"flush_interval,omitempty"`
}

// DefaultConfig returns a sensible default configuration.
func DefaultConfig() Config {
	return Config{
		Enabled:        true,
		MaxInputSize:   64 * 1024, // 64KB
		MaxOutputSize:  64 * 1024, // 64KB
		AsyncBatchSize: 10,
		FlushInterval:  5 * time.Second,
	}
}

// truncateString truncates a string to maxLen bytes, adding "..." if truncated.
func truncateString(s string, maxLen int) string {
	if maxLen <= 0 || len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// truncateOutput truncates output to maxLen bytes when serialized as JSON.
func truncateOutput(output any, maxLen int) any {
	if maxLen <= 0 {
		return output
	}
	data, err := json.Marshal(output)
	if err != nil || len(data) <= maxLen {
		return output
	}
	// Return truncated JSON string representation
	return truncateString(string(data), maxLen)
}

// MultiLogger combines multiple loggers into one.
type MultiLogger struct {
	loggers []POMLLogger
}

// NewMultiLogger creates a logger that writes to multiple backends.
func NewMultiLogger(loggers ...POMLLogger) *MultiLogger {
	return &MultiLogger{loggers: loggers}
}

// LogCall logs to all configured backends.
func (m *MultiLogger) LogCall(ctx context.Context, call POMLCall) error {
	var lastErr error
	for _, l := range m.loggers {
		if err := l.LogCall(ctx, call); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

// Flush flushes all backends.
func (m *MultiLogger) Flush(ctx context.Context) error {
	var lastErr error
	for _, l := range m.loggers {
		if err := l.Flush(ctx); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

// Close closes all backends.
func (m *MultiLogger) Close() error {
	var lastErr error
	for _, l := range m.loggers {
		if err := l.Close(); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

// Add adds a logger to the multi-logger.
func (m *MultiLogger) Add(l POMLLogger) {
	m.loggers = append(m.loggers, l)
}
