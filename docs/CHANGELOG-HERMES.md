# Reasonix-Hermes Changelog

Key milestones in the Hermes fork since June 2026.

## v1.7.0+ (July 2026)

### Session 2026-07-15 (h13) — Golang audit, dead code, t.Parallel, council judge, docs
- Golang patterns audit: `go vet` + `staticcheck` → 5 dead code items removed
- `t.Parallel()`: 96 test functions across 10 custom packages
- Council judge: Fusion Router-inspired `Council.Judge()` with structured JSON (Consensus, Contradictions, CoverageGaps, UniqueInsights, BlindSpots), 6 tests
- Docs: dead link fix, logo concepts removed, 6 stale docs → `CHANGELOG-HERMES.md`
- Vision aux model: `ollamacloud-vision/gemini-3-flash-preview` configured

### Session 2026-07-15 (h12) — Code audit fixes + docs cleanup
- Dockerfile Go 1.24 → 1.25 (matches go.mod)
- Merged duplicate `[desktop]` config sections
- Removed dead `rememberRule` helper, consolidated grant logic
- Memory server migrated from `log.Printf` to structured `slog`
- Helm chart image tag pinned from `latest` to `v1.7.0`
- `docs/PROJECT.md` created — human-oriented project overview
- Deleted 1,997 lines of stale assessment docs → consolidated here

### Session 2026-07-15 (h11) — Completeness sweep + eval GUI + analytics + orchestrate
- Orphan slash completers restored (`/stats`, `/cost`, `/council`, `/learn`, `/publish`, `/todo`)
- Config render.go data-loss bug fixed (10 missing sections)
- Desktop: hotbar "unbound" display fix, eval panel, analytics panel, orchestrate panel, learned patterns panel
- `internal/orchestrate/` — Chain, Pair, CIFix multi-agent workflows (6 tests)
- `CONTRIBUTING.md` rewritten; `docs/index.html` rebranded

### Session 2026-07-15 (h9) — Code health + docs audit + session comparison
- Bridge.ts drift fix (37 method declarations, 5 type mismatches)
- `internal/eval/` — session comparison tool (Jaccard similarity, structural diff, 6 tests)
- SPEC.md §2 overhauled — all 57 internal packages + cmd/ tree documented

### Session 2026-07-15 (h8) — Controller decomposition + bug fixes
- Controller reduced from 3,744 to 2,670 lines (29%); extracted 4 sub-files
- 5 bug fixes: hotbar defaults, desktop layout style missing from settings, render drops profiles/hotbar, netclient mock incompatibility, Mimo backfill gaps
- 14 skills adopted from ~/.hermes/skills
- Slack adapter tests: 23 tests covering all `bot.Adapter` methods

### Session 2026-07-15 (h7) — Audit + 4-phase expansion
- BotGateway session memory leak fixed (eviction loop, 30-min idle timeout)
- `install-source` CLI command
- Dense memory embeddings (`[embedding]` config, OpenAI-compatible `/v1/embeddings`, cosine similarity)
- Autonomous learning loop (`internal/learn/` — pattern detection, skill suggestion, `/learn` command)
- Slack adapter (`internal/bot/slack/` — Socket Mode, DMs + @mentions)

### Session 2026-07-14 (h6) — Session stats persistence
- Agent aggregate counters (tokens in/out, turns)
- Sidecar `.sessionstats` persistence
- Desktop: session stats widget, Wails bindings

### Session 2026-07-14 (h5) — Logo + branding + npm
- Animated Diamond Wing logo (SVG)
- README overhaul (both en + zh-CN)
- `npm i -g reasonix-hermes` — one-line install pipeline

### Session 2026-07-14 (h4) — Desktop widgets + GitHub Action + collab
- Desktop: schedule, cost, publish widgets
- `cmd/reasonix-pr-review/` — GitHub Action for PR review
- `internal/collab/` — WebSocket Hub for live collaboration (8 tests)
- Helm chart + docker-compose for one-click deploy

### Session 2026-07-14 (h3) — Telegram + learn + mesh
- Telegram bot adapter (long-polling, 16 tests)
- `internal/learn/` — self-improving skill loops (16 tests)
- `internal/mesh/` — agent-to-agent MCP mesh (delegate, broadcast, council, 13 tests)

### Session 2026-07-14 — Discord e2e + competitive analysis
- Discord bot: gateway connection, @mentions, DMs, `/model` command, approval flow
- 8 bug fixes (hardcoded Chinese strings, approval parsing, dispatch blocking)
- Competitive landscape analysis: 15+ competitors documented

### Session 2026-07-13 — Discord + Write Mode + D3 + desktop Hermes
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
