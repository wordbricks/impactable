# Step 1 of 10 - Implement the git-impact CLI project scaffold (read SPEC.md and ARCHITECTURE.md)

## Goal
Create the initial `git-impact` Go project scaffold described in `SPEC.md`, including CLI entrypoint stubs, core `internal/gitimpact` stubs, config/context plumbing, initial tests, and a successful `go build ./...` baseline.

## Background
`SPEC.md` defines the Git Impact Analyzer as a new monorepo CLI with `analyze` and `check-sources` commands plus config-driven analysis behavior. `ARCHITECTURE.md` requires thin `cmd/*` entrypoints and behavior inside `internal/*` packages. This step establishes the first shippable foundation before implementing full analysis logic.

## Milestones
- [x] M1 (`completed`): Added `cmd/git-impact/main.go` with thin entrypoint into `internal/gitimpact` and Cobra-backed `analyze` + `check-sources` stubs.
- [x] M2 (`completed`): Created `internal/gitimpact/types.go` with initial domain/config/result structs aligned to `SPEC.md`.
- [x] M3 (`completed`): Implemented `internal/gitimpact/config.go` using Viper for `impact-analyzer.yaml` load/decode with default analysis windows.
- [x] M4 (`completed`): Implemented `internal/gitimpact/context.go` to convert CLI args into validated `AnalysisContext`.
- [x] M5 (`completed`): Added repo-root `./git-impact` executable shim matching existing wrapper style.
- [x] M6 (`completed`): Updated `go.mod`/`go.sum` with Cobra + Viper dependencies and added tests for config loading/context construction plus command-stub behavior.
- [x] M7 (`completed`): Validation passed via `go test ./...` and `go build ./...` (with `GOCACHE=/tmp/go-build-cache` for sandbox compatibility).

## Current progress
- Step 1 scaffold is implemented end-to-end and committed-ready.
- New package surface exists at `internal/gitimpact` (`cli.go`, `types.go`, `config.go`, `context.go`) with command stubs and shared config/context plumbing.
- Tests now cover config defaults/overrides, CLI-arg context conversion, and stub command execution paths.
- Validation baseline is green for the repository after dependency additions.

## Key decisions
- Keep `cmd/git-impact` thin per architecture boundary rules; place behavior in `internal/gitimpact`.
- Start with command stubs and testable config/context primitives to enable incremental follow-up steps.
- Encode defaults in config load path so CLI runs remain predictable without full user config.
- Keep `analyze` and `check-sources` behavior explicitly stubbed by returning `not implemented` sentinel errors after config/context validation.
- Resolve relative config paths against the caller working directory during context construction.

## Remaining issues
- Exact field-level shape for some domain structs may evolve in later steps as WTL phase integration is implemented.
- `analyze` and `check-sources` command runtime logic remains stubbed in this step by design.

## Links
- `SPEC.md`
- `ARCHITECTURE.md`
- `docs/PLANS.md`
- `docs/exec-plans/active/step-1-of-10-implement-the-git-impact-cli-project-scaffold-read-spec-md-and-arch.md`
