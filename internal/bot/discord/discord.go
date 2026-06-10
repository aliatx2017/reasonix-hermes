// Package discord implements the Reasonix bot Adapter interface for Discord
// using discordgo. It translates Discord gateway events into Reasonix
// InboundMessage channels and sends OutboundMessage back to Discord channels.
//
// Architecture: discord.Adapter → bot.BotGateway → control.Controller per session.
// The gateway handles concurrency, debounce, slash commands, approval, and ask
// flows. This adapter only handles platform I/O (Discord ↔ Reasonix types).
package discord

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"

	"reasonix/internal/bot"
	"reasonix/internal/config"

	"github.com/bwmarrin/discordgo"
)

// Adapter implements bot.Adapter for Discord.
type Adapter struct {
	cfg     config.DiscordBotConfig
	logger  *slog.Logger
	session *discordgo.Session
	msgs    chan bot.InboundMessage

	mu       sync.Mutex
	chatType map[string]bot.ChatType // channelID → ChatType cache
}

// New creates a Discord adapter from config.
func New(cfg config.DiscordBotConfig, logger *slog.Logger) bot.Adapter {
	if logger == nil {
		logger = slog.Default()
	}
	return &Adapter{
		cfg:      cfg,
		logger:   logger.With("component", "discord_adapter"),
		msgs:     make(chan bot.InboundMessage, 64),
		chatType: make(map[string]bot.ChatType),
	}
}

// Platform returns the Discord platform identifier.
func (a *Adapter) Platform() bot.Platform { return "discord" }

// Name returns a human-readable name for logging.
func (a *Adapter) Name() string { return "discord" }

// Start connects to Discord gateway and begins listening for messages.
func (a *Adapter) Start(ctx context.Context) error {
	token := os.Getenv(a.cfg.TokenEnv)
	if token == "" {
		// Fallback: try DISCORD_BOT_TOKEN directly
		token = os.Getenv("DISCORD_BOT_TOKEN")
	}
	if token == "" {
		return fmt.Errorf("discord: bot token not found (env %s or DISCORD_BOT_TOKEN)", a.cfg.TokenEnv)
	}

	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		return fmt.Errorf("discord: create session: %w", err)
	}
	a.session = dg

	// Identify intents
	dg.Identify.Intents = discordgo.IntentsGuildMessages |
		discordgo.IntentsDirectMessages |
		discordgo.IntentsMessageContent

	// Register message handler
	dg.AddHandler(a.onMessageCreate)
	dg.AddHandler(a.onReady)

	if err := dg.Open(); err != nil {
		return fmt.Errorf("discord: open gateway: %w", err)
	}

	// Shutdown on context cancel
	go func() {
		<-ctx.Done()
		a.logger.Info("discord: shutting down")
		dg.Close()
	}()

	a.logger.Info("discord: adapter started")
	return nil
}

// Stop closes the Discord gateway connection.
func (a *Adapter) Stop() error {
	if a.session != nil {
		return a.session.Close()
	}
	return nil
}

// Send dispatches an OutboundMessage to Discord.
func (a *Adapter) Send(ctx context.Context, msg bot.OutboundMessage) (bot.SendResult, error) {
	if a.session == nil {
		return bot.SendResult{}, fmt.Errorf("discord: session not initialized")
	}

	content := msg.Text
	if len(content) > 1900 {
		content = content[:1897] + "..."
	}

	// Use Discord embeds for richer messages if Card is provided
	if msg.Card != nil {
		embed := &discordgo.MessageEmbed{
			Title:       msg.Card.Header,
			Description: cardToMarkdown(msg.Card),
			Color:       0x5865F2, // Discord blurple
		}
		m, err := a.session.ChannelMessageSendEmbed(msg.ChatID, embed)
		if err != nil {
			return bot.SendResult{}, fmt.Errorf("discord: send embed: %w", err)
		}
		return bot.SendResult{MessageID: m.ID}, nil
	}

	// For messages with inline keyboard (approval buttons)
	if msg.Keyboard != nil {
		m, err := a.session.ChannelMessageSendComplex(msg.ChatID, &discordgo.MessageSend{
			Content: content,
			Components: keyboardToComponents(msg.Keyboard),
		})
		if err != nil {
			return bot.SendResult{}, fmt.Errorf("discord: send with keyboard: %w", err)
		}
		return bot.SendResult{MessageID: m.ID}, nil
	}

	// Plain message
	if msg.ReplyToMsgID != "" {
		// Reply to specific message
		m, err := a.session.ChannelMessageSendComplex(msg.ChatID, &discordgo.MessageSend{
			Content:   content,
			Reference: &discordgo.MessageReference{MessageID: msg.ReplyToMsgID, ChannelID: msg.ChatID},
		})
		if err != nil {
			// Fallback: send without reply reference
			m, err = a.session.ChannelMessageSend(msg.ChatID, content)
			if err != nil {
				return bot.SendResult{}, fmt.Errorf("discord: send: %w", err)
			}
		}
		return bot.SendResult{MessageID: m.ID}, nil
	}

	m, err := a.session.ChannelMessageSend(msg.ChatID, content)
	if err != nil {
		return bot.SendResult{}, fmt.Errorf("discord: send: %w", err)
	}
	return bot.SendResult{MessageID: m.ID}, nil
}

// SendTyping sends the "typing" indicator to a Discord channel.
func (a *Adapter) SendTyping(ctx context.Context, chatID string) error {
	if a.session == nil {
		return nil
	}
	return a.session.ChannelTyping(chatID)
}

// Messages returns the inbound message channel.
func (a *Adapter) Messages() <-chan bot.InboundMessage {
	return a.msgs
}

// onReady handles the Discord ready event.
func (a *Adapter) onReady(s *discordgo.Session, event *discordgo.Ready) {
	a.logger.Info("discord: logged in", "username", event.User.Username, "guilds", len(event.Guilds))
	s.UpdateGameStatus(0, "Reasonix AI Agent")
}

// onMessageCreate handles incoming Discord messages and translates them
// to InboundMessage events.
func (a *Adapter) onMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore bot's own messages
	if m.Author.ID == s.State.User.ID {
		return
	}

	// Channel filter
	if a.cfg.ChannelID != "" && m.ChannelID != a.cfg.ChannelID {
		return
	}

	// Skip empty messages (embeds, attachments without text)
	if m.Content == "" {
		return
	}

	// Determine chat type
	chatType := a.resolveChatType(s, m)

	// DM filter
	if chatType == bot.ChatDM && !a.cfg.AllowDMs {
		return
	}

	// Extract text: strip bot mention prefix if present
	text := m.Content
	text = stripMention(text, s.State.User.ID)

	select {
	case a.msgs <- bot.InboundMessage{
		Platform:  a.Platform(),
		ChatType:  chatType,
		ChatID:    m.ChannelID,
		UserID:    m.Author.ID,
		UserName:  m.Author.Username,
		Text:      strings.TrimSpace(text),
		MessageID: m.ID,
	}:
	default:
		a.logger.Warn("discord: message channel full, dropping message", "msg_id", m.ID)
	}
}

// resolveChatType determines the ChatType from channel state.
func (a *Adapter) resolveChatType(s *discordgo.Session, m *discordgo.MessageCreate) bot.ChatType {
	a.mu.Lock()
	if ct, ok := a.chatType[m.ChannelID]; ok {
		a.mu.Unlock()
		return ct
	}
	a.mu.Unlock()

	ch, err := s.State.Channel(m.ChannelID)
	if err != nil {
		// Fallback: try direct API
		ch, err = s.Channel(m.ChannelID)
		if err != nil {
			// Can't determine; default to group
			return bot.ChatGroup
		}
	}

	var ct bot.ChatType
	switch ch.Type {
	case discordgo.ChannelTypeDM:
		ct = bot.ChatDM
	case discordgo.ChannelTypeGuildText, discordgo.ChannelTypeGuildPublicThread:
		ct = bot.ChatGuild
	case discordgo.ChannelTypeGuildPrivateThread:
		ct = bot.ChatThread
	case discordgo.ChannelTypeGroupDM:
		ct = bot.ChatGroup
	default:
		ct = bot.ChatGuild
	}

	a.mu.Lock()
	a.chatType[m.ChannelID] = ct
	a.mu.Unlock()

	return ct
}

// stripMention removes <@BOT_ID> or <@!BOT_ID> from text.
func stripMention(text, botID string) string {
	prefixes := []string{
		"<@" + botID + ">",
		"<@!" + botID + ">",
	}
	for _, p := range prefixes {
		text = strings.TrimPrefix(text, p)
	}
	return strings.TrimSpace(text)
}

// cardToMarkdown converts an InteractiveCard to a plain markdown string.
func cardToMarkdown(card *bot.InteractiveCard) string {
	var b strings.Builder
	for _, el := range card.Elements {
		if el.Tag == "markdown" && el.Content != "" {
			b.WriteString(el.Content)
			b.WriteString("\n")
		}
	}
	return b.String()
}

// keyboardToComponents converts bot.InlineKeyboard to Discord button components.
func keyboardToComponents(kb *bot.InlineKeyboard) []discordgo.MessageComponent {
	if kb == nil {
		return nil
	}

	var rows []discordgo.MessageComponent
	for _, row := range kb.Rows {
		var buttons []discordgo.MessageComponent
		for _, btn := range row.Buttons {
			style := discordgo.SecondaryButton
			switch btn.Style {
			case 1:
				style = discordgo.PrimaryButton
			case 2:
				style = discordgo.DangerButton
			}
			buttons = append(buttons, discordgo.Button{
				Label:    btn.Label,
				Style:    style,
				CustomID: btn.CallbackID,
			})
		}
		rows = append(rows, discordgo.ActionsRow{Components: buttons})
	}
	return rows
}