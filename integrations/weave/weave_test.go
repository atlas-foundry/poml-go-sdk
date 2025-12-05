package weave

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/atlas-foundry/poml-go-sdk/integrations"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.APIEndpoint != DefaultAPIEndpoint {
		t.Errorf("expected APIEndpoint %s, got %s", DefaultAPIEndpoint, cfg.APIEndpoint)
	}
	if cfg.TraceEndpoint != DefaultTraceEndpoint {
		t.Errorf("expected TraceEndpoint %s, got %s", DefaultTraceEndpoint, cfg.TraceEndpoint)
	}
	if cfg.StreamName != "poml_calls" {
		t.Errorf("expected StreamName 'poml_calls', got %s", cfg.StreamName)
	}
	if !cfg.Enabled {
		t.Error("expected config to be enabled by default")
	}
}

func TestNewValidation(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr string
	}{
		{
			name:    "missing API key",
			cfg:     Config{Project: "proj", Entity: "team"},
			wantErr: "api_key is required",
		},
		{
			name:    "missing project",
			cfg:     Config{APIKey: "key", Entity: "team"},
			wantErr: "project is required",
		},
		{
			name:    "missing entity",
			cfg:     Config{APIKey: "key", Project: "proj"},
			wantErr: "entity is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(tt.cfg)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !contains(err.Error(), tt.wantErr) {
				t.Errorf("expected error containing %q, got %q", tt.wantErr, err.Error())
			}
		})
	}
}

func TestLoggerWithMockServer(t *testing.T) {
	var callStartReceived, callEndReceived, streamLogReceived bool
	var startPayload map[string]any

	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify auth header
		if r.Header.Get("Authorization") != "Bearer test-api-key" {
			t.Errorf("expected Bearer auth header, got: %s", r.Header.Get("Authorization"))
		}

		switch r.URL.Path {
		case "/stream/log":
			streamLogReceived = true
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer apiServer.Close()

	traceServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/call/start":
			callStartReceived = true
			_ = json.NewDecoder(r.Body).Decode(&startPayload)
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "call-123"})
		case "/call/end":
			callEndReceived = true
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer traceServer.Close()

	cfg := Config{
		Config:        integrations.Config{Enabled: true},
		APIKey:        "test-api-key",
		APIEndpoint:   apiServer.URL,
		TraceEndpoint: traceServer.URL,
		Project:       "test-project",
		Entity:        "test-entity",
		StreamName:    "test-stream",
		Tags:          map[string]string{"env": "test"},
		HTTPClient:    apiServer.Client(),
	}

	logger, err := New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer func() { _ = logger.Close() }()

	call := integrations.POMLCall{
		Operation:       "convert",
		Format:          "openai_chat",
		DocumentID:      "doc-123",
		DocumentVersion: "1.0.0",
		Input:           "test input",
		Output:          map[string]string{"result": "ok"},
		Duration:        100 * time.Millisecond,
		TraceID:         "trace-abc",
		SpanID:          "span-xyz",
		Timestamp:       time.Now(),
	}

	if err := logger.LogCall(context.Background(), call); err != nil {
		t.Errorf("LogCall failed: %v", err)
	}

	// Verify all three calls were made
	if !callStartReceived {
		t.Error("expected call/start to be called")
	}
	if !callEndReceived {
		t.Error("expected call/end to be called")
	}
	if !streamLogReceived {
		t.Error("expected stream/log to be called")
	}

	// Verify payload structure
	if startPayload["project_id"] != "test-entity/test-project" {
		t.Errorf("expected project_id 'test-entity/test-project', got '%v'", startPayload["project_id"])
	}

	op := startPayload["op_name"].(map[string]any)
	if op["name"] != "poml.convert" {
		t.Errorf("expected op_name 'poml.convert', got '%v'", op["name"])
	}

	inputs := startPayload["inputs"].(map[string]any)
	if inputs["document_id"] != "doc-123" {
		t.Errorf("expected document_id 'doc-123', got '%v'", inputs["document_id"])
	}
	if inputs["format"] != "openai_chat" {
		t.Errorf("expected format 'openai_chat', got '%v'", inputs["format"])
	}

	attrs := startPayload["attributes"].(map[string]any)
	if attrs["trace_id"] != "trace-abc" {
		t.Errorf("expected trace_id 'trace-abc', got '%v'", attrs["trace_id"])
	}
	if attrs["env"] != "test" {
		t.Errorf("expected env 'test' from tags, got '%v'", attrs["env"])
	}
}

func TestLoggerWithError(t *testing.T) {
	var endPayload map[string]any

	traceServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/call/start":
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "call-123"})
		case "/call/end":
			_ = json.NewDecoder(r.Body).Decode(&endPayload)
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer traceServer.Close()

	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer apiServer.Close()

	cfg := Config{
		Config:        integrations.Config{Enabled: true},
		APIKey:        "test-key",
		APIEndpoint:   apiServer.URL,
		TraceEndpoint: traceServer.URL,
		Project:       "proj",
		Entity:        "team",
		HTTPClient:    traceServer.Client(),
	}

	logger, err := New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer func() { _ = logger.Close() }()

	call := integrations.POMLCall{
		Operation: "convert",
		Error:     "parse error: invalid syntax",
		Timestamp: time.Now(),
	}

	if err := logger.LogCall(context.Background(), call); err != nil {
		t.Errorf("LogCall failed: %v", err)
	}

	// Verify exception was included
	exception := endPayload["exception"].(map[string]any)
	if exception["type"] != "POMLError" {
		t.Errorf("expected exception type 'POMLError', got '%v'", exception["type"])
	}
	if exception["message"] != "parse error: invalid syntax" {
		t.Errorf("expected exception message, got '%v'", exception["message"])
	}
}

func TestLoggerDisabled(t *testing.T) {
	cfg := Config{
		Config:  integrations.Config{Enabled: false},
		APIKey:  "test-key",
		Project: "proj",
		Entity:  "team",
	}

	logger, err := New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer func() { _ = logger.Close() }()

	// LogCall on disabled logger should succeed without sending
	call := integrations.POMLCall{Operation: "convert"}
	if err := logger.LogCall(context.Background(), call); err != nil {
		t.Errorf("LogCall on disabled logger should succeed: %v", err)
	}
}

func TestLoggerAsyncBatching(t *testing.T) {
	var callCount int

	traceServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/call/start":
			callCount++
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "call-" + string(rune(callCount))})
		case "/call/end":
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer traceServer.Close()

	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer apiServer.Close()

	cfg := Config{
		Config: integrations.Config{
			Enabled:        true,
			AsyncBatchSize: 3,
			FlushInterval:  100 * time.Millisecond,
		},
		APIKey:        "test-key",
		APIEndpoint:   apiServer.URL,
		TraceEndpoint: traceServer.URL,
		Project:       "proj",
		Entity:        "team",
		HTTPClient:    traceServer.Client(),
	}

	logger, err := New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	// Log 2 calls - should not trigger flush yet
	for i := 0; i < 2; i++ {
		call := integrations.POMLCall{
			Operation: "convert",
			Timestamp: time.Now(),
		}
		if err := logger.LogCall(context.Background(), call); err != nil {
			t.Errorf("LogCall failed: %v", err)
		}
	}

	if callCount > 0 {
		t.Errorf("expected no calls before batch threshold, got %d", callCount)
	}

	// Close should flush
	if err := logger.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}

	if callCount != 2 {
		t.Errorf("expected 2 calls after close, got %d", callCount)
	}
}

func TestLoggerTruncation(t *testing.T) {
	var startPayload map[string]any

	traceServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/call/start":
			_ = json.NewDecoder(r.Body).Decode(&startPayload)
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "call-1"})
		case "/call/end":
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer traceServer.Close()

	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer apiServer.Close()

	cfg := Config{
		Config: integrations.Config{
			Enabled:       true,
			MaxInputSize:  20,
			MaxOutputSize: 20,
		},
		APIKey:        "test-key",
		APIEndpoint:   apiServer.URL,
		TraceEndpoint: traceServer.URL,
		Project:       "proj",
		Entity:        "team",
		HTTPClient:    traceServer.Client(),
	}

	logger, err := New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer func() { _ = logger.Close() }()

	call := integrations.POMLCall{
		Operation: "convert",
		Input:     "this is a very long input string that should be truncated",
		Output:    map[string]string{"key": "very long value here"},
		Timestamp: time.Now(),
	}

	if err := logger.LogCall(context.Background(), call); err != nil {
		t.Errorf("LogCall failed: %v", err)
	}

	inputs := startPayload["inputs"].(map[string]any)
	input := inputs["input"].(string)
	if len(input) > 20 {
		t.Errorf("input should be truncated to 20 chars, got %d", len(input))
	}
}

func TestLoggerCloseIdempotent(t *testing.T) {
	cfg := Config{
		Config:  integrations.Config{Enabled: true},
		APIKey:  "test-key",
		Project: "proj",
		Entity:  "team",
	}

	logger, err := New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	// Close multiple times should not panic
	_ = logger.Close()
	_ = logger.Close()
	_ = logger.Close()
}

func TestLoggerAfterClose(t *testing.T) {
	cfg := Config{
		Config:  integrations.Config{Enabled: true},
		APIKey:  "test-key",
		Project: "proj",
		Entity:  "team",
	}

	logger, err := New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	_ = logger.Close()

	// LogCall after close should fail
	err = logger.LogCall(context.Background(), integrations.POMLCall{Operation: "convert"})
	if err == nil {
		t.Error("expected error when logging after close")
	}
	if !contains(err.Error(), "closed") {
		t.Errorf("expected error about being closed, got: %v", err)
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
