# Step 6 of 10 - Implement the Source Check + Collect phase handlers for git-impact

## Goal
Implement the Source Check and Collect phase handlers in `internal/gitimpact/` so the engine can validate required Velen sources, collect GitHub PR/tag/release data, and advance phase state with test coverage.

## Background
- `SPEC.md` section 3.2 defines Source Check (pre-Turn 1) and Collector (Turn 1) responsibilities.
- `SPEC.md` sections 4.1-4.3 define Velen access flow, required source types (GitHub + Analytics), and wait-on-missing-source behavior.
- Existing `internal/gitimpact/` runtime already includes base types, engine loop, observer, Velen client, and `CheckSources`; this step must add phase handlers and registration without redefining existing types.

## Milestones
- [ ] Milestone 1 (not started): Confirm handler contracts and run-context expectations from existing `engine.go`, `check_sources.go`, and tests.
- [ ] Milestone 2 (not started): Add `phase_source_check.go` with `SourceCheckHandler` implementing `PhaseHandler`, including wait handling for missing/non-query-capable sources and wait-response resolution (`y` advance, `n` error).
- [ ] Milestone 3 (not started): Add `phase_collect.go` with `CollectHandler` implementing `PhaseHandler`, using Velen queries for PRs, tags, and releases; parse into `CollectedData`; persist to `runCtx`; return `DirectiveAdvancePhase`.
- [ ] Milestone 4 (not started): Add unit tests in `phase_source_check_test.go` and `phase_collect_test.go` with mockable Velen query/source behavior via interface or injectable functions.
- [ ] Milestone 5 (not started): Add `DefaultHandlers(client *VelenClient) map[Phase]PhaseHandler` registration for SourceCheck + Collect plus temporary Link/Score/Report advance stubs; run `go build ./...` and `go test ./...`.

## Current progress
- Plan created and checked in.
- Implementation milestones are not started.

## Key decisions
- Use additive files/functions only; do not alter or redefine existing foundational types.
- Prefer dependency injection (interface/function fields) in handlers to keep unit tests isolated from real `velen` command execution.
- Keep Link/Score/Report handlers as explicit stubs returning `DirectiveAdvancePhase` until their dedicated steps.

## Remaining issues
- Define exact wait-message wording for Source Check to be clear and deterministic for tests.
- Decide strictness for non-`y` wait responses (reject anything except explicit `y`, with explicit error on `n`).
- Align PR/tag/release SQL parsing with `QueryResult` column order and null-time handling.

## Links
- `SPEC.md` (sections 3.2, 4.1, 4.2, 4.3)
- `internal/gitimpact/engine.go`
- `internal/gitimpact/check_sources.go`
- `internal/gitimpact/velen.go`
- `docs/PLANS.md`
