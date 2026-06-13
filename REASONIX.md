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

- **Upstream synced**: `v1.7.0` merged (commit f6d8cce, 2026-07-14). 55 commits total (47 prior + 8 new). 4 conflicts resolved (controller imports, example.toml alignment, bridge.ts Hermes bindings, GUIDE.md slash commands).
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

- [ ] **Sandbox Windows support** — design doc ready at `docs/WINDOWS-SANDBOX-DESIGN.md`. Implementation: ~350 lines in `internal/sandbox/appcontainer_windows.go`.
- [ ] **Approve duplicate fix** — squash the remaining "Approved." + "No pending action" double-fire from session queue replay.
- [ ] **Doc cleanup** — `HERMES-GUIDE.md` needs Discord bot section; `BOT_GUIDE.md` needs Discord instructions.
- [ ] **Multi-provider expansion** — add Codex/MiniMax/GLM providers (roach-code model).
