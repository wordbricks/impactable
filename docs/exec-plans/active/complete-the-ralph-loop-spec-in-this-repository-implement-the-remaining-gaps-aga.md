# Complete the Ralph Loop spec in this repository (implement remaining gaps)

## Goal
Make this CLI materially closer to Ralph Loop spec-complete by implementing the remaining behavior gaps in command execution, machine-readable output contracts, schema fidelity, and test coverage.

## Background
- The target specification is `specs/ralph-loop/SPEC.md`.
- Vendored guidance exists under `specs/ralph-loop/references/` and should inform implementation details and naming consistency.
- The current implementation already provides a base command surface in `cmd/ralph-loop/main.go` and `internal/ralphloop/*`, but it still needs stronger parity with spec requirements for command behavior and structured outputs.

## Milestones
| ID | Milestone | Status | Exit criteria |
| --- | --- | --- | --- |
| M1 | Gap assessment and contract baseline | not started | Document current vs required behavior for `init`, main run, `ls`, `tail`, and `schema`; identify exact missing fields, modes, and option behavior from the spec and references. |
| M2 | Command parsing and schema parity | not started | Ensure live schema descriptors and parser behavior fully align with required options, defaults, enums, payload schema, and command-level mutability/dry-run metadata. |
| M3 | Structured output contract hardening | not started | Enforce `text/json/ndjson` behavior across commands, keep structured errors in machine-readable modes, and enforce safe `--output-file` sandboxing to current working directory. |
| M4 | Core command behavior completion | not started | Implement remaining runtime behavior gaps in `init`, main loop lifecycle events, `ls`, and `tail` (including pagination/field-mask semantics and NDJSON streaming expectations). |
| M5 | Tests and regression coverage | not started | Add/extend tests for parser, schema output, output modes, command envelopes, and failure paths so the spec-critical contracts are protected. |
| M6 | Final verification and handoff | not started | Run full test suite, summarize completed spec deltas, and produce a concise handoff with known limitations and next actions. |

## Current progress
- Overall status: not started.
- M1: not started.
- M2: not started.
- M3: not started.
- M4: not started.
- M5: not started.
- M6: not started.

## Key decisions
- Use `specs/ralph-loop/SPEC.md` as the normative behavior contract.
- Use `specs/ralph-loop/references/internal/ralphloop/*.go` as implementation guidance, but prefer local repository constraints when conflicts appear.
- Prioritize output contract correctness and test-backed behavior before cosmetic text-mode refinements.
- Keep changes incremental and milestone-scoped to preserve reviewability and bisectability.

## Remaining issues
- Detailed gap list is pending milestone M1.
- Unknown edge-case failures in existing runtime/session orchestration are pending implementation analysis.
- Some behaviors may require explicit tradeoffs if current repository architecture diverges from the upstream reference.

## Links
- Spec: `specs/ralph-loop/SPEC.md`
- Upstream metadata: `specs/ralph-loop/UPSTREAM.md`
- Vendored reference CLI entrypoint: `specs/ralph-loop/references/cmd/ralph-loop/main.go`
- Vendored reference internals: `specs/ralph-loop/references/internal/ralphloop/`
- Local implementation entrypoint: `cmd/ralph-loop/main.go`
- Local implementation internals: `internal/ralphloop/`
