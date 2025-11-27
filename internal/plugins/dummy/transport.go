package dummy

import (
	"fmt"
	"time"

	"github.com/atlas-foundry/poml-go-sdk/internal/plugins/contracts"
	"github.com/atlas-foundry/poml-go-sdk/internal/transport"
	"github.com/hashicorp/go-plugin"
)

// Transport is a noop transport plugin implementation for smoke tests.
type Transport struct{}

func (Transport) Dial(addr string) (string, error) {
	return fmt.Sprintf("dialed %s (noop)", addr), nil
}

func (Transport) Metrics() (transport.Metrics, error) {
	return transport.Metrics{OpenStreams: 0, LatencyP50: 1 * time.Millisecond, LatencyP99: 2 * time.Millisecond}, nil
}

func PluginMap() map[string]plugin.Plugin {
	return contracts.TransportPluginSet(Transport{})
}
