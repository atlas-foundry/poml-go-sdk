package template

import (
	"fmt"
	"reflect"
	"strings"
)

// toBool converts any value to a boolean for conditional evaluation.
func toBool(v any) bool {
	if v == nil {
		return false
	}
	switch val := v.(type) {
	case bool:
		return val
	case int64:
		return val != 0
	case float64:
		return val != 0
	case string:
		return val != ""
	case []any:
		return len(val) > 0
	case map[string]any:
		return len(val) > 0
	default:
		return true
	}
}

// toFloat64 converts numeric values to float64 for arithmetic.
func toFloat64(v any) (float64, bool) {
	switch val := v.(type) {
	case int64:
		return float64(val), true
	case int:
		return float64(val), true
	case float64:
		return val, true
	case float32:
		return float64(val), true
	default:
		return 0, false
	}
}

// negate performs unary negation.
func negate(v any) (any, error) {
	f, ok := toFloat64(v)
	if !ok {
		return nil, fmt.Errorf("cannot negate non-numeric value: %v", v)
	}
	if _, isInt := v.(int64); isInt {
		return -int64(f), nil
	}
	return -f, nil
}

// evalBinary evaluates binary operations.
func evalBinary(op string, left, right any) (any, error) {
	switch op {
	case "+":
		// String concatenation
		if ls, ok := left.(string); ok {
			return ls + fmt.Sprint(right), nil
		}
		if rs, ok := right.(string); ok {
			return fmt.Sprint(left) + rs, nil
		}
		// Numeric addition
		lf, lok := toFloat64(left)
		rf, rok := toFloat64(right)
		if lok && rok {
			if isInt(left) && isInt(right) {
				return int64(lf) + int64(rf), nil
			}
			return lf + rf, nil
		}
		return nil, fmt.Errorf("cannot add %T and %T", left, right)

	case "-":
		lf, lok := toFloat64(left)
		rf, rok := toFloat64(right)
		if lok && rok {
			if isInt(left) && isInt(right) {
				return int64(lf) - int64(rf), nil
			}
			return lf - rf, nil
		}
		return nil, fmt.Errorf("cannot subtract %T from %T", right, left)

	case "*":
		lf, lok := toFloat64(left)
		rf, rok := toFloat64(right)
		if lok && rok {
			if isInt(left) && isInt(right) {
				return int64(lf) * int64(rf), nil
			}
			return lf * rf, nil
		}
		return nil, fmt.Errorf("cannot multiply %T and %T", left, right)

	case "/":
		lf, lok := toFloat64(left)
		rf, rok := toFloat64(right)
		if lok && rok {
			if rf == 0 {
				return nil, fmt.Errorf("division by zero")
			}
			return lf / rf, nil
		}
		return nil, fmt.Errorf("cannot divide %T by %T", left, right)

	case "%":
		lf, lok := toFloat64(left)
		rf, rok := toFloat64(right)
		if lok && rok {
			if rf == 0 {
				return nil, fmt.Errorf("modulo by zero")
			}
			return int64(lf) % int64(rf), nil
		}
		return nil, fmt.Errorf("cannot modulo %T by %T", left, right)

	case "==":
		return equals(left, right), nil

	case "!=":
		return !equals(left, right), nil

	case "<":
		return compare(left, right) < 0, nil

	case "<=":
		return compare(left, right) <= 0, nil

	case ">":
		return compare(left, right) > 0, nil

	case ">=":
		return compare(left, right) >= 0, nil

	case "&&":
		return toBool(left) && toBool(right), nil

	case "||":
		return toBool(left) || toBool(right), nil

	default:
		return nil, fmt.Errorf("unknown operator: %s", op)
	}
}

func isInt(v any) bool {
	switch v.(type) {
	case int, int64, int32:
		return true
	default:
		return false
	}
}

func equals(a, b any) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return reflect.DeepEqual(a, b)
}

func compare(a, b any) int {
	af, aok := toFloat64(a)
	bf, bok := toFloat64(b)
	if aok && bok {
		if af < bf {
			return -1
		}
		if af > bf {
			return 1
		}
		return 0
	}
	// String comparison
	as := fmt.Sprint(a)
	bs := fmt.Sprint(b)
	return strings.Compare(as, bs)
}

// indexAccess handles array/map indexing: arr[0], obj["key"]
func indexAccess(val, idx any) (any, error) {
	switch v := val.(type) {
	case []any:
		i, ok := idx.(int64)
		if !ok {
			if f, ok := idx.(float64); ok {
				i = int64(f)
			} else {
				return nil, fmt.Errorf("array index must be integer, got %T", idx)
			}
		}
		if i < 0 || int(i) >= len(v) {
			return nil, fmt.Errorf("array index out of bounds: %d", i)
		}
		return v[i], nil

	case map[string]any:
		key := fmt.Sprint(idx)
		result, ok := v[key]
		if !ok {
			return nil, nil // undefined key returns nil
		}
		return result, nil

	case string:
		i, ok := idx.(int64)
		if !ok {
			if f, ok := idx.(float64); ok {
				i = int64(f)
			} else {
				return nil, fmt.Errorf("string index must be integer, got %T", idx)
			}
		}
		if i < 0 || int(i) >= len(v) {
			return nil, fmt.Errorf("string index out of bounds: %d", i)
		}
		return string(v[i]), nil

	default:
		return nil, fmt.Errorf("cannot index %T", val)
	}
}

// memberAccess handles dot notation: obj.field
func memberAccess(val any, member string) (any, error) {
	switch v := val.(type) {
	case map[string]any:
		result, ok := v[member]
		if !ok {
			return nil, nil
		}
		return result, nil

	case map[string]string:
		result, ok := v[member]
		if !ok {
			return nil, nil
		}
		return result, nil

	case LoopContext:
		switch member {
		case "index":
			return int64(v.Index), nil
		case "length":
			return int64(v.Length), nil
		case "first":
			return v.First, nil
		case "last":
			return v.Last, nil
		case "value":
			return v.Value, nil
		}
		return nil, fmt.Errorf("unknown loop property: %s", member)

	default:
		return nil, fmt.Errorf("cannot access member %q on %T", member, val)
	}
}

// Built-in functions

var builtinFuncs = map[string]func([]any) (any, error){
	"len":     fnLen,
	"upper":   fnUpper,
	"lower":   fnLower,
	"trim":    fnTrim,
	"split":   fnSplit,
	"join":    fnJoin,
	"int":     fnInt,
	"float":   fnFloat,
	"string":  fnString,
	"range":   fnRange,
	"default": fnDefault,
}

func callFunction(name string, args []any) (any, error) {
	fn, ok := builtinFuncs[name]
	if !ok {
		return nil, fmt.Errorf("unknown function: %s", name)
	}
	return fn(args)
}

func fnLen(args []any) (any, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("len() takes 1 argument, got %d", len(args))
	}
	switch v := args[0].(type) {
	case string:
		return int64(len(v)), nil
	case []any:
		return int64(len(v)), nil
	case map[string]any:
		return int64(len(v)), nil
	default:
		return nil, fmt.Errorf("len() requires string, array, or object")
	}
}

func fnUpper(args []any) (any, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("upper() takes 1 argument")
	}
	return strings.ToUpper(fmt.Sprint(args[0])), nil
}

func fnLower(args []any) (any, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("lower() takes 1 argument")
	}
	return strings.ToLower(fmt.Sprint(args[0])), nil
}

func fnTrim(args []any) (any, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("trim() takes 1 argument")
	}
	return strings.TrimSpace(fmt.Sprint(args[0])), nil
}

func fnSplit(args []any) (any, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("split() takes 2 arguments")
	}
	s := fmt.Sprint(args[0])
	sep := fmt.Sprint(args[1])
	parts := strings.Split(s, sep)
	result := make([]any, len(parts))
	for i, p := range parts {
		result[i] = p
	}
	return result, nil
}

func fnJoin(args []any) (any, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("join() takes 2 arguments")
	}
	arr, ok := args[0].([]any)
	if !ok {
		return nil, fmt.Errorf("join() first argument must be array")
	}
	sep := fmt.Sprint(args[1])
	strs := make([]string, len(arr))
	for i, v := range arr {
		strs[i] = fmt.Sprint(v)
	}
	return strings.Join(strs, sep), nil
}

func fnInt(args []any) (any, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("int() takes 1 argument")
	}
	f, ok := toFloat64(args[0])
	if ok {
		return int64(f), nil
	}
	return nil, fmt.Errorf("cannot convert to int: %v", args[0])
}

func fnFloat(args []any) (any, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("float() takes 1 argument")
	}
	f, ok := toFloat64(args[0])
	if ok {
		return f, nil
	}
	return nil, fmt.Errorf("cannot convert to float: %v", args[0])
}

func fnString(args []any) (any, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("string() takes 1 argument")
	}
	return fmt.Sprint(args[0]), nil
}

func fnRange(args []any) (any, error) {
	if len(args) < 1 || len(args) > 2 {
		return nil, fmt.Errorf("range() takes 1-2 arguments")
	}
	var start, end int64
	if len(args) == 1 {
		f, ok := toFloat64(args[0])
		if !ok {
			return nil, fmt.Errorf("range() argument must be numeric")
		}
		start = 0
		end = int64(f)
	} else {
		sf, sok := toFloat64(args[0])
		ef, eok := toFloat64(args[1])
		if !sok || !eok {
			return nil, fmt.Errorf("range() arguments must be numeric")
		}
		start = int64(sf)
		end = int64(ef)
	}
	result := make([]any, 0, end-start)
	for i := start; i < end; i++ {
		result = append(result, i)
	}
	return result, nil
}

func fnDefault(args []any) (any, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("default() takes 2 arguments")
	}
	if args[0] == nil || args[0] == "" {
		return args[1], nil
	}
	return args[0], nil
}
