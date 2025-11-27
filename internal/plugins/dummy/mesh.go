package dummy

import (
	"log"

	"github.com/atlas-foundry/poml-go-sdk/internal/plugins/contracts"
	"github.com/hashicorp/go-plugin"
)

// MeshLogger logs config/events for testing the plugin wiring.
type MeshLogger struct{}

func (MeshLogger) PushConfig(doc string) error {
	log.Printf("[mesh-plugin] push config:\n%s", doc)
	return nil
}

func (MeshLogger) LogEvent(msg string) error {
	log.Printf("[mesh-plugin] event: %s", msg)
	return nil
}

func MeshPluginMap() map[string]plugin.Plugin {
	return contracts.MeshPluginSet(MeshLogger{})
}
