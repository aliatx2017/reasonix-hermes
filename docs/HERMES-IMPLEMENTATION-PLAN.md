# Hermes Fork — Implementation Plan (June 2026)

> Generated from fresh ecosystem research on June 10, 2026. Prioritized by
> impact × feasibility × differentiation value. Each phase builds on the last.

---

## Phase 0: Foundation Sync (P0 — Week 1-2)

### 0.1 Sync with Upstream v1.5.0

**Status**: Our fork is on an unknown earlier `main-v2` snapshot. Upstream has
released **v1.2.0 through v1.5.0** with major features.

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

## Phase 1: Discord Bot — Real Agent Loop (P0 — Week 2-3)

### 1.1 Replace Toy Implementation with Control.Controller

**Current state**: `bot/main.go` has inline chat history (`[]Message`), no agent
loop, no tool execution, no permission gating. It's a chat mirror, not an agent.

**Target state**: Discord bot uses the same `control.Controller` as every other
frontend (TUI, desktop, HTTP/SSE). This is the architectural invariant the
upstream enforces.

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

## Phase 2: Tests & Polish (P0 — Week 3-4)

### 2.1 Tests for `pkg/mcpbridge`

**Current state**: Zero tests. 5 tools (run, doctor, plan, orchestrate, skills).
Single `main.go` (~13KB).

**Test plan**:
- `mcpbridge_test.go`: Integration test with a real Reasonix binary
  - `TestRunTask`: Send a task, verify non-empty response, verify exit code 0
  - `TestDoctorTask`: Send a diagnostic task, verify structured output
  - `TestPlanTask`: Send a plan request, verify plan structure
  - `TestOrchestrateTask`: Multi-step orchestration
  - `TestSkillsList`: List available skills
  - `TestTimeout`: Verify timeout handling
  - `TestInvalidInput`: Verify error handling for malformed requests
- `mcpbridge_mock_test.go`: Unit tests with a mock Reasonix process

### 2.2 Tests for `pkg/memoryserver`

**Current state**: Zero tests. 3 tools (retain, recall, reflect). Single
`main.go` (~11KB).

**Test plan**:
- `memoryserver_test.go`:
  - `TestRetainAndRecall`: Store a fact, retrieve it
  - `TestRecallEmpty`: Query empty store → empty result
  - `TestReflect`: Semantic search across stored facts
  - `TestCrossSession`: Write facts, restart server, read them back
  - `TestLargePayload`: Handle large memory entries
  - `TestConcurrentAccess`: Multiple simultaneous retain/recall calls

### 2.3 CI Pipeline

Add `.github/workflows/hermes-ci.yml`:
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

## Phase 3: Skills Hub Integration (P1 — Week 4-5)

### 3.1 Auto-Load Skills Into Reasonix Native Registry

**Current state**: 16 skills in `skills-hub/skills/*.md` + `registry.json`. They
are static files — not discoverable by Reasonix's skill system without manual
copying to `.reasonix/skills/`.

**Target state**: `reasonix setup --with-hermes-skills` or automatic detection
that installs our curated skill pack.

**Implementation**:

1. **Create an install script** (`scripts/install-skills.sh` / `.ps1`):
   ```bash
   #!/bin/bash
   SKILLS_DIR="$(dirname "$0")/../skills-hub/skills"
   TARGET="$HOME/.config/reasonix/skills/hermes/"
   mkdir -p "$TARGET"
   cp "$SKILLS_DIR"/*.md "$TARGET/"
   echo "Installed 16 Hermes skills to $TARGET"
   echo "Add to reasonix.toml: [skills] paths = [\"~/.config/reasonix/skills/hermes\"]"
   ```

2. **Register as a skill root** in Reasonix config:
   ```toml
   [skills]
   paths = ["~/.config/reasonix/skills/hermes"]
   ```

3. **Add `install-source` support**: Make `skills-hub/` installable via
   `reasonix install-source` or our MCP bridge's `install_capability` tool.

4. **Sync with awesome-reasonix**: Submit our skills to the community hub
   (`hikari-2424/awesome-reasonix`) for broader distribution.

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

### 4.1 Hook-Based Integration (Like Hindsight-Reasonix)

**Current state**: Standalone MCP server with 3 tools. Works but no integration
with Reasonix's hook system.

**Target state**: Memory server can work in two modes:
1. **MCP mode** (current): External tools for any MCP client
2. **Hook mode** (new): Reasonix `PreToolUse`/`PostToolUse`/`Stop` hooks for
   automatic session memory

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

### 4.2 Storage Backend Options

Add pluggable backends:
- **File** (current): Simple JSON files in `~/.reasonix/memory/`
- **SQLite**: For larger memory stores with indexed search
- **Vector**: Optional embedding-based semantic search (deferred to Phase 6)

### 4.3 Memory Retention Policies

- **TTL-based expiry**: Per-fact TTL, defaults to 90 days
- **Importance scoring**: Frequently recalled facts get longer TTL
- **Project-scoped isolation**: Facts tagged by project hash
- **Cross-project recall**: Global facts available across projects

---

## Phase 5: Multi-Agent & Ecosystem Integration (P2 — Week 6-8)

### 5.1 collab-cli Integration

Add collab-cli as a pre-configured MCP plugin:

```toml
[[plugins]]
name = "collab"
type = "stdio"
command = "collab"
args = ["mcp"]
```

**Value**: Hermes gets 17 MCP tools for free — agent handshake, task management,
shared memory (SHARD.md), agent-to-agent commands, self-review pipeline, LAN
node sync, web dashboard.

**Steps**:
1. Add `collab-cli` to our documentation as recommended integration
2. Pre-configure in `reasonix.example.toml`
3. Test the multi-agent workflow: Hermes Discord bot + collab-cli + Reasonix CLI

### 5.2 VS Code Extension Packaging

Based on `whishi47/deepseekcode-reasonix-vscode` (⭐1, MIT):

**Create `vscode-extension/` in our repo**:
- Fork/adapt the existing extension
- Add Hermes-specific features: Discord bot status indicator, memory server health
- Publish to VS Code Marketplace as "Reasonix Hermes"
- Add to Open VSX Registry for Windsurf/Trae users

**Minimal viable extension**:
- Activity Bar whale icon → launch terminal with `reasonix chat`
- Auto `@file#L10-L20` context injection
- Three keyboard shortcuts (Ctrl+Esc, Ctrl+Shift+Esc, Ctrl+Alt+K)
- Hermes branding

### 5.3 Adversarial Review Skill

Port the kquuen `BLOCK:`/`ALLOW:` review contract as a Hermes skill:

```markdown
---
name: adversarial-review
description: Adversarial code review with structured BLOCK/ALLOW output and 5 attack surfaces
runAs: subagent
model: deepseek-pro
---

You are an adversarial code reviewer. Your default stance is skepticism.
Review the provided code/diff against 5 attack surfaces:

1. **Security**: Injection vectors, missing authz, exposed secrets, path traversal
2. **Correctness**: Edge cases, race conditions, null/nil handling, type safety
3. **Performance**: N+1 queries, unbounded allocations, blocking I/O, memory leaks
4. **Maintainability**: Coupling, naming clarity, missing comments, testability
5. **Coverage**: Untested paths, missing edge case tests, integration test gaps

Your response MUST start with exactly:
BLOCK: <one-line reason>    — if any issue found that should prevent merge
ALLOW: <one-line reason>    — if safe to proceed

Then list findings with file:line, severity (blocker/high/medium/low/info),
confidence (high/medium/low), and concrete fix recommendation.
```

---

## Phase 6: Advanced Features (P3 — Ongoing)

### 6.1 PortaKit-Style Portability

Add `--portable` flag to Hermes builds:
- Auto-detect data directory relative to binary (not `~/.config/reasonix/`)
- Fix path hashes on workspace change (PortaKit's core innovation)
- USB/sync-drive friendly

### 6.2 Vector Memory Backend

For `pkg/memoryserver`: add optional embedding-based semantic search using
DeepSeek's embedding API (or local onnx runtime). Enables "find conversations
about authentication" style queries.

### 6.3 Multi-Model Discord Bot

Let Discord users choose their model per-channel or per-request:
- `/reasonix model flash` — DeepSeek V4 Flash (cheap, fast)
- `/reasonix model pro` — DeepSeek V4 Pro (thorough)
- `/reasonix model mimo` — MiMo v2.5 Pro (creative/文案)

### 6.4 roach-code Pattern: Add More Providers

Study `tmdgusya/roach-code` (⭐34) for its multi-provider architecture. Consider
adding:
- MiniMax provider (multimodal: text/image/video/speech)
- GLM provider (Z.ai)
- Direct Anthropic (already in upstream)

---

## Summary — Priority Matrix

| Phase | Item | Impact | Effort | Dependencies |
|-------|------|--------|--------|--------------|
| **P0** | Sync upstream v1.5.0 | CRITICAL | 2-3 days | None |
| **P0** | Wire Discord bot → Controller | HIGH | 3-5 days | P0 sync |
| **P0** | Tests for pkg/mcpbridge | MEDIUM | 1-2 days | None |
| **P0** | Tests for pkg/memoryserver | MEDIUM | 1-2 days | None |
| **P0** | CI pipeline | MEDIUM | 1 day | None |
| **P1** | Skills hub auto-loading | MEDIUM | 2-3 days | P0 sync |
| **P1** | Memory server hook mode | HIGH | 3-4 days | P0 sync |
| **P1** | Skills hub website | LOW | 2-3 days | None |
| **P2** | collab-cli integration | MEDIUM | 1-2 days | P0 sync |
| **P2** | VS Code extension | MEDIUM | 3-5 days | None |
| **P2** | Adversarial review skill | LOW | 1 day | None |
| **P3** | PortaKit portability | LOW | 3-5 days | P0 sync |
| **P3** | Vector memory backend | LOW | 5-7 days | P1 memory |
| **P3** | Multi-model Discord bot | LOW | 2-3 days | P0 bot |

---

## Success Metrics

| Metric | Current | Target |
|--------|---------|--------|
| Upstream sync | Unknown version | v1.5.0, mergeable in <1 hour |
| Discord bot | Toy chat history | Full agent loop with tool execution |
| Test coverage (pkg/) | 0% | >80% |
| Skills discoverable | Manual file copy | `install-source` or `--with-hermes-skills` |
| Memory persistence | MCP-only | MCP + hooks auto-mode |
| CI | None | Build + test + vet on push/PR |
| Community presence | None | VS Code Marketplace + awesome-reasonix listing |
