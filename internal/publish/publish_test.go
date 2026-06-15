package publish

import (
	"strings"
	"testing"
	"time"

	"reasonix/internal/provider"
)

func TestToJSON(t *testing.T) {
	t.Parallel()
	s := Session{
		Title: "Test Session",
		Model: "deepseek-flash",
		Date:  time.Date(2026, 6, 13, 14, 30, 0, 0, time.UTC),
		Messages: []provider.Message{
			{Role: "user", Content: "hello"},
			{Role: "assistant", Content: "hi there!"},
		},
	}
	out, err := ToJSON(s)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), `"role": "user"`) {
		t.Error("JSON should contain role field")
	}
	if !strings.Contains(string(out), `"content": "hello"`) {
		t.Error("JSON should contain content")
	}
}

func TestToHTMLBasic(t *testing.T) {
	t.Parallel()
	s := Session{
		Title: "Test",
		Model: "deepseek-flash",
		Date:  time.Date(2026, 6, 13, 14, 30, 0, 0, time.UTC),
		Messages: []provider.Message{
			{Role: "user", Content: "hello world"},
			{Role: "assistant", Content: "Here is a response with `code`."},
		},
	}
	html := ToHTML(s)
	if !strings.Contains(html, "<title>Test</title>") {
		t.Error("HTML should have title")
	}
	if !strings.Contains(html, "deepseek-flash") {
		t.Error("HTML should mention model")
	}
	if !strings.Contains(html, "hello world") {
		t.Error("HTML should contain user message")
	}
	if !strings.Contains(html, "msg user") {
		t.Error("HTML should have user message class")
	}
	if !strings.Contains(html, "msg assistant") {
		t.Error("HTML should have assistant message class")
	}
	if !strings.Contains(html, "<code>") {
		t.Error("HTML should wrap inline code")
	}
}

func TestToHTMLCodeBlocks(t *testing.T) {
	t.Parallel()
	s := Session{
		Title: "Code Test",
		Date:  time.Now(),
		Messages: []provider.Message{
			{Role: "assistant", Content: "Here:\n```go\nfunc main() {\n\tfmt.Println(\"hi\")\n}\n```\nDone."},
		},
	}
	html := ToHTML(s)
	if !strings.Contains(html, "language-go") {
		t.Error("HTML should have language class on code block")
	}
	if !strings.Contains(html, "fmt.Println") {
		t.Error("HTML should contain code content")
	}
	if !strings.Contains(html, "<pre><code") {
		t.Error("HTML should wrap code blocks in pre/code")
	}
}

func TestToHTMLReasoningContent(t *testing.T) {
	t.Parallel()
	s := Session{
		Date: time.Now(),
		Messages: []provider.Message{
			{Role: "assistant", ReasoningContent: "I should use a loop here", Content: "ok"},
		},
	}
	html := ToHTML(s)
	if !strings.Contains(html, "💭 Reasoning") {
		t.Error("HTML should show reasoning toggle")
	}
	if !strings.Contains(html, "I should use a loop") {
		t.Error("HTML should contain reasoning text")
	}
}

func TestToHTMLToolCalls(t *testing.T) {
	t.Parallel()
	s := Session{
		Date: time.Now(),
		Messages: []provider.Message{
			{Role: "assistant", Content: "Let me check",
				ToolCalls: []provider.ToolCall{
					{ID: "1", Name: "read_file", Arguments: `{"path":"main.go"}`},
				},
			},
		},
	}
	html := ToHTML(s)
	if !strings.Contains(html, "read_file") {
		t.Error("HTML should show tool call name")
	}
}

func TestToHTMLEmptyTitle(t *testing.T) {
	t.Parallel()
	s := Session{
		Date:     time.Now(),
		Messages: []provider.Message{},
	}
	html := ToHTML(s)
	if !strings.Contains(html, "Session Transcript") {
		t.Error("HTML should have fallback title")
	}
}

func TestToHTMLSystemMessage(t *testing.T) {
	t.Parallel()
	s := Session{
		Date: time.Now(),
		Messages: []provider.Message{
			{Role: "system", Content: "You are a helpful assistant."},
		},
	}
	html := ToHTML(s)
	if !strings.Contains(html, "msg system") {
		t.Error("HTML should have system message class")
	}
}

func TestToHTMLXSS(t *testing.T) {
	t.Parallel()
	s := Session{
		Date: time.Now(),
		Messages: []provider.Message{
			{Role: "user", Content: `<script>alert("xss")</script>`},
		},
	}
	html := ToHTML(s)
	if strings.Contains(html, "<script>") {
		t.Error("HTML should escape script tags")
	}
	if !strings.Contains(html, "&lt;script&gt;") {
		t.Error("HTML should contain escaped script tags")
	}
}

func TestTruncateStr(t *testing.T) {
	t.Parallel()
	if s := truncateStr("hello", 3); s != "hel…" {
		t.Errorf("truncateStr(hello,3) = %q", s)
	}
	if s := truncateStr("hi", 10); s != "hi" {
		t.Errorf("truncateStr(hi,10) = %q", s)
	}
}
