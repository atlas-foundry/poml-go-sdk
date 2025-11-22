package poml

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const diagramSample = `<poml>
  <diagram id="chain-sample" projection="isometric" layout="dagre" unit="u">
    <graph>
      <node id="chain-001" label="telemetry hooks" owner="Vishwakarma" weight="0.13" pct_complete="0.45" x="0" y="0" z="0">
        <style color="#4fd1c5" shape="hex" size="1.2" stroke="#0f172a"/>
        <data key="tags">["telemetry","plan-driver"]</data>
      </node>
      <node id="chain-005" label="metadata sweep" owner="Librarian" weight="0.10" pct_complete="0.60" x="2" y="1" z="0">
        <style color="#a78bfa" shape="circle" size="1.0" stroke="#0f172a"/>
        <data key="tags">["metadata","ci"]</data>
      </node>
      <edge from="chain-001" to="chain-005" kind="depends" directed="true" weight="0.4">
        <style stroke="#475569" width="2" dash="solid" curvature="0.1"/>
      </edge>
    </graph>
    <layer id="grid" z="-1" kind="grid"/>
    <camera azimuth="35" elevation="30" distance="8"/>
  </diagram>
</poml>`

func TestParseDiagramAndExportScene(t *testing.T) {
	doc, err := ParseString(diagramSample)
	if err != nil {
		t.Fatalf("parse diagram: %v", err)
	}
	if len(doc.Diagrams) != 1 {
		t.Fatalf("expected 1 diagram, got %d", len(doc.Diagrams))
	}
	if err := ValidateDiagram(doc.Diagrams[0]); err != nil {
		t.Fatalf("validate diagram: %v", err)
	}
	scene, err := DiagramToScene(doc.Diagrams[0])
	if err != nil {
		t.Fatalf("diagram to scene: %v", err)
	}
	if got := len(scene.Nodes); got != 2 {
		t.Fatalf("expected 2 nodes, got %d", got)
	}
	if got := len(scene.Edges); got != 1 {
		t.Fatalf("expected 1 edge, got %d", got)
	}
	if scene.Camera.Azimuth != "35" || scene.Layers == nil || len(scene.Layers) != 1 {
		t.Fatalf("camera or layers missing: %#v %#v", scene.Camera, scene.Layers)
	}
}

func TestValidateDiagramErrors(t *testing.T) {
	bad := Diagram{
		Graph: DiagramGraph{
			Nodes: []DiagramNode{{ID: "a"}, {ID: "a"}},
			Edges: []DiagramEdge{{From: "missing", To: "a"}},
		},
	}
	if err := ValidateDiagram(bad); err == nil {
		t.Fatalf("expected validation error")
	} else if !strings.Contains(err.Error(), "directed") {
		t.Fatalf("expected missing directed flag to be reported, got %v", err)
	}
}

func TestDocumentValidateIncludesDiagram(t *testing.T) {
	directed := true
	doc := Document{
		Meta:  Meta{ID: "diagram.demo", Version: "1", Owner: "me"},
		Role:  Block{Body: "r"},
		Tasks: []Block{{Body: "t"}},
		Diagrams: []Diagram{{
			ID: "g1",
			Graph: DiagramGraph{
				Nodes: []DiagramNode{{ID: "a"}},
				Edges: []DiagramEdge{{From: "a", To: "missing", Directed: &directed}},
			},
		}},
	}
	if err := doc.Validate(); err == nil {
		t.Fatalf("expected validation error for diagram edge reference")
	} else if !strings.Contains(err.Error(), "diagram") {
		t.Fatalf("expected diagram validation surfaced, got %v", err)
	}
}

const diagramOutOfOrder = `<poml>
  <diagram id="ordering" projection="isometric">
    <graph>
      <node id="b" label="second" x="1" y="0" z="0"/>
      <node id="a" label="first" x="0" y="0" z="0"/>
      <edge from="b" to="a" kind="relates" directed="false" />
      <edge from="a" to="b" kind="relates" directed="true" />
    </graph>
    <layer id="overlay" kind="heatmap"/>
    <layer id="base" kind="grid"/>
    <camera azimuth="0" elevation="0" distance="1"/>
  </diagram>
</poml>`

func TestDiagramToSceneDeterministicOrdering(t *testing.T) {
	doc, err := ParseString(diagramOutOfOrder)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	scene, err := DiagramToScene(doc.Diagrams[0])
	if err != nil {
		t.Fatalf("to scene: %v", err)
	}
	if len(scene.Nodes) != 2 || scene.Nodes[0].ID != "a" || scene.Nodes[1].ID != "b" {
		t.Fatalf("nodes not sorted deterministically: %+v", scene.Nodes)
	}
	if len(scene.Edges) != 2 || scene.Edges[0].From != "a" || scene.Edges[0].To != "b" {
		t.Fatalf("edges not sorted deterministically: %+v", scene.Edges)
	}
	if len(scene.Layers) != 2 || scene.Layers[0].ID != "base" || scene.Layers[1].ID != "overlay" {
		t.Fatalf("layers not sorted deterministically: %+v", scene.Layers)
	}
}

func TestDiagramToScenePreservesOrderWhenRequested(t *testing.T) {
	doc, err := ParseString(diagramOutOfOrder)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	deterministic := false
	scene, err := DiagramToSceneWithOptions(doc.Diagrams[0], SceneExportOptions{Deterministic: &deterministic})
	if err != nil {
		t.Fatalf("to scene: %v", err)
	}
	if len(scene.Nodes) != 2 || scene.Nodes[0].ID != "b" || scene.Nodes[1].ID != "a" {
		t.Fatalf("expected original node order, got %+v", scene.Nodes)
	}
	if len(scene.Edges) != 2 || scene.Edges[0].From != "b" || scene.Edges[0].To != "a" {
		t.Fatalf("expected original edge order, got %+v", scene.Edges)
	}
	if len(scene.Layers) != 2 || scene.Layers[0].ID != "overlay" || scene.Layers[1].ID != "base" {
		t.Fatalf("expected original layer order, got %+v", scene.Layers)
	}
}

func TestDiagramToSceneAttrsAndDirectedDefault(t *testing.T) {
	src := `<poml><diagram id="attrs"><graph>
  <node id="n1" label="x" x="0" y="0" z="0"><style texture="wood" custom="yes"/></node>
  <edge from="n1" to="n1" kind="loop"><style stroke="blue"/></edge>
</graph><layer id="l1" kind="overlay" custom="c1"/><camera/></diagram></poml>`
	doc, err := ParseString(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	scene, err := DiagramToScene(doc.Diagrams[0])
	if err != nil {
		t.Fatalf("to scene: %v", err)
	}
	if scene.Nodes[0].Style["texture"] != "wood" || scene.Nodes[0].Style["custom"] != "yes" {
		t.Fatalf("node style attrs missing: %+v", scene.Nodes[0].Style)
	}
	if scene.Edges[0].Directed {
		t.Fatalf("directed should default false when missing")
	}
	if scene.Edges[0].Style["stroke"] != "blue" {
		t.Fatalf("edge style missing: %+v", scene.Edges[0].Style)
	}
	if scene.Layers[0].Attrs["custom"] != "c1" {
		t.Fatalf("layer attrs missing: %+v", scene.Layers[0].Attrs)
	}
}

func TestGoldenDiagramToScene(t *testing.T) {
	cases := []struct {
		name       string
		pomlFile   string
		sceneFile  string
		shouldFail bool
	}{
		{name: "chain", pomlFile: "chain_sample.poml", sceneFile: "chain_sample_scene.json"},
		{name: "star", pomlFile: "star_sample.poml", sceneFile: "star_sample_scene.json"},
		{name: "grid", pomlFile: "grid_sample.poml", sceneFile: "grid_sample_scene.json"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pomlPath := filepath.Join("testdata", "diagrams", tc.pomlFile)
			body, err := os.ReadFile(pomlPath)
			if err != nil {
				t.Fatalf("read poml fixture: %v", err)
			}
			doc, err := ParseString(string(body))
			if err != nil {
				t.Fatalf("parse poml: %v", err)
			}
			if len(doc.Diagrams) != 1 {
				t.Fatalf("expected 1 diagram, got %d", len(doc.Diagrams))
			}
			if err := ValidateDiagram(doc.Diagrams[0]); err != nil {
				t.Fatalf("validate diagram: %v", err)
			}
			scene, err := DiagramToScene(doc.Diagrams[0])
			if err != nil {
				t.Fatalf("diagram to scene: %v", err)
			}
			expectedPath := filepath.Join("testdata", "diagrams", tc.sceneFile)
			wantBody, err := os.ReadFile(expectedPath)
			if err != nil {
				t.Fatalf("read expected scene: %v", err)
			}
			var want Scene
			if err := json.Unmarshal(wantBody, &want); err != nil {
				t.Fatalf("unmarshal expected scene: %v", err)
			}
			if !reflect.DeepEqual(scene, want) {
				gotBytes, _ := json.MarshalIndent(scene, "", "  ")
				t.Fatalf("scene mismatch.\n got:\n%s\nwant:\n%s", string(gotBytes), string(wantBody))
			}
		})
	}
}

func TestInvalidDiagramFixtures(t *testing.T) {
	cases := []string{"missing_ids.poml"}
	for _, file := range cases {
		t.Run(file, func(t *testing.T) {
			pomlPath := filepath.Join("testdata", "diagrams_invalid", file)
			body, err := os.ReadFile(pomlPath)
			if err != nil {
				t.Fatalf("read poml fixture: %v", err)
			}
			doc, err := ParseString(string(body))
			if err != nil {
				t.Fatalf("parse poml: %v", err)
			}
			if len(doc.Diagrams) != 1 {
				t.Fatalf("expected 1 diagram, got %d", len(doc.Diagrams))
			}
			if err := ValidateDiagram(doc.Diagrams[0]); err == nil {
				t.Fatalf("expected validation error for %s", file)
			}
		})
	}
}
