---
name: documentation
description: Generate or improve code documentation — docstrings, READMEs, API docs, architecture decision records.
runAs: inline
allowedTools:
  - read_file
  - write_file
  - edit_file
  - grep
  - glob
  - ls
---

# Documentation Generator

Generate and improve code documentation. The goal is clarity for the next developer.

## Types of Documentation

### API Docs / Docstrings
- Every exported function, type, and constant gets a doc comment.
- Format: one-sentence summary, then details, then `Usage:` example.
- Go: `// FunctionName does X.` (starts with the symbol name).
- TypeScript/JS: JSDoc `@param`, `@returns`, `@throws`.
- Python: Google-style or Sphinx docstrings.

### README
- Project name and one-line description.
- Quick-start: clone, install, run (3-5 commands).
- Key features (bullet list).
- Link to full docs.

### Architecture Decision Records (ADRs)
- `Title`: decision in one sentence.
- `Status`: proposed / accepted / deprecated / superseded.
- `Context`: why this decision is needed.
- `Decision`: what we chose.
- `Consequences`: what becomes easier, what becomes harder.

### Inline Comments
- Comment **why**, not **what** — the code already says what.
- Mark hacks and workarounds: `// HACK: ...` / `// FIXME: ...` / `// TODO: ...`.
- Explain non-obvious algorithms and magic numbers.

## Process

1. **Identify what needs docs** — exported symbols without comments, complex functions, new features.
2. **Read the code** — understand intent before documenting.
3. **Write or improve** — be concise, be accurate, be helpful.
4. **Don't over-document** — obvious code (`i++`, `return x`) doesn't need comments.

## Anti-patterns

- `// increments i` on `i++` — noise.
- Outdated comments that contradict the code.
- Walls of text explaining simple operations.
- Copy-pasted docstrings that don't match the implementation.
