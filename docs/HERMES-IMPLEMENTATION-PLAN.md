# Hermes Fork — Implementation Plan (June 2026)

> **Note:** Paths updated June 12, 2026: `pkg/mcpbridge/` → `cmd/reasonix-mcpbridge/`,
> `pkg/memoryserver/` → `cmd/reasonix-memoryserver/`. Historical references in
> this document use the original paths.

> Generated from fresh ecosystem research on June 10, 2026. Prioritized by
> impact × feasibility × differentiation value. Each phase builds on the last.

---

## Phase 0: Foundation Sync (P0 — Week 1-2)

### 0.1 Sync with Upstream v1.5.0

**Status**: ✅ DONE. Fork synced to upstream v1.5.0 (commit e5e8f02, 2026-06-25). Clean merge, zero conflicts.

**What we get**:
- Bot gateway (Feishu/Weixin/QQ) — complements our Discord bot
- Goal mode (`/goal`) — autonomous loop with blocked-state audit
- `read_skill` tool — load inline skills in plan mode
- PDF attachment extraction
- Themeable workspace UI (4 variants)
- Tool-approval modes: ask/auto/yolo in desktop
- React 19 + TypeScript 6 frontend
- Subagent transcript continuation
- ACP (Agent Communication Protocol) sessions
- Ctrl+Home/Ctrl+End scroll, Ctrl+Z suspend, `!` shell prefix
- 100+ bug fixes and security patches (CodeQL, Myers diff overflow, SSE stream safety, process group reaping)

**Steps**:
```bash
git fetch upstream
git merge upstream/main-v2 --no-ff
# Resolve conflicts in:
#   - bot/main.go (our Discord bot vs upstream's Feishu/Weixin/QQ)
#   - pkg/mcpbridge/ (our custom package)
#   - pkg/memoryserver/ (our custom package)
#   - go.mod / go.sum
#   - REASONIX.md / AGENTS.md (project memory merge)
go build ./...
go vet ./...
go test ./...
```

**Conflict resolution strategy**: Upstream's `internal/bot/` is separate from our
`bot/` directory — they coexist. The upstream bot gateway is Feishu/Weixin/QQ
(`internal/bot/feishu/`, `internal/bot/weixin/`, `internal/bot/qq/`). Our Discord
bot lives in `bot/` (repo root). No conflict expected on bot code.

**Risks**:
- Desktop frontend dependencies (React 19, TypeScript 6, lucide-react 1.x) may need `npm install` re-run
- `go.mod` Go version bump may require toolchain update
- Custom package API surface may have shifted (agent, control, tool interfaces)

---

### 0.2 Pull Upstream Features Into Hermes Awareness

After sync, document which v1.5.0 features our custom packages should leverage:
- MCP bridge should expose `read_skill` as a tool
- Discord bot should support goal mode (`/goal` command)
- Memory server should integrate with upstream's `memory.Queue` interface

---

## Phase 1: Discord Bot — Real Agent Loop (P0 — Week 2-3) ✅ DONE

### 1.1 Replace Toy Implementation with Control.Controller ✅ DONE

**Completed**: Replaced `simulateReasonix()` toy with a proper `discord.Adapter` 
implementing the upstream `bot.Adapter` interface. The bot now plugs into the 
upstream `BotGateway` which provides session management, concurrency control, 
debounce, slash commands (`/approve`, `/deny`, `/answer`, `/stop`, `/new`, `/status`), 
and `renderSink` for event rendering.

New files:
- `internal/bot/discord/discord.go` — Discord Adapter (Platform, Name, Start, Stop, 
  Send, SendTyping, Messages, plus card/keyboard→Discord component conversion)
- `internal/bot/types.go` — added `PlatformDiscord` constant
- `internal/config/config.go` — added `DiscordBotConfig` struct, `DiscordUsers`/`DiscordGroups` to allowlist
- `internal/cli/bot.go` — wired Discord adapter into bot start command and doctor
- `bot/main.go` — rewritten as thin standalone entry point using `BotGateway`

Standalone binary usage:
```
DISCORD_BOT_TOKEN=... ./bin/reasonix-bot --server GUILD_ID
```
CLI usage:
```
reasonix bot start --channels discord --model deepseek-flash
```

**Target state**: Discord bot uses the same `control.Controller` as every other
frontend (TUI, desktop, HTTP/SSE). This is the architectural invariant the
upstream enforces.

**Key interfaces discovered post-sync**:
- `event.Sink` = `Emit(Event)` — single-method interface, trivial to implement
- `permission.Approver` = `Approve(ctx, toolName, subject, args) (allow, remember, err)`
- `permission.Gate` = `Policy` + optional `Approver`; nil Approver = yolo mode
- `control.Options` has `Sink`, `Policy`, `Hooks`, `OnRemember`, `Registry`, `PluginCtx`,
  `Jobs`, `BalanceURL/Key`, `AutoPlan`, `Classifier`, `Label`, `SystemPrompt`,
  `SessionDir/Path`, `Host`, `Commands`, `Skills/AllSkills/SkillStore/AllSkillStore`,
  `Memory`, `Cleanup`, `WorkspaceRoot`

**Design**:

```
Discord message → Controller.Send(text)
                    │
                    ▼
              control.Controller
                    │
         ┌──────────┼──────────┐
         ▼          ▼          ▼
     agent.Agent  Gate     event.Sink
     (executor)  (perms)  (→ Discord embed stream)
                    │
                    ▼
         Discord channel ← streaming embeds
```

**Implementation**:

1. **Create `bot/controller.go`**: Wraps `control.Controller` with Discord-aware event sink
   ```go
   type DiscordSink struct {
       session *discordgo.Session
       channelID string
   }
   func (s *DiscordSink) Emit(ev event.Event) {
       // Translate agent events → Discord embeds
       // - reasoning_content → spoiler embed
       // - text deltas → streaming message edits
       // - tool dispatch/results → tool-card embeds
       // - ask questions → Discord button components
       // - approval prompts → Discord button components
   }
   ```

2. **Create `bot/session.go`**: Per-channel session management
   ```go
   type BotSession struct {
       Controller *control.Controller
       ChannelID  string
       GuildID    string
       CreatedAt  time.Time
       LastActive time.Time
   }
   ```

3. **Wire configuration**: Read `reasonix.toml` for the bot
   ```go
   cfg := config.Load(/* flag > project > user > defaults */)
   ctrl := control.New(cfg, control.Options{
       Sink:  &DiscordSink{...},
       Label: "discord-bot",
   })
   ```

4. **Permission model**: Discord roles map to Reasonix permission postures
   - Server admin → `yolo` posture
   - Trusted role → `auto` posture
   - Default → `ask` posture (approval via Discord buttons)

5. **Slash commands**:
   - `/reasonix chat <message>` — send a turn
   - `/reasonix model <name>` — switch model
   - `/reasonix plan <task>` — enter plan mode for one task
   - `/reasonix goal <objective>` — start autonomous goal
   - `/reasonix stop` — cancel running turn
   - `/reasonix status` — show session stats (tokens, cache hit rate, cost)

**Dependencies**: `github.com/bwmarrin/discordgo` (already in go.mod)

---

### 1.2 Discord-Aware Permission Approver

Extend `control.Controller`'s `Approver` interface for Discord:

```go
type DiscordApprover struct {
    session   *discordgo.Session
    channelID string
}

func (a *DiscordApprover) Approve(ctx context.Context, req permission.Request) (permission.Decision, error) {
    // Send Discord message with Approve/Deny/Always buttons
    // Wait for button interaction or timeout
    msg, _ := a.session.ChannelMessageSendComplex(a.channelID, &discordgo.MessageSend{
        Embed:   approvalEmbed(req),
        Components: []discordgo.MessageComponent{
            discordgo.ActionsRow{Components: []discordgo.MessageComponent{
                discordgo.Button{Label: "Approve Once", CustomID: "approve_once"},
                discordgo.Button{Label: "Always Allow", CustomID: "approve_always"},
                discordgo.Button{Label: "Deny", CustomID: "deny"},
            }},
        },
    })
    // Wait for interaction...
}
```

---

## Phase 2: Tests & Polish (P0 — Week 3-4) ✅ DONE

### 2.1 Tests for `pkg/mcpbridge` ✅ DONE

**Completed**: 48 tests in `pkg/mcpbridge/main_test.go`, 82% coverage.

Test areas:
- `doctorCheck` — basic report, API key set/unset, reasonix binary detection
- `listSkills` — config dir, project-local fallback, missing dir, empty dir
- `planTask` — no API key, with test server (fake DeepSeek API), API error, empty objective
- `orchestrateTask` — no reasonix binary, with test server, empty task
- `callDeepSeek` — no API key, custom model, empty response, connection error
- `runTask` / `executeTool("reasonix_run")` — missing/empty task, binary not found, workdir, model flag
- `executeTool` dispatch — unknown tool, doctor via dispatch
- JSON-RPC `handleMessage` — initialize, tools/list, invalid JSON, method not found, notifications, tools/call
- HTTP handler — health endpoint, `/mcp` endpoint with JSON-RPC, bad body
- Auth middleware integration — no key, with key, `/health` always public
- `parseSteps` — numbered, single, paren format, step prefix, multi-line
- `isStepHeader` — 7 cases (numbered, step prefix, negative cases)
- `stripStepPrefix` — 3 formats
- `NewBridgeServer` — registers all 5 tools, default/custom API base

### 2.2 Tests for `pkg/memoryserver` ✅ DONE

**Completed**: 52 tests in `pkg/memoryserver/main_test.go`, 89% coverage.

Test areas:
- `retainMemory` — store fact, unique IDs, no-tags case, empty content, very long content, special chars (unicode, emoji, HTML), save error (read-only dir)
- `recallMemory` — keyword search, tag filtering, session filtering, limit, no-match, case-insensitive, empty query returns all
- `reflectOnMemories` — session reflection, empty session, long content truncated, no memories
- `NewMCPServer` / `NewMemoryStore` — creation, directory creation, reload, persistence across instances, mkdir error
- JSON-RPC `handleMessage` — initialize, tools/list, tools/call (retain/recall/reflect), notifications, unknown method/tool, invalid JSON
- `truncateStr` — short, exact, truncated, unicode
- `ServeHTTP` — health endpoint, `/mcp` endpoint, auth enabled (401/403/200)
- `ServeStdio` — EOF, single message
- Auth middleware — no key, with key, health always public

### 2.3 Discord Adapter Tests ✅ DONE

**Completed**: 57 tests in `internal/bot/discord/discord_test.go`, 91% coverage.

Test areas:
- `New()` — config propagation, nil logger fallback, channel buffer
- `Platform()` / `Name()` — returns "discord"
- `Messages()` — non-nil channel
- `Start()` — missing token error (env + DISCORD_BOT_TOKEN fallback)
- `Stop()` — nil session, unconnected session
- `Send()` — nil session, long content truncation, card/embed path, keyboard path, reply-to, plain message
- `SendTyping()` — nil session, with session
- `onReady()` — constructed Ready event
- `onMessageCreate()` — own-message filter, channel filter, empty content, DM filter, mention stripping, field population, channel overflow
- `resolveChatType()` — cache hit, state lookup for all channel types (GuildText, DM, PublicThread, PrivateThread, GroupDM, unknown), not-in-state fallback
- `stripMention()` — 6 cases
- `cardToMarkdown()` — empty, single markdown, multi-element, skip non-markdown
- `keyboardToComponents()` — nil, empty, single button, multi-row, style mapping

### 2.4 HTTP Auth Middleware ✅ DONE

**Completed**: 8 tests in `pkg/httputil/auth_test.go` covering:
- Auth disabled (no key)
- Health endpoint always public
- No auth header → 401
- Invalid format → 401
- Wrong key → 403
- Correct key → 200
- Status reporting

**Consolidation**: Both `pkg/mcpbridge/` and `pkg/memoryserver/` now import
`reasonix/pkg/httputil` instead of duplicating `requireBearer`. Duplicated
code (40+ lines each) deleted.

### 2.5 MCP Bridge: planTask + orchestrateTask ✅ DONE

**Completed**: Both formerly stub functions now fully implemented.

- `planTask(objective)` — calls DeepSeek API directly (reads `DEEPSEEK_API_KEY`,
  `DEEPSEEK_BASE_URL`, `DEEPSEEK_MODEL` from env). Returns `# Execution Plan`
  with numbered steps.
- `orchestrateTask(task)` — decomposes task via DeepSeek API, parses numbered
  steps, runs each via `reasonix run` in parallel goroutines. Returns
  `# Orchestration Results` with decomposition + per-step execution output.
- Helper: `callDeepSeek(system, user)` — shared HTTP client for DeepSeek chat
  completions. Used by both planTask and orchestrateTask.
- Helper: `parseSteps(text)` / `isStepHeader(line)` / `stripStepPrefix(line)` —
  robust step extraction supporting "1." "1)" "Step 1:" formats.

### 2.6 Hook Scripts Hardening ✅ DONE

**Completed**: Both `scripts/retain-hook.sh` and `scripts/reflect-hook.sh` hardened.

Changes:
- Added `command -v curl` and `command -v python3` checks with stderr warnings
- Added `--max-time $HINDSIGHT_TIMEOUT` (default 5s) to curl calls
- Python3 blocks now `sys.exit(1)` on exception (caught by `|| { warning; exit 0 }`)
- Curl failures print diagnostic to stderr
- New env: `HINDSIGHT_TIMEOUT` (seconds, default 5)

Integration test: `scripts/test-hooks-integration.sh` — 12/12 pass.
Tests: skip noise tools, empty tool name, meaningful tool sends retain,
server unreachable, auth header, malformed JSON, reflect with session,
empty session defaults to latest.

### 2.7 CI Pipeline

Add `.github/workflows/ci-hermes.yml` — **already exists** (supplementary CI for desktop frontend). Extend it to also run `go test ./pkg/... ./bot/...`:

```yaml
name: Hermes CI
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.25' }
      - run: go build ./...
      - run: go vet ./...
      - run: go test ./pkg/...
      - run: go test ./bot/...
```

---

## Phase 3: Skills Hub Integration (P1 — Week 4-5) ✅ DONE

### 3.1 Auto-Load Skills Into Reasonix Native Registry ✅ DONE

**Completed**: Created `scripts/install-skills.sh` — a portable installer that:
- Resolves target directory from `$REASONIX_SKILLS_DIR`, `XDG_CONFIG_HOME`, project-local `.reasonix/skills/hermes`, or `~/.config/reasonix/skills/hermes`
- Copies all 16 skill `.md` files + `registry.json`
- Supports `--dry-run` for preview
- Prints `reasonix.toml` config snippet after install

**✅ install_source integrated** (2026-06-11): Created `reasonix-hermes.json` manifest at repo root. Upstream `install_source` auto-discovers skills from GitHub repos:
```bash
reasonix install-source install --source https://github.com/aliatx2017/reasonix-hermes/tree/main/skills-hub/skills
```
This scans the `skills-hub/skills/` directory via GitHub API, validates frontmatter, and copies all 17 skills into `~/.config/reasonix/skills/<name>/SKILL.md` (canonical layout). The `install-skills.sh` script remains as a fallback for non-Reasonix environments.

### 3.2 Add Missing Skills from Community

Review the top community skill packs and curate additions:
- From **reasonix-skill-powers** (⭐45): Adopt the 7-step workflow as a meta-skill
- From **superpowers-reasonix** (⭐10): `tdd` workflow, `verification` pipeline
- From **Deepseek-Reasonix-Autopilot** (⭐2): Study the 113-skill architecture for patterns
- **Adversarial review** skill: Based on kquuen's `BLOCK:`/`ALLOW:` contract

---

### 3.3 Skills Hub Website

Create a simple GitHub Pages site at `aliatx2017.github.io/reasonix-hermes/`:
- Browseable skill catalog rendered from `registry.json`
- One-click install instructions (`curl ... | reasonix install-source -`)
- Rating/feedback via GitHub Issues
- Categorized by: Development, Research, Operations, Quality, Meta

---

## Phase 4: Hermes Memory Server — Productionize (P1 — Week 5-6)

### 4.1 Hook-Based Integration ✅ DONE

**Completed**: Hook scripts and settings template for automatic session memory:

- `scripts/retain-hook.sh` — `PreToolUse` hook that sends tool context to 
  Hindsight memory server (filters out noise tools like `read_file`, `write_file`)
- `scripts/reflect-hook.sh` — `Stop` hook that triggers session reflection
- `scripts/hooks-settings-template.json` — drop-in `.reasonix/settings.json` 
  with `PreToolUse`, `PostToolUse`, and `Stop` hooks wired

Both scripts support `HINDSIGHT_URL` and `HINDSIGHT_KEY` env vars for the 
memory server URL and Bearer auth token.

**HTTP Auth for memory server**: Added `requireBearer` middleware (reads 
`MEMORY_API_KEY` env var). `/health` endpoint is always unauthenticated. 
When `MEMORY_API_KEY` is empty, auth is disabled (backward compatible).

**Hook mode design**:
```toml
[hooks]
pre_turn = "reasonix-memory recall --session $REASONIX_SESSION_ID"
post_turn = "reasonix-memory retain --session $REASONIX_SESSION_ID"
stop = "reasonix-memory reflect --session $REASONIX_SESSION_ID"
```

This mirrors the Hindsight-Reasonix pattern from `houycth/Hindsight-Reasonix`,
but without the upstream patch requirement (if Reasonix hooks already pass
session_id — verify after Phase 0 sync).

### 4.2 Storage Backend Options ✅ DONE (P3)

✅ Completed (Phase 3): All three backends implemented.
- **File**: Simple JSON files in `~/.reasonix/hindsight-memory/`
- **SQLite**: `modernc.org/sqlite` (pure Go, no CGO), WAL journal mode, 3 indexes (session_id, expires_at, importance). `--backend sqlite` flag.
- **Vector**: Sparse TF-IDF cosine similarity search. `semantic=true` flag on `hindsight_recall`. Vectors auto-computed on retain, persisted in both JSON and SQLite backends.

### 4.3 Memory Retention Policies ✅ DONE (P3)

✅ Completed (Phase 3): All retention policies implemented.
- **TTL-based expiry**: Per-fact TTL, defaults to 90 days
- **Importance scoring**: `Importance` field (0.5 → +0.05 per recall, 1%/day decay). `ExpiresAt` auto-computed on retain.
- **Project-scoped isolation**: Facts tagged by session ID (`session_id` filter on recall/reflect)
- **Cross-project recall**: No session_id filter returns all facts across projects
- **Tidy()**: Purges expired entries whose importance has dropped to `minImportanceToKeep`

---

## Phase 5: Multi-Agent & Ecosystem Integration (P2 — Week 6-8)

### 5.1 collab-cli Integration ✅ DONE

Added as pre-configured MCP plugin in `reasonix.example.toml` (2026-06-11).
Entry includes install instructions (`go install github.com/cejkato/collab-cli@latest`).

```toml
[[plugins]]
name    = "collab"
command = "collab"
args    = ["mcp"]
```

### 5.2 VS Code Extension Packaging ✅ DECIDED (2026-06-11)

**Decision**: Fork `whishi47/deepseekcode-reasonix-vscode` (⭐1, MIT) as a
**separate repository** with Hermes branding — Discord bot status indicator,
memory server health, Activity Bar integration. Implementation lives outside
this codebase.

**Reference**: `whishi47/deepseekcode-reasonix-vscode` — 3 keyboard shortcuts,
2.5s readiness delay, compatible with Windsurf/Trae.

### 5.3 Adversarial Review Skill ✅ DONE (2026-06-11)

Created `skills-hub/skills/adversarial-review.md` — ported kquuen `BLOCK:`/`ALLOW:`
review contract. Registered as 17th skill in `registry.json`.

Key design:
- 5 attack surfaces: security, correctness, performance, maintainability, coverage
- Structured `BLOCK:` / `ALLOW:` output format
- Severity levels: blocker, high, medium, low, info
- Confidence levels: high, medium, low
- `runAs: subagent` on `deepseek-pro` model

---

## Phase 6: Advanced Features (P3 — Ongoing)

### 6.1 PortaKit-Style Portability ✅ DONE (2026-06-11)

Implemented via `REASONIX_PORTABLE=1` environment variable:
- `internal/config/config.go`: added `IsPortable()` + `reasonixDir()` helper
- All reasonix data paths (config, sessions, cache, memory, skills, commands)
  redirect to `<binary_dir>/.reasonix/` when portable mode is active
- `pkg/mcpbridge`: `skillDirs()` respects portable mode
- `pkg/memoryserver`: store directory respects portable mode
- USB/sync-drive friendly — all data lives next to the binary

### 6.2 Vector Memory Backend

For `pkg/memoryserver`: add optional embedding-based semantic search using
DeepSeek's embedding API (or local onnx runtime). Enables "find conversations
about authentication" style queries.

### 6.3 Multi-Model Discord Bot (Future)

Let Discord users choose their model per-channel or per-request:
- `/reasonix model flash` — DeepSeek V4 Flash (cheap, fast)
- `/reasonix model pro` — DeepSeek V4 Pro (thorough)
- `/reasonix model mimo` — MiMo v2.5 Pro (creative/文案)

### 6.4 roach-code Pattern: Add More Providers (Future)

Study `tmdgusya/roach-code` (⭐34) for its multi-provider architecture. Consider
adding:
- MiniMax provider (multimodal: text/image/video/speech)
- GLM provider (Z.ai)
- Direct Anthropic (already in upstream)

---

## Summary — Priority Matrix

| Phase | Item | Impact | Effort | Status |
|-------|------|--------|--------|--------|
| **P0** | Sync upstream v1.5.0 | CRITICAL | 2-3 days | ✅ DONE (e5e8f02) |
| **P0** | Wire Discord bot → Controller | HIGH | 3-5 days | ✅ DONE |
| **P0** | Tests for pkg/mcpbridge | MEDIUM | 1-2 days | ✅ DONE (48 tests, 82%) |
| **P0** | Tests for pkg/memoryserver | MEDIUM | 1-2 days | ✅ DONE (52 tests, 89%) |
| **P0** | Tests for internal/bot/discord | MEDIUM | 1-2 days | ✅ DONE (57 tests, 91%) |
| **P0** | Auth middleware consolidation | LOW | 0.5 day | ✅ DONE (httputil shared) |
| **P0** | planTask + orchestrateTask | HIGH | 1-2 days | ✅ DONE (DeepSeek API) |
| **P0** | Hook scripts hardening | MEDIUM | 0.5 day | ✅ DONE (timeout, checks, tests) |
| **P0** | CI pipeline | MEDIUM | 1 day | ✅ DONE (ci-hermes.yml builds+tests all Hermes packages; e2e-discord.yml smoke test) |
| **P1** | Skills hub auto-loading | MEDIUM | 2-3 days | ✅ DONE (install-skills.sh) |
| **P1** | Memory server hook mode | HIGH | 3-4 days | ✅ DONE (hook scripts + auth) |
| **P1** | Skills hub website | LOW | 2-3 days | ✅ DONE (skills-hub/site/index.html) |
| **P2** | collab-cli integration | MEDIUM | 1-2 days | ✅ DONE (reasonix.example.toml) |
| **P2** | VS Code extension | MEDIUM | 3-5 days | ✅ DECIDED (fork separate repo) |
| **P2** | Adversarial review skill | LOW | 1 day | ✅ DONE (skills-hub 17th skill) |
| **P3** | Hook scripts → native Go hooks | MEDIUM | 2-3 days | ✅ DONE (cmd/reasonix-hooks) |
| **P3** | Memory backend: SQLite | MEDIUM | 3-5 days | ✅ DONE (modernc.org/sqlite, WAL) |
| **P3** | Memory TTL + importance | LOW | 2-3 days | ✅ DONE (90d TTL, boost/decay) |
| **P3** | read_skill MCP tool | LOW | 1 day | ✅ DONE (get_skill, 6th tool) |
| **P3** | Discord /goal command | LOW | 1-2 days | ✅ DONE (goal loop + audit) |
| **P3** | PortaKit portability | LOW | 3-5 days | ✅ DONE (REASONIX_PORTABLE) |
| **P3** | Vector memory backend | LOW | 5-7 days | ✅ DONE (TF-IDF cosine sim, semantic=true) |
| **P3** | Multi-model Discord bot | LOW | 2-3 days | ✅ DONE (/model flash|pro|mimo) |

---

## Success Metrics

| Metric | Current | Target |
|--------|---------|--------|
| Upstream sync | v1.5.0 (e5e8f02) ✅ | Mergeable in <1 hour |
| Discord bot | ✅ Full agent loop (discord.Adapter → BotGateway) | Slash commands, approval, sessions |
| Test coverage (pkg/) | 228 tests (hooks 12, mcpbridge 49, memory 63, discord 57, bot 29+) | >80% line coverage ✅ |
| MCP bridge tools | ✅ 6 tools (run, doctor, plan, orchestrate, get_skill, get_skills) | External agent orchestration |
| Hook scripts | ✅ Native Go binary (cmd/reasonix-hooks) + hardened shell fallback | Zero-dependency binary |
| Skills discoverable | ✅ 17 skills + install_source integration (reasonix-hermes.json) + install-skills.sh fallback | GitHub Pages deployable ✅ |
| Memory persistence | ✅ SQLite (WAL) + TTL/importance + vector TF-IDF | ✅ Complete |
| Discord bot features | ✅ /goal + /model (autonomous loop + multi-model) | ✅ Complete |
| Portability | ✅ REASONIX_PORTABLE=1 (portable data dir) | USB/sync-drive friendly |
| CI | ✅ ci-hermes.yml (builds+tests all Hermes packages) + e2e-discord.yml (manual smoke test) | ✅ Complete |
| Community presence | Fork repo (VS Code ext separate) | VS Code Marketplace listing |
