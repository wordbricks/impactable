# Step 7 of 10 (Retry): Implement the Linker phase handler (Turn 2) for git-impact

## Goal
Implement Turn 2 Linker behavior in `internal/gitimpact` to infer deployment timestamps from collected GitHub data, propose feature groups, detect ambiguity windows, and return correct phased-delivery directives.

## Background
- `SPEC.md` section 3.2 defines Linker responsibilities: infer deployment mapping from releases/tags/merge time and propose feature groupings.
- `SPEC.md` section 5.2 defines deployment inference priority: release publish time, then version tag markers, then PR merge time fallback.
- `internal/gitimpact` already has engine directives, `CollectedData`, `LinkedData`, `Deployment`, `FeatureGroup`, and `AmbiguousDeployment` types.
- This retry requires concrete Go implementation and tests, followed by repository-wide build and test verification.

## Milestones
| ID | Milestone | Status | Exit criteria |
| --- | --- | --- | --- |
| M1 | Add Linker phase handler scaffold | not started | `internal/gitimpact/phase_link.go` exists with `LinkHandler` and `Handle(context.Context, *RunContext)` wired to `CollectedData`/`LinkedData`. |
| M2 | Implement deployment inference helpers | not started | `inferDeployment` + `isVersionTag` support priority ordering and 48h window checks with deterministic selection. |
| M3 | Implement feature grouping proposals | not started | `proposeFeatureGroups` groups PRs by `feature/` label prefix and `feature/` branch prefix. |
| M4 | Implement ambiguity detection and wait behavior | not started | `detectAmbiguousDeployments` identifies multi-release/multi-PR 24h windows and `Handle` returns `DirectiveWait` with descriptive message when ambiguous. |
| M5 | Add and pass Linker tests | not started | `internal/gitimpact/phase_link_test.go` has >=5 inference-focused tests and `go test ./...` passes. |
| M6 | Verify full build and finalize commit | not started | `go build ./...` passes and Linker changes are staged and committed cleanly. |

## Current progress
- Completed repo orientation (`AGENTS.md`, `NON_NEGOTIABLE_RULES.md`, `docs/PLANS.md`).
- Read `SPEC.md` sections 3.2 and 5.2 and reviewed existing `internal/gitimpact` phase/type implementations.
- Linker implementation and tests not started yet.

## Key decisions
- Preserve existing phased-delivery contract (`DirectiveAdvancePhase`/`DirectiveWait`) and keep Linker pure over in-memory `CollectedData`.
- Use deterministic helper behavior (sorted/nearest matching by timestamp) to avoid nondeterministic test failures.
- Keep ambiguity handling conservative: pause only when overlapping release windows and merged PR windows indicate unclear mapping.

## Remaining issues
- `CollectedData.Tags` is currently `[]string`; tag timestamp parsing format must be defined in Linker logic to support the `Tag.CreatedAt` inference window.
- Decide exact wait-message detail level to balance clarity and concise terminal prompts.

## Links
- `SPEC.md` (sections 3.2 and 5.2)
- `internal/gitimpact/engine.go`
- `internal/gitimpact/phase_collect.go`
- `docs/PLANS.md`
- `NON_NEGOTIABLE_RULES.md`
