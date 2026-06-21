// hermes_fingerprint_test.go — system-wide guard tests for Hermes code injection
// points in upstream-shared Go files. Upstream merges can silently drop injected
// code when git merge doesn't flag a conflict.
//
// Documented losses (code that existed and was silently dropped by upstream merges):
//   - runWithAutoResume() in controller_turn.go (lost in d75a84a8; unrestorable —
//     architecture changed, IsMaxStepsPause intentionally removed as dead code)
//   - workshopSynthesizer wiring in boot.go (lost; restored h53)
//   - languagePolicy/finalizeSystemPrompt in boot.go (lost; restored h53)
//   - sqz CompressToolOutput wiring in boot.go (lost, fixed h52)
//   - SettingsView Hotbar/Profiles fields in settings_app.go (lost, fixed h52)
//   - render.go Hermes TOML sections (lost, fixed h11)
//   - hotbar keyboard handler in App.tsx (lost 9 days, fixed b6de38ac)
//   - sound field in render.go Notifications (never rendered; added h53)
//
// Intentional removals (not losses):
//   - beep() — removed in 08c0404a (h48) as dead code cleanup
//   - Schedule() → ScheduleNextRuns() — renamed, functionally preserved
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

	checks := map[string][]string{
		// ── internal/boot/boot.go — Hermes agent wiring ──
		"internal/boot/boot.go": {
			"agentlog.Init",
			"applyExchangeRate",
			"CompressToolOutput",
			"VisionProv",
			"ctrl.SetMesh",
			"ctrl.SetLearner",
			"resolveAuxProviders",
			"RemoteSandboxURL",
			"RemoteSandboxToken",
			"WorkshopThreshold",
			"workshopSynthesizer",
			"WorkshopSynthesisText",
			"workshopThreshold",
			"truncateHead",
			"func languagePolicy",
		},

		// ── internal/control/controller.go — Hermes struct fields ──
		"internal/control/controller.go": {
			`reasonix/internal/mesh"`,
			`reasonix/internal/scheduler"`,
			`reasonix/internal/learn"`,
			"\tschedule ",
			"\tmesh ",
			"\tlearner ",
			"learn.Learner",
			"scheduler.Scheduler",
			"Schedule *scheduler.Scheduler",
			"Mesh *mesh.Mesh",
			"SendCtx",
		},

		// ── internal/control/controller_approval.go — Hermes completion sound ──
		"internal/control/controller_approval.go": {
			"ToggleSound",
		},

		// ── internal/config/config.go — Hermes config types ──
		"internal/config/config.go": {
			"HotbarConfig struct",
			"CompressToolOutputEnabled",
			"ActiveProfile",
			"BillingConfig",
			"LearnConfig",
			"MeshConfig",
			"ScheduleConfig",
			"CollabConfig",
			"MarketplaceConfig",
			"EmbeddingConfig",
			"AgentLogConfig",
			"DiscordBotConfig",
			"TelegramBotConfig",
			"LineBotConfig",
			"SlackBotConfig",
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
			"Notifications.Sound",
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
