package template

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// Expression represents a parsed template expression.
type Expression interface {
	Eval(ctx *Context) (any, error)
}

// ParseExpression parses an expression string into an evaluable Expression.
func ParseExpression(s string) (Expression, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, fmt.Errorf("empty expression")
	}

	p := &parser{input: s, pos: 0}
	expr, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	// Ensure we consumed all input
	p.skipWhitespace()
	if p.pos < len(p.input) {
		return nil, fmt.Errorf("unexpected input at position %d: %s", p.pos, p.input[p.pos:])
	}

	return expr, nil
}

type parser struct {
	input string
	pos   int
}

func (p *parser) parseExpression() (Expression, error) {
	return p.parseTernary()
}

func (p *parser) parseTernary() (Expression, error) {
	cond, err := p.parseOr()
	if err != nil {
		return nil, err
	}

	p.skipWhitespace()
	if p.pos < len(p.input) && p.input[p.pos] == '?' {
		p.pos++
		thenExpr, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		p.skipWhitespace()
		if p.pos >= len(p.input) || p.input[p.pos] != ':' {
			return nil, fmt.Errorf("expected ':' in ternary expression")
		}
		p.pos++
		elseExpr, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		return &ternaryExpr{cond: cond, thenExpr: thenExpr, elseExpr: elseExpr}, nil
	}

	return cond, nil
}

func (p *parser) parseOr() (Expression, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}

	for {
		p.skipWhitespace()
		if p.pos+1 < len(p.input) && p.input[p.pos:p.pos+2] == "||" {
			p.pos += 2
			right, err := p.parseAnd()
			if err != nil {
				return nil, err
			}
			left = &binaryExpr{op: "||", left: left, right: right}
		} else {
			break
		}
	}

	return left, nil
}

func (p *parser) parseAnd() (Expression, error) {
	left, err := p.parseEquality()
	if err != nil {
		return nil, err
	}

	for {
		p.skipWhitespace()
		if p.pos+1 < len(p.input) && p.input[p.pos:p.pos+2] == "&&" {
			p.pos += 2
			right, err := p.parseEquality()
			if err != nil {
				return nil, err
			}
			left = &binaryExpr{op: "&&", left: left, right: right}
		} else {
			break
		}
	}

	return left, nil
}

func (p *parser) parseEquality() (Expression, error) {
	left, err := p.parseComparison()
	if err != nil {
		return nil, err
	}

	for {
		p.skipWhitespace()
		if p.pos+1 < len(p.input) {
			op := p.input[p.pos : p.pos+2]
			if op == "==" || op == "!=" {
				p.pos += 2
				right, err := p.parseComparison()
				if err != nil {
					return nil, err
				}
				left = &binaryExpr{op: op, left: left, right: right}
				continue
			}
		}
		break
	}

	return left, nil
}

func (p *parser) parseComparison() (Expression, error) {
	left, err := p.parseAdditive()
	if err != nil {
		return nil, err
	}

	for {
		p.skipWhitespace()
		if p.pos+1 < len(p.input) {
			op := p.input[p.pos : p.pos+2]
			if op == "<=" || op == ">=" {
				p.pos += 2
				right, err := p.parseAdditive()
				if err != nil {
					return nil, err
				}
				left = &binaryExpr{op: op, left: left, right: right}
				continue
			}
		}
		if p.pos < len(p.input) {
			ch := p.input[p.pos]
			if ch == '<' || ch == '>' {
				p.pos++
				right, err := p.parseAdditive()
				if err != nil {
					return nil, err
				}
				left = &binaryExpr{op: string(ch), left: left, right: right}
				continue
			}
		}
		break
	}

	return left, nil
}

func (p *parser) parseAdditive() (Expression, error) {
	left, err := p.parseMultiplicative()
	if err != nil {
		return nil, err
	}

	for {
		p.skipWhitespace()
		if p.pos < len(p.input) {
			ch := p.input[p.pos]
			if ch == '+' || ch == '-' {
				p.pos++
				right, err := p.parseMultiplicative()
				if err != nil {
					return nil, err
				}
				left = &binaryExpr{op: string(ch), left: left, right: right}
				continue
			}
		}
		break
	}

	return left, nil
}

func (p *parser) parseMultiplicative() (Expression, error) {
	left, err := p.parseUnary()
	if err != nil {
		return nil, err
	}

	for {
		p.skipWhitespace()
		if p.pos < len(p.input) {
			ch := p.input[p.pos]
			if ch == '*' || ch == '/' || ch == '%' {
				p.pos++
				right, err := p.parseUnary()
				if err != nil {
					return nil, err
				}
				left = &binaryExpr{op: string(ch), left: left, right: right}
				continue
			}
		}
		break
	}

	return left, nil
}

func (p *parser) parseUnary() (Expression, error) {
	p.skipWhitespace()
	if p.pos < len(p.input) && p.input[p.pos] == '!' {
		p.pos++
		expr, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &unaryExpr{op: "!", expr: expr}, nil
	}
	if p.pos < len(p.input) && p.input[p.pos] == '-' {
		// Check if this is a negative number or subtraction
		if p.pos+1 < len(p.input) && (unicode.IsDigit(rune(p.input[p.pos+1])) || p.input[p.pos+1] == '.') {
			return p.parsePrimary()
		}
		p.pos++
		expr, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &unaryExpr{op: "-", expr: expr}, nil
	}
	return p.parsePostfix()
}

func (p *parser) parsePostfix() (Expression, error) {
	expr, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}

	for {
		p.skipWhitespace()
		if p.pos >= len(p.input) {
			break
		}

		if p.input[p.pos] == '[' {
			p.pos++
			index, err := p.parseExpression()
			if err != nil {
				return nil, err
			}
			p.skipWhitespace()
			if p.pos >= len(p.input) || p.input[p.pos] != ']' {
				return nil, fmt.Errorf("expected ']'")
			}
			p.pos++
			expr = &indexExpr{expr: expr, index: index}
		} else if p.input[p.pos] == '.' {
			p.pos++
			name, err := p.parseIdentifier()
			if err != nil {
				return nil, err
			}
			expr = &memberExpr{expr: expr, member: name}
		} else if p.input[p.pos] == '(' {
			// Function call
			if varExpr, ok := expr.(*variableExpr); ok {
				p.pos++
				args, err := p.parseArguments()
				if err != nil {
					return nil, err
				}
				expr = &callExpr{name: varExpr.name, args: args}
			} else {
				break
			}
		} else {
			break
		}
	}

	return expr, nil
}

func (p *parser) parsePrimary() (Expression, error) {
	p.skipWhitespace()
	if p.pos >= len(p.input) {
		return nil, fmt.Errorf("unexpected end of expression")
	}

	ch := p.input[p.pos]

	// Parenthesized expression
	if ch == '(' {
		p.pos++
		expr, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		p.skipWhitespace()
		if p.pos >= len(p.input) || p.input[p.pos] != ')' {
			return nil, fmt.Errorf("expected ')'")
		}
		p.pos++
		return expr, nil
	}

	// String literal
	if ch == '"' || ch == '\'' {
		return p.parseString()
	}

	// Number literal
	if unicode.IsDigit(rune(ch)) || ch == '-' || ch == '.' {
		return p.parseNumber()
	}

	// Array literal
	if ch == '[' {
		return p.parseArray()
	}

	// Object literal
	if ch == '{' {
		return p.parseObject()
	}

	// Keywords: true, false, null
	if p.matchKeyword("true") {
		return &literalExpr{value: true}, nil
	}
	if p.matchKeyword("false") {
		return &literalExpr{value: false}, nil
	}
	if p.matchKeyword("null") || p.matchKeyword("nil") {
		return &literalExpr{value: nil}, nil
	}

	// Variable reference
	name, err := p.parseIdentifier()
	if err != nil {
		return nil, err
	}
	return &variableExpr{name: name}, nil
}

func (p *parser) parseString() (Expression, error) {
	quote := p.input[p.pos]
	p.pos++
	var sb strings.Builder
	for p.pos < len(p.input) {
		ch := p.input[p.pos]
		if ch == quote {
			p.pos++
			return &literalExpr{value: sb.String()}, nil
		}
		if ch == '\\' && p.pos+1 < len(p.input) {
			p.pos++
			switch p.input[p.pos] {
			case 'n':
				sb.WriteByte('\n')
			case 't':
				sb.WriteByte('\t')
			case 'r':
				sb.WriteByte('\r')
			case '\\':
				sb.WriteByte('\\')
			case '"':
				sb.WriteByte('"')
			case '\'':
				sb.WriteByte('\'')
			default:
				sb.WriteByte(p.input[p.pos])
			}
			p.pos++
			continue
		}
		sb.WriteByte(ch)
		p.pos++
	}
	return nil, fmt.Errorf("unterminated string")
}

func (p *parser) parseNumber() (Expression, error) {
	start := p.pos
	if p.pos < len(p.input) && p.input[p.pos] == '-' {
		p.pos++
	}
	hasDigit := false
	for p.pos < len(p.input) && unicode.IsDigit(rune(p.input[p.pos])) {
		hasDigit = true
		p.pos++
	}
	if p.pos < len(p.input) && p.input[p.pos] == '.' {
		p.pos++
		for p.pos < len(p.input) && unicode.IsDigit(rune(p.input[p.pos])) {
			hasDigit = true
			p.pos++
		}
	}
	if !hasDigit {
		return nil, fmt.Errorf("invalid number")
	}
	numStr := p.input[start:p.pos]
	if strings.Contains(numStr, ".") {
		val, err := strconv.ParseFloat(numStr, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid float: %s", numStr)
		}
		return &literalExpr{value: val}, nil
	}
	val, err := strconv.ParseInt(numStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid integer: %s", numStr)
	}
	return &literalExpr{value: val}, nil
}

func (p *parser) parseArray() (Expression, error) {
	p.pos++ // consume '['
	var elements []Expression
	for {
		p.skipWhitespace()
		if p.pos >= len(p.input) {
			return nil, fmt.Errorf("unterminated array")
		}
		if p.input[p.pos] == ']' {
			p.pos++
			return &arrayExpr{elements: elements}, nil
		}
		elem, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		elements = append(elements, elem)
		p.skipWhitespace()
		if p.pos < len(p.input) && p.input[p.pos] == ',' {
			p.pos++
		}
	}
}

func (p *parser) parseObject() (Expression, error) {
	p.pos++ // consume '{'
	pairs := make(map[string]Expression)
	for {
		p.skipWhitespace()
		if p.pos >= len(p.input) {
			return nil, fmt.Errorf("unterminated object")
		}
		if p.input[p.pos] == '}' {
			p.pos++
			return &objectExpr{pairs: pairs}, nil
		}
		// Parse key
		var key string
		if p.input[p.pos] == '"' || p.input[p.pos] == '\'' {
			keyExpr, err := p.parseString()
			if err != nil {
				return nil, err
			}
			key = keyExpr.(*literalExpr).value.(string)
		} else {
			var err error
			key, err = p.parseIdentifier()
			if err != nil {
				return nil, err
			}
		}
		p.skipWhitespace()
		if p.pos >= len(p.input) || p.input[p.pos] != ':' {
			return nil, fmt.Errorf("expected ':' in object")
		}
		p.pos++
		value, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		pairs[key] = value
		p.skipWhitespace()
		if p.pos < len(p.input) && p.input[p.pos] == ',' {
			p.pos++
		}
	}
}

func (p *parser) parseArguments() ([]Expression, error) {
	var args []Expression
	for {
		p.skipWhitespace()
		if p.pos >= len(p.input) {
			return nil, fmt.Errorf("unterminated function call")
		}
		if p.input[p.pos] == ')' {
			p.pos++
			return args, nil
		}
		arg, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		args = append(args, arg)
		p.skipWhitespace()
		if p.pos < len(p.input) && p.input[p.pos] == ',' {
			p.pos++
		}
	}
}

func (p *parser) parseIdentifier() (string, error) {
	p.skipWhitespace()
	start := p.pos
	if p.pos >= len(p.input) {
		return "", fmt.Errorf("expected identifier")
	}
	ch := rune(p.input[p.pos])
	if !unicode.IsLetter(ch) && ch != '_' {
		return "", fmt.Errorf("expected identifier, got %c", ch)
	}
	for p.pos < len(p.input) {
		ch := rune(p.input[p.pos])
		if !unicode.IsLetter(ch) && !unicode.IsDigit(ch) && ch != '_' {
			break
		}
		p.pos++
	}
	return p.input[start:p.pos], nil
}

func (p *parser) matchKeyword(keyword string) bool {
	if p.pos+len(keyword) > len(p.input) {
		return false
	}
	if p.input[p.pos:p.pos+len(keyword)] != keyword {
		return false
	}
	// Ensure it's not a prefix of a longer identifier
	if p.pos+len(keyword) < len(p.input) {
		ch := rune(p.input[p.pos+len(keyword)])
		if unicode.IsLetter(ch) || unicode.IsDigit(ch) || ch == '_' {
			return false
		}
	}
	p.pos += len(keyword)
	return true
}

func (p *parser) skipWhitespace() {
	for p.pos < len(p.input) && unicode.IsSpace(rune(p.input[p.pos])) {
		p.pos++
	}
}

// Expression implementations

type literalExpr struct {
	value any
}

func (e *literalExpr) Eval(_ *Context) (any, error) {
	return e.value, nil
}

type variableExpr struct {
	name string
}

func (e *variableExpr) Eval(ctx *Context) (any, error) {
	val, ok := ctx.Get(e.name)
	if !ok {
		return nil, fmt.Errorf("undefined variable: %s", e.name)
	}
	return val, nil
}

type binaryExpr struct {
	op          string
	left, right Expression
}

func (e *binaryExpr) Eval(ctx *Context) (any, error) {
	left, err := e.left.Eval(ctx)
	if err != nil {
		return nil, err
	}
	right, err := e.right.Eval(ctx)
	if err != nil {
		return nil, err
	}
	return evalBinary(e.op, left, right)
}

type unaryExpr struct {
	op   string
	expr Expression
}

func (e *unaryExpr) Eval(ctx *Context) (any, error) {
	val, err := e.expr.Eval(ctx)
	if err != nil {
		return nil, err
	}
	switch e.op {
	case "!":
		return !toBool(val), nil
	case "-":
		return negate(val)
	}
	return nil, fmt.Errorf("unknown unary operator: %s", e.op)
}

type ternaryExpr struct {
	cond, thenExpr, elseExpr Expression
}

func (e *ternaryExpr) Eval(ctx *Context) (any, error) {
	cond, err := e.cond.Eval(ctx)
	if err != nil {
		return nil, err
	}
	if toBool(cond) {
		return e.thenExpr.Eval(ctx)
	}
	return e.elseExpr.Eval(ctx)
}

type indexExpr struct {
	expr  Expression
	index Expression
}

func (e *indexExpr) Eval(ctx *Context) (any, error) {
	val, err := e.expr.Eval(ctx)
	if err != nil {
		return nil, err
	}
	idx, err := e.index.Eval(ctx)
	if err != nil {
		return nil, err
	}
	return indexAccess(val, idx)
}

type memberExpr struct {
	expr   Expression
	member string
}

func (e *memberExpr) Eval(ctx *Context) (any, error) {
	val, err := e.expr.Eval(ctx)
	if err != nil {
		return nil, err
	}
	return memberAccess(val, e.member)
}

type callExpr struct {
	name string
	args []Expression
}

func (e *callExpr) Eval(ctx *Context) (any, error) {
	args := make([]any, len(e.args))
	for i, arg := range e.args {
		val, err := arg.Eval(ctx)
		if err != nil {
			return nil, err
		}
		args[i] = val
	}
	return callFunction(e.name, args)
}

type arrayExpr struct {
	elements []Expression
}

func (e *arrayExpr) Eval(ctx *Context) (any, error) {
	result := make([]any, len(e.elements))
	for i, elem := range e.elements {
		val, err := elem.Eval(ctx)
		if err != nil {
			return nil, err
		}
		result[i] = val
	}
	return result, nil
}

type objectExpr struct {
	pairs map[string]Expression
}

func (e *objectExpr) Eval(ctx *Context) (any, error) {
	result := make(map[string]any)
	for k, v := range e.pairs {
		val, err := v.Eval(ctx)
		if err != nil {
			return nil, err
		}
		result[k] = val
	}
	return result, nil
}
