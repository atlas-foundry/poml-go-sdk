package template

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// ExpandOptions controls template expansion behavior.
type ExpandOptions struct {
	BaseDir       string         // Base directory for file includes
	MaxDepth      int            // Maximum include recursion depth (default 10)
	StrictMode    bool           // Error on undefined variables
	Variables     map[string]any // Initial variables
	AllowWebFetch bool           // Allow webpage component fetching
}

// forPattern matches for="var in expr" syntax.
var forPattern = regexp.MustCompile(`^(\w+)\s+in\s+(.+)$`)

// ParseForAttribute parses a for="var in expr" attribute value.
// Returns the variable name and list expression.
func ParseForAttribute(value string) (varName string, listExpr string, ok bool) {
	value = strings.TrimSpace(value)
	matches := forPattern.FindStringSubmatch(value)
	if len(matches) != 3 {
		return "", "", false
	}
	return matches[1], matches[2], true
}

// EvalCondition evaluates a condition expression and returns the boolean result.
func EvalCondition(condition string, ctx *Context) (bool, error) {
	if condition == "" {
		return true, nil
	}

	expr, err := ParseExpression(condition)
	if err != nil {
		return false, fmt.Errorf("parse condition: %w", err)
	}

	result, err := expr.Eval(ctx)
	if err != nil {
		return false, fmt.Errorf("evaluate condition: %w", err)
	}

	return toBool(result), nil
}

// EvalLoop evaluates a loop expression and returns the items to iterate over.
func EvalLoop(listExpr string, ctx *Context) ([]any, error) {
	expr, err := ParseExpression(listExpr)
	if err != nil {
		return nil, fmt.Errorf("parse loop expression: %w", err)
	}

	result, err := expr.Eval(ctx)
	if err != nil {
		return nil, fmt.Errorf("evaluate loop expression: %w", err)
	}

	switch v := result.(type) {
	case []any:
		return v, nil
	case []string:
		items := make([]any, len(v))
		for i, s := range v {
			items[i] = s
		}
		return items, nil
	case []int:
		items := make([]any, len(v))
		for i, n := range v {
			items[i] = n
		}
		return items, nil
	case nil:
		return nil, nil
	default:
		return nil, fmt.Errorf("loop expression must evaluate to array, got %T", result)
	}
}

// LetValue represents a resolved let binding value.
type LetValue struct {
	Name  string
	Value any
}

// ResolveLet resolves a let binding to its value.
// Supports: literal body, value expression, src file import.
func ResolveLet(name, valueExpr, src, body string, ctx *Context, baseDir string) (LetValue, error) {
	var value any

	if src != "" {
		// File import
		loaded, err := LoadFile(src, baseDir)
		if err != nil {
			return LetValue{}, fmt.Errorf("load let src %q: %w", src, err)
		}
		value = loaded
	} else if valueExpr != "" {
		// Expression
		expr, err := ParseExpression(valueExpr)
		if err != nil {
			return LetValue{}, fmt.Errorf("parse let value expression: %w", err)
		}
		val, err := expr.Eval(ctx)
		if err != nil {
			return LetValue{}, fmt.Errorf("evaluate let value: %w", err)
		}
		value = val
	} else if body != "" {
		// Try to parse as JSON
		body = strings.TrimSpace(body)
		if (strings.HasPrefix(body, "{") && strings.HasSuffix(body, "}")) ||
			(strings.HasPrefix(body, "[") && strings.HasSuffix(body, "]")) {
			var jsonVal any
			if err := json.Unmarshal([]byte(body), &jsonVal); err == nil {
				value = jsonVal
			} else {
				// Not valid JSON, use as string
				value = body
			}
		} else {
			value = body
		}
	}

	return LetValue{Name: name, Value: value}, nil
}

// InterpolateWithContext expands all {{ }} expressions in text.
func InterpolateWithContext(text string, ctx *Context) (string, error) {
	return Interpolate(text, ctx)
}
