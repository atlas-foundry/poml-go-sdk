# poml-go-sdk (Atlas plugin)

Goal: Go SDK for POML with API/behavior parity to the Microsoft Python SDK (https://microsoft.github.io/poml/latest/python/).

Current scope
- Parser/encoder for POML (strict XML, CDATA-safe, preserves meta/role/task/input/document/style blocks; preserves unknown elements, attributes, and optional leading/trailing whitespace/comments).
- AST helpers: ordered `Elements` with stable IDs/parent hooks, `Walk`/`Mutate`, builders (`Add*`), validation (required meta/role/task, single meta/role), and configurable encoding options (order/whitespace/header/indent/compact).
- Parsers for string/file/reader plus role/task/input accessors.
- Tests: golden sample, round-trip encode (including unknown), malformed error coverage, large document parsing, builder/mutation coverage.

Parity matrix vs. Python SDK (working list)
- ✅ Preserve unknown elements/attrs, CDATA, and element order; support whitespace/comments round-trip when requested.
- ✅ Validation for required meta/role/task plus unique inputs/doc/style fields; structured error type.
- ✅ Walk/Mutate with stable element IDs and ID lookup.
- ✅ Encode options for header/indent/order/compact/whitespace.
- 🟡 Parse options (whitespace preservation toggle) and parent hooks; more fixtures from Python SDK samples still to port.
- 🟡 CI workflow/coverage thresholds and README examples mirroring Python usage (in progress).

Usage patterns
- Parsing: `doc, _ := poml.ParseFile("x.poml")` (or `ParseReader/ParseString`).
- Walking: `doc.Walk(func(el Element, p ElementPayload) error { ... })`.
- Mutations: `doc.Mutate(func(el Element, p ElementPayload, m *Mutator) error { m.ReplaceBody(el, "new"); m.Remove(el); m.InsertTaskAfter(el, "body"); return nil })`.
- Encoding: `doc.Encode(w)` or `doc.EncodeWithOptions(w, EncodeOptions{Indent: "  ", IncludeHeader: true, PreserveOrder: true, PreserveWS: true})`; `doc.DumpFile("path", opts)` atomically writes to disk.
- Validation: `if err := doc.Validate(); err != nil { ... }`.

Next steps
1) Add parent IDs/comments/whitespace preservation hooks for finer-grained round-trips.
2) Expand fixtures to mirror Python SDK examples and edge cases (bad attributes, missing tags, nested CDATA).
3) Add CI workflow and coverage thresholds; publish usage docs for atlas imports.

Usage in atlas
- Local replace is set in the root go.mod: `replace plugins/poml-go-sdk => ./plugins/poml-go-sdk`.
- After adding as a real submodule/repo, drop the replace and pin a version: `go get plugins/poml-go-sdk@vX.Y.Z`.

If you intend to commit this as a submodule:
1) From repo root: `git submodule add <remote-url> plugins/poml-go-sdk`.
2) Remove the local replace once published and update go.mod to the module’s source.
3) Keep the CI workflow in the submodule to ensure SDK stays green independently.
