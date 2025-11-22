//go:build ignore

// scene_exporter_stub is a tiny proof-of-concept to parse the seed POML diagram
// and emit a normalized scene JSON for renderer pipelines. It is not wired into
// the SDK; run manually with `go run scene_exporter_stub.go`.
package main

import (
	"encoding/json"
	"fmt"

	poml "github.com/atlas-foundry/poml-go-sdk/poml"
)

type Scene struct {
	ID     string       `json:"id"`
	Nodes  []SceneNode  `json:"nodes"`
	Edges  []SceneEdge  `json:"edges"`
	Layers []SceneLayer `json:"layers,omitempty"`
	Camera SceneCamera  `json:"camera"`
}

type SceneNode struct {
	ID          string             `json:"id"`
	Label       string             `json:"label,omitempty"`
	Owner       string             `json:"owner,omitempty"`
	Group       string             `json:"group,omitempty"`
	Weight      string             `json:"weight,omitempty"`
	PctComplete string             `json:"pct_complete,omitempty"`
	Position    [3]float64         `json:"position"`
	Style       map[string]string  `json:"style,omitempty"`
	Tags        []string           `json:"tags,omitempty"`
}

type SceneEdge struct {
	From     string            `json:"from"`
	To       string            `json:"to"`
	Kind     string            `json:"kind,omitempty"`
	Directed bool              `json:"directed"`
	Weight   string            `json:"weight,omitempty"`
	Style    map[string]string `json:"style,omitempty"`
}

type SceneLayer struct {
	ID   string `json:"id"`
	Z    string `json:"z,omitempty"`
	Kind string `json:"kind,omitempty"`
}

type SceneCamera struct {
	Azimuth   string `json:"azimuth,omitempty"`
	Elevation string `json:"elevation,omitempty"`
	Distance  string `json:"distance,omitempty"`
}

func main() {
	path := "docs/refs/poml-horse/seed.poml"
	doc, err := poml.ParseFile(path)
	if err != nil {
		panic(err)
	}
	scene := sceneFromDoc(doc)
	out, err := json.MarshalIndent(scene, "", "  ")
	if err != nil {
		panic(err)
	}
	fmt.Println(string(out))
}

// sceneFromDoc is intentionally narrow: it looks for the sample diagram; once
// the SDK defines diagram structs, replace this with typed parsing.
func sceneFromDoc(_ poml.Document) Scene {
	scene := Scene{
		ID:     "chain-sample",
		Camera: SceneCamera{Azimuth: "35", Elevation: "30", Distance: "8"},
		Layers: []SceneLayer{{ID: "grid", Z: "-1", Kind: "grid"}},
	}
	scene.Nodes = []SceneNode{
		{
			ID:          "chain-001",
			Label:       "telemetry hooks",
			Owner:       "Vishwakarma",
			Weight:      "0.13",
			PctComplete: "0.45",
			Position:    [3]float64{0, 0, 0},
			Style:       map[string]string{"color": "#4fd1c5", "shape": "hex", "size": "1.2", "stroke": "#0f172a"},
			Tags:        []string{"telemetry", "plan-driver"},
		},
		{
			ID:          "chain-005",
			Label:       "metadata sweep",
			Owner:       "Librarian",
			Weight:      "0.10",
			PctComplete: "0.60",
			Position:    [3]float64{2, 1, 0},
			Style:       map[string]string{"color": "#a78bfa", "shape": "circle", "size": "1.0", "stroke": "#0f172a"},
			Tags:        []string{"metadata", "ci"},
		},
	}
	scene.Edges = []SceneEdge{
		{
			From:     "chain-001",
			To:       "chain-005",
			Kind:     "depends",
			Directed: true,
			Weight:   "0.4",
			Style:    map[string]string{"stroke": "#475569", "width": "2", "dash": "solid", "curvature": "0.1"},
		},
	}
	return scene
}
