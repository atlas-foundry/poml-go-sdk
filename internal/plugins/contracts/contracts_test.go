package contracts

import (
	"errors"
	"testing"
	"time"

	"github.com/atlas-foundry/poml-go-sdk/internal/transport"
)

// Mock implementations for testing

type mockMeshService struct {
	pushErr   error
	logErr    error
	pushCalls []string
	logCalls  []string
}

func (m *mockMeshService) PushConfig(doc string) error {
	m.pushCalls = append(m.pushCalls, doc)
	return m.pushErr
}

func (m *mockMeshService) LogEvent(msg string) error {
	m.logCalls = append(m.logCalls, msg)
	return m.logErr
}

type mockObsService struct {
	metricErr   error
	logErr      error
	metricCalls []Metric
	logCalls    []struct {
		msg string
		kv  map[string]string
	}
}

func (m *mockObsService) EmitMetric(metric Metric) error {
	m.metricCalls = append(m.metricCalls, metric)
	return m.metricErr
}

func (m *mockObsService) EmitLog(msg string, kv map[string]string) error {
	m.logCalls = append(m.logCalls, struct {
		msg string
		kv  map[string]string
	}{msg, kv})
	return m.logErr
}

type mockTransportService struct {
	dialResp   string
	dialErr    error
	metricsVal transport.Metrics
	metricsErr error
}

func (m *mockTransportService) Dial(addr string) (string, error) {
	return m.dialResp, m.dialErr
}

func (m *mockTransportService) Metrics() (transport.Metrics, error) {
	return m.metricsVal, m.metricsErr
}

// Tests

func TestHandshakeConfig(t *testing.T) {
	if Handshake.ProtocolVersion != 1 {
		t.Errorf("expected protocol version 1, got %d", Handshake.ProtocolVersion)
	}
	if Handshake.MagicCookieKey != "DROP_SHIP_PLUGIN" {
		t.Errorf("unexpected magic cookie key: %s", Handshake.MagicCookieKey)
	}
	if Handshake.MagicCookieValue != "enabled" {
		t.Errorf("unexpected magic cookie value: %s", Handshake.MagicCookieValue)
	}
}

func TestMeshPluginServer(t *testing.T) {
	mock := &mockMeshService{}
	plugin := &MeshPlugin{Impl: mock}

	server, err := plugin.Server(nil)
	if err != nil {
		t.Fatalf("Server() error: %v", err)
	}
	if server == nil {
		t.Fatal("Server() returned nil")
	}

	rpcServer, ok := server.(*meshRPCServer)
	if !ok {
		t.Fatalf("expected *meshRPCServer, got %T", server)
	}

	// Test PushConfig through RPC server
	err = rpcServer.PushConfig("test-config", &struct{}{})
	if err != nil {
		t.Errorf("PushConfig error: %v", err)
	}
	if len(mock.pushCalls) != 1 || mock.pushCalls[0] != "test-config" {
		t.Errorf("PushConfig not called correctly: %v", mock.pushCalls)
	}

	// Test LogEvent through RPC server
	err = rpcServer.LogEvent("test-event", &struct{}{})
	if err != nil {
		t.Errorf("LogEvent error: %v", err)
	}
	if len(mock.logCalls) != 1 || mock.logCalls[0] != "test-event" {
		t.Errorf("LogEvent not called correctly: %v", mock.logCalls)
	}
}

func TestMeshPluginServerErrors(t *testing.T) {
	expectedErr := errors.New("mock error")
	mock := &mockMeshService{pushErr: expectedErr, logErr: expectedErr}
	plugin := &MeshPlugin{Impl: mock}

	server, _ := plugin.Server(nil)
	rpcServer := server.(*meshRPCServer)

	if err := rpcServer.PushConfig("x", &struct{}{}); err != expectedErr {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}
	if err := rpcServer.LogEvent("x", &struct{}{}); err != expectedErr {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}
}

func TestMeshPluginSet(t *testing.T) {
	mock := &mockMeshService{}
	set := MeshPluginSet(mock)

	if len(set) != 1 {
		t.Fatalf("expected 1 plugin, got %d", len(set))
	}
	if _, ok := set[MeshPluginName]; !ok {
		t.Errorf("expected plugin named %q", MeshPluginName)
	}
}

func TestObservabilityPluginServer(t *testing.T) {
	mock := &mockObsService{}
	plugin := &ObservabilityPlugin{Impl: mock}

	server, err := plugin.Server(nil)
	if err != nil {
		t.Fatalf("Server() error: %v", err)
	}

	rpcServer := server.(*obsRPCServer)

	// Test EmitMetric
	metric := Metric{Name: "test", Value: 42.0, Labels: map[string]string{"env": "test"}}
	err = rpcServer.EmitMetric(metric, &struct{}{})
	if err != nil {
		t.Errorf("EmitMetric error: %v", err)
	}
	if len(mock.metricCalls) != 1 || mock.metricCalls[0].Name != "test" {
		t.Errorf("EmitMetric not called correctly")
	}

	// Test EmitLog
	args := map[string]interface{}{"msg": "hello", "labels": map[string]string{"k": "v"}}
	err = rpcServer.EmitLog(args, &struct{}{})
	if err != nil {
		t.Errorf("EmitLog error: %v", err)
	}
	if len(mock.logCalls) != 1 || mock.logCalls[0].msg != "hello" {
		t.Errorf("EmitLog not called correctly")
	}
}

func TestObservabilityPluginServerEmitLogNoLabels(t *testing.T) {
	mock := &mockObsService{}
	plugin := &ObservabilityPlugin{Impl: mock}
	server, _ := plugin.Server(nil)
	rpcServer := server.(*obsRPCServer)

	// Test EmitLog without labels key
	args := map[string]interface{}{"msg": "test"}
	err := rpcServer.EmitLog(args, &struct{}{})
	if err != nil {
		t.Errorf("EmitLog error: %v", err)
	}
	if len(mock.logCalls) != 1 {
		t.Errorf("EmitLog not called")
	}
}

func TestObservabilityPluginSet(t *testing.T) {
	mock := &mockObsService{}
	set := ObservabilityPluginSet(mock)

	if len(set) != 1 {
		t.Fatalf("expected 1 plugin, got %d", len(set))
	}
	if _, ok := set[ObservabilityPluginName]; !ok {
		t.Errorf("expected plugin named %q", ObservabilityPluginName)
	}
}

func TestTransportPluginServer(t *testing.T) {
	mock := &mockTransportService{
		dialResp:   "connected",
		metricsVal: transport.Metrics{OpenStreams: 5, LatencyP50: time.Millisecond},
	}
	plugin := &TransportPlugin{Impl: mock}

	server, err := plugin.Server(nil)
	if err != nil {
		t.Fatalf("Server() error: %v", err)
	}

	rpcServer := server.(*transportRPCServer)

	// Test Dial
	var resp string
	err = rpcServer.Dial("localhost:8080", &resp)
	if err != nil {
		t.Errorf("Dial error: %v", err)
	}
	if resp != "connected" {
		t.Errorf("expected 'connected', got %q", resp)
	}

	// Test Metrics
	var metrics transport.Metrics
	err = rpcServer.Metrics(struct{}{}, &metrics)
	if err != nil {
		t.Errorf("Metrics error: %v", err)
	}
	if metrics.OpenStreams != 5 {
		t.Errorf("expected OpenStreams=5, got %d", metrics.OpenStreams)
	}
}

func TestTransportPluginServerErrors(t *testing.T) {
	expectedErr := errors.New("dial error")
	mock := &mockTransportService{dialErr: expectedErr, metricsErr: expectedErr}
	plugin := &TransportPlugin{Impl: mock}

	server, _ := plugin.Server(nil)
	rpcServer := server.(*transportRPCServer)

	var resp string
	if err := rpcServer.Dial("x", &resp); err != expectedErr {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}

	var metrics transport.Metrics
	if err := rpcServer.Metrics(struct{}{}, &metrics); err != expectedErr {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}
}

func TestTransportPluginSet(t *testing.T) {
	mock := &mockTransportService{}
	set := TransportPluginSet(mock)

	if len(set) != 1 {
		t.Fatalf("expected 1 plugin, got %d", len(set))
	}
	if _, ok := set[TransportPluginName]; !ok {
		t.Errorf("expected plugin named %q", TransportPluginName)
	}
}

func TestPluginConstants(t *testing.T) {
	if MeshPluginName != "mesh" {
		t.Errorf("unexpected MeshPluginName: %s", MeshPluginName)
	}
	if ObservabilityPluginName != "observability" {
		t.Errorf("unexpected ObservabilityPluginName: %s", ObservabilityPluginName)
	}
	if TransportPluginName != "transport" {
		t.Errorf("unexpected TransportPluginName: %s", TransportPluginName)
	}
}

// Test Plugin.Client methods - these return a client wrapper
func TestMeshPluginClient(t *testing.T) {
	mock := &mockMeshService{}
	plugin := &MeshPlugin{Impl: mock}

	// Client() returns a wrapper that forwards calls via RPC
	// Since we don't have a real RPC client, we test that the method exists
	// and returns a non-nil value
	client, err := plugin.Client(nil, nil)
	if err != nil {
		t.Fatalf("Client() error: %v", err)
	}
	if client == nil {
		t.Fatal("Client() returned nil")
	}
	// Verify it's the right type
	_, ok := client.(*meshRPCClient)
	if !ok {
		t.Fatalf("expected *meshRPCClient, got %T", client)
	}
}

func TestObservabilityPluginClient(t *testing.T) {
	mock := &mockObsService{}
	plugin := &ObservabilityPlugin{Impl: mock}

	client, err := plugin.Client(nil, nil)
	if err != nil {
		t.Fatalf("Client() error: %v", err)
	}
	if client == nil {
		t.Fatal("Client() returned nil")
	}
	_, ok := client.(*obsRPCClient)
	if !ok {
		t.Fatalf("expected *obsRPCClient, got %T", client)
	}
}

func TestTransportPluginClient(t *testing.T) {
	mock := &mockTransportService{}
	plugin := &TransportPlugin{Impl: mock}

	client, err := plugin.Client(nil, nil)
	if err != nil {
		t.Fatalf("Client() error: %v", err)
	}
	if client == nil {
		t.Fatal("Client() returned nil")
	}
	_, ok := client.(*transportRPCClient)
	if !ok {
		t.Fatalf("expected *transportRPCClient, got %T", client)
	}
}
