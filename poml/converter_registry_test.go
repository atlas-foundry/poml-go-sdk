package poml

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestDefaultRegistryPOMLToSceneJSON(t *testing.T) {
	reg := NewConverterRegistry()
	registerDefaultConverters(reg)

	ctx := context.Background()
	diagramsAny, err := reg.Convert(ctx, "poml", "diagram", diagramSample, nil)
	if err != nil {
		t.Fatalf("poml->diagram: %v", err)
	}
	diagrams, ok := diagramsAny.([]Diagram)
	if !ok || len(diagrams) != 1 {
		t.Fatalf("expected []Diagram of len 1, got %T %#v", diagramsAny, diagramsAny)
	}

	sceneAny, err := reg.Convert(ctx, "diagram", "scene", diagrams, nil)
	if err != nil {
		t.Fatalf("diagram->scene: %v", err)
	}
	scenes, ok := sceneAny.([]Scene)
	if !ok || len(scenes) != 1 {
		t.Fatalf("expected []Scene of len 1, got %T %#v", sceneAny, sceneAny)
	}

	jsonAny, err := reg.Convert(ctx, "scene", "scenejson", scenes, map[string]any{"pretty": false})
	if err != nil {
		t.Fatalf("scene->scenejson: %v", err)
	}
	jsonBody, ok := jsonAny.([]byte)
	if !ok {
		t.Fatalf("expected []byte JSON, got %T", jsonAny)
	}
	if !strings.Contains(string(jsonBody), `"id":"chain-sample"`) {
		t.Fatalf("scene JSON missing id: %s", string(jsonBody))
	}
}

func TestSceneJSONRoundTripToPOML(t *testing.T) {
	reg := NewConverterRegistry()
	registerDefaultConverters(reg)

	ctx := context.Background()
	diagramsAny, err := reg.Convert(ctx, "poml", "diagram", diagramSample, nil)
	if err != nil {
		t.Fatalf("poml->diagram: %v", err)
	}
	sceneAny, err := reg.Convert(ctx, "diagram", "scene", diagramsAny, nil)
	if err != nil {
		t.Fatalf("diagram->scene: %v", err)
	}
	jsonAny, err := reg.Convert(ctx, "scene", "scenejson", sceneAny, nil)
	if err != nil {
		t.Fatalf("scene->scenejson: %v", err)
	}

	backSceneAny, err := reg.Convert(ctx, "scenejson", "scene", jsonAny, nil)
	if err != nil {
		t.Fatalf("scenejson->scene: %v", err)
	}
	var scenes []Scene
	switch v := backSceneAny.(type) {
	case Scene:
		scenes = []Scene{v}
	case []Scene:
		scenes = v
	default:
		t.Fatalf("unexpected type from scenejson->scene: %T", backSceneAny)
	}
	if len(scenes) != 1 || scenes[0].ID != "chain-sample" {
		t.Fatalf("unexpected scenes: %#v", scenes)
	}

	backDiagAny, err := reg.Convert(ctx, "scene", "diagram", scenes, nil)
	if err != nil {
		t.Fatalf("scene->diagram: %v", err)
	}
	pomlAny, err := reg.Convert(ctx, "diagram", "poml", backDiagAny, nil)
	if err != nil {
		t.Fatalf("diagram->poml: %v", err)
	}
	pomlText, ok := pomlAny.(string)
	if !ok {
		t.Fatalf("expected string POML, got %T", pomlAny)
	}
	if !strings.Contains(pomlText, "<diagram") || !strings.Contains(pomlText, "chain-001") {
		t.Fatalf("round-tripped POML missing content: %s", pomlText)
	}
}

func TestRegisterDuplicateConverter(t *testing.T) {
	reg := NewConverterRegistry()
	conv := basicConverter{from: "a", to: "b", fn: func(context.Context, any, map[string]any) (any, error) { return nil, nil }}
	if err := reg.Register(conv); err != nil {
		t.Fatalf("first register failed: %v", err)
	}
	if err := reg.Register(conv); !errors.Is(err, ConverterExistsError) {
		t.Fatalf("expected duplicate error, got %v", err)
	}
}

func TestDiagramRoundTripWithBaseDocument(t *testing.T) {
	reg := NewConverterRegistry()
	registerDefaultConverters(reg)

	const pomlDoc = `<poml>
  <meta><id>diagram.doc</id><version>1.0</version><owner>qa</owner></meta>
  <role>diagram role</role>
  <task>diagram task</task>
  <diagram id="diag-attrs" custom="root-attr">
    <graph>
      <node id="n1" x="1" y="2" z="3" custom="yes"/>
    </graph>
    <camera azimuth="10" elevation="20" distance="30" tilt="5"/>
  </diagram>
</poml>`

	ctx := context.Background()
	diagramsAny, err := reg.Convert(ctx, "poml", "diagram", pomlDoc, nil)
	if err != nil {
		t.Fatalf("poml->diagram: %v", err)
	}
	diagrams := diagramsAny.([]Diagram)
	baseDoc, err := ParseString(pomlDoc)
	if err != nil {
		t.Fatalf("parse base: %v", err)
	}

	sceneAny, err := reg.Convert(ctx, "diagram", "scene", diagrams, nil)
	if err != nil {
		t.Fatalf("diagram->scene: %v", err)
	}
	scenes := sceneAny.([]Scene)
	if scenes[0].Meta == nil || scenes[0].Meta["diagram_attrs"] == nil || scenes[0].Meta["camera_attrs"] == nil {
		t.Fatalf("expected diagram and camera attrs in scene meta: %#v", scenes[0].Meta)
	}

	backDiagAny, err := reg.Convert(ctx, "scene", "diagram", scenes, nil)
	if err != nil {
		t.Fatalf("scene->diagram: %v", err)
	}
	backDiagrams := backDiagAny.([]Diagram)
	if len(backDiagrams[0].Attrs) == 0 || len(backDiagrams[0].Camera.Attrs) == 0 {
		t.Fatalf("expected attrs round-tripped onto diagram: %#v", backDiagrams[0])
	}

	outPomlAny, err := reg.Convert(ctx, "diagram", "poml", backDiagrams, map[string]any{"base_document": baseDoc})
	if err != nil {
		t.Fatalf("diagram->poml with base doc: %v", err)
	}
	outPoml := outPomlAny.(string)
	parsed, err := ParseString(outPoml)
	if err != nil {
		t.Fatalf("parse round-tripped poml: %v", err)
	}
	if parsed.Meta.ID != "diagram.doc" || strings.TrimSpace(parsed.Role.Body) != "diagram role" || len(parsed.Tasks) != 1 {
		t.Fatalf("context not preserved in round-trip: meta=%#v role=%q tasks=%d", parsed.Meta, parsed.Role.Body, len(parsed.Tasks))
	}
}
