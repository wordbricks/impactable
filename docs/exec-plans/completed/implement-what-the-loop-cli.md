# Implement WhatTheLoop CLI

## Goal
Implement the minimal WhatTheLoop CLI from `specs/what-the-loop/SPEC.md` in Go using Codex app-server as the turn runtime.

## Background
- The target contract is the vendored WTL spec under `specs/what-the-loop/`.
- The repository already has Go JSON-RPC transport code for Codex app-server under `internal/ralphloop/`, but WTL needs its own package boundary and simpler loop semantics.
- Repository merge rules require tests for new behavior and machine-readable automation output for new commands.

## Milestones
| ID | Milestone | Status | Exit criteria |
| --- | --- | --- | --- |
| M1 | Define package shape and command contract | done (2026-03-20) | Separate `cmd/wtl` and `internal/wtl` exist with a minimal `wtl run` surface and explicit runtime defaults. |
| M2 | Implement engine, policy, observer, and Codex runner | done (2026-03-20) | The CLI can run a prompt through Codex app-server, stream turn output, detect `##WTL_DONE##`, and terminate with spec-aligned status handling. |
| M3 | Add tests and repo doc updates | done (2026-03-20) | Unit tests cover parser, loop directives, event ordering, and CLI exit behavior; architecture and local-operations docs mention the new runtime surface. |
| M4 | Verify and hand off | done (2026-03-20) | `go test ./...` passes and the remaining gaps are documented concisely. |

## Current progress
- Overall status: complete.
- Added `cmd/wtl` and `internal/wtl` with a minimal `wtl run` command surface.
- Implemented a WTL engine with explicit policy and observer roles plus a Codex app-server turn runner.
- Added parser, engine, and run-path tests for completion, retry exhaustion, text output, and structured failures.
- Updated `AGENTS.md`, `ARCHITECTURE.md`, and `docs/design-docs/local-operations.md` for the new runtime surface.

## Key decisions
- Keep WTL isolated from `internal/ralphloop` behavior instead of extending Ralph Loop.
- Preserve the spec's text UX for `wtl run` while adding `json` and `ndjson` output modes to satisfy repository automation rules.
- Use `approvalPolicy: "never"` and a `workspaceWrite` sandbox for the default Codex app-server runner to avoid approval deadlocks in the minimal implementation.

## Remaining issues
- Live integration with a local `codex app-server` binary still depends on the user environment; test coverage remains unit-level rather than end-to-end.
- Wait/resume and phased execution remain out of scope, matching the WTL CLI section of the spec.

## Links
- Spec: `specs/what-the-loop/SPEC.md`
- Codex app-server reference: `docs/references/codex-app-server-llm.txt`
- Existing transport reference: `internal/ralphloop/codex_client.go`
