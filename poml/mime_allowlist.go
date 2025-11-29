package poml

import "strings"

var allowedMediaMIMEs = DefaultAllowedMIMEs()

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
		// Images
		"image/png":     {},
		"image/jpeg":    {},
		"image/jpg":     {},
		"image/svg+xml": {},
		"image/gif":     {},
		"image/webp":    {},
		"image/tiff":    {},
		"image/heic":    {},
		"image/heif":    {},
		"image/avif":    {},
		// Audio
		"audio/mpeg":             {},
		"audio/mp3":              {},
		"audio/wav":              {},
		"audio/ogg":              {},
		"audio/ogg; codecs=opus": {},
		"audio/flac":             {},
		"audio/aac":              {},
		"audio/mp4":              {},
		"audio/opus":             {},
		"audio/webm":             {},
		// Video
		"video/mp4":        {},
		"video/webm":       {},
		"video/quicktime":  {},
		"video/mpeg":       {},
		"video/x-matroska": {},
		"video/x-msvideo":  {},
		"video/x-m4v":      {},
		"video/3gpp":       {},
		// Data
		"application/json":         {},
		"application/xml":          {},
		"application/octet-stream": {},
	}
}
