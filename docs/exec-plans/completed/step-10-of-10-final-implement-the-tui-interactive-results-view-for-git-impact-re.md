# Step 10 of 10 (Final): Implement the TUI Interactive Results View for git-impact

## Goal
Implement the post-analysis interactive results experience for `git-impact` with Bubble Tea, including multi-view navigation (PRs, features, contributors), PR drill-down details, report export (`.md`/`.html`), and wiring from run completion into the results screen.

## Background
- `SPEC.md` section 7.2 defines the interactive results table, keybindings, and save action behavior after analysis completion.
- `SPEC.md` section 7.3 defines PR detail view requirements, including author/timing context, score display, metric confidence breakdown, and agent reasoning text.
- `SPEC.md` section 7.4 defines feature-level aggregated impact presentation.
- `SPEC.md` section 7.5 defines contributor leaderboard fields and ordering.
- `internal/gitimpact` already contains core engine and data model types (`AnalysisResult`, `PRImpact`, `FeatureGroup`, `ContributorStats`) that must be surfaced via TUI and report output.

## Milestones
| ID | Milestone | Status | Exit criteria |
| --- | --- | --- | --- |
| M1 | Add results TUI model (`tui_results.go`) | completed | `ResultsModel` exists with required fields and Bubble Tea `Init`, `Update`, `View` behavior for view switching, cursor movement, drill-down, back navigation, quit, and viewport/table integration. |
| M2 | Implement report exporters (`report.go`) | completed | `SaveMarkdown` and `SaveHTML` render PR impacts, feature groups, and contributor leaderboard from `AnalysisResult` and persist successfully to requested path. |
| M3 | Wire analysis completion to results UI flow | completed | `AnalysisModel` and CLI runtime transition from progress model to `ResultsModel` on `RunCompletedMsg` without losing analysis result payload. |
| M4 | Implement save-flow keybinding integration | completed | `s` key triggers format selection (`md`/`html`) and calls the corresponding report saver, with errors surfaced in the TUI loop. |
| M5 | Add coverage for results + reports | completed | `tui_results_test.go` and `report_test.go` validate view rendering/key handling and markdown/html output structure/content for key scenarios. |
| M6 | Verify build/tests and finalize branch output | completed | `go build ./...` and `go test ./...` pass after changes; final step outputs are staged and committed. |

## Current progress
- Added `internal/gitimpact/tui_results.go` with interactive post-run `ResultsModel` including PR table, feature list, contributor leaderboard, PR/feature drill-down, viewport scrolling, and key handling for `q`, `tab`, `up/down`, `enter`, `esc`, and `s`.
- Added `internal/gitimpact/report.go` with `SaveMarkdown` and `SaveHTML` exports covering PR impacts, feature summaries, and contributor rankings.
- Updated `internal/gitimpact/tui_model.go` to signal result-view handoff through `RunCompletedMsg` payload retention (`ShouldShowResults`/`Result`).
- Updated `cmd/git-impact/main.go` to run progress TUI first, then switch to results TUI and wire save callbacks to markdown/html exporters.
- Added `internal/gitimpact/tui_results_test.go` and `internal/gitimpact/report_test.go` with coverage for navigation, drill-down, save flow, viewport scrolling, and exported report structure.
- Verification complete with sandbox-local cache: `GOCACHE=/tmp/go-build-cache go build ./...` and `GOCACHE=/tmp/go-build-cache go test ./...`.

## Key decisions
- Keep results interactions in a dedicated `ResultsModel` to isolate post-run UX concerns from progress-phase rendering.
- Reuse Bubble Tea `table` and `viewport` components for predictable keyboard navigation and scrolling behavior.
- Centralize file output formatting in `report.go` so the TUI save action and future non-interactive export paths share one output implementation.
- Use in-TUI save format prompting (`s` then `m`/`h`) while wiring actual persistence through `main` via injected save handler callbacks.
- Launch results TUI as a second Bubble Tea program after progress completion to avoid input contention with wait-prompt handling during engine execution.

## Remaining issues
- None identified for Step 10 scope.

## Links
- `SPEC.md` (sections 7.2, 7.3, 7.4, 7.5)
- `internal/gitimpact/tui_model.go`
- `cmd/git-impact/main.go`
- `internal/gitimpact/types.go`
- `docs/PLANS.md`
- `NON_NEGOTIABLE_RULES.md`
