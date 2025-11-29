package poml

import "strings"

// AllowedOpKinds lists suggested op kinds from the proposal; strict validation can check membership when non-empty.
var AllowedOpKinds = []string{
	"builtin",
	"custom",
	"tool",
	"function",
}

func allowOpKind(kind string, allow []string) bool {
	if len(allow) == 0 {
		return true
	}
	kind = strings.TrimSpace(strings.ToLower(kind))
	for _, k := range allow {
		if strings.TrimSpace(strings.ToLower(k)) == kind {
			return true
		}
	}
	return false
}
