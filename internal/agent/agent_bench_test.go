package agent

import (
	"strings"
	"testing"

	"reasonix/internal/provider"
)

// ── Utility function benchmarks ──────────────────────────────────────────────

func BenchmarkHasImagesNoImages(b *testing.B) {
	msgs := make([]provider.Message, 100)
	for i := range msgs {
		msgs[i] = provider.Message{Role: provider.RoleUser, Content: "hello"}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hasImages(msgs)
	}
}

func BenchmarkHasImagesOneImage(b *testing.B) {
	msgs := make([]provider.Message, 100)
	for i := range msgs {
		msgs[i] = provider.Message{Role: provider.RoleUser, Content: "hello"}
	}
	msgs[50] = provider.Message{
		Role:    provider.RoleUser,
		Content: "describe this",
		Images:  []string{"data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8/5+hHgAHggJ/PchI7wAAAABJRU5ErkJggg=="},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hasImages(msgs)
	}
}

func BenchmarkSuccessfulToolCallIDs(b *testing.B) {
	msgs := make([]provider.Message, 200)
	for i := 0; i < 200; i += 2 {
		msgs[i] = provider.Message{Role: provider.RoleUser, Content: "x"}
		msgs[i+1] = provider.Message{
			Role: provider.RoleAssistant,
			ToolCalls: []provider.ToolCall{
				{ID: "call_read", Name: "read_file"},
				{ID: "call_write", Name: "write_file"},
				{ID: "call_bash", Name: "bash"},
			},
		}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		successfulToolCallIDs(msgs)
	}
}

func BenchmarkEmptyFinalNotice(b *testing.B) {
	u := &provider.Usage{PromptTokens: 5000, CompletionTokens: 500, CacheHitTokens: 2000}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		emptyFinalNotice("deepseek", u, 1000)
	}
}

func BenchmarkWithReasoningLanguage(b *testing.B) {
	a := &Agent{}
	input := "Please write a function in Go. " + strings.Repeat("context ", 20)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		a.withReasoningLanguage(input)
	}
}
