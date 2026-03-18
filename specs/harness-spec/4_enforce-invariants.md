# Enforce Invariants with Custom Linters and Structural Tests

Documentation alone doesn't keep a fully agent-generated codebase coherent. Enforce invariants mechanically — not by micromanaging implementations — so coding agents can ship fast without undermining the foundation.

The principle: enforce boundaries centrally, allow autonomy locally. Care deeply about boundaries, correctness, and reproducibility. Within those boundaries, allow significant freedom in how solutions are expressed.

This phase covers the **always-on, pre-merge enforcement layer**. The checks defined here must run through `harnesscli` locally on every branch before a change can merge to `main`. Put merge-blocking invariants here: boundary validation, dependency direction, cross-cutting boundary checks, codebase modularity rules, and other structural tests that protect the repository's shape continuously.

Because this phase is part of the local pre-merge workflow, the checks here must stay fast enough to run routinely on a developer machine. Prefer static analysis, bounded graph checks, and targeted structural validation over long-running end-to-end or full-runtime verification.

---

## Step 1: Understand the codebase architecture

Before writing any linters or tests, map the repository's actual structure:

- **Business domains**: What are the distinct domains in this codebase? (e.g., auth, billing, settings, etc.)
- **Layers within domains**: What layers exist? Identify the dependency direction. A typical layered model looks like: `Types → Config → Repo → Service → Runtime → UI`
- **Modules within domains**: What are the stable modules or bounded areas inside each domain? Which files belong to which module, and what is the allowed public surface for each one?
- **Cross-cutting concerns**: What shared concerns exist? (e.g., auth, connectors, telemetry, feature flags, utils). These should enter through a single explicit interface (e.g., a Providers layer).
- **Existing conventions**: What naming conventions, logging patterns, file organization rules, and type conventions are already in use?

Document the discovered architecture in `docs/ARCHITECTURE.md` if not already done.

---

## Step 2: Define the dependency rules

Define which dependency directions are allowed and which are disallowed. These rules become the source of truth for the pre-merge linters and structural tests in this phase.

For each business domain, specify:

- The ordered set of layers and the permitted dependency direction (forward only)
- The declared modules within that domain and which files/directories belong to each module
- The allowed public entrypoints for each module and which imports must stay internal
- Which cross-cutting modules can be imported and through what interface
- What is explicitly disallowed (e.g., UI importing directly from Repo, Service importing from Runtime)
- What structure is forbidden (e.g., uncategorized top-level feature code, catch-all `misc` modules, kitchen-sink files that mix multiple modules)

Create a machine-readable rules file (e.g., `architecture.json`, `.architecture.yaml`, or similar) that encodes:

```
{
  "layers": ["types", "config", "repo", "providers", "service", "runtime", "ui"],
  "modules": {
    "billing": {
      "roots": ["src/billing"],
      "submodules": ["invoices", "plans", "usage"],
      "publicEntrypoints": ["src/billing/index.ts"]
    }
  },
  "direction": "forward",
  "crossCutting": {
    "providers": ["auth", "connectors", "telemetry", "featureFlags"]
  },
  "disallowed": [
    { "from": "ui", "to": "repo" },
    { "from": "service", "to": "runtime" },
    { "from": "types", "to": "*" }
  ]
}
```

Adapt the schema to match this project's actual architecture. The format should be whatever is easiest to consume by the custom linters.

---

## Step 3: Build custom linters

Create custom lint rules that enforce the architectural invariants mechanically. Implement them as part of the `harnesscli lint` command, organized into separate modules under `harness/src/cmd/lint/` or `harness/src/linters/`.

### Modularize linter implementation

- Do not accumulate all linter logic in a single `shared` file.
- Split the implementation by concern: rules loading, file discovery, import resolution, scan passes, and reporting should live in separate modules.
- Keep any `shared` entrypoint as a thin compatibility barrel only when needed by existing callers.
- When adding a new linter or scan type, put the logic in a focused module instead of extending a monolithic helper.

This matters because custom linters tend to grow quickly. If all scanning logic lives in one file, agents will keep appending unrelated behavior until the linter itself becomes hard to change safely.

### Dependency direction linter

- Parse imports/requires in each file
- Determine which domain and layer each file belongs to (by file path convention)
- Verify that all imports respect the allowed dependency direction
- Flag any import that violates the rules

### Module boundary linter

- Verify that every production file belongs to a declared domain/layer/module from the architecture rules
- Verify that imports cross module boundaries only through declared public entrypoints
- Flag catch-all modules, uncategorized production directories, and files that mix responsibilities from multiple modules without an explicit boundary
- Fail when new code is added outside the declared modular structure unless the architecture rules are updated in the same change

### Boundary parsing linter

- Verify that external data is parsed and validated at boundaries (e.g., API handlers, external integrations)
- The linter should check that boundary modules use validation (the specific library doesn't matter — Zod, joi, typebox, pydantic, serde, etc. are all fine)
- Flag boundary modules that pass raw unvalidated data to internal layers

### Taste invariants

Implement linters for project-specific taste rules. Examples:

- **Structured logging**: All log calls must use structured format (key-value pairs), not string interpolation
- **Naming conventions**: Schema types follow a consistent naming pattern (e.g., `*Schema`, `*Input`, `*Output`)
- **File size limits**: No single file exceeds a configurable line count threshold
- **No cross-layer shortcuts**: No file imports from a layer it shouldn't reach
- **No dumping-ground modules**: Avoid `misc`, `common`, or `helpers` buckets that accumulate unrelated business logic without a declared boundary

### Error messages as remediation instructions

This is critical: every lint error message should include **clear remediation instructions**. When a coding agent hits a lint failure, the error message becomes part of its context. Write error messages that tell the agent exactly what to do.

Bad:
```
Error: illegal import detected
```

Good:
```
Error: ui/settings/SettingsPanel.ts imports from repo/settings/settingsRepo.ts
  Rule: UI layer cannot import directly from Repo layer.
  Fix: Move this data access through the Service layer.
       Import from service/settings/settingsService.ts instead,
       or create a service method that wraps this repo call.
```

---

## Step 4: Build structural tests

Structural tests verify the codebase's shape at test time. Place them alongside other tests or in a dedicated `tests/structural/` directory.

### Domain completeness test

- For each business domain, verify that expected layers exist (e.g., every domain has a `types/` and a `service/` directory)
- Flag domains that are missing expected structure

### Module ownership test

- Verify that every production source file is owned by a declared domain/layer/module
- Flag source files that sit outside the declared modular structure
- Flag domains that accumulate multiple unrelated responsibilities in one module without an explicit architectural declaration

### Dependency graph test

- Build an import graph of the codebase
- Assert that no edges violate the dependency rules from Step 2
- Output the violation as a clear diff: "file A imports file B, but layer X cannot depend on layer Y"

### Cross-cutting boundary test

- Verify that cross-cutting concerns (auth, telemetry, etc.) are only imported through the Providers interface
- Flag any direct import of a cross-cutting module from a domain layer

### Convention conformance tests

- Verify naming conventions are followed across all domains
- Verify file organization matches the expected structure
- Verify exported types match expected patterns
- Verify module entrypoints and internal-only files match the declared modular boundaries

---

## Step 5: Integrate into the harness

Wire the custom linters and structural tests into the existing harness so they run automatically on every branch before merge through local `harnesscli` commands.

Keep Phase 4 checks lightweight enough for repeated local use. If a check routinely takes too long to run before merge, move it out of this phase and into Phase 5's scheduled recurring cleanup flow instead of slowing down the local enforcement loop.

### Add to `harnesscli lint`

The custom linters should run as part of `make lint` (which calls `harnesscli lint`) and must be required in the local pre-merge workflow. They should complete quickly enough to be run on every merge candidate. Either:
- Add them as an additional pass within the `harnesscli lint` command
- Or add a `harnesscli lint --architecture` flag and call it from the main `harnesscli lint` flow

### Add to `harnesscli test`

Structural tests should run as part of `make test` and must also be required before merge. They should be fast — they analyze file structure and imports, not runtime behavior.

### Local enforcement

The linters and structural tests must be runnable locally via `harnesscli` and the local make targets before merge. If any invariant is violated, the local verification fails and the change must not merge to `main`. No exceptions.

---

## Step 6: Verify

Run the local harness checks and confirm:

```bash
make lint      # Custom linters pass
make test      # Structural tests pass
```

These commands should be practical to run before every merge. If they are too slow for routine local use, narrow their scope or move the expensive check into the recurring cleanup system.

Intentionally introduce a violation (e.g., add a disallowed import) and confirm the linter catches it with a clear, actionable error message.

---

## Deliverables

- [ ] Machine-readable architecture rules file
- [ ] Dependency direction linter with remediation-quality error messages
- [ ] Module boundary linter that enforces declared domain/layer/module ownership
- [ ] Boundary parsing linter
- [ ] Taste invariant linters (structured logging, naming, file size, etc.)
- [ ] Linter implementation split into focused modules rather than one monolithic helper
- [ ] Structural tests for domain completeness, module ownership, dependency graph, cross-cutting boundaries
- [ ] Integration into `make lint` and `make test` as required local pre-merge checks
- [ ] Local harness checks pass before merge

## Key principle

Constraints are what allow speed without decay. Once encoded, they apply everywhere at once. Be prescriptive about boundaries, modular structure, and invariants, not about implementations.
