package poml

import "strings"

// isLikelyMIME does a light validation of mime-like strings: must contain one '/'
// and not start/end with it. This is intentionally loose to avoid strict coupling
// to a full registry.
func isLikelyMIME(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	if strings.HasPrefix(s, "/") || strings.HasSuffix(s, "/") {
		return false
	}
	if strings.Count(s, "/") != 1 {
		return false
	}
	return true
}
