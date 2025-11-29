package poml

import "strings"

func isAllowedAttr(name string, allowed []string) bool {
	name = strings.TrimSpace(strings.ToLower(name))
	for _, a := range allowed {
		if name == strings.ToLower(a) {
			return true
		}
	}
	return false
}
