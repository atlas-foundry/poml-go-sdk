package agentops

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
	if !cfg.Enabled {
		t.Error("expected config to be enabled by default")
	}
}

func TestNewValidation(t *testing.T) {
	_, err := New(Config{})
	if err == nil {
		t.Fatal("expected error for missing API key")
	}
	if !contains(err.Error(), "api_key is required") {
		t.Errorf("expected 'api_key is required' error, got: %v", err)
	}
}

func TestLoggerWithMockServer(t *testing.T) {
	var receivedEvents []map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify API key header
		if r.Header.Get("X-Agentops-Api-Key") != "test-key" {
			t.Errorf("expected API key header, got: %s", r.Header.Get("X-Agentops-Api-Key"))
		}

		switch r.URL.Path {
		case "/sessions":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"session_id": "session-123",
			})
		case "/events":
			var event map[string]any
			_ = json.NewDecoder(r.Body).Decode(&event)
			receivedEvents = append(receivedEvents, event)
			w.WriteHeader(http.StatusOK)
		case "/sessions/end":
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	cfg := Config{
		Config:      integrations.Config{Enabled: true},
		APIKey:      "test-key",
		APIEndpoint: server.URL,
		ProjectID:   "test-project",
		AgentID:     "test-agent",
		Tags:        []string{"test", "integration"},
		HTTPClient:  server.Client(),
	}

	logger, err := New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	// Verify session was created
	if logger.SessionID() != "session-123" {
		t.Errorf("expected session ID 'session-123', got '%s'", logger.SessionID())
	}

	call := integrations.POMLCall{
		Operation:       "convert",
		Format:          "openai_chat",
		DocumentID:      "test-doc",
		DocumentVersion: "1.0.0",
		Duration:        100 * time.Millisecond,
		TraceID:         "trace-abc",
		SpanID:          "span-xyz",
		Timestamp:       time.Now(),
		Metadata:        map[string]any{"custom": "data"},
	}

	if err := logger.LogCall(context.Background(), call); err != nil {
		t.Errorf("LogCall failed: %v", err)
	}

	if err := logger.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}

	// Verify event was sent
	if len(receivedEvents) != 1 {
		t.Fatalf("expected 1 event, got %d", len(receivedEvents))
	}

	event := receivedEvents[0]
	if event["event_type"] != "poml_call" {
		t.Errorf("expected event_type 'poml_call', got '%v'", event["event_type"])
	}
	if event["session_id"] != "session-123" {
		t.Errorf("expected session_id 'session-123', got '%v'", event["session_id"])
	}
	if event["agent_id"] != "test-agent" {
		t.Errorf("expected agent_id 'test-agent', got '%v'", event["agent_id"])
	}

	data := event["data"].(map[string]any)
	if data["operation"] != "convert" {
		t.Errorf("expected operation 'convert', got '%v'", data["operation"])
	}
	if data["format"] != "openai_chat" {
		t.Errorf("expected format 'openai_chat', got '%v'", data["format"])
	}
}

func TestLoggerWithProvidedSession(t *testing.T) {
	var sessionCreated bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/sessions":
			sessionCreated = true
			_ = json.NewEncoder(w).Encode(map[string]any{"session_id": "new-session"})
		case "/events":
			w.WriteHeader(http.StatusOK)
		case "/sessions/end":
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	cfg := Config{
		Config:      integrations.Config{Enabled: true},
		APIKey:      "test-key",
		APIEndpoint: server.URL,
		SessionID:   "existing-session",
		HTTPClient:  server.Client(),
	}

	logger, err := New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer func() { _ = logger.Close() }()

	if sessionCreated {
		t.Error("should not create session when SessionID is provided")
	}
	if logger.SessionID() != "existing-session" {
		t.Errorf("expected session ID 'existing-session', got '%s'", logger.SessionID())
	}
}

func TestLoggerDisabled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"session_id": "session-123"})
	}))
	defer server.Close()

	cfg := Config{
		Config:      integrations.Config{Enabled: false},
		APIKey:      "test-key",
		APIEndpoint: server.URL,
		HTTPClient:  server.Client(),
	}

	logger, err := New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer func() { _ = logger.Close() }()

	// LogCall on disabled logger should succeed without actually sending
	call := integrations.POMLCall{Operation: "convert"}
	if err := logger.LogCall(context.Background(), call); err != nil {
		t.Errorf("LogCall on disabled logger should succeed: %v", err)
	}
}

func TestLoggerAsyncBatching(t *testing.T) {
	var eventCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/sessions":
			_ = json.NewEncoder(w).Encode(map[string]any{"session_id": "session-1"})
		case "/events":
			eventCount++
			w.WriteHeader(http.StatusOK)
		case "/sessions/end":
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	cfg := Config{
		Config: integrations.Config{
			Enabled:        true,
			AsyncBatchSize: 5,
			FlushInterval:  100 * time.Millisecond,
		},
		APIKey:      "test-key",
		APIEndpoint: server.URL,
		HTTPClient:  server.Client(),
	}

	logger, err := New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	// Log 3 calls - should not trigger flush yet
	for i := 0; i < 3; i++ {
		call := integrations.POMLCall{
			Operation: "convert",
			Timestamp: time.Now(),
		}
		if err := logger.LogCall(context.Background(), call); err != nil {
			t.Errorf("LogCall failed: %v", err)
		}
	}

	if eventCount > 0 {
		t.Errorf("expected no events before batch threshold, got %d", eventCount)
	}

	// Close should flush remaining
	if err := logger.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}

	if eventCount != 3 {
		t.Errorf("expected 3 events after close, got %d", eventCount)
	}
}

func TestLoggerWithErrorField(t *testing.T) {
	var receivedEvent map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/sessions":
			_ = json.NewEncoder(w).Encode(map[string]any{"session_id": "session-1"})
		case "/events":
			_ = json.NewDecoder(r.Body).Decode(&receivedEvent)
			w.WriteHeader(http.StatusOK)
		case "/sessions/end":
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	cfg := Config{
		Config:      integrations.Config{Enabled: true},
		APIKey:      "test-key",
		APIEndpoint: server.URL,
		HTTPClient:  server.Client(),
	}

	logger, err := New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer func() { _ = logger.Close() }()

	call := integrations.POMLCall{
		Operation: "convert",
		Error:     "parse error: invalid POML",
		Timestamp: time.Now(),
	}

	if err := logger.LogCall(context.Background(), call); err != nil {
		t.Errorf("LogCall failed: %v", err)
	}

	data := receivedEvent["data"].(map[string]any)
	if data["error"] != "parse error: invalid POML" {
		t.Errorf("expected error 'parse error: invalid POML', got '%v'", data["error"])
	}
}

func TestLoggerTruncation(t *testing.T) {
	var receivedEvent map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/sessions":
			_ = json.NewEncoder(w).Encode(map[string]any{"session_id": "session-1"})
		case "/events":
			_ = json.NewDecoder(r.Body).Decode(&receivedEvent)
			w.WriteHeader(http.StatusOK)
		case "/sessions/end":
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	cfg := Config{
		Config: integrations.Config{
			Enabled:       true,
			MaxInputSize:  20,
			MaxOutputSize: 20,
		},
		APIKey:      "test-key",
		APIEndpoint: server.URL,
		HTTPClient:  server.Client(),
	}

	logger, err := New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer func() { _ = logger.Close() }()

	call := integrations.POMLCall{
		Operation: "convert",
		Input:     "this is a very long input string that exceeds the limit",
		Output:    "this is a very long output string that exceeds the limit",
		Timestamp: time.Now(),
	}

	if err := logger.LogCall(context.Background(), call); err != nil {
		t.Errorf("LogCall failed: %v", err)
	}

	data := receivedEvent["data"].(map[string]any)
	input := data["input"].(string)
	if len(input) > 20 {
		t.Errorf("input should be truncated to 20 chars, got %d", len(input))
	}
}

func TestLoggerCloseIdempotent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"session_id": "session-1"})
	}))
	defer server.Close()

	cfg := Config{
		Config:      integrations.Config{Enabled: true},
		APIKey:      "test-key",
		APIEndpoint: server.URL,
		HTTPClient:  server.Client(),
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
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"session_id": "session-1"})
	}))
	defer server.Close()

	cfg := Config{
		Config:      integrations.Config{Enabled: true},
		APIKey:      "test-key",
		APIEndpoint: server.URL,
		HTTPClient:  server.Client(),
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
