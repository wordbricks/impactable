# Step 5 of 10 - Implement CLI args, structured context, initial prompt, and check-sources

## Goal
Implement Step 5 for `git-impact`: wire CLI args into a structured analysis context, generate the agent initial prompt from context + config, add a `check-sources` command with structured results, and cover behavior with tests.

## Background
- Product requirements come from `SPEC.md` section 4.3 (source discovery strategy) and section 7 (CLI interface + structured context passing).
- Merge-blocking repository rules require tests for all new behavior and machine-readable output for automation-facing commands.
- Step scope includes three code surfaces: `internal/gitimpact/context.go`, `internal/gitimpact/check_sources.go`, and `cmd/git-impact/main.go`, plus unit tests.
- Current worktree does not yet contain `internal/gitimpact/` or `cmd/git-impact/`; Step 5 will proceed by implementing or extending those paths per prior step expectations.

## Milestones
| ID | Milestone | Status | Exit criteria |
| --- | --- | --- | --- |
| M1 | Baseline and scaffolding alignment | completed | Verified prior-step `git-impact` scaffolding was absent; created `internal/gitimpact` and `cmd/git-impact` with core shared types (`AnalysisContext`, `Config`, `Source`, `VelenClient`) and CLI/package layout. |
| M2 | Context + prompt implementation | completed | Implemented `NewAnalysisContext(...)` with Viper-backed config loading from `--config`, optional PR/feature population, and `BuildInitialPrompt(...)` containing `since`/`pr`/`feature` plus configured Velen source keys. |
| M3 | CLI command surface wiring | completed | Added Cobra root `git-impact` with persistent `--config` and `--output` (TTY-aware default), `analyze` flags (`--since`, `--pr`, `--feature`), and `check-sources` subcommand wiring. |
| M4 | Source check implementation + output modes | completed | Implemented `CheckSources(...)` and `SourceCheckResult`; command flow runs auth/org/source discovery via Velen client abstraction, matches GitHub/analytics providers by `provider_type`, validates QUERY support, and renders text or JSON output. |
| M5 | Tests and verification | completed | Added mock-client tests in `check_sources_test.go` (success/failure/source capability and call ordering), added context/prompt coverage, and validated `go build ./...` + `go test ./...` passing. |

## Current progress
- Overall status: completed.
- Implemented all Step 5 surfaces:
- `internal/gitimpact/types.go`, `context.go`, `velen.go`, `check_sources.go`, `context_test.go`, `check_sources_test.go`
- `cmd/git-impact/main.go`
- Added module dependencies for Cobra and Viper in `go.mod`/`go.sum`.
- Verification completed successfully:
- `go build ./...`
- `go test ./...`

## Key decisions
- Follow `SPEC.md` section 4.3 as the canonical behavior for source discovery and validation.
- Keep command outputs automation-safe: structured JSON for `--output json`, human summary for text mode.
- Model optional analysis selectors as explicit optional fields in context (`PR number`, `feature`) while preserving `since` and config provenance.
- Treat provider detection as case-insensitive substring matching on `provider_type` for GitHub and analytics family providers.
- Use a `VelenClient` interface with a concrete CLI-backed implementation (`os/exec`) to keep runtime behavior aligned with spec while keeping unit tests deterministic.
- Return a fully populated `SourceCheckResult` on failure paths (with `Error`) and propagate an error for non-zero command behavior.

## Remaining issues
- None in Step 5 scope.

## Links
- Spec (source discovery): `SPEC.md` section 4.3
- Spec (CLI + context): `SPEC.md` section 7
- Merge rules: `NON_NEGOTIABLE_RULES.md`
- Plan policy: `docs/PLANS.md`
- Target files:
  - `internal/gitimpact/context.go`
  - `internal/gitimpact/check_sources.go`
  - `internal/gitimpact/check_sources_test.go`
  - `cmd/git-impact/main.go`
