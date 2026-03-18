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
| M1 | Gap assessment and contract baseline | done (2026-03-18) | Document current vs required behavior for `init`, main run, `ls`, `tail`, and `schema`; identify exact missing fields, modes, and option behavior from the spec and references. |
| M2 | Command parsing and schema parity | done (2026-03-18) | Ensure live schema descriptors and parser behavior fully align with required options, defaults, enums, payload schema, and command-level mutability/dry-run metadata. |
| M3 | Structured output contract hardening | done (2026-03-18) | Enforce `text/json/ndjson` behavior across commands, keep structured errors in machine-readable modes, and enforce safe `--output-file` sandboxing to current working directory. |
| M4 | Core command behavior completion | done (2026-03-18) | Implement remaining runtime behavior gaps in `init`, main loop lifecycle events, `ls`, and `tail` (including pagination/field-mask semantics and NDJSON streaming expectations). |
| M5 | Tests and regression coverage | done (2026-03-18) | Add/extend tests for parser, schema output, output modes, command envelopes, and failure paths so the spec-critical contracts are protected. |
| M6 | Final verification and handoff | done (2026-03-18) | Run full test suite, summarize completed spec deltas, and produce a concise handoff with known limitations and next actions. |

## Current progress
- Overall status: complete (M1-M6 done).
- M1: completed baseline audit of command contracts against `SPEC.md` and vendored references.
- M2: completed parser/schema parity pass including stricter option validation and schema payload contract cleanup.
- M3: completed machine-readable error envelope handling and hardened output-file sandbox validation.
- M4: completed core runtime behavior pass for `init`, `main`, `ls`, and `tail`.
- M5: completed regression coverage expansion for schema/read/dry-run output contracts and envelope behavior.
- M6: completed final verification + handoff summary.

## M2 completion summary (2026-03-18)
- Enforced parser strictness:
  - unknown options now fail fast instead of being treated as positionals,
  - command-specific option compatibility is validated from live command descriptors (`commandOptionSet`),
  - subcommands now enforce positional cardinality (`init` none, `ls`/`tail`/`schema` max one),
  - positive integer validation added for `--page`, `--page-size`, `--lines`, `--max-iterations`, and `--timeout`,
  - `--output` values are validated against `text|json|ndjson`.
- Added alias parity:
  - tail aliases `-n` (for `--lines`) and `-f` (for `--follow`) are supported in parser and exposed in schema descriptors.
- Fixed schema payload contract collision:
  - schema command now uses `target_command` in JSON payload instead of overloading `command`,
  - preserved backward compatibility for legacy `command_name`.
- Improved schema descriptor fidelity:
  - dynamic `--output` default is represented as `text (tty) / json (non-tty)` in descriptors,
  - schema raw payload now clearly includes `command: "schema"` discriminator plus `target_command`.
- Added targeted tests for:
  - parser alias handling,
  - unknown/unsupported option rejection,
  - output enum validation,
  - schema target-command payload parsing/validation,
  - descriptor alias and option-set behavior.

## M3 completion summary (2026-03-18)
- Added centralized failure emission in `Run(...)`:
  - `json` and `ndjson` modes now emit structured failure payloads on parse errors, repo resolution errors, and command execution errors,
  - payload shape is stable: `{"command":"<name>","status":"failed","error":{"code":"command_failed","message":"..."}}`,
  - in machine-readable modes, stderr is left empty for ordinary command failures.
- Added output selection/hinting logic:
  - structured failures now respect command/output/output-file from parsed request where available,
  - parse-time failures use flag-derived hints (`--output`, `--output-file`) to keep failures machine-readable.
- Hardened `--output-file` path sandboxing:
  - path checks are now symlink-aware via canonicalization of existing path segments before containment checks against caller CWD,
  - symlink escapes out of CWD are rejected.
- Reduced mode leakage for `init`:
  - the `Preparing worktree` progress message now emits to stderr only in `text` mode.
- Added regression tests for:
  - JSON and NDJSON structured failures at parse time and runtime,
  - symlink escape rejection and in-root symlink allowance for output path resolution.

## M4 completion summary (2026-03-18)
- Completed `init` runtime contract updates:
  - runtime directories now include `.worktree/<id>/tmp/` in addition to `logs/` and `run/`,
  - `init` success output now includes `deps_installed` and `build_verified` booleans,
  - `.env.example` → `.env` bootstrapping is now performed when `.env` is absent,
  - `DISCODE_WORKTREE_ID` and `DISCODE_RUNTIME_ROOT` are set during preparation.
- Improved dry-run behavior for mutating commands:
  - `init --dry-run` now returns explicit `request` and `side_effects` blocks,
  - main command dry-run now returns an executable request echo plus side-effect plan.
- Implemented richer main-loop lifecycle event streaming in NDJSON:
  - emits `phase.started`, `phase.completed`, and `phase.failed` around setup/coding/pr phases,
  - emits `iteration.started`, `iteration.completed`, and `iteration.failed` in coding loop,
  - run completion now respects requested output format (`json`/`text` single object, `ndjson` terminal event stream),
  - NDJSON streaming honors `--output-file` by mirroring stream output to file and stdout.
- Updated read command runtime behavior:
  - `ls --output ndjson` now emits one session object per line,
  - `ls --page-all --output ndjson` emits one page envelope per line with consistent pagination metadata,
  - `tail --output json` now includes log metadata (`log_path`, selector, match count, pagination totals),
  - `tail --follow` now emits structured NDJSON event records (`command`, `event`, `status`, `ts`, `log_path`, line payload).
- Pagination metadata normalization:
  - read envelopes now use `total_items` and `total_pages`,
  - empty datasets no longer synthesize phantom `{}` items.

## M5 completion summary (2026-03-18)
- Added output-contract regression suite in `internal/ralphloop/output_contract_test.go`.
- New coverage includes:
  - `schema --output json` envelope shape + command filtering (`target_command`),
  - `ls --output ndjson` emits one session record per line (non-envelope mode),
  - `ls --page-all --output ndjson` emits paged envelopes with stable pagination fields,
  - `tail --output json` includes required metadata envelope fields (`log_path`, totals, command),
  - `init --dry-run --output json` includes `request` + `side_effects`,
  - main `--dry-run --output json` includes `result` + `request` + `side_effects`.
- Existing M2/M3 suites continue covering:
  - parser strictness and schema parity (`parser_test.go`, `schema_test.go`),
  - machine-readable failure envelopes and output-path hardening (`run_test.go`, `util_test.go`).
- Verified with `go test ./...` after the new suite.

## M6 final verification + handoff (2026-03-18)
- Verification commands executed:
  - `go test ./...` (pass),
  - `./ralph-loop schema --output json` (pass; schema envelope emitted),
  - `./ralph-loop init --dry-run --output json` (pass; includes `request` + `side_effects`),
  - `./ralph-loop ls --output ndjson` (pass; per-session NDJSON record),
  - `./ralph-loop tail --lines 1 --output json` (pass; metadata + paginated items envelope).
- Completed spec-delta summary:
  - parser/schema contract parity and option/alias validation implemented,
  - machine-readable failure behavior and output-file sandboxing hardened,
  - core runtime behavior advanced for `init`, main lifecycle NDJSON events, `ls`, and `tail`,
  - regression coverage expanded around command/output envelopes and dry-run contracts.
- Handoff outcome:
  - CLI is materially closer to spec-complete for command behavior and machine-readable contracts.
  - Remaining items are now mostly edge-case/spec-interpretation and advanced orchestration behaviors.

## M1 contract baseline (2026-03-18)
### Cross-command contract deltas
| Area | Spec requirement | Current behavior | Gap to close |
| --- | --- | --- | --- |
| Output mode validation | `--output` must be `text|json|ndjson` | Arbitrary strings accepted; unknown format falls back to JSON marshaling path | Validate enum and return structured machine-readable errors |
| Structured machine-readable errors | `json`/`ndjson` must keep structured error envelopes | Errors are plain stderr strings for all modes | Centralize error envelope emission by mode |
| `--output-file` sandboxing | Constrain to caller CWD after symlink resolution | Uses `filepath.Abs`/`Rel` only; no symlink resolution on target path | Resolve symlinks and reject escapes after canonicalization |
| Input hardening | Reject control chars/path traversal/encoded separators/query fragments | No validation for selectors, branch names, output-file paths, or JSON fields | Add shared identifier/path sanitizers and parser enforcement |
| Flag parsing strictness | Agent-first CLI should reject unknown options | Unknown flags become positionals silently | Add per-command unknown-flag rejection and positional cardinality checks |

### `init` command deltas
| Area | Spec requirement | Current behavior | Gap to close |
| --- | --- | --- | --- |
| Success JSON contract | Include `deps_installed` and `build_verified` booleans plus worktree metadata | Returns worktree metadata and `project`, but omits booleans | Add required booleans and keep stable top-level fields |
| Runtime dirs | Ensure `.worktree/<id>/logs/` and `.worktree/<id>/tmp/` | Creates `logs/` and `run/`; no `tmp/` | Add `tmp/` creation |
| Env initialization | Copy `.env.example` to `.env` when needed and set worktree-derived env vars | Not implemented | Implement idempotent env setup |
| Clean git state | Ensure clean state and stash uncommitted changes in target worktree | Stash only when already in linked worktree path; reused external worktree path not fully normalized to spec flow | Normalize clean-state flow for both reused and newly-created worktrees |
| `--dry-run` shape | Return exact request + planned side effects list | Returns worktree/project preview only | Add explicit side-effects list and executable request payload echo |

### Main run command deltas
| Area | Spec requirement | Current behavior | Gap to close |
| --- | --- | --- | --- |
| Output mode contract | Respect `text/json/ndjson` and produce one JSON object for `json` mode | Final emit is forced to JSON regardless of requested output | Respect requested mode for final terminal output |
| NDJSON lifecycle richness | Stream lifecycle events and terminal event with stable fields (`phase`, `iteration`, etc.) | Emits limited events (`run.started`, `iteration.started`, `run.completed`) | Add missing phase/iteration completion/failure events with stable envelope fields |
| Structured failure result | Return command-level structured failure object | Returns Go errors to stderr without JSON/NDJSON envelope | Emit `status:"failed"` + `error{code,message}` in machine-readable modes |
| Loop safeguards | Context overflow compaction and failure retry semantics | Basic retry prompt on failed turn only; no compaction handling | Implement spec-aligned failure/compaction branches |

### `ls` command deltas
| Area | Spec requirement | Current behavior | Gap to close |
| --- | --- | --- | --- |
| NDJSON mode | Emit one session record per line | Emits one paginated envelope line containing `items` array | Emit per-session NDJSON records (or page envelopes only for `--page-all`, per selected contract) |
| Schema/selector fidelity | Selector and output contract must be machine-friendly and validated | Selector accepted with no hardening; empty relative path serialized as empty string | Validate selector input and normalize empty optional fields |
| Read contract consistency | Read commands should have consistent pagination metadata behavior | Uses page metadata envelope; aligns partially but differs from `ls` prose requirement ("JSON array") | Decide canonical `ls` contract and align schema/tests/docs consistently |

### `tail` command deltas
| Area | Spec requirement | Current behavior | Gap to close |
| --- | --- | --- | --- |
| JSON response shape | Include selected log metadata + line/record array | Returns paginated items without explicit log metadata fields | Add log metadata fields (`log_path`, selector, mode) to envelope |
| `follow` NDJSON events | Structured per-line/event records with stable command fields | Streams raw `logRecord` objects only (`line/rendered/raw/line_number`) | Wrap follow records in command/event envelopes |
| Pagination semantics | Support page/page-size/page-all consistently for read commands | Non-follow path paginates records; follow path ignores pagination controls | Validate/define follow-specific pagination behavior and enforce explicitly |

### `schema` command deltas
| Area | Spec requirement | Current behavior | Gap to close |
| --- | --- | --- | --- |
| Command schema fields | Include command name, description, positionals, options/aliases, required, defaults, enum, nested payload schema, mutability, dry-run | Includes most fields but no aliases, partial required metadata, and inconsistent defaults (e.g. `--output` default is `"text|json"`) | Add alias metadata, required flags, and correct default semantics |
| Runtime parser parity | Schema must be generated from same live descriptors/parser semantics | Descriptors exist, but parser allows unknown flags and schema does not reflect permissive behavior | Tighten parser and keep schema descriptor-source authoritative |
| `schema` payload shape | Must describe schema command and allow target command selection | `schemaRequest` uses `command` field as target selector, colliding with top-level command discriminator in raw JSON | Split selector field (e.g. `target_command`) from command discriminator and update parser/schema |
| Read options behavior | `schema` supports `--fields`, pagination, and `--page-all` | `runSchema` ignores `--fields`, `--page`, `--page-size`, `--page-all` | Implement read-option behavior in schema output path |

## Key decisions
- Use `specs/ralph-loop/SPEC.md` as the normative behavior contract.
- Use `specs/ralph-loop/references/internal/ralphloop/*.go` as implementation guidance, but prefer local repository constraints when conflicts appear.
- Prioritize output contract correctness and test-backed behavior before cosmetic text-mode refinements.
- Keep changes incremental and milestone-scoped to preserve reviewability and bisectability.
- Treat command schema descriptors and parser behavior as a single source of truth in M2 so schema and runtime cannot drift.
- Resolve the `ls` JSON contract ambiguity in favor of one canonical envelope style, then lock it with tests and docs in M3/M5.
- Keep `schema` raw JSON backward compatibility via `command_name` while standardizing on `target_command`.
- Use a single top-level failure envelope for machine-readable modes across commands to avoid mode-specific divergence.
- Canonicalized `ls` behavior as: NDJSON emits per-session records by default, and page envelopes only for `--page-all`.
- Keep high-value contract coverage in unit tests that do not require spawning Codex app-server, reserving end-to-end wiring validation for M6/manual verification.

## Remaining issues
- The baseline exposed a spec tension for `ls` JSON shape (array vs paginated envelope) that must be resolved before implementation lock-in.
- `ls --output json` remains envelope-based rather than raw array to preserve cross-read-command pagination consistency; finalize this contract in M5 tests/docs.
- Coding loop context compaction (`ContextWindowExceeded` → `thread/compact/start`) is still pending and should be covered in M5/M6 follow-up.
- Unknown edge-case failures in existing runtime/session orchestration are pending implementation analysis.
- Some behaviors may require explicit tradeoffs if current repository architecture diverges from the upstream reference.

## Links
- Spec: `specs/ralph-loop/SPEC.md`
- Upstream metadata: `specs/ralph-loop/UPSTREAM.md`
- Vendored reference CLI entrypoint: `specs/ralph-loop/references/cmd/ralph-loop/main.go`
- Vendored reference internals: `specs/ralph-loop/references/internal/ralphloop/`
- Local implementation entrypoint: `cmd/ralph-loop/main.go`
- Local implementation internals: `internal/ralphloop/`
