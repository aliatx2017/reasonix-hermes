# GitHub Repo Evaluations — 2026-06-20

Evaluated 17 repositories using the `github-repo-eval` skill framework (surface scan → structure assessment → deep dive → community health).

---

## Adopt ⭐

### headroom (chopratejas/headroom) — 41.7k ★
**What**: Token-saving proxy for LLM applications. 60-95% fewer tokens, same answers.
**Why Adopt**: Exceptionally engineered. Python SDK + Rust core via PyO3, 308 test files, 6+ algorithms (AST-aware CodeCompressor, ML SmartCrusher, CacheAligner). Real benchmarks on GSM8K / TruthfulQA / SQuAD. Proxy mode requires zero code changes. **Installed**: v0.26.0 via pip.
**Caveats**: Pre-1.0 (v0.26.0), rapid release cadence (156 releases). Single-maintainer risk. ONNX models downloaded at runtime.
**Relevance to Reasonix**: Token compression — could integrate via MCP server (`headroom mcp`) or proxy mode.

### markitdown (microsoft/markitdown) — 156k ★
**What**: Lightweight Python utility converting files (PDF, Word, Excel, PowerPoint, HTML, audio, images, ZIP, JSON, EPUB, Outlook MSG, RSS, YouTube) to Markdown.
**Why Adopt**: Market leader. MIT license, Microsoft-backed (AutoGen team). 19 releases in 18 months. Plugin system + MCP server. 15+ format converters. Test vectors with `must_include` assertions. **Installed**: v0.1.5 + markitdown-mcp v0.0.1a4.
**Caveats**: 418 open issues/438 open PRs (backlog noise). Beta status. Some tests skipped in CI.
**Relevance to Reasonix**: File-to-markdown conversion — MCP server available for agent use.

### Agent-Reach (Panniantong/Agent-Reach) — 35.8k ★
**What**: Zero-API-fee CLI giving AI agents web access to 13 platforms (Twitter/X, YouTube, Reddit, GitHub, Bilibili, XiaoHongShu, RSS, web, V2EX, LinkedIn, Xueqiu, Xiaoyuzhou podcasts, Exa search).
**Why Adopt**: Genuinely useful capability layer. Multi-backend architecture with real probing (not just `which()`), 160/162 tests pass, transparent about limitations. Each platform has ordered backend list — if one breaks, fallback takes over. **Installed**: v1.5.0 from GitHub (not on PyPI).
**Caveats**: Reddit has no zero-config path. Cookie-based auth carries platform ban risk. Some upstream tools have stalled maintenance (xhs-cli, rdt-cli). Requires `~/.agent-reach/` write access — sandbox fix applied.
**Relevance to Reasonix**: Web access tool for agent sessions. `agent-reach doctor` to check platform status.

### tolaria (refactoringhq/tolaria) — 16.1k ★
**What**: Desktop PKM app — files-first, Git-first, offline-first, AI-first (7 agent CLI integrations: Claude Code, Codex, Gemini, OpenCode, Hermes, Kiro, Pi). Rust/Tauri + React.
**Why Adopt**: Production-grade. 1,321 releases. AGPL-3.0. Filesystem-native (`.md` files), Git-backed vaults, keyboard-first design. 13K+ byte AGENTS.md for AI agents.
**Caveats**: Desktop app, not something you "install into" Reasonix.
**Relevance to Reasonix**: Reference architecture for desktop PKM + AI integration. AGPL-3.0.

### open-notebook (lfnovo/open-notebook) — 3.2k ★
**What**: Self-hosted Notebook LM alternative. Next.js + FastAPI + SurrealDB. 18 AI providers, multi-speaker podcast generation, LangGraph workflows, 14-language UI.
**Why Adopt**: Production-ready. 775 commits, 38 releases (v1.10.0), 198 tests, Docker Compose deployment. MIT license. Content transformations, vector search, comprehensive REST API.
**Caveats**: Bus factor = 1 (81% commits by author). No integration/E2E tests. Depends on niche SurrealDB.
**Relevance to Reasonix**: Reference for multi-provider AI orchestration + LangGraph workflows.

### Handy (cjpais/Handy) — 24.3k ★
**What**: Offline-first, open-source speech-to-text. 8 engine types (Whisper, Parakeet, Moonshine, SenseVoice, GigaAM, Canary, Cohere, streaming). Tauri v2. GPU acceleration on all platforms.
**Why Adopt**: Best open-source offline STT. 57 releases, 10 CI workflows, signed artifacts (minisign). Clean architecture with mpsc channel coordinator, RAII guards. Nix flake + Homebrew cask.
**Caveats**: Bus factor = 1 (75% commits by author). Feature freeze (stabilization mode). Whisper crashes on some configs. Wayland rough edges.

### drawio-skill (Agents365-ai/drawio-skill) — 4.3k ★
**What**: Agent skill for programmatic draw.io diagram generation. 10k+ shape index, 5 language extractors (Python/JS/Go/Rust), Graphviz auto-layout, AI brand logo resolver (321 logos).
**Why Adopt**: Real Python code (~5,500 LOC), not promptware. 21 tests, 20 releases, 157 commits. MIT license. Bilingual docs (EN + CN). Honest about limitations.
**Caveats**: Bus factor = 1 (97% commits by author). Vision self-check is promptware, not code. Shape index is a copy of jgraph/drawio-mcp's index.

### taste-skill (Leonxlnx/taste-skill) — 47.7k ★
**What**: Anti-"slop" design directions for AI agents doing frontend work. 13 variants: brutalist, minimalist, soft, brandkit, image-to-code, redesign, stitch, + more. Adjustable VARIANCE/MOTION/DENSITY dials.
**Why Adopt**: Demonstrably improves AI-generated UI. Research-backed laziness/truncation analysis. Well-documented scope boundaries (what it's NOT for). **Installed**: 13 sub-skills via install_source.
**Caveats**: Markdown instruction files, not a framework. v2 is experimental. No automated enforcement — depends on agent compliance.
**Relevance to Reasonix**: 13 design-direction project skills installed under `.reasonix/skills/`.

---

## Watch 👀

### ponytail (DietrichGebert/ponytail) — 42.9k ★
**What**: YAGNI system prompt + hooks reducing AI-generated code bloat by ~54% (up to 94% on over-build traps).
**Why Watch**: Best-in-class benchmark transparency. 68 tests. Real multi-platform engineering (14+ agent platform plugins). Authors disclosed their own contamination bug. 97 merged PRs in 8 days.
**Why Not Adopt**: 8 days old. 4 open security CVEs. Bus factor = 1. No lockfile. 42.9k stars in 8 days is suspicious. Check back in 30-60 days.

### skillspector (nvidia/skillspector) — 8.6k ★
**What**: Two-stage (static regex + LLM semantic) skill security scanner. 20 parallel analyzers. MCP tool poisoning detection (TP1-TP4). AST + taint tracking. YARA signatures.
**Why Watch**: NVIDIA-backed. 621 tests. Real vulnerability detection (tested against malicious skills). Apache 2.0. Ahead of the curve on MCP-specific patterns.
**Why Not Adopt**: 5 months old, marked Alpha. No formal releases. 52 open PRs. Critical open bugs (Zip Slip, import alias evasion). Wait for v2.3.0.

### improve (shadcn/improve) — 5.8k ★
**What**: Multi-stage audit pipeline prompt for AI agents: Plan → Execute → Review → Reconcile → Learn.
**Why Watch**: Elegant architecture. Exceptional docs. Self-dogfooding culture. Real hard rules, not vague advice.
**Why Not Adopt**: 10 days old. 9 files, 745 lines of pure promptware — no code, no tests, no CI, no empirical validation. Bus factor = 1.

### NVIDIA/Cosmos — 10.4k ★
**What**: World models (Cosmos3-Nano 16B / Super 64B) — text→image→video→audio→action in one checkpoint. Diffusers + vLLM-Omni + NIM deployment paths.
**Why Watch**: Real model weights on HuggingFace (134k downloads). Exceptional docs. Comprehensive benchmarks. Honest about limitations.
**Why Not Adopt**: This repo is cookbooks/docs only (99.8% Jupyter notebooks) — actual framework lives in cosmos-framework. Non-standard OpenMDW-1.1 license. NVIDIA hardware lock-in. <1 month old. Reasoner "Coming soon."

### openai/plugins — 3.3k ★
**What**: 177 Codex plugin manifests (Figma, Notion, Slack, GitHub, Gmail, Stripe, Vercel, Linear, etc.). OpenAI-maintained.
**Why Watch**: Valuable reference for plugin ecosystem design. Several MCP-backed plugins. Active maintenance (last commit 2 days ago).
**Why Not Adopt**: Not standalone software — config bundles for proprietary Codex. Mixed licensing. 280 commits likely squashed (shallow clone shows 1).

### pm-skills (phuryn/pm-skills) — 20.1k ★
**What**: 68 PM skills across 9 plugins (Opportunity Solution Trees, JTBD, Lean Canvas, Pretotyping, etc.).
**Why Watch**: Genuine PM substance, well-structured architecture, clean validation (0 errors/warnings). AI-tool-agnostic (SKILL.md format).
**Why Not Adopt**: 20.1k stars suspicious (phantom-star accusation #41 open). No CI/CD. Near-zero issue resolution. Single contributor. Content, not software.

### last30days-skill (mvanhorn/last30days-skill) — 41k ★
**What**: 41k-star social research skill. Already adopted in Reasonix project skill catalog (installed in h8 session).

### ian-xiaohei-illustrations (helloianneo/ian-xiaohei-illustrations) — 167 ★
**What**: Single-purpose Chinese-niche illustration style for Codex. Self-contained, real AI-generated example images.
**Why Watch/Adopt**: Adopt if you create Chinese content with AI agents. Skip otherwise. 1 commit (complete as-is).

### gpt-taste — included in taste-skill adoption
Packaged as `gpt-taste` sub-skill within the taste-skill bundle.

---

## Summary Table

| Verdict | Count | Repos |
|---------|-------|-------|
| **Adopt** | 8 | headroom, markitdown, Agent-Reach, tolaria, open-notebook, Handy, drawio-skill, taste-skill |
| **Watch** | 7 | ponytail, skillspector, improve, NVIDIA/Cosmos, openai/plugins, pm-skills, ian-xiaohei-illustrations |
| **Already adopted** | 2 | last30days-skill, gpt-taste |

**Installed this session**: headroom v0.26.0, markitdown v0.1.5 + MCP, agent-reach v1.5.0, taste-skill (13 sub-skills)
**Sandbox fix applied**: `~/.agent-reach` + `~/.local/share` added to write whitelist (`internal/sandbox/sandbox.go`)
