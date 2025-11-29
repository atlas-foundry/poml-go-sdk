package poml

import (
	"encoding/base64"
	"strings"
)

// estimateMediaBytes approximates decoded bytes for data URIs or inline bodies.
// Returns (size, true) when a meaningful estimate is available.
func estimateMediaBytes(src string, body string) (int64, bool) {
	src = strings.TrimSpace(src)
	body = strings.TrimSpace(body)
	if strings.HasPrefix(src, "data:") {
		parts := strings.SplitN(src, ",", 2)
		if len(parts) == 2 {
			payload := parts[1]
			size := int64(base64.StdEncoding.DecodedLen(len(payload)))
			return size, true
		}
	}
	if src != "" {
		return 0, false
	}
	if body != "" {
		return int64(len(body)), true
	}
	return 0, false
}
