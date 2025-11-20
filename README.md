# poml-go-sdk (Atlas plugin)

Goal: Go SDK for POML with API/behavior parity to the Microsoft Python SDK (https://microsoft.github.io/poml/latest/python/).

Current scope
- Parser/encoder for POML (strict XML, CDATA-safe, preserves meta/role/task/input/document/style blocks).
- Helpers: parse from string/file, role text accessor, task body extraction, input walker.
- Tests: golden sample, round-trip encode, malformed error coverage, large document parsing.

Next steps
1) Flesh out full AST and builder/serializer parity (visitor/walker, validation hooks, whitespace/order preservation knobs).
2) Expand fixtures to mirror Python SDK examples and edge cases (bad attributes, missing tags, CDATA nesting).
3) Add CI workflow and coverage thresholds; publish usage docs for atlas imports.

Usage in atlas
- Local replace is set in the root go.mod: `replace plugins/poml-go-sdk => ./plugins/poml-go-sdk`.
- After adding as a real submodule/repo, drop the replace and pin a version: `go get plugins/poml-go-sdk@vX.Y.Z`.

If you intend to commit this as a submodule:
1) From repo root: `git submodule add <remote-url> plugins/poml-go-sdk`.
2) Remove the local replace once published and update go.mod to the module’s source.
3) Keep the CI workflow in the submodule to ensure SDK stays green independently.
