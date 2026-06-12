package main

import (
	"os"
	"testing"

	"reasonix/internal/config"
)

// ── resolveToken ──────────────────────────────────────────────────

func TestResolveToken_FlagTakesPrecedence(t *testing.T) {
	os.Unsetenv("DISCORD_BOT_TOKEN")
	got := resolveToken("flag-token")
	if got != "flag-token" {
		t.Errorf("resolveToken(flag) = %q, want %q", got, "flag-token")
	}
}

func TestResolveToken_FallsBackToEnv(t *testing.T) {
	os.Setenv("DISCORD_BOT_TOKEN", "env-token")
	defer os.Unsetenv("DISCORD_BOT_TOKEN")
	got := resolveToken("")
	if got != "env-token" {
		t.Errorf("resolveToken(\"\") = %q, want %q (from env)", got, "env-token")
	}
}

func TestResolveToken_EmptyWhenNeitherSet(t *testing.T) {
	os.Unsetenv("DISCORD_BOT_TOKEN")
	got := resolveToken("")
	if got != "" {
		t.Errorf("resolveToken(\"\") = %q, want empty string", got)
	}
}

// ── resolveModelName ──────────────────────────────────────────────

func TestResolveModelName_FlagWins(t *testing.T) {
	cfg := &config.Config{
		DefaultModel: "config-default",
	}
	cfg.Bot.Model = "bot-model"
	got := resolveModelName("cli-model", cfg)
	if got != "cli-model" {
		t.Errorf("got %q, want %q (flag should win)", got, "cli-model")
	}
}

func TestResolveModelName_BotModelSecond(t *testing.T) {
	cfg := &config.Config{
		DefaultModel: "config-default",
	}
	cfg.Bot.Model = "bot-model"
	got := resolveModelName("", cfg)
	if got != "bot-model" {
		t.Errorf("got %q, want %q (Bot.Model should be fallback)", got, "bot-model")
	}
}

func TestResolveModelName_DefaultModelThird(t *testing.T) {
	cfg := &config.Config{
		DefaultModel: "config-default",
	}
	got := resolveModelName("", cfg)
	if got != "config-default" {
		t.Errorf("got %q, want %q (DefaultModel should be third fallback)", got, "config-default")
	}
}

func TestResolveModelName_HardcodedFallback(t *testing.T) {
	got := resolveModelName("", &config.Config{})
	if got != "deepseek-flash" {
		t.Errorf("got %q, want %q", got, "deepseek-flash")
	}
}

func TestResolveModelName_NilConfig(t *testing.T) {
	got := resolveModelName("", nil)
	if got != "deepseek-flash" {
		t.Errorf("resolveModelName(\"\", nil) = %q, want %q", got, "deepseek-flash")
	}
}

func TestResolveModelName_FlagWithNilConfig(t *testing.T) {
	got := resolveModelName("cli-model", nil)
	if got != "cli-model" {
		t.Errorf("got %q, want %q", got, "cli-model")
	}
}

// ── resolveWorkspaceRoot ──────────────────────────────────────────

func TestResolveWorkspaceRoot_FlagTakesPrecedence(t *testing.T) {
	got := resolveWorkspaceRoot("/custom/dir")
	if got != "/custom/dir" {
		t.Errorf("got %q, want %q", got, "/custom/dir")
	}
}

func TestResolveWorkspaceRoot_FallsBackToCwd(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Skipf("can't get cwd: %v", err)
	}
	got := resolveWorkspaceRoot("")
	if got != wd {
		t.Errorf("got %q, want current directory %q", got, wd)
	}
}
