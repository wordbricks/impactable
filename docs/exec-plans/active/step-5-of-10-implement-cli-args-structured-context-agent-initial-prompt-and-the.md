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
| M1 | Baseline and scaffolding alignment | not started | Verify existing `git-impact` package/CLI state from prior steps, confirm required types/interfaces (`AnalysisContext`, `Config`, `Source`, Velen client abstraction), and create missing files/directories if absent. |
| M2 | Context + prompt implementation | not started | `internal/gitimpact/context.go` provides `NewAnalysisContext(...)` (config load via Viper) and `BuildInitialPrompt(...)` that includes `since`/`pr`/`feature` values and configured Velen source keys. |
| M3 | CLI command surface wiring | not started | `cmd/git-impact/main.go` defines Cobra root with `--config` and `--output` (TTY-aware default), plus `analyze` (`--since`, `--pr`, `--feature`) and `check-sources` subcommands. |
| M4 | Source check implementation + output modes | not started | `internal/gitimpact/check_sources.go` adds `SourceCheckResult` and `CheckSources(...)` executing `velen auth whoami`, `velen org current`, `velen source list`, provider-type matching, and `SupportsQuery()` validation; `check-sources` command prints human text or JSON result. |
| M5 | Tests and verification | not started | Add `check_sources_test.go` with mock Velen client coverage for success/failure and source selection logic; `go build ./...` and `go test ./...` both pass. |

## Current progress
- Overall status: not started.
- Plan created and staged first to establish implementation scope and acceptance criteria.

## Key decisions
- Follow `SPEC.md` section 4.3 as the canonical behavior for source discovery and validation.
- Keep command outputs automation-safe: structured JSON for `--output json`, human summary for text mode.
- Model optional analysis selectors as explicit optional fields in context (`PR number`, `feature`) while preserving `since` and config provenance.
- Treat provider detection as case-insensitive substring matching on `provider_type` for GitHub and analytics family providers.

## Remaining issues
- Prior-step scaffolding may be missing in this worktree and could require creating `internal/gitimpact` and `cmd/git-impact` from scratch.
- Exact `Config` and Velen client interface shape must be reconciled with existing code before implementation to avoid incompatible assumptions.
- TTY-dependent default output behavior must be validated against the repository’s existing output-mode helpers (if any).

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
