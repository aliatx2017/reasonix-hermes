# Reasonix project memory

This file is loaded into every session's system prompt (the cache-stable prefix),
so keep it concise and durable — it is the project's standing instructions to the
agent. It is the Reasonix analog of Claude Code's CLAUDE.md.

## Conventions

- Go kernel under `internal/`; each package owns one concern and documents it in a
  package comment. Match the surrounding comment density and idiom when editing.
- One transport-agnostic `control.Controller` sits behind every frontend (chat
  TUI, HTTP/SSE serve, Wails desktop). Add behavior to the controller, not a
  frontend, so all three inherit it.
- Cache-first: the system-prompt prefix (base prompt + tools + memory) must stay
  byte-stable across turns so DeepSeek's automatic prefix cache stays warm. Never
  mutate it mid-session — ride the turn tail instead (see `control.Compose`).
- **Upstream sync every session wrap-up**: at the end of every session, fetch
  upstream (`git fetch upstream`) and merge the latest `upstream/main-v2` into
  `main`. Resolve conflicts, rebuild all binaries, run `go build ./... && go vet ./...`,
  and update REASONIX.md with the new sync point. Never leave a session without
  checking whether upstream has new commits.

## Memory

- Hierarchical docs: `REASONIX.md` (this file, committed/shared), `REASONIX.local.md`
  (personal, git-ignored), user-global `~/.config/reasonix/REASONIX.md`, and any
  `REASONIX.md` in an ancestor dir. `AGENTS.md` is accepted as a fallback name.
- `@path` on its own line imports another file's contents.
- `#<note>` in chat quick-adds a line here. The `remember` tool saves durable
  facts to the per-project auto-memory store (frontmatter files + `MEMORY.md`
  index), which loads into the prefix on the next session.

## Notes

- **Upstream synced**: `v1.7.0` (commit ed07684, 2026-07-14). 59 commits merged. 7 CLI binaries + desktop built and passing.
- **Commit**: `21c7266` — session stats persistence (CLI → desktop).
- **npm**: `npm i -g reasonix-hermes` — one-line install (sub-packages at `@aliatx2017/reasonix-hermes-*`). Pipeline verified; publish pending 2FA-bypass token.
- **Key v1.6.0 additions**: vision support (image downscaling + detail knob), built-in Time + Context7 MCP servers, configurable shell interpreter (`[tools.shell]`), notification sound system, token economy composer mode, desktop time filter + custom fonts + status bar customization + Windows ARM64, crash capture (Go panics/breadcrumbs/group summaries), lightweight local history + memory retrieval, Traditional Chinese (zh-TW) locale, updater resilience, agent fixes (decline-ask guard, compaction bounds), desktop hooks UI.
- **Key v1.7.0 additions** (merged 2026-07-14): reasoning language settings (`agent.reasoning_language`), session ownership and state routing integration, checkpoint boundary corrections (optimistic rewind), enriched+memoized shell PATH for MCP stdio subprocesses, dropped phrase-matched approved-plan continuation, desktop golangci-lint CI, `SaveDocForTab` Wails binding.
- **Language policy**: All Chinese comments translated to English in `internal/bot/` and `internal/config/` — SPEC §1 compliance restored. Upstream v1.6.0 still has some Chinese comments in IM bot code; these are upstream-authored and left as-is.
- **Layout**: Executables moved from `pkg/` to `cmd/`: `pkg/mcpbridge/` → `cmd/reasonix-mcpbridge/`, `pkg/memoryserver/` → `cmd/reasonix-memoryserver/`.
- **Bug fixes applied** (June 12, 2026):
  - P0-1: BotGateway session eviction (idle timeout + background goroutine)
  - P0-2: `http.DefaultClient` → `netclient.DefaultClient()` across 7 files
  - P1-1: `sqliteStorage.Close()` deferred in memoryserver main
  - P2-1: `DELETE`+`INSERT` → `INSERT OR REPLACE` upsert
  - P2-2: LIKE wildcard escaping with `ESCAPE '\'` clause
  - P2-3: `io.LimitReader` on 5 unbounded HTTP response reads
- **Linux sandbox**: bubblewrap (bwrap) integration — matches macOS Seatbelt profile (read-only root, writable workspace + toolchain caches, network isolation). Graceful fallback when bwrap missing.
- **Graceful shutdown**: memoryserver handles SIGINT/SIGTERM in HTTP mode. Bot already had it.
- **Automated upstream sync**: `.github/workflows/sync-upstream.yml` runs daily 20:00 UTC (04:00 CST). Clean merge → build+test → push. Conflicts → opens PR.
- **Docs written/updated**:
  - New `docs/HERMES-GUIDE.md` — 1,300+ line comprehensive master guide (19 sections)
  - Rewrote `README.md` and `README.zh-CN.md` for Hermes identity
  - Updated `docs/SPEC.md` §2 (39 packages) and §4 (7 ChunkTypes)
  - Updated `docs/GUIDE.md` and `docs/GUIDE.zh-CN.md` with Hermes features

- **Key differentiators**: Discord bot (real agent loop + /goal + /model), MCP bridge (6 tools), Hindsight memory (3 tools, SQLite + TTL/importance + vector search), 17-skill registry, native Go hooks, portable mode.
- **Research-backed design** (July 2026 arxiv sweep):
  - **Cache-stable prefix**: Validated by Zhang, "Can I Buy Your KV Cache?" (2606.13361) — publisher prefill caching is 49.7× cheaper than re-prefill per agent. Our byte-stable system prompt prefix is this exact optimization applied at the session level.
  - **Sandbox + checkpoint + permission gating**: Validated by Carlucci et al., "Loss of Control Risk" (2606.13474) — the constrain/audit/reverse/halt taxonomy maps to our bwrap/Seatbelt + checkpoints + permission architecture. Operational variability erodes safeguards over time → structural invariants (constitution) are the right defense.
  - **AGENTS.md / REASONIX.md**: Validated by Arabat & Sayagh, "Instructions-as-Code" (2606.13449) — well-structured, long instruction files correlate with 20%+ merge-rate improvement in 15,549 agentic PRs across 148 projects.
- **CodeWhale features** (10/10 done, 2026-07-04): Shell env hooks, parallel sub-agent dispatch (task batch), completion sound (/sound), harness profiles (/profile), constitution system (.reasonix/constitution.json), workshop sidecar (>12KB output synthesis), hotbar (desktop keys 1-7), external sandbox (remote OpenSandbox API), Nix flake, Dockerfile.
- **Desktop built**: `bin/reasonix-desktop` (33MB, Wails v2.12.0 + React 19 + Vite 8 + TypeScript 6). Fixed vite.config.ts `manualChunks` for Vite 8/Rolldown compatibility. HotbarConfig restored.
- **Desktop Hermes enrichment** (12 features across 3 tiers, 2026-07-13):
  - **Tier 1**: Cache economy gauge (hit-rate badge), Hindsight memory dashboard (facts/docs/scopes), Discord bot live monitor (online/offline + sessions), Goal progress widget (bar + status badges)
  - **Tier 2**: Live data polling (5s refresh), StatusBar compact widgets (cache% + Discord dot), Skills hub browser (17 skills with search+category filter)
  - **Tier 3**: Sub-agent task tree (with status badges), Constitution health check (rules viewer + JSON template), Checkpoint file list (per-turn file tracking)
  - **Go backend**: 3 new files (`desktop/hermes_dashboard.go`, `hermes_tier3.go`), 13 view structs, 20+ Wails bindings, 2 new Controller getters
  - **Frontend**: 7 new components in `components/hermes/`, all null-safe with polling. Settings → Hermes tab has 7 sections total.
- **CLI TUI Hermes banner**: Custom ╔═╗ double-border header with ⚚ caduceus, "REASONIX-HERMES" branding, compact stats line (model · turns · msgs · tokens · cache% · cost · uptime). `/stats` toggles detailed session statistics panel.
- **Bug fixes** (2026-07-06): duplicate `price` key in `reasonix.example.toml`, data race in `mockProvider.Stream()` (+`sync.Mutex`), `TestSaveToScopesUserAndProjectFiles` cross-platform fix (`HOME` not `XDG_CONFIG_HOME`).
- **Session 2026-07-13** (Discord + Write Mode + D3 + accent wave):
  - **Write Mode**: Panel integration (Write dock tab), CodeMirror 6 with markdown highlighting, FIM completions (Ctrl+Space → DeepSeek API), Hindsight memory sidebar
  - **Checkpoints**: Diff-vs-current button per file, Myers unified diff via `internal/diff.Build`
  - **Memory graph**: D3 force-directed graph with Badges/Graph toggle, zoom/pan/drag
  - **Desktop Hermes accent**: Window title "Reasonix-Hermes", gold-tinted background, CSS accent underline
  - **CLI `/write`**: Opens .md files in $EDITOR, slash-completion + 3 i18n catalogs
  - **Windows theme detection**: OSC 11 + GetConsoleScreenBufferInfo fallback, cross-compiles clean
  - **Constitution**: `.reasonix/constitution.json` with 7 principles, 6 constraints, 7 rules
  - **Discord bot**: Desktop token-input UI, Go ConnectDiscordBot binding, runtime fully wired, CLI bot start support
  - **Bug fixes**: 5 hermes components null-slice hardened, Go bindings return []T{} instead of nil
  - **Upstream**: Merged 5 new commits (v1.6.1, d40797b) — sandbox nul, cold-resume, GSAP, compact sound
  - **38 files changed**, +2989/-154 across Go backend + TypeScript frontend. All 6 binaries rebuilt (34MB desktop, 27MB CLI, 15MB bot, 14MB memory, 9MB bridge, 8MB hooks)
  - **Desktop**: Hermes accent theme ("hermes" — caduceus gold #d4a853), live data push (Wails event loop replaces 5s polling), token sparkline bar chart (ring buffer on Agent → Controller → Wails binding), compaction timeline panel, checkpoint file preview (Store.FileSnaps → expandable per-turn file list), Write Mode (Go filesystem bindings + React split-pane editor + live markdown preview), memory fact graph (clustered by type with color badges)
  - **Desktop config**: `reasonix.example.toml` now documents 24 sections covering all v1.6.0 + Hermes keys (348 lines, taplo-clean)
  - **Remote sandbox**: 10 httptest-based e2e tests covering commandRemote, Spec.remote(), Run() — all success/error/timeout paths
  - **Go internals**: `WorkspaceSlug`/`slugify` now relativize paths to `$HOME` and replace spaces with dashes (fixed `-Users-alex.maksimchuk-Library-Application Support-reasonix-global-workspace` → `$HOME-Library-Application-Support-reasonix-global-workspace`)
  - **CLI TUI**: Pinned ⚚ REASONIX-HERMES header (never scrolls away), turns/msgs/goal/mem moved to bottom status line, `/stats` panel enhanced with Unicode token sparkline (▁▂▃▄▅▆▇█), compaction timeline, memory facts list, goal progress. Banner padding math fixed (no more negative Repeat panics).
  - **VS Code extension fork**: Deleted — not pursuing.
- **VS Code extension fork**: Removed from plans.

- **Session 2026-07-13 (h2)** (tray icon, Write Mode polish, D3 graph enrichments):
  - **Gold tray icon**: `tray_icon_gold.go` overlays Hermes gold on appicon.png, `UpdateTrayIcon` Wails binding, live-syncs on theme style change
  - **Write Mode split-pane**: 3-way Edit/Split/Preview toggle with editor left/preview right
  - **Write Mode file tabs**: Multi-file open with tab bar, close button, dirty-dot, reopen-to-tab
  - **Write Mode auto-save**: Debounced 2s save, automatic dirty-state clearing
  - **D3 memory type filters**: Colored toggle chips filter nodes by type in graph view
  - **D3 click-to-inspect**: Node click → detail panel with title/description/type
  - **D3 vector similarity**: TF-IDF cosine similarity, cross-type edges for sim > 0.3 (dashed accent)
  - **Upstream**: 2 new commits merged (eb624ee) — legacy migration fixes
  - **7 files changed** (+340/-73). All 6 binaries rebuilt. 2 commits pushed.

- **Session 2026-07-14** (Discord bot e2e test + bug fixes + competitive analysis):
  - **Discord bot**: Fully operational — connects via gateway, responds to @mentions and DMs, supports `/model` slash command, approval flow works via plain-text `approve N`/`deny N`. Config in `[bot]` + `[bot.discord]` sections of `reasonix.toml`.
  - **Bug fixes applied** (8 bugs):
    - P0: 14 hardcoded Chinese strings in `internal/bot/gateway.go` translated to English
    - P0: 3 hardcoded Chinese strings in `internal/bot/render.go` translated to English
    - P1: `AllowlistUserCount` missing Discord users — added
    - P1: `PlatformConfigured` missing Discord case — added
    - P1: Dispatch loop blocked by synchronous `runTurn` → now runs in goroutine
    - P1: `approvalShortcutCommand` couldn't parse "approve 1" — now strips trailing digits
    - P2: Stale `/approve`/`/deny` slash commands deleted from Discord on startup
    - P2: Session queue drained after approval to prevent stale message replay
  - **Message splitting**: Bot now sends entire turn response as one Discord message (was flushing every 500ms). Long messages auto-split at paragraph boundaries with continuation.
  - **Language enforcement**: `language = "en"` in config now injects hard English-only instruction at end of system prompt. Reasoning text also enforced via `reasoning_language = "en"`.
  - **Competitive landscape**: Bot autonomously researched 15+ competitors and wrote `docs/COMPETITIVE-LANDSCAPE-2026.md` (437 lines, 9 sections).
  - **Remaining**: Duplicate "Approved." responses still appear occasionally (queue replay edge case). "No pending action found" sometimes fires alongside valid "Approved." (harmless race, not blocking).

## Next session — ideas & follow-ups

### Session 2026-06-13 (expansion plan execution)

**Completed — 10 features across 4 phases:**

| # | Feature | Package | Tests |
|---|---------|---------|-------|
| 1 | IsGLM wired | `internal/provider/openai/` | +1 |
| 2 | Competitive doc fixed | `docs/COMPETITIVE-LANDSCAPE-2026.md` | — |
| 3 | Cron scheduler engine | `internal/scheduler/` (new) | 15 |
| 4 | Hash-anchored edits | `internal/tool/builtin/` | +3 |
| 5 | Session publishing (HTML/JSON) | `internal/publish/` (new) | 9 |
| 6 | Provider cost tracking | `internal/agent/` + `internal/control/` | — |
| 7 | Tool output compressor | `internal/compress/` (new) | 21 |
| 8 | Scheduler → controller wiring | `internal/control/` + `internal/boot/` | — |
| 9 | `reasonix models refresh` CLI | `internal/cli/models.go` (new) | — |
| 10 | `/publish` + `/cost` slash commands | `internal/cli/` | — |

**Token-saving research adopted:**
- **sqz-style SHA-256 content cache** — repeated tool output → compact references (up to 92% reduction)
- **Repeated-line collapsing** — bash output dedup (up to 58% reduction)
- **JSON minification** — null stripping + empty line removal (up to 45% reduction)
- **Entropy safe mode** — preserves errors, stack traces, diffs (≥2 error markers → verbatim)
- **sqz + context-mode** documented as MCP plugins in `reasonix.example.toml`

**New packages this session (6):**
| Package | Lines | Tests | Purpose |
|---------|-------|-------|---------|
| `internal/scheduler/` | 300 | 15 | Cron-driven automated agent tasks |
| `internal/publish/` | 200 | 9 | Session transcript export (HTML/JSON) |
| `internal/compress/` | 290 | 21 | Tool output token compressor (SHA-256 cache, dedup, JSON minification, safe mode) |
| `internal/provider/openai/` (mod) | +5 | +1 | GLM auto-detection |
| `internal/tool/builtin/` (mod) | +30 | +3 | Hash-anchored edit verification |
| `internal/cli/models.go` | 80 | — | `reasonix models refresh` command |

**Total**: 22 files changed, 7 new files, 433 lines modified + ~950 new lines, 49 new tests. All binaries build clean.

**Session 2026-07-14 (h3)** (Telegram bot + learn + mesh):
- **Telegram bot adapter**: Added `PlatformTelegram`, `TelegramBotConfig`, `internal/bot/telegram/` adapter implementing `bot.Adapter` with long-polling via `go-telegram-bot-api/v5`. 16 tests. Wired into gateway, runtime, CLI, allowlist. Config at `[bot.telegram]`.
- **Self-improving skill loops**: New `internal/learn/` package — `Learner` struct observes turn patterns, detects repeated tool sequences (edit-then-test, write-then-build), `BuildReflectionPrompt` for post-session synthesis. 16 tests. Config at `[learn]`.
- **Agent-to-agent MCP mesh**: New `internal/mesh/` package — `Mesh` struct with JSON-RPC/MCP client, `Delegate`/`Broadcast`/`Query`/`Status` operations, `Council` orchestrator with `Convene`/`Consensus`/`Merge`. 13 tests with httptest mock peers. Config at `[mesh]` with `[[mesh.peers]]`.
- **SPEC.md updated**: Added `telegram/`, `compress/`, `learn/`, `mesh/`, `publish/`, `scheduler/` to layout; bot gateway now shows 5 platforms.
- **Config additions**: `[bot.telegram]`, `[learn]`, `[mesh]` sections. `PlatformTelegram`, `TelegramBotConfig`, `LearnConfig`, `MeshConfig` types.
- **Total**: 7 new files, ~12 files modified, 3 new packages, 45 new tests. All binaries build clean.

**Session 2026-07-14 (h4)** (desktop widgets + GitHub Action + LSP audit + docs + collab + deploy):
- **Desktop schedule/cost/publish widgets**: Added Go bindings (`ScheduleDashboard`, `CostSummary`, `PublishSessionHTML`/`JSON`, `ExportSession`) to `hermes_dashboard.go`; `Schedule()` and `SessionMessages()` controller methods; `IsEnabled()` export on scheduler.Task. Three new React components: `ScheduleWidget.tsx`, `CostWidget.tsx`, `PublishWidget.tsx`. Extended `useHermesLiveData`, `HermesSettings`, `bridge.ts`, `types.ts`.
- **GitHub Action for PR review**: New `cmd/reasonix-pr-review/` CLI (fetches PR metadata + diff from GitHub API, pipes to reasonix for review with 6-dimension prompt encoding paper findings). New `.github/workflows/pr-review.yml`.
- **PR review prompt enhanced**: 6 dimensions including deception detection (RogueAI paper) and verification/trustworthiness checks.
- **LSP wiring**: Audited — confirmed already fully wired by upstream. No new code needed.
- **Research documentation**: 3 paper citations in REASONIX.md (#10 KV Cache validates cache-stable prefix; #17 Loss of Control validates sandbox+constitution; #19 Instructions-as-Code validates AGENTS.md). WINDOWS-SANDBOX-DESIGN.md updated.
- **Evidence-first reasoning skill**: New project skill `.reasonix/skills/evidence-first-reasoning/SKILL.md` — encodes Marozzo & Liò protocol.
- **Live collaboration**: New `internal/collab/` package — WebSocket Hub with subscribe/broadcast/steer protocol, 8 tests. Config at `[collab]`.
- **Helm chart + cloud deploy**: `deploy/helm/reasonix/` (7-file Helm chart), `deploy/docker-compose.yml` (single-node $5 VPS), `deploy/README.md`.
- **Dependencies**: `gorilla/websocket` v1.5.3 added to go.mod.
- **Total**: 15 new files, ~12 files modified, 2 new packages, 3 new React components, 8 new tests, 1 new skill. All builds clean.

**Session 2026-07-14 (h5)** (logo + branding + README overhaul + npm packaging):
- **Animated logo**: Diamond Wing (concentric spinning squares + sparkles + swept wings) — transparent background, 7 initial concepts narrowed to refined animated SVG. Combined lockup with "Reasonix-Hermes" monospace wordmark (rainbow gradient crawl) + "AI CODING AGENT" subtitle.
- **README overhaul**: Removed upstream prebuilt install (npm/brew), fixed all binary names (`reasonix-bridge` → `reasonix-mcpbridge`), `English` → `中文` link to zh-CN README, updated custom code paths, added expansion packs table. Both English + Chinese READMEs synced.
- **npm packaging**: `npm i -g reasonix-hermes` — one-line install. `npm/hermes/` package (7 sub-packages across 6 platforms), `bin/reasonix-hermes.js` runner script, `npm/build-hermes.mjs` cross-compile pipeline, `.github/workflows/release-hermes-npm.yml` CI. Pipeline verified (all 6 platforms cross-compile, ~20MB each). Publish pending npm 2FA-bypass token.
- **nil-slice fix**: `SessionMessages()` return `[]provider.Message{}` instead of `nil` — prevents JSON null crash in React.
- **Build artifacts gitignored**: `npm/.stage-hermes/` added to `.gitignore`.
- **Total**: 10 new files, ~6 files modified. All binaries build, all tests pass.

**Session 2026-07-14 (h6)** (session stats persistence — CLI → desktop visibility):
- **Agent aggregate counters**: Added `sessTokensIn`, `sessTokensOut`, `sessTurns` atomic counters to Agent alongside existing `sessCacheHit`/`sessCacheMiss`/`sessCost`. Incremented from `ChunkUsage` (tokens) and `Run()` start (turns). Getters: `SessionTokensIn()`, `SessionTokensOut()`, `SessionTurns()`.
- **Sidecar persistence**: New `SessionMeta` struct + `SetMeta`/`SaveMeta`/`LoadMeta` methods on Session. Aggregate stats written to `<session>.sessionstats` sidecar file (atomic via tmp+rename, distinct from branch `.meta`). `Save()` auto-writes sidecar; `LoadSession()` auto-reads it; `SetSession()` seeds atomics from loaded metadata so resumed sessions show accurate cumulative stats.
- **Controller wiring**: `SessionTokensIn()`/`SessionTokensOut()`/`SessionTurns()` pass-throughs to Agent; `snapshot()` stamps metadata from Agent before every save.
- **Wails bindings**: `SessionTokensView` struct, `SessionTokens()`/`SessionTokensForTab()` bindings, `HermesDashboardEvent.Tokens` field wired into push event.
- **Frontend**: `SessionTokensView` type, binding methods + mocks, `tokens` field in `HermesLiveData`/`HermesDashboardPayload`, aggregate stats widget in Hermes dashboard (turns/tokens-in/tokens-out).
- **Bug fix**: Naming collision with branch `.meta` sidecar resolved (now `.sessionstats`).
- **Total**: 10 files changed (+276/-7). All Go build/vet/test pass; TypeScript compiles clean. 6 CLI binaries rebuilt.

### Next to build
- [ ] **npm: publish reasonix-hermes** — set 2FA-bypass granular token, push tag

### Next to build
- [ ] **npm: publish reasonix-hermes** — set 2FA-bypass granular token, push tag
- [ ] **4-tab skill store UI** — merge SkillsHubBrowser + MarketplacePanel into unified tabbed panel (LobeHub/Market/MCP/Custom)
- [ ] **Online skill sync** — persist synced skills to config, diff on re-sync, show "N new since last sync"
- [ ] **LINE chat adapter** — port LobeHub's LINE adapter pattern to `internal/bot/line/`
- [ ] **Agent task CRUD UI** — Create/Edit task modal for scheduler
- [ ] **Desktop StatusBar: use backend-provided session stats** — wire `SessionTokens()` binding into StatusBar props instead of client-side computation, so resumed CLI sessions show accurate turns/tokens immediately
- [ ] **Session stat import/export** — add stat fields to publish HTML/JSON exports

### Recently completed
- [x] **Session stats persistence: CLI → desktop** (✅ 2026-07-14) — Agent aggregate counters (tokens in/out, turns), sidecar `.sessionstats` persistence, Controller pass-throughs, Wails bindings + frontend widget. 10 files changed (+276/-7). Resolves "why doesn't desktop show session stats for the CLI."
- [x] **LobeHub marketplace API integration** (✅ 2026-07-14) — Full M2M OAuth2 client (stdlib-only HS256 JWT), auto-registration on first use, paginated skill fetch from 360k+ community skills, `SyncFromLobeHub()` registry merge, Wails binding `SyncLobeHubMarketplace()`, desktop "Sync from LobeHub" button with spin animation, CLI `reasonix marketplace sync` command, 4 httptest-based tests, `[marketplace.lobehub]` config section with 8 fields. Verified end-to-end against live API.
- [x] **LAN skills** (✅ 2026-07-14) — 4 new project skills: `searxng-local` (private web search via LAN SearXNG), `crawl4ai-local` (web crawler with headless browser), `google-maps-scraper` (business listings scraper), `last30days` (41k★ social research skill from mvanhorn). All LAN services verified operational.
- [x] **Competitive landscape survey** (✅ 2026-07-14) — Researched 15+ open-source AI agent platforms: LobeHub (78k★), OpenHands (77k★), Cline (63k★), n8n (192k★), Dify (145k★), AutoGPT (185k★), CrewAI (53k★), Aider (46k★), Cognee (18k★), Microsoft AutoGen (59k★), Roo-Code/ZooCode, Langflow (150k★), Firecrawl (132k★). Identified stealable patterns: 4-tab skill store, virtualized grids, custom modes, agent SDK library pattern.
- [x] **CLI banner + version + savings stats** (✅ 2026-07-14) — Dynamic version from ldflags (v1.7.0 default, no more hardcoded v1.6.0). Diamond Wing ◆ logo replaces caduceus ⚚. Status bar enriched: `aux↓N` (aux provider token savings) + `sqz↓N` (compressor byte savings). Compressor atomic stats wired.
- [x] **Skill marketplace** (✅ 2026-07-14) — Community registry (12 skills, agentskills.io-compatible), `internal/marketplace/` Go package, CLI `reasonix marketplace` command, desktop MarketplacePanel.
- [x] **Ollama Cloud provider + auxiliary model routing** (✅ 2026-07-14) — new `ollamacloud` provider kind, 42 models via OpenAI-compatible API at ollama.com/v1. Auxiliary model config: `[agent.auxiliary]` with compression/vision/web_extract overrides. Compaction summarizer + vision requests auto-route to aux providers (cheaper/faster). Tested live: deepseek-v4-flash for compression, gemini-3-flash-preview for vision.
- [x] **Web UI (serve mode)** (✅ pre-existing, enhanced) — 1160-line SPA at `reasonix serve`, SSE streaming, model switching, approvals. Now works with Ollama Cloud + aux models.
- [x] **Desktop collab panel** (✅ 2026-07-14) — Go bindings (CollabDashboard, startCollabHub with steer forwarding), React CollabPanel component (live badge, watcher count, session list), integrated into useHermesLiveData push + polling.
- [x] **Multi-model council UI** (✅ 2026-07-14) — Controller mesh integration (SetMesh/Council/MeshStatus), boot.go mesh creation from [mesh] config, CLI `/council` slash command (status + task dispatch), desktop CouncilPanel widget (peer list + status).
- [x] **E2E test harness** (✅ 2026-07-14) — New `internal/e2e/` package: Harness struct, SessionInputs, SessionTools, TurnCount, Analyze, AssertTools, AssertTurns, ListSessions, RunAll. 7 tests passing.
