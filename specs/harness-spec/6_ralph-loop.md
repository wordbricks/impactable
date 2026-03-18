# Ralph Loop Prerequisite

The Ralph Loop specification now lives at [`https://github.com/siisee11/ralph-loop.spec/blob/main/SPEC.md`](https://github.com/siisee11/ralph-loop.spec/blob/main/SPEC.md).

Treat Ralph Loop as a prerequisite to the create-harness flow. Before considering this checkpoint satisfied, confirm the target repository already has:

- A stable repo-root `./ralph-loop` entrypoint
- Setup, coding-loop, and PR-agent orchestration implemented from the standalone Ralph Loop spec
- Integration with `harnesscli init` and `docs/exec-plans/`
- Verification that prompt -> plan -> iterations -> commits -> PR works end to end

Once that prerequisite is in place, continue using the create-harness documents to wire the remaining harness structure, observability, invariant enforcement, cleanup, and audit flows around it.
