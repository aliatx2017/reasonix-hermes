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
	"net/http"
	"os"
	"strings"
	"sync"

	"reasonix/internal/bot"
	"reasonix/internal/config"
	"reasonix/internal/netclient"

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
	dg.AddHandler(a.onInteractionCreate)

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
	if len(content) <= 1900 {
		return a.sendSingle(msg, content)
	}
	// Split long messages at paragraph/line boundaries to avoid mid-word cuts.
	chunks := splitContent(content, 1900)
	var lastID string
	for i, chunk := range chunks {
		m := msg
		// Only the first chunk replies to the original message.
		if i > 0 {
			m.ReplyToMsgID = ""
		}
		result, err := a.sendSingle(m, chunk)
		if err != nil {
			return result, err
		}
		lastID = result.MessageID
	}
	return bot.SendResult{MessageID: lastID}, nil
}

func (a *Adapter) sendSingle(msg bot.OutboundMessage, content string) (bot.SendResult, error) {
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
			Content:    content,
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

// NotifyWebhook sends a message to the configured Discord webhook URL
// (if WebhookURLEnv is set and the env var resolves). Used for lifecycle
// notifications (startup, shutdown, errors) without the gateway connection.
func (a *Adapter) NotifyWebhook(ctx context.Context, content string) error {
	envName := a.cfg.WebhookURLEnv
	if envName == "" {
		return nil
	}
	url := os.Getenv(envName)
	if url == "" {
		return nil
	}
	body := strings.NewReader(fmt.Sprintf(`{"content":%q}`, content))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return fmt.Errorf("discord webhook: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := netclient.DefaultClient().Do(req)
	if err != nil {
		return fmt.Errorf("discord webhook: %w", err)
	}
	resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("discord webhook: HTTP %d", resp.StatusCode)
	}
	return nil
}

// onReady handles the Discord ready event.
func (a *Adapter) onReady(s *discordgo.Session, event *discordgo.Ready) {
	a.logger.Info("discord: logged in", "username", event.User.Username, "guilds", len(event.Guilds))
	if err := s.UpdateGameStatus(0, "Reasonix AI Agent"); err != nil {
		a.logger.Warn("discord: failed to set game status", "err", err)
	}

	// Register slash commands on the configured guild (server). Global commands
	// would work too but take up to an hour to propagate; guild commands are instant.
	// First, delete any stale commands from previous bot versions.
	existing, err := s.ApplicationCommands(s.State.User.ID, a.cfg.ServerID)
	if err == nil {
		for _, cmd := range existing {
			if cmd.Name == "approve" || cmd.Name == "deny" {
				_ = s.ApplicationCommandDelete(s.State.User.ID, a.cfg.ServerID, cmd.ID)
				a.logger.Info("discord: deleted stale command", "cmd", cmd.Name)
			}
		}
	}

	cmd := &discordgo.ApplicationCommand{
		Name:        "model",
		Description: "Show or switch the AI model for this channel",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "model",
				Description: "Model to switch to (leave empty to show current)",
				Required:    false,
				Choices: []*discordgo.ApplicationCommandOptionChoice{
					{Name: "DeepSeek Flash (fast)", Value: "flash"},
					{Name: "DeepSeek Pro (deep)", Value: "pro"},
					{Name: "MiMo Pro (planner)", Value: "mimo"},
				},
			},
		},
	}
	if a.cfg.ServerID != "" {
		_, err := s.ApplicationCommandCreate(s.State.User.ID, a.cfg.ServerID, cmd)
		if err != nil {
			a.logger.Warn("discord: failed to register /model command on guild", "guild", a.cfg.ServerID, "err", err)
		} else {
			a.logger.Info("discord: registered /model command", "guild", a.cfg.ServerID)
		}
	}
}

// onMessageCreate handles incoming Discord messages and translates them
// to InboundMessage events.
func (a *Adapter) onMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore bot's own messages. Belt-and-suspenders: check both the author ID
	// (which requires State.User to be populated) and known bot status-response
	// patterns. The ID check can miss messages that arrive before the READY event
	// populates State.User, and double-send of approval responses can happen when
	// the gateway's handler and the sink's render overlap; the text filter catches
	// any that slip through.
	if m.Author.ID == s.State.User.ID {
		return
	}
	if botOwnStatusMessage(m.Content) {
		a.logger.Debug("discord: filtered bot status message", "content", m.Content[:min(40, len(m.Content))])
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

// onInteractionCreate handles Discord slash commands and other interactions.
// It converts them to text-message format so the gateway's existing slash-command
// parser handles them uniformly across all platforms.
func (a *Adapter) onInteractionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Only handle application commands for now
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}
	data := i.ApplicationCommandData()
	switch data.Name {
	case "model":
		modelName := ""
		if len(data.Options) > 0 {
			modelName = data.Options[0].StringValue()
		}
		text := "/model"
		if modelName != "" {
			text = "/model " + modelName
		}

		// Resolve user identity: i.Member is nil for DM interactions; i.User
		// is the fallback. Guild commands always have Member populated.
		userID := ""
		userName := ""
		if i.Member != nil && i.Member.User != nil {
			userID = i.Member.User.ID
			userName = i.Member.User.Username
		} else if i.User != nil {
			userID = i.User.ID
			userName = i.User.Username
		}

		// Respond ephemerally so only the caller sees it
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Processing /" + text + "...",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})

		// Push as a synthetic text message into the gateway pipeline
		select {
		case a.msgs <- bot.InboundMessage{
			Platform:  a.Platform(),
			ChatType:  bot.ChatGuild,
			ChatID:    i.ChannelID,
			UserID:    userID,
			UserName:  userName,
			Text:      text,
			MessageID: i.ID,
		}:
		default:
			a.logger.Warn("discord: message channel full, dropping interaction", "id", i.ID)
		}
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

// botOwnStatusMessage reports whether the text matches a known bot status
// response — a belt-and-suspenders filter for the rare case where the bot-author
// ID check misses a self-message.
var botStatusMessages = []string{
	"Approved.",
	"Denied.",
	"No pending action found.",
	"No pending approval in the current session.",
	"Task stopped.",
	"New session started.",
	"Answer submitted.",
	"Usage: /approve",
	"Usage: /deny",
	"Usage: /answer",
}

func botOwnStatusMessage(text string) bool {
	text = strings.TrimSpace(text)
	for _, s := range botStatusMessages {
		if text == s || strings.HasPrefix(text, s) {
			return true
		}
	}
	return false
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

// splitContent splits text into chunks ≤ maxLen, breaking at paragraph boundaries
// (double newline), then line boundaries, then word boundaries.
func splitContent(text string, maxLen int) []string {
	if len(text) <= maxLen {
		return []string{text}
	}
	var chunks []string
	for len(text) > maxLen {
		// Try paragraph break
		cut := lastBreakBefore(text[:maxLen], "\n\n")
		if cut < maxLen/2 {
			// Try line break
			cut = lastBreakBefore(text[:maxLen], "\n")
		}
		if cut < maxLen/2 {
			// Try space (word boundary)
			cut = lastBreakBefore(text[:maxLen], " ")
		}
		if cut < maxLen/2 {
			cut = maxLen // force-split
		}
		chunks = append(chunks, text[:cut])
		text = text[cut:]
		// Trim leading whitespace on continuation chunks
		text = strings.TrimLeft(text, "\n ")
	}
	if len(text) > 0 {
		chunks = append(chunks, text)
	}
	return chunks
}

func lastBreakBefore(s, sep string) int {
	i := strings.LastIndex(s, sep)
	if i < 0 {
		return -1
	}
	return i + len(sep)
}
