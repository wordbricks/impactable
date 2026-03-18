# Plans

Execution plans are first-class repository artifacts.

## Locations

- `docs/exec-plans/active/`
  - In-flight work with current milestone status.
- `docs/exec-plans/completed/`
  - Archived plans that reflect what actually shipped.
- `docs/exec-plans/tech-debt-tracker.md`
  - Known debt, missing invariants, and follow-up cleanup items.

## When To Create A Plan

- Create a checked-in plan for multi-step work, cross-file refactors, harness changes, and anything likely to span more than one coding turn.
- Small one-file fixes can stay lightweight, but the decision and rationale should still be discoverable from code and commit history.

## Minimum Plan Shape

- Goal
- Background
- Milestones
- Current progress
- Key decisions
- Remaining issues
- Links

## Freshness

- Update plan status as milestones move.
- Move completed plans into `docs/exec-plans/completed/` instead of deleting them.
