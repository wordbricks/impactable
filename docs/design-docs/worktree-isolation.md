# Worktree Isolation

## Goal

Every worktree gets its own stable runtime identity and local resources.

## Worktree ID

- Resolve the canonical repo root for the current working tree.
- Use the basename of that path as the human-readable prefix.
- Append a stable short hash of the canonical path.
- Allow `DISCODE_WORKTREE_ID` to override the derived value.

Example:

`impactable-a1b2c3d4`

## Runtime Root

The harness uses:

`.worktree/<worktree_id>/`

Subdirectories currently reserved:

- `run/`
- `logs/`
- `tmp/`
- `demo-app/`
- `observability/`

## Port Allocation

- Default app port is derived from the worktree hash.
- Explicit overrides win in this order:
  - `DISCODE_APP_PORT`
  - `APP_PORT`
  - `PORT`
- When a derived port is already in use, the harness probes the next deterministic candidate in a bounded range.

## Lifecycle

- `harnesscli init` creates the runtime root and metadata.
- `harnesscli boot start` starts a deterministic local demo app under the current worktree runtime root.
- `harnesscli boot status` reports the stored metadata and live health.
- `harnesscli boot stop` terminates the managed process and removes stale lock metadata while preserving logs.
