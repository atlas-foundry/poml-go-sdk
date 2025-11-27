package transport

import (
	"context"
	"errors"
	"io"
	"time"
)

// Connection exposes a bidirectional stream plus metadata.
type Connection interface {
	io.ReadWriteCloser
	RemoteAddr() string
	LocalAddr() string
}

// Metrics captures lightweight transport stats; implementers can extend.
type Metrics struct {
	OpenStreams   int64
	BytesSent     int64
	BytesReceived int64
	LatencyP50    time.Duration
	LatencyP99    time.Duration
	PacketsSent   int64
	PacketsRecv   int64
	PacketsLost   int64
}

// Transport is the abstraction over H3/QUIC backends (quic-go, go-msquic).
// Implementations should be safe for concurrent use.
type Transport interface {
	Listen(ctx context.Context, addr string) (Listener, error)
	Dial(ctx context.Context, addr string) (Connection, error)
	Metrics(ctx context.Context) (Metrics, error)
}

// Listener accepts streams/connections.
type Listener interface {
	Accept(ctx context.Context) (Connection, error)
	Close() error
	Addr() string
}

// ErrNotImplemented is a helper for stub transports.
var ErrNotImplemented = errors.New("not implemented")
