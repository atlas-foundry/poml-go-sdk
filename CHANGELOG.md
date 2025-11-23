# [1.4.0](https://github.com/atlas-foundry/poml-go-sdk/compare/v1.3.0...v1.4.0) (2025-11-23)


### Features

* add ValidateWithOptions with extended mode handling ([95704ba](https://github.com/atlas-foundry/poml-go-sdk/commit/95704bae4b84787b8ffeeffd7d33df7e1f4034ad))

# [1.3.0](https://github.com/atlas-foundry/poml-go-sdk/compare/v1.2.1...v1.3.0) (2025-11-23)


### Features

* add ExtendedMode toggles to parse/convert options ([a1045af](https://github.com/atlas-foundry/poml-go-sdk/commit/a1045af530a515dff69dad10b506e043a9e11631))

## [1.2.1](https://github.com/atlas-foundry/poml-go-sdk/compare/v1.2.0...v1.2.1) (2025-11-23)


### Bug Fixes

* avoid concurrent ws writes and add request metrics/auth ([9e264bc](https://github.com/atlas-foundry/poml-go-sdk/commit/9e264bc43350e57c008d0a6e398ba72d2dfe7072))

# [1.2.0](https://github.com/atlas-foundry/poml-go-sdk/compare/v1.1.0...v1.2.0) (2025-11-23)


### Bug Fixes

* resolve ineffassign lint warnings in tracing ([0ead083](https://github.com/atlas-foundry/poml-go-sdk/commit/0ead083dee74702083c65964bb3bd87ffeb20d4e))


### Features

* add diff/patch endpoints to poml mcp ([8d7c512](https://github.com/atlas-foundry/poml-go-sdk/commit/8d7c512c06ca2016895ec3869d505c5d083dccfe))
* add metrics endpoint and optional auth for all MCP routes ([3405258](https://github.com/atlas-foundry/poml-go-sdk/commit/3405258d220a96268e517eb2e3d3b43b24c6ba6d))
* add OTEL stdout tracing toggle and diff/patch docs ([98a976a](https://github.com/atlas-foundry/poml-go-sdk/commit/98a976a3ed407be37b6ad1d56de9993640209a3c))
* add OTLP tracing options and update docs ([68a86cb](https://github.com/atlas-foundry/poml-go-sdk/commit/68a86cb1850299fd5d3228024f55603dc338ef52))
* add pomp CLI with poml mcp AST server ([f6ae2de](https://github.com/atlas-foundry/poml-go-sdk/commit/f6ae2de373d6529d7139f6c1c00f9829af9547b5))
* add tools/diagram/roundtrip endpoints to poml mcp ([cff3722](https://github.com/atlas-foundry/poml-go-sdk/commit/cff3722e505a6ec0e564b2499ca0af881de18fcf))
* add watch/SSE support and tracing toggle to poml mcp ([6805c6f](https://github.com/atlas-foundry/poml-go-sdk/commit/6805c6f9029109250f9263f6a7deaa02023e54ba))
* add WebSocket watch endpoint and tracing toggle ([12e7cd4](https://github.com/atlas-foundry/poml-go-sdk/commit/12e7cd4f5eed802d2878cddaa630767d82ed1826))
* enhance ws watch with auth, heartbeat, and tests ([d970ffa](https://github.com/atlas-foundry/poml-go-sdk/commit/d970ffa7e4b391f01fd6ac34624bea51bfddc652))
* extend poml mcp with validate/convert/search endpoints ([3143c6b](https://github.com/atlas-foundry/poml-go-sdk/commit/3143c6ba372fe4b08eaf2fedb193e309b262b2d9))

## [1.1.0](https://github.com/atlas-foundry/poml-go-sdk/compare/v1.0.0...v1.1.0) (2025-11-23)

### Features

* add OpenTelemetry tracing hooks and span tests for parse/validate/convert (opt-in, no behavioral change by default)
* add VHS demos and documentation plan/roadmap to guide onboarding
* refresh badges and documentation strategy; add roadmap and spec coverage notes

### CI/CD

* align Linux/macOS/Windows/vet/release jobs on Go 1.25.4 with go mod tidy + coverage on all
* drop Codecov upload; keep coverage as CI artifact

### Documentation

* add docs plan (POML), VHS tapes/GIFs, and clarify semver expectations

### Build/Tooling

* add missing docs render script for MDX

# 1.0.0 (2025-11-23)


### Bug Fixes

* formatting in renderer_test.go ([715dbad](https://github.com/atlas-foundry/poml-go-sdk/commit/715dbad1e10e358e3e7df67c669820452c8ced7f))
* make path tests tolerant of /private prefix ([8531afa](https://github.com/atlas-foundry/poml-go-sdk/commit/8531afa858a281f73c4d725cea05511477f51662))
* normalize line endings in renderer tests for windows compatibility ([9401a6a](https://github.com/atlas-foundry/poml-go-sdk/commit/9401a6a8f8b5a9aed7534d4f3d2de468c21bb8ce))
* satisfy lint checks ([fff8906](https://github.com/atlas-foundry/poml-go-sdk/commit/fff8906130add83f71bf6fd0745ccc79d0f175f7))
* use marked for docs generation instead of missing mdx cli ([ab3276e](https://github.com/atlas-foundry/poml-go-sdk/commit/ab3276ee38c5a5bf235057aafdea094097777d11))
