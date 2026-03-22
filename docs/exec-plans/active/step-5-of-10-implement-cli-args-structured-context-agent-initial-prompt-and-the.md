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
1. **Context construction and prompt assembly** — `not started`
   - Implement `NewAnalysisContext(...)` to load config via Viper and populate `AnalysisContext`.
   - Implement `BuildInitialPrompt(...)` to inject since/pr/feature inputs and configured Velen source keys.
2. **CLI command surface** — `not started`
   - Build root Cobra command (`git-impact`) with persistent `--config` and `--output` flags.
   - Add `analyze` subcommand with `--since`, `--pr`, `--feature`; print placeholder plus parsed context JSON.
3. **Source check implementation** — `not started`
   - Add `CheckSources(...)` and `SourceCheckResult` in `internal/gitimpact/check_sources.go`.
   - Execute `WhoAmI`, `CurrentOrg`, and `ListSources`; detect GitHub and Analytics providers case-insensitively.
   - Set support flags via `SupportsQuery()` and apply config-key fallback matching.
4. **Command wiring and output modes** — `not started`
   - Wire `check-sources` subcommand to call `CheckSources`.
   - Emit text summary for human output and JSON payload for machine-readable output.
5. **Validation and tests** — `not started`
   - Add table-driven tests in `check_sources_test.go`.
   - Run `go build ./...` and `go test ./...` and fix any breakages.

## Current progress
- Step plan created.
- No implementation changes started yet.

## Key decisions
- Reuse existing `Config`, `AnalysisContext`, `Source`, Velen client, and engine types as-is.
- Keep output behavior aligned with non-negotiable machine-readable automation rules.
- Keep `analyze` behavior minimal in this step (context parsing + placeholder output), deferring full analysis execution.

## Remaining issues
- Confirm exact output shape expected by downstream automation for text vs JSON modes once command wiring is in place.
- Verify provider-type fallback behavior against real `velen source list` payload variations.

## Links
- `SPEC.md` (sections 4.3 and 7)
- `NON_NEGOTIABLE_RULES.md`
- `docs/PLANS.md`
- `internal/gitimpact/`
- `cmd/git-impact/main.go`
