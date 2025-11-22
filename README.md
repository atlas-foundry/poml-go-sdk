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
- 🟡 Experimental: diagram tags (graph/node/edge/layer/camera) parsed/encoded and exportable to scene JSON (deterministic ordering toggle) for renderers (deck.gl/Graphviz planned).
- 🔴 Missing Python-level APIs: prompt builder/tag helpers, multimedia ingestion, tool/schema/runtime tags, format converters (dict/pydantic/OpenAI/LangChain), tracing backends, and fixtures parity.

Python parity target
- Upstream reference: `microsoft/poml` Python SDK @ commit `1e24b4262161f54b5ee30ac6de7ea7c0440f435f` (target_version in `ai/plans/python_parity.poml`).
- Tag coverage: meta/role/task/input/document/style/messages/tool-definition/request/response/output-schema/runtime/image implemented; tool-result/error variants not yet modeled.
- Converters: message_dict/dict/openai_chat/langchain implemented; pydantic/dataclass export pending.
- Builders/helpers: role/task/input/doc/style/message/tool-definition/request/response/schema/runtime/image helpers present; higher-level prompt builders still TODO.
- Multimedia: data-URI/image helpers and BaseDir-aware file ingestion present; broader multimedia (audio/video), MIME sniffing beyond extensions pending.
- Tracing: TODO (set_trace/trace_artifact equivalents not yet wired).
- CI/docs: GitHub Actions runs `gofmt` + `go test ./...`; coverage thresholds, parity badge, and expanded README examples are still pending.

Milestones to reach Python parity
1) **Tag coverage**: model additional tags used by Python SDK (message roles, tool-definition/request/response, output-schema, runtime, multimedia assets) and extend validation/AST to carry them.
2) **Format converters**: implement `poml(..., format=...)`-style exports for message_dict/dict/openai_chat/langchain/pydantic, including tool-calls and schema/runtime propagation.
3) **Builder/tag helpers**: add a Go prompt builder and tag helpers akin to Python `Prompt`/`_TagLib` so users can author POML programmatically with captions/styles.
4) **Multimedia support**: support file/buffer/base64 ingestion, MIME detection, and data URI/base64 emission to mirror image/audio handling in Python tests.
5) **Tracing**: implement tracing hooks (`set_trace`/`trace_artifact` equivalents) with env-based opt-in and optional backend hand-offs.
6) **Parity tests/fixtures**: port Python example fixtures (`examples/*.poml` + expects) and core tests (`test_basic.py`, `test_poml_formats.py`, `test_examples.py`) to Go, then gate CI on them with coverage thresholds.
7) **Docs/CI**: document the above APIs in README with usage snippets matching Python, and keep CI workflow green across the expanded surface.

Usage patterns
- Parsing: `doc, _ := poml.ParseFile("x.poml")` (or `ParseReader/ParseString`). Use `ParseFileStrict/ParseStringStrict/ParseReaderStrict` to enforce validation on read. Use `ParseFileFast/ParseStringFast/ParseReaderFast` to skip whitespace/comment capture for faster, lower-memory decode.
- Walking: `doc.Walk(func(el Element, p ElementPayload) error { ... })`.
- Mutations: `doc.Mutate(func(el Element, p ElementPayload, m *Mutator) error { m.ReplaceBody(el, "new"); m.Remove(el); m.InsertTaskAfter(el, "body"); return nil })`.
- Encoding: `doc.Encode(w)` or `doc.EncodeWithOptions(w, EncodeOptions{Indent: "  ", IncludeHeader: true, PreserveOrder: true, PreserveWS: true})`; `doc.DumpFile("path", opts)` atomically writes to disk.
- Validation: `if err := doc.Validate(); err != nil { ... }`. Parser helper opts include `ParseReaderWithOptions(r, ParseOptions{Validate: true})` to enforce validation during decode when desired.
- Converters: `Convert(doc, FormatOpenAIChat, ConvertOptions{BaseDir: "/assets", MaxImageBytes: 1<<20})` reads images relative to `BaseDir` with symlink-aware containment, rejects path escapes, defaults to a 10MB cap for file reads (override via `MaxImageBytes`; negative disables; data URIs unaffected), and requires `AllowAbsImagePaths` for absolute paths when no `BaseDir` is provided. Hints/examples/content parts (`<hint>`, `<example>`, `<cp>`), tool aliases (`<tool>`), and object tags round-trip and are emitted as user content across converters.
- Validation/fit-for-purpose: see `docs/fit_for_purpose.md` for goals/benchmarks/checks; CI should run strict parsing, converter golden comparisons, and security tests (path containment, size caps).

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
