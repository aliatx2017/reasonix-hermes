---
name: refactoring
description: Systematic code refactoring with safety checks — extract methods, rename symbols, simplify conditionals.
runAs: inline
allowedTools:
  - read_file
  - edit_file
  - grep
  - glob
  - ls
  - bash
---

# Refactoring

Systematic refactoring with safety checks. Every change must be verifiable.

## Process

1. **Identify the target** — read the code to understand its structure and dependencies.
2. **Plan the refactoring** — state clearly what you will change and why.
3. **Apply one change at a time** — each edit must be atomic and reversible.
4. **Verify after each step** — run existing tests or build to confirm nothing broke.

## Refactoring Catalog

### Extract Method
- Find a cohesive block of code with a single responsibility.
- Move it to a new, well-named function.
- Replace the original block with a call to the new function.

### Rename Symbol
- Find all references (use grep) before renaming.
- Rename the definition and every reference atomically.
- Verify the build still passes.

### Simplify Conditional
- Replace nested `if` with guard clauses.
- Extract complex boolean expressions into named functions.
- Replace `switch`/`if-else` chains with map lookups or polymorphism where appropriate.

### Remove Duplication
- Find duplicated blocks using grep or manual inspection.
- Extract the common logic into a shared function.
- Ensure both call sites use the new function identically.

### Improve Names
- Names should reveal intent: `processData` → `calculateInvoiceTotal`.
- Avoid abbreviations unless they are domain-standard (e.g. `url`, `id`, `db`).
- Keep names scoped: short names for short scopes, longer names for wider scopes.

## Safety Rules
- Never refactor and add features in the same step.
- Run `go build ./...` (or the project's build command) after each change.
- If a refactoring breaks the build, revert immediately and try a smaller step.
- Do not refactor generated code.
