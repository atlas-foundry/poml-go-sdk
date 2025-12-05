package mlflow

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

	if cfg.TrackingURI != "http://localhost:5000" {
		t.Errorf("expected TrackingURI http://localhost:5000, got %s", cfg.TrackingURI)
	}
	if cfg.ExperimentName != "poml-prompts" {
		t.Errorf("expected ExperimentName poml-prompts, got %s", cfg.ExperimentName)
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
			name:    "missing tracking URI",
			cfg:     Config{ExperimentName: "test"},
			wantErr: "tracking_uri is required",
		},
		{
			name:    "missing experiment",
			cfg:     Config{TrackingURI: "http://localhost:5000"},
			wantErr: "experiment_name or experiment_id is required",
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
	var requests []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.URL.Path)

		switch r.URL.Path {
		case "/api/2.0/mlflow/experiments/get-by-name":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"experiment": map[string]any{
					"experiment_id": "test-exp-123",
				},
			})
		case "/api/2.0/mlflow/experiments/create":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"experiment_id": "test-exp-123",
			})
		case "/api/2.0/mlflow/runs/create":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"run": map[string]any{
					"info": map[string]any{
						"run_id": "test-run-456",
					},
				},
			})
		case "/api/2.0/mlflow/runs/log-batch":
			w.WriteHeader(http.StatusOK)
		case "/api/2.0/mlflow/runs/update":
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	cfg := Config{
		Config:         integrations.Config{Enabled: true},
		TrackingURI:    server.URL,
		ExperimentName: "test-experiment",
		HTTPClient:     server.Client(),
	}

	logger, err := New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer func() { _ = logger.Close() }()

	call := integrations.POMLCall{
		Operation:  "convert",
		Format:     "openai_chat",
		DocumentID: "test-doc",
		Duration:   50 * time.Millisecond,
		Timestamp:  time.Now(),
	}

	if err := logger.LogCall(context.Background(), call); err != nil {
		t.Errorf("LogCall failed: %v", err)
	}

	// Verify requests were made
	expectedPaths := []string{
		"/api/2.0/mlflow/experiments/get-by-name",
		"/api/2.0/mlflow/runs/create",
		"/api/2.0/mlflow/runs/log-batch", // metrics
		"/api/2.0/mlflow/runs/log-batch", // params
		"/api/2.0/mlflow/runs/update",
	}
	for i, path := range expectedPaths {
		if i >= len(requests) {
			t.Errorf("missing request for %s", path)
			continue
		}
		if requests[i] != path {
			t.Errorf("request %d: expected %s, got %s", i, path, requests[i])
		}
	}
}

func TestLoggerDisabled(t *testing.T) {
	cfg := Config{
		Config:         integrations.Config{Enabled: false},
		TrackingURI:    "http://localhost:5000",
		ExperimentID:   "test-exp",
		ExperimentName: "ignored",
	}

	logger, err := New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	call := integrations.POMLCall{Operation: "convert"}
	if err := logger.LogCall(context.Background(), call); err != nil {
		t.Errorf("LogCall on disabled logger should succeed: %v", err)
	}
}

func TestLoggerAsyncBatching(t *testing.T) {
	var requestCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		switch r.URL.Path {
		case "/api/2.0/mlflow/experiments/get-by-name":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"experiment": map[string]any{"experiment_id": "exp-1"},
			})
		case "/api/2.0/mlflow/runs/create":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"run": map[string]any{"info": map[string]any{"run_id": "run-1"}},
			})
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	cfg := Config{
		Config: integrations.Config{
			Enabled:        true,
			AsyncBatchSize: 3,
			FlushInterval:  100 * time.Millisecond,
		},
		TrackingURI:    server.URL,
		ExperimentName: "test",
		HTTPClient:     server.Client(),
	}

	logger, err := New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	// Initial request count after experiment setup
	initialCount := requestCount

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

	// Should still be batched
	if requestCount > initialCount {
		t.Errorf("expected no additional requests for batched calls, got %d", requestCount-initialCount)
	}

	// Close should flush
	if err := logger.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}

	// Now requests should have been made
	if requestCount == initialCount {
		t.Error("expected requests after close/flush")
	}
}

func TestLoggerWithError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/2.0/mlflow/experiments/get-by-name":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"experiment": map[string]any{"experiment_id": "exp-1"},
			})
		case "/api/2.0/mlflow/runs/create":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"run": map[string]any{"info": map[string]any{"run_id": "run-1"}},
			})
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	cfg := Config{
		Config:         integrations.Config{Enabled: true},
		TrackingURI:    server.URL,
		ExperimentName: "test",
		HTTPClient:     server.Client(),
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
}

func TestLoggerTruncation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/2.0/mlflow/experiments/get-by-name":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"experiment": map[string]any{"experiment_id": "exp-1"},
			})
		case "/api/2.0/mlflow/runs/create":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"run": map[string]any{"info": map[string]any{"run_id": "run-1"}},
			})
		default:
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
		TrackingURI:    server.URL,
		ExperimentName: "test",
		HTTPClient:     server.Client(),
	}

	logger, err := New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer func() { _ = logger.Close() }()

	call := integrations.POMLCall{
		Operation: "convert",
		Input:     "this is a very long input string that should be truncated",
		Output:    map[string]string{"key": "this is a very long value"},
		Timestamp: time.Now(),
	}

	if err := logger.LogCall(context.Background(), call); err != nil {
		t.Errorf("LogCall failed: %v", err)
	}
}

func TestLoggerCloseIdempotent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"experiment": map[string]any{"experiment_id": "exp-1"},
		})
	}))
	defer server.Close()

	cfg := Config{
		Config:         integrations.Config{Enabled: true},
		TrackingURI:    server.URL,
		ExperimentName: "test",
		HTTPClient:     server.Client(),
	}

	logger, err := New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	// Close multiple times should not panic
	if err := logger.Close(); err != nil {
		t.Errorf("first Close failed: %v", err)
	}
	if err := logger.Close(); err != nil {
		t.Errorf("second Close failed: %v", err)
	}
}

func TestLoggerAfterClose(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"experiment": map[string]any{"experiment_id": "exp-1"},
		})
	}))
	defer server.Close()

	cfg := Config{
		Config:         integrations.Config{Enabled: true},
		TrackingURI:    server.URL,
		ExperimentName: "test",
		HTTPClient:     server.Client(),
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
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr, 0))
}

func containsAt(s, substr string, start int) bool {
	for i := start; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
