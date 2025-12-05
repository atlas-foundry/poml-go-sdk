package dummy

import (
	"testing"
	"time"

	"github.com/atlas-foundry/poml-go-sdk/internal/plugins/contracts"
)

func TestMeshLoggerPushConfig(t *testing.T) {
	logger := MeshLogger{}
	err := logger.PushConfig("test config")
	if err != nil {
		t.Errorf("PushConfig returned error: %v", err)
	}
}

func TestMeshLoggerLogEvent(t *testing.T) {
	logger := MeshLogger{}
	err := logger.LogEvent("test event")
	if err != nil {
		t.Errorf("LogEvent returned error: %v", err)
	}
}

func TestMeshPluginMap(t *testing.T) {
	pm := MeshPluginMap()
	if len(pm) != 1 {
		t.Fatalf("expected 1 plugin, got %d", len(pm))
	}
	if _, ok := pm[contracts.MeshPluginName]; !ok {
		t.Errorf("expected mesh plugin in map")
	}
}

func TestTransportDial(t *testing.T) {
	tr := Transport{}
	resp, err := tr.Dial("localhost:8080")
	if err != nil {
		t.Errorf("Dial returned error: %v", err)
	}
	expected := "dialed localhost:8080 (noop)"
	if resp != expected {
		t.Errorf("expected %q, got %q", expected, resp)
	}
}

func TestTransportMetrics(t *testing.T) {
	tr := Transport{}
	metrics, err := tr.Metrics()
	if err != nil {
		t.Errorf("Metrics returned error: %v", err)
	}
	if metrics.OpenStreams != 0 {
		t.Errorf("expected OpenStreams=0, got %d", metrics.OpenStreams)
	}
	if metrics.LatencyP50 != 1*time.Millisecond {
		t.Errorf("expected LatencyP50=1ms, got %v", metrics.LatencyP50)
	}
	if metrics.LatencyP99 != 2*time.Millisecond {
		t.Errorf("expected LatencyP99=2ms, got %v", metrics.LatencyP99)
	}
}

func TestPluginMap(t *testing.T) {
	pm := PluginMap()
	if len(pm) != 1 {
		t.Fatalf("expected 1 plugin, got %d", len(pm))
	}
	if _, ok := pm[contracts.TransportPluginName]; !ok {
		t.Errorf("expected transport plugin in map")
	}
}
