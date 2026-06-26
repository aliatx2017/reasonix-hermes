package config

import "testing"

func TestDefaultAutoPlanOff(t *testing.T) {
	if got := Default().Agent.AutoPlan; got != "off" {
		t.Fatalf("default auto_plan = %q, want off", got)
	}
}

func TestDefaultReasoningLanguageAuto(t *testing.T) {
	if got := Default().ReasoningLanguage(); got != "auto" {
		t.Fatalf("default reasoning_language = %q, want auto", got)
	}
}

func TestDefaultMemoryCompilerEnabled(t *testing.T) {
	cfg := Default()
	if !cfg.MemoryCompilerEnabled() {
		t.Fatal("default memory compiler = false, want true")
	}
}

func TestDefaultDesktopAppearanceAutoGraphite(t *testing.T) {
	cfg := Default()
	if got := cfg.DesktopTheme(); got != "auto" {
		t.Fatalf("default desktop theme = %q, want auto", got)
	}
	if got := cfg.DesktopThemeStyle(); got != "" {
		t.Fatalf("default desktop theme style = %q, want empty so frontend resolves graphite", got)
	}
}

func TestDefaultDesktopMetricsOn(t *testing.T) {
	cfg := Default()
	if !cfg.DesktopMetrics() {
		t.Fatal("default desktop metrics = false, want true")
	}
	disabled := false
	cfg.Desktop.Metrics = &disabled
	if cfg.DesktopMetrics() {
		t.Fatal("desktop metrics explicit false = true, want false")
	}
}

func TestCompressToolOutputEnabledDefault(t *testing.T) {
	// nil config → default true
	if got := (*Config)(nil).CompressToolOutputEnabled(); !got {
		t.Fatal("nil Config.CompressToolOutputEnabled() = false, want true")
	}
	// default config → true
	if got := Default().CompressToolOutputEnabled(); !got {
		t.Fatal("Default().CompressToolOutputEnabled() = false, want true")
	}
	// explicit false → false
	cfg := Default()
	disabled := false
	cfg.Agent.CompressToolOutput = &disabled
	if cfg.CompressToolOutputEnabled() {
		t.Fatal("explicit false = true, want false")
	}
}

// --- Hermes config field guard tests ---
// Each verifies a custom struct field exists on Config and has the expected
// zero-value type. If an upstream merge drops the field, the test fails at
// compile time.

func TestBillingConfigExists(t *testing.T) {
	cfg := Default()
	// Must compile: cfg.Billing.AutoExchangeRate exists.
	if cfg.Billing.AutoExchangeRate { // zero-value is false
		t.Fatal("default Billing.AutoExchangeRate = true, want false")
	}
}

func TestLearnConfigExists(t *testing.T) {
	cfg := Default()
	if cfg.Learn.Enabled {
		t.Fatal("default Learn.Enabled = true, want false")
	}
	if cfg.Learn.MaxPatterns != 0 {
		t.Fatal("default Learn.MaxPatterns != 0")
	}
}

func TestMeshConfigExists(t *testing.T) {
	cfg := Default()
	if cfg.Mesh.Enabled {
		t.Fatal("default Mesh.Enabled = true, want false")
	}
}

func TestScheduleConfigExists(t *testing.T) {
	cfg := Default()
	if cfg.Schedule.Tasks != nil {
		t.Fatal("default Schedule.Tasks != nil")
	}
}

func TestCollabConfigExists(t *testing.T) {
	cfg := Default()
	if cfg.Collab.Enabled {
		t.Fatal("default Collab.Enabled = true, want false")
	}
}

func TestMarketplaceConfigExists(t *testing.T) {
	cfg := Default()
	if cfg.Marketplace.LobeHub.Enabled {
		t.Fatal("default Marketplace.LobeHub.Enabled = true, want false")
	}
}

func TestEmbeddingConfigExists(t *testing.T) {
	cfg := Default()
	if cfg.Embedding.Provider != "" {
		t.Fatalf("default Embedding.Provider = %q, want empty", cfg.Embedding.Provider)
	}
}

func TestBotDiscordConfigExists(t *testing.T) {
	cfg := Default()
	if cfg.Bot.Discord.Enabled {
		t.Fatal("default Bot.Discord.Enabled = true, want false")
	}
}

func TestBotTelegramConfigExists(t *testing.T) {
	cfg := Default()
	if cfg.Bot.Telegram.Enabled {
		t.Fatal("default Bot.Telegram.Enabled = true, want false")
	}
}

func TestBotLineConfigExists(t *testing.T) {
	cfg := Default()
	if cfg.Bot.Line.Enabled {
		t.Fatal("default Bot.Line.Enabled = true, want false")
	}
}

func TestBotSlackConfigExists(t *testing.T) {
	cfg := Default()
	if cfg.Bot.Slack.Enabled {
		t.Fatal("default Bot.Slack.Enabled = true, want false")
	}
}

func TestNotificationsSoundFieldExists(t *testing.T) {
	cfg := Default()
	if cfg.Notifications.Sound { // zero-value is false
		t.Fatal("default Notifications.Sound = true, want false")
	}
}

func TestSandboxRemoteFieldsExist(t *testing.T) {
	cfg := Default()
	if cfg.Sandbox.RemoteSandboxURL != "" {
		t.Fatalf("default Sandbox.RemoteSandboxURL = %q, want empty", cfg.Sandbox.RemoteSandboxURL)
	}
	if cfg.Sandbox.RemoteSandboxToken != "" {
		t.Fatalf("default Sandbox.RemoteSandboxToken = %q, want empty", cfg.Sandbox.RemoteSandboxToken)
	}
}

func TestAgentAuxiliaryFieldsExist(t *testing.T) {
	cfg := Default()
	if cfg.Agent.Auxiliary.Compression.IsSet() {
		t.Fatal("default Agent.Auxiliary.Compression.IsSet() = true, want false")
	}
	if cfg.Agent.Auxiliary.Vision.IsSet() {
		t.Fatal("default Agent.Auxiliary.Vision.IsSet() = true, want false")
	}
	if cfg.Agent.Auxiliary.WebExtract.IsSet() {
		t.Fatal("default Agent.Auxiliary.WebExtract.IsSet() = true, want false")
	}
}

func TestAgentlogConfigExists(t *testing.T) {
	cfg := Default()
	// AgentLog fields exist; zero-values are reasonable defaults.
	if cfg.AgentLog.Enabled {
		t.Fatal("default AgentLog.Enabled = true, want false")
	}
}
