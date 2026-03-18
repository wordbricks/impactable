# Non-Negotiable Rules

These rules are absolute. Violations block merge.

## Rule 1: New Behavior Must Be Tested

- Every new command, module, and branch of behavior needs tests before merge.
- Changes to existing behavior must update tests in the same change.
- For this repository, that means both Go tests for `internal/ralphloop` and Rust tests for `harnesscli` when those surfaces change.

## Rule 2: Machine-Readable Automation Is Required

- Automation-facing commands must support structured output.
- JSON is the default for non-TTY execution.
- Errors in automation flows must be emitted as structured payloads, not prose-only failures.

## Rule 3: No Blind Sleeps For Readiness

- Any boot or lifecycle command must verify readiness with a real probe.
- If readiness cannot be proven, the command must fail non-zero and report the failing resource.

## Rule 4: Repository Knowledge Is Canonical

- Durable product, runtime, and architecture guidance belongs in versioned repo docs.
- `AGENTS.md` stays navigational; substantive guidance belongs in `ARCHITECTURE.md` or `docs/`.
