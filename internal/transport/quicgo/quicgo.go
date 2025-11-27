package quicgo

import (
	"context"

	"github.com/atlas-foundry/poml-go-sdk/internal/transport"
)

// Transport is a placeholder quic-go-backed transport.
// TODO: wire real quic-go implementation.
type Transport struct{}

func (Transport) Listen(ctx context.Context, addr string) (transport.Listener, error) {
	return nil, transport.ErrNotImplemented
}

func (Transport) Dial(ctx context.Context, addr string) (transport.Connection, error) {
	return nil, transport.ErrNotImplemented
}

func (Transport) Metrics(ctx context.Context) (transport.Metrics, error) {
	return transport.Metrics{}, nil
}
