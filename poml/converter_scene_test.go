package poml

import (
	"encoding/json"
	"testing"
)

const multiDiagram = `<poml>
  <diagram id="g1">
    <graph>
      <node id="a1" x="0" y="0" z="0"/>
    </graph>
    <camera azimuth="0" elevation="0" distance="1"/>
  </diagram>
  <diagram id="g2">
    <graph>
      <node id="b1" x="1" y="1" z="0"/>
      <edge from="b1" to="b1" kind="loop" directed="false"/>
    </graph>
    <camera azimuth="10" elevation="20" distance="30"/>
  </diagram>
</poml>`

func TestConvertSceneMultipleDiagrams(t *testing.T) {
	doc, err := ParseString(multiDiagram)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	sceneAny, err := Convert(doc, FormatScene, ConvertOptions{})
	if err != nil {
		t.Fatalf("convert scene: %v", err)
	}
	scenes, ok := sceneAny.([]Scene)
	if !ok {
		t.Fatalf("expected []Scene, got %T", sceneAny)
	}
	if len(scenes) != 2 {
		t.Fatalf("expected 2 scenes, got %d", len(scenes))
	}
	if scenes[0].ID != "g1" || scenes[1].ID != "g2" {
		t.Fatalf("unexpected scene ordering: %+v", scenes)
	}
	if scenes[0].Camera.Distance != "1" || scenes[1].Camera.Azimuth != "10" {
		t.Fatalf("cameras not preserved: %+v %+v", scenes[0].Camera, scenes[1].Camera)
	}
}

func TestConvertSceneJSONMultipleDiagrams(t *testing.T) {
	doc, err := ParseString(multiDiagram)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	jsonAny, err := Convert(doc, FormatSceneJSON, ConvertOptions{})
	if err != nil {
		t.Fatalf("convert scenejson: %v", err)
	}
	jsonBody, ok := jsonAny.(string)
	if !ok {
		t.Fatalf("expected string scene JSON, got %T", jsonAny)
	}
	var scenes []Scene
	if err := json.Unmarshal([]byte(jsonBody), &scenes); err != nil {
		t.Fatalf("unmarshal scenes: %v", err)
	}
	if len(scenes) != 2 || scenes[0].ID != "g1" || scenes[1].ID != "g2" {
		t.Fatalf("unexpected scenes: %+v", scenes)
	}
}
