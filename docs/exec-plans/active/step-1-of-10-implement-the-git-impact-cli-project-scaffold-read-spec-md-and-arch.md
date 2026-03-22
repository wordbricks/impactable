# Step 1 of 10 - Implement the git-impact CLI project scaffold (read SPEC.md and ARCHITECTURE.md)

## Goal
Create the initial `git-impact` Go project scaffold described in `SPEC.md`, including CLI entrypoint stubs, core `internal/gitimpact` stubs, config/context plumbing, initial tests, and a successful `go build ./...` baseline.

## Background
`SPEC.md` defines the Git Impact Analyzer as a new monorepo CLI with `analyze` and `check-sources` commands plus config-driven analysis behavior. `ARCHITECTURE.md` requires thin `cmd/*` entrypoints and behavior inside `internal/*` packages. This step establishes the first shippable foundation before implementing full analysis logic.

## Milestones
- [ ] M1 (`not started`): Add `cmd/git-impact/main.go` with Cobra root command and stub `analyze` + `check-sources` subcommands, delegating implementation surface to `internal/gitimpact`.
- [ ] M2 (`not started`): Create `internal/gitimpact/types.go` with initial domain types (`Config`, `AnalysisContext`, `PR`, `Deployment`, `FeatureGroup`, `ContributorStats`, `PRImpact`, `AnalysisResult`) aligned to `SPEC.md` terminology.
- [ ] M3 (`not started`): Implement `internal/gitimpact/config.go` for YAML config loading via Viper using `impact-analyzer.yaml` schema from SPEC section 8, with defaults for analysis windows.
- [ ] M4 (`not started`): Implement `internal/gitimpact/context.go` for converting CLI arguments into `AnalysisContext`.
- [ ] M5 (`not started`): Add repo-root `./git-impact` executable shim matching existing wrapper style (`./ralph-loop`, `./wtl`).
- [ ] M6 (`not started`): Add/update dependencies in `go.mod` for Cobra and Viper and write tests for config loading + context construction.
- [ ] M7 (`not started`): Run validation (`go build ./...`, relevant tests), address failures, and confirm scaffold compiles cleanly.

## Current progress
- Execution plan initialized.
- Implementation milestones not yet started.

## Key decisions
- Keep `cmd/git-impact` thin per architecture boundary rules; place behavior in `internal/gitimpact`.
- Start with command stubs and testable config/context primitives to enable incremental follow-up steps.
- Encode defaults in config load path so CLI runs remain predictable without full user config.

## Remaining issues
- Exact field-level shape for all domain structs may evolve in later steps as WTL phase integration is implemented.
- Source-check and analyze command runtime logic remain stubbed in this step by design.

## Links
- `SPEC.md`
- `ARCHITECTURE.md`
- `docs/PLANS.md`
- `docs/exec-plans/active/step-1-of-10-implement-the-git-impact-cli-project-scaffold-read-spec-md-and-arch.md`
