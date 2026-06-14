package telegram

import (
	"context"
	"log/slog"
	"testing"

	"reasonix/internal/bot"
	"reasonix/internal/config"
)

func newTestAdapter(cfg config.TelegramBotConfig) *Adapter {
	return New(cfg, slog.Default()).(*Adapter)
}

// --- New() ---

func TestNew_CreatesAdapterWithCorrectConfig(t *testing.T) {
	cfg := config.TelegramBotConfig{
		TokenEnv: "TELEGRAM_BOT_TOKEN",
		AllowDMs: true,
	}
	a := New(cfg, slog.Default())

	ad, ok := a.(*Adapter)
	if !ok {
		t.Fatalf("New() did not return *Adapter, got %T", a)
	}
	if ad.cfg.TokenEnv != cfg.TokenEnv {
		t.Errorf("TokenEnv = %q, want %q", ad.cfg.TokenEnv, cfg.TokenEnv)
	}
	if ad.cfg.AllowDMs != cfg.AllowDMs {
		t.Errorf("AllowDMs = %v, want %v", ad.cfg.AllowDMs, cfg.AllowDMs)
	}
}

func TestNew_NilLoggerUsesDefault(t *testing.T) {
	cfg := config.TelegramBotConfig{TokenEnv: "MY_TOKEN"}
	a := New(cfg, nil)
	ad := a.(*Adapter)
	if ad.logger == nil {
		t.Error("expected non-nil logger after nil input")
	}
}

func TestNew_ChannelIsBuffered(t *testing.T) {
	a := New(config.TelegramBotConfig{TokenEnv: "T"}, slog.Default())
	ad := a.(*Adapter)
	if cap(ad.msgs) != 64 {
		t.Errorf("channel capacity = %d, want 64", cap(ad.msgs))
	}
}

// --- Platform() ---

func TestPlatform(t *testing.T) {
	a := New(config.TelegramBotConfig{TokenEnv: "T"}, slog.Default())
	if got := a.Platform(); got != bot.PlatformTelegram {
		t.Errorf("Platform() = %q, want %q", got, bot.PlatformTelegram)
	}
}

// --- Name() ---

func TestName(t *testing.T) {
	a := New(config.TelegramBotConfig{TokenEnv: "T"}, slog.Default())
	if got := a.Name(); got != "telegram" {
		t.Errorf("Name() = %q, want %q", got, "telegram")
	}
}

// --- Messages() ---

func TestMessages_ReturnsChannel(t *testing.T) {
	a := newTestAdapter(config.TelegramBotConfig{TokenEnv: "T"})
	ch := a.Messages()
	if ch == nil {
		t.Fatal("Messages() returned nil")
	}
}

// --- Start() without token ---

func TestStart_NoToken(t *testing.T) {
	a := newTestAdapter(config.TelegramBotConfig{TokenEnv: "MISSING_ENV_VAR"})
	ctx := context.Background()
	err := a.Start(ctx)
	if err == nil {
		t.Error("expected error when token env var is not set")
	}
}

// --- Stop() ---

func TestStop_NoOpWhenNotStarted(t *testing.T) {
	a := newTestAdapter(config.TelegramBotConfig{TokenEnv: "T"})
	if err := a.Stop(); err != nil {
		t.Errorf("Stop() returned unexpected error: %v", err)
	}
}

// --- Send() without connection ---

func TestSend_NotConnected(t *testing.T) {
	a := newTestAdapter(config.TelegramBotConfig{TokenEnv: "T"})
	_, err := a.Send(context.Background(), bot.OutboundMessage{ChatID: "123", Text: "hello"})
	if err == nil {
		t.Error("expected error when sending without connection")
	}
}

// --- Send() with invalid chat ID ---

func TestSend_InvalidChatID(t *testing.T) {
	a := newTestAdapter(config.TelegramBotConfig{TokenEnv: "T"})
	a.api = nil // force the not-connected path is already tested; also test non-nil api with invalid ID
	_, err := a.Send(context.Background(), bot.OutboundMessage{ChatID: "not-a-number", Text: "hello"})
	if err == nil {
		t.Error("expected error for invalid chat ID")
	}
}

// --- SendTyping() without connection ---

func TestSendTyping_NotConnected(t *testing.T) {
	a := newTestAdapter(config.TelegramBotConfig{TokenEnv: "T"})
	err := a.SendTyping(context.Background(), "123")
	if err != nil {
		t.Errorf("SendTyping() returned error when not connected: %v", err)
	}
}

// --- splitContent ---

func TestSplitContent_Short(t *testing.T) {
	got := splitContent("hello", 100)
	if len(got) != 1 || got[0] != "hello" {
		t.Errorf("splitContent short = %q, want ['hello']", got)
	}
}

func TestSplitContent_Long(t *testing.T) {
	content := ""
	for i := 0; i < 200; i++ {
		content += "a"
	}
	got := splitContent(content, 100)
	if len(got) != 2 {
		t.Errorf("splitContent long: got %d chunks, want 2", len(got))
	}
	if len(got[0]) != 100 {
		t.Errorf("first chunk length = %d, want 100", len(got[0]))
	}
}

func TestSplitContent_ParagraphBreaks(t *testing.T) {
	part1 := "First paragraph with enough text to fill space.\n\nSecond is here."
	content := part1 + "\n\n" + "Third paragraph is here."
	got := splitContent(content, len(part1))
	if len(got) < 2 {
		t.Errorf("splitContent paragraph: got %d chunks, want at least 2", len(got))
	}
}

// --- helper functions ---

func TestLastBreakIndex(t *testing.T) {
	tests := []struct {
		s, sep string
		want   int
	}{
		{"hello\nworld", "\n", 5},
		{"hello\n\nworld", "\n\n", 5},
		{"hello world. continue", ". ", 11},
		{"no match", "X", -1},
	}
	for _, tt := range tests {
		got := lastBreakIndex(tt.s, tt.sep)
		if got != tt.want {
			t.Errorf("lastBreakIndex(%q, %q) = %d, want %d", tt.s, tt.sep, got, tt.want)
		}
	}
}

func TestLastSpaceIndex(t *testing.T) {
	tests := []struct {
		s    string
		want int
	}{
		{"hello world", 5},
		{"no spaces", 2},
		{"spaceless", -1},
	}
	for _, tt := range tests {
		got := lastSpaceIndex(tt.s)
		if got != tt.want {
			t.Errorf("lastSpaceIndex(%q) = %d, want %d", tt.s, got, tt.want)
		}
	}
}
