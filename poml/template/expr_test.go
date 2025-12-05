package template

import "testing"

func TestParseExpression(t *testing.T) {
	tests := []struct {
		input   string
		wantErr bool
	}{
		{"x", false},
		{"42", false},
		{`"hello"`, false},
		{"true", false},
		{"false", false},
		{"null", false},
		{"x + y", false},
		{"a == b", false},
		{"a && b", false},
		{"a || b", false},
		{"!x", false},
		{"a ? b : c", false},
		{"arr[0]", false},
		{"obj.field", false},
		{"len(arr)", false},
		{"[1, 2, 3]", false},
		{`{"a": 1}`, false},
		{"", true},
		{"(", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			_, err := ParseExpression(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseExpression(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestExpressionEval(t *testing.T) {
	ctx := NewContext(map[string]any{
		"x":   int64(10),
		"y":   int64(5),
		"s":   "hello",
		"arr": []any{int64(1), int64(2), int64(3)},
		"obj": map[string]any{"a": int64(1), "b": "test"},
	})

	tests := []struct {
		input string
		want  any
	}{
		{"42", int64(42)},
		{"3.14", 3.14},
		{`"hello"`, "hello"},
		{"true", true},
		{"false", false},
		{"x", int64(10)},
		{"x + y", int64(15)},
		{"x - y", int64(5)},
		{"x * y", int64(50)},
		{"x / y", 2.0},
		{"x % 3", int64(1)},
		{"x == 10", true},
		{"x != 10", false},
		{"x > y", true},
		{"x < y", false},
		{"x >= 10", true},
		{"x <= 10", true},
		{"true && true", true},
		{"true && false", false},
		{"true || false", true},
		{"false || false", false},
		{"!true", false},
		{"!false", true},
		{"-x", int64(-10)},
		{"x > 5 ? 1 : 0", int64(1)},
		{"x < 5 ? 1 : 0", int64(0)},
		{"arr[0]", int64(1)},
		{"arr[2]", int64(3)},
		{"obj.a", int64(1)},
		{"obj.b", "test"},
		{`obj["a"]`, int64(1)},
		{"len(arr)", int64(3)},
		{"len(s)", int64(5)},
		{"upper(s)", "HELLO"},
		{"lower(s)", "hello"},
		{"[1, 2][0]", int64(1)},
		{`{"x": 1}.x`, int64(1)},
		{"s + \" world\"", "hello world"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			expr, err := ParseExpression(tt.input)
			if err != nil {
				t.Fatalf("ParseExpression() error = %v", err)
			}
			got, err := expr.Eval(ctx)
			if err != nil {
				t.Fatalf("Eval() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("Eval(%q) = %v (%T), want %v (%T)", tt.input, got, got, tt.want, tt.want)
			}
		})
	}
}

func TestExpressionErrors(t *testing.T) {
	ctx := NewContext(nil)

	tests := []struct {
		input string
	}{
		{"undefined"},
		{"1 / 0"},
		{"arr[0]"}, // arr not defined
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			expr, err := ParseExpression(tt.input)
			if err != nil {
				return // parse error is acceptable
			}
			_, err = expr.Eval(ctx)
			if err == nil {
				t.Errorf("Eval(%q) should error", tt.input)
			}
		})
	}
}

func TestBuiltinFunctions(t *testing.T) {
	ctx := NewContext(nil)

	tests := []struct {
		input string
		want  any
	}{
		{`len("abc")`, int64(3)},
		{`upper("hello")`, "HELLO"},
		{`lower("HELLO")`, "hello"},
		{`trim("  hi  ")`, "hi"},
		{`split("a,b,c", ",")[1]`, "b"},
		{`join(["a", "b"], "-")`, "a-b"},
		{`int(3.7)`, int64(3)},
		{`float(3)`, 3.0},
		{`string(42)`, "42"},
		{`default(null, "fallback")`, "fallback"},
		{`default("value", "fallback")`, "value"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			expr, err := ParseExpression(tt.input)
			if err != nil {
				t.Fatalf("ParseExpression() error = %v", err)
			}
			got, err := expr.Eval(ctx)
			if err != nil {
				t.Fatalf("Eval() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("Eval(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestRangeFunction(t *testing.T) {
	ctx := NewContext(nil)

	expr, err := ParseExpression("range(3)")
	if err != nil {
		t.Fatalf("ParseExpression() error = %v", err)
	}
	got, err := expr.Eval(ctx)
	if err != nil {
		t.Fatalf("Eval() error = %v", err)
	}
	arr, ok := got.([]any)
	if !ok {
		t.Fatalf("range() should return array, got %T", got)
	}
	if len(arr) != 3 {
		t.Errorf("range(3) length = %d, want 3", len(arr))
	}
}
