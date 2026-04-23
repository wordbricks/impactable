# Architecture

## Purpose

This repository currently contains four related systems:

1. A Go implementation of `git-impact`, a phased Git impact analyzer CLI that runs each analysis phase as a Codex app-server WTL Agent turn and uses OneQuery for source discovery and read-only queries.
2. A Go implementation of `ralph-loop`, an agent-first CLI that prepares a worktree, drives Codex through a setup and coding loop, and streams structured run data.
3. A Go implementation of `wtl`, a minimal WhatTheLoop CLI that runs a single prompt loop through Codex app-server using explicit engine, policy, and observer roles.
4. A Rust `harnesscli` bootstrap that turns the repository into a harnessed codebase with stable commands for smoke, lint, typecheck, test, audit, init, boot, observability, and cleanup.

The long-term product direction in [`SPEC.md`](SPEC.md) is a Git impact analyzer. The code that exists today includes a Codex app-server phase-agent runtime for the `git-impact analyze` path plus focused Go handlers used by package-level tests.

## Package Boundaries

### Go runtime

- `cmd/ralph-loop`
  - Thin CLI entrypoint.
  - Resolves the current working directory and hands control to `internal/ralphloop`.
- `cmd/wtl`
  - Thin CLI entrypoint.
  - Resolves the current working directory and hands control to `internal/wtl`.
- `cmd/git-impact`
  - Cobra CLI entrypoint for `analyze` and `check-sources`.
  - Loads config, builds analysis context, runs the phased engine with the Codex app-server agent runtime by default, and optionally starts the Bubble Tea TUI.
- `internal/ralphloop`
  - Parsing, schema generation, worktree management, session tracking, log handling, JSON-RPC transport, and orchestration.
  - This package is the only place that should contain Ralph Loop behavior.
- `internal/wtl`
  - Parsing, prompt intake, WTL engine/policy/observer coordination, structured output handling, and Codex app-server turn execution for the WTL CLI.
  - Exposes a reusable Codex app-server thread/turn client used by product-specific WTL integrations.
  - This package is the only place that should contain WTL behavior.
- `internal/gitimpact`
  - Config parsing, OneQuery subprocess integration, Codex phase-agent prompts and result parsing, package-level phase helpers, source checks, TUI models, and report rendering.
  - This package is the only place that should contain Git Impact Analyzer behavior.

Dependency direction:

- `cmd/ralph-loop` -> `internal/ralphloop`
- `cmd/wtl` -> `internal/wtl`
- `cmd/git-impact` -> `internal/gitimpact`
- `internal/gitimpact` -> `internal/wtl` for Codex app-server thread/turn execution
- `internal/ralphloop` -> Go standard library
- `internal/wtl` -> Go standard library
- `internal/gitimpact` -> Go standard library + CLI/TUI/config libraries

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
- `./git-impact`
  - Repo-root executable wrapper around `go run ./cmd/git-impact`.
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
