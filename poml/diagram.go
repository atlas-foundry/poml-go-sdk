package poml

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Diagram represents a diagram block with graph and camera/layer metadata.
type Diagram struct {
	ID         string         `xml:"id,attr"`
	Projection string         `xml:"projection,attr"`
	Layout     string         `xml:"layout,attr"`
	Unit       string         `xml:"unit,attr"`
	Graph      DiagramGraph   `xml:"graph"`
	Layers     []DiagramLayer `xml:"layer"`
	Camera     DiagramCamera  `xml:"camera"`
	Attrs      []xml.Attr     `xml:",any,attr"`
}

// DiagramGraph holds nodes and edges.
type DiagramGraph struct {
	Nodes []DiagramNode `xml:"node"`
	Edges []DiagramEdge `xml:"edge"`
}

// DiagramNode describes a node in the diagram.
type DiagramNode struct {
	ID          string         `xml:"id,attr"`
	Label       string         `xml:"label,attr"`
	Group       string         `xml:"group,attr"`
	Owner       string         `xml:"owner,attr"`
	Weight      string         `xml:"weight,attr"`
	PctComplete string         `xml:"pct_complete,attr"`
	X           string         `xml:"x,attr"`
	Y           string         `xml:"y,attr"`
	Z           string         `xml:"z,attr"`
	Styles      []DiagramStyle `xml:"style"`
	Data        []DiagramData  `xml:"data"`
	Attrs       []xml.Attr     `xml:",any,attr"`
}

// DiagramEdge describes a directed/undirected edge.
type DiagramEdge struct {
	From     string         `xml:"from,attr"`
	To       string         `xml:"to,attr"`
	Kind     string         `xml:"kind,attr"`
	Directed *bool          `xml:"directed,attr"`
	Weight   string         `xml:"weight,attr"`
	Styles   []DiagramStyle `xml:"style"`
	Attrs    []xml.Attr     `xml:",any,attr"`
}

// DiagramStyle carries styling hints.
type DiagramStyle struct {
	Color     string     `xml:"color,attr"`
	Shape     string     `xml:"shape,attr"`
	Size      string     `xml:"size,attr"`
	Stroke    string     `xml:"stroke,attr"`
	Width     string     `xml:"width,attr"`
	Dash      string     `xml:"dash,attr"`
	Curvature string     `xml:"curvature,attr"`
	Texture   string     `xml:"texture,attr"`
	Attrs     []xml.Attr `xml:",any,attr"`
}

// DiagramLayer describes background/overlay layers.
type DiagramLayer struct {
	ID    string     `xml:"id,attr"`
	Z     string     `xml:"z,attr"`
	Kind  string     `xml:"kind,attr"`
	Attrs []xml.Attr `xml:",any,attr"`
}

// DiagramCamera defines camera positioning.
type DiagramCamera struct {
	Azimuth   string     `xml:"azimuth,attr"`
	Elevation string     `xml:"elevation,attr"`
	Distance  string     `xml:"distance,attr"`
	Attrs     []xml.Attr `xml:",any,attr"`
}

// DiagramData carries arbitrary JSON-ish payload keyed by name.
type DiagramData struct {
	Key  string `xml:"key,attr"`
	Body string `xml:",innerxml"`
}

// Scene is a normalized representation for renderer adapters.
type Scene struct {
	ID     string         `json:"id"`
	Nodes  []SceneNode    `json:"nodes"`
	Edges  []SceneEdge    `json:"edges"`
	Layers []SceneLayer   `json:"layers,omitempty"`
	Camera SceneCamera    `json:"camera"`
	Meta   map[string]any `json:"meta,omitempty"`
}

type SceneNode struct {
	ID          string            `json:"id"`
	Label       string            `json:"label,omitempty"`
	Owner       string            `json:"owner,omitempty"`
	Group       string            `json:"group,omitempty"`
	Weight      string            `json:"weight,omitempty"`
	PctComplete string            `json:"pct_complete,omitempty"`
	Position    [3]float64        `json:"position"`
	Style       map[string]string `json:"style,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	Attrs       map[string]string `json:"attrs,omitempty"`
}

type SceneEdge struct {
	From     string            `json:"from"`
	To       string            `json:"to"`
	Kind     string            `json:"kind,omitempty"`
	Directed bool              `json:"directed"`
	Weight   string            `json:"weight,omitempty"`
	Style    map[string]string `json:"style,omitempty"`
	Attrs    map[string]string `json:"attrs,omitempty"`
}

type SceneLayer struct {
	ID    string            `json:"id"`
	Z     string            `json:"z,omitempty"`
	Kind  string            `json:"kind,omitempty"`
	Attrs map[string]string `json:"attrs,omitempty"`
}

type SceneCamera struct {
	Azimuth   string `json:"azimuth,omitempty"`
	Elevation string `json:"elevation,omitempty"`
	Distance  string `json:"distance,omitempty"`
}

// SceneRenderer represents a renderer that can consume normalized scenes (deck.gl, Graphviz, etc.).
type SceneRenderer interface {
	Render(scene Scene) error
}

// SceneExportOptions control conversion from Diagram to Scene.
type SceneExportOptions struct {
	// Deterministic sorts nodes/edges/layers for golden tests; when false, preserves input order.
	Deterministic *bool
}

var defaultSceneExportOptions = SceneExportOptions{Deterministic: ptrBool(true)}

// DiagramToScene converts a Diagram into a normalized Scene.
func DiagramToScene(d Diagram) (Scene, error) {
	return DiagramToSceneWithOptions(d, defaultSceneExportOptions)
}

// DiagramToSceneWithOptions converts a Diagram into a Scene with export controls.
func DiagramToSceneWithOptions(d Diagram, opts SceneExportOptions) (Scene, error) {
	deterministic := true
	if opts.Deterministic != nil {
		deterministic = *opts.Deterministic
	}

	scene := Scene{
		ID:     d.ID,
		Camera: SceneCamera{Azimuth: d.Camera.Azimuth, Elevation: d.Camera.Elevation, Distance: d.Camera.Distance},
	}
	nodes := append([]DiagramNode(nil), d.Graph.Nodes...)
	edges := append([]DiagramEdge(nil), d.Graph.Edges...)
	layers := append([]DiagramLayer(nil), d.Layers...)
	if deterministic {
		sort.Slice(nodes, func(i, j int) bool { return nodes[i].ID < nodes[j].ID })
		sort.Slice(edges, func(i, j int) bool {
			ai, aj := edges[i], edges[j]
			if ai.From != aj.From {
				return ai.From < aj.From
			}
			if ai.To != aj.To {
				return ai.To < aj.To
			}
			if ai.Kind != aj.Kind {
				return ai.Kind < aj.Kind
			}
			return ai.Weight < aj.Weight
		})
		sort.Slice(layers, func(i, j int) bool {
			if layers[i].ID == layers[j].ID {
				return layers[i].Kind < layers[j].Kind
			}
			return layers[i].ID < layers[j].ID
		})
	}
	for _, n := range nodes {
		pos := [3]float64{parseFloat(n.X), parseFloat(n.Y), parseFloat(n.Z)}
		node := SceneNode{
			ID:          n.ID,
			Label:       n.Label,
			Owner:       n.Owner,
			Group:       n.Group,
			Weight:      n.Weight,
			PctComplete: n.PctComplete,
			Position:    pos,
			Style:       styleMap(n.Styles),
			Attrs:       attrsMap(n.Attrs),
		}
		for _, ds := range n.Data {
			if ds.Key == "tags" {
				if tags, ok := parseStringArray(ds.Body); ok {
					node.Tags = tags
				}
			}
		}
		scene.Nodes = append(scene.Nodes, node)
	}
	for _, e := range edges {
		directed := false
		if e.Directed != nil {
			directed = *e.Directed
		}
		scene.Edges = append(scene.Edges, SceneEdge{
			From:     e.From,
			To:       e.To,
			Kind:     e.Kind,
			Directed: directed,
			Weight:   e.Weight,
			Style:    styleMap(e.Styles),
			Attrs:    attrsMap(e.Attrs),
		})
	}
	for _, l := range layers {
		scene.Layers = append(scene.Layers, SceneLayer{
			ID:    l.ID,
			Z:     l.Z,
			Kind:  l.Kind,
			Attrs: attrsMap(l.Attrs),
		})
	}
	return scene, nil
}

// ValidateDiagram performs structural validation of a diagram.
func ValidateDiagram(d Diagram) error {
	var errs []string
	var details []ValidationDetail
	if strings.TrimSpace(d.ID) == "" {
		errs = append(errs, "diagram missing id")
		details = append(details, ValidationDetail{Element: ElementDiagram, Field: "id", Message: "missing id"})
	}
	nodeIDs := make(map[string]struct{})
	for i, n := range d.Graph.Nodes {
		if strings.TrimSpace(n.ID) == "" {
			errs = append(errs, fmt.Sprintf("node[%d] missing id", i))
			details = append(details, ValidationDetail{Element: ElementDiagram, Field: "node.id", Message: fmt.Sprintf("node %d missing id", i)})
		} else {
			if _, dup := nodeIDs[n.ID]; dup {
				errs = append(errs, "duplicate node id "+n.ID)
				details = append(details, ValidationDetail{Element: ElementDiagram, Field: "node.id", Message: fmt.Sprintf("duplicate node id %s", n.ID)})
			}
			nodeIDs[n.ID] = struct{}{}
		}
	}
	for i, e := range d.Graph.Edges {
		if strings.TrimSpace(e.From) == "" || strings.TrimSpace(e.To) == "" {
			errs = append(errs, fmt.Sprintf("edge[%d] missing from/to", i))
			details = append(details, ValidationDetail{Element: ElementDiagram, Field: "edge.from_to", Message: fmt.Sprintf("edge %d missing from/to", i)})
		} else {
			if _, ok := nodeIDs[e.From]; !ok {
				errs = append(errs, "edge from references missing node "+e.From)
				details = append(details, ValidationDetail{Element: ElementDiagram, Field: "edge.from", Message: fmt.Sprintf("edge %d references missing node %s", i, e.From)})
			}
			if _, ok := nodeIDs[e.To]; !ok {
				errs = append(errs, "edge to references missing node "+e.To)
				details = append(details, ValidationDetail{Element: ElementDiagram, Field: "edge.to", Message: fmt.Sprintf("edge %d references missing node %s", i, e.To)})
			}
		}
		if e.Directed == nil {
			errs = append(errs, fmt.Sprintf("edge[%d] missing directed flag", i))
			details = append(details, ValidationDetail{Element: ElementDiagram, Field: "edge.directed", Message: fmt.Sprintf("edge %d missing directed flag", i)})
		}
	}
	if len(errs) > 0 {
		return &ValidationError{Issues: errs, Details: details}
	}
	return nil
}

func parseFloat(val string) float64 {
	if val == "" {
		return 0
	}
	f, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return 0
	}
	return f
}

func styleMap(styles []DiagramStyle) map[string]string {
	var m map[string]string
	set := func(k, v string) {
		if v == "" {
			return
		}
		if m == nil {
			m = make(map[string]string)
		}
		m[k] = v
	}
	for _, st := range styles {
		set("color", st.Color)
		set("shape", st.Shape)
		set("size", st.Size)
		set("stroke", st.Stroke)
		set("width", st.Width)
		set("dash", st.Dash)
		set("curvature", st.Curvature)
		set("texture", st.Texture)
		for _, a := range st.Attrs {
			set(a.Name.Local, a.Value)
		}
	}
	return m
}

func attrsMap(attrs []xml.Attr) map[string]string {
	if len(attrs) == 0 {
		return nil
	}
	m := make(map[string]string, len(attrs))
	for _, a := range attrs {
		m[a.Name.Local] = a.Value
	}
	return m
}

func parseStringArray(body string) ([]string, bool) {
	var arr []string
	if err := json.Unmarshal([]byte(body), &arr); err != nil {
		return nil, false
	}
	return arr, true
}

func ptrBool(v bool) *bool {
	return &v
}
