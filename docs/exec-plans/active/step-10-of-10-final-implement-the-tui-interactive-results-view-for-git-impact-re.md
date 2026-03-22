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
| M1 | Add results TUI model (`tui_results.go`) | not started | `ResultsModel` exists with required fields and Bubble Tea `Init`, `Update`, `View` behavior for view switching, cursor movement, drill-down, back navigation, quit, and viewport/table integration. |
| M2 | Implement report exporters (`report.go`) | not started | `SaveMarkdown` and `SaveHTML` render PR impacts, feature groups, and contributor leaderboard from `AnalysisResult` and persist successfully to requested path. |
| M3 | Wire analysis completion to results UI flow | not started | `AnalysisModel` and CLI runtime transition from progress model to `ResultsModel` on `RunCompletedMsg` without losing analysis result payload. |
| M4 | Implement save-flow keybinding integration | not started | `s` key triggers format selection (`md`/`html`) and calls the corresponding report saver, with errors surfaced in the TUI loop. |
| M5 | Add coverage for results + reports | not started | `tui_results_test.go` and `report_test.go` validate view rendering/key handling and markdown/html output structure/content for key scenarios. |
| M6 | Verify build/tests and finalize branch output | not started | `go build ./...` and `go test ./...` pass after changes; final step outputs are staged and committed. |

## Current progress
- Repository guidance reviewed (`AGENTS.md`, `NON_NEGOTIABLE_RULES.md`, `ARCHITECTURE.md`, `docs/PLANS.md`).
- SPEC sections 7.2, 7.3, 7.4, and 7.5 reviewed for final-step UI/report requirements.
- Implementation has not started; this document defines execution order and completion criteria.

## Key decisions
- Keep results interactions in a dedicated `ResultsModel` to isolate post-run UX concerns from progress-phase rendering.
- Reuse Bubble Tea `table` and `viewport` components for predictable keyboard navigation and scrolling behavior.
- Centralize file output formatting in `report.go` so the TUI save action and future non-interactive export paths share one output implementation.

## Remaining issues
- Save-format prompting must be integrated cleanly in Bubble Tea without degrading keyboard navigation expectations.
- Result-to-view mapping may require careful handling for empty datasets (no PRs/features/contributors).
- Main program transition semantics need to avoid racey handoff between run completion and interactive results initialization.

## Links
- `SPEC.md` (sections 7.2, 7.3, 7.4, 7.5)
- `internal/gitimpact/tui_model.go`
- `cmd/git-impact/main.go`
- `internal/gitimpact/types.go`
- `docs/PLANS.md`
- `NON_NEGOTIABLE_RULES.md`
