---
name: migration-assistant
description: Assist with framework/library migrations: analyze breaking changes, update imports, adapt patterns.
runAs: inline
allowedTools:
  - read_file
  - edit_file
  - write_file
  - grep
  - glob
  - ls
  - bash
---

# Migration Assistant

Guide for migrating between framework or library versions. Systematically analyze, plan, and execute migrations.

## Process

### 1. Scope
- Identify all files that use the target library/framework: `grep` for imports.
- Read the migration guide / changelog for the target version.
- List every breaking change that applies to this codebase.

### 2. Plan
- Order the changes from mechanical to semantic:
  1. **Mechanical**: renamed APIs, moved packages, removed deprecated functions.
  2. **Structural**: changed patterns, new required parameters, different lifecycle.
  3. **Behavioral**: changed defaults, different error handling, async/sync shifts.

### 3. Execute
- Apply mechanical changes first (they're often scriptable).
- Verify the build after each batch of changes.
- For structural changes, adapt one file at a time.
- Run tests after each significant change.

### 4. Verify
- `go build ./...` or equivalent for your language.
- Run the full test suite.
- Manual smoke test of affected features.
- Check for warnings/deprecations that remain.

## Common Migration Scenarios

### Go Module Upgrades
```bash
go get -u ./...
go mod tidy
go build ./...
go vet ./...
```

### Framework Major Version
```bash
# 1. Read the official migration guide
# 2. Update import paths (grep + multi_edit)
# 3. Adapt to new APIs
# 4. Update config/schema files
# 5. Run tests
```

### Database Migrations
- Always write a reversible migration.
- Test the rollback path.
- Back up data before running on production.

## Safety Rules
- Commit before starting a migration.
- One change at a time; build and test between changes.
- If you get stuck on a change for more than 10 minutes, flag it for manual review.
- Never mix migration changes with feature work.
