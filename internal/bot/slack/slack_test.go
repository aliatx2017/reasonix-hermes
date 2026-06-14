package slack

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"reasonix/internal/bot"
	"reasonix/internal/config"
)

func newTestAdapter(cfg config.SlackBotConfig) *Adapter {
	return New(cfg, slog.Default()).(*Adapter)
}

// --- New() ---

func TestNew_CreatesAdapterWithCorrectConfig(t *testing.T) {
	cfg := config.SlackBotConfig{
		TokenEnv:    "SLACK_BOT_TOKEN",
		AppTokenEnv: "SLACK_APP_TOKEN",
	}
	a := New(cfg, slog.Default())

	ad, ok := a.(*Adapter)
	if !ok {
		t.Fatalf("New() did not return *Adapter, got %T", a)
	}
	if ad.cfg.TokenEnv != cfg.TokenEnv {
		t.Errorf("TokenEnv = %q, want %q", ad.cfg.TokenEnv, cfg.TokenEnv)
	}
	if ad.cfg.AppTokenEnv != cfg.AppTokenEnv {
		t.Errorf("AppTokenEnv = %q, want %q", ad.cfg.AppTokenEnv, cfg.AppTokenEnv)
	}
}

func TestNew_NilLoggerUsesDefault(t *testing.T) {
	cfg := config.SlackBotConfig{TokenEnv: "MY_TOKEN", AppTokenEnv: "MY_APP_TOKEN"}
	a := New(cfg, nil)
	ad := a.(*Adapter)
	if ad.logger == nil {
		t.Error("expected non-nil logger after nil input")
	}
}

func TestNew_ChannelIsBuffered(t *testing.T) {
	a := New(config.SlackBotConfig{TokenEnv: "T", AppTokenEnv: "A"}, slog.Default())
	ad := a.(*Adapter)
	if cap(ad.msgCh) != 100 {
		t.Errorf("channel capacity = %d, want 100", cap(ad.msgCh))
	}
}

// --- Platform() ---

func TestPlatform(t *testing.T) {
	a := New(config.SlackBotConfig{TokenEnv: "T", AppTokenEnv: "A"}, slog.Default())
	if got := a.Platform(); got != bot.PlatformSlack {
		t.Errorf("Platform() = %q, want %q", got, bot.PlatformSlack)
	}
}

// --- Name() ---

func TestName(t *testing.T) {
	a := New(config.SlackBotConfig{TokenEnv: "T", AppTokenEnv: "A"}, slog.Default())
	if got := a.Name(); got != "slack" {
		t.Errorf("Name() = %q, want %q", got, "slack")
	}
}

// --- Messages() ---

func TestMessages_ReturnsChannel(t *testing.T) {
	a := newTestAdapter(config.SlackBotConfig{TokenEnv: "T", AppTokenEnv: "A"})
	ch := a.Messages()
	if ch == nil {
		t.Fatal("Messages() returned nil")
	}
}

// --- Start() without tokens ---

func TestStart_NoToken(t *testing.T) {
	a := newTestAdapter(config.SlackBotConfig{TokenEnv: "MISSING_ENV_VAR", AppTokenEnv: "SLACK_APP_TOKEN"})
	ctx := context.Background()
	err := a.Start(ctx)
	if err == nil {
		t.Error("expected error when token env var is not set")
	}
}

func TestStart_NoAppToken(t *testing.T) {
	t.Setenv("SLACK_BOT_TOKEN", "xoxb-test")
	a := newTestAdapter(config.SlackBotConfig{TokenEnv: "SLACK_BOT_TOKEN", AppTokenEnv: "MISSING_APP_TOKEN"})
	ctx := context.Background()
	err := a.Start(ctx)
	if err == nil {
		t.Error("expected error when app token env var is not set")
	}
}

// --- Stop() ---

func TestStop_NoOpWhenNotStarted(t *testing.T) {
	a := newTestAdapter(config.SlackBotConfig{TokenEnv: "T", AppTokenEnv: "A"})
	if err := a.Stop(); err != nil {
		t.Errorf("Stop() returned unexpected error: %v", err)
	}
}

func TestStop_Idempotent(t *testing.T) {
	a := newTestAdapter(config.SlackBotConfig{TokenEnv: "T", AppTokenEnv: "A"})
	if err := a.Stop(); err != nil {
		t.Errorf("first Stop() returned error: %v", err)
	}
	if err := a.Stop(); err != nil {
		t.Errorf("second Stop() returned error: %v", err)
	}
}

// --- Send() without connection ---

func TestSend_NotConnected(t *testing.T) {
	a := newTestAdapter(config.SlackBotConfig{TokenEnv: "T", AppTokenEnv: "A"})
	_, err := a.Send(context.Background(), bot.OutboundMessage{ChatID: "C123", Text: "hello"})
	if err == nil {
		t.Error("expected error when sending without connection")
	}
}

// --- SendTyping() ---

func TestSendTyping_NoOp(t *testing.T) {
	a := newTestAdapter(config.SlackBotConfig{TokenEnv: "T", AppTokenEnv: "A"})
	err := a.SendTyping(context.Background(), "C123")
	if err != nil {
		t.Errorf("SendTyping() returned unexpected error: %v", err)
	}
}

// --- WebhookURL() ---

func TestWebhookURL_ReturnsEmpty(t *testing.T) {
	a := newTestAdapter(config.SlackBotConfig{TokenEnv: "T", AppTokenEnv: "A"})
	if got := a.WebhookURL(); got != "" {
		t.Errorf("WebhookURL() = %q, want empty string", got)
	}
}

// --- splitSlackMessage ---

func TestSplitSlackMessage_Short(t *testing.T) {
	got := splitSlackMessage("hello")
	if len(got) != 1 || got[0] != "hello" {
		t.Errorf("splitSlackMessage short = %q, want ['hello']", got)
	}
}

func TestSplitSlackMessage_Long(t *testing.T) {
	content := ""
	for i := 0; i < 5000; i++ {
		content += "a"
	}
	got := splitSlackMessage(content)
	if len(got) != 2 {
		t.Errorf("splitSlackMessage long: got %d chunks, want 2", len(got))
	}
	if len(got[0]) != 4000 {
		t.Errorf("first chunk length = %d, want 4000", len(got[0]))
	}
}

func TestSplitSlackMessage_ParagraphBreaks(t *testing.T) {
	// Build content where a paragraph boundary sits just before 4000 bytes.
	part1 := ""
	for i := 0; i < 3980; i++ {
		part1 += "a"
	}
	part1 += "\n\n"
	content := part1 + "Second paragraph is here."
	got := splitSlackMessage(content)
	if len(got) < 2 {
		t.Errorf("splitSlackMessage paragraph: got %d chunks, want at least 2", len(got))
	}
	if len(got[0]) < 3900 {
		t.Errorf("first chunk too short (%d), should have stopped at paragraph break", len(got[0]))
	}
}

func TestSplitSlackMessage_AtMaxLen(t *testing.T) {
	content := ""
	for i := 0; i < 4000; i++ {
		content += "b"
	}
	got := splitSlackMessage(content)
	if len(got) != 1 || len(got[0]) != 4000 {
		t.Errorf("splitSlackMessage exactly maxLen: got %d chunks, want 1", len(got))
	}
}

// --- parseSlackTS ---

func TestParseSlackTS_Valid(t *testing.T) {
	ts := "1234567890.123456"
	got := parseSlackTS(ts)
	want := time.Unix(1234567890, 0)
	if !got.Equal(want) {
		t.Errorf("parseSlackTS(%q) = %v, want %v", ts, got, want)
	}
}

func TestParseSlackTS_Malformed(t *testing.T) {
	before := time.Now()
	got := parseSlackTS("not-a-timestamp")
	if got.Before(before.Add(-time.Second)) {
		t.Errorf("parseSlackTS malformed returned %v, want time.Now()", got)
	}
}

func TestParseSlackTS_Empty(t *testing.T) {
	before := time.Now()
	got := parseSlackTS("")
	if got.Before(before.Add(-time.Second)) {
		t.Errorf("parseSlackTS empty returned %v, want time.Now()", got)
	}
}

// --- enqueueMessage drops when not running ---

func TestEnqueueMessage_NotRunning(t *testing.T) {
	a := newTestAdapter(config.SlackBotConfig{TokenEnv: "T", AppTokenEnv: "A"})
	// Drain the channel first to ensure it's clear
	drainCh(a.msgCh)
	a.enqueueMessage("hello", "U123", "C456", "ts.001", "")
	select {
	case <-a.msgCh:
		t.Error("enqueueMessage should not enqueue when not running")
	default:
		// expected — no message enqueued
	}
}

// --- enqueueMessage with thread ---

func TestEnqueueMessage_UsesThreadedChatID(t *testing.T) {
	a := newTestAdapter(config.SlackBotConfig{TokenEnv: "T", AppTokenEnv: "A"})
	a.mu.Lock()
	a.running = true
	a.mu.Unlock()
	drainCh(a.msgCh)

	a.enqueueMessage("threaded reply", "U123", "C456", "ts.001", "thread-ts-1")

	select {
	case msg := <-a.msgCh:
		if msg.ChatID != "C456:thread-ts-1" {
			t.Errorf("ChatID = %q, want 'C456:thread-ts-1'", msg.ChatID)
		}
	default:
		t.Error("message should be enqueued when running")
	}
}

// --- enqueueMessage without thread ---

func TestEnqueueMessage_UsesBareChatID(t *testing.T) {
	a := newTestAdapter(config.SlackBotConfig{TokenEnv: "T", AppTokenEnv: "A"})
	a.mu.Lock()
	a.running = true
	a.mu.Unlock()
	drainCh(a.msgCh)

	a.enqueueMessage("normal message", "U123", "C456", "ts.002", "")

	select {
	case msg := <-a.msgCh:
		if msg.ChatID != "C456" {
			t.Errorf("ChatID = %q, want 'C456'", msg.ChatID)
		}
	default:
		t.Error("message should be enqueued when running")
	}
}

// drainCh empties a buffered channel.
func drainCh(ch <-chan bot.InboundMessage) {
	for {
		select {
		case <-ch:
		default:
			return
		}
	}
}
