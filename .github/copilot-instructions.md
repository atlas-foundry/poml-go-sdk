# GitHub Copilot instructions (coding agent)

## Repo facts
- Go 1.25 SDK for POML (Prompt Orchestration Markup Language) with parser, validator, renderer, and converters.
- Docs live in POML (`*.poml`) with extended tags; `README.mdx` is the only Markdown landing page (this file is an instruction exception).

## High-signal references
- `README.poml` for structured overview and usage patterns.
- `AGENTS.poml` for doc policy and quick rules.
- `docs/fit_for_purpose.poml` for coverage/validation goals.
- `poml/testdata/examples/*` for fixtures and golden outputs.

## Working rules
- Keep new docs in POML; do not add other Markdown/org/mdx files (besides `README.mdx` and this instructions file). Use extended tags like `<hint>`, `<example>`, `<cp>`, `<object>`, `<list>`.
- Link any high-signal new docs from `README.mdx` for discoverability.
- Preserve safety defaults in converters: BaseDir containment, symlink-aware path checks, and 10MB default caps for image/audio/video (override via options only when intentional).
- Maintain round-trip fidelity for tags and fixtures; update goldens in `poml/testdata/examples` only when behavior changes intentionally.

## Testing/build
- Format Go changes with `gofmt`.
- Run `go test ./...` (CI enforces coverage thresholds; baseline ~82.6% per `docs/fit_for_purpose.poml`). Use `/usr/local/go/bin/go test` if PATH issues arise.
- Add/adjust tests alongside behavior changes, especially around converters and multimedia handling.

## Output expectations
- Provide concise, PR-ready edits with brief reasoning and test results in the summary.
