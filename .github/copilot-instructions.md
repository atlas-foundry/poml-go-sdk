# GitHub Copilot instructions (coding agent)

## Purpose
This is a Go 1.25 SDK for **POML** (Prompt Orchestration Markup Language) — a structured, spec-driven language for orchestrating AI prompts. The SDK provides parsing, validation, rendering, and format conversion capabilities with strong type safety and multi-modal support.

**Key philosophy:** This repository uses POML for all documentation. The only exceptions are `README.mdx` (for GitHub landing page) and this instructions file.

## Repo facts
- Go 1.25 SDK with parser, validator, renderer, and converters
- Multi-modal support for text, images, audio, video, and documents
- Safety-first design with BaseDir containment, symlink checks, and 10MB default media caps
- Docs live in POML (`*.poml`) with extended tags: `<hint>`, `<example>`, `<cp>`, `<object>`, `<list>`, `<tool>`

## High-signal references
- `README.poml` for structured overview and usage patterns.
- `AGENTS.poml` for doc policy and quick rules.
- `docs/fit_for_purpose.poml` for coverage/validation goals.
- `poml/testdata/examples/*` for fixtures and golden outputs.

## Coding standards
- Follow Go best practices documented in `go.quality.poml`:
  - No unchecked errors (always handle or wrap errors)
  - Defensive bounds checks for slices/arrays
  - Avoid data races (use proper locking)
  - Small, focused functions with single responsibility
  - Use `crypto/rand` for sensitive values, not `math/rand`
- Use `gofmt` for formatting (enforced by CI)
- Write deterministic tests; add fuzzing for parsers

## Working rules
- Keep new docs in POML; do not add other Markdown/org/mdx files (besides `README.mdx` and this instructions file). Use extended tags like `<hint>`, `<example>`, `<cp>`, `<object>`, `<list>`.
- Link any high-signal new docs from `README.mdx` for discoverability.
- Preserve safety defaults in converters: BaseDir containment, symlink-aware path checks, and 10MB default caps for image/audio/video (override via options only when intentional).
- Maintain round-trip fidelity for tags and fixtures; update goldens in `poml/testdata/examples` only when behavior changes intentionally.

## Git workflow
- Use Conventional Commits format for all commits:
  - `feat:` for new features ✨
  - `fix:` for bug fixes 🐛
  - `docs:` for documentation 📘
  - `test:` for tests 🧪
  - `refactor:` for refactoring 🧠
  - `chore:` for maintenance 🧹
  - `ci:` for CI/CD changes ⚙️
  - Add `!` for breaking changes (e.g., `feat!:`)
- All work on feature branches; PRs require code review
- CI enforces: linting, tests, coverage thresholds (~82.6% baseline)

## Testing/build
- Format Go changes with `gofmt`.
- Run `go test ./...` (CI enforces coverage thresholds; baseline ~82.6% per `docs/fit_for_purpose.poml`). Use `/usr/local/go/bin/go test` if PATH issues arise.
- Add/adjust tests alongside behavior changes, especially around converters and multimedia handling.
- Tests must be deterministic; use fuzzing for parser robustness.

## Boundaries and restrictions
**What NOT to do:**
- Never modify or remove working tests unless fixing a bug in those specific tests
- Never add non-POML documentation files (except `README.mdx` and this file)
- Never commit secrets, API keys, or sensitive data
- Never bypass safety defaults (BaseDir checks, size caps) without explicit justification
- Never modify golden test outputs (`poml/testdata/examples/*.expected.*`) without verifying the behavior change is intentional
- Ignore unrelated broken tests or build failures; only fix issues related to your changes

## Output expectations
- Provide concise, PR-ready edits with brief reasoning and test results in the summary.
