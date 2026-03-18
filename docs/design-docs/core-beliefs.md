# Core Beliefs

## Product Beliefs

- The repository should converge on a single operator surface for automation instead of accumulating fragile shell entrypoints.
- The long-term product is a Git impact analyzer, but the immediate engineering priority is reliable harnessing and agent operability.
- Worktree-local behavior is preferable to shared mutable state because it keeps agent runs isolated and reproducible.
- Structured command output matters more than pretty terminal output for automation paths.

## Agent-First Operating Principles

1. **Repository knowledge is the system of record.**
   If a decision matters, it must be encoded in code, markdown, schema, or a checked-in plan.

2. **What the agent cannot see does not exist.**
   Product intent, architecture constraints, and operating conventions need to be discoverable in-repo.

3. **Enforce boundaries centrally, allow autonomy locally.**
   Rules belong in command contracts, tests, and CI rather than informal expectation.

4. **Corrections are cheap, waiting is expensive.**
   Prefer short feedback loops, fast validation commands, and follow-up fixes over long-lived ambiguity.

5. **Prefer boring technology.**
   The current Go and Rust choices are intentional because both are stable, well-known, and easy to automate.

6. **Encode taste once, enforce continuously.**
   Naming, output contracts, and lifecycle behavior should be captured in code and docs so every run sees the same standard.

7. **Treat documentation as executable infrastructure.**
   Docs are part of the harness. If runtime behavior changes, the corresponding canonical doc must change too.
