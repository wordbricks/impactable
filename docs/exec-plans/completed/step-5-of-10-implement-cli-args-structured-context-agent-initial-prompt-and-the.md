# Step 5 of 10: Implement CLI args, structured context, initial prompt, and check-sources

## Goal
Implement Step 5 for `git-impact` by wiring CLI arguments into a structured analysis context, building the agent initial prompt from that context and config, and adding a `check-sources` flow that validates required Velen source connectivity.

## Background
`SPEC.md` section 7 requires `git-impact analyze` and `git-impact check-sources`, with analyze arguments passed into a structured context object and included in the agent's initial prompt. `SPEC.md` section 4.3 requires source discovery on every analyze run: list sources, identify GitHub and Analytics sources by provider type, confirm query capability, and surface gaps.

Existing types and engine already exist in `internal/gitimpact/` and must be reused (no type redefinition). This step adds implementation glue in:
- `internal/gitimpact/context.go`
- `internal/gitimpact/check_sources.go`
- `cmd/git-impact/main.go`
- tests for source detection and fallback behavior

## Milestones
1. **Context construction and prompt assembly** — `completed`
   - Implement `NewAnalysisContext(...)` to load config via Viper and populate `AnalysisContext`.
   - Implement `BuildInitialPrompt(...)` to inject since/pr/feature inputs and configured Velen source keys.
2. **CLI command surface** — `completed`
   - Build root Cobra command (`git-impact`) with persistent `--config` and `--output` flags.
   - Add `analyze` subcommand with `--since`, `--pr`, `--feature`; print placeholder plus parsed context JSON.
3. **Source check implementation** — `completed`
   - Add `CheckSources(...)` and `SourceCheckResult` in `internal/gitimpact/check_sources.go`.
   - Execute `WhoAmI`, `CurrentOrg`, and `ListSources`; detect GitHub and Analytics providers case-insensitively.
   - Set support flags via `SupportsQuery()` and apply config-key fallback matching.
4. **Command wiring and output modes** — `completed`
   - Wire `check-sources` subcommand to call `CheckSources`.
   - Emit text summary for human output and JSON payload for machine-readable output.
5. **Validation and tests** — `completed`
   - Add table-driven tests in `check_sources_test.go`.
   - Run `go build ./...` and `go test ./...` and fix any breakages.

## Current progress
- Implemented `NewAnalysisContext(since, prNum, feature, configPath)` and `BuildInitialPrompt` in `internal/gitimpact/context.go`.
- Added `internal/gitimpact/check_sources.go` with provider-type detection, query-capability checks, and config-key fallback.
- Reworked `cmd/git-impact/main.go` to a full Cobra command surface with root persistent flags and wired `analyze` / `check-sources`.
- Added `internal/gitimpact/check_sources_test.go` table-driven coverage plus helper-process command failure coverage.
- Updated context tests for the new `NewAnalysisContext` signature and added prompt-content assertions.
- Validation complete: `GOCACHE=/tmp/go-build-cache go build ./...` and `GOCACHE=/tmp/go-build-cache go test ./...` both pass.

## Key decisions
- Reuse existing `Config`, `AnalysisContext`, `Source`, Velen client, and engine types as-is.
- Keep output behavior aligned with non-negotiable machine-readable automation rules by defaulting non-TTY output to JSON.
- Keep `analyze` behavior minimal in this step (context parsing + placeholder output), while also emitting the generated initial prompt.
- Keep `check-sources` output non-fatal for missing source capability by returning structured status (`ok`/`issues`) and per-source errors.

## Remaining issues
- Confirm downstream automation envelope expectations (`status` semantics and field naming) before integrating into the next analysis phase.
- Verify provider-type fallback behavior against real-world `velen source list` payload variations beyond test fixtures.

## Links
- `SPEC.md` (sections 4.3 and 7)
- `NON_NEGOTIABLE_RULES.md`
- `docs/PLANS.md`
- `internal/gitimpact/`
- `cmd/git-impact/main.go`
