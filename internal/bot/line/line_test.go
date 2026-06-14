package line

import (
	"context"
	"log/slog"
	"testing"

	"reasonix/internal/bot"
	"reasonix/internal/config"
)

func newTestAdapter(cfg config.LineBotConfig) *Adapter {
	return New(cfg, slog.Default()).(*Adapter)
}

func TestNew_CreatesAdapterWithCorrectConfig(t *testing.T) {
	cfg := config.LineBotConfig{
		TokenEnv:  "LINE_CHANNEL_TOKEN",
		SecretEnv: "LINE_CHANNEL_SECRET",
		AllowDMs:  true,
	}
	a := New(cfg, slog.Default())

	ad, ok := a.(*Adapter)
	if !ok {
		t.Fatalf("New() did not return *Adapter, got %T", a)
	}
	if ad.cfg.TokenEnv != cfg.TokenEnv {
		t.Errorf("TokenEnv = %q, want %q", ad.cfg.TokenEnv, cfg.TokenEnv)
	}
	if ad.cfg.SecretEnv != cfg.SecretEnv {
		t.Errorf("SecretEnv = %q, want %q", ad.cfg.SecretEnv, cfg.SecretEnv)
	}
}

func TestNew_NilLoggerUsesDefault(t *testing.T) {
	cfg := config.LineBotConfig{TokenEnv: "MY_TOKEN", SecretEnv: "MY_SECRET"}
	a := New(cfg, nil)
	ad := a.(*Adapter)
	if ad.logger == nil {
		t.Error("expected non-nil logger after nil input")
	}
}

func TestNew_ChannelIsBuffered(t *testing.T) {
	a := New(config.LineBotConfig{TokenEnv: "T", SecretEnv: "S"}, slog.Default())
	ad := a.(*Adapter)
	if cap(ad.msgs) != 64 {
		t.Errorf("channel capacity = %d, want 64", cap(ad.msgs))
	}
}

func TestPlatform(t *testing.T) {
	a := New(config.LineBotConfig{TokenEnv: "T", SecretEnv: "S"}, slog.Default())
	if got := a.Platform(); got != bot.PlatformLine {
		t.Errorf("Platform() = %q, want %q", got, bot.PlatformLine)
	}
}

func TestName(t *testing.T) {
	a := New(config.LineBotConfig{TokenEnv: "T", SecretEnv: "S"}, slog.Default())
	if got := a.Name(); got != "line" {
		t.Errorf("Name() = %q, want %q", got, "line")
	}
}

func TestMessages_ReturnsChannel(t *testing.T) {
	a := newTestAdapter(config.LineBotConfig{TokenEnv: "T", SecretEnv: "S"})
	ch := a.Messages()
	if ch == nil {
		t.Fatal("Messages() returned nil")
	}
}

func TestStop(t *testing.T) {
	a := newTestAdapter(config.LineBotConfig{TokenEnv: "T", SecretEnv: "S"})
	if err := a.Stop(); err != nil {
		t.Errorf("Stop() error = %v", err)
	}
}

func TestSend_UninitializedClient(t *testing.T) {
	a := newTestAdapter(config.LineBotConfig{TokenEnv: "T", SecretEnv: "S"})
	_, err := a.Send(context.Background(), bot.OutboundMessage{
		ChatID: "room-1",
		Text:   "hello",
	})
	if err == nil {
		t.Error("expected error when client is nil")
	}
}

func TestSend_NoReplyToken(t *testing.T) {
	a := newTestAdapter(config.LineBotConfig{TokenEnv: "T", SecretEnv: "S"})
	_, err := a.Send(context.Background(), bot.OutboundMessage{
		Text:   "hello",
		ChatID: "room-1",
		// ReplyToMsgID intentionally empty — LINE requires it.
	})
	if err == nil {
		t.Error("expected error when ReplyToMsgID is empty")
	}
}

func TestSendTyping_NoOp(t *testing.T) {
	a := newTestAdapter(config.LineBotConfig{TokenEnv: "T", SecretEnv: "S"})
	if err := a.SendTyping(context.Background(), "room-1"); err != nil {
		t.Errorf("SendTyping() error = %v (expected no-op)", err)
	}
}

func TestLastNewlineBefore(t *testing.T) {
	tests := []struct {
		s      string
		limit  int
		expect int
	}{
		{"hello\nworld", 10, 5},
		{"no newlines here", 20, -1},
		{"a\nb\nc", 3, 3}, // \n at position 3 (just after b)
		{"", 10, -1},
	}
	for _, tt := range tests {
		got := lastNewlineBefore(tt.s, tt.limit)
		if got != tt.expect {
			t.Errorf("lastNewlineBefore(%q, %d) = %d, want %d", tt.s, tt.limit, got, tt.expect)
		}
	}
}
