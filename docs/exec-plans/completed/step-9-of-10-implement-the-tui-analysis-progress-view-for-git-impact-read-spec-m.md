# Step 9 of 10: Implement the TUI Analysis Progress View for git-impact

## Goal
Implement the analysis-progress TUI for `git-impact analyze` (per `SPEC.md` section 7.1), wire it into the runtime engine loop, and validate with tests and full package build/test.

## Historical note
This plan originally introduced a local default-engine helper path (`NewDefaultEngine`, `DefaultHandlers`, `engine_defaults.go`). That path was later removed after `git-impact analyze` standardized on the Codex app-server agent engine for all phases.

## Background
- `SPEC.md` section 7.1 defines a progress UI with title, separator, progress bar, phase-status list, and wait prompt area.
- `internal/gitimpact/tui_model.go` currently has a minimal model and text view that does not match the required structure.
- `internal/gitimpact/tui_bridge.go` already defines Bubble Tea message types and `TUIObserver`.
- `cmd/git-impact/main.go` `analyze` command is currently a placeholder output and does not run the phased engine.
- `internal/gitimpact/engine.go` supports observer notifications and phased execution.

## Milestones
- [x] M1 (`completed`): Expanded `AnalysisModel` in `internal/gitimpact/tui_model.go` with required fields (`phases`, `currentPhase`, `iteration`, `totalPhases`, `isWaiting`, `waitMessage`, `spinner`, `done`, `result`, `err`) and phase-aware status transitions for all engine bridge messages.
- [x] M2 (`completed`): Implemented `View()` with a SPEC-aligned progress layout (title, separator, progress bar/turn + phase label, phase rows with done/running/waiting icons, and conditional wait message block).
- [x] M3 (`completed`): Added `internal/gitimpact/tui_model_test.go` covering `Update()` behavior for `TurnStartedMsg`, `PhaseAdvancedMsg`, `WaitEnteredMsg`, `WaitResolvedMsg`, `RunCompletedMsg`, `RunExhaustedMsg`, and spinner tick handling.
- [x] M4 (`completed`, later removed): Added `NewDefaultEngine(client *OneQueryClient, observer Observer, waitHandler WaitHandler) *Engine` plus `DefaultHandlers()` in `internal/gitimpact/` with all phase handlers registered.
- [x] M5 (`completed`): Updated `cmd/git-impact/main.go` `analyze` command to instantiate and run Bubble Tea + `TUIObserver`, construct `RunContext`, execute engine on main goroutine, and wait for TUI shutdown.

## Current progress
- Repository guidance reviewed: `AGENTS.md`, `NON_NEGOTIABLE_RULES.md`, `docs/PLANS.md`.
- Implemented full progress TUI model and view in `internal/gitimpact/tui_model.go`.
- Added `internal/gitimpact/tui_model_test.go`. The temporary `engine_defaults_test.go` and `engine_defaults.go` files from the removed default-engine path no longer exist.
- Wired `cmd/git-impact/main.go` analyze flow to run engine + Bubble Tea concurrently in interactive text mode and non-interactive engine execution for JSON/piped modes.
- Verification complete: `go build ./...` and `go test ./...` both passed (using `GOCACHE=/tmp/go-build-cache` in sandbox).

## Key decisions
- Treat this step as a direct implementation step, not a mock/prototype: all required Go code and tests will be committed.
- Keep phase progression source-of-truth in engine observer messages (TUI should only react to incoming messages, not infer hidden state transitions).
- Preserve machine-readable CLI behavior (`--output json`) while keeping phase execution on the Codex app-server agent engine.
- Run interactive progress TUI only for terminal text output to avoid renderer noise in automation/piped output.

## Remaining issues
- None for this milestone.

## Links
- `SPEC.md` (section 7.1)
- `internal/gitimpact/tui_model.go`
- `internal/gitimpact/tui_bridge.go`
- `internal/gitimpact/engine.go`
- `cmd/git-impact/main.go`
- `docs/PLANS.md`
