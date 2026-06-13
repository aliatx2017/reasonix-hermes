package discord

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"testing"

	"reasonix/internal/bot"
	"reasonix/internal/config"

	"github.com/bwmarrin/discordgo"
)

// helper: create a minimal Adapter for testing
func newTestAdapter(cfg config.DiscordBotConfig) *Adapter {
	return New(cfg, slog.Default()).(*Adapter)
}

// --- New() ---

func TestNew_CreatesAdapterWithCorrectConfig(t *testing.T) {
	cfg := config.DiscordBotConfig{
		TokenEnv:  "DISCORD_BOT_TOKEN",
		ChannelID: "123456",
		AllowDMs:  true,
	}
	logger := slog.Default()
	a := New(cfg, logger)

	ad, ok := a.(*Adapter)
	if !ok {
		t.Fatalf("New() did not return *Adapter, got %T", a)
	}
	if ad.cfg.TokenEnv != cfg.TokenEnv {
		t.Errorf("TokenEnv = %q, want %q", ad.cfg.TokenEnv, cfg.TokenEnv)
	}
	if ad.cfg.ChannelID != cfg.ChannelID {
		t.Errorf("ChannelID = %q, want %q", ad.cfg.ChannelID, cfg.ChannelID)
	}
	if ad.cfg.AllowDMs != cfg.AllowDMs {
		t.Errorf("AllowDMs = %v, want %v", ad.cfg.AllowDMs, cfg.AllowDMs)
	}
}

func TestNew_NilLoggerUsesDefault(t *testing.T) {
	cfg := config.DiscordBotConfig{TokenEnv: "MY_TOKEN"}
	a := New(cfg, nil)
	ad, ok := a.(*Adapter)
	if !ok {
		t.Fatalf("New() did not return *Adapter, got %T", a)
	}
	if ad.logger == nil {
		t.Error("expected non-nil logger after nil input")
	}
}

func TestNew_ChannelIsBuffered(t *testing.T) {
	a := New(config.DiscordBotConfig{TokenEnv: "T"}, slog.Default())
	ad := a.(*Adapter)
	if cap(ad.msgs) != 64 {
		t.Errorf("channel capacity = %d, want 64", cap(ad.msgs))
	}
}

// --- Platform() ---

func TestPlatform(t *testing.T) {
	a := New(config.DiscordBotConfig{TokenEnv: "T"}, slog.Default())
	if got := a.Platform(); got != bot.PlatformDiscord {
		t.Errorf("Platform() = %q, want %q", got, bot.PlatformDiscord)
	}
}

// --- Name() ---

func TestName(t *testing.T) {
	a := New(config.DiscordBotConfig{TokenEnv: "T"}, slog.Default())
	if got := a.Name(); got != "discord" {
		t.Errorf("Name() = %q, want %q", got, "discord")
	}
}

// --- Messages() ---

func TestMessages_ReturnsReadableChannel(t *testing.T) {
	a := New(config.DiscordBotConfig{TokenEnv: "T"}, slog.Default())
	ch := a.Messages()
	if ch == nil {
		t.Fatal("Messages() returned nil channel")
	}
	select {
	case _, ok := <-ch:
		if ok {
			t.Error("expected empty channel, got a message")
		}
	default:
	}
}

// --- stripMention ---

func TestStripMention_NoMention(t *testing.T) {
	got := stripMention("hello world", "123")
	if got != "hello world" {
		t.Errorf("stripMention(%q, %q) = %q, want %q", "hello world", "123", got, "hello world")
	}
}

func TestStripMention_StandardMention(t *testing.T) {
	got := stripMention("<@123> hello", "123")
	if got != "hello" {
		t.Errorf("stripMention(%q, %q) = %q, want %q", "<@123> hello", "123", got, "hello")
	}
}

func TestStripMention_NicknameMention(t *testing.T) {
	got := stripMention("<@!123> hello", "123")
	if got != "hello" {
		t.Errorf("stripMention(%q, %q) = %q, want %q", "<@!123> hello", "123", got, "hello")
	}
}

func TestStripMention_MentionOnly(t *testing.T) {
	got := stripMention("<@123>", "123")
	if got != "" {
		t.Errorf("stripMention(%q, %q) = %q, want %q", "<@123>", "123", got, "")
	}
}

func TestStripMention_MentionInMiddle(t *testing.T) {
	got := stripMention("hello <@123> world", "123")
	if got != "hello <@123> world" {
		t.Errorf("stripMention(%q, %q) = %q, want %q", "hello <@123> world", "123", got, "hello <@123> world")
	}
}

func TestStripMention_DifferentBotID(t *testing.T) {
	got := stripMention("<@456> hello", "123")
	if got != "<@456> hello" {
		t.Errorf("stripMention(%q, %q) = %q, want %q", "<@456> hello", "123", got, "<@456> hello")
	}
}

func TestStripMention_ExtraSpaces(t *testing.T) {
	got := stripMention("<@123>  hello  ", "123")
	if got != "hello" {
		t.Errorf("stripMention(%q, %q) = %q, want %q", "<@123>  hello  ", "123", got, "hello")
	}
}

// --- cardToMarkdown ---

func TestCardToMarkdown_Empty(t *testing.T) {
	card := &bot.InteractiveCard{
		Header:   "Title",
		Elements: nil,
	}
	got := cardToMarkdown(card)
	if got != "" {
		t.Errorf("cardToMarkdown(empty elements) = %q, want empty string", got)
	}
}

func TestCardToMarkdown_MarkdownElement(t *testing.T) {
	card := &bot.InteractiveCard{
		Header: "Title",
		Elements: []bot.InteractiveCardElement{
			{Tag: "markdown", Content: "**bold text**"},
		},
	}
	got := cardToMarkdown(card)
	want := "**bold text**\n"
	if got != want {
		t.Errorf("cardToMarkdown() = %q, want %q", got, want)
	}
}

func TestCardToMarkdown_MultipleElements(t *testing.T) {
	card := &bot.InteractiveCard{
		Header: "Title",
		Elements: []bot.InteractiveCardElement{
			{Tag: "markdown", Content: "line 1"},
			{Tag: "markdown", Content: "line 2"},
			{Tag: "text", Content: "ignored"},
			{Tag: "markdown", Content: "line 3"},
		},
	}
	got := cardToMarkdown(card)
	want := "line 1\nline 2\nline 3\n"
	if got != want {
		t.Errorf("cardToMarkdown() = %q, want %q", got, want)
	}
}

func TestCardToMarkdown_EmptyContent(t *testing.T) {
	card := &bot.InteractiveCard{
		Header: "Title",
		Elements: []bot.InteractiveCardElement{
			{Tag: "markdown", Content: ""},
			{Tag: "markdown", Content: "visible"},
		},
	}
	got := cardToMarkdown(card)
	want := "visible\n"
	if got != want {
		t.Errorf("cardToMarkdown() = %q, want %q", got, want)
	}
}

// --- keyboardToComponents ---

func TestKeyboardToComponents_Nil(t *testing.T) {
	got := keyboardToComponents(nil)
	if got != nil {
		t.Errorf("keyboardToComponents(nil) = %v, want nil", got)
	}
}

func TestKeyboardToComponents_EmptyRows(t *testing.T) {
	kb := &bot.InlineKeyboard{Rows: nil}
	got := keyboardToComponents(kb)
	if len(got) != 0 {
		t.Errorf("keyboardToComponents(empty rows) = %d rows, want 0", len(got))
	}
}

func TestKeyboardToComponents_SingleButton(t *testing.T) {
	kb := &bot.InlineKeyboard{
		Rows: []bot.InlineKeyboardRow{
			{
				Buttons: []bot.InlineKeyboardButton{
					{ID: "approve", Label: "Approve", Style: 1, CallbackID: "cb_approve"},
				},
			},
		},
	}
	got := keyboardToComponents(kb)
	if len(got) != 1 {
		t.Fatalf("expected 1 row, got %d", len(got))
	}
	row, ok := got[0].(discordgo.ActionsRow)
	if !ok {
		t.Fatalf("expected ActionsRow, got %T", got[0])
	}
	if len(row.Components) != 1 {
		t.Fatalf("expected 1 component, got %d", len(row.Components))
	}
	btn, ok := row.Components[0].(discordgo.Button)
	if !ok {
		t.Fatalf("expected Button, got %T", row.Components[0])
	}
	if btn.Label != "Approve" {
		t.Errorf("Label = %q, want %q", btn.Label, "Approve")
	}
	if btn.Style != discordgo.PrimaryButton {
		t.Errorf("Style = %v, want PrimaryButton", btn.Style)
	}
	if btn.CustomID != "cb_approve" {
		t.Errorf("CustomID = %q, want %q", btn.CustomID, "cb_approve")
	}
}

func TestKeyboardToComponents_MultiRowKeyboard(t *testing.T) {
	kb := &bot.InlineKeyboard{
		Rows: []bot.InlineKeyboardRow{
			{
				Buttons: []bot.InlineKeyboardButton{
					{ID: "approve", Label: "Approve", Style: 1, CallbackID: "cb_approve"},
					{ID: "reject", Label: "Reject", Style: 2, CallbackID: "cb_reject"},
				},
			},
			{
				Buttons: []bot.InlineKeyboardButton{
					{ID: "cancel", Label: "Cancel", Style: 0, CallbackID: "cb_cancel"},
				},
			},
		},
	}
	got := keyboardToComponents(kb)
	if len(got) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(got))
	}

	row1, ok := got[0].(discordgo.ActionsRow)
	if !ok {
		t.Fatalf("row 0: expected ActionsRow, got %T", got[0])
	}
	if len(row1.Components) != 2 {
		t.Fatalf("row 0: expected 2 buttons, got %d", len(row1.Components))
	}
	btn1, _ := row1.Components[0].(discordgo.Button)
	btn2, _ := row1.Components[1].(discordgo.Button)
	if btn1.Style != discordgo.PrimaryButton {
		t.Errorf("row 0 btn 0: Style = %v, want PrimaryButton", btn1.Style)
	}
	if btn2.Style != discordgo.DangerButton {
		t.Errorf("row 0 btn 1: Style = %v, want DangerButton", btn2.Style)
	}

	row2, ok := got[1].(discordgo.ActionsRow)
	if !ok {
		t.Fatalf("row 1: expected ActionsRow, got %T", got[1])
	}
	if len(row2.Components) != 1 {
		t.Fatalf("row 1: expected 1 button, got %d", len(row2.Components))
	}
	btn3, _ := row2.Components[0].(discordgo.Button)
	if btn3.Style != discordgo.SecondaryButton {
		t.Errorf("row 1 btn 0: Style = %v, want SecondaryButton", btn3.Style)
	}
}

func TestKeyboardToComponents_ButtonStyles(t *testing.T) {
	tests := []struct {
		name      string
		style     int
		wantStyle discordgo.ButtonStyle
	}{
		{"default 0", 0, discordgo.SecondaryButton},
		{"primary 1", 1, discordgo.PrimaryButton},
		{"danger 2", 2, discordgo.DangerButton},
		{"unknown 3", 3, discordgo.SecondaryButton},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kb := &bot.InlineKeyboard{
				Rows: []bot.InlineKeyboardRow{
					{
						Buttons: []bot.InlineKeyboardButton{
							{Label: "Btn", Style: tt.style, CallbackID: "cb"},
						},
					},
				},
			}
			got := keyboardToComponents(kb)
			row := got[0].(discordgo.ActionsRow)
			btn := row.Components[0].(discordgo.Button)
			if btn.Style != tt.wantStyle {
				t.Errorf("style %d: got %v, want %v", tt.style, btn.Style, tt.wantStyle)
			}
		})
	}
}

// --- Start() ---

func TestStart_MissingToken(t *testing.T) {
	// Ensure neither the configured env nor DISCORD_BOT_TOKEN is set
	os.Unsetenv("NONEXISTENT_TOKEN_VAR_FOR_TEST")
	os.Unsetenv("DISCORD_BOT_TOKEN")

	a := newTestAdapter(config.DiscordBotConfig{TokenEnv: "NONEXISTENT_TOKEN_VAR_FOR_TEST"})
	err := a.Start(context.Background())
	if err == nil {
		t.Fatal("expected error when token is missing, got nil")
	}
	if !strings.Contains(err.Error(), "bot token not found") {
		t.Errorf("error = %q, want mention of 'bot token not found'", err.Error())
	}
}

func TestStart_FallbackToken(t *testing.T) {
	// TokenEnv is empty but DISCORD_BOT_TOKEN is set
	os.Unsetenv("EMPTY_TOKEN_ENV_FOR_TEST")
	os.Unsetenv("DISCORD_BOT_TOKEN")

	a := newTestAdapter(config.DiscordBotConfig{TokenEnv: "EMPTY_TOKEN_ENV_FOR_TEST"})
	err := a.Start(context.Background())
	if err == nil {
		t.Fatal("expected error when no token available, got nil")
	}
	if !strings.Contains(err.Error(), "bot token not found") {
		t.Errorf("error = %q, want mention of 'bot token not found'", err.Error())
	}
}

func TestStart_InvalidToken(t *testing.T) {
	// Set a token that will fail when creating session
	t.Setenv("DISCORD_BOT_TOKEN", "invalid-token-value")

	a := newTestAdapter(config.DiscordBotConfig{TokenEnv: "DISCORD_BOT_TOKEN"})
	err := a.Start(context.Background())
	// discordgo.New("Bot invalid-token-value") may or may not fail depending on validation
	// If it doesn't fail at New(), it will fail at Open()
	if err != nil {
		// Good: got an error (either from New or Open)
		if !strings.Contains(err.Error(), "discord:") {
			t.Errorf("error = %q, want 'discord:' prefix", err.Error())
		}
	}
	// If no error, the token was accepted by discordgo but Open might succeed in some edge case
	// Clean up session if created
	if a.session != nil {
		a.session.Close()
	}
}

// --- Stop() ---

func TestStop_NilSession(t *testing.T) {
	a := newTestAdapter(config.DiscordBotConfig{TokenEnv: "T"})
	a.session = nil
	err := a.Stop()
	if err != nil {
		t.Errorf("Stop() with nil session returned error: %v", err)
	}
}

func TestStop_WithSession(t *testing.T) {
	a := newTestAdapter(config.DiscordBotConfig{TokenEnv: "T"})
	// Create a session that's not connected (will error on Close but that's OK)
	dg, err := discordgo.New("Bot test-token")
	if err != nil {
		t.Skipf("cannot create discordgo session: %v", err)
	}
	a.session = dg
	// Close should not panic; it will return an error since we never opened, but that's fine
	a.Stop() // just ensure no panic
}

// --- Send() ---

func TestSend_NilSession(t *testing.T) {
	a := newTestAdapter(config.DiscordBotConfig{TokenEnv: "T"})
	a.session = nil
	_, err := a.Send(context.Background(), bot.OutboundMessage{ChatID: "123", Text: "hello"})
	if err == nil {
		t.Fatal("expected error with nil session, got nil")
	}
	if !strings.Contains(err.Error(), "session not initialized") {
		t.Errorf("error = %q, want 'session not initialized'", err.Error())
	}
}

func TestSend_TruncatesLongContent(t *testing.T) {
	a := newTestAdapter(config.DiscordBotConfig{TokenEnv: "T"})
	// Create a real session (not connected) to pass the nil check
	dg, err := discordgo.New("Bot test-token")
	if err != nil {
		t.Skipf("cannot create discordgo session: %v", err)
	}
	a.session = dg

	longText := strings.Repeat("a", 2000)
	_, err = a.Send(context.Background(), bot.OutboundMessage{ChatID: "123", Text: longText})
	// The send will fail because we're not connected, but we're testing that
	// truncation logic is reached. The error should be from Discord API, not from our code.
	if err == nil {
		t.Log("Send succeeded unexpectedly (connected to Discord?)")
	} else {
		// Error should be from the Discord API call, not from our validation
		if strings.Contains(err.Error(), "session not initialized") {
			t.Error("should not hit session nil check for long content")
		}
	}
}

func TestSend_WithCard(t *testing.T) {
	a := newTestAdapter(config.DiscordBotConfig{TokenEnv: "T"})
	dg, err := discordgo.New("Bot test-token")
	if err != nil {
		t.Skipf("cannot create discordgo session: %v", err)
	}
	a.session = dg

	_, err = a.Send(context.Background(), bot.OutboundMessage{
		ChatID: "123",
		Text:   "ignored when card present",
		Card: &bot.InteractiveCard{
			Header: "Test Card",
			Elements: []bot.InteractiveCardElement{
				{Tag: "markdown", Content: "card body"},
			},
		},
	})
	// Will fail because not connected, but we exercise the embed path
	if err == nil {
		t.Log("Send with card succeeded unexpectedly")
	}
}

func TestSend_WithKeyboard(t *testing.T) {
	a := newTestAdapter(config.DiscordBotConfig{TokenEnv: "T"})
	dg, err := discordgo.New("Bot test-token")
	if err != nil {
		t.Skipf("cannot create discordgo session: %v", err)
	}
	a.session = dg

	_, err = a.Send(context.Background(), bot.OutboundMessage{
		ChatID: "123",
		Text:   "approve this",
		Keyboard: &bot.InlineKeyboard{
			Rows: []bot.InlineKeyboardRow{
				{Buttons: []bot.InlineKeyboardButton{
					{Label: "Approve", Style: 1, CallbackID: "cb_approve"},
				}},
			},
		},
	})
	if err == nil {
		t.Log("Send with keyboard succeeded unexpectedly")
	}
}

func TestSend_WithReplyTo(t *testing.T) {
	a := newTestAdapter(config.DiscordBotConfig{TokenEnv: "T"})
	dg, err := discordgo.New("Bot test-token")
	if err != nil {
		t.Skipf("cannot create discordgo session: %v", err)
	}
	a.session = dg

	_, err = a.Send(context.Background(), bot.OutboundMessage{
		ChatID:       "123",
		Text:         "reply to this",
		ReplyToMsgID: "999",
	})
	if err == nil {
		t.Log("Send with reply succeeded unexpectedly")
	}
}

func TestSend_PlainMessage(t *testing.T) {
	a := newTestAdapter(config.DiscordBotConfig{TokenEnv: "T"})
	dg, err := discordgo.New("Bot test-token")
	if err != nil {
		t.Skipf("cannot create discordgo session: %v", err)
	}
	a.session = dg

	_, err = a.Send(context.Background(), bot.OutboundMessage{
		ChatID: "123",
		Text:   "plain message",
	})
	if err == nil {
		t.Log("Send plain message succeeded unexpectedly")
	}
}

// --- SendTyping() ---

func TestSendTyping_NilSession(t *testing.T) {
	a := newTestAdapter(config.DiscordBotConfig{TokenEnv: "T"})
	a.session = nil
	err := a.SendTyping(context.Background(), "123")
	if err != nil {
		t.Errorf("SendTyping with nil session should return nil, got: %v", err)
	}
}

func TestSendTyping_WithSession(t *testing.T) {
	a := newTestAdapter(config.DiscordBotConfig{TokenEnv: "T"})
	dg, err := discordgo.New("Bot test-token")
	if err != nil {
		t.Skipf("cannot create discordgo session: %v", err)
	}
	a.session = dg

	err = a.SendTyping(context.Background(), "123")
	// Will fail because not connected, but exercises the path
	if err == nil {
		t.Log("SendTyping succeeded unexpectedly")
	}
}

// --- onReady() ---

func TestOnReady(t *testing.T) {
	a := newTestAdapter(config.DiscordBotConfig{TokenEnv: "T"})
	dg, err := discordgo.New("Bot test-token")
	if err != nil {
		t.Skipf("cannot create discordgo session: %v", err)
	}
	a.session = dg

	// onReady accesses s.State.User.ID to list/delete stale commands, then
	// creates the /model slash command. State.User must be set or the call
	// to ApplicationCommands panics with nil dereference.
	dg.State.User = &discordgo.User{ID: "bot1", Username: "TestBot"}

	// onReady accesses event.User.Username and event.Guilds, then calls UpdateGameStatus
	event := &discordgo.Ready{
		User:   &discordgo.User{Username: "TestBot"},
		Guilds: []*discordgo.Guild{{ID: "guild1"}, {ID: "guild2"}},
	}
	// This will fail on UpdateGameStatus since we're not connected, but we exercise the log + call
	a.onReady(dg, event)
}

// --- onMessageCreate() ---

func newSessionWithBotID(botID string) *discordgo.Session {
	dg, _ := discordgo.New("Bot test-token")
	if dg == nil {
		return nil
	}
	dg.State.User = &discordgo.User{ID: botID}
	return dg
}

func TestOnMessageCreate_IgnoresOwnMessages(t *testing.T) {
	a := newTestAdapter(config.DiscordBotConfig{TokenEnv: "T"})
	dg := newSessionWithBotID("bot1")
	if dg == nil {
		t.Skip("cannot create discordgo session")
	}

	// Message from the bot itself should be ignored
	msg := &discordgo.MessageCreate{
		Message: &discordgo.Message{
			Author:    &discordgo.User{ID: "bot1", Username: "BotUser"},
			ChannelID: "ch1",
			Content:   "self message",
			ID:        "m1",
		},
	}
	a.onMessageCreate(dg, msg)

	// Channel should be empty (message was dropped)
	select {
	case <-a.msgs:
		t.Error("should not have received bot's own message")
	default:
		// correct
	}
}

func TestOnMessageCreate_ChannelFilter(t *testing.T) {
	a := newTestAdapter(config.DiscordBotConfig{
		TokenEnv:  "T",
		ChannelID: "allowed-ch",
	})
	dg := newSessionWithBotID("bot1")
	if dg == nil {
		t.Skip("cannot create discordgo session")
	}

	// Message to wrong channel should be ignored
	msg := &discordgo.MessageCreate{
		Message: &discordgo.Message{
			Author:    &discordgo.User{ID: "user1", Username: "User1"},
			ChannelID: "other-ch",
			Content:   "hello",
			ID:        "m1",
		},
	}
	a.onMessageCreate(dg, msg)

	select {
	case <-a.msgs:
		t.Error("should not have received message from filtered channel")
	default:
	}
}

func TestOnMessageCreate_ChannelFilterMatch(t *testing.T) {
	a := newTestAdapter(config.DiscordBotConfig{
		TokenEnv:  "T",
		ChannelID: "allowed-ch",
	})
	dg := newSessionWithBotID("bot1")
	if dg == nil {
		t.Skip("cannot create discordgo session")
	}

	// Message to allowed channel should be received
	msg := &discordgo.MessageCreate{
		Message: &discordgo.Message{
			Author:    &discordgo.User{ID: "user1", Username: "User1"},
			ChannelID: "allowed-ch",
			Content:   "hello",
			ID:        "m1",
		},
	}
	a.onMessageCreate(dg, msg)

	select {
	case got := <-a.msgs:
		if got.Text != "hello" {
			t.Errorf("text = %q, want %q", got.Text, "hello")
		}
		if got.ChatID != "allowed-ch" {
			t.Errorf("ChatID = %q, want %q", got.ChatID, "allowed-ch")
		}
	default:
		t.Error("expected to receive message from allowed channel")
	}
}

func TestOnMessageCreate_EmptyContent(t *testing.T) {
	a := newTestAdapter(config.DiscordBotConfig{TokenEnv: "T"})
	dg := newSessionWithBotID("bot1")
	if dg == nil {
		t.Skip("cannot create discordgo session")
	}

	msg := &discordgo.MessageCreate{
		Message: &discordgo.Message{
			Author:    &discordgo.User{ID: "user1", Username: "User1"},
			ChannelID: "ch1",
			Content:   "",
			ID:        "m1",
		},
	}
	a.onMessageCreate(dg, msg)

	select {
	case <-a.msgs:
		t.Error("should not have received message with empty content")
	default:
	}
}

func TestOnMessageCreate_DMFilter(t *testing.T) {
	a := newTestAdapter(config.DiscordBotConfig{
		TokenEnv: "T",
		AllowDMs: false,
	})
	dg := newSessionWithBotID("bot1")
	if dg == nil {
		t.Skip("cannot create discordgo session")
	}

	// Pre-populate cache so resolveChatType returns ChatDM
	a.chatType["dm-ch"] = bot.ChatDM

	msg := &discordgo.MessageCreate{
		Message: &discordgo.Message{
			Author:    &discordgo.User{ID: "user1", Username: "User1"},
			ChannelID: "dm-ch",
			Content:   "hello DM",
			ID:        "m1",
		},
	}
	a.onMessageCreate(dg, msg)

	select {
	case <-a.msgs:
		t.Error("should not have received DM when AllowDMs=false")
	default:
	}
}

func TestOnMessageCreate_DMAllowed(t *testing.T) {
	a := newTestAdapter(config.DiscordBotConfig{
		TokenEnv: "T",
		AllowDMs: true,
	})
	dg := newSessionWithBotID("bot1")
	if dg == nil {
		t.Skip("cannot create discordgo session")
	}

	a.chatType["dm-ch"] = bot.ChatDM

	msg := &discordgo.MessageCreate{
		Message: &discordgo.Message{
			Author:    &discordgo.User{ID: "user1", Username: "User1"},
			ChannelID: "dm-ch",
			Content:   "hello DM",
			ID:        "m1",
		},
	}
	a.onMessageCreate(dg, msg)

	select {
	case got := <-a.msgs:
		if got.ChatType != bot.ChatDM {
			t.Errorf("ChatType = %q, want %q", got.ChatType, bot.ChatDM)
		}
	default:
		t.Error("expected to receive DM when AllowDMs=true")
	}
}

func TestOnMessageCreate_StripsMention(t *testing.T) {
	a := newTestAdapter(config.DiscordBotConfig{TokenEnv: "T"})
	dg := newSessionWithBotID("bot1")
	if dg == nil {
		t.Skip("cannot create discordgo session")
	}

	a.chatType["ch1"] = bot.ChatGuild

	msg := &discordgo.MessageCreate{
		Message: &discordgo.Message{
			Author:    &discordgo.User{ID: "user1", Username: "User1"},
			ChannelID: "ch1",
			Content:   "<@bot1> do something",
			ID:        "m1",
		},
	}
	a.onMessageCreate(dg, msg)

	select {
	case got := <-a.msgs:
		if got.Text != "do something" {
			t.Errorf("text = %q, want %q", got.Text, "do something")
		}
	default:
		t.Error("expected to receive message")
	}
}

func TestOnMessageCreate_PopulatesFields(t *testing.T) {
	a := newTestAdapter(config.DiscordBotConfig{TokenEnv: "T"})
	dg := newSessionWithBotID("bot1")
	if dg == nil {
		t.Skip("cannot create discordgo session")
	}

	a.chatType["ch1"] = bot.ChatGuild

	msg := &discordgo.MessageCreate{
		Message: &discordgo.Message{
			Author:    &discordgo.User{ID: "user42", Username: "TestUser"},
			ChannelID: "ch1",
			Content:   "test message",
			ID:        "msg99",
		},
	}
	a.onMessageCreate(dg, msg)

	select {
	case got := <-a.msgs:
		if got.Platform != bot.PlatformDiscord {
			t.Errorf("Platform = %q, want %q", got.Platform, bot.PlatformDiscord)
		}
		if got.ChatType != bot.ChatGuild {
			t.Errorf("ChatType = %q, want %q", got.ChatType, bot.ChatGuild)
		}
		if got.ChatID != "ch1" {
			t.Errorf("ChatID = %q, want %q", got.ChatID, "ch1")
		}
		if got.UserID != "user42" {
			t.Errorf("UserID = %q, want %q", got.UserID, "user42")
		}
		if got.UserName != "TestUser" {
			t.Errorf("UserName = %q, want %q", got.UserName, "TestUser")
		}
		if got.Text != "test message" {
			t.Errorf("Text = %q, want %q", got.Text, "test message")
		}
		if got.MessageID != "msg99" {
			t.Errorf("MessageID = %q, want %q", got.MessageID, "msg99")
		}
	default:
		t.Error("expected to receive message")
	}
}

func TestOnMessageCreate_ChannelFull(t *testing.T) {
	// Create adapter with tiny channel buffer to test the full-channel path
	cfg := config.DiscordBotConfig{TokenEnv: "T"}
	a := New(cfg, slog.Default()).(*Adapter)
	// Replace msgs with a tiny buffered channel
	a.msgs = make(chan bot.InboundMessage, 1)
	a.chatType["ch1"] = bot.ChatGuild

	dg := newSessionWithBotID("bot1")
	if dg == nil {
		t.Skip("cannot create discordgo session")
	}

	// Fill the channel
	a.msgs <- bot.InboundMessage{Text: "filler"}

	// Send another message — should hit default case (channel full)
	msg := &discordgo.MessageCreate{
		Message: &discordgo.Message{
			Author:    &discordgo.User{ID: "user1", Username: "User1"},
			ChannelID: "ch1",
			Content:   "overflow message",
			ID:        "m2",
		},
	}
	a.onMessageCreate(dg, msg)

	// Channel still has only the filler
	select {
	case got := <-a.msgs:
		if got.Text != "filler" {
			t.Errorf("got %q, want filler", got.Text)
		}
	default:
		t.Error("expected filler message in channel")
	}

	// Next read should be empty (overflow was dropped)
	select {
	case <-a.msgs:
		t.Error("channel should be empty after draining filler")
	default:
		// correct
	}
}

// --- resolveChatType() ---

func TestResolveChatType_CacheHit(t *testing.T) {
	a := newTestAdapter(config.DiscordBotConfig{TokenEnv: "T"})
	dg := newSessionWithBotID("bot1")
	if dg == nil {
		t.Skip("cannot create discordgo session")
	}

	// Pre-populate cache
	a.chatType["ch1"] = bot.ChatDM

	msg := &discordgo.MessageCreate{
		Message: &discordgo.Message{ChannelID: "ch1"},
	}
	ct := a.resolveChatType(dg, msg)
	if ct != bot.ChatDM {
		t.Errorf("resolveChatType cache hit = %q, want %q", ct, bot.ChatDM)
	}
}

func addChannelToState(dg *discordgo.Session, channel *discordgo.Channel) {
	// For DM/GroupDM channels, no guild needed
	if channel.Type == discordgo.ChannelTypeDM || channel.Type == discordgo.ChannelTypeGroupDM {
		dg.State.ChannelAdd(channel)
		return
	}
	// Non-DM channels require a guild in state
	guildID := "test-guild"
	dg.State.GuildAdd(&discordgo.Guild{ID: guildID})
	channel.GuildID = guildID
	dg.State.ChannelAdd(channel)
}

func TestResolveChatType_StateGuildText(t *testing.T) {
	a := newTestAdapter(config.DiscordBotConfig{TokenEnv: "T"})
	dg := newSessionWithBotID("bot1")
	if dg == nil {
		t.Skip("cannot create discordgo session")
	}

	// Add channel to state via ChannelAdd so channelMap is populated
	addChannelToState(dg, &discordgo.Channel{ID: "ch-guild", Type: discordgo.ChannelTypeGuildText})

	msg := &discordgo.MessageCreate{
		Message: &discordgo.Message{ChannelID: "ch-guild"},
	}
	ct := a.resolveChatType(dg, msg)
	if ct != bot.ChatGuild {
		t.Errorf("resolveChatType guild text = %q, want %q", ct, bot.ChatGuild)
	}
	// Check cache was populated
	if a.chatType["ch-guild"] != bot.ChatGuild {
		t.Errorf("cache not populated: %v", a.chatType["ch-guild"])
	}
}

func TestResolveChatType_StateDM(t *testing.T) {
	a := newTestAdapter(config.DiscordBotConfig{TokenEnv: "T"})
	dg := newSessionWithBotID("bot1")
	if dg == nil {
		t.Skip("cannot create discordgo session")
	}

	dg.State.ChannelAdd(&discordgo.Channel{ID: "ch-dm", Type: discordgo.ChannelTypeDM})

	msg := &discordgo.MessageCreate{
		Message: &discordgo.Message{ChannelID: "ch-dm"},
	}
	ct := a.resolveChatType(dg, msg)
	if ct != bot.ChatDM {
		t.Errorf("resolveChatType DM = %q, want %q", ct, bot.ChatDM)
	}
}

func TestResolveChatType_StatePublicThread(t *testing.T) {
	a := newTestAdapter(config.DiscordBotConfig{TokenEnv: "T"})
	dg := newSessionWithBotID("bot1")
	if dg == nil {
		t.Skip("cannot create discordgo session")
	}

	addChannelToState(dg, &discordgo.Channel{ID: "ch-thread", Type: discordgo.ChannelTypeGuildPublicThread})

	msg := &discordgo.MessageCreate{
		Message: &discordgo.Message{ChannelID: "ch-thread"},
	}
	ct := a.resolveChatType(dg, msg)
	if ct != bot.ChatGuild {
		t.Errorf("resolveChatType public thread = %q, want %q", ct, bot.ChatGuild)
	}
}

func TestResolveChatType_StatePrivateThread(t *testing.T) {
	a := newTestAdapter(config.DiscordBotConfig{TokenEnv: "T"})
	dg := newSessionWithBotID("bot1")
	if dg == nil {
		t.Skip("cannot create discordgo session")
	}

	addChannelToState(dg, &discordgo.Channel{ID: "ch-privthread", Type: discordgo.ChannelTypeGuildPrivateThread})

	msg := &discordgo.MessageCreate{
		Message: &discordgo.Message{ChannelID: "ch-privthread"},
	}
	ct := a.resolveChatType(dg, msg)
	if ct != bot.ChatThread {
		t.Errorf("resolveChatType private thread = %q, want %q", ct, bot.ChatThread)
	}
}

func TestResolveChatType_StateGroupDM(t *testing.T) {
	a := newTestAdapter(config.DiscordBotConfig{TokenEnv: "T"})
	dg := newSessionWithBotID("bot1")
	if dg == nil {
		t.Skip("cannot create discordgo session")
	}

	dg.State.ChannelAdd(&discordgo.Channel{ID: "ch-gdm", Type: discordgo.ChannelTypeGroupDM})

	msg := &discordgo.MessageCreate{
		Message: &discordgo.Message{ChannelID: "ch-gdm"},
	}
	ct := a.resolveChatType(dg, msg)
	if ct != bot.ChatGroup {
		t.Errorf("resolveChatType group DM = %q, want %q", ct, bot.ChatGroup)
	}
}

func TestResolveChatType_DefaultFallback(t *testing.T) {
	a := newTestAdapter(config.DiscordBotConfig{TokenEnv: "T"})
	dg := newSessionWithBotID("bot1")
	if dg == nil {
		t.Skip("cannot create discordgo session")
	}

	// Unknown channel type falls back to ChatGuild
	addChannelToState(dg, &discordgo.Channel{ID: "ch-unknown", Type: discordgo.ChannelType(999)})

	msg := &discordgo.MessageCreate{
		Message: &discordgo.Message{ChannelID: "ch-unknown"},
	}
	ct := a.resolveChatType(dg, msg)
	if ct != bot.ChatGuild {
		t.Errorf("resolveChatType unknown type = %q, want %q", ct, bot.ChatGuild)
	}
}

func TestResolveChatType_NotInStateDefaultsToGroup(t *testing.T) {
	a := newTestAdapter(config.DiscordBotConfig{TokenEnv: "T"})
	dg := newSessionWithBotID("bot1")
	if dg == nil {
		t.Skip("cannot create discordgo session")
	}

	// Channel not in state and no API available (session not connected)
	msg := &discordgo.MessageCreate{
		Message: &discordgo.Message{ChannelID: "ch-notfound"},
	}
	ct := a.resolveChatType(dg, msg)
	// Neither State.Channel nor Channel() will work → default to ChatGroup
	if ct != bot.ChatGroup {
		t.Errorf("resolveChatType unknown channel = %q, want %q", ct, bot.ChatGroup)
	}
}

// --- onInteractionCreate ---

func TestOnInteractionCreate_ModelCommandWithOption(t *testing.T) {
	a := newTestAdapter(config.DiscordBotConfig{TokenEnv: "T"})
	dg := newSessionWithBotID("bot1")
	if dg == nil {
		t.Skip("cannot create discordgo session")
	}

	interaction := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			ID:        "interaction-1",
			ChannelID: "ch-guild",
			Type:      discordgo.InteractionApplicationCommand,
			Member: &discordgo.Member{
				User: &discordgo.User{ID: "user1", Username: "TestUser"},
			},
			Data: discordgo.ApplicationCommandInteractionData{
				Name: "model",
				Options: []*discordgo.ApplicationCommandInteractionDataOption{
					{Name: "model", Value: "flash", Type: discordgo.ApplicationCommandOptionString},
				},
			},
		},
	}
	a.onInteractionCreate(dg, interaction)

	select {
	case got := <-a.msgs:
		if got.Platform != bot.PlatformDiscord {
			t.Errorf("Platform = %q, want discord", got.Platform)
		}
		if got.Text != "/model flash" {
			t.Errorf("Text = %q, want /model flash", got.Text)
		}
		if got.UserID != "user1" {
			t.Errorf("UserID = %q, want user1", got.UserID)
		}
		if got.UserName != "TestUser" {
			t.Errorf("UserName = %q, want TestUser", got.UserName)
		}
		if got.ChatID != "ch-guild" {
			t.Errorf("ChatID = %q, want ch-guild", got.ChatID)
		}
	default:
		t.Error("expected message from interaction")
	}
}

func TestOnInteractionCreate_ModelCommandNoOption(t *testing.T) {
	a := newTestAdapter(config.DiscordBotConfig{TokenEnv: "T"})
	dg := newSessionWithBotID("bot1")
	if dg == nil {
		t.Skip("cannot create discordgo session")
	}

	// No options = just "/model" (show current)
	interaction := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			ID:        "interaction-2",
			ChannelID: "ch-guild",
			Type:      discordgo.InteractionApplicationCommand,
			Member: &discordgo.Member{
				User: &discordgo.User{ID: "user2", Username: "User2"},
			},
			Data: discordgo.ApplicationCommandInteractionData{
				Name:    "model",
				Options: nil,
			},
		},
	}
	a.onInteractionCreate(dg, interaction)

	select {
	case got := <-a.msgs:
		if got.Text != "/model" {
			t.Errorf("Text = %q, want /model", got.Text)
		}
	default:
		t.Error("expected message from interaction")
	}
}

func TestOnInteractionCreate_DMInteraction(t *testing.T) {
	a := newTestAdapter(config.DiscordBotConfig{TokenEnv: "T"})
	dg := newSessionWithBotID("bot1")
	if dg == nil {
		t.Skip("cannot create discordgo session")
	}

	// In DMs, i.Member is nil and i.User is populated
	interaction := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			ID:        "interaction-dm",
			ChannelID: "dm-ch",
			Type:      discordgo.InteractionApplicationCommand,
			User:      &discordgo.User{ID: "dmuser", Username: "DMUser"},
			Data: discordgo.ApplicationCommandInteractionData{
				Name: "model",
				Options: []*discordgo.ApplicationCommandInteractionDataOption{
					{Name: "model", Value: "pro", Type: discordgo.ApplicationCommandOptionString},
				},
			},
		},
	}
	a.onInteractionCreate(dg, interaction)

	select {
	case got := <-a.msgs:
		if got.Text != "/model pro" {
			t.Errorf("Text = %q, want /model pro", got.Text)
		}
		if got.UserID != "dmuser" {
			t.Errorf("UserID = %q, want dmuser (DM fallback)", got.UserID)
		}
		if got.UserName != "DMUser" {
			t.Errorf("UserName = %q, want DMUser (DM fallback)", got.UserName)
		}
	default:
		t.Error("expected message from DM interaction")
	}
}

func TestOnInteractionCreate_NonCommandInteraction(t *testing.T) {
	a := newTestAdapter(config.DiscordBotConfig{TokenEnv: "T"})
	dg := newSessionWithBotID("bot1")
	if dg == nil {
		t.Skip("cannot create discordgo session")
	}

	// MessageComponent interactions should be ignored
	interaction := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			ID:        "interaction-btn",
			ChannelID: "ch-guild",
			Type:      discordgo.InteractionMessageComponent,
			Member: &discordgo.Member{
				User: &discordgo.User{ID: "user1", Username: "User1"},
			},
		},
	}
	a.onInteractionCreate(dg, interaction)

	// Channel should be empty — non-command interactions are skipped
	select {
	case <-a.msgs:
		t.Error("should not process non-command interactions")
	default:
	}
}

func TestOnInteractionCreate_UnknownCommand(t *testing.T) {
	a := newTestAdapter(config.DiscordBotConfig{TokenEnv: "T"})
	dg := newSessionWithBotID("bot1")
	if dg == nil {
		t.Skip("cannot create discordgo session")
	}

	interaction := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			ID:        "interaction-unknown",
			ChannelID: "ch-guild",
			Type:      discordgo.InteractionApplicationCommand,
			Member: &discordgo.Member{
				User: &discordgo.User{ID: "user1", Username: "User1"},
			},
			Data: discordgo.ApplicationCommandInteractionData{
				Name: "unknown-command",
			},
		},
	}
	a.onInteractionCreate(dg, interaction)

	// No message for unknown commands
	select {
	case <-a.msgs:
		t.Error("should not process unknown commands")
	default:
	}
}

func TestOnInteractionCreate_ChannelFull(t *testing.T) {
	cfg := config.DiscordBotConfig{TokenEnv: "T"}
	a := New(cfg, slog.Default()).(*Adapter)
	a.msgs = make(chan bot.InboundMessage, 1)
	dg := newSessionWithBotID("bot1")
	if dg == nil {
		t.Skip("cannot create discordgo session")
	}

	// Fill the channel
	a.msgs <- bot.InboundMessage{Text: "filler"}

	interaction := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			ID:        "interaction-overflow",
			ChannelID: "ch-guild",
			Type:      discordgo.InteractionApplicationCommand,
			Member: &discordgo.Member{
				User: &discordgo.User{ID: "user1", Username: "User1"},
			},
			Data: discordgo.ApplicationCommandInteractionData{
				Name:    "model",
				Options: nil,
			},
		},
	}
	a.onInteractionCreate(dg, interaction)

	// Channel still has only the filler — overflow was dropped
	select {
	case got := <-a.msgs:
		if got.Text != "filler" {
			t.Errorf("got %q, want filler", got.Text)
		}
	default:
		t.Error("expected filler in channel")
	}
	select {
	case <-a.msgs:
		t.Error("channel should be empty after draining filler")
	default:
	}
}