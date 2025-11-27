package langgraph

import (
	"log"
	"os"
	"os/exec"

	"github.com/atlas-foundry/poml-go-sdk/internal/plugins/contracts"
	"github.com/hashicorp/go-plugin"
)

// Executor loads plugins and can dispatch stub graph steps.
type Executor struct {
	Transport contracts.TransportService
	Mesh      contracts.MeshService
	client    *plugin.Client
}

// LoadTransport starts a plugin binary and returns an executor with a transport client wired.
func LoadTransport(path string) (*Executor, error) {
	clearReattachEnv()
	client := plugin.NewClient(&plugin.ClientConfig{
		Cmd:              exec.Command(path),
		HandshakeConfig:  contracts.Handshake,
		Plugins:          map[string]plugin.Plugin{contracts.TransportPluginName: &contracts.TransportPlugin{}},
		AllowedProtocols: []plugin.Protocol{plugin.ProtocolNetRPC},
	})
	rpcClient, err := client.Client()
	if err != nil {
		client.Kill()
		return nil, err
	}
	raw, err := rpcClient.Dispense(contracts.TransportPluginName)
	if err != nil {
		client.Kill()
		return nil, err
	}
	transportSvc := raw.(contracts.TransportService)
	return &Executor{Transport: transportSvc, client: client}, nil
}

// LoadMeshLogger starts a mesh plugin that only logs events.
func (e *Executor) LoadMeshLogger(path string) error {
	clearReattachEnv()
	client := plugin.NewClient(&plugin.ClientConfig{
		Cmd:              exec.Command(path),
		HandshakeConfig:  contracts.Handshake,
		Plugins:          map[string]plugin.Plugin{contracts.MeshPluginName: &contracts.MeshPlugin{}},
		AllowedProtocols: []plugin.Protocol{plugin.ProtocolNetRPC},
	})
	rpcClient, err := client.Client()
	if err != nil {
		client.Kill()
		return err
	}
	raw, err := rpcClient.Dispense(contracts.MeshPluginName)
	if err != nil {
		client.Kill()
		return err
	}
	e.Mesh = raw.(contracts.MeshService)
	e.client = client
	return nil
}

// ExecuteTransportPing is a stub LangGraph node executor that pings a transport plugin.
func (e *Executor) ExecuteTransportPing(addr string) error {
	if e.Transport == nil {
		return nil
	}
	out, err := e.Transport.Dial(addr)
	if err != nil {
		return err
	}
	log.Printf("[langgraph] transport dial result: %s", out)
	if e.Mesh != nil {
		_ = e.Mesh.LogEvent("transport dialed " + addr)
	}
	return nil
}

// Close shuts down any plugin client.
func (e *Executor) Close() {
	if e.client != nil {
		e.client.Kill()
	}
}

func clearReattachEnv() {
	_ = os.Unsetenv("PLUGIN_REATTACH_CONFIG")
	_ = os.Unsetenv("PLUGIN_REATTACH_PROVIDERS")
}
