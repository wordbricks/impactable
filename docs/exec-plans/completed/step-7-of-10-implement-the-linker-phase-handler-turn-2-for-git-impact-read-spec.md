# Step 7 of 10 - Implement the Linker phase handler (Turn 2) for git-impact

## Goal
Implement the Linker phase handler in `internal/gitimpact/` so Turn 2 infers deployments from collected GitHub data, proposes feature groupings, handles ambiguity through wait/resume, and persists outputs to `runCtx.LinkedData`.

## Background
- `SPEC.md` section 3.2 defines Linker responsibilities for Turn 2.
- `SPEC.md` section 5.2 defines deployment inference priority: releases, then version tags, then PR merge fallback.
- `SPEC.md` section 6 defines feature grouping strategies: label-based, branch-prefix, and time+author clustering.
- Existing `internal/gitimpact` engine/types/previous handlers already provide `PhaseHandler`, directives, run context, and wait-response loop; this step must extend behavior without redefining base types.

## Milestones
- [x] Milestone 1 (completed): Confirmed existing `internal/gitimpact` contracts for `PhaseHandler`, `RunContext.CollectedData`, `RunContext.LinkedData`, and wait-response flow in the engine.
- [x] Milestone 2 (completed): Added `internal/gitimpact/phase_link.go` with `LinkHandler` implementing `Handle(ctx, runCtx)` and baseline validation/flow control.
- [x] Milestone 3 (completed): Implemented deployment inference priority (GitHub releases -> version tags -> PR merge fallback), including ambiguous deployment detection.
- [x] Milestone 4 (completed): Implemented wait directive behavior for ambiguous deployment resolution, including parsing `y`/`n`/`skip` responses.
- [x] Milestone 5 (completed): Implemented feature grouping proposals for label prefix, branch prefix, and same-author time clustering.
- [x] Milestone 6 (completed): Added `internal/gitimpact/phase_link_test.go` covering release inference, tag inference, fallback behavior, feature grouping by label, and ambiguous wait directive.
- [x] Milestone 7 (completed): Ran `go build ./...` and `go test ./...`; repository is green.

## Current progress
- Linked phase implementation is in place in `internal/gitimpact/phase_link.go`.
- Linker tests are in place in `internal/gitimpact/phase_link_test.go`.
- Validation completed:
  - `GOCACHE=/tmp/go-build-cache go build ./...`
  - `go test ./...`

## Key decisions
- Preserve all existing base types and engine contracts; implement only additive Linker logic and tests.
- Keep ambiguity handling deterministic and explicit so engine wait/resume behavior remains testable.
- Favor deterministic grouping/inference rules that are easy to assert in unit tests.
- Parse wait responses as strict `y` / `n` / `skip` values after ambiguity is raised, with explicit abort on `n`.

## Remaining issues
- None for this step.

## Links
- `SPEC.md` (sections 3.2, 5.2, 6)
- `internal/gitimpact/engine.go`
- `internal/gitimpact/phase_collect.go`
- `internal/gitimpact/phase_source_check.go`
- `docs/PLANS.md`
