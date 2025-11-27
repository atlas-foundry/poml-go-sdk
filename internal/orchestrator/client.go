package orchestrator

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"
)

// Message is the control-plane payload sent to the orchestrator.
type Message struct {
	Type    string `json:"type"`
	Source  string `json:"source,omitempty"`
	Payload string `json:"payload,omitempty"`
}

// Client maintains a control-plane connection to the orchestrator.
// It is intentionally minimal: send register/heartbeat/metrics/trace.
type Client struct {
	addr       string
	source     string
	hbInterval time.Duration

	mu   sync.Mutex
	conn net.Conn
	stop chan struct{}
}

// NewClient connects and registers with the orchestrator.
func NewClient(addr, source string, heartbeat time.Duration) (*Client, error) {
	if heartbeat <= 0 {
		heartbeat = 5 * time.Second
	}
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("dial orchestrator: %w", err)
	}
	c := &Client{
		addr:       addr,
		source:     source,
		hbInterval: heartbeat,
		conn:       conn,
		stop:       make(chan struct{}),
	}
	if err := c.send(Message{Type: "register", Source: source}); err != nil {
		_ = conn.Close()
		return nil, err
	}
	go c.heartbeat()
	go c.drainAcks()
	return c, nil
}

// Close terminates the connection.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	select {
	case <-c.stop:
	default:
		close(c.stop)
	}
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// SendMetrics ships a metrics payload (JSON string) to the orchestrator.
func (c *Client) SendMetrics(payload string) {
	_ = c.send(Message{Type: "metrics", Source: c.source, Payload: payload})
}

// SendTrace ships a sampled trace/qlog payload (opaque string).
func (c *Client) SendTrace(payload string) {
	_ = c.send(Message{Type: "trace", Source: c.source, Payload: payload})
}

func (c *Client) heartbeat() {
	t := time.NewTicker(c.hbInterval)
	defer t.Stop()
	for {
		select {
		case <-c.stop:
			return
		case <-t.C:
			_ = c.send(Message{Type: "heartbeat", Source: c.source})
		}
	}
}

func (c *Client) drainAcks() {
	sc := bufio.NewScanner(c.conn)
	for sc.Scan() {
		select {
		case <-c.stop:
			return
		default:
		}
	}
}

func (c *Client) send(msg Message) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn == nil {
		return fmt.Errorf("orchestrator client disconnected")
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	_, err = c.conn.Write(append(data, '\n'))
	return err
}
