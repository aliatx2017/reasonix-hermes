// Package slack implements the Reasonix bot Adapter interface for Slack
// using the slack-go/slack library with Socket Mode.
package slack

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"

	"reasonix/internal/bot"
	"reasonix/internal/config"
)

// Adapter implements bot.Adapter for Slack via Socket Mode.
type Adapter struct {
	cfg    config.SlackBotConfig
	logger *slog.Logger

	mu      sync.Mutex
	client  *slack.Client
	sock    *socketmode.Client
	ctx     context.Context
	cancel  context.CancelFunc
	running bool
	msgCh   chan bot.InboundMessage
	userID  string // bot's own Slack user ID
}

// New creates a Slack adapter from config.
func New(cfg config.SlackBotConfig, logger *slog.Logger) bot.Adapter {
	if logger == nil {
		logger = slog.Default()
	}
	return &Adapter{
		cfg:    cfg,
		logger: logger.With("component", "slack_adapter"),
		msgCh:  make(chan bot.InboundMessage, 100),
	}
}

// Platform returns the Slack platform identifier.
func (a *Adapter) Platform() bot.Platform { return bot.PlatformSlack }

// Name returns the adapter name.
func (a *Adapter) Name() string { return "slack" }

// Start connects to Slack via Socket Mode.
func (a *Adapter) Start(parentCtx context.Context) error {
	token := os.Getenv(a.cfg.TokenEnv)
	if token == "" {
		return fmt.Errorf("slack: bot token not found (env %s)", a.cfg.TokenEnv)
	}

	appToken := os.Getenv(a.cfg.AppTokenEnv)
	if appToken == "" {
		return fmt.Errorf("slack: app token not found (env %s)", a.cfg.AppTokenEnv)
	}

	a.ctx, a.cancel = context.WithCancel(parentCtx)
	a.client = slack.New(token, slack.OptionDebug(false))
	a.sock = socketmode.New(a.client, socketmode.OptionDebug(false))

	a.mu.Lock()
	a.running = true
	a.mu.Unlock()

	// Fetch bot identity.
	auth, err := a.client.AuthTestContext(a.ctx)
	if err != nil {
		a.logger.Warn("slack: auth test failed", "err", err)
	} else {
		a.userID = auth.UserID
		a.logger.Info("slack: connected", "user", auth.User, "team", auth.Team)
	}

	// Socket mode event loop.
	go func() {
		for {
			select {
			case <-a.ctx.Done():
				return
			case evt, ok := <-a.sock.Events:
				if !ok {
					return
				}
				switch evt.Type {
				case socketmode.EventTypeEventsAPI:
					apiEvent, ok := evt.Data.(slackevents.EventsAPIEvent)
					if !ok {
						continue
					}
					if err := a.sock.Ack(*evt.Request); err != nil {
						a.logger.Warn("slack: ack", "err", err)
					}
					a.handleEvent(apiEvent)
				case socketmode.EventTypeDisconnect:
					a.logger.Info("slack: socket disconnected")
				}
			}
		}
	}()

	if err := a.sock.RunContext(a.ctx); err != nil {
		if a.ctx.Err() != nil {
			return nil // graceful shutdown
		}
		return fmt.Errorf("slack: socket run: %w", err)
	}
	return nil
}

// Stop shuts down the adapter.
func (a *Adapter) Stop() error {
	a.mu.Lock()
	a.running = false
	a.mu.Unlock()
	if a.cancel != nil {
		a.cancel()
	}
	return nil
}

// Messages returns the inbound message channel.
func (a *Adapter) Messages() <-chan bot.InboundMessage {
	return a.msgCh
}

// Send dispatches an OutboundMessage to Slack.
func (a *Adapter) Send(_ context.Context, msg bot.OutboundMessage) (bot.SendResult, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.client == nil {
		return bot.SendResult{}, fmt.Errorf("slack: not connected")
	}

	chunks := splitSlackMessage(msg.Text)
	var lastTS string
	for _, chunk := range chunks {
		opts := []slack.MsgOption{
			slack.MsgOptionText(chunk, false),
		}
		if lastTS != "" {
			opts = append(opts, slack.MsgOptionTS(lastTS))
		}
		_, ts, err := a.client.PostMessageContext(a.ctx, msg.ChatID, opts...)
		if err != nil {
			return bot.SendResult{}, fmt.Errorf("slack: send: %w", err)
		}
		lastTS = ts
	}
	return bot.SendResult{MessageID: lastTS}, nil
}

// SendTyping is a no-op for Slack Socket Mode.
func (a *Adapter) SendTyping(_ context.Context, _ string) error { return nil }

// WebhookURL returns empty — Slack uses Socket Mode, not webhooks.
func (a *Adapter) WebhookURL() string { return "" }

// --- internal ---

func (a *Adapter) handleEvent(evt slackevents.EventsAPIEvent) {
	switch e := evt.InnerEvent.Data.(type) {
	case *slack.MessageEvent:
		a.handleMessage(e)
	}
}

func (a *Adapter) handleMessage(evt *slack.MessageEvent) {
	if evt.BotID != "" || evt.User == "" || evt.Text == "" {
		return
	}
	if evt.User == a.userID {
		return
	}
	// Respond to DMs and @mentions of the bot.
	isDM := strings.HasPrefix(evt.Channel, "D")
	mentioned := strings.Contains(evt.Text, fmt.Sprintf("<@%s>", a.userID))
	if !isDM && !mentioned {
		return
	}

	text := strings.ReplaceAll(evt.Text, fmt.Sprintf("<@%s>", a.userID), "")
	text = strings.TrimSpace(text)

	a.enqueueMessage(text, evt.User, evt.Channel, evt.Timestamp, evt.ThreadTimestamp)
}

func (a *Adapter) enqueueMessage(text, userID, channelID, msgID, threadTS string) {
	a.mu.Lock()
	running := a.running
	a.mu.Unlock()
	if !running {
		return
	}

	chatID := channelID
	if threadTS != "" {
		chatID = fmt.Sprintf("%s:%s", channelID, threadTS)
	}

	msg := bot.InboundMessage{
		ChatID:    chatID,
		UserID:    userID,
		Text:      text,
		MessageID: msgID,
		Platform:  bot.PlatformSlack,
	}

	select {
	case a.msgCh <- msg:
	default:
		a.logger.Warn("slack: inbound channel full, dropping message", "user", userID)
	}
}

func splitSlackMessage(text string) []string {
	const maxLen = 4000
	if len(text) <= maxLen {
		return []string{text}
	}
	var chunks []string
	remaining := text
	for len(remaining) > maxLen {
		cut := maxLen
		if idx := strings.LastIndex(remaining[:maxLen], "\n\n"); idx > maxLen/2 {
			cut = idx + 2
		} else if idx := strings.LastIndex(remaining[:maxLen], "\n"); idx > maxLen/2 {
			cut = idx + 1
		}
		chunks = append(chunks, remaining[:cut])
		remaining = remaining[cut:]
	}
	if len(remaining) > 0 {
		chunks = append(chunks, remaining)
	}
	return chunks
}
