# Handy (cjpais/Handy) Evaluation

## Surface Scan

| Metric | Value |
|--------|-------|
| **Stars** | 24,300 ★ |
| **Forks** | 2,100 |
| **Open Issues** | 94 |
| **Open PRs** | 81 (480 closed) |
| **Last Release** | v0.8.3 (Apr 28, 2026) |
| **License** | MIT |
| **Total Releases** | 57 |
| **Release Cadence** | ~every 5-7 days (v0.7.0 Jan 20 → v0.8.3 Apr 28) |
| **Languages** | Rust 45%, TypeScript 42%, Nix 9% |
| **Total Contributors** | 120 |
| **Total Commits** | 1,016 |

---

## Structure

### Layout
```
Handy/
├── src/                    # React/TypeScript frontend (120 files, 8.7k LOC)
│   ├── components/         # Settings UI, onboarding, model-selector
│   ├── stores/             # Zustand state (modelStore, settingsStore)
│   ├── i18n/               # Internationalization
│   └── overlay/            # Recording overlay window
├── src-tauri/              # Rust backend (48 files, 13.7k LOC)
│   ├── src/
│   │   ├── main.rs         # Entry point
│   │   ├── lib.rs          # Tauri setup, manager init, tray
│   │   ├── actions.rs      # Hotkey action dispatch + transcription pipeline
│   │   ├── clipboard.rs    # Text paste engine (clipboard/direct/script)
│   │   ├── input.rs        # Enigo keyboard/mouse simulation
│   │   ├── transcription_coordinator.rs  # Single-threaded lifecycle
│   │   ├── managers/       # Audio, model, transcription, history
│   │   ├── commands/       # Tauri IPC commands
│   │   ├── shortcut/       # Global keyboard shortcuts
│   │   └── settings.rs     # Settings persistence
│   ├── Cargo.toml          # 70+ Rust dependencies
│   └── tauri.conf.json     # Tauri v2 config
├── tests/                  # Playwright E2E tests
├── .github/workflows/      # 10 CI workflows
└── nix/                    # NixOS module + flake
```

### Build System
- **Frontend:** Bun + Vite + React 18 + TypeScript + Tailwind CSS 4
- **Backend:** Cargo + Rust + Tauri v2.10
- **Build command:** `bun install && bun run tauri build`
- **Mock test adapter:** CI swaps in `transcription_mock.rs` to avoid compiling whisper/Vulkan

### Tests
- **65 Rust unit tests** across 10 files (clipboard, model, history, settings, audio_toolkit, portable, etc.)
- **2 Playwright E2E tests** (basic HTML structure)
- **11 CI workflows:** test (Rust), build, release, code-quality, playwright, nix-check, PR test build, main build
- PR checks reduced from ~30 min to ~1 min via mock adapter

---

## Deep Dive: Hero Feature Traces

### Feature 1: Hotkey-to-Transcription Pipeline (The Core Loop)

**README claim:** "Press a configurable keyboard shortcut → speak → release → get transcribed text pasted"

**Trace:**
1. `src-tauri/src/main.rs` → calls `handy_app_lib::run(cli_args)`
2. `src-tauri/src/lib.rs::run()` → sets up Tauri, creates `TranscriptionCoordinator`, registers global shortcuts
3. Global shortcut fires → `shortcut/` module → sends event to `TranscriptionCoordinator::send_input()`
4. `src-tauri/src/transcription_coordinator.rs` → single-threaded state machine (`Idle` → `Recording` → `Processing`) using an `mpsc` channel with 30ms debounce
5. `src-tauri/src/actions.rs` → `TranscribeAction::start()` → loads ASR model + VAD in parallel, initiates recording via `AudioRecordingManager::try_start_recording()`, plays audio feedback, shows overlay
6. `TranscribeAction::stop()` → stops recording → `TranscriptionManager::transcribe(samples)` → saves WAV concurrently → `process_transcription_output()` (handles Chinese variant conversion + LLM post-processing) → `utils::paste()` 
7. `src-tauri/src/clipboard.rs::paste()` → supports 4 methods: clipboard (Ctrl+V), direct typing (enigo), external script, or native Linux tools (wtype/xdotool/dotool/kwtype/ydotool)
8. Returns text to foreground app

**Reality:** ✅ Fully implemented and traced. The architecture is clean and well-structured. The `TranscriptionCoordinator` serializes all events through one thread to eliminate race conditions. The `FinishGuard` drop-guard pattern ensures pipeline completion signals are always sent.

---

### Feature 2: Multiple Speech Recognition Models

**README claim:** "Whisper models (Small/Medium/Turbo/Large) with GPU acceleration + Parakeet V3 CPU-optimized"

**Trace:**
1. `src-tauri/src/managers/model.rs::ModelManager` → discovers models (pre-packaged + custom `.bin` files), manages downloads
2. `src-tauri/src/managers/transcription.rs` → `LoadedEngine` enum with **8 variants**:
   - `Whisper(WhisperEngine)` — GPU acceleration via Metal/Vulkan/CUDA/DirectML
   - `Parakeet(ParakeetModel)` — CPU-optimized ONNX
   - `Moonshine(MoonshineModel)` + `MoonshineStreaming(StreamingModel)`
   - `SenseVoice(SenseVoiceModel)` — multilingual
   - `GigaAM(GigaAMModel)` — Russian-optimized
   - `Canary(CanaryModel)` — Nvidia's model
   - `Cohere(CohereModel)` — newest addition (v0.8.2)
3. GPU enumeration runs async on startup (pre-warmed via background thread) to avoid UI freeze
4. Model unloads after configurable idle timeout (saves RAM)
5. Custom Whisper GGML models auto-discovered from `models/` directory

**Reality:** ✅ Exceeds README claims. Actually supports 7+ engine types, not just Whisper + Parakeet. GPU acceleration configurable per-accelerator. Known issue: Whisper models crash on some Windows/Linux configurations (acknowledged in README).

---

### Feature 3: LLM Post-Processing Pipeline

**README claim:** Not explicitly in the main README, but described in roadmap as "post-processing" — the architecture reveals a sophisticated LLM pipeline.

**Trace:**
1. `src-tauri/src/actions.rs::post_process_transcription()` → async function called when `post_process: true`
2. Supports **7+ provider types**: OpenAI-compatible, OpenRouter, Z.AI, AWS Bedrock (Mantle), Custom, Apple Intelligence (macOS), and more via config
3. **Structured output mode** (`provider.supports_structured_output`) — sends JSON schema alongside the request for deterministic parsing of the `transcription` field
4. **Legacy mode** — replaces `${output}` template variable in the prompt
5. **Apple Intelligence integration** — uses native Swift APIs via `tauri-nspanel` bridge (`src-tauri/swift/apple_intelligence.swift`)
6. **Reasoning config** — disables thinking-mode for local models to avoid latency
7. Chinese variant conversion via `ferrous-opencc` (OpenCC) handles Simplified/Traditional Chinese
8. Custom words via Whisper's `initial_prompt` field

**Reality:** ✅ Sophisticated implementation with structured output support, multiple providers, and graceful fallbacks. Feature is well-abstracted and extensible via the provider system.

---

## Community Health

| Metric | Data |
|--------|------|
| **Primary maintainer** | cjpais (759 commits, ~75% of total) |
| **Top contributors** | Viren Mohindra (24), Vlad Gerasimov (24), Evgeny Khudoba (12) |
| **Total contributors** | 120 |
| **First-time contributors** | Multiple per release (9 in v0.8.3, 5 in v0.8.2) |
| **Open PRs / Closed** | 81 open / 480 closed (~85% close rate) |
| **Issue response time** | Generally within 1-2 days (many issues get quick triage) |
| **Bus factor** | **Moderate concern** — 1 person dominates commits, but community is growing |
| **Feature Freeze** | Currently in effect; new feature PRs rejected; bug fixes prioritized |
| **Sponsors** | Wordcab, Epicenter, Bolt AI |
| **Community channels** | Discord, email, GitHub Discussions |

---

## Claims vs. Reality

| Claim | Status | Evidence |
|-------|--------|----------|
| "Free, open source, and extensible speech-to-text" | ✅ **True** | MIT license, 57 releases, extensible via custom models + LLM providers |
| "Works completely offline" | ✅ **True** | All inference local; LLM post-processing is optional and provider-based |
| "Cross-platform: Windows, macOS, Linux" | ✅ **True** | CI builds for all three, platform-specific dependencies managed |
| "Privacy-focused — your voice stays on your computer" | ✅ **True** | No telemetry, opt-in analytics planned, fully local processing |
| "Whisper models with GPU acceleration" | ✅ **True** | Metal (macOS), Vulkan (Linux), DirectML (Windows), CUDA |
| "Parakeet V3 CPU-optimized with auto language detection" | ✅ **True** | CPU-only operation, ~5x real-time on i5 |
| "Works on macOS Intel and Apple Silicon" | ✅ **True** | Separate arm64/x64 builds, Homebrew cask available |
| "50+ releases" | ✅ **True** | 57 releases, frequent cadence |
| "Wayland support" | ⚠️ **Partial** | Requires wtype/dotool; overlay issues; GNOME/KDE workarounds documented; limited on KDE Wayland |
| "Completely stable" | ⚠️ **Known issues** | Whisper model crashes on some configs; WebKit DMA-BUF issues on Linux; memory leak in overlay |

---

## Red Flags

1. **🔴 Bus Factor (Single Maintainer):** cjpais accounts for ~75% of all commits. While community is growing, the project has a bus factor of ~1. The feature freeze suggests maintainer is stretched thin.

2. **🔴 94 Open Issues + 81 Open PRs:** The PR backlog is significant (81 open vs 480 closed). This suggests the maintainer struggles to review contributions in a timely manner.

3. **🔴 Whisper Model Crashes:** Acknowledged major issue where Whisper models crash on certain Windows/Linux configurations. "Help wanted" tagged — no fix yet.

4. **🔴 Wayland Support Is Rough:** Only partial Wayland support with multiple caveats. Requires manual setup of wtype/dotool. Overlay issues on KDE. Requires DE-specific workarounds. This is a significant Linux UX issue.

5. **🔴 Feature Freeze:** Currently rejecting new features. Indicates the project is in a stabilization phase — not the best time to contribute features.

6. **🔴 macOS Build Requires Full Xcode:** `build.rs` fails with only Command Line Tools (requires Xcode for FoundationModelsMacros plugin). Open issue #1448.

---

## Green Flags

1. **🟢 Massive Community Adoption:** 24,300 stars and 2,100 forks indicate strong community validation and interest.

2. **🟢 Excellent Documentation:** README is comprehensive with clear known-issues section, troubleshooting guides, platform-specific notes, manual model installation, release verification, and Linux startup troubleshooting. Multiple docs files (CONTRIBUTING.md, BUILD.md, AGENTS.md, CRUSH.md).

3. **🟢 Strong CI/CD Pipeline:** 10 GitHub Actions workflows covering tests (Rust + Playwright), builds, releases, code quality, Nix checks. Smart mock adapter for fast CI (~1 min PR checks). Automated releases with 29 assets per release.

4. **🟢 Architectural Quality:** Clean separation of concerns — Tauri commands, managers, actions, transcription coordinator with mpsc channel, drop-guard patterns, RAII guards, structured error types, async/fallback patterns throughout.

5. **🟢 Model Diversity:** 8 engine types (Whisper, Parakeet, Moonshine, SenseVoice, GigaAM, Canary, Cohere, streaming) — unmatched by any other open-source speech-to-text app.

6. **🟢 Release Integrity:** All artifacts signed with minisign, verification instructions provided, checksums in release assets.

7. **🟢 i18n + Accessibility:** Full internationalization (translations for 15+ languages), Apple Intelligence integration, accessibility permissions onboarding.

8. **🟢 Extensibility:** Custom Whisper models auto-discovered, external paste scripts, multiple LLM providers, CLI flags for remote control, Raycast integration, portable mode.

9. **🟢 Nix Support:** Flake-based reproducible builds with NixOS module — excellent for declarative setups.

---

## Verdict: **Adopt** ⭐

**Handy is the best open-source, offline-first speech-to-text application available.** It earns a strong **Adopt** recommendation for individual use and most team scenarios.

**Adopt it if you:**
- Want a free, private, offline speech-to-text tool that actually works
- Need multiple model backends (Whisper + Parakeet + others) with GPU acceleration
- Value cross-platform support (especially macOS + Windows)
- Are comfortable with some Linux rough edges (Wayland limitations documented)

**Be aware of:**
- Bus factor is real — have a fork strategy if this is production-critical
- The feature freeze means no new features until stability improves
- Linux Wayland experience requires manual setup and has caveats
- Whisper model crashes affect some configurations (use Parakeet as fallback)

**For teams / commercial use:** Fork and stabilize. The MIT license makes this easy. The architecture is clean enough to maintain in-house. The 57-release track record and active community suggest the project will continue improving.

**Rating:** ⭐⭐⭐⭐½ (4.5/5) — Held back only by single-maintainer risk and known stability issues being actively worked on.
