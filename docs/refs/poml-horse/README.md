# poml-horse (private seed, poml-go-sdk local)

Goal
- POML-first diagram pipeline to emit renderer-agnostic scene JSON and render high-quality isometric (2.5D) graphs. MVP: render an isometric graph from POML via deck.gl/luma.gl with solid aesthetics and export SVG/PNG.

Contents
- `seed.poml`: schema, architecture, work plan, MVP, and a minimal sample `<diagram>`.
- `seed.prompt.poml`: automation brief for OpenAI/Codex to drive toward ~50% MVP.
- `scene_exporter_stub.go`: tiny proof to parse the seed and emit a scene JSON skeleton (not part of the SDK build).

Notes
- Private research; do not publish. Mirrors the outline kept in the Atlas repo but colocated here for poml-go-sdk work.
- Schema is renderer-agnostic; adapters will target deck.gl/luma.gl first, with optional Graphviz/Mermaid/Three.js/glTF routes.***
