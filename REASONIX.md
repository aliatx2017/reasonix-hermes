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

- **Upstream synced**: `v1.6.0` merged (commit 6cf4a95, 2026-07-13). 40 commits total, 6 conflicts in latest sync resolved.
- **Key v1.6.0 additions**: vision support (image downscaling + detail knob), built-in Time + Context7 MCP servers, configurable shell interpreter (`[tools.shell]`), notification sound system, token economy composer mode, desktop time filter + custom fonts + status bar customization + Windows ARM64, crash capture (Go panics/breadcrumbs/group summaries), lightweight local history + memory retrieval, Traditional Chinese (zh-TW) locale, updater resilience, agent fixes (decline-ask guard, compaction bounds), desktop hooks UI.
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
- **Session 2026-07-12** (Hermes enrichment wave):
  - **Desktop**: Hermes accent theme ("hermes" — caduceus gold #d4a853), live data push (Wails event loop replaces 5s polling), token sparkline bar chart (ring buffer on Agent → Controller → Wails binding), compaction timeline panel, checkpoint file preview (Store.FileSnaps → expandable per-turn file list), Write Mode (Go filesystem bindings + React split-pane editor + live markdown preview), memory fact graph (clustered by type with color badges)
  - **Desktop config**: `reasonix.example.toml` now documents 24 sections covering all v1.6.0 + Hermes keys (348 lines, taplo-clean)
  - **Remote sandbox**: 10 httptest-based e2e tests covering commandRemote, Spec.remote(), Run() — all success/error/timeout paths
  - **Go internals**: `WorkspaceSlug`/`slugify` now relativize paths to `$HOME` and replace spaces with dashes (fixed `-Users-alex.maksimchuk-Library-Application Support-reasonix-global-workspace` → `$HOME-Library-Application-Support-reasonix-global-workspace`)
  - **CLI TUI**: Pinned ⚚ REASONIX-HERMES header (never scrolls away), turns/msgs/goal/mem moved to bottom status line, `/stats` panel enhanced with Unicode token sparkline (▁▂▃▄▅▆▇█), compaction timeline, memory facts list, goal progress. Banner padding math fixed (no more negative Repeat panics).
  - **VS Code extension fork**: Deleted — not pursuing.
- **VS Code extension fork**: Removed from plans.

## Next session — ideas & follow-ups

- [ ] **Upstream sync check** — `git fetch upstream && git log HEAD..upstream/main-v2 --oneline`. Merge if any new commits.
- [ ] **Discord bot integration test** — verify bot still starts and responds after gateway changes from upstream merge.
- [ ] **Write Mode FIM + CodeMirror** — the Markdown editor has a textarea; integrate CodeMirror 6 or Monaco for syntax highlighting and FIM completions via DeepSeek API.
- [ ] **Write Mode Hindsight injection** — inject relevant memory facts into the Write Mode context.
- [ ] **Cross-session memory graph D3** — upgrade MemoryFactGraph from clustered badges to a D3/vis.js force-directed graph with vector similarity links.
- [ ] **Write Mode panel integration** — add WriteMode component to the desktop workspace sidebar as an actual panel (currently a standalone component, not wired into the layout).
- [ ] **Checkpoint actual diffs** — currently shows pre-edit content; add unified diff against current file state.
- [ ] **Desktop Hermes accent in tray/title bar** — apply the Hermes gold accent to window chrome on supported platforms.
- [ ] **CLI TUI `/write` command** — open Write Mode as a separate Bubble Tea TUI session.
- [ ] **Sandbox Windows support** — investigate Windows Job Objects or Hyper-V containers as sandbox backend.
- [ ] **CLI terminal theme detection** — auto-detect terminal background and switch theme (currently no-op on Windows).
