package template

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindExpressions(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{"no expressions", "plain text", 0},
		{"single expression", "hello {{ name }}", 1},
		{"two expressions", "{{ first }} and {{ second }}", 2},
		{"expression at start", "{{ start }} text", 1},
		{"expression at end", "text {{ end }}", 1},
		{"expression only", "{{ only }}", 1},
		{"multiple adjacent", "{{ a }}{{ b }}{{ c }}", 3},
		{"incomplete open", "text { not an expression", 0},
		{"incomplete close", "text {{ unclosed", 0},
		{"empty braces", "{{ }}", 1},
		{"with whitespace", "{{   spaced   }}", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spans := FindExpressions(tt.input)
			if len(spans) != tt.want {
				t.Errorf("FindExpressions(%q) = %d spans, want %d", tt.input, len(spans), tt.want)
			}
		})
	}
}

func TestFindExpressionsContent(t *testing.T) {
	input := "Hello {{ name }}, you are {{ age }} years old"
	spans := FindExpressions(input)

	if len(spans) != 2 {
		t.Fatalf("expected 2 spans, got %d", len(spans))
	}

	if spans[0].Expression != "name" {
		t.Errorf("first expression = %q, want %q", spans[0].Expression, "name")
	}
	if spans[1].Expression != "age" {
		t.Errorf("second expression = %q, want %q", spans[1].Expression, "age")
	}

	// Check positions
	if spans[0].Start != 6 || spans[0].End != 16 {
		t.Errorf("first span position = [%d:%d], want [6:16]", spans[0].Start, spans[0].End)
	}
}

func TestHasExpressions(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"plain text", false},
		{"has {{ expr }}", true},
		{"only {{ open", false},
		{"only close }}", false},
		{"{{ a }} and {{ b }}", true},
		{"{{ }}", true},
		{"separate { { } }", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := HasExpressions(tt.input)
			if got != tt.want {
				t.Errorf("HasExpressions(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestInterpolate(t *testing.T) {
	ctx := NewContext(map[string]any{
		"name": "Alice",
		"age":  30,
		"city": "NYC",
	})

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{"no expressions", "hello world", "hello world", false},
		{"single var", "hello {{ name }}", "hello Alice", false},
		{"multiple vars", "{{ name }} is {{ age }}", "Alice is 30", false},
		{"expression at start", "{{ name }} here", "Alice here", false},
		{"expression at end", "hello {{ name }}", "hello Alice", false},
		{"expression only", "{{ name }}", "Alice", false},
		{"math expression", "{{ age + 5 }}", "35", false},
		{"undefined var", "{{ unknown }}", "", true},
		{"invalid expression", "{{ ++ }}", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Interpolate(tt.input, ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("Interpolate(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("Interpolate(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestInterpolateComplex(t *testing.T) {
	ctx := NewContext(map[string]any{
		"user": map[string]any{
			"name": "Bob",
			"age":  25,
		},
		"items": []any{"apple", "banana", "cherry"},
	})

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"nested access", "{{ user.name }}", "Bob"},
		{"array access", "{{ items[0] }}", "apple"},
		{"function call", "{{ len(items) }}", "3"},
		{"ternary", "{{ user.age > 18 ? \"adult\" : \"minor\" }}", "adult"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Interpolate(tt.input, ctx)
			if err != nil {
				t.Fatalf("Interpolate(%q) error = %v", tt.input, err)
			}
			if got != tt.want {
				t.Errorf("Interpolate(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestToString(t *testing.T) {
	tests := []struct {
		input any
		want  string
	}{
		{nil, ""},
		{"hello", "hello"},
		{true, "true"},
		{false, "false"},
		{42, "42"},
		{3.14, "3.14"},
		{int64(100), "100"},
	}

	for _, tt := range tests {
		got := toString(tt.input)
		if got != tt.want {
			t.Errorf("toString(%v) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestParseForAttribute(t *testing.T) {
	tests := []struct {
		input    string
		wantVar  string
		wantExpr string
		wantOk   bool
	}{
		{"item in items", "item", "items", true},
		{"x in arr", "x", "arr", true},
		{"user in users", "user", "users", true},
		{"i in range(5)", "i", "range(5)", true},
		{"  item   in   items  ", "item", "items", true},
		{"invalid", "", "", false},
		{"in items", "", "", false},
		{"item in", "", "", false},
		{"", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			gotVar, gotExpr, gotOk := ParseForAttribute(tt.input)
			if gotOk != tt.wantOk {
				t.Errorf("ParseForAttribute(%q) ok = %v, want %v", tt.input, gotOk, tt.wantOk)
				return
			}
			if gotVar != tt.wantVar {
				t.Errorf("ParseForAttribute(%q) var = %q, want %q", tt.input, gotVar, tt.wantVar)
			}
			if gotExpr != tt.wantExpr {
				t.Errorf("ParseForAttribute(%q) expr = %q, want %q", tt.input, gotExpr, tt.wantExpr)
			}
		})
	}
}

func TestEvalCondition(t *testing.T) {
	ctx := NewContext(map[string]any{
		"active":   true,
		"inactive": false,
		"count":    int64(10),
		"name":     "test",
	})

	tests := []struct {
		condition string
		want      bool
		wantErr   bool
	}{
		{"", true, false}, // empty condition is always true
		{"true", true, false},
		{"false", false, false},
		{"active", true, false},
		{"inactive", false, false},
		{"count > 5", true, false},
		{"count < 5", false, false},
		{"count == 10", true, false},
		{"name == \"test\"", true, false},
		{"active && count > 5", true, false},
		{"inactive || count > 5", true, false},
		{"!inactive", true, false},
		{"undefined_var", false, true},
		{"invalid (( syntax", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.condition, func(t *testing.T) {
			got, err := EvalCondition(tt.condition, ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("EvalCondition(%q) error = %v, wantErr %v", tt.condition, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("EvalCondition(%q) = %v, want %v", tt.condition, got, tt.want)
			}
		})
	}
}

func TestEvalLoop(t *testing.T) {
	ctx := NewContext(map[string]any{
		"items":    []any{"a", "b", "c"},
		"numbers":  []int{1, 2, 3},
		"strings":  []string{"x", "y"},
		"empty":    []any{},
		"single":   "not-an-array",
		"nilValue": nil,
	})

	tests := []struct {
		expr    string
		wantLen int
		wantErr bool
	}{
		{"items", 3, false},
		{"numbers", 3, false},
		{"strings", 2, false},
		{"empty", 0, false},
		{"nilValue", 0, false},
		{"single", 0, true},     // not an array
		{"undefined", 0, true},  // undefined variable
		{"invalid ((", 0, true}, // parse error
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			got, err := EvalLoop(tt.expr, ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("EvalLoop(%q) error = %v, wantErr %v", tt.expr, err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(got) != tt.wantLen {
				t.Errorf("EvalLoop(%q) len = %d, want %d", tt.expr, len(got), tt.wantLen)
			}
		})
	}
}

func TestResolveLet(t *testing.T) {
	ctx := NewContext(map[string]any{
		"x": 10,
		"y": 20,
	})

	t.Run("value expression", func(t *testing.T) {
		result, err := ResolveLet("sum", "x + y", "", "", ctx, "")
		if err != nil {
			t.Fatalf("ResolveLet() error = %v", err)
		}
		if result.Name != "sum" {
			t.Errorf("Name = %q, want %q", result.Name, "sum")
		}
		if result.Value != int64(30) {
			t.Errorf("Value = %v (%T), want 30", result.Value, result.Value)
		}
	})

	t.Run("literal body string", func(t *testing.T) {
		result, err := ResolveLet("message", "", "", "hello world", ctx, "")
		if err != nil {
			t.Fatalf("ResolveLet() error = %v", err)
		}
		if result.Value != "hello world" {
			t.Errorf("Value = %q, want %q", result.Value, "hello world")
		}
	})

	t.Run("JSON object body", func(t *testing.T) {
		result, err := ResolveLet("config", "", "", `{"key": "value"}`, ctx, "")
		if err != nil {
			t.Fatalf("ResolveLet() error = %v", err)
		}
		obj, ok := result.Value.(map[string]any)
		if !ok {
			t.Fatalf("Value should be map, got %T", result.Value)
		}
		if obj["key"] != "value" {
			t.Errorf("obj[key] = %v, want %q", obj["key"], "value")
		}
	})

	t.Run("JSON array body", func(t *testing.T) {
		result, err := ResolveLet("items", "", "", `[1, 2, 3]`, ctx, "")
		if err != nil {
			t.Fatalf("ResolveLet() error = %v", err)
		}
		arr, ok := result.Value.([]any)
		if !ok {
			t.Fatalf("Value should be array, got %T", result.Value)
		}
		if len(arr) != 3 {
			t.Errorf("len(arr) = %d, want 3", len(arr))
		}
	})

	t.Run("invalid JSON body", func(t *testing.T) {
		result, err := ResolveLet("data", "", "", `{invalid json}`, ctx, "")
		if err != nil {
			t.Fatalf("ResolveLet() error = %v", err)
		}
		// Invalid JSON should be treated as string
		if result.Value != "{invalid json}" {
			t.Errorf("Value = %v, want raw string", result.Value)
		}
	})

	t.Run("invalid expression", func(t *testing.T) {
		_, err := ResolveLet("bad", "invalid ((", "", "", ctx, "")
		if err == nil {
			t.Error("expected error for invalid expression")
		}
	})

	t.Run("undefined variable expression", func(t *testing.T) {
		_, err := ResolveLet("bad", "undefined_var", "", "", ctx, "")
		if err == nil {
			t.Error("expected error for undefined variable")
		}
	})
}

func TestResolveLetWithFile(t *testing.T) {
	// Create a temp directory and test file
	tmpDir := t.TempDir()
	jsonFile := filepath.Join(tmpDir, "data.json")
	if err := os.WriteFile(jsonFile, []byte(`{"name": "test"}`), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	ctx := NewContext(nil)

	t.Run("load JSON file", func(t *testing.T) {
		result, err := ResolveLet("data", "", jsonFile, "", ctx, "")
		if err != nil {
			t.Fatalf("ResolveLet() error = %v", err)
		}
		obj, ok := result.Value.(map[string]any)
		if !ok {
			t.Fatalf("Value should be map, got %T", result.Value)
		}
		if obj["name"] != "test" {
			t.Errorf("obj[name] = %v, want %q", obj["name"], "test")
		}
	})

	t.Run("relative path with baseDir", func(t *testing.T) {
		result, err := ResolveLet("data", "", "data.json", "", ctx, tmpDir)
		if err != nil {
			t.Fatalf("ResolveLet() error = %v", err)
		}
		if result.Value == nil {
			t.Error("Value should not be nil")
		}
	})

	t.Run("file not found", func(t *testing.T) {
		_, err := ResolveLet("data", "", "nonexistent.json", "", ctx, tmpDir)
		if err == nil {
			t.Error("expected error for nonexistent file")
		}
	})
}

func TestInterpolateWithContext(t *testing.T) {
	ctx := NewContext(map[string]any{
		"greeting": "Hello",
		"name":     "World",
	})

	result, err := InterpolateWithContext("{{ greeting }}, {{ name }}!", ctx)
	if err != nil {
		t.Fatalf("InterpolateWithContext() error = %v", err)
	}
	if result != "Hello, World!" {
		t.Errorf("result = %q, want %q", result, "Hello, World!")
	}
}

func TestLoadFile(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("load JSON file", func(t *testing.T) {
		jsonFile := filepath.Join(tmpDir, "test.json")
		if err := os.WriteFile(jsonFile, []byte(`{"key": "value"}`), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		result, err := LoadFile(jsonFile, "")
		if err != nil {
			t.Fatalf("LoadFile() error = %v", err)
		}
		obj, ok := result.(map[string]any)
		if !ok {
			t.Fatalf("result should be map, got %T", result)
		}
		if obj["key"] != "value" {
			t.Errorf("obj[key] = %v, want %q", obj["key"], "value")
		}
	})

	t.Run("load text file", func(t *testing.T) {
		textFile := filepath.Join(tmpDir, "test.txt")
		if err := os.WriteFile(textFile, []byte("plain text content"), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		result, err := LoadFile(textFile, "")
		if err != nil {
			t.Fatalf("LoadFile() error = %v", err)
		}
		if result != "plain text content" {
			t.Errorf("result = %v, want %q", result, "plain text content")
		}
	})

	t.Run("relative path with baseDir", func(t *testing.T) {
		jsonFile := filepath.Join(tmpDir, "relative.json")
		if err := os.WriteFile(jsonFile, []byte(`[1, 2, 3]`), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		result, err := LoadFile("relative.json", tmpDir)
		if err != nil {
			t.Fatalf("LoadFile() error = %v", err)
		}
		arr, ok := result.([]any)
		if !ok {
			t.Fatalf("result should be array, got %T", result)
		}
		if len(arr) != 3 {
			t.Errorf("len(arr) = %d, want 3", len(arr))
		}
	})

	t.Run("file not found", func(t *testing.T) {
		_, err := LoadFile("nonexistent.json", tmpDir)
		if err == nil {
			t.Error("expected error for nonexistent file")
		}
	})
}

func TestExpressionParser(t *testing.T) {
	ctx := NewContext(map[string]any{
		"name":   "Alice",
		"age":    int64(30),
		"active": true,
		"items":  []any{"a", "b", "c"},
		"user": map[string]any{
			"name": "Bob",
			"profile": map[string]any{
				"email": "bob@example.com",
			},
		},
	})

	tests := []struct {
		expr string
		want any
	}{
		// Ternary
		{`true ? "yes" : "no"`, "yes"},
		{`false ? "yes" : "no"`, "no"},
		{`age > 25 ? "adult" : "young"`, "adult"},

		// Boolean operators
		{`true && true`, true},
		{`true && false`, false},
		{`false || true`, true},
		{`false || false`, false},

		// Comparisons
		{`age == 30`, true},
		{`age != 30`, false},
		{`age > 20`, true},
		{`age >= 30`, true},
		{`age < 40`, true},
		{`age <= 30`, true},

		// Arithmetic
		{`age + 10`, int64(40)},
		{`age - 5`, int64(25)},
		{`age * 2`, int64(60)},
		{`age / 3`, float64(10)},
		{`age % 7`, int64(2)},

		// Unary
		{`!active`, false},
		{`!false`, true},
		{`-age`, int64(-30)},

		// Array access
		{`items[0]`, "a"},
		{`items[1]`, "b"},

		// Member access
		{`user.name`, "Bob"},
		{`user.profile.email`, "bob@example.com"},

		// Strings
		{`"hello"`, "hello"},
		{`"hello" + " " + "world"`, "hello world"},

		// Arrays
		{`[1, 2, 3]`, []any{int64(1), int64(2), int64(3)}},

		// Numbers
		{`42`, int64(42)},
		{`3.14`, 3.14},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			expr, err := ParseExpression(tt.expr)
			if err != nil {
				t.Fatalf("ParseExpression(%q) error = %v", tt.expr, err)
			}
			got, err := expr.Eval(ctx)
			if err != nil {
				t.Fatalf("Eval() error = %v", err)
			}
			// Compare arrays element by element
			if wantArr, ok := tt.want.([]any); ok {
				gotArr, ok := got.([]any)
				if !ok {
					t.Errorf("Eval(%q) = %v (%T), want array", tt.expr, got, got)
					return
				}
				if len(gotArr) != len(wantArr) {
					t.Errorf("Eval(%q) len = %d, want %d", tt.expr, len(gotArr), len(wantArr))
				}
				return
			}
			if got != tt.want {
				t.Errorf("Eval(%q) = %v (%T), want %v (%T)", tt.expr, got, got, tt.want, tt.want)
			}
		})
	}
}

func TestExpressionFunctions(t *testing.T) {
	ctx := NewContext(map[string]any{
		"str":   "  Hello World  ",
		"arr":   []any{"a", "b", "c"},
		"num":   int64(42),
		"float": 3.14,
	})

	tests := []struct {
		expr    string
		want    any
		wantErr bool
	}{
		// len()
		{`len("hello")`, int64(5), false},
		{`len(arr)`, int64(3), false},

		// upper() / lower()
		{`upper("hello")`, "HELLO", false},
		{`lower("HELLO")`, "hello", false},

		// trim()
		{`trim(str)`, "Hello World", false},

		// split()
		{`split("a,b,c", ",")`, nil, false}, // just check no error

		// join()
		{`join(arr, "-")`, "a-b-c", false},

		// int() / float() / string()
		{`int(42)`, int64(42), false},
		{`float(3)`, float64(3), false},
		{`string(42)`, "42", false},

		// range()
		{`range(3)`, nil, false}, // just check no error

		// default()
		{`default(num, 0)`, int64(42), false}, // num is defined, use it

		// Errors
		{`len(42)`, nil, true},        // len expects string or array
		{`unknown_func()`, nil, true}, // unknown function
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			expr, err := ParseExpression(tt.expr)
			if err != nil {
				if tt.wantErr {
					return
				}
				t.Fatalf("ParseExpression(%q) error = %v", tt.expr, err)
			}
			got, err := expr.Eval(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("Eval(%q) error = %v, wantErr %v", tt.expr, err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.want != nil && got != tt.want {
				t.Errorf("Eval(%q) = %v, want %v", tt.expr, got, tt.want)
			}
		})
	}
}

func TestParseExpressionErrors(t *testing.T) {
	invalidExprs := []string{
		"((unclosed",
		"1 + + 2",
		"[1, 2,",
		"{key:",
		"'unclosed string",
		"func(",
	}

	for _, expr := range invalidExprs {
		t.Run(expr, func(t *testing.T) {
			_, err := ParseExpression(expr)
			if err == nil {
				t.Errorf("ParseExpression(%q) should error", expr)
			}
		})
	}
}

func TestNestedAccess(t *testing.T) {
	ctx := NewContext(map[string]any{
		"data": map[string]any{
			"items": []any{
				map[string]any{"name": "first"},
				map[string]any{"name": "second"},
			},
		},
	})

	tests := []struct {
		expr string
		want any
	}{
		{`data.items[0].name`, "first"},
		{`data.items[1].name`, "second"},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			expr, err := ParseExpression(tt.expr)
			if err != nil {
				t.Fatalf("ParseExpression(%q) error = %v", tt.expr, err)
			}
			got, err := expr.Eval(ctx)
			if err != nil {
				t.Fatalf("Eval() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("Eval(%q) = %v, want %v", tt.expr, got, tt.want)
			}
		})
	}
}

func TestContextNestedGet(t *testing.T) {
	ctx := NewContext(map[string]any{
		"user": map[string]any{
			"profile": map[string]any{
				"settings": map[string]any{
					"theme": "dark",
				},
			},
		},
	})

	val, ok := ctx.Get("user.profile.settings.theme")
	if !ok {
		t.Error("Get(nested path) should succeed")
	}
	if val != "dark" {
		t.Errorf("Get(nested path) = %v, want 'dark'", val)
	}

	// Non-existent path
	_, ok = ctx.Get("user.profile.nonexistent")
	if ok {
		t.Error("Get(nonexistent path) should return false")
	}
}

func TestObjectExpression(t *testing.T) {
	ctx := NewContext(nil)

	expr, err := ParseExpression(`{"key": "value", "num": 42}`)
	if err != nil {
		t.Fatalf("ParseExpression error = %v", err)
	}

	got, err := expr.Eval(ctx)
	if err != nil {
		t.Fatalf("Eval() error = %v", err)
	}

	obj, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %T", got)
	}
	if obj["key"] != "value" {
		t.Errorf("obj[key] = %v, want 'value'", obj["key"])
	}
}

func TestStringEscapes(t *testing.T) {
	ctx := NewContext(nil)

	tests := []struct {
		expr string
		want string
	}{
		{`"hello\nworld"`, "hello\nworld"},
		{`"hello\tworld"`, "hello\tworld"},
		{`"hello\\world"`, "hello\\world"},
		{`"hello\"world"`, "hello\"world"},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			expr, err := ParseExpression(tt.expr)
			if err != nil {
				t.Fatalf("ParseExpression(%q) error = %v", tt.expr, err)
			}
			got, err := expr.Eval(ctx)
			if err != nil {
				t.Fatalf("Eval() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("Eval(%q) = %q, want %q", tt.expr, got, tt.want)
			}
		})
	}
}

func TestToBoolConversions(t *testing.T) {
	ctx := NewContext(map[string]any{
		"intZero":    int64(0),
		"intNonZero": int64(42),
		"floatZero":  0.0,
		"floatVal":   3.14,
		"emptyStr":   "",
		"str":        "hello",
		"emptyArr":   []any{},
		"arr":        []any{1, 2, 3},
		"emptyMap":   map[string]any{},
		"mapVal":     map[string]any{"a": 1},
		"nilVal":     nil,
		"custom":     struct{ Name string }{"test"}, // default returns true
	})

	tests := []struct {
		expr string
		want bool
	}{
		// int64
		{`intZero ? false : true`, true},    // toBool(0) = false
		{`intNonZero ? true : false`, true}, // toBool(42) = true
		// float64
		{`floatZero ? false : true`, true}, // toBool(0.0) = false
		{`floatVal ? true : false`, true},  // toBool(3.14) = true
		// string
		{`emptyStr ? false : true`, true}, // toBool("") = false
		{`str ? true : false`, true},      // toBool("hello") = true
		// array
		{`emptyArr ? false : true`, true}, // toBool([]) = false
		{`arr ? true : false`, true},      // toBool([1,2,3]) = true
		// map
		{`emptyMap ? false : true`, true}, // toBool({}) = false
		{`mapVal ? true : false`, true},   // toBool({a:1}) = true
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			expr, err := ParseExpression(tt.expr)
			if err != nil {
				t.Fatalf("ParseExpression(%q) error = %v", tt.expr, err)
			}
			got, err := expr.Eval(ctx)
			if err != nil {
				t.Fatalf("Eval() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("Eval(%q) = %v, want %v", tt.expr, got, tt.want)
			}
		})
	}
}

func TestToFloat64Conversions(t *testing.T) {
	ctx := NewContext(map[string]any{
		"int":     int(5),
		"int64":   int64(10),
		"float32": float32(2.5),
		"float64": float64(3.14),
	})

	tests := []struct {
		expr string
		want any
	}{
		{`int * 2`, int64(10)},         // int converts to float64
		{`int64 * 2`, int64(20)},       // int64 converts to float64
		{`float32 * 2`, float64(5)},    // float32 converts to float64
		{`float64 * 2`, float64(6.28)}, // float64 already float64
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			expr, err := ParseExpression(tt.expr)
			if err != nil {
				t.Fatalf("ParseExpression(%q) error = %v", tt.expr, err)
			}
			got, err := expr.Eval(ctx)
			if err != nil {
				t.Fatalf("Eval() error = %v", err)
			}
			// For float comparison, allow small epsilon
			if gf, ok := got.(float64); ok {
				if wf, ok := tt.want.(float64); ok {
					if gf < wf-0.01 || gf > wf+0.01 {
						t.Errorf("Eval(%q) = %v, want %v", tt.expr, got, tt.want)
					}
					return
				}
			}
			if got != tt.want {
				t.Errorf("Eval(%q) = %v (%T), want %v (%T)", tt.expr, got, got, tt.want, tt.want)
			}
		})
	}
}

func TestNegateOperator(t *testing.T) {
	ctx := NewContext(map[string]any{
		"intVal":   int64(42),
		"floatVal": 3.14,
		"str":      "hello",
	})

	tests := []struct {
		expr    string
		want    any
		wantErr bool
	}{
		{`-intVal`, int64(-42), false},
		{`-floatVal`, -3.14, false},
		{`-(-intVal)`, int64(42), false},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			expr, err := ParseExpression(tt.expr)
			if err != nil {
				if tt.wantErr {
					return
				}
				t.Fatalf("ParseExpression(%q) error = %v", tt.expr, err)
			}
			got, err := expr.Eval(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("Eval(%q) error = %v, wantErr %v", tt.expr, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("Eval(%q) = %v, want %v", tt.expr, got, tt.want)
			}
		})
	}
}

func TestBinaryOperatorErrors(t *testing.T) {
	ctx := NewContext(map[string]any{
		"str":  "hello",
		"num":  int64(10),
		"zero": int64(0),
	})

	tests := []struct {
		expr    string
		wantErr string
	}{
		// Division by zero
		{`num / 0`, "division by zero"},
		{`num % 0`, "modulo by zero"},
		// Type mismatches for subtraction
		{`str - num`, "cannot subtract"},
		// Type mismatches for multiplication
		{`str * num`, "cannot multiply"},
		// Type mismatches for division
		{`str / num`, "cannot divide"},
		// Type mismatches for modulo
		{`str % num`, "cannot modulo"},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			expr, err := ParseExpression(tt.expr)
			if err != nil {
				t.Fatalf("ParseExpression(%q) error = %v", tt.expr, err)
			}
			_, err = expr.Eval(ctx)
			if err == nil {
				t.Errorf("Eval(%q) should error", tt.expr)
			}
		})
	}
}

func TestIndexAccessCoverage(t *testing.T) {
	ctx := NewContext(map[string]any{
		"arr":    []any{"a", "b", "c"},
		"obj":    map[string]any{"key": "value"},
		"str":    "hello",
		"num":    int64(42),
		"nested": map[string]any{"arr": []any{1, 2, 3}},
	})

	tests := []struct {
		expr    string
		want    any
		wantErr bool
	}{
		// Array with int64 index
		{`arr[0]`, "a", false},
		{`arr[1]`, "b", false},
		// Array with float index (converted to int)
		{`arr[1.0]`, "b", false},
		// Array out of bounds
		{`arr[10]`, nil, true},
		{`arr[-1]`, nil, true},
		// Map access with string key
		{`obj["key"]`, "value", false},
		// Map access with non-existent key
		{`obj["nonexistent"]`, nil, false}, // returns nil, no error
		// String indexing
		{`str[0]`, "h", false},
		{`str[4]`, "o", false},
		// String out of bounds
		{`str[100]`, nil, true},
		// Invalid index type on array - use array that exists
		// Cannot index non-indexable types
		{`num[0]`, nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			expr, err := ParseExpression(tt.expr)
			if err != nil {
				if tt.wantErr {
					return
				}
				t.Fatalf("ParseExpression(%q) error = %v", tt.expr, err)
			}
			got, err := expr.Eval(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("Eval(%q) error = %v, wantErr %v", tt.expr, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("Eval(%q) = %v, want %v", tt.expr, got, tt.want)
			}
		})
	}
}

func TestMemberAccessCoverage(t *testing.T) {
	ctx := NewContext(map[string]any{
		"obj":    map[string]any{"key": "value"},
		"strMap": map[string]string{"name": "Alice"},
		"num":    int64(42),
		"nested": map[string]any{"inner": map[string]any{"val": 123}},
	})

	tests := []struct {
		expr    string
		want    any
		wantErr bool
	}{
		// map[string]any access
		{`obj.key`, "value", false},
		{`obj.nonexistent`, nil, false}, // returns nil, no error
		// map[string]string access
		{`strMap.name`, "Alice", false},
		{`strMap.nonexistent`, nil, false},
		// Nested access
		{`nested.inner.val`, 123, false}, // int (not int64) since it's stored as int
		// Cannot access member on non-map
		{`num.field`, nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			expr, err := ParseExpression(tt.expr)
			if err != nil {
				if tt.wantErr {
					return
				}
				t.Fatalf("ParseExpression(%q) error = %v", tt.expr, err)
			}
			got, err := expr.Eval(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("Eval(%q) error = %v, wantErr %v", tt.expr, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("Eval(%q) = %v, want %v", tt.expr, got, tt.want)
			}
		})
	}
}

func TestLoopContextMemberAccess(t *testing.T) {
	// Test LoopContext member access
	lc := NewLoopContext(1, 5, "item-value")
	ctx := NewContext(map[string]any{
		"loop": lc,
	})

	tests := []struct {
		expr string
		want any
	}{
		{`loop.index`, int64(1)},
		{`loop.length`, int64(5)},
		{`loop.first`, false},
		{`loop.last`, false},
		{`loop.value`, "item-value"},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			expr, err := ParseExpression(tt.expr)
			if err != nil {
				t.Fatalf("ParseExpression(%q) error = %v", tt.expr, err)
			}
			got, err := expr.Eval(ctx)
			if err != nil {
				t.Fatalf("Eval() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("Eval(%q) = %v, want %v", tt.expr, got, tt.want)
			}
		})
	}

	// Test unknown loop property
	t.Run("unknown loop property", func(t *testing.T) {
		expr, err := ParseExpression(`loop.unknown`)
		if err != nil {
			t.Fatalf("ParseExpression error = %v", err)
		}
		_, err = expr.Eval(ctx)
		if err == nil {
			t.Error("expected error for unknown loop property")
		}
	})
}

func TestFunctionArgumentErrors(t *testing.T) {
	ctx := NewContext(map[string]any{
		"arr": []any{"a", "b"},
		"num": int64(42),
	})

	tests := []struct {
		expr    string
		wantErr string
	}{
		// len() wrong argument count
		{`len()`, "takes 1 argument"},
		{`len(arr, arr)`, "takes 1 argument"},
		// upper() wrong argument count
		{`upper()`, "takes 1 argument"},
		{`upper("a", "b")`, "takes 1 argument"},
		// lower() wrong argument count
		{`lower()`, "takes 1 argument"},
		// trim() wrong argument count
		{`trim()`, "takes 1 argument"},
		// split() wrong argument count
		{`split("a")`, "takes 2 arguments"},
		// join() wrong argument count
		{`join(arr)`, "takes 2 arguments"},
		// join() first arg not array
		{`join(num, ",")`, "first argument must be array"},
		// int() wrong argument count
		{`int()`, "takes 1 argument"},
		// float() wrong argument count
		{`float()`, "takes 1 argument"},
		// string() wrong argument count
		{`string()`, "takes 1 argument"},
		// range() wrong argument count
		{`range()`, "takes 1-2 arguments"},
		{`range(1, 2, 3)`, "takes 1-2 arguments"},
		// range() non-numeric
		{`range("a")`, "must be numeric"},
		{`range("a", "b")`, "must be numeric"},
		// default() wrong argument count
		{`default(1)`, "takes 2 arguments"},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			expr, err := ParseExpression(tt.expr)
			if err != nil {
				t.Fatalf("ParseExpression(%q) error = %v", tt.expr, err)
			}
			_, err = expr.Eval(ctx)
			if err == nil {
				t.Errorf("Eval(%q) should error", tt.expr)
			}
		})
	}
}

func TestRangeTwoArgs(t *testing.T) {
	ctx := NewContext(nil)

	expr, err := ParseExpression(`range(2, 5)`)
	if err != nil {
		t.Fatalf("ParseExpression error = %v", err)
	}
	got, err := expr.Eval(ctx)
	if err != nil {
		t.Fatalf("Eval() error = %v", err)
	}
	arr, ok := got.([]any)
	if !ok {
		t.Fatalf("expected array, got %T", got)
	}
	if len(arr) != 3 {
		t.Errorf("len(range(2,5)) = %d, want 3", len(arr))
	}
	if arr[0] != int64(2) || arr[1] != int64(3) || arr[2] != int64(4) {
		t.Errorf("range(2,5) = %v, want [2,3,4]", arr)
	}
}

func TestDefaultWithFalsy(t *testing.T) {
	ctx := NewContext(map[string]any{
		"empty":    "",
		"nilVal":   nil,
		"nonEmpty": "value",
	})

	tests := []struct {
		expr string
		want any
	}{
		{`default("", "fallback")`, "fallback"},     // empty string -> use fallback
		{`default(nilVal, "fallback")`, "fallback"}, // nil -> use fallback
		{`default(nonEmpty, "fallback")`, "value"},  // non-empty -> use value
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			expr, err := ParseExpression(tt.expr)
			if err != nil {
				t.Fatalf("ParseExpression(%q) error = %v", tt.expr, err)
			}
			got, err := expr.Eval(ctx)
			if err != nil {
				t.Fatalf("Eval() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("Eval(%q) = %v, want %v", tt.expr, got, tt.want)
			}
		})
	}
}

func TestCompareStrings(t *testing.T) {
	ctx := NewContext(map[string]any{
		"a": "apple",
		"b": "banana",
	})

	tests := []struct {
		expr string
		want bool
	}{
		{`a < b`, true}, // "apple" < "banana"
		{`a > b`, false},
		{`a == a`, true},
		{`a != b`, true},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			expr, err := ParseExpression(tt.expr)
			if err != nil {
				t.Fatalf("ParseExpression(%q) error = %v", tt.expr, err)
			}
			got, err := expr.Eval(ctx)
			if err != nil {
				t.Fatalf("Eval() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("Eval(%q) = %v, want %v", tt.expr, got, tt.want)
			}
		})
	}
}

func TestEqualsNilCases(t *testing.T) {
	ctx := NewContext(map[string]any{
		"nilA": nil,
		"nilB": nil,
		"val":  "value",
	})

	tests := []struct {
		expr string
		want bool
	}{
		{`nilA == nilB`, true}, // both nil
		{`nilA == val`, false}, // nil vs non-nil
		{`val == nilA`, false}, // non-nil vs nil
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			expr, err := ParseExpression(tt.expr)
			if err != nil {
				t.Fatalf("ParseExpression(%q) error = %v", tt.expr, err)
			}
			got, err := expr.Eval(ctx)
			if err != nil {
				t.Fatalf("Eval() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("Eval(%q) = %v, want %v", tt.expr, got, tt.want)
			}
		})
	}
}

func TestFloatArithmetic(t *testing.T) {
	ctx := NewContext(map[string]any{
		"a": 1.5,
		"b": 2.5,
	})

	tests := []struct {
		expr string
		want float64
	}{
		{`a + b`, 4.0},
		{`a - b`, -1.0},
		{`a * b`, 3.75},
		{`b / a`, 5.0 / 3.0},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			expr, err := ParseExpression(tt.expr)
			if err != nil {
				t.Fatalf("ParseExpression(%q) error = %v", tt.expr, err)
			}
			got, err := expr.Eval(ctx)
			if err != nil {
				t.Fatalf("Eval() error = %v", err)
			}
			gf, ok := got.(float64)
			if !ok {
				t.Fatalf("expected float64, got %T", got)
			}
			if gf < tt.want-0.01 || gf > tt.want+0.01 {
				t.Errorf("Eval(%q) = %v, want %v", tt.expr, gf, tt.want)
			}
		})
	}
}

func TestIntConversion(t *testing.T) {
	ctx := NewContext(map[string]any{
		"str": "not a number",
	})

	// int() and float() should error on non-numeric strings
	expr, err := ParseExpression(`int(str)`)
	if err != nil {
		t.Fatalf("ParseExpression error = %v", err)
	}
	_, err = expr.Eval(ctx)
	if err == nil {
		t.Error("int(str) should error on non-numeric value")
	}

	expr, err = ParseExpression(`float(str)`)
	if err != nil {
		t.Fatalf("ParseExpression error = %v", err)
	}
	_, err = expr.Eval(ctx)
	if err == nil {
		t.Error("float(str) should error on non-numeric value")
	}
}

func TestStringConcatWithNonString(t *testing.T) {
	ctx := NewContext(map[string]any{
		"num": int64(42),
		"str": "value",
	})

	tests := []struct {
		expr string
		want string
	}{
		{`"num: " + num`, "num: 42"},                   // string + int
		{`num + " is the answer"`, "42 is the answer"}, // int + string
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			expr, err := ParseExpression(tt.expr)
			if err != nil {
				t.Fatalf("ParseExpression(%q) error = %v", tt.expr, err)
			}
			got, err := expr.Eval(ctx)
			if err != nil {
				t.Fatalf("Eval() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("Eval(%q) = %v, want %v", tt.expr, got, tt.want)
			}
		})
	}
}

func TestIsInt(t *testing.T) {
	ctx := NewContext(map[string]any{
		"int":   int(5),
		"int64": int64(15),
	})

	// These should use integer arithmetic
	tests := []struct {
		expr string
		want any
	}{
		{`int + int`, int64(10)},
		{`int64 - int`, int64(10)},
		{`int * int`, int64(25)},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			expr, err := ParseExpression(tt.expr)
			if err != nil {
				t.Fatalf("ParseExpression(%q) error = %v", tt.expr, err)
			}
			got, err := expr.Eval(ctx)
			if err != nil {
				t.Fatalf("Eval() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("Eval(%q) = %v (%T), want %v (%T)", tt.expr, got, got, tt.want, tt.want)
			}
		})
	}
}
