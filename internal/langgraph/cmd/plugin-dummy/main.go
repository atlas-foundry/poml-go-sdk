package main

import (
	"github.com/atlas-foundry/poml-go-sdk/internal/plugins/contracts"
	"github.com/atlas-foundry/poml-go-sdk/internal/plugins/dummy"
	"github.com/hashicorp/go-plugin"
)

func main() {
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: contracts.Handshake,
		Plugins:         mergePluginMaps(dummy.PluginMap(), dummy.MeshPluginMap()),
	})
}

func mergePluginMaps(maps ...map[string]plugin.Plugin) map[string]plugin.Plugin {
	merged := make(map[string]plugin.Plugin, len(maps))
	for _, m := range maps {
		for name, plug := range m {
			merged[name] = plug
		}
	}
	return merged
}
