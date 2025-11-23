package poml

import (
	"context"
	"sync"
	"testing"
)

// Ensure ConverterRegistry handles concurrent Register/List without panicking or races.
func TestConverterRegistryConcurrency(t *testing.T) {
	reg := NewConverterRegistry()
	pairs := []struct {
		from string
		to   string
	}{
		{"poml", "diagram"},
		{"diagram", "scene"},
		{"scene", "scenejson"},
		{"poml", "openai_chat"},
		{"poml", "dict"},
	}

	wg := sync.WaitGroup{}
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			for _, p := range pairs {
				_ = reg.Register(basicConverter{
					from: p.from,
					to:   p.to,
					fn: func(_ context.Context, input any, _ map[string]any) (any, error) {
						return map[string]any{"ok": idx}, nil
					},
				})
			}
		}(i)
	}

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = reg.List()
		}()
	}
	wg.Wait()

	// Ensure listing sees all formats.
	list := reg.List()
	for _, p := range pairs {
		found := false
		for _, d := range list {
			if d.From == p.from && d.To == p.to {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected converter %s->%s to be registered", p.from, p.to)
		}
	}
}

// Ensure Convert can be called in parallel on the same Document safely.
func TestParallelConvert(t *testing.T) {
	doc := Document{
		Meta:  Meta{ID: "parallel", Version: "1", Owner: "oss"},
		Role:  Block{Body: "r"},
		Tasks: []Block{{Body: "t"}},
		Elements: []Element{
			{Type: ElementMeta},
			{Type: ElementRole},
			{Type: ElementTask, Index: 0},
		},
	}

	formats := []Format{FormatDict, FormatMessageDict, FormatOpenAIChat}
	wg := sync.WaitGroup{}
	for i := 0; i < 20; i++ {
		for _, f := range formats {
			wg.Add(1)
			go func(fmt Format) {
				defer wg.Done()
				if _, err := Convert(doc, fmt, ConvertOptions{}); err != nil {
					t.Errorf("parallel convert %s: %v", fmt, err)
				}
			}(f)
		}
	}
	wg.Wait()
}
