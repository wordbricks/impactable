# Step 3 of 10 - Implement the WTL engine and PhasedDeliveryPolicy for the git-impact tool

## Goal
Implement the git-impact WTL execution engine and phased-delivery control flow so analysis progresses across Source Check, Collect, Link, Score, and Report with deterministic directive handling, observer lifecycle hooks, wait/resume behavior, and test coverage.

## Background
- `SPEC.md` section 3 defines the git-impact architecture as a single WTL run split into ordered phases.
- `SPEC.md` section 3.1 defines phased directives and explicit `wait` behavior where terminal user input resumes the run.
- `internal/wtl` already provides the repository's baseline engine/policy loop pattern and testing style for directives and loop exhaustion.
- This step introduces `internal/gitimpact` as the git-impact-specific phased engine surface for later CLI and TUI integration.

## Milestones
| ID | Milestone | Status | Exit criteria |
| --- | --- | --- | --- |
| M1 | Define gitimpact engine contract | not started | `internal/gitimpact/engine.go` defines phases, directives, turn result, handler interface, run context, and analysis data structs with compile-safe types. |
| M2 | Implement phased-delivery loop mechanics | not started | `Engine.Run` executes ordered phase progression with retry limits, continue semantics, wait/resume callback flow, and completion/exhaustion paths. |
| M3 | Add observer integration surface | not started | Observer callbacks are wired for turn start, phase advance, wait entered/resolved, run complete, and run exhausted lifecycle points. |
| M4 | Add tests for core control flow | not started | `internal/gitimpact/engine_test.go` validates phase progression, retry logic, and wait handling behavior with deterministic assertions. |
| M5 | Verify repository health | not started | `go build ./...` and `go test ./...` both succeed with the new git-impact engine package included. |

## Current progress
- Plan created.
- Milestones M1-M5 are not started.

## Key decisions
- Mirror the existing `internal/wtl` pattern for loop mechanics while specializing directives/phases for git-impact.
- Keep wait handling callback-driven (`WaitHandler`) so later TUI or terminal prompt adapters can plug in without changing engine logic.
- Use explicit ordered phase list in engine control flow instead of implicit policy state to keep progression auditable.
- Keep retry handling phase-local with a fixed maximum of 3 retries per phase directive path.

## Remaining issues
- Exact runtime behavior for non-terminal `DirectiveContinue` in a phase without external state mutation may need refinement in later steps if handlers do not naturally converge.
- Domain types are introduced minimally in this step and may be expanded when collectors/linkers/scorers are implemented.

## Links
- Product spec: `SPEC.md`
- WTL package reference: `internal/wtl/engine.go`
- WTL policy reference: `internal/wtl/policy.go`
- WTL tests reference: `internal/wtl/engine_test.go`
- Plans policy: `docs/PLANS.md`
