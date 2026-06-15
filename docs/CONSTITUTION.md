# Constitution System

> `.reasonix/constitution.json` — structured project invariants the agent
> checks before every tool call. Principles, constraints, and code-level rules
> that define the project's non-negotiable standards.

## What Is a Constitution?

A **constitution** is a set of structured invariants — things that must always be
true, regardless of who is operating the agent. It lives at `.reasonix/constitution.json`
in your project root and is loaded at session start.

It serves three purposes:

1. **Guardrails** — prevents the agent from violating project standards
2. **Onboarding** — new contributors can read the constitution to understand
   project conventions instantly
3. **Auditability** — the constitution is versioned; you can track when rules
   were added or changed

## File Format

```json
{
  "version": 1,
  "principles": [
    "Evidence-first: verify before claiming",
    "Cache-stable: never mutate the system prompt mid-session",
    "Interface-first: add behavior to the Controller, not individual frontends"
  ],
  "constraints": [
    "Never hardcode a switch on model/provider name",
    "Never commit secrets or API keys — use api_key_env",
    "Always sync upstream before ending a session"
  ],
  "rules": [
    {
      "id": "spec-first",
      "description": "Change the SPEC.md contract before changing the code",
      "scope": "internal/**/*.go",
      "severity": "error"
    },
    {
      "id": "controller-seam",
      "description": "Add behavior to control.Controller, not individual frontends",
      "scope": "internal/control/**/*.go",
      "severity": "error"
    },
    {
      "id": "go-vet-clean",
      "description": "go build ./... && go vet ./... must pass before committing",
      "scope": "*.go",
      "severity": "error"
    },
    {
      "id": "no-nil-slices",
      "description": "Wails Go bindings must return empty slices ([]T{}), never nil",
      "scope": "desktop/**/*.go",
      "severity": "error"
    },
    {
      "id": "i18n-complete",
      "description": "New i18n fields must be populated in all 3 catalogs (en, zh, zh-TW)",
      "scope": "internal/i18n/**/*.go",
      "severity": "error"
    },
    {
      "id": "init-registration",
      "description": "New built-in tools/providers must self-register via init()",
      "scope": "internal/tool/**/*.go internal/provider/**/*.go",
      "severity": "warning"
    }
  ]
}
```

### Structure

| Field | Type | Purpose |
|-------|------|---------|
| `version` | int | Schema version (1) |
| `principles` | string[] | High-level design philosophies |
| `constraints` | string[] | Hard, non-negotiable restrictions |
| `rules` | object[] | Code-level checks with scope and severity |

### Rule Fields

| Field | Purpose |
|-------|---------|
| `id` | Unique identifier (kebab-case) |
| `description` | What the rule enforces |
| `scope` | Glob pattern for files it applies to |
| `severity` | `error` (must not violate) or `warning` (should avoid) |

## Loading

The constitution is loaded at session start by `internal/constitution/`. The agent
sees a condensed version of it in its system prompt. The `constitution` tool can
be called mid-session to re-read the file.

## Desktop Health Check

The desktop Hermes dashboard has a **Constitution Health** panel that shows:

- Whether the constitution file is loaded
- Which principles and constraints are active
- A rule-by-rule status (pass/fail based on the current session's actions so far)

## Creating a Constitution

1. Create `.reasonix/constitution.json` in your project root
2. Start with 3-5 principles that define your project's philosophy
3. Add constraints for things you *never* want the agent to do
4. Add rules for code-level patterns you want enforced
5. Version it (`"version": 1`) and commit it

The constitution is checked at tool-call time. If a rule is violated, the agent
receives a warning. Error-severity rules will block the tool call.

## Related

- `docs/SPEC.md` §5 — constitution as part of project configuration
- `AGENTS.md` — the project-level agent instructions (complements the constitution)
- `.reasonix/constitution.json` — the Reasonix-Hermes project's own constitution
