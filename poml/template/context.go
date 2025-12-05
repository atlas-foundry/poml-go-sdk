// Package template provides template expansion for POML documents.
package template

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Context holds variables for template expansion.
type Context struct {
	vars   map[string]any
	parent *Context
}

// NewContext creates a new template context with optional initial variables.
func NewContext(vars map[string]any) *Context {
	if vars == nil {
		vars = make(map[string]any)
	}
	return &Context{vars: vars}
}

// Child creates a child context that inherits from this context.
func (c *Context) Child() *Context {
	return &Context{
		vars:   make(map[string]any),
		parent: c,
	}
}

// Get retrieves a variable value, searching parent contexts if not found.
func (c *Context) Get(name string) (any, bool) {
	// Handle dot notation for nested access
	parts := strings.Split(name, ".")
	val, ok := c.getSimple(parts[0])
	if !ok {
		return nil, false
	}

	// Navigate nested structure
	for _, part := range parts[1:] {
		switch v := val.(type) {
		case map[string]any:
			val, ok = v[part]
			if !ok {
				return nil, false
			}
		case map[string]string:
			val, ok = v[part]
			if !ok {
				return nil, false
			}
		default:
			return nil, false
		}
	}

	return val, true
}

func (c *Context) getSimple(name string) (any, bool) {
	if val, ok := c.vars[name]; ok {
		return val, true
	}
	if c.parent != nil {
		return c.parent.getSimple(name)
	}
	return nil, false
}

// Set stores a variable in the current context.
func (c *Context) Set(name string, value any) {
	c.vars[name] = value
}

// SetAll merges multiple variables into the current context.
func (c *Context) SetAll(vars map[string]any) {
	for k, v := range vars {
		c.vars[k] = v
	}
}

// All returns a flattened map of all variables (including parent contexts).
func (c *Context) All() map[string]any {
	result := make(map[string]any)

	// Start with parent variables
	if c.parent != nil {
		for k, v := range c.parent.All() {
			result[k] = v
		}
	}

	// Override with current variables
	for k, v := range c.vars {
		result[k] = v
	}

	return result
}

// LoadFile loads a JSON or text file and returns its contents.
// JSON files are parsed into map[string]any or []any.
// Other files are returned as strings.
func LoadFile(path string, baseDir string) (any, error) {
	if !filepath.IsAbs(path) && baseDir != "" {
		path = filepath.Join(baseDir, path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("load file %s: %w", path, err)
	}

	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".json" {
		var result any
		if err := json.Unmarshal(data, &result); err != nil {
			return nil, fmt.Errorf("parse JSON %s: %w", path, err)
		}
		return result, nil
	}

	return string(data), nil
}

// LoopContext provides loop iteration metadata.
type LoopContext struct {
	Index  int
	Length int
	First  bool
	Last   bool
	Value  any
}

// NewLoopContext creates a loop context for the given iteration.
func NewLoopContext(index, length int, value any) LoopContext {
	return LoopContext{
		Index:  index,
		Length: length,
		First:  index == 0,
		Last:   index == length-1,
		Value:  value,
	}
}
