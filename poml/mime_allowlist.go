package poml

import "strings"

var allowedMediaMIMEs = DefaultAllowedMIMEs()

// allowMIME reports whether the syntax passes basic validation and optional allowlist.
func allowMIME(s string) bool {
	return allowMIMEWithList(s, allowedMediaMIMEs)
}

func allowMIMEWithList(s string, allow map[string]struct{}) bool {
	if allow == nil {
		allow = allowedMediaMIMEs
	}
	s = normalizeMIME(s)
	if !isLikelyMIME(s) {
		return false
	}
	if len(allow) == 0 {
		return true
	}
	_, ok := allow[s]
	return ok
}

func normalizeMIME(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// DefaultAllowedMIMEs returns a fresh copy of the strict allowlist.
func DefaultAllowedMIMEs() map[string]struct{} {
	return map[string]struct{}{
		"image/png":                {},
		"image/jpeg":               {},
		"image/jpg":                {},
		"image/svg+xml":            {},
		"image/gif":                {},
		"image/webp":               {},
		"audio/mpeg":               {},
		"audio/mp3":                {},
		"audio/wav":                {},
		"audio/ogg":                {},
		"video/mp4":                {},
		"video/webm":               {},
		"application/json":         {},
		"application/xml":          {},
		"application/octet-stream": {},
	}
}
