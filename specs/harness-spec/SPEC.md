# create-harness Spec

`create-harness/` is a portable blueprint for bootstrapping a harness engineering system in any repository. It provides phase-by-phase instructions that an agent (or human) follows sequentially to install documentation structure, execution environments, observability, invariant enforcement, automated cleanup, and a final audit around a separately specified Ralph Loop.

## Goal

Turn any repository into a fully harnessed, agent-operable codebase where:

- Documentation is structured and navigable, not monolithic.
- The app boots deterministically per Git worktree with isolated resources.
- Logs, metrics, and traces are queryable locally without external infrastructure.
- Architectural boundaries are enforced mechanically, not by convention alone.
- Technical debt is detected, graded, and cleaned up automatically.
- An autonomous agent loop (Ralph Loop) is already available and can drive a task from prompt to pull request.
- A single audit command verifies the entire harness is wired and passing.

## Prerequisite

Apply the standalone Ralph Loop spec at [`https://github.com/siisee11/ralph-loop.spec/blob/main/SPEC.md`](https://github.com/siisee11/ralph-loop.spec/blob/main/SPEC.md) before treating the create-harness flow as complete. The create-harness documents assume the repository already has a working `./ralph-loop` command and then build the surrounding harness system around that capability.

## Phases

| Phase | File | Description |
|-------|------|-------------|
| 1 | [`1_harness_structure.md`](./1_harness_structure.md) | Repository documentation structure |
| 2 | [`2_execution-env-setup.md`](./2_execution-env-setup.md) | Worktree-aware execution environment |
| 3 | [`3_observability-stack-setup.md`](./3_observability-stack-setup.md) | Per-worktree observability stack |
| 4 | [`4_enforce-invariants.md`](./4_enforce-invariants.md) | Custom linters and structural tests |
| 5 | [`5_recurring-cleanup.md`](./5_recurring-cleanup.md) | Automated tech debt cleanup |
| 6 | [`6_ralph-loop.md`](./6_ralph-loop.md) | Ralph Loop prerequisite handoff |
| 7 | [`7_implement-harness-audit.md`](./7_implement-harness-audit.md) | End-to-end harness audit |

Apply the Ralph Loop prerequisite before closing out the create-harness sequence. Each phase document contains self-contained instructions for the create-harness portion of the system.

## Checklist

The [`harness-scaffolding-checklist.md`](./harness-scaffolding-checklist.md) tracks completion status across all phases with per-item checkboxes.

## Key Constraints

- **Single Rust harness system of record.** Harness behavior should live in `harnesscli` subcommands. Do not require shell wrapper entrypoints for harness operations when the Rust CLI can serve as the stable interface directly.
- **Every command has a test.** Tests live as `#[cfg(test)]` modules or under `harness/tests/`.
- **Machine-readable output is mandatory.** Every `harnesscli` command must support structured output. Default to JSON in non-TTY contexts, use NDJSON for streaming or paginated responses, and return structured JSON errors on failure.
- **Codebase modularity is enforced mechanically.** Production code must live inside declared domains, layers, or modules with explicit boundaries; `harnesscli` lint/test checks should fail when code bypasses that structure.
- **Worktree isolation.** All runtime resources (ports, temp dirs, data dirs, logs) are derived from a deterministic worktree ID.
- **No blind sleeps.** Readiness is healthcheck-based.
- **Ralph Loop is externalized.** The autonomous coding loop is specified in [`https://github.com/siisee11/ralph-loop.spec/blob/main/SPEC.md`](https://github.com/siisee11/ralph-loop.spec/blob/main/SPEC.md); create-harness integrates around it rather than redefining it inline.
- **Portable.** This directory contains only instructions, templates, and reference artifacts. Any generated harness code still lives in the target repository.

## Directory Structure

```
create-harness/
├── SPEC.md                          # this file
├── harness-scaffolding-checklist.md # phase completion tracker
├── 1_harness_structure.md           # Phase 1 instructions
├── 2_execution-env-setup.md         # Phase 2 instructions
├── 3_observability-stack-setup.md   # Phase 3 instructions
├── 4_enforce-invariants.md          # Phase 4 instructions
├── 5_recurring-cleanup.md           # Phase 5 instructions
├── 6_ralph-loop.md                  # Phase 6 prerequisite handoff
├── 7_implement-harness-audit.md     # Phase 7 instructions
├── references/                      # LLM-friendly docs and reference implementations
└── templates/                       # Template files (e.g. NON_NEGOTIABLE_RULES.md)
```
