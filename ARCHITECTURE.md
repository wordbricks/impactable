# Architecture

## Purpose

This repository currently contains two related systems:

1. A Go implementation of `ralph-loop`, an agent-first CLI that prepares a worktree, drives Codex through a setup and coding loop, and streams structured run data.
2. A Go implementation of `wtl`, a minimal WhatTheLoop CLI that runs a single prompt loop through Codex app-server using explicit engine, policy, and observer roles.
3. A Rust `harnesscli` bootstrap that turns the repository into a harnessed codebase with stable commands for smoke, lint, typecheck, test, audit, init, boot, observability, and cleanup.

The long-term product direction in [`SPEC.md`](SPEC.md) is a Git impact analyzer. The code that exists today is still mostly harness and orchestration infrastructure.

## Package Boundaries

### Go runtime

- `cmd/ralph-loop`
  - Thin CLI entrypoint.
  - Resolves the current working directory and hands control to `internal/ralphloop`.
- `cmd/wtl`
  - Thin CLI entrypoint.
  - Resolves the current working directory and hands control to `internal/wtl`.
- `internal/ralphloop`
  - Parsing, schema generation, worktree management, session tracking, log handling, JSON-RPC transport, and orchestration.
  - This package is the only place that should contain Ralph Loop behavior.
- `internal/wtl`
  - Parsing, prompt intake, WTL engine/policy/observer coordination, structured output handling, and Codex app-server turn execution for the WTL CLI.
  - This package is the only place that should contain WTL behavior.

Dependency direction:

- `cmd/ralph-loop` -> `internal/ralphloop`
- `cmd/wtl` -> `internal/wtl`
- `internal/ralphloop` -> Go standard library
- `internal/wtl` -> Go standard library

### Rust harness runtime

- `harness/src/main.rs`
  - `clap` entrypoint and shared process exit handling.
- `harness/src/cmd/*`
  - One module per command group.
- `harness/src/util`
  - Shared output, process, worktree, and filesystem helpers.

Dependency direction:

- `main` -> `cmd`, `util`
- `cmd/*` -> `util`
- `util` -> Rust stdlib plus small serialization helpers

## Repository Zones

- `specs/`
  - Imported upstream specs and references. Treat these as vendored inputs, not primary implementation locations.
- `docs/design-docs/`
  - Canonical operational and engineering rationale.
- `docs/product-specs/`
  - Product-facing behavior descriptions, including the harness demo app contract.
- `docs/exec-plans/`
  - Checked-in execution plans and technical debt tracking.
- `.worktree/`
  - Runtime state generated per Git worktree by the harness system.

## Entry Points

- `./ralph-loop`
  - Repo-root executable wrapper around `go run ./cmd/ralph-loop`.
- `./wtl`
  - Repo-root executable wrapper around `go run ./cmd/wtl`.
- `cargo build --release --manifest-path harness/Cargo.toml`
  - Builds `harnesscli`.
- `make smoke`, `make check`, `make test`
  - Stable top-level validation commands once the harness is built.

## Boundary Rules

- Do not add new Ralph Loop logic under `cmd/`.
- Do not add new WTL logic under `cmd/`.
- Do not place durable operating guidance in `AGENTS.md`.
- Do not edit imported spec files unless the change is explicitly a spec sync.
- All new automation-facing repository operations should prefer `harnesscli` subcommands over ad hoc shell scripts.
