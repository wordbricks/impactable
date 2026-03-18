# Implement the Phase 1 Foundation MVP from SPEC.md in this repository (add a Go CLI)

## Goal
Implement the Phase 1 Foundation MVP from `SPEC.md` by delivering a Go-based `git-impact` CLI vertical slice with test coverage, config loading, Velen integration abstractions, source checks, a single-PR impact analysis path, and report generation scaffolding.

## Background
- `SPEC.md` defines Phase 1 scope as the foundation for Velen-backed data access, single-PR analysis, and CLI output.
- `ARCHITECTURE.md` defines package boundaries: thin `cmd/*` entrypoints and behavior in `internal/*` packages.
- `NON_NEGOTIABLE_RULES.md` requires tests for new behavior and structured automation output.
- `docs/PLANS.md` requires this plan format and active status tracking.

## Milestones
| ID | Milestone | Status | Exit criteria |
| --- | --- | --- | --- |
| M1 | CLI and package foundation | not started | Add/confirm `cmd/git-impact` entrypoint and `internal/gitimpact` command scaffolding with structured command/result envelopes. |
| M2 | Config loading and validation | not started | Implement config file loading and validation for Velen org/source mappings and analysis windows, with deterministic defaults and tests. |
| M3 | Velen integration abstractions and source checks | not started | Implement Velen client abstractions (`auth`, `org`, `source`, `query`) and `check-sources` flow with capability validation and tests. |
| M4 | Single-PR impact analysis path | not started | Implement `analyze --pr` MVP flow (collect/link/score) for one metric with structured output and tests. |
| M5 | Report scaffolding and output surfaces | not started | Implement report-generation scaffolding and output mode plumbing for terminal/JSON plus file-oriented markdown/html hooks. |
| M6 | Verification and plan/doc updates | not started | Run Go tests for new behaviors, close coverage gaps in failure branches, and update execution artifacts for handoff. |

## Current progress
- Overall status: not started.
- M1: not started.
- M2: not started.
- M3: not started.
- M4: not started.
- M5: not started.
- M6: not started.

## Key decisions
- Keep implementation aligned to Phase 1 only; defer multi-metric, feature grouping, and leaderboard work to later phases.
- Preserve architecture boundaries (`cmd/git-impact` thin; `internal/gitimpact` behavior).
- Treat machine-readable structured output as the primary contract for automation-facing commands.
- Isolate Velen subprocess usage behind abstractions to keep tests deterministic.

## Remaining issues
- Confirm exact config schema shape and compatibility expectations for existing repository fixtures.
- Decide final report scaffold file layout and naming for markdown/html outputs.
- Validate assumptions about Velen source capability metadata across providers.

## Links
- Product spec: `SPEC.md`
- Architecture: `ARCHITECTURE.md`
- Non-negotiable rules: `NON_NEGOTIABLE_RULES.md`
- Plan policy: `docs/PLANS.md`
- Design doc index: `docs/design-docs/index.md`
- Product spec index: `docs/product-specs/index.md`
