package stylesheet

import "testing"

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantLen int
		wantErr bool
	}{
		{"empty", "", 0, false},
		{"single rule", `{"hint": {"syntax": "markdown"}}`, 1, false},
		{"multiple rules", `{"hint": {"syntax": "markdown"}, ".important": {"priority": "1"}}`, 2, false},
		{"invalid json", `{invalid}`, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ss, err := Parse(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(ss.Rules) != tt.wantLen {
				t.Errorf("Parse() got %d rules, want %d", len(ss.Rules), tt.wantLen)
			}
		})
	}
}

func TestMatchSelector(t *testing.T) {
	tests := []struct {
		selector  string
		tagName   string
		className string
		want      bool
	}{
		{"hint", "hint", "", true},
		{"hint", "task", "", false},
		{"Hint", "hint", "", true}, // case insensitive
		{".important", "hint", "important", true},
		{".important", "hint", "other important stuff", true},
		{".important", "hint", "other", false},
		{".important", "hint", "", false},
		{"", "hint", "", false},
	}

	for _, tt := range tests {
		name := tt.selector + "_" + tt.tagName + "_" + tt.className
		t.Run(name, func(t *testing.T) {
			got := MatchSelector(tt.selector, tt.tagName, tt.className)
			if got != tt.want {
				t.Errorf("MatchSelector(%q, %q, %q) = %v, want %v",
					tt.selector, tt.tagName, tt.className, got, tt.want)
			}
		})
	}
}

func TestStylesheetApply(t *testing.T) {
	ss, err := Parse(`{
		"hint": {"syntax": "markdown", "priority": "1"},
		".important": {"priority": "10"}
	}`)
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	tests := []struct {
		tagName   string
		className string
		wantProps map[string]string
	}{
		{"hint", "", map[string]string{"syntax": "markdown", "priority": "1"}},
		{"task", "", map[string]string{}},
		{"hint", "important", map[string]string{"syntax": "markdown", "priority": "10"}}, // class overrides
		{"task", "important", map[string]string{"priority": "10"}},
	}

	for _, tt := range tests {
		name := tt.tagName + "_" + tt.className
		t.Run(name, func(t *testing.T) {
			got := ss.Apply(tt.tagName, tt.className)
			if len(got) != len(tt.wantProps) {
				t.Errorf("Apply() got %d props, want %d", len(got), len(tt.wantProps))
			}
			for k, v := range tt.wantProps {
				if got[k] != v {
					t.Errorf("Apply()[%q] = %q, want %q", k, got[k], v)
				}
			}
		})
	}
}

func TestSpecificity(t *testing.T) {
	tests := []struct {
		selector string
		want     int
	}{
		{"hint", 1},
		{".important", 10},
	}

	for _, tt := range tests {
		t.Run(tt.selector, func(t *testing.T) {
			got := Specificity(tt.selector)
			if got != tt.want {
				t.Errorf("Specificity(%q) = %d, want %d", tt.selector, got, tt.want)
			}
		})
	}
}

func TestMerge(t *testing.T) {
	ss1, _ := Parse(`{"hint": {"a": "1"}}`)
	ss2, _ := Parse(`{"task": {"b": "2"}}`)

	merged := Merge(ss1, ss2)
	if len(merged.Rules) != 2 {
		t.Errorf("Merge() got %d rules, want 2", len(merged.Rules))
	}
}
