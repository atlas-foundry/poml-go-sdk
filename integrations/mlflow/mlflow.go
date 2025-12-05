// Package mlflow provides MLflow integration for POML call logging.
//
// MLflow is an open source platform for managing the end-to-end machine learning lifecycle.
// This integration logs POML operations as MLflow runs with metrics and parameters.
//
// Usage:
//
//	logger, err := mlflow.New(mlflow.Config{
//	    TrackingURI: "http://localhost:5000",
//	    ExperimentName: "poml-prompts",
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
package mlflow

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

// Config holds MLflow-specific configuration.
type Config struct {
	integrations.Config
	// TrackingURI is the MLflow tracking server URI (e.g., "http://localhost:5000").
	TrackingURI string `json:"tracking_uri"`
	// ExperimentName is the MLflow experiment to log to.
	ExperimentName string `json:"experiment_name"`
	// ExperimentID overrides ExperimentName if set.
	ExperimentID string `json:"experiment_id,omitempty"`
	// RunName is an optional name for the run.
	RunName string `json:"run_name,omitempty"`
	// Tags are additional tags to add to each run.
	Tags map[string]string `json:"tags,omitempty"`
	// HTTPClient is an optional custom HTTP client.
	HTTPClient *http.Client `json:"-"`
}

// DefaultConfig returns default MLflow configuration.
func DefaultConfig() Config {
	return Config{
		Config:         integrations.DefaultConfig(),
		TrackingURI:    "http://localhost:5000",
		ExperimentName: "poml-prompts",
	}
}

// Logger implements POMLLogger for MLflow.
type Logger struct {
	config       Config
	client       *http.Client
	experimentID string

	mu      sync.Mutex
	batch   []integrations.POMLCall
	closed  bool
	closeCh chan struct{}
	wg      sync.WaitGroup
}

// New creates a new MLflow logger.
func New(cfg Config) (*Logger, error) {
	if cfg.TrackingURI == "" {
		return nil, fmt.Errorf("mlflow: tracking_uri is required")
	}
	if cfg.ExperimentName == "" && cfg.ExperimentID == "" {
		return nil, fmt.Errorf("mlflow: experiment_name or experiment_id is required")
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

	// Get or create experiment
	if cfg.ExperimentID != "" {
		l.experimentID = cfg.ExperimentID
	} else {
		expID, err := l.getOrCreateExperiment(context.Background(), cfg.ExperimentName)
		if err != nil {
			return nil, fmt.Errorf("mlflow: get/create experiment: %w", err)
		}
		l.experimentID = expID
	}

	// Start batch flusher if async
	if cfg.AsyncBatchSize > 0 && cfg.FlushInterval > 0 {
		l.wg.Add(1)
		go l.flushLoop()
	}

	return l, nil
}

// LogCall logs a POML operation to MLflow.
func (l *Logger) LogCall(ctx context.Context, call integrations.POMLCall) error {
	if !l.config.Enabled {
		return nil
	}

	l.mu.Lock()
	if l.closed {
		l.mu.Unlock()
		return fmt.Errorf("mlflow: logger is closed")
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
	// Create a new run
	runID, err := l.createRun(ctx, call)
	if err != nil {
		return fmt.Errorf("mlflow: create run: %w", err)
	}

	// Log metrics
	metrics := []mlflowMetric{
		{Key: "duration_ms", Value: float64(call.Duration.Milliseconds()), Timestamp: call.Timestamp.UnixMilli()},
	}
	if err := l.logMetrics(ctx, runID, metrics); err != nil {
		return fmt.Errorf("mlflow: log metrics: %w", err)
	}

	// Log parameters
	params := []mlflowParam{
		{Key: "operation", Value: call.Operation},
		{Key: "document_id", Value: call.DocumentID},
	}
	if call.Format != "" {
		params = append(params, mlflowParam{Key: "format", Value: call.Format})
	}
	if call.DocumentVersion != "" {
		params = append(params, mlflowParam{Key: "document_version", Value: call.DocumentVersion})
	}
	if call.TraceID != "" {
		params = append(params, mlflowParam{Key: "trace_id", Value: call.TraceID})
	}
	if call.Error != "" {
		params = append(params, mlflowParam{Key: "error", Value: call.Error})
	}
	if err := l.logParams(ctx, runID, params); err != nil {
		return fmt.Errorf("mlflow: log params: %w", err)
	}

	// End the run
	status := "FINISHED"
	if call.Error != "" {
		status = "FAILED"
	}
	if err := l.endRun(ctx, runID, status); err != nil {
		return fmt.Errorf("mlflow: end run: %w", err)
	}

	return nil
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

// MLflow REST API types

type mlflowMetric struct {
	Key       string  `json:"key"`
	Value     float64 `json:"value"`
	Timestamp int64   `json:"timestamp"`
}

type mlflowParam struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type mlflowTag struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func (l *Logger) getOrCreateExperiment(ctx context.Context, name string) (string, error) {
	// Try to get existing experiment
	resp, err := l.doRequest(ctx, "GET", "/api/2.0/mlflow/experiments/get-by-name?experiment_name="+name, nil)
	if err == nil {
		var result struct {
			Experiment struct {
				ExperimentID string `json:"experiment_id"`
			} `json:"experiment"`
		}
		if err := json.Unmarshal(resp, &result); err == nil && result.Experiment.ExperimentID != "" {
			return result.Experiment.ExperimentID, nil
		}
	}

	// Create new experiment
	body := map[string]string{"name": name}
	resp, err = l.doRequest(ctx, "POST", "/api/2.0/mlflow/experiments/create", body)
	if err != nil {
		return "", err
	}

	var result struct {
		ExperimentID string `json:"experiment_id"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}
	return result.ExperimentID, nil
}

func (l *Logger) createRun(ctx context.Context, call integrations.POMLCall) (string, error) {
	tags := []mlflowTag{
		{Key: "poml.operation", Value: call.Operation},
		{Key: "poml.document_id", Value: call.DocumentID},
	}
	if call.Format != "" {
		tags = append(tags, mlflowTag{Key: "poml.format", Value: call.Format})
	}
	for k, v := range l.config.Tags {
		tags = append(tags, mlflowTag{Key: k, Value: v})
	}

	body := map[string]any{
		"experiment_id": l.experimentID,
		"start_time":    call.Timestamp.UnixMilli(),
		"tags":          tags,
	}
	if l.config.RunName != "" {
		body["run_name"] = l.config.RunName
	}

	resp, err := l.doRequest(ctx, "POST", "/api/2.0/mlflow/runs/create", body)
	if err != nil {
		return "", err
	}

	var result struct {
		Run struct {
			Info struct {
				RunID string `json:"run_id"`
			} `json:"info"`
		} `json:"run"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}
	return result.Run.Info.RunID, nil
}

func (l *Logger) logMetrics(ctx context.Context, runID string, metrics []mlflowMetric) error {
	body := map[string]any{
		"run_id":  runID,
		"metrics": metrics,
	}
	_, err := l.doRequest(ctx, "POST", "/api/2.0/mlflow/runs/log-batch", body)
	return err
}

func (l *Logger) logParams(ctx context.Context, runID string, params []mlflowParam) error {
	body := map[string]any{
		"run_id": runID,
		"params": params,
	}
	_, err := l.doRequest(ctx, "POST", "/api/2.0/mlflow/runs/log-batch", body)
	return err
}

func (l *Logger) endRun(ctx context.Context, runID string, status string) error {
	body := map[string]any{
		"run_id":   runID,
		"status":   status,
		"end_time": time.Now().UnixMilli(),
	}
	_, err := l.doRequest(ctx, "POST", "/api/2.0/mlflow/runs/update", body)
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

	req, err := http.NewRequestWithContext(ctx, method, l.config.TrackingURI+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
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
