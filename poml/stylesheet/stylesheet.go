// Package stylesheet provides CSS-like styling for POML documents.
package stylesheet

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// Stylesheet represents a parsed stylesheet with rules.
type Stylesheet struct {
	Rules []Rule
}

// Rule represents a single stylesheet rule with selector and properties.
type Rule struct {
	Selector   string            // tag name or .className
	Properties map[string]string // attribute overrides
}

// Parse parses a stylesheet from JSON format.
// Format: {"selector": {"prop": "value", ...}, ...}
func Parse(jsonData string) (*Stylesheet, error) {
	jsonData = strings.TrimSpace(jsonData)
	if jsonData == "" {
		return &Stylesheet{}, nil
	}

	var raw map[string]map[string]string
	if err := json.Unmarshal([]byte(jsonData), &raw); err != nil {
		return nil, fmt.Errorf("parse stylesheet JSON: %w", err)
	}

	ss := &Stylesheet{
		Rules: make([]Rule, 0, len(raw)),
	}

	for selector, props := range raw {
		ss.Rules = append(ss.Rules, Rule{
			Selector:   selector,
			Properties: props,
		})
	}

	return ss, nil
}

// MatchSelector checks if an element matches a selector.
// Supports:
// - Tag name: "hint", "task", "role"
// - Class name: ".important", ".highlight"
func MatchSelector(selector string, tagName string, className string) bool {
	selector = strings.TrimSpace(selector)
	if selector == "" {
		return false
	}

	// Class selector
	if strings.HasPrefix(selector, ".") {
		targetClass := strings.TrimPrefix(selector, ".")
		classes := strings.Fields(className)
		for _, c := range classes {
			if c == targetClass {
				return true
			}
		}
		return false
	}

	// Tag selector
	return strings.EqualFold(selector, tagName)
}

// Apply applies stylesheet rules to get properties for an element.
// Returns merged properties from all matching rules.
// Rules are applied in specificity order (tag selectors first, then class selectors).
func (s *Stylesheet) Apply(tagName, className string) map[string]string {
	result := make(map[string]string)

	// Collect matching rules with their specificity
	type match struct {
		rule        Rule
		specificity int
	}
	var matches []match

	for _, rule := range s.Rules {
		if MatchSelector(rule.Selector, tagName, className) {
			matches = append(matches, match{rule, Specificity(rule.Selector)})
		}
	}

	// Sort by specificity (lower first, so higher specificity wins)
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].specificity < matches[j].specificity
	})

	// Apply rules in order
	for _, m := range matches {
		for k, v := range m.rule.Properties {
			result[k] = v
		}
	}

	return result
}

// Specificity returns the specificity of a selector (for ordering).
// Class selectors have higher specificity than tag selectors.
func Specificity(selector string) int {
	if strings.HasPrefix(selector, ".") {
		return 10 // class
	}
	return 1 // tag
}

// Merge combines two stylesheets, with later rules taking precedence.
func Merge(sheets ...*Stylesheet) *Stylesheet {
	result := &Stylesheet{}
	for _, ss := range sheets {
		if ss != nil {
			result.Rules = append(result.Rules, ss.Rules...)
		}
	}
	return result
}
