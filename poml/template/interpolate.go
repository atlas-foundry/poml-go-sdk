package template

import (
	"fmt"
	"strings"
)

// ExpressionSpan identifies a template expression within text.
type ExpressionSpan struct {
	Start      int
	End        int
	Expression string
}

// FindExpressions locates all {{ expression }} spans in text.
func FindExpressions(text string) []ExpressionSpan {
	var spans []ExpressionSpan
	i := 0
	for i < len(text)-3 {
		if text[i] == '{' && text[i+1] == '{' {
			start := i
			i += 2
			// Find closing }}
			depth := 1
			exprStart := i
			for i < len(text)-1 && depth > 0 {
				if text[i] == '{' && text[i+1] == '{' {
					depth++
					i += 2
				} else if text[i] == '}' && text[i+1] == '}' {
					depth--
					if depth == 0 {
						spans = append(spans, ExpressionSpan{
							Start:      start,
							End:        i + 2,
							Expression: strings.TrimSpace(text[exprStart:i]),
						})
						i += 2
					} else {
						i += 2
					}
				} else {
					i++
				}
			}
		} else {
			i++
		}
	}
	return spans
}

// Interpolate replaces all {{ expression }} in text with evaluated values.
func Interpolate(text string, ctx *Context) (string, error) {
	spans := FindExpressions(text)
	if len(spans) == 0 {
		return text, nil
	}

	var result strings.Builder
	lastEnd := 0

	for _, span := range spans {
		// Copy text before this expression
		result.WriteString(text[lastEnd:span.Start])

		// Parse and evaluate expression
		expr, err := ParseExpression(span.Expression)
		if err != nil {
			return "", fmt.Errorf("parse expression %q: %w", span.Expression, err)
		}

		val, err := expr.Eval(ctx)
		if err != nil {
			return "", fmt.Errorf("evaluate expression %q: %w", span.Expression, err)
		}

		// Convert result to string
		result.WriteString(toString(val))

		lastEnd = span.End
	}

	// Copy remaining text
	result.WriteString(text[lastEnd:])

	return result.String(), nil
}

// toString converts any value to its string representation.
func toString(v any) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case bool:
		if val {
			return "true"
		}
		return "false"
	default:
		return fmt.Sprint(v)
	}
}

// HasExpressions returns true if text contains any {{ }} expressions.
func HasExpressions(text string) bool {
	return strings.Contains(text, "{{") && strings.Contains(text, "}}")
}
