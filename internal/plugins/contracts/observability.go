package contracts

import (
	"net/rpc"

	"github.com/hashicorp/go-plugin"
)

const ObservabilityPluginName = "observability"

type Metric struct {
	Name   string
	Value  float64
	Labels map[string]string
}

// ObservabilityService is a stub for OTEL/metrics exporters.
type ObservabilityService interface {
	EmitMetric(Metric) error
	EmitLog(msg string, kv map[string]string) error
}

type ObservabilityPlugin struct {
	Impl ObservabilityService
}

func (p *ObservabilityPlugin) Server(*plugin.MuxBroker) (interface{}, error) {
	return &obsRPCServer{Impl: p.Impl}, nil
}

func (p *ObservabilityPlugin) Client(_ *plugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &obsRPCClient{client: c}, nil
}

type obsRPCServer struct {
	Impl ObservabilityService
}

func (s *obsRPCServer) EmitMetric(m Metric, _ *struct{}) error {
	return s.Impl.EmitMetric(m)
}

func (s *obsRPCServer) EmitLog(args map[string]interface{}, _ *struct{}) error {
	msg, _ := args["msg"].(string)
	labels := map[string]string{}
	if raw, ok := args["labels"].(map[string]string); ok {
		labels = raw
	}
	return s.Impl.EmitLog(msg, labels)
}

type obsRPCClient struct {
	client *rpc.Client
}

func (c *obsRPCClient) EmitMetric(m Metric) error {
	return c.client.Call("Plugin.EmitMetric", m, &struct{}{})
}

func (c *obsRPCClient) EmitLog(msg string, kv map[string]string) error {
	args := map[string]interface{}{"msg": msg, "labels": kv}
	return c.client.Call("Plugin.EmitLog", args, &struct{}{})
}

func ObservabilityPluginSet(impl ObservabilityService) map[string]plugin.Plugin {
	return map[string]plugin.Plugin{
		ObservabilityPluginName: &ObservabilityPlugin{Impl: impl},
	}
}
