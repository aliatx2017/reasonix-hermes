// Package line implements the Reasonix bot Adapter interface for LINE
// using the line-bot-sdk-go v8. LINE pushes events via webhook, so this adapter
// runs a small HTTP server to receive events and translates them into
// Reasonix InboundMessage channels. Outbound messages are sent via LINE's
// reply API.
//
// Architecture: line.Adapter → bot.BotGateway → control.Controller per session.
// The gateway handles concurrency, debounce, slash commands, approval, and ask
// flows. This adapter only handles platform I/O (LINE ↔ Reasonix types).
package line

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"

	"reasonix/internal/bot"
	"reasonix/internal/config"

	"github.com/line/line-bot-sdk-go/v8/linebot"
)

// Adapter implements bot.Adapter for LINE.
type Adapter struct {
	cfg      config.LineBotConfig
	logger   *slog.Logger
	client   *linebot.Client
	msgs     chan bot.InboundMessage
	cancel   context.CancelFunc
	server   *http.Server
	listener net.Listener
}

// New creates a LINE adapter from config.
func New(cfg config.LineBotConfig, logger *slog.Logger) bot.Adapter {
	if logger == nil {
		logger = slog.Default()
	}
	return &Adapter{
		cfg:    cfg,
		logger: logger.With("component", "line_adapter"),
		msgs:   make(chan bot.InboundMessage, 64),
	}
}

// Platform returns the LINE platform identifier.
func (a *Adapter) Platform() bot.Platform { return bot.PlatformLine }

// Name returns a human-readable name for logging.
func (a *Adapter) Name() string { return "line" }

// Start creates the LINE bot client and begins an HTTP server for webhooks.
func (a *Adapter) Start(ctx context.Context) error {
	token := os.Getenv(a.cfg.TokenEnv)
	if token == "" {
		return fmt.Errorf("line: channel token not found (env %s)", a.cfg.TokenEnv)
	}
	secret := os.Getenv(a.cfg.SecretEnv)
	if secret == "" {
		return fmt.Errorf("line: channel secret not found (env %s)", a.cfg.SecretEnv)
	}

	client, err := linebot.New(secret, token)
	if err != nil {
		return fmt.Errorf("line: create bot: %w", err)
	}
	a.client = client
	a.logger.Info("line: client created")

	ctx, cancel := context.WithCancel(ctx)
	a.cancel = cancel

	// Start HTTP server on a free port for LINE webhook.
	mux := http.NewServeMux()
	mux.HandleFunc("/webhook", a.handleWebhook)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		cancel()
		return fmt.Errorf("line: listen: %w", err)
	}
	a.listener = listener

	addr := listener.Addr().String()
	a.server = &http.Server{Handler: mux}
	a.logger.Info("line: webhook server listening", "addr", addr, "path", "/webhook")

	go func() {
		if err := a.server.Serve(listener); err != nil && err != http.ErrServerClosed {
			a.logger.Error("line: server error", "err", err)
		}
	}()

	// Shutdown on context cancel.
	go func() {
		<-ctx.Done()
		a.logger.Info("line: shutting down")
		a.server.Shutdown(context.Background())
		close(a.msgs)
	}()

	a.logger.Info("line: adapter started", "webhook_url", "http://"+addr+"/webhook")
	return nil
}

// Stop shuts down the webhook server.
func (a *Adapter) Stop() error {
	if a.cancel != nil {
		a.cancel()
	}
	return nil
}

// Messages returns the inbound message channel.
func (a *Adapter) Messages() <-chan bot.InboundMessage {
	return a.msgs
}

// Send sends an outbound message via LINE's reply API.
func (a *Adapter) Send(ctx context.Context, msg bot.OutboundMessage) (bot.SendResult, error) {
	if a.client == nil {
		return bot.SendResult{}, fmt.Errorf("line: client not initialized")
	}

	// Use the message's reply-to ID as the LINE reply token.
	replyToken := msg.ReplyToMsgID
	if replyToken == "" {
		return bot.SendResult{}, fmt.Errorf("line: no reply token in outbound message")
	}

	text := msg.Text
	if text == "" {
		return bot.SendResult{}, nil
	}

	// Split long messages at paragraph boundaries, up to LINE's 5000-char limit.
	const maxChars = 5000
	var texts []string
	for len(text) > maxChars {
		split := maxChars
		// Try to split at last paragraph boundary within limit.
		if idx := lastNewlineBefore(text, maxChars); idx > maxChars/2 {
			split = idx
		}
		texts = append(texts, text[:split])
		text = text[split:]
	}
	texts = append(texts, text)

	var lastMsgID string
	for _, t := range texts {
		reply := linebot.NewTextMessage(t)
		res, err := a.client.ReplyMessage(replyToken, reply).Do()
		if err != nil {
			return bot.SendResult{}, fmt.Errorf("line: reply: %w", err)
		}
		lastMsgID = res.RequestID
	}

	return bot.SendResult{MessageID: lastMsgID}, nil
}

// SendTyping sends a "user is typing" indicator (not supported by LINE's reply API).
func (a *Adapter) SendTyping(ctx context.Context, chatID string) error {
	// LINE doesn't have a typing indicator API — no-op.
	return nil
}

// handleWebhook processes incoming LINE webhook events.
func (a *Adapter) handleWebhook(w http.ResponseWriter, r *http.Request) {
	events, err := a.client.ParseRequest(r)
	if err != nil {
		a.logger.Warn("line: parse webhook error", "err", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	for _, event := range events {
		if event.Type == linebot.EventTypeMessage {
			a.handleMessageEvent(event)
		}
	}

	w.WriteHeader(http.StatusOK)
}

// handleMessageEvent translates a LINE text message into a bot.InboundMessage.
func (a *Adapter) handleMessageEvent(e *linebot.Event) {
	// Only handle text messages.
	textMsg, ok := e.Message.(*linebot.TextMessage)
	if !ok {
		return
	}

	src := e.Source
	chatType := bot.ChatDM
	chatID := src.UserID

	if src.GroupID != "" {
		chatType = bot.ChatGroup
		chatID = src.GroupID
	}
	if src.RoomID != "" {
		chatType = bot.ChatGroup
		chatID = src.RoomID
	}

	msg := bot.InboundMessage{
		Platform:  bot.PlatformLine,
		ChatType:  chatType,
		ChatID:    chatID,
		UserID:    src.UserID,
		UserName:  src.UserID, // LINE doesn't expose display name in message events
		Text:      textMsg.Text,
		MessageID: e.ReplyToken,
		Raw:       e,
	}

	select {
	case a.msgs <- msg:
	default:
		a.logger.Warn("line: message dropped — channel full")
	}
}

func lastNewlineBefore(s string, limit int) int {
	for i := limit; i >= 0; i-- {
		if i < len(s) && s[i] == '\n' {
			return i
		}
	}
	return -1
}
