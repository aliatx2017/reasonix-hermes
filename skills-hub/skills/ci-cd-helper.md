---
name: ci-cd-helper
description: Generate and fix CI/CD pipeline configurations (GitHub Actions, GitLab CI, Jenkins).
runAs: inline
allowedTools:
  - read_file
  - write_file
  - edit_file
  - grep
  - glob
  - ls
  - bash
---

# CI/CD Helper

Generate, review, and fix CI/CD pipeline configurations.

## Supported Platforms

- **GitHub Actions** (`.github/workflows/*.yml`)
- **GitLab CI** (`.gitlab-ci.yml`)
- **Jenkins** (`Jenkinsfile`)

## Pipeline Patterns

### Standard Stages
1. **Lint & Format** — fast feedback (< 2 min)
2. **Build** — compile the project
3. **Test** — unit + integration tests
4. **Security Scan** — dependency audit, SAST
5. **Package** — build artifacts (Docker, binaries)
6. **Deploy** — to staging, then production (with approval gate)

### Go Project Template (GitHub Actions)

```yaml
name: CI
on: [push, pull_request]
jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.25' }
      - run: go vet ./...
      - run: go build ./...
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.25' }
      - run: go test -race -coverprofile=coverage.out ./...
```

## Review Checklist

1. Does every push/PR trigger CI?
2. Are secrets stored in the platform's secret manager (not in YAML)?
3. Are dependency caches configured?
4. Are timeouts set on each job?
5. Is the Go/Node/Python version pinned?
6. Are deploy steps gated (manual approval for production)?
7. Does CI fail fast on lint issues before running expensive tests?

## Common Fixes

| Problem | Fix |
|---------|-----|
| "No space left on device" | Add `df -h` check, clean up `/tmp` |
| Flaky test in CI | Add `--count=3` for Go, retries for e2e |
| Slow CI (>10 min) | Parallelize jobs, cache dependencies, split test suites |
| Secret leaked in logs | Use `::add-mask::` (GitHub Actions), set +x off |
