# Step 8 of 10: Implement the Impact Scorer phase handler (Turn 3) for git-impact

## Goal
Implement Turn 3 scoring behavior in `internal/gitimpact` so deployments are evaluated against analytics metrics, PR impacts are scored with confidence reasoning, contributor rollups are computed, and phase control advances correctly.

## Status
Superseded. `git-impact analyze` now executes the score phase through the Codex app-server agent runtime. The earlier local `ScoreHandler` plan and its `phase_score.go` implementation were removed once they stopped serving any runtime path.

## Background
- `SPEC.md` section 3.2 defines Impact Scorer responsibilities: schema exploration, metric querying, confidence judgment, and natural-language reasoning.
- `SPEC.md` section 5.1 defines PR-level score semantics as agent judgment over meaningful metrics with confidence adjustment.
- `SPEC.md` sections 5.3 and 5.4 define default before/after windows and handling of confounding overlapping deployments.
- `internal/gitimpact` already contains `RunContext`, `LinkedData`, `ScoredData`, and phase directive types required for Turn 3 integration.

## Milestones
| ID | Milestone | Status | Exit criteria |
| --- | --- | --- | --- |
| M1 | Add Impact Scorer handler scaffold | obsolete | The planned local `ScoreHandler` path was removed after the Codex app-server phase-agent path became the only analyze runtime. |
| M2 | Implement schema discovery + metric query flow | obsolete | This behavior is now delegated to the agent score turn. |
| M3 | Implement score/confidence/reasoning generation | obsolete | This behavior is now delegated to the agent score turn. |
| M4 | Implement contributor rollup | obsolete | This behavior is now delegated to the agent score turn. |
| M5 | Add scorer tests | obsolete | The removed `phase_score.go` / `phase_score_test.go` path is no longer part of the runtime. |
| M6 | Verify and finalize | obsolete | No local scorer path remains to verify. |

## Current progress
- Historical plan retained for traceability only.
- The local scorer path referenced in this plan no longer exists in the codebase.

## Key decisions
- Current runtime design delegates score-phase metric selection and scoring reasoning to the Codex app-server agent turn.

## Remaining issues
- No local scorer implementation remains under this plan.

## Links
- `SPEC.md` (sections 3.2, 5.1, 5.3, 5.4)
- `internal/gitimpact/engine.go`
- `internal/gitimpact/types.go`
- `internal/gitimpact/phase_collect.go`
- `internal/gitimpact/phase_link.go`
- `internal/gitimpact/agent_runtime.go`
- `docs/PLANS.md`
- `NON_NEGOTIABLE_RULES.md`
