# Local Operations

## Primary Commands

### Go Ralph Loop

- `./ralph-loop schema --output json`
  - Show the live command contract.
- `./ralph-loop init --dry-run --output json`
  - Preview worktree initialization details.
- `./ralph-loop "<prompt>" --output ndjson --preserve-worktree`
  - Run the loop and keep the generated worktree for inspection.

### Go WTL

- `printf 'Summarize this repository.\n' | ./wtl run`
  - Run the minimal WTL loop with text output in a terminal.
- `printf 'Summarize this repository.\n' | ./wtl run --output json`
  - Return one machine-readable run summary object.
- `printf 'Summarize this repository.\n' | ./wtl run --output ndjson`
  - Stream structured lifecycle events for the run.

### Harness CLI

- `cargo build --release --manifest-path harness/Cargo.toml`
  - Build `harnesscli`.
- `harness/target/release/harnesscli smoke`
  - Fast compile sanity check for the Go code.
- `harness/target/release/harnesscli lint`
  - Formatting plus static analysis checks.
- `harness/target/release/harnesscli typecheck`
  - Full repository build validation.
- `harness/target/release/harnesscli test`
  - Go tests plus Rust harness tests.
- `harness/target/release/harnesscli audit . --output json`
  - Verify required harness files and directories exist.
- `harness/target/release/harnesscli init`
  - Create the current worktree runtime root and metadata.
- `harness/target/release/harnesscli boot start`
  - Start the deterministic local demo app for this worktree.

### Make Targets

- `make smoke`
- `make lint`
- `make typecheck`
- `make check`
- `make test`

## Environment Variables

### Ralph Loop

- `RALPH_LOOP_CODEX_COMMAND`
  - Overrides the command used to start Codex app-server.

### WTL

- `WTL_CODEX_COMMAND`
  - Overrides the command used to start Codex app-server for `wtl`.
- `WTL_MODEL`
  - Overrides the model used for `wtl run`.

### Harness

- `HARNESS_SMOKE_CMD`
- `HARNESS_LINT_CMD`
- `HARNESS_TYPECHECK_CMD`
- `HARNESS_TEST_CMD`
  - Override the default command run by the matching `harnesscli` subcommand.

- `DISCODE_WORKTREE_ID`
  - Override the derived worktree ID.

- `APP_PORT_BASE`
- `DISCODE_APP_PORT`
- `PORT`
  - Override demo app port selection.

## Troubleshooting

- If `./ralph-loop` cannot talk to Codex, confirm `codex app-server` works in your shell or set `RALPH_LOOP_CODEX_COMMAND`.
- If `./wtl run` cannot talk to Codex, confirm `codex app-server` works in your shell or set `WTL_CODEX_COMMAND`.
- If `harnesscli boot start` reports a busy port, rerun with `DISCODE_APP_PORT` or stop the conflicting process.
- If automation output looks human-oriented in scripts, pass `--output json` explicitly even though non-TTY defaults should already select JSON.
