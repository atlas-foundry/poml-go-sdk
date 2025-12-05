package integrations

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if !cfg.Enabled {
		t.Error("default config should be enabled")
	}
	if cfg.MaxInputSize != 64*1024 {
		t.Errorf("expected MaxInputSize 64KB, got %d", cfg.MaxInputSize)
	}
	if cfg.MaxOutputSize != 64*1024 {
		t.Errorf("expected MaxOutputSize 64KB, got %d", cfg.MaxOutputSize)
	}
	if cfg.AsyncBatchSize != 10 {
		t.Errorf("expected AsyncBatchSize 10, got %d", cfg.AsyncBatchSize)
	}
	if cfg.FlushInterval != 5*time.Second {
		t.Errorf("expected FlushInterval 5s, got %v", cfg.FlushInterval)
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{"no limit", "hello world", 0, "hello world"},
		{"within limit", "hello", 10, "hello"},
		{"at limit", "hello", 5, "hello"},
		{"over limit", "hello world", 8, "hello..."},
		{"very short limit", "hello", 3, "hel"},
		{"empty string", "", 10, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateString(tt.input, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncateString(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
			}
		})
	}
}

func TestTruncateOutput(t *testing.T) {
	tests := []struct {
		name   string
		output any
		maxLen int
	}{
		{"no limit", map[string]string{"key": "value"}, 0},
		{"within limit", "short", 100},
		{"over limit", "this is a very long string that should be truncated", 20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateOutput(tt.output, tt.maxLen)
			if got == nil {
				t.Error("truncateOutput returned nil")
			}
		})
	}
}

// mockLogger implements POMLLogger for testing
type mockLogger struct {
	calls    []POMLCall
	flushErr error
	closeErr error
	logErr   error
}

func (m *mockLogger) LogCall(ctx context.Context, call POMLCall) error {
	if m.logErr != nil {
		return m.logErr
	}
	m.calls = append(m.calls, call)
	return nil
}

func (m *mockLogger) Flush(ctx context.Context) error {
	return m.flushErr
}

func (m *mockLogger) Close() error {
	return m.closeErr
}

func TestMultiLogger(t *testing.T) {
	logger1 := &mockLogger{}
	logger2 := &mockLogger{}

	multi := NewMultiLogger(logger1, logger2)

	call := POMLCall{
		Operation: "convert",
		Format:    "openai_chat",
		Timestamp: time.Now(),
	}

	// Test LogCall
	if err := multi.LogCall(context.Background(), call); err != nil {
		t.Errorf("LogCall failed: %v", err)
	}
	if len(logger1.calls) != 1 {
		t.Errorf("expected 1 call in logger1, got %d", len(logger1.calls))
	}
	if len(logger2.calls) != 1 {
		t.Errorf("expected 1 call in logger2, got %d", len(logger2.calls))
	}

	// Test Flush
	if err := multi.Flush(context.Background()); err != nil {
		t.Errorf("Flush failed: %v", err)
	}

	// Test Close
	if err := multi.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

func TestMultiLoggerErrors(t *testing.T) {
	expectedErr := errors.New("test error")
	logger1 := &mockLogger{}
	logger2 := &mockLogger{logErr: expectedErr}

	multi := NewMultiLogger(logger1, logger2)

	call := POMLCall{Operation: "convert"}
	err := multi.LogCall(context.Background(), call)
	if err != expectedErr {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}
	// First logger should still have received the call
	if len(logger1.calls) != 1 {
		t.Errorf("expected 1 call in logger1, got %d", len(logger1.calls))
	}
}

func TestMultiLoggerAdd(t *testing.T) {
	multi := NewMultiLogger()
	logger := &mockLogger{}

	multi.Add(logger)

	call := POMLCall{Operation: "parse"}
	if err := multi.LogCall(context.Background(), call); err != nil {
		t.Errorf("LogCall failed: %v", err)
	}
	if len(logger.calls) != 1 {
		t.Errorf("expected 1 call, got %d", len(logger.calls))
	}
}

func TestPOMLCallFields(t *testing.T) {
	now := time.Now()
	call := POMLCall{
		TraceID:         "trace-123",
		SpanID:          "span-456",
		Operation:       "convert",
		Format:          "openai_chat",
		DocumentID:      "doc-789",
		DocumentVersion: "1.0.0",
		Input:           "test input",
		Output:          map[string]string{"result": "ok"},
		Duration:        100 * time.Millisecond,
		Error:           "",
		Timestamp:       now,
		Metadata:        map[string]any{"key": "value"},
	}

	if call.TraceID != "trace-123" {
		t.Errorf("expected TraceID trace-123, got %s", call.TraceID)
	}
	if call.Duration != 100*time.Millisecond {
		t.Errorf("expected Duration 100ms, got %v", call.Duration)
	}
	if call.Metadata["key"] != "value" {
		t.Errorf("expected metadata key=value, got %v", call.Metadata["key"])
	}
}
