# Changelog

All notable changes to the Go line (Reasonix 1.0+) are recorded here. The legacy
`0.x` TypeScript history lives on the [`v1`](https://github.com/esengine/DeepSeek-Reasonix/tree/v1)
branch.

## Hermes fork additions

Our fork (`aliatx2017/reasonix-hermes`) adds:

- **Discord bot gateway** (`bot/`, `internal/bot/discord/`) — slash commands + /goal autonomous loop via discordgo
- **MCP bridge server** (`cmd/reasonix-mcpbridge/`) — 6 tools: run, doctor, plan, orchestrate, get_skill, get_skills
- **Hindsight memory server** (`cmd/reasonix-memoryserver/`) — 3 tools: retain, recall, reflect. SQLite + file backends, TTL/importance scoring
- **Native Go hook runner** (`cmd/reasonix-hooks/`) — zero-dependency binary for pre/post/stop hooks
- **Shared auth middleware** (`pkg/httputil/`) — Bearer token auth, consolidated from mcpbridge+memoryserver
- **MCP utilities** (`pkg/mcputil/`) — shared MCP types and server helpers
- **Skills hub** (`skills-hub/`) — 17 curated community skills with registry.json (incl. adversarial-review)
- **Hardened hook scripts** (`scripts/`) — retain-hook.sh, reflect-hook.sh with dep checks, timeout, integration test
- **Portable mode** — `REASONIX_PORTABLE=1` redirects all data to `<binary_dir>/.reasonix/`
- **Ecosystem reference** (`reasonix-deepseek-ecosystem-2026.md`) — comprehensive survey
- **Research findings** (`docs/RESEARCH-FINDINGS-JUNE-2026.md`) — June 2026 deep-web sweep
- **Implementation plan** (`docs/HERMES-IMPLEMENTATION-PLAN.md`) — phased roadmap

### Hermes v1.6.1-h1 (2026-07-13)

**Write Mode (4 features):**
- **Panel integration**: Write Mode is now a "Write" tab in the desktop right dock (alongside Overview/Files/Changed), with i18n in en/zh/zh-TW
- **CodeMirror 6**: Replaced plain textarea with CodeMirror 6 editor — markdown syntax highlighting, line wrapping, history, autocompletion, Ctrl+S save
- **FIM completions**: Ctrl+Space triggers Fill-in-the-Middle completions via Go binding → DeepSeek `/v1/completions` (prefix/suffix). Toolbar FIM button for programmatic trigger
- **Hindsight injection**: Togglable Memory sidebar (Brain button) — filters all memory facts by keyword overlap with file name + content, shows top 10 as type-colored pills

**Checkpoints & Memory:**
- **Checkpoint actual diffs**: "Diff vs current" button per file in CheckpointFileList. Go binding computes unified diff (Myers via `internal/diff.Build`) between checkpoint snapshot and current file. Rendered via DiffView
- **D3 force-directed memory graph**: MemoryFactGraph toggle between "Badges" (original) and "Graph" (D3 force-directed). Nodes colored by type, sized by title, intra-type cluster edges, zoom/pan, drag, tooltips

**Desktop & CLI:**
- **Hermes accent**: Window title "Reasonix-Hermes", gold-tinted background (#1e1c13/#faf7ef), 2px gold (#d4a853) underline on `.app-chrome` header, `data-accent="hermes"` attribute
- **`/write` CLI command**: Lists .md files in workspace, opens in `$EDITOR`/`$VISUAL` as detached process. Slash completion + 3 i18n catalogs
- **Windows terminal theme detection**: `theme_osc_windows.go` — OSC 11 query (Windows Terminal/WezTerm) + `GetConsoleScreenBufferInfo` fallback with 16-color→RGB mapping
- **Constitution**: `.reasonix/constitution.json` with 7 principles, 6 constraints, 7 code-level rules tailored to the project

**Discord bot integration:**
- **Desktop Settings → Bots**: Discord as 4th install target with token-input flow (no QR needed). `ConnectDiscordBot` validates token via Discord API, saves config via `upsertBotConnection`
- **Runtime wired**: `EnabledPlatforms`, `AdapterBindings`, `PlatformConfigured`, `rememberAllowlist` (users + groups) all handle Discord platform
- **CLI**: `reasonix bot start --channels discord` with allowlist maps updated

**Bug fixes:**
- All hermes dashboard components hardened against nil-slice → `.length` crashes (Wails JSON null from nil Go slices). Go bindings now return `[]T{}` instead of `nil` for 5 methods
- `SubagentTreePanel`, `ConstitutionHealthPanel`, `CompactionTimeline`, `TokenBreakdownChart` — null guards added
- `CheckpointFileList` Go binding restored after accidental truncation during CheckpointFileDiff edits

**Upstream synced**: 5 new commits merged (d40797b) — sandbox nul redirect, cold-resume toggle, GSAP refactor, compact sound controls

### Hermes v1.5.0-h3 (2026-06-12)

**Layout & Docs:**
- **Layout**: Moved executables `pkg/mcpbridge/` → `cmd/reasonix-mcpbridge/` and `pkg/memoryserver/` → `cmd/reasonix-memoryserver/` — Go convention: `cmd/` for binaries, `pkg/` for libraries
- **Docs**: Wrote comprehensive `docs/HERMES-GUIDE.md` (1,300+ lines, 19 sections) covering all upstream + Hermes features
- **README**: Rewrote project README for Hermes fork identity, clear upstream attribution, Hermes feature table
- **SPEC.md**: Updated §2 Layout from 11 to 39 packages, §4 ChunkType from 4 to 7 values
- **GUIDE.md**: Added `/goal` and `/effort` to slash-command list
- **Language policy**: Translated all Chinese comments in `internal/bot/` and `internal/config/` to English (SPEC §1 compliance)
- **Bug fixes**: BotGateway session eviction (P0), `http.DefaultClient` → `netclient.DefaultClient()` (P0), `sqliteStorage` UPSERT + LIKE escaping + `Close()` defer + response body limits (P1-P2)

### Hermes v1.5.0-h2 (2026-06-11)

**P2 — Multi-Agent & Ecosystem:**
- **collab-cli integration**: Pre-configured MCP plugin in `reasonix.example.toml`
- **Adversarial review skill**: `skills-hub/skills/adversarial-review.md` — BLOCK:/ALLOW: contract, 5 attack surfaces (17th skill in registry)
- **VS Code extension**: Decided to fork `whishi47/deepseekcode-reasonix-vscode` as a separate repo

**P3 — Advanced Features:**
- **Multi-model Discord bot**: `/model flash|pro|mimo` command, per-session model preferences stored in `modelPrefs` map, `/new` recreates controller on model switch
- **Vector memory backend**: Sparse TF-IDF cosine similarity search. `semantic=true` flag on `hindsight_recall`. Vectors auto-computed on retain, persisted in both JSON and SQLite backends
- **`get_skill` MCP tool**: 6th tool in `cmd/reasonix-mcpbridge` — reads skill bodies from 3 directory sources, supports `<name>.md` and `<name>/SKILL.md` layouts
- **Discord `/goal` command**: Autonomous goal loop via `BotGateway` — `/goal <obj>` sets + runs, `/goal status`, `/goal clear`. Inherits 50-turn cap + 3-repeat blocked-state audit from controller
- **Native Go hook runner**: `cmd/reasonix-hooks/main.go` — zero-dependency binary replacing shell scripts. Retain/reflect actions, noise-tool filtering, JSON-RPC POSTs to memory server
- **Memory: TTL + importance scoring**: 90-day default TTL, `Importance` field (0.5→+0.05/recall, 1%/day decay), `ExpiresAt` auto-computed, `Tidy()` purges expired, `Recall` skips expired + boosts importance + extends expiry
- **Memory: SQLite backend**: `sqliteStorage` via `modernc.org/sqlite` (pure Go, no CGO). WAL journal mode, 3 indexes (session_id, expires_at, importance). `--backend sqlite` flag. Pluggable `Storage` interface
- **PortaKit portability**: `REASONIX_PORTABLE=1` redirects all data (config, sessions, cache, memory, skills, commands) to `<binary_dir>/.reasonix/`. Added `IsPortable()` + `reasonixDir()` to config package. mcpbridge + memoryserver respect portable mode

**Docs:**
- `docs/HERMES-IMPLEMENTATION-PLAN.md` — P2/P3 statuses updated
- `REASONIX.md` — Next Session TODOs cleared, key differentiators updated
- `CHANGELOG.md` — this entry
- `skills-hub/site/index.html` — static browseable skills catalog (GitHub Pages ready)

### Hermes v1.5.0-h1 (2026-06-25)

- **planTask**: DeepSeek API integration — reads `DEEPSEEK_API_KEY`, `DEEPSEEK_BASE_URL`, `DEEPSEEK_MODEL` from env, returns `# Execution Plan` with numbered steps
- **orchestrateTask**: Decomposes task via DeepSeek API, parses numbered steps, runs each via `reasonix run` in parallel goroutines, returns `# Orchestration Results`
- **callDeepSeek**: Shared HTTP client for DeepSeek chat completions (used by plan+orchestrate)
- **parseSteps/isStepHeader/stripStepPrefix**: Step extraction supporting "1." "1)" "Step 1:" formats
- **Auth consolidation**: Deleted duplicated `requireBearer` from mcpbridge+memoryserver; both now import `reasonix/pkg/httputil`
- **go.mod fix**: `discordgo` flipped from `// indirect` to direct via `go mod tidy`
- **Test coverage**: 165 tests across 4 packages (85.9% aggregate). mcpbridge 82%, memoryserver 89%, discord 91%
- **Hook scripts hardened**: `command -v curl/python3` checks, `--max-time $HINDSIGHT_TIMEOUT` (default 5s), python3 exception handling, curl failure diagnostics
- **Integration test**: `scripts/test-hooks-integration.sh` — 12/12 pass (fake server, auth, noise filter, unreachable, malformed JSON)
- **RESEARCH-FINDINGS updated**: Marked v1.5.0 as synced, added §1.1 post-sync custom additions table

## [1.0.0] — 2026-06-03

First stable release — a **ground-up rewrite in Go**. Not an upgrade of the `0.x`
TypeScript line; a new codebase that becomes the default (`main-v2`).

### Highlights

- **Go kernel**: a single static binary (CGO-free), cross-compiled for
  darwin/linux/windows on amd64 + arm64. Distributed via npm (the package wraps
  the native binary), Homebrew (`esengine/reasonix` tap), and release archives;
  no Node runtime needed to run it.
- **Agent core**: the loop, built-in tools (read/write/edit/multi_edit/glob/grep/
  ls/bash/web_fetch/todo_write), permission gate, sandboxed bash, and the
  DeepSeek prefix-cache–oriented design.
- **Subagents**: `task` plus explore/research/review/security_review skill agents.
- **Skills & hooks**: Claude-Code-style skills (`internal/skill`) and hooks
  (`internal/hook`), symlink-aware and slash-integrated.
- **MCP client**: connect external servers over stdio / Streamable HTTP; reads
  `[[plugins]]` and a Claude-Code `.mcp.json`.
- **Code intelligence via CodeGraph**: a tree-sitter symbol/call graph
  (`codegraph_*` tools) replaces embedding semantic search — no embedding service
  or API cost. Fetched into a local cache on first use (or `reasonix codegraph
  install`) and indexed in the background, so installs and startup stay fast.
- **Plan mode** with evidence-backed step sign-off (`complete_step`).
- **Memory**: `REASONIX.md` hierarchy + auto-memory, folded into the cache-stable
  prefix.
- **ACP** (`reasonix acp`) and an HTTP/SSE server frontend; desktop app (Wails).

### Fixed

- **File encoding support restored** — GBK/GB18030 (and other non-UTF-8) files
  can now be read, edited, and grepped correctly. The v2 rewrite had dropped
  v1's encoding detection; files in CJK Windows charsets were silently misread
  or rejected as binary. The read/edit/write round-trip now preserves the
  original file encoding. (#2637)

### Notes

- Versions: the legacy TypeScript line stays in `0.x`; the Go line starts at
  `1.0.0`. See [docs/MIGRATING.md](docs/MIGRATING.md).
- Release archives ship a bare binary; CodeGraph is fetched on first use. Windows
  support for the fetched runtime is unverified — install `codegraph` on PATH if
  the auto-fetch doesn't resolve there.

[1.0.0]: https://github.com/esengine/DeepSeek-Reasonix/releases/tag/v1.0.0
