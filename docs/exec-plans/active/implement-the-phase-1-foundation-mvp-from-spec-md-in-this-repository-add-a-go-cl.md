# Implement the Phase 1 Foundation MVP from SPEC.md in this repository (add a Go CLI)

## Goal
Deliver the Phase 1 Foundation MVP for the Git Impact Analyzer by adding a Go-first CLI path that supports config-driven execution, Velen-backed source checks, single-PR impact analysis, and report-generation scaffolding with test-backed contracts.

## Background
- `SPEC.md` defines the product direction as a Git Impact Analyzer and identifies Phase 1 as the MVP foundation.
- `ARCHITECTURE.md` currently centers the Go surface on `cmd/ralph-loop` and `internal/ralphloop`; this work extends that runtime toward the Git impact domain while respecting package boundaries.
- `NON_NEGOTIABLE_RULES.md` requires tests for new behavior and machine-readable automation output.
- This execution plan is scoped to foundation capabilities only: abstractions and end-to-end wiring for a single PR analysis path, not full multi-metric or feature-level expansion.

## Milestones
| ID | Milestone | Status | Exit criteria |
| --- | --- | --- | --- |
| M1 | CLI foundation and command surface | completed | Add Git impact CLI command surface and request/response contracts for `analyze`, `check-sources`, and report-scaffold output modes with machine-readable envelopes. |
| M2 | Config loading and validation | completed | Implement config file loading for Velen/source mappings and analysis window settings with validation + deterministic defaults and unit tests. |
| M3 | Velen integration abstractions and source checks | completed | Introduce Velen client abstractions (auth/org/source/query primitives), implement source discovery + required-source capability checks, and test with mock/fake executors. |
| M4 | Single-PR impact analysis path | completed | Implement collector/linker/scorer MVP flow for one PR: fetch PR metadata, perform before/after metric comparison for one metric, and compute single-PR impact score. |
| M5 | Report generation scaffolding | not started | Add report-domain scaffolding and CLI output adapters for terminal/JSON plus file scaffold hooks for Markdown/HTML expansion paths. |
| M6 | Test coverage, verification, and docs alignment | not started | Add coverage for new command paths and failure branches, run `go test ./...`, and update execution artifacts/docs needed for handoff. |

## Current progress
- Overall status: in progress.
- M1: completed.
- M2: completed.
- M3: completed.
- M4: completed.
- M5: not started.
- M6: not started.

## Milestone updates
### M1 - CLI foundation and command surface (completed)
- Added a new Go CLI entrypoint at `cmd/git-impact` plus root wrapper script `./git-impact`.
- Introduced `internal/gitimpact` with explicit command parsing and command descriptors for:
  - `analyze`
  - `check-sources`
  - `report-scaffold`
  - `schema` (surface introspection helper)
- Implemented machine-readable envelopes and structured error payloads for JSON/NDJSON flows.
- Added request/response contract scaffolds for:
  - analysis invocation contract (`analyze`)
  - source-check contract (`check-sources`)
  - report output-mode contract (`report-scaffold`, including terminal/json/markdown/html modes)
- Added test coverage for parser behavior, command option compatibility, schema contracts, envelope output, and structured failure output.
- Verification for this milestone: `go test ./...` passed.

### M2 - Config loading and validation (completed)
- Added `internal/gitimpact/config.go` with config loading for:
  - Velen org and source-role mapping (`github`, `warehouse`, `analytics`)
  - analysis window settings (`before_window_days`, `after_window_days`, `cooldown_hours`, `min_confidence`)
- Implemented deterministic defaults for analysis settings when omitted:
  - `before_window_days=7`
  - `after_window_days=7`
  - `cooldown_hours=24`
  - `min_confidence=0.6`
- Added config validation rules for required Velen fields and numeric bounds.
- Wired `analyze`, `check-sources`, and `report-scaffold` command paths to load and validate config before producing command envelopes.
- Added focused tests for config defaults, relative path resolution, and validation failures, and updated envelope/runtime tests to use real config fixtures.
- Verification for this milestone: `go test ./...` passed.

### M3 - Velen integration abstractions and source checks (completed)
- Added Velen abstraction layer in `internal/gitimpact/velen.go`:
  - `VelenClient` interface with auth/org/source/query primitives (`WhoAmI`, `CurrentOrg`, `ListSources`, `ShowSource`, `Query`)
  - CLI-backed implementation using a command-executor abstraction (`os/exec` in production)
  - JSON parsers tolerant to multiple response envelope shapes for source list/detail payloads
- Added source-check evaluator in `internal/gitimpact/source_checks.go`:
  - runs auth/org/source discovery flow
  - validates required role->source mappings against discovered sources
  - enforces `QUERY` capability checks per required source
  - emits structured per-role status (`ok`, `missing`, `failed`) plus readiness summary
- Wired `check-sources` runtime path to execute real Velen checks through the abstraction and emit structured metadata (`identity`, org match, discovered source count).
- Added test coverage with fake/mocked dependencies:
  - fake Velen client tests for role/capability/missing/org-mismatch branches
  - fake command executor tests for CLI integration primitive parsing
- updated command output contract test for `check-sources` to run via injected fake Velen client factory.
- Verification for this milestone: `go test ./...` passed.

### M4 - Single-PR impact analysis path (completed)
- Added collector/linker/scorer implementation for `analyze --pr` in `internal/gitimpact/analysis.go`:
  - collector: fetches PR metadata (`pr_number`, `title`, `author`, `merged_at`) via GitHub source query.
  - linker: resolves deployment timestamp from warehouse source, with fallback to `merged_at + cooldown_hours` when no deployment row is available.
  - scorer: computes before/after comparison for one metric (`conversion_rate`) over configured analysis windows.
- Added MVP impact score calculation using the single-metric model:
  - `impact_score = delta_ratio * weight * confidence * 10`
  - clamped to `[-10, 10]`.
- Wired `runAnalyze` to execute the single-PR path and emit structured stage + result payloads under `result.single_pr`.
- Enforced explicit Phase 1 scope by requiring `--pr` for `analyze` in this milestone.
- Added tests for:
  - full single-PR analyze flow and score computation via fake Velen query responses.
  - deployment-link fallback behavior.
  - invalid PR metadata failure handling.
  - structured command failure when `--pr` is missing.
- Verification for this milestone: `go test ./...` passed.

## Key decisions
- Treat `SPEC.md` Section 11 Phase 1 as the implementation boundary and defer Phase 2+ items.
- Keep Velen access behind testable interfaces so business logic is decoupled from subprocess execution details.
- Prefer deterministic, machine-readable command responses by default for automation-facing flows.
- Ship a narrow but complete vertical slice (single-PR path) before broadening metric/feature support.
- Preserve existing repository boundaries (`cmd/*` thin entrypoint, `internal/*` implementation logic).
- Added a dedicated Go runtime boundary (`cmd/git-impact` -> `internal/gitimpact`) instead of extending `internal/ralphloop`.
- Added `schema` command support early to make the command contract machine-discoverable for automation clients.
- Used a deterministic, repository-local YAML subset parser to avoid introducing external dependency risk in the foundation milestone.
- Keep Velen invocation behind a command executor abstraction to allow deterministic tests without shelling out to a real `velen` binary.
- For Phase 1 analysis, `analyze` is intentionally constrained to single-PR mode (`--pr`) rather than mixed multi-PR/since-window execution.

## Remaining issues
- Final package layout for new Git impact modules should be validated against existing `internal/ralphloop` ownership to avoid boundary drift.
- Query templates and source-role mapping heuristics may need follow-up refinement once real Velen source metadata is observed.
- Report scaffolding format and output location contracts must be stabilized before Phase 2 report expansion.

## Links
- Product spec: `SPEC.md`
- Architecture and boundaries: `ARCHITECTURE.md`
- Merge-blocking rules: `NON_NEGOTIABLE_RULES.md`
- Plan policy: `docs/PLANS.md`
- Plan location: `docs/exec-plans/active/implement-the-phase-1-foundation-mvp-from-spec-md-in-this-repository-add-a-go-cl.md`
