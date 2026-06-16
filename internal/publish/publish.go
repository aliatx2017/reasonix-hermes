// Package publish exports agent session transcripts as HTML or JSON for
// sharing, review, or archival. HTML output is self-contained (all CSS inline,
// no external dependencies) with syntax-highlighted code blocks.
package publish

import (
	"encoding/json"
	"fmt"
	"html"
	"strings"
	"time"

	"reasonix/internal/provider"
)

// Session is the minimum data needed to publish a transcript.
// Stat fields (TokensIn, TokensOut, Turns, Cost) are optional — when all are
// zero, the stats line is omitted from HTML output.
type Session struct {
	Title     string             `json:"title,omitempty"`
	Model     string             `json:"model,omitempty"`
	Date      time.Time          `json:"date"`
	Messages  []provider.Message `json:"messages"`
	TokensIn  int                `json:"tokensIn,omitempty"`
	TokensOut int                `json:"tokensOut,omitempty"`
	Turns     int                `json:"turns,omitempty"`
	Cost      float64            `json:"cost,omitempty"`
}

// ToJSON serializes the session as a JSON document.
func ToJSON(s Session) ([]byte, error) {
	out, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("publish: marshal: %w", err)
	}
	return out, nil
}

// ToHTML renders the session as a self-contained HTML document with inline CSS
// and basic syntax highlighting for code blocks.
func ToHTML(s Session) string {
	var b strings.Builder

	// ── document head ──
	b.WriteString(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>`)
	b.WriteString(html.EscapeString(s.Title))
	if s.Title == "" {
		b.WriteString("Session Transcript")
	}
	b.WriteString(`</title>
<style>
:root { color-scheme: light dark; }
body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; max-width: 900px; margin: 0 auto; padding: 2rem 1rem; line-height: 1.6; color: #1a1a2e; background: #fafafa; }
@media (prefers-color-scheme: dark) { body { color: #e0e0e0; background: #1a1a2e; } }
h1 { font-size: 1.6rem; margin-bottom: 0.3rem; }
.meta { color: #666; font-size: 0.9rem; margin-bottom: 2rem; }
@media (prefers-color-scheme: dark) { .meta { color: #999; } }
.msg { margin-bottom: 1.5rem; padding: 1rem; border-radius: 8px; }
.msg.system { background: #f0f0f5; border-left: 3px solid #999; }
.msg.user { background: #e8f0fe; border-left: 3px solid #4285f4; }
.msg.assistant { background: #fff; border-left: 3px solid #34a853; border: 1px solid #e0e0e0; }
.msg.tool { background: #fff8e1; border-left: 3px solid #f9ab00; font-size: 0.9rem; }
@media (prefers-color-scheme: dark) {
  .msg.system { background: #2a2a3a; }
  .msg.user { background: #1a2a4a; }
  .msg.assistant { background: #1a2a2e; border-color: #333; }
  .msg.tool { background: #2a2a1a; }
}
.role { font-weight: 600; font-size: 0.85rem; text-transform: uppercase; letter-spacing: 0.05em; margin-bottom: 0.5rem; color: #555; }
@media (prefers-color-scheme: dark) { .role { color: #aaa; } }
pre { background: #f4f4f4; padding: 1rem; border-radius: 6px; overflow-x: auto; font-size: 0.9rem; line-height: 1.5; }
@media (prefers-color-scheme: dark) { pre { background: #111; } }
code { font-family: 'SF Mono', 'Fira Code', 'Cascadia Code', 'Consolas', monospace; font-size: 0.9em; }
.reasoning { background: #fdf6e3; border-left: 3px solid #b58900; padding: 0.8rem; margin: 0.5rem 0; border-radius: 4px; font-style: italic; font-size: 0.9rem; color: #5c4b1a; }
@media (prefers-color-scheme: dark) { .reasoning { background: #2a2510; color: #c4b04a; } }
.tool-call { font-size: 0.85rem; color: #666; margin-top: 0.5rem; }
@media (prefers-color-scheme: dark) { .tool-call { color: #999; } }
.footer { margin-top: 3rem; padding-top: 1rem; border-top: 1px solid #ddd; font-size: 0.8rem; color: #999; }
.stats { background: #f0f7f0; padding: 0.8rem 1rem; border-radius: 8px; margin-bottom: 1.5rem; font-size: 0.9rem; color: #2d4a2d; display: inline-block; }
@media (prefers-color-scheme: dark) { .stats { background: #1a2a1a; color: #a0d0a0; } }
</style>
</head>
<body>
`)

	// ── header ──
	if s.Title != "" {
		b.WriteString("<h1>")
		b.WriteString(html.EscapeString(s.Title))
		b.WriteString("</h1>\n")
	}
	b.WriteString(`<p class="meta">`)
	if s.Model != "" {
		b.WriteString("Model: <strong>")
		b.WriteString(html.EscapeString(s.Model))
		b.WriteString("</strong> · ")
	}
	b.WriteString(s.Date.Format("January 2, 2006 15:04"))
	b.WriteString(" · ")
	b.WriteString(fmt.Sprintf("%d messages", len(s.Messages)))
	b.WriteString("</p>\n")

	// ── session stats (when available) ──
	if s.TokensIn+s.TokensOut > 0 || s.Turns > 0 {
		b.WriteString(`<div class="stats">`)
		if s.TokensIn+s.TokensOut > 0 {
			b.WriteString(fmt.Sprintf("Tokens: %d↓ / %d↑", s.TokensIn, s.TokensOut))
			if s.Cost > 0 {
				b.WriteString(fmt.Sprintf(" · Cost: $%.4f", s.Cost))
			}
			b.WriteString(" · ")
		}
		if s.Turns > 0 {
			b.WriteString(fmt.Sprintf("Turns: %d", s.Turns))
		}
		b.WriteString("</div>\n")
	}

	// ── messages ──
	for _, m := range s.Messages {
		writeMessageHTML(&b, m)
	}

	// ── footer ──
	b.WriteString(`<p class="footer">Generated by Reasonix Hermes · `)
	b.WriteString(s.Date.Format(time.RFC3339))
	b.WriteString("</p>\n</body>\n</html>")
	return b.String()
}

func writeMessageHTML(b *strings.Builder, m provider.Message) {
	role := string(m.Role)
	cls := "msg " + role

	b.WriteString("<div class=\"")
	b.WriteString(cls)
	b.WriteString("\">\n")

	// Role label
	label := strings.ToUpper(role[:1]) + role[1:]
	b.WriteString("<div class=\"role\">")
	b.WriteString(label)
	if m.Name != "" {
		b.WriteString(" · ")
		b.WriteString(html.EscapeString(m.Name))
	}
	b.WriteString("</div>\n")

	// Tool calls
	for _, tc := range m.ToolCalls {
		b.WriteString("<div class=\"tool-call\">🔧 <strong>")
		b.WriteString(html.EscapeString(tc.Name))
		b.WriteString("</strong>")
		if tc.Arguments != "" {
			b.WriteString(" <code>")
			b.WriteString(html.EscapeString(truncateStr(tc.Arguments, 120)))
			b.WriteString("</code>")
		}
		b.WriteString("</div>\n")
	}

	// Reasoning content
	if m.ReasoningContent != "" {
		b.WriteString("<details class=\"reasoning\"><summary>💭 Reasoning</summary>")
		b.WriteString(formatContent(m.ReasoningContent))
		b.WriteString("</details>\n")
	}

	// Main content
	if m.Content != "" {
		b.WriteString(formatContent(m.Content))
	}

	b.WriteString("</div>\n")
}

// formatContent converts plain text with code fences and inline backticks into HTML.
func formatContent(text string) string {
	var b strings.Builder
	inCode := false
	lines := strings.Split(text, "\n")

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			if inCode {
				b.WriteString("</code></pre>\n")
				inCode = false
			} else {
				lang := strings.TrimPrefix(trimmed, "```")
				lang = strings.TrimSpace(lang)
				b.WriteString("<pre><code")
				if lang != "" {
					b.WriteString(" class=\"language-")
					b.WriteString(html.EscapeString(lang))
					b.WriteString("\"")
				}
				b.WriteString(">")
				inCode = true
			}
			continue
		}
		if inCode {
			b.WriteString(html.EscapeString(line))
			if i < len(lines)-1 {
				b.WriteString("\n")
			}
		} else {
			b.WriteString("<p>")
			b.WriteString(formatInline(line))
			if line == "" {
				b.WriteString("&nbsp;")
			}
			b.WriteString("</p>\n")
		}
	}
	if inCode {
		b.WriteString("</code></pre>\n")
	}
	return b.String()
}

// formatInline converts inline backticks to <code> spans.
func formatInline(s string) string {
	var b strings.Builder
	inBacktick := false
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if ch == '`' {
			if inBacktick {
				b.WriteString("</code>")
				inBacktick = false
			} else {
				b.WriteString("<code>")
				inBacktick = true
			}
		} else {
			b.WriteString(string(ch))
		}
	}
	// Unclosed backtick — close it.
	if inBacktick {
		b.WriteString("</code>")
	}
	// Escape HTML entities in non-code text.
	raw := b.String()
	// We can't just html.EscapeString because it would double-escape the <code> tags.
	// Instead, we split on <code> / </code> and only escape the text portions.
	return escapeOutsideTags(raw)
}

// escapeOutsideTags escapes HTML only in text portions outside <code>/</code> tags.
func escapeOutsideTags(s string) string {
	var b strings.Builder
	inTag := false
	tagStart := 0
	for i := 0; i < len(s); i++ {
		if !inTag && i+6 <= len(s) && s[i:i+6] == "<code>" {
			b.WriteString(html.EscapeString(s[tagStart:i]))
			b.WriteString("<code>")
			inTag = true
			i += 5
			tagStart = i + 1
			continue
		}
		if inTag && i+7 <= len(s) && s[i:i+7] == "</code>" {
			b.WriteString(html.EscapeString(s[tagStart:i]))
			b.WriteString("</code>")
			inTag = false
			i += 6
			tagStart = i + 1
			continue
		}
	}
	b.WriteString(html.EscapeString(s[tagStart:]))
	return b.String()
}

func truncateStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	// Truncate by runes, not bytes, to avoid breaking multi-byte characters.
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "…"
}
