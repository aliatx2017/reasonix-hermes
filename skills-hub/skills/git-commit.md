---
name: git-commit
description: Generate conventional commit messages from staged diffs. Follows conventional commits spec.
runAs: inline
allowedTools:
  - bash
  - read_file
---

# Git Commit Generator

Generate conventional commit messages by analyzing staged changes.

## Conventional Commits Format

```
<type>(<scope>): <description>

[body]

[footer]
```

### Types

| Type | When |
|------|------|
| `feat` | New feature |
| `fix` | Bug fix |
| `docs` | Documentation only |
| `style` | Formatting, whitespace (no code change) |
| `refactor` | Code change that's not a fix or feature |
| `perf` | Performance improvement |
| `test` | Adding or fixing tests |
| `ci` | CI/CD changes |
| `chore` | Build, deps, tooling |
| `revert` | Reverting a previous commit |

### Rules

- Description: imperative, lowercase, ≤72 chars, no period at end.
- Scope: optional, e.g. `fix(auth): handle expired tokens`.
- Body: explain **what** and **why**, not **how**. Wrap at 72 chars.
- Footer: `BREAKING CHANGE:` or `Closes #123`, `Refs #456`.

## Process

1. Run `git diff --staged` (or `git diff` if nothing staged).
2. Analyze the diff to understand what changed.
3. Determine the primary type and optional scope.
4. Craft the commit message.
5. Output the final message ready for `git commit -m "..."`.

## Examples

```
feat(api): add user avatar upload endpoint

Support multipart upload to /users/{id}/avatar with S3 backing.
Resized to 200x200 and 400x400 variants. Closes #234.
```

```
fix(session): handle expired refresh tokens gracefully

Previously returned 500 when a refresh token was expired.
Now returns 401 with a clear message so the client can redirect to login.
```
