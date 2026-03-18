# Implement Recurring Cleanup Process

Build a recurring, automated cleanup system that encodes golden principles into the repository and continuously enforces them. This functions like garbage collection for technical debt — human taste is captured once, then enforced continuously on every line of code.

The goal: on a regular cadence, background tasks scan for deviations from golden principles, update quality grades, and open targeted refactoring pull requests. Most of these PRs should be reviewable in under a minute and safe to automerge.

This phase is **not** the per-commit merge gate. Unlike Phase 4, the principles here do not need to be fully satisfied on every commit before merge. Instead, define a broader set of golden principles that are checked exhaustively on a recurring schedule (daily by default) through `harnesscli cleanup ...` commands. Use this phase for repository-wide hygiene, drift detection, grading, and small cleanup PR generation.

---

## Step 1: Define golden principles

Before building automation, codify the golden principles for this repository. These are opinionated, mechanical rules that keep the codebase legible and consistent for future agent runs.

Do not duplicate checks that are already owned by Phase 4. Phase 4 supplies the always-on merge-blocking enforcement for architectural and structural invariants through `harnesscli lint` and `harnesscli test`. Phase 5 should focus on recurring repository-wide hygiene, grading, and cleanup work that is valuable to run daily but is not required to block every commit before merge.

Explore the codebase and define principles in a machine-readable file (`golden-principles.yaml` or similar). Each principle should have:

- **id**: Short identifier (e.g., `prefer-shared-utils`, `no-inline-secrets`)
- **description**: What the principle enforces and why
- **detection**: How to find violations (grep pattern, AST rule, file structure check, etc.)
- **remediation**: What the fix looks like — specific enough for an agent to act on
- **severity**: `warn` or `error` — whether a violation blocks merge or just opens a cleanup PR
- **automerge**: Whether cleanup PRs for this principle are safe to automerge

Start with the following baseline principles and adapt to this project. The list should grow over time as new patterns are identified.

### Repository-wide hygiene and safety principles

```yaml
principles:
  - id: no-inline-secrets
    description: >
      Source files and docs must not contain real credentials or token-shaped secret values.
    detection_kind: secret-scan
    remediation: >
      Move the value into environment or config and keep examples obviously fake.
    severity: error
    automerge: false

  - id: test-coverage-for-new-code
    description: >
      New or modified modules must have corresponding test files;
      untested production code must not be merged.
    detection_kind: test-coverage
    remediation: >
      Add a test file covering the new or changed behavior.
    severity: error
    automerge: false
```

### Code quality principles (severity: warn)

```yaml
  - id: prefer-shared-utilities
    description: >
      Common operations (concurrency helpers, retry logic, date formatting,
      path manipulation) must use shared utilities rather than hand-rolled
      inline implementations. Keeps invariants centralized.
    detection_kind: duplicate-utility
    remediation: >
      Check the shared utility package for an existing helper. If none exists,
      add one there with tests rather than inlining a one-off implementation.
    severity: warn
    automerge: false

  - id: no-wildcard-re-exports
    description: >
      Modules must not use `export *` which obscures the public API and makes
      dependency tracing harder for agents.
    detection_kind: naming-convention
    remediation: >
      Replace `export *` with explicit named exports so the module boundary is legible.
    severity: warn
    automerge: true

  - id: no-dead-code
    description: >
      Remove unused exports, unreachable branches, and stale feature flags.
      Dead code misleads agents into thinking it's still relevant.
    detection_kind: dead-code
    remediation: >
      Delete the dead code. If it was a public API, verify no external consumers exist first.
    severity: warn
    automerge: true

  - id: consistent-error-handling
    description: >
      All errors must be handled explicitly. No swallowed catches, no ignored rejections.
      Error paths must log structured context.
    detection_kind: error-handling
    remediation: >
      Add structured error logging with relevant context. If the error is intentionally
      ignored, add an explicit comment explaining why.
    severity: warn
    automerge: true

  - id: no-todo-outside-tests
    description: >
      Production code and docs must not accumulate untracked TODO placeholders.
    detection_kind: todo-scan
    remediation: >
      Remove the placeholder or move the follow-up into
      docs/exec-plans/tech-debt-tracker.md with a concrete next step.
    severity: warn
    automerge: true

```

Adapt the detection kinds to the project's language and tooling. Favor recurring checks that are repository-wide and non-blocking for day-to-day iteration. For example, `todo-scan`, `secret-scan`, `duplicate-utility`, `test-coverage`, `dead-code`, and `error-handling` can be implemented as static scans, AST analysis, or heuristic grep patterns. Do not re-register detection kinds that are already enforced as Phase 4 pre-merge invariants.

---

## Step 2: Build the scanner

Implement the `harnesscli cleanup scan` subcommand that:

1. Reads the golden principles file
2. For each principle, runs the detection logic against the codebase
3. Outputs a structured report of all violations found

Support `--output json|ndjson|text`, defaulting to JSON in non-TTY contexts. `--output ndjson` should emit one JSON object per violation plus a terminal summary object so large scans can be streamed safely. This command should be callable manually and from scheduled automation; it is not the default pre-merge gate.

The report format should be JSON:

```json
{
  "timestamp": "2025-01-15T04:00:00Z",
  "violations": [
    {
      "principle_id": "prefer-shared-utils",
      "file": "src/billing/utils/formatCurrency.ts",
      "line": 12,
      "description": "Local formatCurrency duplicates shared/utils/currency.ts",
      "severity": "warn",
      "remediation": "Replace with import from @shared/utils/currency"
    }
  ],
  "summary": {
    "total": 5,
    "by_severity": { "warn": 4, "error": 1 },
    "by_principle": { "prefer-shared-utils": 2, "no-dead-code": 3 }
  }
}
```

The scanner should be fast — it runs frequently. Prefer static analysis (grep, AST parsing, import graph analysis) over runtime checks.

---

## Step 3: Build the quality grader

Implement the `harnesscli cleanup grade` subcommand that:

1. Runs the scanner
2. Computes a quality grade for the codebase based on violation counts and severities
3. Writes the grade to a trackable file (e.g., `docs/generated/quality-grade.json`)

The command's stdout should also support the shared structured output contract so agents can consume the computed grade without reading files from disk.

The grade file should include:

```json
{
  "grade": "B+",
  "score": 87,
  "timestamp": "2025-01-15T04:00:00Z",
  "trend": "improving",
  "breakdown": {
    "prefer-shared-utils": { "violations": 2, "max_score": 15, "score": 11 },
    "no-inline-secrets": { "violations": 0, "max_score": 25, "score": 25 },
    "no-dead-code": { "violations": 3, "max_score": 20, "score": 14 }
  },
  "previous": {
    "grade": "B",
    "score": 83,
    "timestamp": "2025-01-14T04:00:00Z"
  }
}
```

The grading formula should be configurable. Principles with `severity: error` should weigh more heavily than `warn`.

---

## Step 4: Build the cleanup PR generator

Implement the `harnesscli cleanup fix` subcommand that:

1. Runs the scanner to find violations
2. Groups violations by principle
3. For each group, creates a focused branch and applies the fix
4. Opens a pull request with:
   - Title: `cleanup(<principle_id>): <short description>`
   - Body: which principle was violated, what files were changed, and why
   - Label: `cleanup`, `automerge` (if the principle allows it)

Each PR should be small and focused — one principle, one logical group of fixes. The goal is PRs reviewable in under a minute.

The fix logic per principle can be:
- **Automated**: The command applies the fix directly (e.g., deleting dead code, replacing a local helper with a shared import)
- **Agent-assisted**: The command creates the branch with a description of what needs to change, and a coding agent completes the fix

Start with automated fixes for simple principles (dead code removal, import replacement) and agent-assisted for complex ones (boundary validation refactoring).
Support `--output json|ndjson|text`, defaulting to JSON in non-TTY contexts. `--output ndjson` should stream one event per branch, PR, or fix attempt so agents can follow long-running cleanup work incrementally.

---

## Step 5: Set up the recurring schedule

### GitHub Actions workflow

Create `.github/workflows/recurring-cleanup.yml`:

```yaml
name: Recurring Cleanup

on:
  schedule:
    - cron: "0 4 * * *"  # Daily at 4am UTC
  workflow_dispatch:

jobs:
  scan-and-grade:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v4

      # Add language/runtime setup steps needed for this repository.

      - name: Build harness CLI
        run: cargo build --release --manifest-path harness/Cargo.toml

      - name: Run scanner
        run: harness/target/release/harnesscli cleanup scan > scan-report.json

      - name: Update quality grade
        run: harness/target/release/harnesscli cleanup grade

      - name: Commit grade update
        run: |
          git config user.name "cleanup-bot"
          git config user.email "cleanup-bot@noreply"
          git add docs/generated/quality-grade.json
          git diff --cached --quiet || git commit -m "chore: update quality grade"
          git push

  open-cleanup-prs:
    needs: scan-and-grade
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v4

      # Add language/runtime setup steps needed for this repository.

      - name: Build harness CLI
        run: cargo build --release --manifest-path harness/Cargo.toml

      - name: Generate cleanup PRs
        run: harness/target/release/harnesscli cleanup fix
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

Customize the cron schedule for this project's cadence. Daily is a good default — frequent enough to catch drift early, not so frequent it creates noise. This scheduled workflow is the primary enforcement path for Phase 5.

---

## Step 6: Integrate with existing harness

- Expose the recurring cleanup flow through `harnesscli cleanup {scan,grade,fix}` so it can be run manually by humans and agents
- Keep the primary enforcement path in the scheduled workflow rather than the per-commit merge gate
- The quality grade should be checked in the recurring workflow — if the grade drops below a configurable threshold, the workflow should warn (or fail) and surface the regression
- Add `make scan` and `make grade` targets to `Makefile.harness` for on-demand local runs:

```makefile
scan: harness-build
	@$(HARNESS) cleanup scan

grade: harness-build
	@$(HARNESS) cleanup grade
```

---

## Step 7: Verify

1. Run `harnesscli cleanup scan` and confirm it produces a valid violation report
2. Run `harnesscli cleanup grade` and confirm it produces a quality grade
3. Intentionally introduce a violation and confirm the scanner catches it
4. Run `harnesscli cleanup fix` on a test branch and confirm it opens a well-formed PR
5. Confirm the scheduled workflow (or an equivalent manual invocation) surfaces severe violations and generates the expected grade/cleanup output without being wired as a required per-commit merge gate

---

## Deliverables

- [ ] `golden-principles.yaml` — machine-readable principle definitions
- [ ] `harnesscli cleanup scan` — scans for violations, outputs JSON report
- [ ] `harnesscli cleanup grade` — computes and writes quality grade
- [ ] `harnesscli cleanup fix` — generates focused cleanup PRs
- [ ] `.github/workflows/recurring-cleanup.yml` — daily scheduled workflow
- [ ] `make scan` and `make grade` targets in `Makefile.harness`
- [ ] Daily scheduled workflow is the primary enforcement path; cleanup commands are available for on-demand runs via `harnesscli`
- [ ] Quality grade tracked in `docs/generated/quality-grade.json`

## Key principle

Technical debt is a high-interest loan. Pay it down continuously in small increments. Human taste is captured once in golden principles, then enforced continuously on every line of code. Catch bad patterns daily, not weekly.
