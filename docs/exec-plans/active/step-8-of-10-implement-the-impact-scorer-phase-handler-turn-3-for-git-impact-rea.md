# Step 8 of 10: Implement the Impact Scorer phase handler (Turn 3) for git-impact

## Goal
Implement Turn 3 scoring behavior in `internal/gitimpact` so deployments are evaluated against analytics metrics, PR impacts are scored with confidence reasoning, contributor rollups are computed, and phase control advances correctly.

## Background
- `SPEC.md` section 3.2 defines Impact Scorer responsibilities: schema exploration, metric querying, confidence judgment, and natural-language reasoning.
- `SPEC.md` section 5.1 defines PR-level score semantics as agent judgment over meaningful metrics with confidence adjustment.
- `SPEC.md` sections 5.3 and 5.4 define default before/after windows and handling of confounding overlapping deployments.
- `internal/gitimpact` already contains `RunContext`, `LinkedData`, `ScoredData`, and phase directive types required for Turn 3 integration.

## Milestones
| ID | Milestone | Status | Exit criteria |
| --- | --- | --- | --- |
| M1 | Add Impact Scorer handler scaffold | not started | `internal/gitimpact/phase_score.go` defines `ScoreHandler` with `Handle(context.Context, *RunContext)` and required helper functions. |
| M2 | Implement schema discovery + metric query flow | not started | Handler queries analytics `information_schema.columns`, selects first usable metric, and runs before/after metric queries per deployment. |
| M3 | Implement score/confidence/reasoning generation | not started | Each deployment yields `PRImpact` with score (0-10), confidence (`high`/`medium`/`low`), and reasoning including confounding context. |
| M4 | Implement contributor rollup | not started | PR impacts are grouped by author to compute average score and top PR in `ContributorStats`. |
| M5 | Add scorer tests | completed | `internal/gitimpact/phase_score_test.go` covers score normalization, confidence thresholds, contributor rollup, and empty-schema graceful behavior. |
| M6 | Verify and finalize | completed | `go build ./...` and `go test ./...` pass; scorer changes are staged and committed. |

## Current progress
- Repository guardrails and architecture docs reviewed.
- Required spec sections identified and read (`3.2`, `5.1`, `5.3`, `5.4`).
- Added scorer test coverage for success-path metric querying and contributor rollup, in addition to existing score/confidence and empty-schema degradation coverage.
- Hardened scoring behavior for deployments missing `deployed_at` by assigning a neutral score with explicit low-confidence reasoning.
- Verified the full repository with `go build ./...` and `go test ./...` (both passing on 2026-03-22).

## Key decisions
- Reuse existing `VelenClient.Query` integration pattern from prior phase handlers for deterministic testability.
- Keep default windows aligned with config defaults and spec defaults (7 days each when unset).
- Treat deployment overlap density as the primary confidence baseline and surface overlap details in reasoning text.
- For missing deployment timestamps, degrade gracefully instead of producing synthetic date-window queries.

## Remaining issues
- Analytics schemas may vary widely; first-metric discovery and date filtering logic must be robust to sparse metadata.
- Some deployments may not map to PR authors in collected data; rollup behavior must handle missing author values safely.
- Milestones M1-M4 are still marked pending in this plan artifact and require status reconciliation with implemented scorer behavior.

## Links
- `SPEC.md` (sections 3.2, 5.1, 5.3, 5.4)
- `internal/gitimpact/engine.go`
- `internal/gitimpact/types.go`
- `internal/gitimpact/phase_collect.go`
- `internal/gitimpact/phase_link.go`
- `docs/PLANS.md`
- `NON_NEGOTIABLE_RULES.md`
