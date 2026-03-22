# Step 2 of 10: Implement the Velen CLI wrapper for the git-impact tool

## Goal
Implement a production-safe Velen CLI wrapper in `internal/gitimpact` that can authenticate context, discover sources, run read-only queries, and return structured results/errors for automation.

## Background
- `SPEC.md` section 4 requires all external data access through Velen CLI commands (`auth whoami`, `org current`, `source list/show`, `query`).
- `SPEC.md` section 11 requires read-only query behavior, org verification before analysis, and source availability checks.
- Non-negotiable repository rules require test coverage for all new behavior and machine-readable structured error handling.
- `internal/gitimpact` is not present in this worktree yet, so this step must create the package surface needed for Velen integration.

## Milestones
| ID | Milestone | Status | Exit criteria |
| --- | --- | --- | --- |
| M1 | Define Velen client and result/error types | not started | `internal/gitimpact` contains `VelenClient`, `WhoAmIResult`, `OrgResult`, `Source`, `QueryResult`, `VelenError`, and `Source.SupportsQuery()` with JSON tags aligned to CLI payloads. |
| M2 | Implement safe command execution wrapper | not started | Client runs `velen` via `os/exec` with timeout context, captures stdout+stderr, parses JSON, and maps non-zero exits to structured `VelenError`. |
| M3 | Implement command methods | not started | `WhoAmI`, `CurrentOrg`, `ListSources`, `ShowSource`, and `Query` call the wrapper with correct args and decode expected payloads. |
| M4 | Add unit tests with fake velen binary | not started | Table-driven tests cover success, JSON parse failures, timeout handling, command failure mapping, and SQL argument passing for `Query`. |
| M5 | Validate build and test suite | not started | `go build ./...` and `go test ./...` pass with the new package and tests. |

## Current progress
- Plan created.
- Implementation not started.

## Key decisions
- Use direct `exec.CommandContext` argument lists only; never shell execution.
- Keep a default timeout of 30 seconds via constructor, while allowing caller override.
- Treat non-zero Velen exits as structured errors that include code and combined stderr/stdout diagnostics.
- Favor deterministic tests using a fake helper process rather than requiring a real Velen installation.

## Remaining issues
- Confirm final field-level JSON shapes against actual Velen output if differences appear during integration.
- Decide whether future steps need richer query metadata beyond `columns`, `rows`, and `row_count`.

## Links
- Spec: `SPEC.md` (sections 4 and 11)
- Plan policy: `docs/PLANS.md`
- Merge blockers: `NON_NEGOTIABLE_RULES.md`
- Architecture boundaries: `ARCHITECTURE.md`
