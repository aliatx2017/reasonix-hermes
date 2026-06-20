// Package telegram implements the Reasonix bot Adapter interface for Telegram
// using the go-telegram-bot-api. It translates Telegram update events into
// Reasonix InboundMessage channels and sends OutboundMessage back to Telegram
// chats.
//
// Architecture: telegram.Adapter → bot.BotGateway → control.Controller per session.
// The gateway handles concurrency, debounce, slash commands, approval, and ask
// flows. This adapter only handles platform I/O (Telegram ↔ Reasonix types).
package telegram

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"reasonix/internal/bot"
	"reasonix/internal/config"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Adapter implements bot.Adapter for Telegram.
type Adapter struct {
	cfg     config.TelegramBotConfig
	logger  *slog.Logger
	api     *tgbotapi.BotAPI
	updates tgbotapi.UpdatesChannel
	msgs    chan bot.InboundMessage
	cancel  context.CancelFunc
}

// New creates a Telegram adapter from config.
func New(cfg config.TelegramBotConfig, logger *slog.Logger) bot.Adapter {
	if logger == nil {
		logger = slog.Default()
	}
	return &Adapter{
		cfg:    cfg,
		logger: logger.With("component", "telegram_adapter"),
		msgs:   make(chan bot.InboundMessage, 64),
	}
}

// Platform returns the Telegram platform identifier.
func (a *Adapter) Platform() bot.Platform { return bot.PlatformTelegram }

// Name returns a human-readable name for logging.
func (a *Adapter) Name() string { return "telegram" }

// Start connects to Telegram Bot API and begins receiving updates via long polling.
func (a *Adapter) Start(ctx context.Context) error {
	token := os.Getenv(a.cfg.TokenEnv)
	if token == "" {
		return fmt.Errorf("telegram: bot token not found (env %s)", a.cfg.TokenEnv)
	}

	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return fmt.Errorf("telegram: create bot: %w", err)
	}
	a.api = api
	a.logger.Info("telegram: authorized", "username", api.Self.UserName)

	ctx, cancel := context.WithCancel(ctx)
	a.cancel = cancel

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	a.updates = api.GetUpdatesChan(u)

	// Poll updates in a goroutine
	go func() {
		defer close(a.msgs)
		for {
			select {
			case <-ctx.Done():
				return
			case update, ok := <-a.updates:
				if !ok {
					return
				}
				a.handleUpdate(update)
			}
		}
	}()

	// Shutdown on context cancel
	go func() {
		<-ctx.Done()
		a.logger.Info("telegram: shutting down")
		api.StopReceivingUpdates()
	}()

	a.logger.Info("telegram: adapter started")
	return nil
}

// Stop stops the long-polling loop.
func (a *Adapter) Stop() error {
	if a.cancel != nil {
		a.cancel()
	}
	return nil
}

// Send dispatches an OutboundMessage to Telegram.
func (a *Adapter) Send(ctx context.Context, msg bot.OutboundMessage) (bot.SendResult, error) {
	if a.api == nil {
		return bot.SendResult{}, fmt.Errorf("telegram: not connected")
	}

	// Parse ChatID as int64 (Telegram chat IDs are numeric)
	var chatID int64
	if _, err := fmt.Sscanf(msg.ChatID, "%d", &chatID); err != nil {
		return bot.SendResult{}, fmt.Errorf("telegram: invalid chat ID %q: %w", msg.ChatID, err)
	}

	content := msg.Text

	// Split long messages
	if len(content) > 4000 {
		chunks := splitContent(content, 4000)
		var lastID int
		for i, chunk := range chunks {
			tgMsg := tgbotapi.NewMessage(chatID, chunk)
			if i == 0 {
				// Only reply to the first chunk
				if msg.ReplyToMsgID != "" {
					var replyID int
					if _, err := fmt.Sscanf(msg.ReplyToMsgID, "%d", &replyID); err == nil {
						tgMsg.ReplyToMessageID = replyID
					}
				}
			}
			result, err := a.api.Send(tgMsg)
			if err != nil {
				return bot.SendResult{}, fmt.Errorf("telegram: send chunk: %w", err)
			}
			lastID = result.MessageID
		}
		return bot.SendResult{MessageID: fmt.Sprintf("%d", lastID)}, nil
	}

	tgMsg := tgbotapi.NewMessage(chatID, content)
	if msg.ReplyToMsgID != "" {
		var replyID int
		if _, err := fmt.Sscanf(msg.ReplyToMsgID, "%d", &replyID); err == nil {
			tgMsg.ReplyToMessageID = replyID
		}
	}

	result, err := a.api.Send(tgMsg)
	if err != nil {
		return bot.SendResult{}, fmt.Errorf("telegram: send: %w", err)
	}
	return bot.SendResult{MessageID: fmt.Sprintf("%d", result.MessageID)}, nil
}

// SendTyping sends the "typing" action to a Telegram chat.
func (a *Adapter) SendTyping(ctx context.Context, chatID string) error {
	if a.api == nil {
		return nil
	}
	var id int64
	if _, err := fmt.Sscanf(chatID, "%d", &id); err != nil {
		return fmt.Errorf("telegram: invalid chat ID: %w", err)
	}
	_, err := a.api.Request(tgbotapi.NewChatAction(id, tgbotapi.ChatTyping))
	return err
}

// Messages returns the inbound message channel.
func (a *Adapter) Messages() <-chan bot.InboundMessage {
	return a.msgs
}

func (a *Adapter) handleUpdate(update tgbotapi.Update) {
	if update.Message == nil {
		return
	}
	m := update.Message

	// Ignore bot's own messages
	if m.From != nil && m.From.IsBot {
		return
	}

	chatID := fmt.Sprintf("%d", m.Chat.ID)
	chatType := bot.ChatGroup
	var threadID string

	if m.Chat.IsPrivate() {
		// DM: only respond if AllowDMs enabled (defaults to true)
		if !a.cfg.AllowDMs && a.cfg.Enabled {
			return
		}
		chatType = bot.ChatDM
	} else if m.Chat.IsSuperGroup() || m.Chat.IsGroup() {
		chatType = bot.ChatGroup
		if m.IsCommand() {
			// TODO: handle group commands with @botname
			_ = chatType // placeholder; will be replaced when command handling is added
		}
	} else if m.Chat.IsChannel() {
		chatType = bot.ChatGuild
	}

	userID := ""
	userName := ""
	if m.From != nil {
		userID = fmt.Sprintf("%d", m.From.ID)
		userName = m.From.FirstName
		if m.From.LastName != "" {
			userName += " " + m.From.LastName
		}
		if m.From.UserName != "" {
			if userName == "" {
				userName = "@" + m.From.UserName
			}
		}
	}

	// Collect media URLs (photos, documents)
	var mediaURLs []string
	if len(m.Photo) > 0 {
		// Use the largest photo (last in the array)
		largest := m.Photo[len(m.Photo)-1]
		if link, err := a.api.GetFileDirectURL(largest.FileID); err == nil {
			mediaURLs = append(mediaURLs, link)
		}
	}
	if m.Document != nil {
		if link, err := a.api.GetFileDirectURL(m.Document.FileID); err == nil {
			mediaURLs = append(mediaURLs, link)
		}
	}

	ib := bot.InboundMessage{
		Platform:  bot.PlatformTelegram,
		ChatType:  chatType,
		ChatID:    chatID,
		UserID:    userID,
		UserName:  userName,
		Text:      m.Text,
		MessageID: fmt.Sprintf("%d", m.MessageID),
		ThreadID:  threadID,
		MediaURLs: mediaURLs,
	}

	select {
	case a.msgs <- ib:
	default:
		a.logger.Warn("telegram: inbound channel full, dropping message", "chat", chatID)
	}
}

// splitContent splits a long message at paragraph or line boundaries.
func splitContent(content string, maxLen int) []string {
	if len(content) <= maxLen {
		return []string{content}
	}
	var chunks []string
	for len(content) > maxLen {
		// Prefer paragraph breaks
		idx := lastBreakIndex(content[:maxLen], "\n\n")
		if idx < 0 {
			idx = lastBreakIndex(content[:maxLen], "\n")
		}
		if idx < 0 {
			idx = lastBreakIndex(content[:maxLen], ". ")
		}
		if idx < 0 {
			idx = lastSpaceIndex(content[:maxLen])
		}
		if idx <= 0 {
			idx = maxLen
		}
		chunks = append(chunks, content[:idx])
		// Skip the delimiter
		skip := idx
		for skip < len(content) && (content[skip] == ' ' || content[skip] == '\n') {
			skip++
		}
		content = content[skip:]
	}
	if len(content) > 0 {
		chunks = append(chunks, content)
	}
	return chunks
}

func lastBreakIndex(s, sep string) int {
	for i := len(s) - len(sep); i >= 0; i-- {
		if s[i:i+len(sep)] == sep {
			return i
		}
	}
	return -1
}

func lastSpaceIndex(s string) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == ' ' {
			return i
		}
	}
	return -1
}
