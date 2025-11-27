package contracts

import "github.com/hashicorp/go-plugin"

// Shared handshake for all plugins.
var Handshake = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "DROP_SHIP_PLUGIN",
	MagicCookieValue: "enabled",
}
