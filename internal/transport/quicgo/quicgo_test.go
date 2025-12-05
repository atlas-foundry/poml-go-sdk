package quicgo

import (
	"context"
	"testing"

	"github.com/atlas-foundry/poml-go-sdk/internal/transport"
)

func TestTransportListen(t *testing.T) {
	tr := Transport{}
	listener, err := tr.Listen(context.Background(), "localhost:0")
	if err != transport.ErrNotImplemented {
		t.Errorf("expected ErrNotImplemented, got %v", err)
	}
	if listener != nil {
		t.Errorf("expected nil listener")
	}
}

func TestTransportDial(t *testing.T) {
	tr := Transport{}
	conn, err := tr.Dial(context.Background(), "localhost:8080")
	if err != transport.ErrNotImplemented {
		t.Errorf("expected ErrNotImplemented, got %v", err)
	}
	if conn != nil {
		t.Errorf("expected nil connection")
	}
}

func TestTransportMetrics(t *testing.T) {
	tr := Transport{}
	metrics, err := tr.Metrics(context.Background())
	if err != nil {
		t.Errorf("Metrics returned error: %v", err)
	}
	// Stub returns zero-value metrics
	if metrics.OpenStreams != 0 {
		t.Errorf("expected zero OpenStreams")
	}
}

func TestTransportImplementsInterface(t *testing.T) {
	var _ transport.Transport = Transport{}
}
