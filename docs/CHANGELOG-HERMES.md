# Reasonix-Hermes Changelog

Key milestones in the Hermes fork since June 2026.

## Unreleased

### Session 2026-08-24 — deferred subsystem sweep: memory durability, collab hygiene, orchestration bounds, learner caps

Worked the five items the 2026-08-07 session surfaced but deferred. Each was re-verified against current code first, and each fix carries a regression test confirmed to fail without it. 5 commits, no release cut.

- **fix(memoryserver) `05d381bd`** — the deferred item was logged as write amplification ("rewrites the whole table per write"), and that was real, but underneath it sat a data bug on the **default** backend: `sqliteStorage.Save` was upsert-only with no `DELETE`, so a row absent from the incoming slice stayed in the table. `Tidy()` purges expired low-importance entries from `ms.entries` and then persists — the purged row survived and came back on the next `load()`. The upsert-only form had been introduced deliberately to close a DELETE-all-then-reinsert race, so the fix keeps per-row upserts and deletes the stored ids missing from the incoming set inside the same transaction. Separately, `Save` was the only write path, so every `Retain` re-serialized and re-upserted every row: the `Storage` contract now has `Save` (full replace, used by `Tidy`, where decay touches nearly every entry anyway) plus `SaveDelta` (changed entries only). `fileStorage.SaveDelta` folds into a full document rewrite, since a JSON file cannot be updated in place.
- **fix(collab) `082acb6c`** — the hub's read loop had no read limit, no read deadline, and no keepalive, while writes already carried a 5s deadline. Any client could send an arbitrarily large frame and have it buffered in full, and a peer that vanished without closing left `ReadMessage` blocked forever, pinning a goroutine and a connection for the life of the process. Frames now cap at 1 MiB, a read deadline is refreshed by pongs, and pings go out well inside it. Keepalive timings are per-`Hub` fields defaulting to 60s/30s so tests drive a real ping/pong cycle without mutating a global; ping writes take `peer.mu` because `Broadcast` writes the same connection.
- **fix(collab) `62e9778a`** — `removePeer` unregistered the peer from every `peerSet` but left the now-empty `peerSet` in `h.sessions`. Nothing else removed those keys, so the map grew by one permanent entry per session ID ever subscribed. It now drops a session once its last peer leaves; readers already treated a missing session as no watchers.
- **fix(mcpbridge) `690992dc`** — two ways model output controlled our resource use. `orchestrate_task` spawned one goroutine per step up front with the step list coming from `parseSteps` over a model-written decomposition (the semaphore capped concurrent *work* at 3, not the goroutine count); and `runReasonix` built its 5-minute timeout from `context.Background()` rather than the caller's context, so the 15-minute orchestration budget only ever applied to the decomposition call — worst-case wall time was `ceil(N/3) x 5` minutes, unbounded in a model-chosen `N`. Now: decomposition capped at 10 steps (dropped steps reported), a fixed 3-worker pool, per-step timeouts derived from the orchestration context, and the queue stops being fed once that context is done, leaving unstarted steps marked as such.
- **fix(learn) `29a8ce15`** — `Load` assigned the persisted slices straight onto the `Learner`, so a sidecar written under a larger `max_observations` restored every entry; `New` clamps `maxObs` to 2000 and `Observe` evicts one entry per call, so the ring-buffer invariant was only restored gradually, and `detectPatterns`' pattern ceiling was bypassed entirely. Load now trims on the way in, keeping the newest observations, and applies the same ceiling — shared as `patternCap()` instead of duplicated.
- **Still open**: the fifth deferred item, **scheduled tasks reported as successful when they were silently dropped**, is diagnosed but not fixed. `SendCtx` (`internal/control/controller_turn.go`) discards its context and unconditionally returns nil; `Send` reaches `runGuarded`, which returns immediately when `c.running` is already true. So a cron task firing during an active turn vanishes while `runTask` records `Success: true, Summary: "task dispatched"`, and the scheduler's 10-minute timeout is honored by nothing. The fix needs `internal/control/controller.go` (split `runGuarded` into a shared `startGuarded` with a synchronous variant that returns `ErrTurnRunning`), which could not be edited in this session — an unrelated local tooling failure, not a code problem.

## v1.12.1 (August 2026)

### Session 2026-08-07 — harness audit: scheduler race, cache-guard gate, compressor + memory-server fixes

- **Repo audit (token-saving/caching focus)**: full read of the cache-first docs and code paths — the compressor, `cache_shape.go` miss diagnostics, `prune`, `compact`, provider cache accounting, tool-schema assembly, and the coordinator. Confirmed the prefix KV cache is the dominant cost lever and is well-executed (sorted + canonicalized tool schemas, turn-tail memory injection, separate planner/executor sessions to protect each prefix). The in-process compressor measures ~0.5% and overlaps the Headroom proxy.
- **fix(scheduler) `46083879`**: `fireDue` read `s.config.Tasks`/`s.nextRun` with no lock held while the desktop-reachable `AddTask`/`RemoveTask` mutate them under `s.mu` — a concurrent map read+write (a fatal Go panic) plus a torn slice, and `RemoveTask`'s reslice could shift a live `*Task` pointer. Snapshot due tasks by value under the lock and run them outside it; `runTask` takes `Task` by value; locked `Tasks()` and the startup count log. Added a `-race` regression test.
- **ci(cache-guard) `d48e18c5`**: `TestReleaseCacheHitGuard` already ran in CI and the release scripts, but nothing ever set `REASONIX_CACHE_GUARD_STRICT`, so it could only emit a warning annotation — never fail a build or block a release. Set STRICT in `ci.yml` (test + race jobs) and `scripts/cache-guard.sh` so a real prefix-cache regression now fails CI and blocks the release jobs that `needs: cache-guard`. Current margin is comfortable (92–95% vs 90% threshold, 0 low cases).
- **fix(compress) `32e45de7`**: the SHA-256 dedup replaced repeats with a `[content unchanged since turn N]` pointer, but `PruneStaleToolResults`/compaction can later elide/fold turn N → a dangling reference to content no longer in the window. Now emits a self-contained marker (`identical to earlier tool output this session … re-run the tool`) carrying a first-line summary; added a no-bloat guard (small repeats return verbatim); and fixed silent stats — the JSON path now increments `jsonFieldsStripped` + `BytesSaved`, and a cache hit counts `len(raw)-len(marker)` rather than the full raw length.
- **fix(memoryserver) `849cd693`**: `Retain` held `ms.mu` across the blocking `embedOne` HTTP call (bounded only by the 120s client timeout), freezing every concurrent Recall/Retain/SearchDense/Reflect. Compute the dense vector before taking the lock (matching `SearchDense`). Replaced the hand-rolled `sqrt`/`sqrtFallback` (Newton seeded at `z=x`, ~30% error on large norm-squared values) with `math.Sqrt` in `denseCosine`.
- **style `ca50e5c2`**: `gofmt -w` on 41 root-module files that were already gofmt-dirty on `HEAD` (the `ci.yml` gofmt gate was therefore red on `main`); whitespace only, no semantic changes.
- **npm release**: `hermes-npm-v1.12.1` cut and pushed → `release-hermes-npm.yml` published all 7 packages (`reasonix-hermes@1.12.1`, `dist-tags.latest`). Verified the 6-platform cross-compile locally (staged, no publish) before tagging, and verified live after (`npm view` → 1.12.1 on main + sub-package). Patch bump: everything since v1.12.0 is bug-fixes/CI/style/docs.
- **Also surfaced (from a custom-subsystem bug-sweep; NOT yet fixed)**: collab `h.sessions` grows unbounded (empty peer-sets never removed) + no WS read-limit/keepalive; scheduled tasks are dropped-but-reported-success when a turn is already running (`SendCtx` ignores its 10-min ctx); the SQLite memory backend rewrites the whole table on every write; mcpbridge `orchestrate_task` spawns unbounded goroutines from model output; `learn.Load` ignores its `maxObs` cap.
- **Total**: 5 commits (`46083879`, `d48e18c5`, `32e45de7`, `849cd693`, `ca50e5c2`), 1 npm release (7 packages, v1.12.1).

## v1.12.0 (July 2026)

### Session 2026-07-21 — npm release catch-up + upstream sync discontinued

- **Repo research**: full pass over docs, architecture, and codebase health. `go vet`/`go build` clean. Spot-checked ~8 findings from the historical `AUDIT-2026-07-15.md`/`docs/ACTION-PLAN.md` (WebSocket `CheckOrigin`, memory-server `SearchDense` lock, Feishu token timing-safety, `internal/memory` path traversal, hooks `session`/`session_id` mismatch) — all already fixed in current code; those audit docs are self-marked historical/stale.
- **Upstream sync discontinued**: found `main` had drifted 1,796 commits behind `upstream/main-v2` because `sync-upstream.yml` had failed every single scheduled run since 2026-06-23 (28/28) — the default `GITHUB_TOKEN` lacks the `workflows` permission needed to push a conflict-resolution branch touching `.github/workflows/*.yml`, so the automated "open a PR for manual resolution" fallback silently never worked. Decision: stop tracking upstream rather than fix the automation — Hermes now diverges intentionally. Disabled the workflow's cron schedule (kept `workflow_dispatch` for optional manual use). Updated README, AGENTS.md, and `docs/PROJECT.md` to reflect the new policy.
- **Bug fix — `TestFeishuMarkdownPostContent`**: was failing on `main`, and had been *mis-diagnosed* in the 2026-07-11 changelog entry as "pre-existing, unrelated to the dependency bumps." Actual root cause: `github.com/larksuite/oapi-sdk-go/v3`'s `SimpleMarkdownToPost` dropped its manual regex link-parsing (splitting into `text`+`a` post elements) somewhere between v3.9.4 and v3.9.5, in favor of wrapping raw markdown in a single `md` tag block — a direct behavior change from that dependency bump. Updated the test to assert current SDK behavior (commit `7bbd3b0e`).
- **Bug fix — local pre-commit date gate** (not committed; file is gitignored): `.reasonix/verify-session.sh` hardcoded `'2026-07'` as "a future/wrong date" — correct when written in June 2026, but permanently broken (blocking every commit) once the calendar reached July. Rewired to compute the threshold from `date +%Y-%m` instead of a fixed string.
- **npm release**: `hermes-npm-v1.12.0` tag cut and pushed, publishing the 163 commits accumulated since `hermes-npm-v1.11.0` (2026-06-21) — h58-h61 security/concurrency audit fixes, boot refactor, bot session pooling, coverage/fuzz hardening, an upstream merge (v1.11.1, 9c27591e), 4 dependency-update batches (Go root + desktop modules, desktop frontend npm, site astro), and the Feishu test fix above. Verified locally (staged 6-platform cross-compile via `node npm/build-hermes.mjs`) before tagging, then verified live (`npm view`, fresh `npm install reasonix-hermes` smoke test). No `v1.x` (CLI/Homebrew) or `desktop-v1.x` tags cut this session — npm only.
- **Total**: 2 commits (`7bbd3b0e` test fix, doc-sweep commit), 1 npm release (7 packages), 1 workflow disabled.

## v1.10.x (June 2026)

### Session 2026-06-26 (h60) — Deferred P2 items: boot refactor, bot pooling, coverage, fuzz

Completed all 5 deferred P2 items from h59. 8 files modified, 3 new files created.

- **P2-01**: `internal/boot/boot.go` decomposed into `builder` struct in `internal/boot/builder.go`. `Build()` is now a 40-line thin orchestrator that delegates to 11 phase methods (`loadConfig`, `buildProviders`, `buildPrompt`, `buildToolRegistry`, `buildPlugins`, `buildPermissions`, `buildSubagents`, `buildToolSurface`, `buildExecutor`, `buildLearner`, `assemble`). Hermes fingerprint test updated for new file split.
- **P2-02**: `internal/bot/gateway.go` — `SharedPluginPool` struct (reference-counted per workspace root) reuses one `plugin.Host` per root across all bot sessions. `getOrCreateSession` acquires a shared host before `boot.Build`; `evictIdleSessions` + `Stop` call `pool.Release` on session teardown. Eliminates duplicate MCP subprocess spawns when multiple users connect to the same workspace.
- **P2-05**: Coverage raised from baseline to: `internal/collab` 73.9% → 91.8% (EchoWSHandler, token auth, bad-JSON, steer, empty-sessionID tests); `cmd/reasonix-memoryserver` 37.1% → 76.8% (sqlite_storage CRUD, upsert, wildcard search, bad-JSON rows, Tidy boosts/purge, SetEmbedder, denseCosine, sqrt, newEmbeddingClient, Embed/embedOne, SearchDense, MCP handle dense-recall path, error paths). `internal/scheduler` already at 98% — no work needed.
- **P2-06**: Fuzz tests created: `pkg/mcputil/fuzz_test.go` (JSON-RPC parsing), `internal/tool/builtin/webfetch_fuzz_test.go` (IP bytes + URL validation), `internal/config/fuzz_test.go` (TOML decoding), `cmd/reasonix-memoryserver/fuzz_test.go` (SQL LIKE escaping), `internal/scheduler/fuzz_test.go` (cron parsing + NextAfter).
- **P2-13**: `.github/workflows/ci.yml` coverage threshold fixed (bash syntax error) and raised from 60% → 65% interim.

### Session 2026-06-26 (h59) — ACTION-PLAN full audit: P0 + P1 + P2 hardening

Complete pass through all 60 items in `docs/ACTION-PLAN.md` (P0–P2). 20 new fixes landed across 32 files; 40 items were already clean from prior sessions. 5 multi-day P2 items deferred (P2-01/02/05/06/13).

**P0+P1 fixes (8 items, commit 3c1bc909):**

- **P0-09**: `cmd/reasonix-hooks/main.go` — `doRetain`/`doReflect` return `error`; `main()` calls `os.Exit(1)` on failure → hook runner sees `DecisionWarn` not a silent pass.
- **P0-10**: `.github/workflows/ci-hermes.yml` — `test` + `race` jobs expanded from a subset to `go test ./...`, covering all 83 packages.
- **P1-01**: Collab WebSocket token auth — `?token=` query-param check via `subtle.ConstantTimeCompare` in `handleWS`; `Token` field added to `CollabConfig` + config renderer.
- **P1-02**: `internal/cli/upgrade.go` — `io.ReadAll` capped at 128 MiB via `io.LimitReader`.
- **P1-03**: Seven `0o644` → `0o600` call-sites in `desktop/` (write_mode, heartbeat, crash_pending, app, sessions, tabs).
- **P1-09**: `App.cleanupWg sync.WaitGroup` tracks `delayedDesktopSessionCleanup`/`delayedDesktopSessionTrash` goroutines; `shutdown()` waits before teardown.
- **P1-10**: mcpbridge SIGTERM handler — `os.Exit(0)` → `os.Stdin.Close()` for graceful stdio shutdown through defers.
- **P1-14**: Compressor LRU eviction — evicts oldest-turn entry when `len(cache) >= maxCache` (512).

**P2 fixes (12 items, commit b44c5aaa):**

- **P2-04**: `--pprof <addr>` flag added to `cmd/reasonix-mcpbridge` and `cmd/reasonix-memoryserver`; starts `http.ListenAndServe` with `net/http/pprof` registered on `DefaultServeMux`.
- **P2-07**: Mesh peer init TTL — replaced package-level `var initializedPeers sync.Map` (never expired, shared across instances) with per-`Peer` `initialized bool` + `initializedAt time.Time` + `initMu sync.Mutex`; re-handshakes after 5-minute TTL or on connection error.
- **P2-09**: `MemoryStore.Recall` read-path — access-count boosts and importance bumps deferred to `Tidy()` via `pendingBoosts map[string]bool`; eliminates `save()` call on every search query.
- **P2-10**: `cmd/reasonix-mcpbridge/main.go` — `log.Fatal`/`log.Println`/`log.SetPrefix` → `slog.Info`/`slog.Error`.
- **P2-11**: `.golangci.yml` — `gosec` and `gocritic` enabled with targeted exclusions (G104/G204/G304/G306/G404 for gosec; `commentFormatting`/`hugeParam`/`whyNoLint` for gocritic).
- **P2-16**: Helm chart — `config_version` 1→2, pod+container `securityContext` (`runAsNonRoot`, `readOnlyRootFilesystem`, `allowPrivilegeEscalation: false`, `drop: ALL`), collab port conditional fixed from `if .Values.service.type` → `if .Values.components.bot.enabled`.
- **P2-17**: `serve/auth.go` — `sessionKeyForPasswordHash` + `generateToken` return `(T, error)` instead of panicking; `newAuthGate` + `serve.New` propagate the error; `desktop/app.go` `mediaTokenStore.create` logs+returns empty on rand failure; `desktop/updater_app.go` `os.Exit(0)` removed.
- **P2-18**: `internal/installsource/apply.go:225` — `%v` → `%w` for rollback error wrapping.
- **P2-20**: `internal/netclient/netclient.go` — `DefaultClient()` returns a shared `*http.Client` via `sync.Once` instead of allocating per call.

**Already clean (P2 items verified):** P2-03 (sqlite default), P2-08 (WS write deadlines), P2-12 (govulncheck blocking), P2-14 (multi-arch Docker), P2-15 (Dockerfile HEALTHCHECK), P2-19 (collab bind-error propagation).

- **Build**: `go build ./...` + `go vet ./...` clean across both commits.

### Session 2026-06-21 (h54) — agent-reach MCP + headroom proxy + taste-skill + npm v1.11.0 + upstream merge

- **agent-reach e2e**: Verified sandbox whitelist fix works — agent-reach v1.5.0 runs inside bash=enforce sandbox from reasonix CLI sessions. `~/.agent-reach` + `~/.local/share` write access confirmed.
- **agent-reach MCP server**: `.reasonix/scripts/agent-reach-mcp` (148 lines), 3 tools: `get_status` (doctor report, 7/13 channels), `read_web` (Jina Reader → markdown), `youtube_subtitles` (yt-dlp extraction). Registered as `[[plugins]]` in reasonix.toml. E2E verified.
- **headroom proxy**: v0.26.0 wired end-to-end. Proxy running on :8787, routing reasonix → DeepSeek. 2 new providers (deepseek-headroom, deepseek-flash-headroom). CLI TUI status bar +◈↓N% chip, desktop StatusBar +HeadroomGaugeCompact, Wails binding + live data hook. Docs: HEADROOM.md (304 lines), HERMES-GUIDE §16.28, AGENTS.md row.
- **headroom + markitdown MCP**: Both verified end-to-end. `headroom_stats`, `headroom_compress`, `markitdown convert_to_markdown` all functional.
- **Taste-skill desktop audit**: Found Hermes theme style was a ghost — registered in `theme.ts` but had zero CSS rules. Added full `:root[data-theme-style="hermes"]` CSS block (~190 lines): dark mode (gold accent #d4a853, warm surfaces), light mode (parchment, gold #b88c3c), auto/@media variant. Desktop rebuilt.
- **Doc-sweep**: 6 stale claims fixed across 5 docs (README, HERMES-GUIDE, AGENTS.md, REASONIX.md, index.html). HERMES-GUIDE +§16.27 Agent Reach MCP + §16.28 Headroom Proxy.
- **npm v1.11.0**: `v1.11.0` + `hermes-npm-v1.11.0` tags pushed. 6-platform binaries cross-compiled and staged. CI publishes to npm via OIDC.
- **Upstream**: merged 043e6183 (10 commits: Creation tool flow + session trash fix). 1 conflict (MarkdownRenderer.tsx, accepted upstream superset). 31st sync. Fingerprint guards 34/34 pass.
- **Build**: All 9 binaries rebuilt. Full test suite (76 packages) + desktop tests + tsc clean.

### Session 2026-06-22 (h55) — upstream sync (28 commits, 32nd sync)

- **Upstream**: merged f2a475a2 (28 commits) — scroll reliability (f7a47d4a), image-path containment (1e7a4f10/d68ed3b8), grace round on tool-call budget (c29a39cd), crash stats 30d (05d2d209), /reload-cmd (4da0a184), lazy MCP schema cache (eb9200f5), transcript reveal fix (48f7af9b), model-switcher compact (d692e25e), session sidecars with trash (47166a37), workspace vision images (bb4dbca0).
- **Conflicts**: 10 files — agent.go (graceRound + learner outcomes), refs.go (path containment), go.mod/go.sum, package.json (d3), useScrollManager.ts, lockfiles, pages.yml, site files.
- **refs.go changes**: Removed Hermes image-fallthrough from classifyRef/detectRefsMode — upstream's stricter containment handles image security. visionRefImageDataURL + visionFileImageDataURL cover workspace + attachment images.
- **Build**: 9 binaries rebuilt, go build/vet/test clean, fingerprint guards 34/34 pass.
- **Doc-sweep**: 22 stale claims fixed across 10 docs (SPEC, README×2, HERMES-GUIDE, SKILLS-CATALOG, BOT_GUIDE×2, CONFIG_PATHS.zh-CN, GUIDE×2, MIGRATING).

### Session 2026-06-22 (h56) — upstream sync (28 commits, 33rd sync) + headroom daemon + dead code

- **Upstream**: merged 7aa97d31 (28 commits) — desktop MCP headers (e19fbdcc), remote MCP header support, rewind/plan prompt consolidation (87b40dfe), incremental project config writes (2419ca27 + b61e4a77), session deletion perf (51eea606 + 8bdc4d35), skill root dedupe (a64a6a8a + 05cf7a8b), ACP config switch hardening (6a84a4d3..23280359), ask-user decision integration (658bc7ce), short choice planner context (6ad1df07), creation tool card scrollbar (917e036e), CLI session migration for desktop (b61e4a77 + 01613306), post-merge bot review fixes (71068eab).
- **Conflicts**: 4 files — package.json (hotbar-guard.test.ts + d3/@codemirror/biome deps), useController.ts (console.warn), boot.go (UserDecisionPolicy + languagePolicy), config.go (DefaultSystemPrompt shortened, UserDecisionPolicy constant).
- **Headroom daemon**: launchd plist at `.reasonix/headroom/com.headroom.proxy.plist`, loaded and verified healthy. Proxy v0.26.0 auto-starts on login. docs/HEADROOM.md updated with corrected flags (dropped broken `--backend anyllm`), verify/manage commands, plist reference.
- **Dead code**: Removed `visionImageDataURLFromPath` + `readImageFile` (48 lines, `internal/control/attachments.go`). Zero callers — orphaned by upstream os.OpenRoot path containment in h55.
- **Markitdown verified**: PDF/DOCX/XLSX all convert correctly via `file://` URIs (MCP server).
- **Doc-sweep**: 1 stale claim fixed (PROJECT.md upstream commit 051239b6→7aa97d31).
- **Build**: All 9 binaries rebuilt. Full test suite (76 packages) + go vet + tsc + fingerprint guards 34/34 all pass.

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
