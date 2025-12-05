// Package weave provides Weights & Biases Weave integration for POML call logging.
//
// Weave is W&B's toolkit for developing and evaluating LLM applications.
// This integration logs POML operations as Weave calls for tracing and evaluation.
//
// Usage:
//
//	logger, err := weave.New(weave.Config{
//	    APIKey: "your-wandb-api-key",
//	    Project: "my-project",
//	    Entity: "my-team",
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer logger.Close()
//
//	// Log a POML call
//	err = logger.LogCall(ctx, integrations.POMLCall{
//	    Operation: "convert",
//	    Format: "openai_chat",
//	    DocumentID: "my-prompt",
//	    Duration: 50 * time.Millisecond,
//	})
package weave

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/atlas-foundry/poml-go-sdk/integrations"
)

const (
	// DefaultAPIEndpoint is the W&B Weave API endpoint.
	DefaultAPIEndpoint = "https://api.wandb.ai/weave"
	// DefaultTraceEndpoint is the W&B trace ingestion endpoint.
	DefaultTraceEndpoint = "https://trace.wandb.ai"
)

// Config holds Weave-specific configuration.
type Config struct {
	integrations.Config
	// APIKey is the W&B API key (required).
	APIKey string `json:"api_key"`
	// APIEndpoint is the Weave API endpoint (optional).
	APIEndpoint string `json:"api_endpoint,omitempty"`
	// TraceEndpoint is the trace ingestion endpoint (optional).
	TraceEndpoint string `json:"trace_endpoint,omitempty"`
	// Project is the W&B project name.
	Project string `json:"project"`
	// Entity is the W&B entity (team or user).
	Entity string `json:"entity"`
	// StreamName is the name of the stream table.
	StreamName string `json:"stream_name,omitempty"`
	// Tags are additional tags to add to each call.
	Tags map[string]string `json:"tags,omitempty"`
	// HTTPClient is an optional custom HTTP client.
	HTTPClient *http.Client `json:"-"`
}

// DefaultConfig returns default Weave configuration.
func DefaultConfig() Config {
	return Config{
		Config:        integrations.DefaultConfig(),
		APIEndpoint:   DefaultAPIEndpoint,
		TraceEndpoint: DefaultTraceEndpoint,
		StreamName:    "poml_calls",
	}
}

// Logger implements POMLLogger for Weave.
type Logger struct {
	config Config
	client *http.Client

	mu      sync.Mutex
	batch   []integrations.POMLCall
	closed  bool
	closeCh chan struct{}
	wg      sync.WaitGroup
}

// New creates a new Weave logger.
func New(cfg Config) (*Logger, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("weave: api_key is required")
	}
	if cfg.Project == "" {
		return nil, fmt.Errorf("weave: project is required")
	}
	if cfg.Entity == "" {
		return nil, fmt.Errorf("weave: entity is required")
	}
	if cfg.APIEndpoint == "" {
		cfg.APIEndpoint = DefaultAPIEndpoint
	}
	if cfg.TraceEndpoint == "" {
		cfg.TraceEndpoint = DefaultTraceEndpoint
	}
	if cfg.StreamName == "" {
		cfg.StreamName = "poml_calls"
	}

	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}

	l := &Logger{
		config:  cfg,
		client:  client,
		closeCh: make(chan struct{}),
	}

	// Start batch flusher if async
	if cfg.AsyncBatchSize > 0 && cfg.FlushInterval > 0 {
		l.wg.Add(1)
		go l.flushLoop()
	}

	return l, nil
}

// LogCall logs a POML operation to Weave.
func (l *Logger) LogCall(ctx context.Context, call integrations.POMLCall) error {
	if !l.config.Enabled {
		return nil
	}

	l.mu.Lock()
	if l.closed {
		l.mu.Unlock()
		return fmt.Errorf("weave: logger is closed")
	}

	// Truncate if needed
	call.Input = truncateString(call.Input, l.config.MaxInputSize)
	call.Output = truncateOutput(call.Output, l.config.MaxOutputSize)

	// Async batching
	if l.config.AsyncBatchSize > 0 {
		l.batch = append(l.batch, call)
		shouldFlush := len(l.batch) >= l.config.AsyncBatchSize
		l.mu.Unlock()
		if shouldFlush {
			return l.Flush(ctx)
		}
		return nil
	}
	l.mu.Unlock()

	// Sync logging
	return l.logCallSync(ctx, call)
}

func (l *Logger) logCallSync(ctx context.Context, call integrations.POMLCall) error {
	// Create a Weave call object
	weaveCall := weaveCallStart{
		Project: fmt.Sprintf("%s/%s", l.config.Entity, l.config.Project),
		Op: weaveOp{
			Name: fmt.Sprintf("poml.%s", call.Operation),
		},
		StartedAt: call.Timestamp,
		Inputs: map[string]any{
			"document_id":      call.DocumentID,
			"document_version": call.DocumentVersion,
			"format":           call.Format,
			"input":            call.Input,
		},
		Attributes: map[string]any{
			"trace_id": call.TraceID,
			"span_id":  call.SpanID,
		},
	}
	for k, v := range l.config.Tags {
		weaveCall.Attributes[k] = v
	}

	// Start the call
	callID, err := l.startCall(ctx, weaveCall)
	if err != nil {
		return fmt.Errorf("weave: start call: %w", err)
	}

	// End the call
	endedAt := call.Timestamp.Add(call.Duration)
	callEnd := weaveCallEnd{
		CallID:  callID,
		EndedAt: endedAt,
		Output:  call.Output,
	}
	if call.Error != "" {
		callEnd.Exception = &weaveException{
			Type:    "POMLError",
			Message: call.Error,
		}
	}

	if err := l.endCall(ctx, callEnd); err != nil {
		return fmt.Errorf("weave: end call: %w", err)
	}

	// Also log to stream table for evaluation
	return l.logToStream(ctx, call)
}

// Flush sends any batched calls.
func (l *Logger) Flush(ctx context.Context) error {
	l.mu.Lock()
	if len(l.batch) == 0 {
		l.mu.Unlock()
		return nil
	}
	batch := l.batch
	l.batch = nil
	l.mu.Unlock()

	var lastErr error
	for _, call := range batch {
		if err := l.logCallSync(ctx, call); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

// Close flushes and closes the logger.
func (l *Logger) Close() error {
	l.mu.Lock()
	if l.closed {
		l.mu.Unlock()
		return nil
	}
	l.closed = true
	close(l.closeCh)
	l.mu.Unlock()

	l.wg.Wait()
	return l.Flush(context.Background())
}

func (l *Logger) flushLoop() {
	defer l.wg.Done()
	ticker := time.NewTicker(l.config.FlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			_ = l.Flush(context.Background())
		case <-l.closeCh:
			return
		}
	}
}

// Weave API types

type weaveOp struct {
	Name string `json:"name"`
}

type weaveCallStart struct {
	Project    string         `json:"project_id"`
	Op         weaveOp        `json:"op_name"`
	StartedAt  time.Time      `json:"started_at"`
	Inputs     map[string]any `json:"inputs"`
	Attributes map[string]any `json:"attributes,omitempty"`
}

type weaveCallEnd struct {
	CallID    string          `json:"id"`
	EndedAt   time.Time       `json:"ended_at"`
	Output    any             `json:"output"`
	Exception *weaveException `json:"exception,omitempty"`
}

type weaveException struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

type weaveStreamRow struct {
	StreamName string         `json:"stream_name"`
	Project    string         `json:"project_id"`
	Timestamp  time.Time      `json:"timestamp"`
	Data       map[string]any `json:"data"`
}

func (l *Logger) startCall(ctx context.Context, call weaveCallStart) (string, error) {
	resp, err := l.doRequest(ctx, l.config.TraceEndpoint, "POST", "/call/start", call)
	if err != nil {
		return "", err
	}

	var result struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}
	return result.ID, nil
}

func (l *Logger) endCall(ctx context.Context, call weaveCallEnd) error {
	_, err := l.doRequest(ctx, l.config.TraceEndpoint, "POST", "/call/end", call)
	return err
}

func (l *Logger) logToStream(ctx context.Context, call integrations.POMLCall) error {
	row := weaveStreamRow{
		StreamName: l.config.StreamName,
		Project:    fmt.Sprintf("%s/%s", l.config.Entity, l.config.Project),
		Timestamp:  call.Timestamp,
		Data: map[string]any{
			"operation":        call.Operation,
			"format":           call.Format,
			"document_id":      call.DocumentID,
			"document_version": call.DocumentVersion,
			"duration_ms":      call.Duration.Milliseconds(),
			"trace_id":         call.TraceID,
			"span_id":          call.SpanID,
			"error":            call.Error,
			"success":          call.Error == "",
		},
	}
	for k, v := range call.Metadata {
		row.Data[k] = v
	}

	_, err := l.doRequest(ctx, l.config.APIEndpoint, "POST", "/stream/log", row)
	return err
}

func (l *Logger) doRequest(ctx context.Context, baseURL, method, path string, body any) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, baseURL+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+l.config.APIKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := l.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// Helper functions
func truncateString(s string, maxLen int) string {
	if maxLen <= 0 || len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

func truncateOutput(output any, maxLen int) any {
	if maxLen <= 0 {
		return output
	}
	data, err := json.Marshal(output)
	if err != nil || len(data) <= maxLen {
		return output
	}
	return truncateString(string(data), maxLen)
}
