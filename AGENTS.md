# Reasonix Hermes

Customized Reasonix AI coding agent based on upstream [esengine/deepseek-reasonix](https://github.com/esengine/deepseek-reasonix) with added MCP bridges, Discord bot, skills hub, and community tooling.

## Syncing with Upstream

**Discontinued as of 2026-07-21.** Hermes has intentionally diverged from
upstream — syncing is no longer a goal. The daily `sync-upstream.yml`
automation is disabled (schedule trigger removed, `workflow_dispatch` kept
for optional one-off use). It had in fact been silently failing every run
since 2026-06-23 (28 consecutive failures, `main` ~1,796 commits behind
`upstream/main-v2` by the time this was noticed) because the default
`GITHUB_TOKEN` lacks the `workflows` permission needed to push a
conflict-resolution branch that touches `.github/workflows/*.yml`.

Manual (only if you ever want to pull specific upstream fixes again):

```bash
git fetch upstream
git merge upstream/main-v2     # merge upstream changes
# resolve any conflicts in our custom files
git push origin main
```

## Commands

```bash
# Build the CLI
go build -o bin/reasonix ./cmd/reasonix

# Build our custom binaries
go build -o bin/reasonix-mcpbridge ./cmd/reasonix-mcpbridge
go build -o bin/reasonix-memoryserver ./cmd/reasonix-memoryserver
go build -o bin/reasonix-bot ./bot
go build -o bin/reasonix-hooks ./cmd/reasonix-hooks

# Install skills via upstream install_source
reasonix install-source install --source https://github.com/aliatx2017/reasonix-hermes/tree/main/skills-hub/skills

# Build the Desktop app (Wails + React 19 + frontend)
cd desktop/frontend && npm install && cd ../..
cd desktop && wails build -o ../bin/reasonix-desktop

# Build + vet everything
go build ./...
go vet ./...

# Run tests (83 Go packages, 76 test packages pass)
go test ./cmd/... ./pkg/... ./internal/bot/...

# Run the CLI
./bin/reasonix chat
./bin/reasonix run "task description"
./bin/reasonix setup

# Run the Discord bot
export DISCORD_BOT_TOKEN="..." DISCORD_SERVER_ID="..."
./bin/reasonix-bot

# Desktop dev mode
cd desktop/frontend && npm install && cd ../..
cd desktop && wails dev
```

## Architecture

```
cmd/reasonix/          CLI entry point (delegates to internal/cli/)
cmd/reasonix-pr-review/ PR review CLI for GitHub Actions
internal/              Reasonix engine (56 packages plus testutil)
  agent/               Core agent loop, compaction, subagents
  agentlog/            Structured operational logging (slog → stderr + AGENT_LOG)
  boot/                Controller assembly, tool wiring
  bot/                 Multi-platform bot gateway (Discord/QQ/Feishu/WeChat/Telegram)
  collab/              Live collaboration WebSocket hub
  compress/            Tool output token compressor
  config/              TOML config loader + model fallback
  constitution/        Structured project invariants (.reasonix/constitution.json)
  control/             Transport-agnostic Controller
  eval/                Session comparison and evaluation (Jaccard, structural diff)
  learn/               Self-improving skill loops (pattern detection)
  mesh/                Agent-to-agent MCP mesh (delegate, broadcast, council, judge)
  orchestrate/         Multi-agent orchestration (chain, pair, CI-fix workflows)
  permission/          Tool-call permission gating
  plugin/              MCP client (stdio, HTTP, SSE)
  provider/            LLM providers (Anthropic, OpenAI/DeepSeek)
  publish/             Session transcript export (HTML/JSON)
  scheduler/           Cron-driven automated agent tasks
  skill/               Built-in skills registry
  tool/                Built-in tools (bash, read, write, edit, lsp_*, etc.)
  ...
pkg/                   ── Our custom additions ──
  httputil/            Shared Bearer auth middleware
  mcputil/             Shared MCP types and server helpers
cmd/                   ── Our custom binaries ──
  reasonix-mcpbridge/  MCP bridge server (6 tools)
  reasonix-memoryserver/ Hindsight MCP server (3 tools)
  reasonix-pr-review/  PR review CLI
bot/                   Discord bot gateway
cmd/reasonix-hooks/    Native Go hook runner
deploy/                Helm chart + docker-compose for one-click deploy
desktop/               Wails v2 desktop app + React 19 frontend
skills-hub/            17-skill community registry + static catalog site
```

## Our Customizations

| Layer | What | Why |
|-------|------|-----|
| `pkg/mcputil/` + `pkg/httputil/` | Shared Go libraries | Bearer auth middleware + MCP types/helpers |
| `cmd/reasonix-mcpbridge/` | MCP bridge server (6 tools) | Expose Reasonix to Claude Code/Codex via MCP |
| `cmd/reasonix-memoryserver/` | Hindsight memory (3 tools, SQLite, TTL, vector) | Cross-session persistent memory with dense+sparse vector search |
| `internal/acp/` | Agent Client Protocol | stdio JSON-RPC 2.0 adapter — editor/IDE integrations connect via ACP |
| `bot/` + `internal/bot/discord/` | Discord bot (+ /goal + /model) | Discord integration (upstream has Feishu/WeChat/QQ only) |
| `internal/bot/telegram/` | Telegram bot adapter | Long-polling Telegram integration via go-telegram-bot-api/v5 |
| `internal/bot/line/` | LINE bot adapter | Webhook-based LINE integration via line-bot-sdk-go/v8 |
| `internal/bot/slack/` | Slack bot adapter | Socket Mode Slack integration via slack-go/slack |
| `cmd/reasonix-hooks/` | Native Go hook runner | Zero-dependency binary for PreToolUse/Stop hooks |
| `cmd/learner-live-test/` | Learner live e2e binary | 5-turn real-LLM learner validation with tool-call tracing |
| `skills-hub/` | 17 community skills + catalog site | Curated skill registry with frontmatter playbooks |
| `internal/learn/` | Self-improving skill loops | Observes agent patterns, detects repeated sequences, generates skills |
| `internal/mesh/` | Agent-to-agent MCP mesh | Peer delegation, broadcast, council + judge (structured Fusion Router-inspired analysis) |
| `internal/collab/` | Live collaboration hub | WebSocket session sharing between Reasonix instances |
| `internal/compress/` | Tool output token compressor | SHA-256 cache, repeated-line collapsing, JSON minification |
| `internal/scheduler/` | Cron-driven task scheduler | Automated agent runs at scheduled times |
| `internal/publish/` | Session transcript export | Self-contained HTML + JSON export |
| `internal/marketplace/` | Community skill registry | agentskills.io-compatible + LobeHub marketplace sync (850+ agents) |
| `internal/provider/ollamacloud/` | Ollama Cloud provider | 42 models via ollama.com/v1 OpenAI-compatible API |
| `internal/constitution/` | Project invariants | Structured principles/constraints/rules from .reasonix/constitution.json |
| `internal/e2e/` | Regression testing harness | Replay-based session testing (inputs, turns, assertions) |
| `internal/eval/` | Session comparison tool | Structural diff, Jaccard similarity, CLI + desktop binding |
| `internal/orchestrate/` | Multi-agent orchestration | Chain, pair, and CI-fix workflows (6 tests) |
| `cmd/reasonix-pr-review/` | PR review CLI | Fetches PR diff, runs review with 6-dimension prompt |
| `npm/hermes/` | npm package | One-line install: `npm i -g reasonix-hermes` (7 sub-packages) |
| `deploy/` | Helm chart + docker-compose | One-command deploy to K8s or $5 VPS |
| `desktop/` | Wails v2 desktop app | React 19 frontend + Go kernel; Hermes dashboard; live data push |
| `internal/agentlog/` | Operational agent logging | Structured JSON logging via slog; api_call + tool_exec telemetry |
| `internal/billing/exchange.go` | Live exchange rate | CNY→USD live fetch from exchangerate-api.com; [billing] config |
| `.reasonix/skills/research/` (+2 siblings) | Combined research pipeline | `/research <topic>` 5-phase auto-chaining: outline → approve → batch subagents → report → Discord publish (SearXNG + Crawl4AI) |
| `.reasonix/scripts/discord-publish.sh` | Discord publish script | Webhook-based script posts report.md to Discord channel; auto-loaded from `.reasonix/.discord-webhook` |
| `.reasonix/scripts/agent-reach-mcp` | Agent Reach MCP server | Python MCP wrapper: get_status, read_web (Jina), youtube_subtitles (yt-dlp) — registered as [[plugins]] |
| `headroom proxy` (v0.26.0) | LLM context optimization proxy | Transparent proxy: compression, SHA-256 cache, prefix-freeze, 20-92% token savings — see docs/HEADROOM.md |
| `desktop/frontend/src/styles.css` (hermes) | Hermes theme CSS | Gold-accented theme style (#d4a853), dark + light + auto variants — taste-skill audit fix |
| `internal/cli/eval.go` | `/eval` slash command | Eval-driven development: define, check, report, list, clean subcommands |
| `internal/cli/learn.go` | `/learn` slash command | Learner pattern detection UI: patterns + trajectories subcommands, Controller-wired |
| `reasonix-hermes.json` | Install source manifest | `reasonix install-source install --source ...` |
| `.github/workflows/ci-hermes.yml` | Supplementary CI | Desktop frontend build + Hermes package tests in CI |
| `.github/workflows/pr-review.yml` | PR review action | Auto-reviews PRs with Reasonix |
| `.github/workflows/release-hermes-npm.yml` | npm release pipeline | Cross-compiles 6 platforms → npm publish |

## Docs

- **[Changelog](docs/CHANGELOG-HERMES.md)** — Hermes fork milestones, expansion packs, bot platforms
- **[Ecosystem Reference](reasonix-deepseek-ecosystem-2026.md)** — full landscape: MCP bridges, skills, desktop, IDE, forks, cost model, protocols, use cases

## Notes

- Upstream remote: `https://github.com/esengine/deepseek-reasonix.git` (branch `main-v2`) — kept for reference only
- **Upstream sync**: discontinued 2026-07-21. Last successful merge: v1.11.1 (9c27591e, 2026-06-23), 34 syncs total. Intentionally diverging going forward; `sync-upstream.yml` schedule disabled.
- Our fork: `https://github.com/aliatx2017/reasonix-hermes.git` (branch `main`)
- To pull upstream updates (opt-in, not routine): `git fetch upstream && git merge upstream/main-v2`
- `reasonix.toml` is gitignored (upstream convention) — never commit secrets
- Discord bot uses `github.com/bwmarrin/discordgo` (added to go.mod)
- Discord bot must use `control.Controller` like every other frontend — not inline chat history
- **Tests**: 83 Go packages total, 76 test packages pass. ~2,250+ test cases across all packages. `go test ./...`
- **New packages (custom)**: `internal/acp/` (Agent Client Protocol), `internal/learn/` (self-improving skill loops), `internal/mesh/` (agent-to-agent MCP mesh), `internal/collab/` (live collaboration WebSocket hub), `internal/compress/` (tool output token compressor), `internal/scheduler/` (cron-driven tasks), `internal/publish/` (session transcript export), `internal/bot/telegram/`, `internal/bot/line/`, `internal/bot/slack/` (multi-platform bot adapters), `internal/e2e/` (regression testing harness), `internal/marketplace/` (community skill registry + LobeHub sync), `internal/provider/ollamacloud/` (Ollama Cloud API provider), `internal/constitution/` (project invariants), `internal/agentlog/` (operational JSON logging with log rotation), `internal/billing/` (live CNY→USD exchange rate), `cmd/reasonix-pr-review/` (PR review CLI), `cmd/e2ebench/` (e2e benchmark tool), `cmd/learner-live-test/` (learner e2e validation).
- **CodeWhale features** (10/10 done, 2026-06-04): Shell env hooks, parallel sub-agent batch dispatch, completion sound, harness profiles, constitution system, workshop sidecar, desktop hotbar, external sandbox, Nix flake, Dockerfile.
- **CI & tooling** (2026-06-06): `biome format` check on desktop frontend (105 files), `wails build` CI job, `taplo` TOML lint (CI + pre-commit hook), Go `go-version-file: go.mod` (toolchain 1.26.4), 7-job Hermes CI pipeline all-green.
- **Bug fixes** (2026-06-06): duplicate `price` key in `reasonix.example.toml`, data race in `mockProvider.Stream()`, `TestSaveToScopes` cross-platform fix.
- **New packages**: `internal/constitution/` (structured project invariants from `.reasonix/constitution.json`)
- **New files**: `flake.nix`, `Dockerfile`, `.dockerignore`, `internal/sandbox/remote.go`
- **Config additions**: `[notifications].sound`, `active_profile`, `[profiles.<name>]` blocks, `[sandbox].remote_sandbox_url`, `[sandbox].remote_sandbox_token`
- **2026-06-12 session** — 13 features shipped: Hermes accent theme, live data push (Wails events), token sparkline chart, compaction timeline, checkpoint file preview, Write Mode (Go fs bindings + React editor), memory fact graph, reasonix.example.toml full update, remote sandbox e2e tests, workspace slug fix ($HOME relativization), CLI TUI enhancements (pinned banner, bottom counters, /stats sparkline+compaction+memory+goal). Built CLI (26MB) + desktop (33MB). VS Code fork removed.
- **2026-06-14 session** (Ollama Cloud + aux models + 4 features):
  - **Ollama Cloud provider**: New `ollamacloud` provider kind, 42 models, OpenAI-compatible at ollama.com/v1. `reasoning` field name fix in openai provider.
  - **Auxiliary model routing**: `[agent.auxiliary]` config block — compression/vision/web_extract each take their own provider+model. Agent routes compaction summarizer through compressionProv, vision requests through visionProv when images present. Tested with `deepseek-v4-flash` (compression) + `gemini-3-flash-preview` (vision). **Vision pipeline hardened** (2026-06-15 h14): `classifyRef` now detects arbitrary filesystem images, `visionImageDataURLFromPath` reads non-attachment images, workspace-path fallthrough for images, and `properties` defaulted to `{}` in empty-object schemas for Gemini/Ollama Cloud compatibility.
  - **Desktop collab panel**: Go collab Hub + CollabDashboard binding, React CollabPanel (live badge, watcher count, session list), integrated into live data push + polling.
  - **Multi-model council UI**: Controller mesh integration (SetMesh/Council/MeshStatus), boot.go mesh creation, CLI `/council` command, desktop CouncilPanel.
  - **E2E test harness**: New `internal/e2e/` — Harness, SessionInputs, SessionTools, Analyze, AssertTools/Turns, RunAll. 7 tests.
  - **Skill marketplace**: Community registry (12 skills, agentskills.io-compatible SKILL.md format), `internal/marketplace/` Go package, CLI `reasonix marketplace` command, desktop MarketplacePanel with tag filters + install buttons.
  - **LobeHub marketplace API integration**: Full M2M OAuth2 client at `internal/marketplace/lobehub_client.go` (stdlib-only HS256 JWT), auto-registration, paginated sync from 360k+ community skills at `market.lobehub.com`, desktop "Sync from LobeHub" button, CLI `reasonix marketplace sync`, `[marketplace.lobehub]` config section. Zero new dependencies.
  - **LAN skills**: 4 project skills (`searxng-local`, `crawl4ai-local`, `google-maps-scraper`, `last30days`) for local network services at 192.168.1.214.
  - **Total**: 30+ files changed, 3 new Go packages, 4 new React components, 80+ tests. All binaries rebuilt. Upstream synced to ed07684.
- **2026-06-26 session (h60)** (deferred P2 items: boot refactor, bot pooling, coverage, fuzz):
  - **P2-01 boot decomposition**: `internal/boot/builder.go` (new, 1000+ lines) — `builder` struct with 11 phase methods; `Build()` reduced to 40-line orchestrator. Hermes fingerprint test updated for file split.
  - **P2-02 bot session pooling**: `SharedPluginPool` in `internal/bot/gateway.go` — ref-counted `plugin.Host` per workspace root. Eliminates duplicate MCP subprocess spawns for concurrent IM sessions.
  - **P2-05 coverage**: `internal/collab` 73.9%→91.8% (EchoWSHandler, token auth, steer, bad-JSON tests); `cmd/reasonix-memoryserver` 37.1%→76.8% (sqlite CRUD, Tidy, embedding client, SearchDense, MCP handle). `internal/scheduler` already 98%.
  - **P2-06 fuzz tests**: 5 fuzz test files — mcputil (JSON-RPC), webfetch (IP+URL), config (TOML), memoryserver (SQL LIKE), scheduler (cron).
  - **P2-13 CI coverage gate**: Fixed bash syntax error; threshold 60%→65%.
  - **Total**: 11 files changed, 1 new Go file (`builder.go`), 3 new test files. All 83 packages build+pass.
- **2026-06-14 session (h6)** (banner + version + savings stats):
  - **Dynamic version**: `resolveVersion()` in `style.go` — uses ldflags first, then `git describe --tags --match 'v*'`, falls back to `"v1.10.0"` only as last resort. Pinned banner shows live git tag in dev builds.
  - **Diamond Wing logo**: `◆` replaces `⚚` caduceus in pinned header + session banner, gold accent preserved.
  - **Savings stats in status bar**: `aux↓N` (tokens saved via auxiliary providers) + `sqz↓N` (bytes saved by compressor). Atomic counters in compressor, exposed through agent → controller → TUI.
  - **Total**: 6 files changed, +91/-12 lines. Committed f0ba51b.
- **2026-07-11 session** (Dependabot PR batch merge):
  - **Merged 4 open Dependabot PRs into main**: astro 7.0.4→7.0.6 (`/site`), Go root module 6 updates (charm bubbles/bubbletea/lipgloss, larksuite oapi-sdk-go, line-bot-sdk-go, golang.org/x/text), Go `/desktop` module 2 updates (wails + deps), desktop frontend npm 7 updates (React 19.2.7, vite 8.1.3, typescript 6.0.3, biome 2.5.2, codemirror, katex, gsap, etc.).
  - Verified via `go build`/`go vet`/`go test` (root + desktop modules), `npm run build` (site/astro), `pnpm build`/`pnpm test` (desktop frontend) — all green; confirmed the handful of failures present (theme-auto-background, bundle-contract, send-failed, settings-refresh-snapshot, app-chrome-tabs frontend suites + `TestFeishuMarkdownPostContent` in `internal/bot/feishu`) are pre-existing on unmodified `main`, unrelated to the bumps. **Correction (2026-07-21): this was wrong for `TestFeishuMarkdownPostContent` — see below.**
  - `go mod tidy` fixup for `desktop/go.mod`/`go.sum` post-bump (small indirect-dep drift).
  - PRs #22–#25 closed as merged on GitHub; local + remote `main` fast-forwarded to `0b304434`.
  - **Total**: 5 commits, 8 files changed (+273/-438 lines).
- **2026-07-21 session** (npm release catch-up + upstream sync policy decision):
  - **Research**: full repo/docs/codebase audit. Found `main` had drifted 1,796 commits behind `upstream/main-v2` because `sync-upstream.yml` had been failing every single scheduled run since 2026-06-23 (28/28) — the default `GITHUB_TOKEN` lacks the `workflows` permission needed to push a conflict-resolution branch touching `.github/workflows/*.yml`, so the "open a PR for manual resolution" fallback silently never worked.
  - **Decision**: stop tracking upstream. Disabled `sync-upstream.yml`'s cron schedule (kept `workflow_dispatch` for optional manual use) instead of fixing the permission — divergence is now intentional. See "Syncing with Upstream" section above.
  - **npm release**: cut and published `hermes-npm-v1.12.0` — 163 commits since the last npm publish (`hermes-npm-v1.11.0`, 2026-06-21: upstream v1.11.1 merge, h58–h61 audit hardening, boot refactor + bot pooling, coverage/fuzz, 4 dependency-update batches). Verified locally (staged 6-platform cross-compile via `node npm/build-hermes.mjs`) before tagging; verified live after (`npm view`, fresh `npm install reasonix-hermes` smoke test — reports `hermes-npm-v1.12.0`). No `v1.x`/`desktop-v1.x` tags cut — npm only.
  - **Bug fix — `TestFeishuMarkdownPostContent`**: this was NOT pre-existing/unrelated to the 2026-07-11 dependency bump as previously logged (see correction above). Root cause: `github.com/larksuite/oapi-sdk-go/v3`'s `SimpleMarkdownToPost` dropped its manual regex link-parsing (text+`a` elements) somewhere between v3.9.4→v3.9.5 in favor of wrapping raw markdown in a single `md` tag. Updated the test to assert the current SDK behavior. Fixed in `7bbd3b0e`, shipped in `1.12.0`.
  - **Bug fix — local pre-commit date gate**: `.reasonix/verify-session.sh` (gitignored, local-only) hardcoded `'2026-07'` as "the future" — true when written in June, permanently broken (blocks every commit) once the calendar reached July. Rewired to compute the threshold from `date +%Y-%m` instead. Not committed (file is gitignored by design) — noted here for anyone else running the same local hook setup.
  - Spot-checked ~8 findings from the historical `AUDIT-2026-07-15.md`/`docs/ACTION-PLAN.md` (WebSocket `CheckOrigin`, memory-server `SearchDense` lock, Feishu token timing-safety, `internal/memory` path traversal, hooks `session`/`session_id` mismatch) — all already fixed in current code; those docs are self-marked historical/stale.
  - **Total**: 2 commits (`7bbd3b0e` fix + changelog, tag `hermes-npm-v1.12.0`), 1 npm release (7 packages).
- **2026-08-07 session** (harness audit + fixes + npm v1.12.1):
  - **Audit** (token-saving/caching focus): traced the cache-first paths (compressor, `cache_shape.go`, prune, compaction, provider cache accounting, tool-schema assembly, coordinator). Conclusion: the prefix KV cache is the dominant lever and is well-built; the in-process `internal/compress` compressor measures ~0.5% and overlaps the Headroom proxy.
  - **fix(scheduler) `46083879`**: crash-class data race — `fireDue` read `s.config.Tasks`/`s.nextRun` unlocked while UI-driven `AddTask`/`RemoveTask` mutate them under `s.mu` (concurrent map read+write = fatal panic; torn slice; `RemoveTask` reslice shifting a live `*Task`). Snapshot due tasks by value under the lock, run outside it; `runTask` by value; locked `Tasks()` + startup count. Added a `-race` test.
  - **ci(cache-guard) `d48e18c5`**: nothing set `REASONIX_CACHE_GUARD_STRICT`, so the existing prefix-cache guard could only warn, never fail. Set STRICT in `ci.yml` (test + race) and `scripts/cache-guard.sh` → a real regression now fails CI and blocks the release jobs that `needs: cache-guard`. Margin 92–95% vs 90%, 0 low cases.
  - **fix(compress) `32e45de7`**: replaced the turn-anchored dedup marker (`content unchanged since turn N`, which prune/compaction can turn into a dangling ref) with a self-contained "identical to earlier tool output … re-run the tool" marker; added a no-bloat guard; fixed silent stats (JSON path now counts `jsonFieldsStripped`+`BytesSaved`; cache-hit `BytesSaved` = `len(raw)-len(marker)`).
  - **fix(memoryserver) `849cd693`**: moved the blocking `embedOne` HTTP call out of `ms.mu` in `Retain` (was freezing every concurrent op for up to the 120s client timeout); replaced hand-rolled `sqrt`/`sqrtFallback` (~30% error on large norms) with `math.Sqrt` in `denseCosine`.
  - **style `ca50e5c2`**: `gofmt -w` on 41 root-module files already gofmt-dirty on `HEAD` (the `ci.yml` gofmt gate was red on `main`); whitespace only.
 - **npm release**: `hermes-npm-v1.12.1` tagged + pushed → CI published all 7 packages; verified live (`npm view` → 1.12.1, `latest`). Patch bump (bug-fixes/CI/style/docs only). Verified 6-platform stage locally before tagging.
 - **Deferred (surfaced by a subsystem bug-sweep, not yet fixed)**: collab unbounded `h.sessions` + no WS read-limit/keepalive; scheduler drops-but-reports-success when a turn is in flight (`SendCtx` ignores ctx); SQLite backend rewrites the whole table per write; mcpbridge `orchestrate_task` unbounded goroutines; `learn.Load` ignores `maxObs`. **(All 5 fixed 2026-08-24 — see below.)**
 - **Total**: 5 commits (`46083879`, `d48e18c5`, `32e45de7`, `849cd693`, `ca50e5c2`), 1 npm release (7 packages, v1.12.1).
- **2026-08-24 session** (deferred subsystem sweep — all 5 items fixed, 6 commits, no release):
 - Re-verified all five deferred items against current code before touching anything; every fix ships a regression test that was confirmed to fail without it.
 - **fix(memoryserver) `05d381bd`**: the item was logged as write amplification, but under it was a data bug on the **default** backend — `sqliteStorage.Save` was upsert-only with no `DELETE`, so `Tidy()`'s purge never became durable and the expired row came back on the next `load()`. Delete the stored ids missing from the incoming set inside the same transaction (keeping per-row upserts, which exist to avoid a DELETE-all race). Also split the contract: `Save` = full replace (Tidy), new `SaveDelta` = changed entries only, so a `Retain` no longer rewrites every row. `fileStorage.SaveDelta` folds to a full rewrite — a JSON file can't be updated in place.
 - **fix(collab) `082acb6c`**: no read limit, no read deadline, no keepalive, while writes already had a 5s deadline. Cap frames at 1 MiB; read deadline refreshed by pongs; ping inside the window. Keepalive timings are per-`Hub` fields (default 60s/30s) so tests drive a real ping/pong cycle without a global. Ping writes take `peer.mu` since `Broadcast` writes the same conn.
 - **fix(collab) `62e9778a`**: `removePeer` left empty `peerSet`s in `h.sessions` — one permanent entry per session ID ever subscribed. Drop the session with its last peer.
 - **fix(mcpbridge) `690992dc`**: `orchestrate_task` spawned one goroutine per model-chosen step (the semaphore capped work, not goroutines), and `runReasonix` built its 5-min timeout from `context.Background()`, so the 15-min orchestration budget bounded only the decomposition call (worst case `ceil(N/3)×5` min). Cap at 10 steps, fixed 3-worker pool, per-step timeout derived from the orchestration ctx, stop feeding the queue when it's done.
 - **fix(learn) `29a8ce15`**: `Load` restored persisted slices wholesale, bypassing `maxObs` and the pattern ceiling. Trim on load (keep newest); `patternCap()` now shared with `detectPatterns`.
 - **fix(control) `40c4bfcc`** (fifth item, completed in a follow-up session after the local tooling failure cleared): `SendCtx` discarded its ctx and always returned nil; `Send` → `runGuarded` returns silently when `c.running`, so a cron task firing mid-turn vanished while `runTask` recorded `Success: true, Summary: "task dispatched"` and the 10-min deadline bound nothing. Split `runGuarded` into `startGuarded(parent, body)` — same lifecycle (autosave, panic recovery, closing `TurnDone`), but the turn ctx derives from `parent`, it returns `ErrTurnRunning` instead of returning silently, and it hands back a buffered error channel. `runGuarded` is now the fire-and-forget wrapper; `SendCtx` waits on the channel. Both `SendCtx` callers wanted the synchronous form (`cmd/learner-live-test` dropped its `Running()` poll, which raced the turn goroutine). Due scheduler tasks now run serially — the agent runs one turn at a time, so parallel dispatch would only make all but one fail.
 - **Total**: 6 commits (`05d381bd`, `082acb6c`, `62e9778a`, `690992dc`, `29a8ce15`, `40c4bfcc`), no release.

## roach-code Multi-Provider Research

**Repo**: `tmdgusya/roach-code` (⭐34, v1.3.5, 44 commits, 15 releases)

A multi-model rewrite of deepseek-reasonix that generalizes from DeepSeek-only to any provider. Same architecture (same internal/ layout, CLI, tools, MCP client) — a rebrand + multi-provider extension of upstream.

**Provider additions beyond upstream:**
| Provider | Kind | Notes |
|----------|------|-------|
| Codex/OpenAI | `codex` | Responses API + ChatGPT OAuth login (`roach-code codex login`) |
| MiniMax | `minimax` | Multimodal: text, image, video, speech, music, vision |
| GLM | `glm` | Z.ai API — Chinese LLM provider |

**Key patterns worth adopting:**
1. **`roach-code models` / `roach-code models refresh`** — CLI command to list/refresh configured models. Upstream Reasonix has model switching but no dedicated model list command.
2. **OAuth login flow** (`roach-code codex login`) — browser-based OAuth for ChatGPT subscribers. Pattern for adding auth-bound providers.
3. **Self-update** (`roach-code update`) — downloads latest release binary. Already in upstream goreleaser but not surfaced as a CLI command.
4. **Install scripts** (`install.sh`, `install.ps1`) — bash/PowerShell installers with SHA256 verification. Upstream relies on npm/brew/prebuilt archives.
5. **Config namespace** — uses `roach-code.toml` / `~/.config/roach-code/` / `.roach-code/` (not `reasonix.toml` / `.reasonix/`). Simplest way to avoid conflicts when both are installed.
6. **Short alias** (`roach`) — `make install` creates `roach` symlink. Upstream has `dsnix` alias built-in.

**Relevance to Hermes**: The `internal/provider/` registry already supports adding new providers via `init()` registration. Adding MiniMax/GLM would follow the same pattern as `provider/openai/` and `provider/anthropic/`. The key work is implementing each provider's wire format (OpenAI-compatible for MiniMax/GLM, proprietary for Codex).

