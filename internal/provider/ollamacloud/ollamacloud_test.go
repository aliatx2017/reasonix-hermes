package ollamacloud

import (
	"testing"

	"reasonix/internal/provider"
)

func TestNew(t *testing.T) {
	p, err := New(provider.Config{
		Name:  "ollamacloud",
		Model: "deepseek-v4-flash",
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if p == nil {
		t.Fatal("New returned nil provider")
	}
	if p.Name() != "ollamacloud" {
		t.Errorf("Name = %q, want %q", p.Name(), "ollamacloud")
	}
}

func TestNewCustomBaseURL(t *testing.T) {
	p, err := New(provider.Config{
		Name:    "ollamacloud-custom",
		BaseURL: "https://custom.ollama.example.com/v1",
		Model:   "gemma4:31b",
	})
	if err != nil {
		t.Fatalf("New with custom base URL: %v", err)
	}
	if p == nil {
		t.Fatal("New returned nil provider")
	}
}

func TestNewMissingModel(t *testing.T) {
	_, err := New(provider.Config{
		Name: "ollamacloud",
	})
	if err == nil {
		t.Fatal("expected error for missing model")
	}
}

func TestRegistration(t *testing.T) {
	kinds := provider.Kinds()
	found := false
	for _, k := range kinds {
		if k == "ollamacloud" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("ollamacloud not found in registered kinds: %v", kinds)
	}
}
