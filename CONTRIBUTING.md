# Contributing to Reasonix-Hermes

Thank you for your interest in contributing to the Reasonix-Hermes fork! This
guide covers everything you need to get started. Reasonix-Hermes extends upstream
[esengine/deepseek-reasonix](https://github.com/esengine/deepseek-reasonix) with
MCP bridges, multi-platform bots, Hindsight memory, 17-skill registry, and a
Wails desktop app.

## Prerequisites

- **Go 1.25+** — the project targets the latest stable Go release
- **Git** — for version control
- **Node.js 22+** (required for desktop frontend TypeScript builds)
- **Wails v2** (optional) — only if you work on the desktop app (`desktop/`)

## Getting started

```bash
git clone https://github.com/aliatx2017/reasonix-hermes.git
cd reasonix-hermes
go build ./...             # builds everything
go test ./...              # runs the full test suite
go vet ./...               # static analysis
```

Build all Hermes binaries:
```bash
go build -o bin/reasonix ./cmd/reasonix
go build -o bin/reasonix-bot ./bot
go build -o bin/reasonix-mcpbridge ./cmd/reasonix-mcpbridge
go build -o bin/reasonix-memory ./cmd/reasonix-memoryserver
go build -o bin/reasonix-hooks ./cmd/reasonix-hooks
go build -o bin/reasonix-pr-review ./cmd/reasonix-pr-review
```

Desktop (Wails + React 19 + TypeScript):
```bash
cd desktop/frontend && npm install && cd ../..
cd desktop && wails build -o ../bin/reasonix-desktop
tsc --noEmit                     # in desktop/frontend — must pass before committing
```

## Project structure

| Directory | Purpose |
|-----------|---------|
| `cmd/reasonix` | CLI entry point |
| `cmd/reasonix-hooks` | [Hermes] Native Go hook runner |
| `cmd/reasonix-mcpbridge` | [Hermes] MCP bridge server |
| `cmd/reasonix-memoryserver` | [Hermes] Hindsight memory server |
| `cmd/reasonix-pr-review` | [Hermes] PR review CLI |
| `internal/agent` | Agent loop, session, coordinator |
| `internal/bot` | [Hermes] Multi-platform bot gateway (Discord/QQ/Feishu/WeChat/Telegram/LINE/Slack) |
| `internal/cli` | TUI, subcommands, setup wizard |
| `internal/collab` | [Hermes] Live collaboration WebSocket hub |
| `internal/compress` | [Hermes] Tool output token compressor |
| `internal/config` | TOML configuration loading |
| `internal/constitution` | [Hermes] Project invariants (.reasonix/constitution.json) |
| `internal/control` | Transport-agnostic controller |
| `internal/eval` | [Hermes] Session comparison and evaluation |
| `internal/learn` | [Hermes] Self-improving skill loops |
| `internal/marketplace` | [Hermes] Community skill registry + LobeHub sync |
| `internal/mesh` | [Hermes] Agent-to-agent MCP mesh |
| `internal/orchestrate` | [Hermes] Multi-agent orchestration (chain, pair, CI-fix) |
| `internal/publish` | [Hermes] Session transcript export |
| `internal/scheduler` | [Hermes] Cron-driven automated tasks |
| `internal/tool/builtin` | Built-in tools (bash, read_file, …) |
| `internal/provider` | Model-backend abstraction |
| `internal/provider/openai` | OpenAI-compatible provider (DeepSeek, MiMo, GLM) |
| `internal/provider/ollamacloud` | [Hermes] Ollama Cloud API provider |
| `internal/provider/anthropic` | Anthropic Messages API |
| `internal/plugin` | MCP client (stdio + HTTP) |
| `internal/event` | Typed event stream |
| `internal/hook` | Shell hooks (PreToolUse, …) |
| `internal/memory` | REASONIX.md hierarchy + auto-memory |
| `internal/skill` | Skill discovery from Markdown |
| `internal/sandbox` | OS-level sandboxing |
| `internal/serve` | HTTP/SSE server frontend |
| `internal/checkpoint` | Snapshot-based rewind |
| `internal/acp` | Agent Client Protocol (stdio JSON-RPC) |
| `desktop/` | Wails-based desktop app (separate Go module) |
| `docs/` | Engineering spec, migration guide, feature guides |
| `npm/` | [Hermes] npm packaging pipeline |
| `deploy/` | [Hermes] Helm chart + docker-compose |
| `skills-hub/` | [Hermes] Curated community skill registry |

[Hermes] = Reasonix-Hermes custom additions (not in upstream).

### Dependency direction

```
cli → {agent, plugin, config} → {tool, provider}
```

Built-in subpackages import their parent to self-register via `init()`.
Parents never import children.

## Development workflow

### Building

```bash
make build          # go build ./...  (CLI + plugin-example only)
make test           # go test ./...
make vet            # go vet ./...
make fmt            # gofmt -w .
make hooks          # install git hooks (pre-push: go vet)
make cross          # cross-compile for all 6 targets
```

To build all Hermes binaries including desktop:
```bash
go build ./...                          # all Go packages
go build -o bin/reasonix-desktop ./desktop  # Wails desktop (needs wails)
cd desktop/frontend && npx tsc --noEmit    # TypeScript check
```

### Pre-commit checks

Before committing, run ALL of:
```bash
go build ./... && go vet ./...         # Go builds + static analysis
go test ./...                           # all tests must pass
cd desktop/frontend && npx tsc --noEmit  # TypeScript compiles clean
```

CI enforces these automatically. No PR merges with failing checks.

### Code style

- `gofmt` is enforced by CI — format before committing
- Follow existing patterns: wrap errors with `fmt.Errorf("...: %w", err)`
- Library code never calls `os.Exit` or prints to stdout/stderr
- Only `cli/` and `main/` decide exit codes and user-facing messages
- Exported identifiers must have doc comments
- **English only** for all communication and code authored in this fork
- Chinese comments in `internal/bot/` are upstream-authored — leave them; new code is English-only

### Commit messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
feat(glob): add ** recursive pattern support
fix: replace silent error discards with structured logging
test(event): add comprehensive unit tests for event package
docs: add CONTRIBUTING.md
ci: add golangci-lint and govulncheck
```

## Hermes-specific conventions

These rules are enforced by the `.reasonix/constitution.json` and CI. Violations
block merges.

### Wails/desktop rules
- **`no-nil-slices`**: Wails Go bindings must return empty slices (`[]T{}`), never
  `nil` — `nil` serializes as `null` in JSON, crashing `.length`/`.map` in React.
- **`typescript-clean`**: `tsc --noEmit` must pass before committing desktop changes.
- **`controller-seam`**: Add behavior to `control.Controller`, not individual
  frontends, so CLI/HTTP/Desktop all inherit it.

### Config rules
- **`config-render-complete`**: Every `toml:"…"` tag in `internal/config/config.go`
  must have a corresponding render in `internal/config/render.go`. Run
  `go test ./internal/config/... -run TestRender` to verify round-trip.
- **`i18n-complete`**: New i18n fields must be populated in all 3 catalogs
  (`en`, `zh`, `zh-TW`). `TestCatalogsComplete` enforces this.

### Code rules
- **`spec-first`**: Change `docs/SPEC.md` before changing `internal/` code.
- **`init-registration`**: New built-in tools/providers must self-register via `init()`.
- **`go-vet-clean`**: `go build ./... && go vet ./...` must pass before committing.

### Upstream sync
Always sync upstream before wrapping up a session:
```bash
git fetch upstream
git merge upstream/main-v2
# resolve conflicts, then:
go build ./... && go vet ./... && go test ./...
```
The `.github/workflows/sync-upstream.yml` workflow runs this daily at 20:00 UTC.

## Adding a new built-in tool

1. Create `internal/tool/builtin/mytool.go`
2. Implement the `tool.Tool` interface: `Name()`, `Description()`, `Schema()`, `ReadOnly()`, `Execute()`
3. Register via `func init() { tool.RegisterBuiltin(myTool{}) }`
4. Add tests in `internal/tool/builtin/builtin_test.go` or a separate `mytool_test.go`
5. The tool is automatically available — `main` blank-imports `builtin`

## Adding a new model provider

(For MCP tool servers see `internal/plugin` instead — that's a different layer.)

1. Create `internal/provider/myprovider/`
2. Implement `provider.Provider`: `Name()`, `Stream()`
3. Register via `func init() { provider.Register("mykind", New) }`
4. The provider is available from config with `kind = "mykind"`

## Adding i18n strings

1. Add the field to `internal/i18n/i18n.go` (`Messages` struct)
2. Add the value in all three catalogs: `messages_en.go`, `messages_zh.go`,
   `messages_zh_tw.go`
3. The `TestCatalogsComplete` test fails if any locale is missing a key.
   Three catalogs: `en`, `zh`, `zh-TW`.

## Adding a new rendering section

When adding a new `[section]` to `config.go`, you MUST add its counterpart in
`render.go`. Otherwise `Config.Save()` silently drops the section. Verify with:
```bash
go test ./internal/config/... -run TestRenderTOMLRoundTrips
```

## Submitting changes

1. Fork the repository
2. Create a feature branch from `main`
3. Make your changes with tests
4. Ensure **all** of these pass:
   - `go build ./... && go vet ./...`
   - `go test ./...`
   - `cd desktop/frontend && npx tsc --noEmit`
5. Ensure `gofmt -l .` shows no changes
6. Submit a pull request to `main`

## Reporting issues

Open an issue on GitHub with:
- Steps to reproduce
- Expected vs actual behavior
- Go version and OS
- Relevant logs or error messages

## License

By contributing, you agree that your contributions will be licensed under the
same license as the project.
