# Step 9 of 10: Implement the TUI Analysis Progress View for git-impact

## Goal
Implement the analysis-progress TUI for `git-impact analyze` (per `SPEC.md` section 7.1), wire it into the runtime engine loop, add default engine construction helpers, and validate with tests and full package build/test.

## Background
- `SPEC.md` section 7.1 defines a progress UI with title, separator, progress bar, phase-status list, and wait prompt area.
- `internal/gitimpact/tui_model.go` currently has a minimal model and text view that does not match the required structure.
- `internal/gitimpact/tui_bridge.go` already defines Bubble Tea message types and `TUIObserver`.
- `cmd/git-impact/main.go` `analyze` command is currently a placeholder output and does not run the phased engine.
- `internal/gitimpact/engine.go` supports observer notifications and phased execution but lacks a convenience constructor that registers all default handlers.

## Milestones
- [ ] M1 (`not started`): Expand `AnalysisModel` in `internal/gitimpact/tui_model.go` with required fields (`phases`, `currentPhase`, `iteration`, `totalPhases`, `isWaiting`, `waitMessage`, `spinner`, `done`, `result`, `err`) and phase-aware status transitions for all engine bridge messages.
- [ ] M2 (`not started`): Implement `View()` to render a SPEC-aligned progress layout (title, separator, progress bar/turn + phase label, phase rows with done/running/waiting icons, and conditional wait message block).
- [ ] M3 (`not started`): Add `internal/gitimpact/tui_model_test.go` covering `Update()` behavior for `TurnStartedMsg`, `PhaseAdvancedMsg`, `WaitEnteredMsg`, `WaitResolvedMsg`, `RunCompletedMsg`, `RunExhaustedMsg`, and spinner tick handling.
- [ ] M4 (`not started`): Add `NewDefaultEngine(client *VelenClient, observer Observer, waitHandler WaitHandler) *Engine` in `internal/gitimpact/` to build an `Engine` with all phase handlers registered.
- [ ] M5 (`not started`): Update `cmd/git-impact/main.go` `analyze` command to instantiate and run Bubble Tea + `TUIObserver`, construct `RunContext`, execute engine on main goroutine, wait for TUI shutdown, and validate with `go build ./...` and `go test ./...`.

## Current progress
- Repository guidance reviewed: `AGENTS.md`, `NON_NEGOTIABLE_RULES.md`, `docs/PLANS.md`.
- Relevant implementation surfaces identified: `SPEC.md` 7.1, `internal/gitimpact/tui_model.go`, `internal/gitimpact/tui_bridge.go`, `internal/gitimpact/engine.go`, `cmd/git-impact/main.go`.
- No code changes started yet.

## Key decisions
- Treat this step as a direct implementation step, not a mock/prototype: all required Go code and tests will be committed.
- Keep phase progression source-of-truth in engine observer messages (TUI should only react to incoming messages, not infer hidden state transitions).
- Keep default engine registration centralized via `NewDefaultEngine` to avoid duplicating handler wiring in CLI entrypoints.
- Maintain machine-readable CLI behavior (`--output json`) where existing non-interactive paths require structured output.

## Remaining issues
- Need to finalize how `analyze --output json` should behave while adding an interactive TUI path for terminal output.
- Need to ensure Bubble Tea lifecycle coordination prevents deadlocks between engine completion and program shutdown.
- Need to confirm report-phase default handler behavior (complete directive and output payload expectations).

## Links
- `SPEC.md` (section 7.1)
- `internal/gitimpact/tui_model.go`
- `internal/gitimpact/tui_bridge.go`
- `internal/gitimpact/engine.go`
- `cmd/git-impact/main.go`
- `docs/PLANS.md`
