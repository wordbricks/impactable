# Step 4 of 10: Implement the WTL Observer -> Bubble Tea Msg bridge for git-impact

## Goal
Implement the Observer-to-TUI bridge for git-impact so WTL lifecycle events are emitted as Bubble Tea messages, consumed by a minimal analysis progress model, and validated by tests.

## Background
- `SPEC.md` section 3 and 3.1 define a phase-driven WTL run and an explicit Observer -> Bubble Tea `Msg` mapping.
- Required message mapping for this step: `TurnStarted`, `PhaseAdvanced`, `WaitEntered`, `WaitResolved`, `RunCompleted`, `RunExhausted`.
- The current worktree has `internal/wtl` engine code, while `internal/gitimpact` is not yet present and will need to be introduced as part of this step's implementation.
- Repository merge rules require tests for new behavior (`NON_NEGOTIABLE_RULES.md` Rule 1).

## Milestones
| ID | Milestone | Status | Exit criteria |
| --- | --- | --- | --- |
| M1 | Add Bubble Tea dependencies to module | not started | `go.mod` includes `bubbletea`, `bubbles`, and `lipgloss`; dependency graph resolves via `go get`. |
| M2 | Define Observer->Msg bridge types and adapter | not started | `internal/gitimpact/tui_bridge.go` defines all required `tea.Msg` structs and a `TUIObserver` implementing the git-impact `Observer` interface with `program.Send(...)` dispatch. |
| M3 | Add minimal analysis progress model | not started | `internal/gitimpact/tui_model.go` defines `AnalysisModel`, `PhaseStatus`, `Init`, `Update`, and `View` handling all bridge message types with simple text rendering. |
| M4 | Add bridge message dispatch tests | not started | `internal/gitimpact/tui_bridge_test.go` verifies each Observer callback emits the expected Bubble Tea message payload. |
| M5 | Verify build and test health | not started | `go build ./...` and `go test ./...` pass after Step 4 changes. |

## Current progress
- Overall status: not started.
- Planning context loaded from `AGENTS.md`, `NON_NEGOTIABLE_RULES.md`, `docs/PLANS.md`, `ARCHITECTURE.md`, and `SPEC.md` sections 3 and 3.1.
- Implementation has not begun in this setup step.

## Key decisions
- Keep this step focused on the Observer-to-TUI bridge and minimal model state transitions; defer richer TUI visuals/interactions to Step 9 as scoped.
- Use one Bubble Tea `Msg` type per Observer event exactly as specified to keep event handling explicit and testable.
- Treat the bridge as an adapter layer in `internal/gitimpact` to avoid coupling WTL core internals directly to Bubble Tea update logic.

## Remaining issues
- `internal/gitimpact` source files (`engine.go`, `observer.go`) referenced by the task are not present in the current worktree and must be created or aligned in the implementation step.
- Exact `Phase` and `AnalysisResult` type definitions must match the Step 4 implementation context when code work starts.

## Links
- Product spec: `SPEC.md` (sections 3, 3.1)
- Plan policy: `docs/PLANS.md`
- Merge gates: `NON_NEGOTIABLE_RULES.md`
- System boundaries: `ARCHITECTURE.md`
