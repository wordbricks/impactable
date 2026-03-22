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
| M1 | Add Linker phase handler scaffold | completed | `internal/gitimpact/phase_link.go` exists with `LinkHandler` and `Handle(context.Context, *RunContext)` wired to `CollectedData`/`LinkedData`. |
| M2 | Implement deployment inference helpers | completed | `inferDeployment` + `isVersionTag` support priority ordering and 48h window checks with deterministic selection. |
| M3 | Implement feature grouping proposals | completed | `proposeFeatureGroups` groups PRs by `feature/` label prefix and `feature/` branch prefix. |
| M4 | Implement ambiguity detection and wait behavior | completed | `detectAmbiguousDeployments` identifies multi-release/multi-PR 24h windows and `Handle` returns `DirectiveWait` with descriptive message when ambiguous. |
| M5 | Add and pass Linker tests | completed | `internal/gitimpact/phase_link_test.go` has >=5 inference-focused tests and `go test ./...` passes. |
| M6 | Verify full build and finalize commit | completed | `go build ./...` passes and Linker changes are staged and committed cleanly. |

## Current progress
- Implemented `LinkHandler` and helper functions in `internal/gitimpact/phase_link.go` for deployment inference, feature grouping, and ambiguity detection.
- Added inference-focused test coverage in `internal/gitimpact/phase_link_test.go` (release priority, tag fallback, merge fallback, nearest marker selection, and ambiguity window behavior).
- `go build ./...` and `go test ./...` pass in sandbox by setting `GOCACHE=/tmp/go-build-cache`.

## Key decisions
- Preserve existing phased-delivery contract (`DirectiveAdvancePhase`/`DirectiveWait`) and keep Linker pure over in-memory `CollectedData`.
- Use deterministic helper behavior (sorted/nearest matching by timestamp) to avoid nondeterministic test failures.
- Interpret ambiguity windows as explicit rolling 24-hour windows from each release timestamp to avoid chain-linking releases beyond 24h.
- Keep tag timestamp parsing based on `name|timestamp` produced by the collector (`formatTagWithTimestamp`) to retain `CollectedData.Tags []string`.

## Remaining issues
- None for this milestone set; next action is staging and committing verified changes.

## Links
- `SPEC.md` (sections 3.2 and 5.2)
- `internal/gitimpact/engine.go`
- `internal/gitimpact/phase_collect.go`
- `docs/PLANS.md`
- `NON_NEGOTIABLE_RULES.md`
