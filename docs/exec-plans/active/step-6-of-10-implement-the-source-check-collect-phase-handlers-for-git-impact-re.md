# Step 6 of 10 - Implement the Source Check + Collect phase handlers for git-impact

## Goal
Implement the Source Check and Collect phase handlers in `internal/gitimpact/` so the engine can validate required OneQuery sources, collect GitHub PR/tag/release data, and advance phase state with test coverage.

## Background
- `SPEC.md` section 3.2 defines Source Check (pre-Turn 1) and Collector (Turn 1) responsibilities.
- `SPEC.md` sections 4.1-4.3 define OneQuery access flow, required source types (GitHub + Analytics), and wait-on-missing-source behavior.
- Existing `internal/gitimpact/` runtime already includes base types, engine loop, observer, OneQuery client, and `CheckSources`; this step must add phase handlers and registration without redefining existing types.

## Milestones
- [x] Milestone 1 (completed): Confirm handler contracts and run-context expectations from existing `engine.go`, `check_sources.go`, and tests.
- [x] Milestone 2 (completed): Add `phase_source_check.go` with `SourceCheckHandler` implementing `PhaseHandler`, including wait handling for missing/non-query-capable sources and wait-response resolution (`y` advance, `n` error).
- [x] Milestone 3 (completed): Add `phase_collect.go` with `CollectHandler` implementing `PhaseHandler`, using OneQuery queries for PRs, tags, and releases; parse into `CollectedData`; persist to `runCtx`; return `DirectiveAdvancePhase`.
- [x] Milestone 4 (completed): Add unit tests in `phase_source_check_test.go` and `phase_collect_test.go` with mockable OneQuery query/source behavior via interface or injectable functions.
- [ ] Milestone 5 (obsolete): This step originally planned a local `DefaultHandlers` registration path. The runtime now executes `git-impact analyze` through the Codex app-server agent engine, and the temporary default-engine path has been removed.

## Current progress
- Plan created and checked in.
- Milestone 1 completed by confirming the `PhaseHandler` contract (`Handle(context.Context, *RunContext)`) and engine wait-cycle behavior (`runCtx.AnalysisCtx.LastWaitResponse` populated after `DirectiveWait`).
- Confirmed `CheckSources(ctx, client, cfg)` already encapsulates auth/org/source discovery and returns the exact readiness bits needed by `SourceCheckHandler` (`GitHubOK`, `AnalyticsOK`, and `Errors`).
- Confirmed `RunContext.CollectedData` shape (`PRs []PR`, `Tags []string`, `Releases []Release`) matches Step 6 collector output targets without adding or redefining core types.
- Milestone 2 completed by adding `internal/gitimpact/phase_source_check.go` with `SourceCheckHandler` that:
  - runs `CheckSources` through injectable function field (defaults to `CheckSources`);
  - advances immediately when GitHub + Analytics are both QUERY-capable;
  - issues `DirectiveWait` with deterministic message when not ready;
  - resolves wait input from `LastWaitResponse` (`y` => advance, `n` => hard error, other => validation error).
- Milestone 3 completed by adding `internal/gitimpact/phase_collect.go` with `CollectHandler` that:
  - issues the required PR/tag/release SQL queries via injectable query function (defaults to `OneQueryClient.Query`);
  - derives `{since}` from `runCtx.AnalysisCtx.Since` (fallback `1970-01-01`);
  - parses PR/tag/release rows into existing `PR`, `Release`, and `CollectedData` types;
  - stores parsed output in `runCtx.CollectedData` and returns `DirectiveAdvancePhase`.
- Milestone 4 completed with handler-focused unit tests:
  - `phase_source_check_test.go`: verifies ready->advance, not-ready->wait, wait-response handling (`y` advance, `n` error), and checker-error propagation.
  - `phase_collect_test.go`: verifies required SQL execution/order, parsed data population in `runCtx.CollectedData`, missing-source-key failure, query-error wrapping, and invalid-row parse failure.
  - `go test ./internal/gitimpact` passes with the new tests.

## Key decisions
- Use additive files/functions only; do not alter or redefine existing foundational types.
- Prefer dependency injection (interface/function fields) in handlers to keep unit tests isolated from real `onequery` command execution.
- Keep Link/Score/Report handlers as explicit stubs returning `DirectiveAdvancePhase` until their dedicated steps.
- Treat wait responses as phase-local state read from `runCtx.AnalysisCtx.LastWaitResponse` so source-check confirmation can resolve within the existing engine retry loop.
- Use config-driven GitHub source key (`cfg.OneQuery.Sources.GitHub`) for collection queries, with SQL assembled deterministically for test assertions.
- Use strict wait-response semantics for source check (`y`/`n` only) to avoid silent continuation on ambiguous input.
- Keep collector parsing defensive for mixed JSON row scalar types (`string`, `float64`, `[]interface{}`, `json.Number`) to avoid coupling to one OneQuery JSON decoder shape.

## Remaining issues
- Historical note: the temporary local default-engine registration path planned for this step was later dropped after the Codex app-server agent engine became the only `git-impact analyze` runtime.

## Links
- `SPEC.md` (sections 3.2, 4.1, 4.2, 4.3)
- `internal/gitimpact/engine.go`
- `internal/gitimpact/check_sources.go`
- `internal/gitimpact/onequery.go`
- `docs/PLANS.md`
