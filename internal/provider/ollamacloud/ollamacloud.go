// Package ollamacloud implements the Ollama Cloud API provider. Ollama Cloud
// exposes an OpenAI-compatible /chat/completions endpoint at ollama.com/v1,
// so this provider delegates to the openai provider with Ollama Cloud defaults
// and plain OpenAI reasoning protocol (no DeepSeek/MiniMax-specific handling).
//
// It self-registers under the "ollamacloud" kind, so a [[providers]] entry
// with kind = "ollamacloud" resolves here. The default base URL is
// https://ollama.com/v1. Model names support colon-delimited tags
// (e.g. "gemma4:31b", "deepseek-v4-pro").
package ollamacloud

import (
	"strings"

	"reasonix/internal/provider"

	// Ensure openai is registered so provider.New("openai", ...) can resolve it.
	_ "reasonix/internal/provider/openai"
)

const defaultBaseURL = "https://ollama.com/v1"

func init() {
	provider.Register("ollamacloud", New)
}

// New builds an Ollama Cloud provider from a resolved config. It delegates to
// the openai provider with Ollama Cloud defaults: base URL set to ollama.com/v1,
// reasoning protocol forced to "none" (skip DeepSeek/MiniMax detection), and
// reasoning effort defaulted to "low" for cheapest thinking.
func New(cfg provider.Config) (provider.Provider, error) {
	if cfg.BaseURL == "" {
		cfg.BaseURL = defaultBaseURL
	}
	if cfg.Extra == nil {
		cfg.Extra = map[string]any{}
	}
	// Always force "none" protocol — boot may set an empty one which still
	// triggers IsDeepSeek detection in the openai provider.
	cfg.Extra["reasoning_protocol"] = "none"
	// Default to "low" effort (cheapest) unless explicitly set.
	e, _ := cfg.Extra["effort"].(string)
	if strings.TrimSpace(e) == "" {
		cfg.Extra["effort"] = "low"
	}
	return provider.New("openai", cfg)
}
