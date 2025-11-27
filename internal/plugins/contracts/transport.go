package contracts

import (
	"net/rpc"

	"github.com/atlas-foundry/poml-go-sdk/internal/transport"
	"github.com/hashicorp/go-plugin"
)

const TransportPluginName = "transport"

// TransportService is the interface exposed over RPC.
type TransportService interface {
	Dial(addr string) (string, error)
	Metrics() (transport.Metrics, error)
}

// TransportPlugin wires TransportService into go-plugin using net/rpc.
type TransportPlugin struct {
	Impl TransportService
}

func (p *TransportPlugin) Server(*plugin.MuxBroker) (interface{}, error) {
	return &transportRPCServer{Impl: p.Impl}, nil
}

func (p *TransportPlugin) Client(_ *plugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &transportRPCClient{client: c}, nil
}

type transportRPCServer struct {
	Impl TransportService
}

func (s *transportRPCServer) Dial(args string, resp *string) error {
	out, err := s.Impl.Dial(args)
	if err != nil {
		return err
	}
	*resp = out
	return nil
}

func (s *transportRPCServer) Metrics(_ struct{}, resp *transport.Metrics) error {
	out, err := s.Impl.Metrics()
	if err != nil {
		return err
	}
	*resp = out
	return nil
}

type transportRPCClient struct {
	client *rpc.Client
}

func (c *transportRPCClient) Dial(addr string) (string, error) {
	var resp string
	err := c.client.Call("Plugin.Dial", addr, &resp)
	return resp, err
}

func (c *transportRPCClient) Metrics() (transport.Metrics, error) {
	var resp transport.Metrics
	err := c.client.Call("Plugin.Metrics", struct{}{}, &resp)
	return resp, err
}

// Helper to declare plugin map.
func TransportPluginSet(impl TransportService) map[string]plugin.Plugin {
	return map[string]plugin.Plugin{
		TransportPluginName: &TransportPlugin{Impl: impl},
	}
}
