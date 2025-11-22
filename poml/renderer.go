package poml

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// Renderer renders a normalized Scene to a target representation.
type Renderer interface {
	Render(Scene) ([]byte, error)
}

// DeckGLRenderer emits the Scene as JSON compatible with deck.gl consumers.
type DeckGLRenderer struct{}

// Render marshals the Scene to JSON.
func (r DeckGLRenderer) Render(scene Scene) ([]byte, error) {
	return json.MarshalIndent(scene, "", "  ")
}

// GraphvizRenderer emits Graphviz DOT text for a Scene.
type GraphvizRenderer struct {
	// Directed overrides the scene edge directed flag; when nil, uses edge.Directed.
	Directed *bool
}

// Render converts the scene into DOT. Deterministic ordering is preserved/sorted for stability.
func (r GraphvizRenderer) Render(scene Scene) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteString("digraph G {\n")
	// Nodes
	nodes := append([]SceneNode(nil), scene.Nodes...)
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].ID < nodes[j].ID })
	for _, n := range nodes {
		fmt.Fprintf(&buf, "  %q%s;\n", n.ID, buildDOTNodeAttrs(n))
	}
	// Edges
	edges := append([]SceneEdge(nil), scene.Edges...)
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].From != edges[j].From {
			return edges[i].From < edges[j].From
		}
		return edges[i].To < edges[j].To
	})
	for _, e := range edges {
		directed := e.Directed
		if r.Directed != nil {
			directed = *r.Directed
		}
		arrow := "->"
		if !directed {
			arrow = "--"
		}
		attrs := buildDOTAttrs(map[string]string{
			"label":    e.Kind,
			"color":    e.Style["stroke"],
			"penwidth": e.Style["width"],
			"style":    e.Style["dash"],
			"weight":   e.Weight,
		})
		fmt.Fprintf(&buf, "  %q %s %q%s;\n", e.From, arrow, e.To, attrs)
	}
	buf.WriteString("}\n")
	return buf.Bytes(), nil
}

func buildDOTNodeAttrs(n SceneNode) string {
	attrs := map[string]string{}
	label := n.Label
	if label == "" {
		label = n.ID
	}
	attrs["label"] = label
	// Map common shapes
	switch strings.ToLower(n.Style["shape"]) {
	case "circle":
		attrs["shape"] = "circle"
	case "square", "box":
		attrs["shape"] = "box"
	case "hex", "hexagon":
		attrs["shape"] = "hexagon"
	case "diamond":
		attrs["shape"] = "diamond"
	}
	if fill := n.Style["color"]; fill != "" {
		attrs["fillcolor"] = fill
		attrs["style"] = appendStyle(attrs["style"], "filled")
	}
	if stroke := n.Style["stroke"]; stroke != "" {
		attrs["color"] = stroke
	}
	attrs["pos"] = fmt.Sprintf("%.3f,%.3f!", n.Position[0], n.Position[1])
	return buildDOTAttrs(attrs)
}

func buildDOTAttrs(m map[string]string) string {
	var parts []string
	for k, v := range m {
		if strings.TrimSpace(v) == "" {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s=%q", k, v))
	}
	if len(parts) == 0 {
		return ""
	}
	sort.Strings(parts)
	return " [" + strings.Join(parts, ",") + "]"
}

func appendStyle(existing, extra string) string {
	if strings.TrimSpace(extra) == "" {
		return existing
	}
	if strings.TrimSpace(existing) == "" {
		return extra
	}
	return existing + "," + extra
}
