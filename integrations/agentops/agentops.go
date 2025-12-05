// Package agentops provides AgentOps integration for POML call logging.
//
// AgentOps is an observability platform for AI agents, providing real-time
// monitoring, debugging, and analytics for agent operations.
//
// Usage:
//
//	logger, err := agentops.New(agentops.Config{
//	    APIKey: "your-api-key",
//	    ProjectID: "your-project",
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
package agentops

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
	// DefaultAPIEndpoint is the AgentOps API endpoint.
	DefaultAPIEndpoint = "https://api.agentops.ai/v2"
)

// Config holds AgentOps-specific configuration.
type Config struct {
	integrations.Config
	// APIKey is the AgentOps API key (required).
	APIKey string `json:"api_key"`
	// APIEndpoint is the AgentOps API endpoint (optional, defaults to production).
	APIEndpoint string `json:"api_endpoint,omitempty"`
	// ProjectID is the AgentOps project ID.
	ProjectID string `json:"project_id,omitempty"`
	// SessionID groups related calls into a session.
	SessionID string `json:"session_id,omitempty"`
	// AgentID identifies the agent making the calls.
	AgentID string `json:"agent_id,omitempty"`
	// Tags are additional tags to add to each event.
	Tags []string `json:"tags,omitempty"`
	// HTTPClient is an optional custom HTTP client.
	HTTPClient *http.Client `json:"-"`
}

// DefaultConfig returns default AgentOps configuration.
func DefaultConfig() Config {
	return Config{
		Config:      integrations.DefaultConfig(),
		APIEndpoint: DefaultAPIEndpoint,
	}
}

// Logger implements POMLLogger for AgentOps.
type Logger struct {
	config    Config
	client    *http.Client
	sessionID string

	mu      sync.Mutex
	batch   []integrations.POMLCall
	closed  bool
	closeCh chan struct{}
	wg      sync.WaitGroup
}

// New creates a new AgentOps logger.
func New(cfg Config) (*Logger, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("agentops: api_key is required")
	}
	if cfg.APIEndpoint == "" {
		cfg.APIEndpoint = DefaultAPIEndpoint
	}

	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}

	l := &Logger{
		config:    cfg,
		client:    client,
		sessionID: cfg.SessionID,
		closeCh:   make(chan struct{}),
	}

	// Create session if not provided
	if l.sessionID == "" {
		sessionID, err := l.createSession(context.Background())
		if err != nil {
			return nil, fmt.Errorf("agentops: create session: %w", err)
		}
		l.sessionID = sessionID
	}

	// Start batch flusher if async
	if cfg.AsyncBatchSize > 0 && cfg.FlushInterval > 0 {
		l.wg.Add(1)
		go l.flushLoop()
	}

	return l, nil
}

// SessionID returns the current session ID.
func (l *Logger) SessionID() string {
	return l.sessionID
}

// LogCall logs a POML operation to AgentOps.
func (l *Logger) LogCall(ctx context.Context, call integrations.POMLCall) error {
	if !l.config.Enabled {
		return nil
	}

	l.mu.Lock()
	if l.closed {
		l.mu.Unlock()
		return fmt.Errorf("agentops: logger is closed")
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
	event := agentOpsEvent{
		EventType: "poml_call",
		Timestamp: call.Timestamp.Format(time.RFC3339Nano),
		SessionID: l.sessionID,
		AgentID:   l.config.AgentID,
		Data: map[string]any{
			"operation":        call.Operation,
			"format":           call.Format,
			"document_id":      call.DocumentID,
			"document_version": call.DocumentVersion,
			"duration_ms":      call.Duration.Milliseconds(),
			"trace_id":         call.TraceID,
			"span_id":          call.SpanID,
			"input":            call.Input,
			"output":           call.Output,
			"error":            call.Error,
			"metadata":         call.Metadata,
		},
		Tags: l.config.Tags,
	}

	return l.sendEvent(ctx, event)
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

// Close flushes, ends the session, and closes the logger.
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
	if err := l.Flush(context.Background()); err != nil {
		return err
	}
	return l.endSession(context.Background())
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

// AgentOps API types

type agentOpsEvent struct {
	EventType string         `json:"event_type"`
	Timestamp string         `json:"timestamp"`
	SessionID string         `json:"session_id"`
	AgentID   string         `json:"agent_id,omitempty"`
	Data      map[string]any `json:"data"`
	Tags      []string       `json:"tags,omitempty"`
}

type agentOpsSession struct {
	SessionID string         `json:"session_id"`
	ProjectID string         `json:"project_id,omitempty"`
	AgentID   string         `json:"agent_id,omitempty"`
	Tags      []string       `json:"tags,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

func (l *Logger) createSession(ctx context.Context) (string, error) {
	session := agentOpsSession{
		ProjectID: l.config.ProjectID,
		AgentID:   l.config.AgentID,
		Tags:      l.config.Tags,
		Metadata: map[string]any{
			"sdk":     "poml-go-sdk",
			"version": "0.0.8",
		},
	}

	resp, err := l.doRequest(ctx, "POST", "/sessions", session)
	if err != nil {
		return "", err
	}

	var result struct {
		SessionID string `json:"session_id"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}
	return result.SessionID, nil
}

func (l *Logger) endSession(ctx context.Context) error {
	body := map[string]any{
		"session_id": l.sessionID,
		"end_state":  "success",
	}
	_, err := l.doRequest(ctx, "POST", "/sessions/end", body)
	return err
}

func (l *Logger) sendEvent(ctx context.Context, event agentOpsEvent) error {
	_, err := l.doRequest(ctx, "POST", "/events", event)
	return err
}

func (l *Logger) doRequest(ctx context.Context, method, path string, body any) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, l.config.APIEndpoint+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("X-Agentops-Api-Key", l.config.APIKey)
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
