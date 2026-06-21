# Changelog

All notable changes to the Go line (Reasonix 1.0+) are recorded here. The legacy
`0.x` TypeScript history lives on the [`v1`](https://github.com/esengine/DeepSeek-Reasonix/tree/v1)
branch.

## Unreleased

### Changed

- Agent runtime defaults now leave both executor and dedicated planner tool-call
  rounds unlimited (`max_steps = 0`, `planner_max_steps = 0`). Step limits now
  come from the user/global config only; project `reasonix.toml` does not
  override them.

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
- **Changelog** (`docs/CHANGELOG-HERMES.md`) — Hermes fork milestones, expansion packs, bot platforms

### Hermes v1.7.0-h1 (2026-06-15)

**New bot platforms:**
- **LINE adapter** (`internal/bot/line/`): Webhook server via line-bot-sdk-go/v8, 11 tests, wired into gateway/runtime/allowlist
- **Slack adapter** (`internal/bot/slack/`): Socket Mode with slack-go/slack v0.26.0, DMs + @mentions, 23 tests
- **Telegram adapter** (`internal/bot/telegram/`): Long-polling via go-telegram-bot-api/v5, 16 tests

**New providers:**
- **Ollama Cloud** (`internal/provider/ollamacloud/`): 42 models via ollama.com/v1 OpenAI-compatible API
- **Auxiliary model routing**: `[agent.auxiliary]` config with compression/vision/web_extract overrides

**New packages:**
- `internal/learn/` — self-improving skill loops (pattern detection, skill generation, 16 tests)
- `internal/mesh/` — agent-to-agent MCP mesh (delegate, broadcast, council, 13 tests)
- `internal/collab/` — WebSocket live collaboration hub (subscribe/broadcast/steer, 8 tests)
- `internal/compress/` — tool output token compressor (SHA-256 cache, dedup, JSON minification, 21 tests)
- `internal/scheduler/` — cron-driven automated agent tasks (15 tests)
- `internal/publish/` — session transcript export as HTML/JSON (9 tests)
- `internal/marketplace/` — community skill registry + LobeHub sync (M2M OAuth2, /api/v1/agents endpoint)
- `internal/constitution/` — structured project invariants from .reasonix/constitution.json
- `internal/e2e/` — replay-based regression testing harness (7 tests)
- `internal/eval/` — session comparison tool (6 tests): `reasonix eval compare <a> <b>`

**Desktop Hermes enrichment (10+ widgets):**
- Cache economy gauge, Hindsight memory dashboard, Discord bot live monitor, goal progress
- Live data push (Wails event loop), token sparkline bar chart, compaction timeline
- Checkpoint file preview, memory fact graph (D3 force-directed), sub-agent task tree
- Constitution health panel, collab panel, council panel
- Schedule widget (CRUD), cost widget, publish widget
- 4-tab SkillStorePanel (LobeHub/Market/MCP/Custom)
- Hotbar config, harness profiles, accent theme ("hermes" gold)

**Features:**
- `reasonix marketplace` CLI command (list, search, sync, install)
- `reasonix eval compare` CLI command
- Session stats persistence: CLI → desktop (sidecar `.sessionstats` files)
- Dense memory embeddings: `[embedding]` config, OpenAI-compatible `/v1/embeddings`, cosine search
- StatusBar compact chips: cache%, Discord dot, sqz/aux savings
- Controller decomposition: `controller.go` 3,744 → 2,670 lines (4 sub-files)

### Hermes v1.6.1-h2 (2026-06-13)

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

**Upstream synced**: 7 new commits merged (eb624ee) — sandbox nul redirect, cold-resume toggle, GSAP refactor, compact sound controls, legacy migration fixes

**v1.6.1-h2 additions (same session):**
- **Gold tray icon**: `tray_icon_gold.go` — overlays Hermes gold (#d4a853) at 30% opacity on the system tray icon. `UpdateTrayIcon` Wails binding live-syncs the icon on theme style changes
- **Write Mode split-pane**: 3-way Edit/Split/Preview toggle. Split shows editor left, markdown preview right
- **Write Mode file tabs**: Multiple open markdown files with a tab bar, close button, dirty-dot indicator. Reopening a file switches to its existing tab
- **Write Mode auto-save**: Debounced 2s save after last edit, dirty state clears automatically
- **D3 memory graph type filters**: Colored toggle chips (user/project/feedback/reference/local) filter nodes in graph view
- **D3 click-to-inspect**: Click any graph node → detail panel with title, description, type, close button. Selected node gets white stroke highlight
- **D3 vector similarity links**: TF-IDF cosine similarity between fact descriptions; cross-type edges added for sim > 0.3, rendered as dashed accent lines

### Hermes v1.7.0-h1 (2026-06-15) — Controller Decomposition + Skill Adoption + Bug Fixes

**Controller decomposition:**
- `internal/control/controller.go` reduced from 3,744 to 2,670 lines (29% reduction)
- Extracted 4 focused sub-files: `controller_memory.go` (128 lines), `controller_mesh.go` (44 lines), `controller_approval.go` (585 lines), `controller_checkpoints.go` (365 lines)
- SPEC.md §2 Layout updated with control/ package sub-file documentation

**Security hardening:**
- Editor shell injection: replaced `exec.Command("sh","-lc",...)` with `exec.Command(editor, path)` in `internal/cli/mcp_manager_actions.go` (2 call sites). Removed dead `shellQuote` helper.

**Bug fixes (5):**
- Hotbar "unbound(default)": `hotbarView()` in `desktop/settings_app.go` now falls back to built-in defaults
- DesktopLayoutStyle missing from `SettingsView` struct — added field and population
- Render drops profiles/hotbar: `internal/config/render.go` now renders `[desktop.hotbar]` and `[profiles.<name>]` blocks
- netclient mock incompatibility: `DefaultClient()` now uses `http.DefaultTransport` directly
- Mimo backfill gaps: providers now backfill models, clear mixed-model prices, skip custom URLs

**Slack adapter tests:**
- New `internal/bot/slack/slack_test.go` with 23 tests covering all 7 `bot.Adapter` methods + nil-logger fix in `slack.go`

**Skill adoption (14 from ~/.hermes/skills):**
- Architecture: `cache-first-architecture`, `cost-aware-llm-pipeline`, `anti-patterns`
- Verification: `ready-means-tested`, `pre-action-gate`
- MCP & Go: `go-mcp-server`, `native-mcp`
- Analysis: `github-repo-eval`, `intent-gap-analysis`, `godmode`
- Workflow: `simplify-code`, `spike`, `shell-quoting-ssh`, `upstream-repo-audit`

**Verification:** `go build ./...` + `go vet ./...` pass. All 66 test packages pass (`go test ./internal/...`).


### Hermes v1.6.1-h3 (2026-06-14) — Windows Sandbox + Multi-Provider + Bot Fixes

**Windows AppContainer sandbox** (`internal/sandbox/appcontainer_windows.go`, ~360 lines):
- OS-level process isolation via CreateProcess with SECURITY_CAPABILITIES — available since Win8+, no external dependencies
- Profile creation (CreateAppContainerProfile), capability SID derivation (internetClient/internetClientServer), WriteRoot ACL granting (SetNamedSecurityInfo via ACLFromEntries)
- `ExecAppContainer()` launcher with CreateProcess + pipe I/O for stdout/stderr capture, integrated into bash tool via `sandbox.IsAppContainer()` guard
- Stubs for non-Windows (`appcontainer_stub.go`), `seatbelt_other.go` build tag narrowed to `!darwin && !windows`
- Refactored `writeAllowDirs()` from seatbelt_darwin.go → sandbox.go for cross-platform sharing
- Package doc updated to list all three backends (macOS Seatbelt, Linux bubblewrap, Windows AppContainer)

**Multi-provider expansion:**
- **GLM (Z.ai/bigmodel.cn)**: `IsGLM()` host detection function + 9-case test in openai provider. Uses standard OpenAI reasoning_effort — no special protocol needed. Example config entry in `reasonix.example.toml`
- **MiniMax**: Already supported via existing `IsMiniMax()` detection; added example config entry (minimax-m3, binary thinking knob)
- **Codex**: Works via openai kind with api.openai.com endpoint for standard models; full Responses API implementation deferred

**Bot bug fixes (3):**
- **Approve race condition** (`gateway.go`): `normalizeApprovalShortcut` retries up to 500ms for approval event to register in gateway state — fixes race where fast "approve"/"y" arrives before agent's ApprovalRequest callback
- **Duplicate approve guard** (`gateway.go`): `/approve` and `/deny` handlers now verify approval is still pending before sending "Approved."/"Denied." — prevents duplicate responses from double-clicks/retries
- **Bot message dedup** (`discord.go`): Added `botOwnStatusMessage()` matching known bot responses (Approved., Denied., No pending action, Task stopped, etc.) as belt-and-suspenders filter in `onMessageCreate`
- **approvalShortcutCommand fix** (`gateway.go`): Standalone digits "1"/"2"/"0" now handled before trailing-digit stripping — was destroyed by TrimRight in prior Hermes fix

**Agent behavioral rules** (`.reasonix/constitution.json`):
- Added 2 ERROR-severity rules: `never-say-fixed` (banned word "fixed") and `substantiate-every-claim` (no assertion without evidence). Auto-formatted into system prompt via `internal/constitution.Format()`

**Test fixes (6 pre-existing failures):**
- Discord `TestOnReady` nil pointer — `dg.State.User` wasn't set before calling onReady
- 5 gateway tests — updated Chinese→English expected strings to match translated gateway messages
- All 5 bot test packages now pass green (`go test ./internal/bot/...`)

**Competitive landscape verification:**
- All 14 star counts verified via GitHub API — within 0.3% accuracy. 7+ feature claims across 9 competitors confirmed. No corrections needed.

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
- `docs/CHANGELOG-HERMES.md` — consolidated Hermes fork changelog
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
