package poml

import (
	"strings"
)

// ComponentSet represents enabled and disabled components parsed from a components spec.
type ComponentSet struct {
	Enabled  map[string]bool
	Disabled map[string]bool
}

// ParseComponents parses a components specification string.
// Format: "component1,+component2,-component3"
// - No prefix or "+" prefix means enabled
// - "-" prefix means disabled
func ParseComponents(spec string) ComponentSet {
	cs := ComponentSet{
		Enabled:  make(map[string]bool),
		Disabled: make(map[string]bool),
	}

	if spec == "" {
		return cs
	}

	parts := strings.Split(spec, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		if strings.HasPrefix(part, "-") {
			name := strings.TrimPrefix(part, "-")
			if name != "" {
				cs.Disabled[name] = true
			}
		} else {
			name := strings.TrimPrefix(part, "+")
			if name != "" {
				cs.Enabled[name] = true
			}
		}
	}

	return cs
}

// IsEnabled returns true if the component is explicitly enabled or not disabled.
// If no components are specified (empty spec), all components are enabled.
func (cs ComponentSet) IsEnabled(name string) bool {
	if cs.Disabled[name] {
		return false
	}
	// If no enabled list specified, everything not disabled is enabled
	if len(cs.Enabled) == 0 {
		return true
	}
	return cs.Enabled[name]
}

// elementToComponent maps ElementType to component names for filtering.
var elementToComponent = map[ElementType]string{
	ElementTable:        "table",
	ElementFolder:       "folder",
	ElementWebpage:      "webpage",
	ElementConversation: "conversation",
	ElementImage:        "image",
	ElementAudio:        "audio",
	ElementVideo:        "video",
	ElementDiagram:      "diagram",
	ElementOp:           "op",
	ElementFigure:       "figure",
	ElementObject:       "object",
	ElementText:         "text",
}

// FilterDocumentByComponents returns a copy of the document with disabled components removed.
// This is applied during conversion, not parsing, to preserve the original document.
func FilterDocumentByComponents(doc *Document, spec string) *Document {
	if spec == "" {
		return doc
	}

	cs := ParseComponents(spec)
	if len(cs.Disabled) == 0 && len(cs.Enabled) == 0 {
		return doc
	}

	// Create a shallow copy
	filtered := *doc

	// Filter elements slice
	var filteredElements []Element
	for _, el := range doc.Elements {
		component, hasMapping := elementToComponent[el.Type]
		if hasMapping && !cs.IsEnabled(component) {
			continue // Skip disabled component
		}
		filteredElements = append(filteredElements, el)
	}
	filtered.Elements = filteredElements

	// Filter typed slices based on components
	if !cs.IsEnabled("image") {
		filtered.Images = nil
	}
	if !cs.IsEnabled("audio") {
		filtered.Audios = nil
	}
	if !cs.IsEnabled("video") {
		filtered.Videos = nil
	}
	if !cs.IsEnabled("diagram") {
		filtered.Diagrams = nil
	}
	if !cs.IsEnabled("op") {
		filtered.Ops = nil
	}
	if !cs.IsEnabled("figure") {
		filtered.Figures = nil
	}
	if !cs.IsEnabled("object") {
		filtered.Objects = nil
	}
	if !cs.IsEnabled("text") {
		filtered.Texts = nil
	}

	return &filtered
}
