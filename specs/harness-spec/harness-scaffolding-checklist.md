# Harness Scaffolding Checklist

Apply the following phases in order to scaffold a complete harness engineering system for this repository.

---

## Phase 1: Repository Documentation Structure

Apply the instructions in [`1_harness_structure.md`](./1_harness_structure.md).

This phase sets up the documentation hierarchy:

- [ ] `AGENTS.md` — compact table-of-contents entrypoint (~100 lines, navigation only)
- [ ] `ARCHITECTURE.md` — top-level map of domains, boundaries, dependencies, entrypoints
- [ ] `NON_NEGOTIABLE_RULES.md` — absolute rules that block merge unconditionally (use `create-harness/templates/NON_NEGOTIABLE_RULES.md` as template)
- [ ] `docs/PLANS.md`
- [ ] `docs/design-docs/index.md`
- [ ] `docs/design-docs/core-beliefs.md` — product beliefs + agent-first operating principles (see `harness_structure.md`)
- [ ] `docs/design-docs/local-operations.md` — local commands, env vars, and troubleshooting for humans and agents
- [ ] `docs/design-docs/worktree-isolation.md` — worktree ID derivation, runtime roots, cleanup, and failure handling
- [ ] `docs/design-docs/observability-shim.md` — telemetry architecture and query contract
- [ ] `docs/exec-plans/active/`
- [ ] `docs/exec-plans/completed/`
- [ ] `docs/exec-plans/tech-debt-tracker.md`
- [ ] `docs/product-specs/index.md`
- [ ] `docs/product-specs/harness-demo-app.md` — deterministic demo surface used for browser validation
- [ ] `docs/references/` — copy contents from `create-harness/references/` as seed
- [ ] `docs/generated/`

Key rules:
- `AGENTS.md` is a navigation document, not a knowledge document. Move any substantive guidance into `docs/`.
- Real source of truth lives in `docs/` and top-level documents, not in `AGENTS.md`.
- Prefer many small, maintainable documents over one giant document.
- Documentation must reflect real code and real operating practices.
- **All harness behavior lives in a single Rust CLI** called `harnesscli`. Thin shell wrappers are allowed only as stable entrypoints that immediately delegate to `harnesscli` or another versioned harness executable.
- **Every command must have a corresponding test.** Tests live alongside the Rust source as `#[cfg(test)]` modules or in integration test files under `harness/tests/`.

---

## Phase 2: Execution Environment Setup

Apply the instructions in [`2_execution-env-setup.md`](./2_execution-env-setup.md).

This phase makes the app bootable per Git worktree for isolated development:

- [ ] Worktree-aware boot flow with derived worktree ID
- [ ] Isolated runtime resources per worktree (ports, temp dirs, logs, etc.)
- [ ] Single command to boot the app for the current worktree
- [ ] Launch contract returning metadata (`app_url`, `port`, `healthcheck_url`, `worktree_id`, `runtime_root`, and observability metadata when available)
- [ ] Healthcheck-based readiness (no blind sleeps)
- [ ] `harnesscli init` — idempotent environment initialization with JSON output contract
- [ ] `harnesscli boot {start,status,stop}` — machine-readable lifecycle commands with JSON/NDJSON output
- [ ] `agent-browser` skill installed for UI investigation
- [ ] Example reproducibility and validation flow

---

## Phase 3: Observability Stack

Apply the instructions in [`3_observability-stack-setup.md`](./3_observability-stack-setup.md).

This phase sets up ephemeral, per-worktree telemetry so the agent can query logs, metrics, and traces:

- [ ] Vector config template for telemetry collection and fan-out
- [ ] Victoria Logs — log storage with LogQL API
- [ ] Victoria Metrics — metrics storage with PromQL API
- [ ] Victoria Traces — trace storage with TraceQL API
- [ ] All ports and data dirs derived from worktree ID
- [ ] App instrumented with OpenTelemetry SDK (logs, metrics, traces to Vector)
- [ ] `harnesscli observability start` — starts the stack with health checks
- [ ] `harnesscli observability stop` — tears down the stack and cleans up
- [ ] `harnesscli observability query` — convenience wrapper for LogQL/PromQL/TraceQL queries
- [ ] Observability commands default to structured output in non-TTY contexts and support NDJSON for streaming queries
- [ ] Integrated with worktree app boot flow

---

## Phase 4: Enforce Invariants

Apply the instructions in [`4_enforce-invariants.md`](./4_enforce-invariants.md).

This phase enforces architectural boundaries and taste mechanically via custom linters and structural tests. These are the required local pre-merge checks that must pass through `harnesscli` before a change merges to `main`:

- [ ] Machine-readable architecture rules file (dependency directions, allowed edges)
- [ ] Declared domain/layer/module ownership rules for production code
- [ ] Dependency direction linter — verifies imports respect layer ordering
- [ ] Module boundary linter — verifies production code stays inside declared modules and crosses boundaries only through allowed entrypoints
- [ ] Boundary parsing linter — verifies external data is validated at boundaries
- [ ] Taste invariant linters (structured logging, naming conventions, file size limits)
- [ ] Linter implementation is modularized by concern; avoid one monolithic `shared` helper
- [ ] All lint error messages include clear remediation instructions for agents
- [ ] Structural tests for domain completeness, module ownership, and dependency graph validation
- [ ] Cross-cutting boundary tests (shared concerns only via Providers interface)
- [ ] Integrated into `make lint` and `make test` as required pre-merge checks

---

## Phase 5: Recurring Cleanup Process

Apply the instructions in [`5_recurring-cleanup.md`](./5_recurring-cleanup.md).

This phase encodes golden principles and builds automated garbage collection for technical debt. These checks run as a recurring full sweep, not as a required per-commit merge gate:

- [ ] `golden-principles.yaml` — machine-readable principle definitions with detection and remediation
- [ ] `harnesscli cleanup scan` — scans for violations, outputs JSON report
- [ ] `harnesscli cleanup grade` — computes and tracks quality grade
- [ ] `harnesscli cleanup fix` — generates focused, small cleanup PRs
- [ ] Cleanup commands support JSON/NDJSON output modes for large scans and long-running fix operations
- [ ] `.github/workflows/recurring-cleanup.yml` — daily scheduled scan, grade update, and PR generation
- [ ] `make scan` and `make grade` targets in `Makefile.harness`
- [ ] Daily scheduled workflow is the primary enforcement path for cleanup checks
- [ ] Quality grade tracked in `docs/generated/quality-grade.json`

---

## Phase 6: Ralph Loop Prerequisite

Apply the instructions in [`6_ralph-loop.md`](./6_ralph-loop.md).

This checkpoint confirms the standalone Ralph Loop spec has already been applied before the create-harness flow is considered complete:

- [ ] [`https://github.com/siisee11/ralph-loop.spec/blob/main/SPEC.md`](https://github.com/siisee11/ralph-loop.spec/blob/main/SPEC.md) has been reviewed and applied
- [ ] Repo-root `./ralph-loop` entrypoint is available in the target repository
- [ ] Ralph Loop setup, coding-loop, and PR-agent orchestration are implemented
- [ ] Ralph Loop integrates with `harnesscli init` and `docs/exec-plans/`
- [ ] End-to-end verification passes: prompt -> worktree -> plan -> iterations -> commits -> PR

---

## Phase 7: Harness Engineering Audit

Apply the instructions in [`7_implement-harness-audit.md`](./7_implement-harness-audit.md).

This is the final phase — it verifies everything from all prior phases is wired together and passing:

- [ ] `Makefile.harness` with smoke/test/lint/typecheck/check/ci targets
- [ ] `Makefile` includes `Makefile.harness`
- [ ] `harnesscli smoke` — fast sanity check
- [ ] `harnesscli test` — full test suite
- [ ] `harnesscli lint` — static analysis
- [ ] `harnesscli typecheck` — type checking
- [ ] `harnesscli init`, `harnesscli boot`, and `harnesscli observability` command groups implemented
- [ ] `harnesscli audit` — audits all files and directories exist
- [ ] Every `harnesscli` command supports `--output json|ndjson|text`, defaults to JSON in non-TTY contexts, and returns structured JSON errors
- [ ] `harness/Cargo.toml` — Rust crate for the `harnesscli` CLI
- [ ] `.github/workflows/harness.yml` — CI workflow running `make ci`
- [ ] `harnesscli` CLI builds successfully (`cargo build --release -p harness`)
- [ ] `harness audit .` passes
- [ ] `make ci` succeeds
