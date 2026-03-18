# Technical Debt Tracker

## Open Items

- The repository has a bootstrap `harnesscli`, but later harness-spec phases still need full invariant enforcement and a richer observability stack.
- `ralph-loop` exists as a Go implementation and should be brought closer to the imported upstream reference over time.
- The long-term Git impact analyzer product surface in `SPEC.md` is mostly specified but not implemented yet.

## Usage

- Add debt items with clear remediation notes.
- Link active execution plans or PRs when a cleanup effort starts.
- Move resolved items into commit history rather than silently deleting context.
