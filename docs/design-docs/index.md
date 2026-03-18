# Design Docs Index

| Document | Canonical Topic | Owner | Intended Audience | Update When |
| --- | --- | --- | --- | --- |
| [`core-beliefs.md`](./core-beliefs.md) | Product beliefs and agent-first operating principles | Repo maintainers | Humans and agents | Product direction or operating model changes |
| [`local-operations.md`](./local-operations.md) | Local command surface, env vars, troubleshooting | Repo maintainers | Humans and agents | Commands, env vars, or validation flows change |
| [`worktree-isolation.md`](./worktree-isolation.md) | Worktree ID derivation, runtime roots, and port allocation | Repo maintainers | Humans and agents | Boot/runtime behavior changes |
| [`observability-shim.md`](./observability-shim.md) | Current telemetry data flow and local query contract | Repo maintainers | Humans and agents | Observability data paths or query contract changes |

## Status

- These documents are the canonical operational layer for the repository.
- `AGENTS.md` should only point here, not duplicate this content.
