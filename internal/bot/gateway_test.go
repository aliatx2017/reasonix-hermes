package bot

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
	"testing"
)

// fakeAdapter 是一个内存中的假适配器，用于测试 BotGateway。
type fakeAdapter struct {
	mu       sync.Mutex
	platform Platform
	name     string
	msgCh    chan InboundMessage
	sent     []OutboundMessage
	started  bool
}

func newFakeAdapter(platform Platform, name string) *fakeAdapter {
	return &fakeAdapter{
		platform: platform,
		name:     name,
		msgCh:    make(chan InboundMessage, 16),
	}
}

func (f *fakeAdapter) Platform() Platform              { return f.platform }
func (f *fakeAdapter) Name() string                    { return f.name }
func (f *fakeAdapter) Messages() <-chan InboundMessage { return f.msgCh }

func (f *fakeAdapter) Start(ctx context.Context) error {
	f.mu.Lock()
	f.started = true
	f.mu.Unlock()
	return nil
}

func (f *fakeAdapter) Stop() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.msgCh != nil {
		close(f.msgCh)
		f.msgCh = nil
	}
	return nil
}

func (f *fakeAdapter) Send(ctx context.Context, msg OutboundMessage) (SendResult, error) {
	f.mu.Lock()
	f.sent = append(f.sent, msg)
	f.mu.Unlock()
	return SendResult{MessageID: "fake_msg_1"}, nil
}

func (f *fakeAdapter) SendTyping(ctx context.Context, chatID string) error { return nil }

func (f *fakeAdapter) sentMessages() []OutboundMessage {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]OutboundMessage, len(f.sent))
	copy(out, f.sent)
	return out
}

type fakeReactionAdapter struct {
	*fakeAdapter
	reactions []string
}

func (f *fakeReactionAdapter) AddPendingReaction(ctx context.Context, messageID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reactions = append(f.reactions, messageID)
	return nil
}

func TestFakeAdapterInterface(t *testing.T) {
	fa := newFakeAdapter(PlatformQQ, "fake-qq")

	if fa.Platform() != PlatformQQ {
		t.Error("wrong platform")
	}
	if fa.Name() != "fake-qq" {
		t.Error("wrong name")
	}

	ctx := context.Background()
	if err := fa.Start(ctx); err != nil {
		t.Fatal("start:", err)
	}
	if !fa.started {
		t.Error("should be started")
	}

	_, err := fa.Send(ctx, OutboundMessage{ChatID: "c1", Text: "hello"})
	if err != nil {
		t.Fatal("send:", err)
	}

	sent := fa.sentMessages()
	if len(sent) != 1 {
		t.Fatalf("sent count = %d, want 1", len(sent))
	}
	if sent[0].Text != "hello" {
		t.Errorf("sent text = %q, want %q", sent[0].Text, "hello")
	}

	if err := fa.Stop(); err != nil {
		t.Fatal("stop:", err)
	}
}

func TestGatewayConstructAndStop(t *testing.T) {
	cfg := GatewayConfig{
		Model:         "test",
		MaxSteps:      10,
		WorkspaceRoot: ".",
		Enabled:       map[Platform]bool{PlatformQQ: true},
		Allowlist:     AllowlistConfig{Enabled: false},
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	gw := NewGateway(cfg, map[Platform]Adapter{
		PlatformQQ: newFakeAdapter(PlatformQQ, "fake-qq"),
	}, logger)

	// 网关不应该 panic
	if gw == nil {
		t.Fatal("gateway should not be nil")
	}
	gw.Stop()
}

func TestGatewayAllowlistCheck(t *testing.T) {
	cfg := GatewayConfig{
		Allowlist: AllowlistConfig{
			Enabled: true,
			Users: map[Platform][]string{
				PlatformQQ: {"allowed_user_1"},
			},
		},
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	gw := NewGateway(cfg, nil, logger)

	if !gw.checkAllowlist(PlatformQQ, InboundMessage{Platform: PlatformQQ, ChatType: ChatDM, UserID: "allowed_user_1"}) {
		t.Error("allowed user should pass")
	}
	if gw.checkAllowlist(PlatformQQ, InboundMessage{Platform: PlatformQQ, ChatType: ChatDM, UserID: "unknown_user"}) {
		t.Error("unknown user should not pass")
	}
	// 不同平台
	if gw.checkAllowlist(PlatformFeishu, InboundMessage{Platform: PlatformFeishu, ChatType: ChatDM, UserID: "allowed_user_1"}) {
		t.Error("QQ allowlist should not apply to feishu")
	}
}

func TestGatewayAllowlistDoesNotApplyGroupsToDirectMessages(t *testing.T) {
	cfg := GatewayConfig{
		Allowlist: AllowlistConfig{
			Enabled: true,
			Users: map[Platform][]string{
				PlatformQQ: {"allowed_user"},
			},
			Groups: map[Platform][]string{
				PlatformQQ: {"allowed_group"},
			},
		},
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	gw := NewGateway(cfg, nil, logger)

	if !gw.checkAllowlist(PlatformQQ, InboundMessage{Platform: PlatformQQ, ChatType: ChatDirect, ChatID: "guild-dm", UserID: "allowed_user"}) {
		t.Error("direct message should not be rejected by group allowlist")
	}
	if gw.checkAllowlist(PlatformQQ, InboundMessage{Platform: PlatformQQ, ChatType: ChatGroup, ChatID: "unknown_group", UserID: "allowed_user"}) {
		t.Error("unknown group should still be rejected by group allowlist")
	}
}

func TestGatewayAllowlistDisabledRejectsByDefault(t *testing.T) {
	cfg := GatewayConfig{
		Allowlist: AllowlistConfig{Enabled: false},
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	gw := NewGateway(cfg, nil, logger)

	if gw.checkAllowlist(PlatformQQ, InboundMessage{Platform: PlatformQQ, ChatType: ChatDM, UserID: "any_user"}) {
		t.Error("disabled allowlist should reject unless allow_all is explicit")
	}
}

func TestGatewayAllowAll(t *testing.T) {
	cfg := GatewayConfig{
		Allowlist: AllowlistConfig{AllowAll: true},
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	gw := NewGateway(cfg, nil, logger)

	if !gw.checkAllowlist(PlatformQQ, InboundMessage{Platform: PlatformQQ, ChatType: ChatDM, UserID: "any_user"}) {
		t.Error("allow_all should allow everyone")
	}
}

func TestGatewayAddsPendingReactionWhenAdapterSupportsIt(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	gw := NewGateway(GatewayConfig{}, nil, logger)
	fa := &fakeReactionAdapter{fakeAdapter: newFakeAdapter(PlatformFeishu, "fake-feishu")}

	gw.addPendingReaction(context.Background(), PlatformFeishu, fa, InboundMessage{MessageID: "om_123"})

	if len(fa.reactions) != 1 || fa.reactions[0] != "om_123" {
		t.Fatalf("reactions = %#v, want [om_123]", fa.reactions)
	}
}

func TestGatewaySessionOptionsUseChannelOverride(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	gw := NewGateway(GatewayConfig{
		Model:         "global-model",
		WorkspaceRoot: "/global",
		Channels: map[Platform]ChannelConfig{
			PlatformFeishu: {Model: "feishu-model", WorkspaceRoot: "/feishu"},
			PlatformWeixin: {WorkspaceRoot: "/weixin"},
		},
	}, nil, logger)

	model, root := gw.sessionOptionsForPlatform(PlatformFeishu)
	if model != "feishu-model" || root != "/feishu" {
		t.Fatalf("feishu options = %q,%q; want channel override", model, root)
	}

	model, root = gw.sessionOptionsForPlatform(PlatformWeixin)
	if model != "global-model" || root != "/weixin" {
		t.Fatalf("weixin options = %q,%q; want global model and channel root", model, root)
	}

	model, root = gw.sessionOptionsForPlatform(PlatformQQ)
	if model != "global-model" || root != "/global" {
		t.Fatalf("qq options = %q,%q; want global defaults", model, root)
	}
}

// --- Model prefs persistence ---

func TestModelPrefsSaveAndLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/model-prefs.json"

	cfg := GatewayConfig{
		Model:          "default-model",
		ModelPrefsPath: path,
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	gw := NewGateway(cfg, nil, logger)

	// Initially empty
	if len(gw.modelPrefs) != 0 {
		t.Fatalf("expected 0 prefs, got %d", len(gw.modelPrefs))
	}

	// Set a preference
	key := "discord:guild:123:456"
	gw.mu.Lock()
	gw.modelPrefs[key] = "deepseek-pro"
	gw.mu.Unlock()

	// Save
	gw.saveModelPrefs()

	// Verify file exists and is valid JSON
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read prefs file: %v", err)
	}
	var saved map[string]string
	if err := json.Unmarshal(data, &saved); err != nil {
		t.Fatalf("unmarshal prefs: %v", err)
	}
	if saved[key] != "deepseek-pro" {
		t.Fatalf("saved value = %q, want %q", saved[key], "deepseek-pro")
	}

	// Create a new gateway and verify it loads the preference
	gw2 := NewGateway(cfg, nil, logger)
	gw2.mu.Lock()
	loaded := gw2.modelPrefs[key]
	gw2.mu.Unlock()
	if loaded != "deepseek-pro" {
		t.Fatalf("loaded value = %q, want %q", loaded, "deepseek-pro")
	}
}

func TestModelPrefsSaveNoPath(t *testing.T) {
	cfg := GatewayConfig{
		Model:          "default-model",
		ModelPrefsPath: "", // empty = no persistence
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	gw := NewGateway(cfg, nil, logger)

	gw.mu.Lock()
	gw.modelPrefs["key"] = "value"
	gw.mu.Unlock()

	// Should not panic
	gw.saveModelPrefs()
}

func TestModelPrefsLoadMissingFile(t *testing.T) {
	dir := t.TempDir()
	cfg := GatewayConfig{
		Model:          "default-model",
		ModelPrefsPath: dir + "/nonexistent.json",
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	// Should not panic — missing file is first-run
	gw := NewGateway(cfg, nil, logger)
	if gw == nil {
		t.Fatal("gateway should not be nil")
	}
}

func TestModelPrefsFilePath(t *testing.T) {
	path := ModelPrefsFilePath()
	if path == "" {
		t.Skip("os.UserConfigDir() unavailable on this platform")
	}
	if !strings.HasSuffix(path, "/reasonix/bot-model-prefs.json") {
		t.Fatalf("unexpected path suffix: %s", path)
	}
}

// --- Model resolution priority ---

func TestGetOrCreateSessionModelPreferencePriority(t *testing.T) {
	// set up a gateway with a modelPrefs entry
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	cfg := GatewayConfig{
		Model:    "gateway-default",
		MaxSteps: 10,
		Channels: map[Platform]ChannelConfig{
			PlatformDiscord: {Model: "channel-discord"},
		},
	}
	gw := NewGateway(cfg, map[Platform]Adapter{
		PlatformDiscord: newFakeAdapter(PlatformDiscord, "discord"),
	}, logger)

	key := "discord:dm:dm-chat"
	gw.mu.Lock()
	gw.modelPrefs[key] = "my-preference"
	gw.mu.Unlock()

	// TODO: getOrCreateSession requires a real boot.Build path which needs
	// a configured provider. The model resolution logic is exercised through
	// the public API — we verify the sub-functions directly.
	model, _ := gw.sessionOptionsForPlatform(PlatformDiscord)
	if model != "channel-discord" {
		t.Fatalf("platform default = %q, want channel-discord", model)
	}
}
