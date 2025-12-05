package poml

import "testing"

func TestParseComponents(t *testing.T) {
	tests := []struct {
		spec         string
		wantEnabled  []string
		wantDisabled []string
	}{
		{"", nil, nil},
		{"table", []string{"table"}, nil},
		{"+table", []string{"table"}, nil},
		{"-table", nil, []string{"table"}},
		{"table,image", []string{"table", "image"}, nil},
		{"table,-image", []string{"table"}, []string{"image"}},
		{"+table,+image,-video", []string{"table", "image"}, []string{"video"}},
		{" table , +image , -video ", []string{"table", "image"}, []string{"video"}},
		{",,table,,", []string{"table"}, nil},
	}

	for _, tt := range tests {
		t.Run(tt.spec, func(t *testing.T) {
			cs := ParseComponents(tt.spec)

			for _, name := range tt.wantEnabled {
				if !cs.Enabled[name] {
					t.Errorf("expected %q to be enabled", name)
				}
			}
			for _, name := range tt.wantDisabled {
				if !cs.Disabled[name] {
					t.Errorf("expected %q to be disabled", name)
				}
			}
		})
	}
}

func TestComponentSetIsEnabled(t *testing.T) {
	tests := []struct {
		spec      string
		component string
		want      bool
	}{
		{"", "table", true},               // empty spec = all enabled
		{"table", "table", true},          // explicitly enabled
		{"table", "image", false},         // not in enabled list
		{"-table", "table", false},        // explicitly disabled
		{"-table", "image", true},         // not disabled, no enabled list
		{"table,-image", "table", true},   // enabled
		{"table,-image", "image", false},  // disabled
		{"table,-image", "video", false},  // not in enabled list
		{"-image,-video", "table", true},  // not disabled, no enabled list
		{"-image,-video", "image", false}, // disabled
	}

	for _, tt := range tests {
		name := tt.spec + "_" + tt.component
		t.Run(name, func(t *testing.T) {
			cs := ParseComponents(tt.spec)
			got := cs.IsEnabled(tt.component)
			if got != tt.want {
				t.Errorf("ComponentSet(%q).IsEnabled(%q) = %v, want %v",
					tt.spec, tt.component, got, tt.want)
			}
		})
	}
}

func TestFilterDocumentByComponents(t *testing.T) {
	doc := &Document{
		Images:   []Image{{Src: "test.png"}},
		Audios:   []Media{{Src: "test.mp3"}},
		Videos:   []Media{{Src: "test.mp4"}},
		Diagrams: []Diagram{{ID: "test-diagram"}},
		Elements: []Element{
			{Type: ElementImage, Index: 0},
			{Type: ElementAudio, Index: 0},
			{Type: ElementVideo, Index: 0},
			{Type: ElementDiagram, Index: 0},
			{Type: ElementRole, Index: -1},
		},
	}

	t.Run("empty spec returns original", func(t *testing.T) {
		result := FilterDocumentByComponents(doc, "")
		if result != doc {
			t.Error("expected same document for empty spec")
		}
	})

	t.Run("disable image", func(t *testing.T) {
		result := FilterDocumentByComponents(doc, "-image")
		if result.Images != nil {
			t.Error("expected Images to be nil")
		}
		if result.Audios == nil {
			t.Error("expected Audios to remain")
		}

		// Check elements filtered
		for _, el := range result.Elements {
			if el.Type == ElementImage {
				t.Error("expected image elements to be filtered out")
			}
		}
	})

	t.Run("disable multiple", func(t *testing.T) {
		result := FilterDocumentByComponents(doc, "-image,-audio,-video")
		if result.Images != nil {
			t.Error("expected Images to be nil")
		}
		if result.Audios != nil {
			t.Error("expected Audios to be nil")
		}
		if result.Videos != nil {
			t.Error("expected Videos to be nil")
		}
		if result.Diagrams == nil {
			t.Error("expected Diagrams to remain")
		}
	})

	t.Run("non-component elements preserved", func(t *testing.T) {
		result := FilterDocumentByComponents(doc, "-image,-audio,-video,-diagram")
		hasRole := false
		for _, el := range result.Elements {
			if el.Type == ElementRole {
				hasRole = true
			}
		}
		if !hasRole {
			t.Error("expected role element to be preserved")
		}
	})
}
