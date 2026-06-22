# Reasonix-Hermes Changelog

Key milestones in the Hermes fork since June 2026.

## v1.10.x (June 2026)

### Session 2026-06-21 (h54) — agent-reach MCP + taste-skill desktop audit + npm v1.11.0

- **agent-reach e2e**: Verified sandbox whitelist fix works — agent-reach v1.5.0 runs inside bash=enforce sandbox from reasonix CLI sessions. `~/.agent-reach` + `~/.local/share` write access confirmed.
- **agent-reach MCP server**: `.reasonix/scripts/agent-reach-mcp` (148 lines), 3 tools: `get_status` (doctor report, 7/13 channels), `read_web` (Jina Reader → markdown), `youtube_subtitles` (yt-dlp extraction). Registered as `[[plugins]]` in reasonix.toml. E2E verified — both `get_status` and `read_web` called successfully from reasonix session.
- **headroom + markitdown MCP**: Both verified end-to-end. `headroom_stats` (session stats), `headroom_compress` (router:noop on short input), `markitdown convert_to_markdown` (example.com → clean markdown).
- **Taste-skill desktop audit**: Found Hermes theme style was a ghost — registered in `theme.ts` but had zero CSS rules. Selecting "Hermes" changed only the tray icon (Go-side), not the UI. Added full `:root[data-theme-style="hermes"]` CSS block (~190 lines): dark mode (warm surfaces #0a0906→#1a1812, gold accent #d4a853, cream text #f2efe5), light mode (parchment surfaces, gold #b88c3c), auto/@media variant. Desktop rebuilt (39.8MB), CSS syntax check + guard tests all pass.
- **npm v1.11.0**: `v1.11.0` + `hermes-npm-v1.11.0` tags pushed. 6-platform binaries cross-compiled and staged. CI publishes to npm via OIDC trusted publishing.
- **Build**: 2 files committed (styles.css + AGENTS.md), 2 new files created (script + MCP). All 9 binaries rebuilt. Full test suite (76 packages) + desktop tests + tsc clean. All 10 `.reasonix/check` steps pass.

### Session 2026-06-21 (h51) — SettingsPanel split + silent-catch completion

- **SettingsPanel.tsx split**: 5,511-line monolith reduced to 2,152 lines (61% reduction). Extracted 3 new files: `settings-shared.tsx` (192L — layout primitives, SectionProps, ref helpers, proxy mode), `ModelsSection.tsx` (1,768L — Models, Providers, ModelPicker, ProviderEditor, StepLimitControl), `BotsSection.tsx` (1,457L — Bot config, connections, QR install, credentials). Extraction done incrementally (verify tsc after each batch) — previous 2 sessions got stuck attempting it all at once, burning 69 API calls in an unsolvable edit-compile loop.
- **Silent-catch fix** (completes h50 audit): 25+ `.catch(() => ...)` sites across 13 files now log `console.warn` before returning fallback values (App.tsx, WorkspacePanel, MemoryPanel, WriteMode, SettingsPanel, CapabilitiesPanel, hermes/AnalyticsPanel, LearnedPatternsPanel, EvalPanel, MarketplacePanel, SkillStorePanel, useController.ts, sessionExport.tsx). No shared `useBackend` hook — fallback types and side effects are too diverse for a single abstraction. Zero remaining silent `.catch(() =>` patterns.
- **Build**: tsc --noEmit 0 errors, go build ./... + go vet ./... clean, all 77 test packages pass.

### Session 2026-06-21 (h53) — upstream sync (desktop shortcuts + sidebar Cmd+1-10 nav)

- **Upstream**: merged 051239b6 (18 commits: desktop shortcut integration, sidebar Cmd+1-10 topic navigation, Cmd+K palette double-press fix, ⌘B/⌘⇧B shell/sidebar toggle swap, panel spacing + light-mode background fixes, keyboard shortcut settings panel wiring, ⌘↑/↓ native scroll restore, sidebar topic numbering fixes). 30 syncs total.
- **Conflicts**: 4 — all resolved (accepting upstream). `keyboardShortcuts.ts` (ShortcutAction additions + shell.toggle→sidebar.toggle swap), `CommandPalette.tsx` (active=-1 + useCallback/useLayoutEffect), `keyboard-shortcuts.test.ts` (topicShortcutIndexFromEvent import), `app-chrome-tabs.test.ts` (bg-elev CSS variable).
- **Build**: All 9 binaries rebuilt (wails 25.6s). Full test suite: 76 packages pass, zero failures. tsc --noEmit 0 errors. Desktop Go tests: 3/3 pass. h52 guard tests re-verified: 24/24 pass.
- **Hotbar fix**: Keyboard handler for digit keys 1-7 restored (lost since v1.6.0 merge Jun 12 — 9 days). RightDockMode "none" also restored. 11-assertion guard test prevents recurrence.
- **System-wide fingerprint guard**: 92 fingerprints across 17 shared files (69 Go + 23 TS) now run as pre-commit gate. Upstream merges that silently drop any Hermes code block fail `.reasonix/check`.
- **Git-history audit**: 4 confirmed silent losses from upstream merges found. 3 restored: `languagePolicy` (hard English-only enforcement), `workshopSynthesizer` + `WorkshopThreshold` wiring (background synthesis sidecar), `sound` field in render.go. 1 unrestorable: `runWithAutoResume` (architecture changed).
- **Constitution**: new `frontend-guard-tests` rule — structural guard tests must pass before committing desktop changes.

### Session 2026-06-21 (h52) — regression fixes + 32 guard tests

- **Two regressions fixed** (both lost during prior upstream merges with zero guard tests):
  - **sqz counter missing from CLI status bar**: `cfg.Agent.CompressToolOutput` (`*bool`, default true) existed but was never mapped to `agent.Options.CompressToolOutput` in boot.go. Compressor always created disabled → `BytesSaved` stayed 0 → sqz counter never rendered. Fixed by adding `CompressToolOutputEnabled()` getter (follows `ColdResumePruneEnabled` pattern) and wiring `CompressToolOutput: cfg.CompressToolOutputEnabled()` in boot.go agent.Options.
  - **Desktop hotbar always unconfigured**: `SettingsView` struct in `desktop/settings_app.go` had no `Hotbar`/`Profiles`/`ActiveProfile` fields — zero data reached the frontend. Also missing `SetDesktopHotbar`/`SetProfiles` Wails bindings. Fixed by: adding `json` tags to `HotbarConfig`, creating Go `ProfileView` type, adding 3 fields to `SettingsView`, populating them in `Settings()` with `hotbarView()` helper (fills defaults), adding 2 Wails bindings.
- **32 regression guard tests** across 3 files to prevent silent loss during upstream merges:
  - `internal/config/default_test.go`: +15 tests — every Hermes config struct field (Billing, Learn, Mesh, Schedule, Collab, Marketplace, Embedding, 4 bot platforms, Notifications.Sound, Sandbox.Remote*, Agent.Auxiliary, AgentLog) now has a compile-time guard
  - `internal/boot/boot_test.go`: +5 tests — exchange rate wiring, compress wiring, schedule→controller, mesh→controller, learner→controller
  - `desktop/settings_app_test.go`: +12 tests — hotbar struct presence + persist, 8 Wails bindings (language, appearance, layout, telemetry, close behavior, display mode, status bar style, status bar items), SetProfiles
- **Build**: go build/vet/test all pass, tsc --noEmit 0 errors, 9 binaries rebuilt, desktop tests passed.

### Session 2026-06-21 (h50) — upstream sync + coding-standards audit + doc-sweep

- **Upstream**: merged 9ada1417 (13 commits: agent step limits user-global, cancelled batch results preservation, TUI cancel-escape integration, pwsh chaining + install path, todo cleared state fix, auto-plan user-level). 3 conflicts resolved: CHANGELOG.md, settings-refresh-snapshot test, agent.go (kept outcomes return for Hermes learner). 29 syncs total.
- **Coding-standards audit**: 70 silent `.catch(() => {})` → `console.warn` across 27 desktop frontend files (useController.ts, StatusBar.tsx, Composer.tsx, 9 hermes components, WriteMode.tsx, etc.) — backend failures now visible in console instead of silently swallowed. 10 `any` holes typed (SetDesktopHotbar→HotbarView, SetProfiles→Record<string,ProfileView>, 4 Wails EventsOn payloads, 5 Promise<any>→proper types in bridge.ts). tsc --noEmit 0 errors.
- **Doc-sweep**: 102 docs inventoried, 12 stale claims fixed across 11 files (CODEMAPS source paths ×4, binary name, year ×2, pricing date, ecosystem version ×4). SPEC.md enriched with `pkg/` tree. QQ bot section added to BOT_GUIDE (EN + ZH-CN). i18n verified balanced via TestCatalogsComplete (335 keys × 3 catalogs).
- **Build**: All 9 binaries rebuilt. Full test suite (77 packages) + tsc clean.

## v1.9.x (June 2026)

### Session 2026-06-20 (h46) — upstream sync + audit bug fixes + desktop CSS fix

- **Upstream**: merged 91fe06d (4 commits: MiMo built-ins migration to custom providers, codebase-memory MCP auto-indexing). 4 conflicts resolved: README.md, README.zh-CN.md (upstream bullet-point foundation section), desktop/settings_app_test.go (MiMo test replacement), internal/config/config.go (deleted Hermes MiMo backfill functions). 27 syncs total (~540 commits).
- **Upstream regression fixed**: `normalizeLegacyDesktopProviderAccessForSettings` in `desktop/settings_app.go` was calling `cfg.SaveTo(path)` on project config path using `RenderScopeProject`, which stripped the `[desktop]` section — destroying desktop settings (theme, style, close_behavior) on disk. Removed spurious save. Bug also confirmed on clean upstream checkout.
- **Audit verification**: analyzed 34 claims from `docs/reasonix-audit-06202026.md` (2 versions). Found 5 real bugs, debunked 10 false claims, identified 4 self-debunked items, ranked 15 enhancement ideas into 4 tiers.
- **5 bugs fixed** (audit findings):
  - `cmd/reasonix-hooks/main.go`: `json.Marshal` errors now handled explicitly (was `_`)
  - `cmd/reasonix-mcpbridge/main.go`: flag parsing rewritten as loop-based; `--http --port N` works in any order
  - `cmd/reasonix-memoryserver/main.go` (3 fixes): `--port`/`--http`/`--backend` consolidated into single loop parser; `Recall()` now persists boosts via `ms.save()`; hourly `Tidy()` goroutine added
- **Desktop CSS fix**: added `max-height: 40vh; overflow-y: auto` to `.msg--user .msg__text` in `desktop/frontend/src/styles.css` — long user messages now get their own scrollbar instead of consuming the entire viewport.
- **Files**: 8 changed (3 Go binaries, 1 desktop Go, 1 CSS, 2 docs). All 9 binaries rebuilt. Full test suite + desktop tests + tsc clean.

### Session 2026-06-20 (h45) — session notes + next-session todos

- **Docs**: REASONIX.md updated with h45 session wrap-up notes. Next-session todos saved for CHANGELOG catch-up, CODEMAPS regeneration, HERMES-GUIDE reorder.
- **Clean working tree**: No uncommitted changes.

### Session 2026-06-20 (h44) — upstream sync (v1.10.0) + historyToolCall fix

- **Upstream**: merged 49c1476 (8 commits, v1.10.0). 4 conflicts resolved: desktop/app.go (channel functions moved), desktop/tabs.go (whitespace), desktop/settings_app.go (readOnly field), internal/boot/boot.go (classifier code moved — re-applied all Hermes customizations on top of upstream refactored boot.go: agentlog, exchange rate, aux providers, remote sandbox, learn/mesh/schedule).
- **historyToolCall Arguments + canArchive fix**: restored full tool call arguments and results in historyMessages after upstream "slim history payloads" optimization (d562e37) broke `TestHistoryMessagesIncludeAssistantReasoning`.
- **Build**: All 9 binaries rebuilt. All 83 test packages pass. Desktop tests + tsc clean.

### Session 2026-06-20 (h43) — desktop SessionAPI migration complete

- **SessionAPI port migration**: `WorkspaceTab.Ctrl`, `activeCtrl()`, `activeCtrlLocked()`, `ctrlByTabID()` all changed from `*control.Controller` to `control.SessionAPI` across 12 non-test files + 6 test files. Zero `*control.Controller` remains in the desktop Go tree.
- **HermesState sub-port**: Extended with Mesh/Schedule/AddScheduledTask/RemoveScheduledTask.
- **API surface**: SessionHistory + TurnControl + Status extended with CheckpointFileSnaps, SessionMessages, ActivePricing. All 79 desktop `ctrl.*` methods now on the port.
- **Build**: Desktop builds, tests, tsc — all green. 9 binaries rebuilt.

### Session 2026-06-20 (h42) — upstream merge + Hermes SessionAPI port extension

- **Upstream**: merged c202f97 (2 commits: CLI TUI migrated to SessionAPI port). No conflicts — clean auto-merge.
- **Hermes methods surfaced on port**: Added HermesState sub-port (Learner/Council/MeshStatus), extended Status (SessionTokensIn/Out/Turns/Cost, CompressStats, AuxTokens, TurnUsageHistory, CompactionHistory), extended Goals (GoalTurns/GoalBlocks).
- **Build**: All 9 binaries rebuilt. All 72 test packages pass. tsc + desktop tests clean.

### Session 2026-06-20 (h41) — upstream merge + pnpm migration

- **Upstream**: merged 2909ef1 (20 commits). 3 conflicts resolved: controller.go struct + constructor, tabs.go legacy migration. Resolutions: kept Hermes fields (schedule/mesh/learner/learnerLoaded) alongside upstream memMu + approvalManager; adopted upstream double-check-under-lock pattern in tabs.go; accepted package-lock.json deletion (pnpm migration).
- **GoalMachine**: Added `Turns()`/`Blocks()` getters.
- **Desktop**: pnpm install + tsc + wails build. All 9 binaries rebuilt. All 72 test packages pass.

### Session 2026-06-19 (h40) — tray icon fix + CODEMAPS + HERMES-GUIDE reorder + doc-sweep

- **Tray icon fix**: `UpdateTrayIcon` argument mismatch resolved — 0 args in Go, 1 in frontend.
- **CODEMAPS regenerated**: 5 files (architecture, backend, dependencies, data, frontend) — all stale since 2026-06-06.
- **HERMES-GUIDE reorder**: §16 sections renumbered 16.1–16.26 sequentially, 2 duplicates removed.
- **Doc-sweep**: 4 stale claims fixed — SPEC 71→70 packages, PROJECT + AGENTS upstream commits.
- **Files**: 12 changed (+222/-147).

### Session 2026-06-19 (h39) — upstream merges (33 + 5 commits) + doc-sweep + GH issues

- **Upstream**: merged db43de8 (33 commits: list_sessions/read_session tools, MCP session reinit, history normalization, todo panel fixes, crash stats, credential hardening) + 7032f39 (5 commits: skill scripts listing). 15 conflicts resolved (3 Go, 12 TSX/CSS).
- **HljsDiff.tsx type fix**: stale `diffRowsFromUnifiedDiff` import removed.
- **Doc-sweep**: 6 stale claims fixed — SPEC.md 69→71 packages + sessiontool entry, AGENTS.md test counts, PROJECT.md 75→71, README binary list.
- **GitHub issues**: #13, #14, #15 closed.
- **Build**: 9 binaries rebuilt. All tests pass.

### Session 2026-06-19 (h36) — learn pipeline end-to-end + HERMES-GUIDE renumbering + dead-code refactor

- **Learn pipeline wired end-to-end**: `SuggestSkill` → desktop `LearnedPatterns()` binding, `/learn reflect` subcommand → `BuildReflectionPrompt` → agent turn.
- **HERMES-GUIDE §16 renumbering**: 16.1–16.26 sequential with zero duplicates, TOC synced.
- **Dead-code test refactor**: `billing.Fetch` → `FetchWithClient`, `parseSlackTS` deleted, `qqSendURL` → adapter method.
- **Host checks consolidated**: 2 → 1 (`.reasonix/check` helper).
- **Doc-sweep**: 5 stale claims fixed — index.html v1.8.x→v1.9.x, SPEC.md 70→69 packages, CODEMAPS Go 1.24→1.25 + 5→7 binaries, AUDIT historical note.
- **Files**: 17 changed (+89/-122).

### Session 2026-06-19 (h35) — desktop bot live monitor, log rotation e2e, skill rewrites, domain model, bug fixes, dead code

- **Desktop bot live monitor**: Multi-platform status badges replacing Discord-only. New `BotPlatformStatus` struct (per-platform); `BotLiveStatusView` now carries `[]BotPlatformStatus`. Gateway gained `PlatformSessionCount()` + `HasPlatform()`. Frontend: `BotLiveMonitor` component renders per-platform chips with platform-specific icons (Discord/Telegram/LINE/Slack), green/gray dots, session counts, and webhook tooltips. Wails push events (3s) + 10s polling fallback.
- **Log rotation e2e**: Verified end-to-end — 14MB `agent.log` auto-rotated to `agent.log.1` on `Init()`, fresh log created with all 6 event types present.
- **5 skill rewrites** (v2.0, applying `writing-great-skills` principles): `pre-action-gate` (62→37 lines), `ready-means-tested` (61→45), `cache-first-architecture` (90→65), `cost-aware-llm-pipeline` (130→63), `doc-sweep` (145→107).
- **`diagnosing-bugs`**: Removed stale first frontmatter block (sediment).
- **Domain model**: Created `CONTEXT.md` — 18 canonical terms across 6 clusters with `_Avoid_` alternatives. Created `docs/adr/0001-cache-first-immutable-prefix.md` and `docs/adr/0002-controller-seam.md`.
- **Bug fixes (2)**: `TestClearSessionRemovesRunningJobArtifacts` + `TestToWireUsageWithPricing` — config isolation + currency field.
- **Intent-gap analysis**: 4 priority fixes — SPEC §1.2 "Single static binary" → "Multiple static binaries", §1.3 dependency language updated, package counts 69→70, nil slice fix.
- **Dead code cleanup**: 4 symbols removed — `IsMaxStepsPause`, `SchemaTokenCosts`, `ToolSchemaCost`, `MemoryStore.Stats` + `MemoryStats`.
- **Doc-sweep**: 41 docs inventoried, CONTEXT.md + ADRs cross-linked. verify-session.sh binary count 8→9.
- **Files**: 25 changed, 3 new (CONTEXT.md, 2 ADRs). 314 insertions, 477 deletions (net -163).

### Session 2026-06-19 (h34) — currency symbol root-cause fix + agent log rotation

- **Currency symbol ¥ root cause**: Sub-agent Usage events from task tool, skill runner, planner, and classifier bypassed exchange-rate cloning — 4 sites in `boot.go`. Extracted `applyExchangeRate()` helper; all 4 sites now use cloned pricing with `ExchangeRate > 0`.
- **Agent log rotation**: Self-rotation on `Init()` — checks `agent.log` size, rotates `.log` → `.log.1`, shifts chain up to `max_backups` (default 5). New `[agentlog]` config section.
- **Agent log coverage audit**: All 8 contracted event types verified logged. 7 new agentlog tests.
- **Doc-sweep**: SPEC.md tree 69→69 (5 sub-packages added), README.zh-CN.md bot list fixed, HERMES-GUIDE §16.23 +log rotation, PROJECT.md +agentlog/billing.
- **Files**: 7 changed.

### Session 2026-06-18 (h33) — learner Success gap fix, sidecar persistence, live e2e

- **ToolCallInfo.Success populated**: `executeBatch` now returns outcomes alongside results — learner knows which tools failed.
- **Learner sidecar persistence**: Patterns + observations saved to `<session>.learning` JSON sidecar via `snapshot()`. Auto-loads on session resume. 4 new tests.
- **Live learner e2e**: `cmd/learner-live-test/` — 5 real DeepSeek turns detected `workflow-bash (confidence=4)`.
- **7 new tests**: 3 agent integration + 4 learn persistence.
- **Agentlog spec + gap fixes**: `tool_exec` now logs `success`, `api_call` logs `cost` and `err`, new `agent.turn` and `agent.compact` events.
- **Skills adoption**: 5 skills from mattpocock/skills + SKILLS-CATALOG.md (125+ skills).
- **Files**: 11 changed, 9 new.

### Session 2026-06-18 (h32) — agentlog stderr bleed fix + currency symbol fix + log enrichment

- **Agentlog stderr bleed**: Removed `os.Stderr` from logger config — file and `io.Discard` only (stderr was leaking structured JSON into TUI output).
- **Currency symbol fix**: `Symbol()` now returns `"$"` when `ExchangeRate > 0`. Config `currency` field set to `"CNY"`.
- **Agent log enrichment**: Added `cache_miss` (bool), `err` (string), `truncated` (bool) fields to `api_call` events.
- **Files**: 3 changed.


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

### Session 2026-06-17 (h28) — 3 audit fixes: MCP bridge graceful shutdown, learn deferral, SQLite default backend

- **MCP bridge stdio graceful shutdown**: Added SIGINT/SIGTERM signal handler (was dying mid-message; HTTP mode already had graceful shutdown).
- **Learn pattern detection deferral**: Moved from `Observe()` (O(n²) every session) to `Patterns()`/`BuildReflectionPrompt()` (O(n) on read only).
- **Memory server default backend**: Changed from file to SQLite (write-amplification-safe; file still available via `--backend file`).
- **Build**: All 7 CLI binaries rebuilt. Build/vet/test/tsc/verify-session green.

### Session 2026-06-17 (h27) — upstream sync + 7 audit bug fixes

- **Upstream merged**: v1.9.x (ef1f38c, 6 commits) — tool-call name/args backfill on old-session replay, desktop perf (redundant session reload avoidance), Windows Authenticode signing CI, test additions.
- **1 conflict resolved**: desktop/app.go — accepted upstream's `OpenChannelSessionForTab` + `setTabReadOnly` (refactored `rebindTabToLoadedSessionPath`), removed Hermes duplicates.
- **7 audit bug fixes** from the June 2026 deep audit:
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

## v1.8.x (June 2026)

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

## v1.7.0+ (June 2026)

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

## v1.6.1 (June 2026)
- Upstream merge: sandbox nul, cold-resume, GSAP, compact sound

## v1.6.0 (June 2026) — Fork Foundation
- Vision support (image downscaling, detail knob)
- Built-in Time + Context7 MCP servers
- Configurable shell interpreter
- Notification sound system, token economy composer mode
- Desktop: time filter, custom fonts, status bar customization, Windows ARM64
- Crash capture (Go panics/breadcrumbs/group summaries)
- Local history + memory retrieval, Traditional Chinese (zh-TW) locale


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
