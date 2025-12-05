package token

import "testing"

func TestDefaultCounter(t *testing.T) {
	counter := DefaultCounter{}

	tests := []struct {
		input string
		want  int64
	}{
		{"", 0},
		{"a", 1},
		{"test", 1},
		{"hello world", 3},
		{"The quick brown fox jumps over the lazy dog", 11},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := counter.Count(tt.input)
			if err != nil {
				t.Errorf("Count() error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("Count(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		input string
		want  int64
	}{
		{"", 0},
		{"test", 1},
		{"hello world", 3},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := EstimateTokens(tt.input)
			if got != tt.want {
				t.Errorf("EstimateTokens(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestCountChars(t *testing.T) {
	tests := []struct {
		input string
		want  int64
	}{
		{"", 0},
		{"abc", 3},
		{"hello world", 11},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := CountChars(tt.input)
			if got != tt.want {
				t.Errorf("CountChars(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestCountWords(t *testing.T) {
	tests := []struct {
		input string
		want  int64
	}{
		{"", 0},
		{"word", 1},
		{"hello world", 2},
		{"  multiple   spaces  ", 2},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := CountWords(tt.input)
			if got != tt.want {
				t.Errorf("CountWords(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestTruncateToCharLimit(t *testing.T) {
	tests := []struct {
		input string
		limit int64
		want  string
	}{
		{"short", 10, "short"},
		{"hello world", 5, "he..."},
		{"hello world", 8, "hello..."},
		{"hello world testing", 11, "hello..."},
	}

	for _, tt := range tests {
		name := tt.input[:min(len(tt.input), 10)]
		t.Run(name, func(t *testing.T) {
			got := TruncateToCharLimit(tt.input, tt.limit)
			if len(got) > int(tt.limit) {
				t.Errorf("TruncateToCharLimit() length %d exceeds limit %d", len(got), tt.limit)
			}
		})
	}
}

func TestEnforceLimitWithPriority(t *testing.T) {
	contents := []PrioritizedContent{
		{Content: "low priority content", Priority: 1, ID: "low"},
		{Content: "high priority", Priority: 10, ID: "high"},
		{Content: "medium", Priority: 5, ID: "med"},
	}

	t.Run("no limit", func(t *testing.T) {
		result := EnforceLimitWithPriority(contents, 0)
		if len(result) != len(contents) {
			t.Errorf("expected all content with no limit")
		}
	})

	t.Run("tight limit", func(t *testing.T) {
		result := EnforceLimitWithPriority(contents, 15)
		if len(result) == 0 {
			t.Errorf("expected at least one result")
		}
		// Should prioritize high priority content
		if result[0].ID != "high" {
			t.Errorf("expected high priority content first, got %s", result[0].ID)
		}
	})

	t.Run("large limit", func(t *testing.T) {
		result := EnforceLimitWithPriority(contents, 1000)
		if len(result) != 3 {
			t.Errorf("expected all content with large limit, got %d", len(result))
		}
	})
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
