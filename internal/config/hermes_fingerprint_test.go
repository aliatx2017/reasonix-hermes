// hermes_fingerprint_test.go — system-wide guard tests for Hermes code injection
// points in upstream-shared Go files. Upstream merges can silently drop injected
// code when git merge doesn't flag a conflict (text around the injection changes
// but the injected block itself doesn't overlap — merge drops it cleanly).
//
// Documented losses:
//   - sqz CompressToolOutput wiring in boot.go (lost, fixed h52)
//   - SettingsView Hotbar/Profiles fields in settings_app.go (lost, fixed h52)
//   - render.go Hermes TOML sections (lost, fixed h11)
//
// Every fingerprint was verified against live code before commit.
// If any check fails, an upstream merge just destroyed Hermes code.

package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHermesFingerprintsGo(t *testing.T) {
	repoRoot := findHermesRepoRoot(t)

	// Each entry: file path relative to repo root → required substrings.
	checks := map[string][]string{
		// ── internal/boot/boot.go — Hermes agent wiring ──
		"internal/boot/boot.go": {
			"agentlog.Init",        // operational logging init
			"applyExchangeRate",    // CNY→USD exchange rate
			"CompressToolOutput",   // sqz compressor wiring
			"VisionProv",           // auxiliary vision provider
			"ctrl.SetMesh",         // mesh council wiring
			"ctrl.SetLearner",      // learner pattern detection
		},

		// ── internal/control/controller.go — Hermes struct fields ──
		// Uses raw string literals because controller.go uses tab indentation.
		"internal/control/controller.go": {
			`reasonix/internal/mesh"`,
			`reasonix/internal/scheduler"`,
			`reasonix/internal/learn"`,
			"\tschedule ", // tab-indented field (line 92-ish)
			"\tmesh ",     // tab-indented field (line 93-ish)
			"\tlearner ",  // tab-indented field (line 94-ish)
			"Schedule *scheduler.Scheduler",
			"Mesh *mesh.Mesh",
		},

		// ── internal/config/config.go — Hermes config types ──
		"internal/config/config.go": {
			"HotbarConfig struct",       // hotbar config type
			"CompressToolOutputEnabled", // compress enabled getter
			"ActiveProfile",             // active profile field
			"BillingConfig",             // billing config type
			"LearnConfig",               // learn config type
			"MeshConfig",                // mesh config type
			"ScheduleConfig",            // schedule config type
			"CollabConfig",              // collab config type
			"MarketplaceConfig",         // marketplace config type
			"EmbeddingConfig",           // embedding config type
			"AgentLogConfig",            // agent log config type
			"DiscordBotConfig",          // Discord bot config type
			"TelegramBotConfig",         // Telegram bot config type
			"LineBotConfig",             // LINE bot config type
			"SlackBotConfig",            // Slack bot config type
		},

		// ── internal/config/render.go — Hermes TOML section rendering ──
		"internal/config/render.go": {
			"[desktop.hotbar]",
			"[schedule]",
			"[learn]",
			"[mesh]",
			"[bot.discord]",
			"[bot.telegram]",
			"[bot.line]",
			"[bot.slack]",
			"active_profile",
		},

		// ── internal/config/edit_test.go — Hermes theme test ──
		"internal/config/edit_test.go": {
			`"Hermes", "hermes"`,
		},

		// ── desktop/settings_app.go — Hermes SettingsView + Wails bindings ──
		"desktop/settings_app.go": {
			"// Hermes —",
			"SetDesktopHotbar",
			"SetProfiles",
			"func hotbarView(",
		},

		// ── desktop/main.go — branding ──
		"desktop/main.go": {
			`"Reasonix-Hermes"`,
		},

		// ── desktop/tray.go — gold Hermes tray icon ──
		"desktop/tray.go": {
			"isHermesTheme",
		},
	}

	for file, fingerprints := range checks {
		path := filepath.Join(repoRoot, file)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("%s: cannot read: %v", file, err)
			continue
		}
		content := string(data)
		for _, fp := range fingerprints {
			if !strings.Contains(content, fp) {
				t.Errorf("%s: MISSING %q — upstream merge may have dropped Hermes code", file, fp)
			}
		}
	}
}

func findHermesRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("cannot find repo root (no go.mod found)")
		}
		dir = parent
	}
}
