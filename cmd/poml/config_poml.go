package main

import (
	"encoding/json"
	"strings"

	"github.com/atlas-foundry/poml-go-sdk/poml"
)

// extractListFromConfig pulls a JSON array list from <object name="key">...</object> bodies.
func extractListFromConfig(doc poml.Document, key string) []string {
	var values []string
	for _, obj := range doc.Objects {
		name := ""
		for _, a := range obj.Attrs {
			if a.Name.Local == "name" {
				name = strings.TrimSpace(a.Value)
				break
			}
		}
		if name != key {
			continue
		}
		body := strings.TrimSpace(obj.Body)
		if body == "" {
			body = strings.TrimSpace(obj.Data)
		}
		if body == "" {
			continue
		}
		var list []string
		if err := json.Unmarshal([]byte(body), &list); err == nil {
			values = append(values, list...)
		}
	}
	return values
}
