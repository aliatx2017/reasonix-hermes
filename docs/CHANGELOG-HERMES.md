# Reasonix-Hermes Changelog

Key milestones in the Hermes fork since June 2026.

## v1.9.x (July 2026)

### Session 2026-06-18 (h31) — learner wiring, exchange rates, agent logging, Discord fix

- **Upstream**: merged ×2 — ebea82b (6 commits, heartbeat task system) + ba7a50b (50 commits, goal enforcement, parallel_tasks, /prometheus interview, shared MCP host, cache-impact guard). 3 conflicts resolved across CONTRIBUTING.md, GUIDE.md, chat_tui.go. 18 syncs total (~352 commits).
- **Learner Observe wired**: `agent.Run()` calls `learner.Observe()` via defer with collected tool calls. `boot.go` passes `Learner` to `agent.Options`. Desktop `LearnedPatterns()` stub replaced — now reads from `ctrl.Learner().Patterns()`/`.Trajectories()`.
- **MaxObservations config**: added to `LearnConfig`, plumbed through `boot.go` + `render.go`.
- **Discord dup "Approved." fix**: removed explicit `"Approved."`/`"Denied."` sends from `gateway.go` `handleSlashCommand`. Agent's natural output now provides acknowledgment.
- **Currency display**: default symbol `¥` → `$`. Added `ExchangeRate` field to `Pricing` — CNY costs multiply by rate for USD display.
- **Live CNY→USD exchange rate**: `internal/billing/exchange.go` fetches live rate from `api.exchangerate-api.com` on startup when `[billing] auto_exchange_rate = true`. Falls back to `0.14` on error. Zero new dependencies.
- **Agent operational logging**: `internal/agentlog/` — structured JSON logging via `slog` to stderr (and file via `AGENT_LOG` env). Covers boot, API calls (`model`, `in`/`out`/`total` tokens, `cache_hit`, `latency_ms`), and tool execution (`tool`, `duration_ms`, `result_bytes`).
- **Doc-sweep**: fixed package counts (AGENTS.md 75→69, SPEC.md 55→69), added agentlog to architecture tree + SPEC §2, enriched HERMES-GUIDE §16.9 (exchange rate), §16.16 (Observe wiring), new §16.23 (operational logging).
- **Files**: 15 changed, 3 new (`agentlog.go`, `agentlog_test.go`, `exchange.go`).

### Session 2026-06-18 (h30) — doc-sweep, upstream sync (a3e63f5)

- **Upstream**: a3e63f5 (5 commits) — blank tab title fixes, project tree folder UX. Clean auto-merge.
- **Helm**: image tag v1.8.2 → v1.9.1
- **Doc-sweep**: cross-linked HOWTO-FORCE-ENGLISH.md + HOWTO-TOKEN-SAVING.md + TOKEN-SAVINGS-ANALYSIS.md from README and HERMES-GUIDE. Verified: all 55 packages in SPEC.md accurate, no stale binary names, 11 v1.8.1 references are historical (feature-introduction dates, not stale).
- **Files**: 3 changed (+34/-1).

### Session 2026-06-18 (h29) — learn live-push + Discord deny TOCTOU

- **Upstream synced**: fb4c0c5 (5 commits) — auto research skill, desktop startup settings perf, bundle split, SettingsPanel CPU fix, MarkdownRenderer. 2 conflicts (ModelSwitcher.tsx, bridge.ts) + 2 i18n keys added.
- **Learn live-push**: `LearnPatterns`/`LearnTrajectories` fields added to `HermesDashboardEvent` struct (Go) and `HermesLiveData`/`HermesDashboardPayload` (TS). Wired into Wails event loop + polling fallback in `useHermesLiveData.ts`.
- **Discord deny TOCTOU**: `gateway.go` — deny handler now holds lock through `Approve()` (was releasing between check and call, causing "No pending action found" to fire alongside valid approvals).
- **Files**: 3 changed (+21/-5).

### Session 2026-06-17 (h27) — upstream sync + 7 audit bug fixes

- **Upstream merged**: v1.9.x (ef1f38c, 6 commits) — tool-call name/args backfill on old-session replay, desktop perf (redundant session reload avoidance), Windows Authenticode signing CI, test additions.
- **1 conflict resolved**: desktop/app.go — accepted upstream's `OpenChannelSessionForTab` + `setTabReadOnly` (refactored `rebindTabToLoadedSessionPath`), removed Hermes duplicates.
- **7 audit bug fixes** from the July 2026 deep audit:
  - §1.2: API key length disclosure already fixed (confirmed)
  - §2.3: Path traversal in `findSkillFile` — rejects `..`, `/`, `\` in skill names
  - §1.15: Hooks now exits 1 on errors (was always 0, masking failures)
  - §1.5: Memory server `Recall` — removed `ms.save()` call on every recall (write amplification); bumps persist on next Retain/Tidy
  - §1.12: Collab `Start()` — uses `net.Listen` first, returns bind errors to caller
  - §1.11: Compressor — `turn` field made `atomic.Int32` (no data race); cache capped at 512 with oldest-half eviction
  - §1.14: Publish — guards against empty role string (was panicking on `role[:1]`)
  - §4.4: MCP util — `MaxBytesReader` (10 MB) on both HTTP handlers
  - §4.2: MCP bridge `orchestrateTask` — 15-minute total timeout context; `callDeepSeek` accepts parent context
- **Build**: All 7 CLI binaries rebuilt. Build/vet/test/tsc all green.

### Session 2026-06-17 (h26) — upstream v1.9.1, /learn wiring, doc-sweep

- **Upstream merged**: v1.9.1 (f944dfb, 64 commits) — v1.9.0 + v1.9.1 releases, `plan_mode_allowed_tools` config option, desktop runtime refactor (window state/theme), settings refresh lag fix, bash detached fix, no-auth custom model providers, scroll position session switch fix, status bar item fix.
- **8 conflicts resolved**: agent.go, boot.go, config.go, controller.go, theme.ts, windowState.ts, package.json, pnpm-lock.yaml. All Hermes fields preserved + new PlanModeAllowedTools wired through full chain.
- **Post-merge TS fixes**: Transcript.tsx `tabId` prop (scroll-pin fix #4584), windowState.ts unused wailsjs imports.
- **/learn slash command wired**: Learner through Controller→boot→CLI TUI. `/learn patterns` (confidence-badged) + `/learn trajectories` subcommands. New files: `internal/cli/learn.go`, Controller learner field/getter/setter. i18n: `CmdLearn` in all 3 catalogs.
- **All 8 binaries rebuilt**: Jun 17 timestamps against v1.9.1.
- **Doc-sweep**: 6 docs de-staled (v1.8.x→v1.9.x) — README, README.zh-CN, HERMES-GUIDE, PROJECT.md, RELEASING.md. SPEC.md §2 verified comprehensive (57 packages, 7 bots).

## v1.8.x (July 2026)

### Session 2026-06-16 (h22) — npm publish, upstream merges, token-saving doc
- **npm publish**: `npm i -g reasonix-hermes` — one-line install published for all 6 platforms (darwin/linux/windows × arm64/amd64). Trusted publishing (OIDC) wired — future releases are CI-driven, token-free.
- **Token-saving compressor**: `docs/HOWTO-TOKEN-SAVING.md` — 800-line step-by-step guide for grafting sqz into any Reasonix fork. Covers SHA-256 content cache, repeated-line collapsing, JSON minification, safe mode, and full integration chain (8 steps, 8 files).
- **Upstream merged** (a2709fc, 4 commits): Removed bundled MCP servers (Time/Context7), added codeindex fallback tool, desktop user message actions, edit replay fixes, auto Graphite theme, app icons. Controller consolidation — 126 duplicate declarations cleaned from Hermes sub-files.
- **Upstream merged** (8f3ae36, 18 commits): Credential store backends, Reasonix home asset migrations, config path migration, `/migration-rescue` slash command, desktop project tree visual overhaul (scroll/height/icons/disclosure), keyboard accessibility fixes.
- **Bug fixes**: 3 pre-existing test failures (hooks `session→session_id`, mcpbridge stale key-length check). Rune truncation audit — 2 byte-index bugs in `compress.firstLine()` fixed.
- **Reasoning language**: Verified intact after upstream vision merge — full chain (config→boot→agent→turn injection) operational.
- **CI**: ci-hermes.yml covers 17 packages (exceeds claimed 14+). 7 jobs: lint/vet, test, race, desktop frontend, Wails build, Hermes packages, TOML lint.

### Session 2026-06-?? (h??) — Research workflow, /eval command, upstream sync, crawl4ai/searxng
- **Deep Research workflow adopted**: 5 skills in `.reasonix/skills/` inspired by `Weizhena/Deep-Research-skills` (1.1k★): `/research` (outline), `/research-add-items`, `/research-add-fields`, `/research-deep` (parallel subagents), `/research-report` (markdown synthesis). Phase 1 uses SearXNG (local multi-engine search) + Crawl4AI (JS-rendered page extraction); Phase 2 dispatches parallel task subagents per item; Phase 3 generates comprehensive markdown reports.
- **/eval slash command**: `internal/cli/eval.go` — define, check, report, list, clean subcommands for eval-driven development. Supports PASS/FAIL/MANUAL criteria, pass@1/pass@3 metrics, and ship/blocked recommendations.
- **Upstream merged**: 10 commits (0706284) — compaction retention policy (`keepPolicy`), Wails drag rejection fix, esbuild audit advisory fix. 5 conflicts resolved.
- **crawl4ai/searxng integration**: Research skills now use searxng-local (`:30053`) for structured multi-engine discovery and crawl4ai-local (`:11235`) for deep JavaScript-rendered content extraction. Fallback chain: searxng → crawl4ai → web_fetch.

### Session 2026-06-15 (h22) — audit fixes + doc sweep + upstream vision merge
- **Deep audit (15 fixes across 12 files)**: Verified 20+ claims from `AUDIT-2026-06-15.md` against live code. 14 confirmed + fixed: hooks `session→session_id` (reflect was silently broken), memory SearchDense Lock→RLock, Tidy() preserves CreatedAt via LastDecayAt, memory server binds 127.0.0.1, collab CheckOrigin restricted to localhost + write deadlines, mcputil ReadHeaderTimeout, MCP bridge workdir validation + key length fix, bot gateway TOCTOU lock, compressor SetTurn mutex, scheduler parentCtx propagation, byte→rune truncation in publish.go, mesh atomic request IDs + initialize cache, CI test coverage 3→14+ packages. 1 claim debunked (netclient uses http.DefaultTransport with connection pooling).
- **eval.go rewrite**: Eliminated 60% code duplication (962→500 lines, 19 duplicated functions → unified `evalOutput` callbacks). Fixed 4 bugs: "vet" now runs `go vet`, pass@3 is true run-level metric via `evalComputePassAtN`, build failure output shown, error swallowing replaced with explicit status. Restored `eval compare` subcommand. New `eval_test.go`: 45+ test cases covering pure functions, integration tests, pattern matching.
- **Upstream merged**: 3 commits (a4cea91) — per-model vision capabilities (`VisionModels` on ProviderEntry), explicit vision model selection preservation. Mimo backfill updated with VisionModels population respecting nil-vs-empty-slice distinction.
- **Doc sweep**: 6 files updated — v1.8.0→v1.8.x across README, HERMES-GUIDE, PROJECT.md; AGENTS.md sync point bcd310d→a4cea91 (161 commits/5 syncs).

### Session 2026-06-15 (h18) — Controller decomposition, desktop fix, base prompt, wedge tests
- **Controller decomposition (phase 2)**: controller.go 2,670 → 1,245 lines (53% reduction). Extracted controller_turn.go, controller_mcp.go, controller_stats.go. Now 7 focused sub-files.
- **Desktop hotbar/profiles fix**: Root cause — Go SettingsView was missing Hotbar, Profiles, ActiveProfile fields. Added types, defaults, SetDesktopHotbar/SetProfiles Wails bindings that persist to reasonix.toml.
- **Base prompt hardened**: Removed "briefly summarize", added complete_step evidence protocol.
- **Config**: collab enabled (127.0.0.1:19922), mesh enabled, 5 profiles (daily/review/plan/vision/yolo).
- **Desktop release**: desktop-v1.8.2 tagged + pushed, 37MB arm64 binary.
- **Wedge tests**: +13 edge-case tests (compress +4, collab +3, mesh +2, learn +2, publish +2).
- **Build convention**: "build all binaries = 7 binaries" encoded — user runs ./bin/reasonix chat and open desktop/build/bin/reasonix-desktop.app.

### Session 2026-06-15 (h15) — Vision restore, logo, upstream v1.8.x merge, 13 test fixes
- **Vision pipeline**: `[agent.auxiliary.vision]` restored in `reasonix.toml` (lost via render.go data-loss bug). Correct TOML structure: `[agent.auxiliary]` intermediate table required for BurntSushi/toml parsing. Screenshot analyzed end-to-end via `ollamacloud-vision/gemini-3-flash-preview`.
- **Logo**: Diamond `◆` removed from `docs/logo-animated.svg` and `docs/logo.svg` — now `Reasonix-Hermes`.
- **Upstream merge**: 71 commits from `a029618` → `8ab6d3b` — model switcher with provider grouping + search, Young diagram/KaTeX desktop rendering, inline math fixes, ⌘W/Ctrl+W tab close, slashed LaTeX forms, history payload perf, crash capture improvements.
- **13 test failures fixed** (zero tolerance): CWD isolation via `t.Chdir()`, HOME isolation for crash tests, restored Hermes fields dropped by merge (`SessionMeta` channel metadata, `TabMeta.ReadOnly`, `DesktopLayoutStyle`, `snapshotTab`, `OpenChannelSessionForTab`, live reasoning language propagation).
- **Constitution**: `zero-test-failures` ERROR rule — no test failure tolerated, ever, no "pre-existing" excuse.

## v1.7.0+ (July 2026)

### Session 2026-06-15 (h13) — Golang audit, dead code, t.Parallel, council judge, docs
- Golang patterns audit: `go vet` + `staticcheck` → 5 dead code items removed
- `t.Parallel()`: 96 test functions across 10 custom packages
- Council judge: Fusion Router-inspired `Council.Judge()` with structured JSON (Consensus, Contradictions, CoverageGaps, UniqueInsights, BlindSpots), 6 tests
- Docs: dead link fix, logo concepts removed, 6 stale docs → `CHANGELOG-HERMES.md`
- Vision aux model: `ollamacloud-vision/gemini-3-flash-preview` configured

### Session 2026-06-15 (h12) — Code audit fixes + docs cleanup
- Dockerfile Go 1.24 → 1.25 (matches go.mod)
- Merged duplicate `[desktop]` config sections
- Removed dead `rememberRule` helper, consolidated grant logic
- Memory server migrated from `log.Printf` to structured `slog`
- Helm chart image tag pinned from `latest` to `v1.7.0`
- `docs/PROJECT.md` created — human-oriented project overview
- Deleted 1,997 lines of stale assessment docs → consolidated here

### Session 2026-06-15 (h11) — Completeness sweep + eval GUI + analytics + orchestrate
- Orphan slash completers restored (`/stats`, `/cost`, `/council`, `/learn`, `/publish`, `/todo`)
- Config render.go data-loss bug fixed (10 missing sections)
- Desktop: hotbar "unbound" display fix, eval panel, analytics panel, orchestrate panel, learned patterns panel
- `internal/orchestrate/` — Chain, Pair, CIFix multi-agent workflows (6 tests)
- `CONTRIBUTING.md` rewritten; `docs/index.html` rebranded

### Session 2026-06-15 (h9) — Code health + docs audit + session comparison
- Bridge.ts drift fix (37 method declarations, 5 type mismatches)
- `internal/eval/` — session comparison tool (Jaccard similarity, structural diff, 6 tests)
- SPEC.md §2 overhauled — all 57 internal packages + cmd/ tree documented

### Session 2026-06-15 (h8) — Controller decomposition + bug fixes
- Controller reduced from 3,744 to 2,670 lines (29%); extracted 4 sub-files
- 5 bug fixes: hotbar defaults, desktop layout style missing from settings, render drops profiles/hotbar, netclient mock incompatibility, Mimo backfill gaps
- 14 skills adopted from ~/.hermes/skills
- Slack adapter tests: 23 tests covering all `bot.Adapter` methods

### Session 2026-06-15 (h7) — Audit + 4-phase expansion
- BotGateway session memory leak fixed (eviction loop, 30-min idle timeout)
- `install-source` CLI command
- Dense memory embeddings (`[embedding]` config, OpenAI-compatible `/v1/embeddings`, cosine similarity)
- Autonomous learning loop (`internal/learn/` — pattern detection, skill suggestion, `/learn` command)
- Slack adapter (`internal/bot/slack/` — Socket Mode, DMs + @mentions)

### Session 2026-06-14 (h6) — Session stats persistence
- Agent aggregate counters (tokens in/out, turns)
- Sidecar `.sessionstats` persistence
- Desktop: session stats widget, Wails bindings

### Session 2026-06-14 (h5) — Logo + branding + npm
- Animated Diamond Wing logo (SVG)
- README overhaul (both en + zh-CN)
- `npm i -g reasonix-hermes` — one-line install pipeline

### Session 2026-06-14 (h4) — Desktop widgets + GitHub Action + collab
- Desktop: schedule, cost, publish widgets
- `cmd/reasonix-pr-review/` — GitHub Action for PR review
- `internal/collab/` — WebSocket Hub for live collaboration (8 tests)
- Helm chart + docker-compose for one-click deploy

### Session 2026-06-14 (h3) — Telegram + learn + mesh
- Telegram bot adapter (long-polling, 16 tests)
- `internal/learn/` — self-improving skill loops (16 tests)
- `internal/mesh/` — agent-to-agent MCP mesh (delegate, broadcast, council, 13 tests)

### Session 2026-06-14 — Discord e2e + competitive analysis
- Discord bot: gateway connection, @mentions, DMs, `/model` command, approval flow
- 8 bug fixes (hardcoded Chinese strings, approval parsing, dispatch blocking)
- Competitive landscape analysis: 15+ competitors documented

### Session 2026-06-13 — Discord + Write Mode + D3 + desktop Hermes
- Write Mode: CodeMirror 6 editor with markdown, FIM completions, Hindsight sidebar
- D3 force-directed memory graph with badges, zoom/pan, vector similarity edges
- Desktop Hermes accent: gold theme, live data push, token sparkline, compaction timeline
- Discord bot: desktop token-input UI, ConnectDiscordBot binding, CLI bot start
- 7 new React components, 20+ Wails bindings, 38 files changed

## v1.6.1 (July 2026)
- Upstream merge: sandbox nul, cold-resume, GSAP, compact sound

## v1.6.0 (June 2026) — Fork Foundation
- Vision support (image downscaling, detail knob)
- Built-in Time + Context7 MCP servers
- Configurable shell interpreter
- Notification sound system, token economy composer mode
- Desktop: time filter, custom fonts, status bar customization, Windows ARM64
- Crash capture (Go panics/breadcrumbs/group summaries)
- Local history + memory retrieval, Traditional Chinese (zh-TW) locale

### Session 2026-06-18 (h33) — learner Success gap fix, sidecar persistence, live e2e
- **ToolCallInfo.Success populated**: `executeBatch` now returns outcomes alongside results — learner knows which tools failed (`write_file✗` vs `bash✓`).
- **Learner sidecar persistence**: Patterns + observations saved to `<session>.learning` JSON sidecar via `snapshot()`. Auto-loads on session resume. 4 new tests.
- **Live learner e2e**: `cmd/learner-live-test/` — 5 real DeepSeek turns detected `workflow-bash (confidence=4)`. Full `agent.Run()→learner.Observe()→detectPatterns()` chain proven.
- **7 new tests**: 3 agent integration (observe chain, failure tracking, disabled no-op) + 4 learn persistence (save/load round-trip, missing file, turn counter resume).
- **Agentlog spec + gap fixes**: `tool_exec` now logs `success` (bool), `api_call` logs `cost` (float) and `err` on failure, new `agent.turn` (turn+steps) and `agent.compact` (ratio+messages+kept) events. Full event contract in `internal/agentlog/agentlog.go` package doc.
- **Skills adoption**: 5 skills from mattpocock/skills (diagnosing-bugs, writing-great-skills, domain-modeling, codebase-design, prototype) + SKILLS-CATALOG.md documenting all 125+ skills.
- **Files**: 11 changed, 9 new (skills, tests, catalog, agentlog spec).

## Expansion Packs (June–July 2026)
| Feature | Package | Tests |
|---------|---------|-------|
| Cron scheduler | `internal/scheduler/` | 15 |
| Session publishing (HTML/JSON) | `internal/publish/` | 9 |
| Tool output compressor | `internal/compress/` | 21 |
| Hash-anchored edits | `internal/tool/builtin/` | +3 |
| Ollama Cloud provider (42 models) | `internal/provider/ollamacloud/` | — |
| Auxiliary model routing | `internal/agent/` | — |
| Skill marketplace + LobeHub sync | `internal/marketplace/` | 12 |
| Constitution system | `internal/constitution/` | — |
| E2E test harness | `internal/e2e/` | 7 |
| Remote sandbox (OpenSandbox) | `internal/sandbox/` | 10 |
| LINE bot adapter | `internal/bot/line/` | 11 |
| Fusion Router-inspired council judge | `internal/mesh/` | +6 |
| `t.Parallel()` to 96 test functions | 10 packages | — |

## Bot Platform Support
| Platform | Adapter | Status |
|----------|---------|--------|
| Discord | `internal/bot/discord/` | ✅ E2E tested |
| Telegram | `internal/bot/telegram/` | ✅ 16 tests |
| LINE | `internal/bot/line/` | ✅ 11 tests |
| Slack | `internal/bot/slack/` | ✅ 23 tests |
| Feishu/Lark | upstream | ✅ |
| WeChat | upstream | ✅ |
| QQ | upstream | ✅ |
