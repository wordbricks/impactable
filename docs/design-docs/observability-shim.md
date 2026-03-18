# Observability Shim

## Current State

This repository is at the start of the harness-spec observability work. The current implementation is a local observability shim, not the full Vector + Victoria stack described by later phases of `specs/harness-spec`.

## Data Flow

- `harnesscli boot start` writes process logs under `.worktree/<worktree_id>/logs/`.
- `harnesscli observability start` creates a per-worktree observability metadata file and declares the local query endpoints that will be used by future phases.
- `harnesscli observability query` currently supports local log-file queries and returns structured output.

## Query Contract

- Default output is JSON in non-TTY mode.
- `--output ndjson` is available for line-oriented query results.
- Query responses include:
  - `worktree_id`
  - `runtime_root`
  - `kind`
  - `items`

## Upgrade Path

Later harness phases should replace this shim with a real per-worktree telemetry stack:

- Vector for collection and fan-out
- VictoriaLogs for logs
- VictoriaMetrics for metrics
- VictoriaTraces for traces

When that happens, this document must be updated before merge.
