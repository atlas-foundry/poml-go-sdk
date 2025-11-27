package contracts

import (
	"net/rpc"

	"github.com/hashicorp/go-plugin"
)

const MeshPluginName = "mesh"

// MeshService allows logging or pushing mesh configs; kept simple for stubs.
type MeshService interface {
	PushConfig(doc string) error
	LogEvent(msg string) error
}

type MeshPlugin struct {
	Impl MeshService
}

func (p *MeshPlugin) Server(*plugin.MuxBroker) (interface{}, error) {
	return &meshRPCServer{Impl: p.Impl}, nil
}

func (p *MeshPlugin) Client(_ *plugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &meshRPCClient{client: c}, nil
}

type meshRPCServer struct {
	Impl MeshService
}

func (s *meshRPCServer) PushConfig(doc string, _ *struct{}) error {
	return s.Impl.PushConfig(doc)
}

func (s *meshRPCServer) LogEvent(msg string, _ *struct{}) error {
	return s.Impl.LogEvent(msg)
}

type meshRPCClient struct {
	client *rpc.Client
}

func (c *meshRPCClient) PushConfig(doc string) error {
	return c.client.Call("Plugin.PushConfig", doc, &struct{}{})
}

func (c *meshRPCClient) LogEvent(msg string) error {
	return c.client.Call("Plugin.LogEvent", msg, &struct{}{})
}

func MeshPluginSet(impl MeshService) map[string]plugin.Plugin {
	return map[string]plugin.Plugin{
		MeshPluginName: &MeshPlugin{Impl: impl},
	}
}
