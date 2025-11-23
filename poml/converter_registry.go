package poml

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// Converter turns input of one format into another (e.g., poml -> diagram -> scene -> scenejson).
// It is intentionally generic to keep the SDK pluggable for downstream renderers and importers.
type Converter interface {
	From() string
	To() string
	Convert(ctx context.Context, input any, opts map[string]any) (any, error)
}

// ConverterRegistry is a threadsafe registry for converters.
type ConverterRegistry struct {
	mu         sync.RWMutex
	converters map[string]Converter
}

// NewConverterRegistry builds an empty registry.
func NewConverterRegistry() *ConverterRegistry {
	return &ConverterRegistry{converters: make(map[string]Converter)}
}

// ConverterExistsError indicates a duplicate registration attempt.
var ConverterExistsError = errors.New("converter already registered")

// Register adds a converter. Returns ConverterExistsError when a from->to pair already exists.
func (r *ConverterRegistry) Register(conv Converter) error {
	if conv == nil {
		return errors.New("converter is nil")
	}
	key := converterKey(conv.From(), conv.To())
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.converters[key]; exists {
		return fmt.Errorf("%w: %s", ConverterExistsError, key)
	}
	r.converters[key] = conv
	return nil
}

// List returns descriptors for registered converters.
func (r *ConverterRegistry) List() []ConverterDescriptor {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]ConverterDescriptor, 0, len(r.converters))
	for _, c := range r.converters {
		out = append(out, ConverterDescriptor{From: strings.ToLower(c.From()), To: strings.ToLower(c.To())})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].From == out[j].From {
			return out[i].To < out[j].To
		}
		return out[i].From < out[j].From
	})
	return out
}

// ConverterDescriptor captures a registered mapping.
type ConverterDescriptor struct {
	From string
	To   string
}

// Convert dispatches to a registered converter.
func (r *ConverterRegistry) Convert(ctx context.Context, from, to string, input any, opts map[string]any) (any, error) {
	key := converterKey(from, to)
	r.mu.RLock()
	conv, ok := r.converters[key]
	r.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("no converter for %s", key)
	}
	return conv.Convert(ctx, input, opts)
}

// DefaultConverterRegistry is pre-populated with built-in converters for poml/diagram/scene.
var DefaultConverterRegistry = newDefaultConverterRegistry()

func newDefaultConverterRegistry() *ConverterRegistry {
	reg := NewConverterRegistry()
	registerDefaultConverters(reg)
	return reg
}

func converterKey(from, to string) string {
	return strings.ToLower(from) + "->" + strings.ToLower(to)
}

// registerDefaultConverters wires built-ins onto the provided registry.
func registerDefaultConverters(reg *ConverterRegistry) {
	// ignore duplicate errors to allow idempotent init in tests
	_ = reg.Register(basicConverter{
		from: "poml",
		to:   "diagram",
		fn: func(_ context.Context, input any, _ map[string]any) (any, error) {
			switch v := input.(type) {
			case string:
				doc, err := ParseString(v)
				if err != nil {
					return nil, err
				}
				return doc.Diagrams, nil
			case []byte:
				doc, err := ParseReader(strings.NewReader(string(v)))
				if err != nil {
					return nil, err
				}
				return doc.Diagrams, nil
			case Document:
				return v.Diagrams, nil
			default:
				return nil, fmt.Errorf("poml->diagram converter expects string, []byte, or Document, got %T", input)
			}
		},
	})
	_ = reg.Register(basicConverter{
		from: "diagram",
		to:   "poml",
		fn: func(_ context.Context, input any, opts map[string]any) (any, error) {
			indent := "  "
			if v, ok := opts["indent"].(string); ok && v != "" {
				indent = v
			}
			var diagrams []Diagram
			switch v := input.(type) {
			case Diagram:
				diagrams = []Diagram{v}
			case []Diagram:
				diagrams = v
			default:
				return nil, fmt.Errorf("diagram->poml converter expects Diagram or []Diagram, got %T", input)
			}
			baseDoc := Document{}
			if v, ok := opts["base_document"]; ok {
				switch b := v.(type) {
				case Document:
					baseDoc = b
				case *Document:
					if b != nil {
						baseDoc = *b
					}
				default:
					return nil, fmt.Errorf("base_document must be Document or *Document, got %T", v)
				}
			}
			baseDoc.Diagrams = diagrams
			var sb strings.Builder
			if err := baseDoc.EncodeWithOptions(&sb, EncodeOptions{Indent: indent, IncludeHeader: true, PreserveOrder: true}); err != nil {
				return nil, err
			}
			return sb.String(), nil
		},
	})
	_ = reg.Register(basicConverter{
		from: "diagram",
		to:   "scene",
		fn: func(_ context.Context, input any, opts map[string]any) (any, error) {
			exportOpts := defaultSceneExportOptions
			if v, ok := opts["scene_export"].(SceneExportOptions); ok {
				exportOpts = v
			}
			switch v := input.(type) {
			case Diagram:
				return DiagramToSceneWithOptions(v, exportOpts)
			case []Diagram:
				out := make([]Scene, 0, len(v))
				for _, d := range v {
					scene, err := DiagramToSceneWithOptions(d, exportOpts)
					if err != nil {
						return nil, err
					}
					out = append(out, scene)
				}
				return out, nil
			default:
				return nil, fmt.Errorf("diagram->scene converter expects Diagram or []Diagram, got %T", input)
			}
		},
	})
	_ = reg.Register(basicConverter{
		from: "scene",
		to:   "diagram",
		fn: func(_ context.Context, input any, _ map[string]any) (any, error) {
			switch v := input.(type) {
			case Scene:
				return sceneToDiagram(v), nil
			case []Scene:
				out := make([]Diagram, 0, len(v))
				for _, sc := range v {
					out = append(out, sceneToDiagram(sc))
				}
				return out, nil
			default:
				return nil, fmt.Errorf("scene->diagram converter expects Scene or []Scene, got %T", input)
			}
		},
	})
	_ = reg.Register(basicConverter{
		from: "scene",
		to:   "scenejson",
		fn: func(_ context.Context, input any, opts map[string]any) (any, error) {
			pretty := true
			if v, ok := opts["pretty"].(bool); ok {
				pretty = v
			}
			marshal := func(v any) ([]byte, error) {
				if pretty {
					return json.MarshalIndent(v, "", "  ")
				}
				return json.Marshal(v)
			}
			switch v := input.(type) {
			case Scene:
				return marshal(v)
			case []Scene:
				return marshal(v)
			default:
				return nil, fmt.Errorf("scene->scenejson converter expects Scene or []Scene, got %T", input)
			}
		},
	})
	_ = reg.Register(basicConverter{
		from: "scenejson",
		to:   "scene",
		fn: func(_ context.Context, input any, _ map[string]any) (any, error) {
			switch v := input.(type) {
			case string:
				return decodeSceneJSON([]byte(v))
			case []byte:
				return decodeSceneJSON(v)
			default:
				return nil, fmt.Errorf("scenejson->scene converter expects string or []byte, got %T", input)
			}
		},
	})
}

type basicConverter struct {
	from string
	to   string
	fn   func(ctx context.Context, input any, opts map[string]any) (any, error)
}

func (c basicConverter) From() string { return c.from }
func (c basicConverter) To() string   { return c.to }
func (c basicConverter) Convert(ctx context.Context, input any, opts map[string]any) (any, error) {
	return c.fn(ctx, input, opts)
}

func sceneToDiagram(scene Scene) Diagram {
	diagram := Diagram{
		ID: scene.ID,
		Graph: DiagramGraph{
			Nodes: make([]DiagramNode, 0, len(scene.Nodes)),
			Edges: make([]DiagramEdge, 0, len(scene.Edges)),
		},
		Layers: make([]DiagramLayer, 0, len(scene.Layers)),
		Camera: DiagramCamera{
			Azimuth:   scene.Camera.Azimuth,
			Elevation: scene.Camera.Elevation,
			Distance:  scene.Camera.Distance,
		},
	}
	if m := attrsFromMeta(scene.Meta, "diagram_attrs"); len(m) > 0 {
		diagram.Attrs = m
	}
	if m := attrsFromMeta(scene.Meta, "camera_attrs"); len(m) > 0 {
		diagram.Camera.Attrs = m
	}
	for _, n := range scene.Nodes {
		node := DiagramNode{
			ID:          n.ID,
			Label:       n.Label,
			Group:       n.Group,
			Owner:       n.Owner,
			Weight:      n.Weight,
			PctComplete: n.PctComplete,
			X:           formatFloat(n.Position[0]),
			Y:           formatFloat(n.Position[1]),
			Z:           formatFloat(n.Position[2]),
			Attrs:       attrsFromMap(n.Attrs),
		}
		if len(n.Style) > 0 {
			node.Styles = append(node.Styles, styleFromMap(n.Style))
		}
		if len(n.Tags) > 0 {
			if data, err := json.Marshal(n.Tags); err == nil {
				node.Data = append(node.Data, DiagramData{Key: "tags", Body: string(data)})
			}
		}
		diagram.Graph.Nodes = append(diagram.Graph.Nodes, node)
	}
	for _, e := range scene.Edges {
		diagram.Graph.Edges = append(diagram.Graph.Edges, DiagramEdge{
			From:     e.From,
			To:       e.To,
			Kind:     e.Kind,
			Directed: ptrBool(e.Directed),
			Weight:   e.Weight,
			Styles:   stylesFromMap(e.Style),
			Attrs:    attrsFromMap(e.Attrs),
		})
	}
	for _, l := range scene.Layers {
		diagram.Layers = append(diagram.Layers, DiagramLayer{
			ID:    l.ID,
			Z:     l.Z,
			Kind:  l.Kind,
			Attrs: attrsFromMap(l.Attrs),
		})
	}
	return diagram
}

func styleFromMap(m map[string]string) DiagramStyle {
	ds := DiagramStyle{}
	attrs := make(map[string]string)
	for k, v := range m {
		switch strings.ToLower(k) {
		case "color":
			ds.Color = v
		case "shape":
			ds.Shape = v
		case "size":
			ds.Size = v
		case "stroke":
			ds.Stroke = v
		case "width":
			ds.Width = v
		case "dash":
			ds.Dash = v
		case "curvature":
			ds.Curvature = v
		case "texture":
			ds.Texture = v
		default:
			attrs[k] = v
		}
	}
	if len(attrs) > 0 {
		ds.Attrs = attrsFromMap(attrs)
	}
	return ds
}

func stylesFromMap(m map[string]string) []DiagramStyle {
	if len(m) == 0 {
		return nil
	}
	return []DiagramStyle{styleFromMap(m)}
}

func attrsFromMap(m map[string]string) []xml.Attr {
	if len(m) == 0 {
		return nil
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	attrs := make([]xml.Attr, 0, len(keys))
	for _, k := range keys {
		attrs = append(attrs, xml.Attr{Name: xml.Name{Local: k}, Value: m[k]})
	}
	return attrs
}

func formatFloat(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}

func decodeSceneJSON(body []byte) (any, error) {
	trim := strings.TrimSpace(string(body))
	if strings.HasPrefix(trim, "{") {
		var scene Scene
		if err := json.Unmarshal(body, &scene); err != nil {
			return nil, err
		}
		return scene, nil
	}
	var scenes []Scene
	if err := json.Unmarshal(body, &scenes); err != nil {
		return nil, err
	}
	return scenes, nil
}

func attrsFromMeta(meta map[string]any, key string) []xml.Attr {
	if len(meta) == 0 {
		return nil
	}
	raw, ok := meta[key]
	if !ok || raw == nil {
		return nil
	}
	m := make(map[string]string)
	switch v := raw.(type) {
	case map[string]string:
		for k, val := range v {
			m[k] = val
		}
	case map[string]any:
		for k, val := range v {
			if s, ok := val.(string); ok {
				m[k] = s
			}
		}
	}
	return attrsFromMap(m)
}
