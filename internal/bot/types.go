// Package bot implements the Reasonix multi-channel IM bot message gateway,
// supporting QQ, Feishu, WeChat, and Discord. Architecture follows the
// Hermes Agent gateway/adapter/session pattern.
package bot

import "context"

// Platform identifies the IM platform.
type Platform string

const (
	PlatformQQ      Platform = "qq"
	PlatformFeishu  Platform = "feishu"
	PlatformWeixin  Platform = "weixin"
	PlatformDiscord Platform = "discord"
)

// ChatType identifies the conversation type.
type ChatType string

const (
	ChatDM     ChatType = "dm"
	ChatGroup  ChatType = "group"
	ChatGuild  ChatType = "guild"
	ChatDirect ChatType = "direct"
	ChatThread ChatType = "thread"
)

// SessionSource is a compound session identifier for generating a stable
// session key.
type SessionSource struct {
	Platform     Platform `json:"platform"`
	ConnectionID string   `json:"connection_id,omitempty"`
	Domain       string   `json:"domain,omitempty"`
	ChatType     ChatType `json:"chat_type"`
	ChatID       string   `json:"chat_id"`
	UserID       string   `json:"user_id"`
	ThreadID     string   `json:"thread_id,omitempty"`
}

// InboundMessage is an incoming message received from any platform.
type InboundMessage struct {
	Platform     Platform `json:"platform"`
	ConnectionID string   `json:"connection_id,omitempty"`
	Domain       string   `json:"domain,omitempty"`
	ChatType     ChatType `json:"chat_type"`
	ChatID       string   `json:"chat_id"`
	UserID       string   `json:"user_id"`
	UserName     string   `json:"user_name"`
	// OperatorID, when set, is the authenticated actor gated by the allowlist; UserID stays routing-only.
	OperatorID string   `json:"operator_id,omitempty"`
	Text       string   `json:"text"`
	MessageID  string   `json:"message_id"`
	ThreadID   string   `json:"thread_id,omitempty"`
	MediaURLs  []string `json:"media_urls,omitempty"`
	Raw        any      `json:"-"`
}

// Session derives the SessionSource from this message.
func (m InboundMessage) Session() SessionSource {
	return SessionSource{
		Platform:     m.Platform,
		ConnectionID: m.ConnectionID,
		Domain:       m.Domain,
		ChatType:     m.ChatType,
		ChatID:       m.ChatID,
		UserID:       m.UserID,
		ThreadID:     m.ThreadID,
	}
}

// OutboundMessage is a message to be sent to a platform.
type OutboundMessage struct {
	ConnectionID string           `json:"connection_id,omitempty"`
	Domain       string           `json:"domain,omitempty"`
	ChatID       string           `json:"chat_id"`
	ChatType     ChatType         `json:"chat_type,omitempty"`
	Text         string           `json:"text,omitempty"`
	MediaURLs    []string         `json:"media_urls,omitempty"`
	ReplyToMsgID string           `json:"reply_to_msg_id,omitempty"`
	Keyboard     *InlineKeyboard  `json:"keyboard,omitempty"`
	Card         *InteractiveCard `json:"card,omitempty"`
}

// InlineKeyboard is an inline keyboard (used for QQ approvals).
type InlineKeyboard struct {
	Rows []InlineKeyboardRow `json:"rows"`
}

// InlineKeyboardRow is a row of buttons.
type InlineKeyboardRow struct {
	Buttons []InlineKeyboardButton `json:"buttons"`
}

// InlineKeyboardButton is a single button.
type InlineKeyboardButton struct {
	ID         string `json:"id"`
	Label      string `json:"label"`
	Style      int    `json:"style,omitempty"` // 0=default, 1=primary, 2=danger
	CallbackID string `json:"callback_id,omitempty"`
}

// InteractiveCard is an interactive card (used for Feishu approvals/ask).
type InteractiveCard struct {
	Header   string                   `json:"header"`
	Elements []InteractiveCardElement `json:"elements"`
}

// InteractiveCardElement is an element within a card.
type InteractiveCardElement struct {
	Tag     string         `json:"tag"`
	Content string         `json:"content,omitempty"`
	Extra   map[string]any `json:"extra,omitempty"`
}

// SendResult is the result of sending a message.
type SendResult struct {
	MessageID string `json:"message_id,omitempty"`
	Err       error  `json:"err,omitempty"`
}

// Adapter is the platform adapter interface. Each platform implements one.
type Adapter interface {
	// Platform returns the platform identifier.
	Platform() Platform

	// Start starts the adapter and connects to the platform gateway.
	Start(ctx context.Context) error

	// Stop gracefully shuts down the adapter.
	Stop() error

	// Send sends an outbound message.
	Send(ctx context.Context, msg OutboundMessage) (SendResult, error)

	// SendTyping sends a "user is typing" indicator.
	SendTyping(ctx context.Context, chatID string) error

	// Messages returns the inbound message channel.
	Messages() <-chan InboundMessage

	// Name returns the adapter instance name (for logging).
	Name() string
}

// MessageHandler is the callback BotGateway uses to process inbound messages.
type MessageHandler func(ctx context.Context, msg InboundMessage)
