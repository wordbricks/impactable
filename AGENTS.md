# Repository Map

Use this file as a table of contents only. Canonical guidance lives in the linked documents.

```
AGENTS.md
ARCHITECTURE.md
NON_NEGOTIABLE_RULES.md
docs/
├── design-docs/
│   ├── index.md
│   ├── core-beliefs.md
│   ├── local-operations.md
│   ├── observability-shim.md
│   └── worktree-isolation.md
├── exec-plans/
│   ├── active/
│   ├── completed/
│   └── tech-debt-tracker.md
├── generated/
├── product-specs/
│   ├── index.md
│   └── harness-demo-app.md
├── references/
│   └── codex-app-server-llm.txt
├── PLANS.md
```

## Start Here

- Rules that block merge: `NON_NEGOTIABLE_RULES.md`
- System map and package boundaries: `ARCHITECTURE.md`
- Design doc index and ownership: `docs/design-docs/index.md`
- Product docs: `docs/product-specs/index.md`
- Execution plan policy: `docs/PLANS.md`

## Runtime Surfaces

- Ralph Loop CLI: `./ralph-loop`
- Harness CLI: `harness/target/release/harnesscli`
- Harness Make targets: `Makefile.harness`

## Specs In Repo

- Current product spec: `SPEC.md`
- Ralph Loop spec import: `specs/ralph-loop/SPEC.md`
- Harness spec import: `specs/harness-spec/SPEC.md`

## Working Rules

- Keep this file short and navigational.
- Put substantive guidance in the linked docs, not here.
- Update the relevant canonical doc when code or operating practice changes.
