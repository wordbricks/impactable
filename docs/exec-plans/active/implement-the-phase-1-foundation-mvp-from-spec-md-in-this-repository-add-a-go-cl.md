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
| M1 | CLI and package foundation | completed | Add/confirm `cmd/git-impact` entrypoint and `internal/gitimpact` command scaffolding with structured command/result envelopes. |
| M2 | Config loading and validation | completed | Implement config file loading and validation for Velen org/source mappings and analysis windows, with deterministic defaults and tests. |
| M3 | Velen integration abstractions and source checks | completed | Implement Velen client abstractions (`auth`, `org`, `source`, `query`) and `check-sources` flow with capability validation and tests. |
| M4 | Single-PR impact analysis path | completed | Implement `analyze --pr` MVP flow (collect/link/score) for one metric with structured output and tests. |
| M5 | Report scaffolding and output surfaces | not started | Implement report-generation scaffolding and output mode plumbing for terminal/JSON plus file-oriented markdown/html hooks. |
| M6 | Verification and plan/doc updates | not started | Run Go tests for new behaviors, close coverage gaps in failure branches, and update execution artifacts for handoff. |

## Current progress
- Overall status: in progress.
- M1: completed. `cmd/git-impact` thin entrypoint is in place, and success outputs now use one envelope shape (`command`, `status`, optional `request`/`config`, and `result`) across `analyze`, `check-sources`, `report-scaffold`, and `schema`.
- M2: completed. Config loading supports cwd-relative/default path resolution, validates required `velen` org/source role mappings and analysis windows, applies deterministic defaults for omitted analysis fields, and now rejects non-finite confidence values (`NaN`/`Inf`) in addition to out-of-range values. Coverage added for default-path loading, partial analysis defaults, and invalid numeric parsing.
- M3: completed. Velen CLI integration abstractions for `auth whoami`, `org current`, `source list/show`, and `query` are in place; `check-sources` now verifies each required mapped source via `source show` and enforces `QUERY` capability from source detail, with tests covering positive, missing, unsupported, and detail-lookup failure paths.
- M4: completed. `analyze --pr` runs collector/linker/scorer for one MVP metric (`conversion_rate`) with deployment fallback and structured response staging. Added validation so scorer payloads must include finite numeric metric values and positive sample sizes, plus tests for invalid metric values, invalid sample sizes, and command-level structured failure envelopes when analysis pipeline validation fails.
- M5: not started.
- M6: not started.

## Key decisions
- Keep implementation aligned to Phase 1 only; defer multi-metric, feature grouping, and leaderboard work to later phases.
- Preserve architecture boundaries (`cmd/git-impact` thin; `internal/gitimpact` behavior).
- Treat machine-readable structured output as the primary contract for automation-facing commands.
- Isolate Velen subprocess usage behind abstractions to keep tests deterministic.
- Standardize successful command responses on a single envelope contract with a top-level `result` object so automation clients do not need command-specific top-level field parsing.
- Keep Phase 1 config parsing strict for required `velen` and `analysis` fields while treating missing analysis values as deterministic defaults.
- Reject non-finite numeric values in config validation so analysis confidence remains a stable 0..1 scalar contract.
- Use `velen source show` as the capability authority for required sources and treat detail lookup failure on listed sources as a readiness failure.
- Treat scorer outputs as strict numeric contracts: malformed/non-finite metric values and non-positive samples fail analysis instead of silently coercing to zero.

## Remaining issues
- Decide final report scaffold file layout and naming for markdown/html outputs.

## Links
- Product spec: `SPEC.md`
- Architecture: `ARCHITECTURE.md`
- Non-negotiable rules: `NON_NEGOTIABLE_RULES.md`
- Plan policy: `docs/PLANS.md`
- Design doc index: `docs/design-docs/index.md`
- Product spec index: `docs/product-specs/index.md`
