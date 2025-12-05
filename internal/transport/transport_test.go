package transport

import (
	"testing"
)

func TestErrNotImplemented(t *testing.T) {
	if ErrNotImplemented == nil {
		t.Fatal("ErrNotImplemented should not be nil")
	}
	if ErrNotImplemented.Error() != "not implemented" {
		t.Errorf("unexpected error message: %s", ErrNotImplemented.Error())
	}
}

func TestMetricsZeroValue(t *testing.T) {
	var m Metrics
	if m.OpenStreams != 0 {
		t.Errorf("expected zero OpenStreams")
	}
	if m.BytesSent != 0 {
		t.Errorf("expected zero BytesSent")
	}
	if m.BytesReceived != 0 {
		t.Errorf("expected zero BytesReceived")
	}
	if m.LatencyP50 != 0 {
		t.Errorf("expected zero LatencyP50")
	}
	if m.LatencyP99 != 0 {
		t.Errorf("expected zero LatencyP99")
	}
	if m.PacketsSent != 0 {
		t.Errorf("expected zero PacketsSent")
	}
	if m.PacketsRecv != 0 {
		t.Errorf("expected zero PacketsRecv")
	}
	if m.PacketsLost != 0 {
		t.Errorf("expected zero PacketsLost")
	}
}
