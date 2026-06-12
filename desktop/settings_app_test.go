package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"reasonix/internal/config"
	"reasonix/internal/provider"
)

func TestWithFreshSystemPromptReplacesExistingSystemMessage(t *testing.T) {
	msgs := []provider.Message{
		{Role: provider.RoleSystem, Content: "old", ReasoningContent: "stale", ReasoningSignature: "sig", ToolCalls: []provider.ToolCall{{ID: "call", Name: "noop"}}, ToolCallID: "tool", Name: "name"},
		{Role: provider.RoleUser, Content: "hello"},
	}

	got := withFreshSystemPrompt(msgs, "new")
	if got[0].Content != "new" {
		t.Fatalf("system prompt = %q, want new", got[0].Content)
	}
	if got[0].ReasoningContent != "" || got[0].ReasoningSignature != "" || len(got[0].ToolCalls) != 0 || got[0].ToolCallID != "" || got[0].Name != "" {
		t.Fatalf("system metadata should be cleared, got %+v", got[0])
	}
	if got[1].Content != "hello" {
		t.Fatalf("non-system message changed: %+v", got[1])
	}
	if msgs[0].Content != "old" {
		t.Fatalf("input slice was mutated: %+v", msgs[0])
	}
}

func TestWithFreshSystemPromptPrependsMissingSystemMessage(t *testing.T) {
	msgs := []provider.Message{{Role: provider.RoleUser, Content: "hello"}}

	got := withFreshSystemPrompt(msgs, "new")
	if len(got) != 2 || got[0].Role != provider.RoleSystem || got[0].Content != "new" {
		t.Fatalf("expected prepended system prompt, got %+v", got)
	}
	if got[1].Content != "hello" {
		t.Fatalf("existing user message changed: %+v", got[1])
	}
}

func TestProviderViewFromEntry_FiltersNonChatModels(t *testing.T) {
	p := config.ProviderEntry{
		Name: "mimo-api",
		Models: []string{
			"mimo-v2", "mimo-v2-pro",
			"mimo-v2-asr", "mimo-v2-tts",
			"mimo-v2-tts-voiceclone", "mimo-v2-tts-voicedesign",
		},
	}
	view := providerViewFromEntry(p, true, false)
	want := []string{"mimo-v2", "mimo-v2-pro"}
	if !reflect.DeepEqual(view.Models, want) {
		t.Errorf("ProviderView.Models = %v, want %v", view.Models, want)
	}
}

func TestFetchProviderModelsFiltersNonChatModels(t *testing.T) {
	t.Setenv("TEST_PROVIDER_KEY", "test-key")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/models" {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"object": "list",
			"data": []map[string]string{
				{"id": "mimo-v2.5-pro", "object": "model"},
				{"id": "mimo-v2.5-asr", "object": "model"},
				{"id": "mimo-v2.5-tts", "object": "model"},
			},
		})
	}))
	defer srv.Close()

	got, err := NewApp().FetchProviderModels(ProviderView{
		Name:      "mimo-api",
		BaseURL:   srv.URL,
		APIKeyEnv: "TEST_PROVIDER_KEY",
	})
	if err != nil {
		t.Fatalf("FetchProviderModels: %v", err)
	}
	want := []string{"mimo-v2.5-pro"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("FetchProviderModels = %v, want %v", got, want)
	}
}

func TestSaveProviderFiltersNonChatModels(t *testing.T) {
	isolateDesktopUserDirs(t)

	app := NewApp()
	if err := app.SaveProvider(ProviderView{
		Name:      "mimo-api",
		Kind:      "openai",
		BaseURL:   "https://api.xiaomimimo.com/v1",
		Models:    []string{"mimo-v2.5-asr", "mimo-v2.5-pro", "mimo-v2.5-tts"},
		Default:   "mimo-v2.5-asr",
		APIKeyEnv: "MIMO_API_KEY",
	}); err != nil {
		t.Fatalf("SaveProvider: %v", err)
	}

	cfg := config.LoadForEdit(config.UserConfigPath())
	got, ok := cfg.Provider("mimo-api")
	if !ok {
		t.Fatal("saved provider not found")
	}
	want := []string{"mimo-v2.5-pro"}
	if !reflect.DeepEqual(got.ModelList(), want) {
		t.Errorf("saved provider models = %v, want %v", got.ModelList(), want)
	}
	if got.DefaultModel() != "mimo-v2.5-pro" {
		t.Errorf("saved provider default = %q, want mimo-v2.5-pro", got.DefaultModel())
	}
}

func TestSetAgentParamsPersistsStepLimitsToUserConfig(t *testing.T) {
	isolateDesktopUserDirs(t)

	app := NewApp()
	if err := app.SetAgentParams(0.35, 37, 9, "custom system"); err != nil {
		t.Fatalf("SetAgentParams: %v", err)
	}

	view := app.Settings()
	if view.Agent.MaxSteps != 37 || view.Agent.PlannerMaxSteps != 9 {
		t.Fatalf("Settings().Agent = %+v, want maxSteps=37 plannerMaxSteps=9", view.Agent)
	}
	if view.Agent.Temperature != 0.35 || view.Agent.SystemPrompt != "custom system" {
		t.Fatalf("Settings().Agent did not preserve other agent params: %+v", view.Agent)
	}

	cfg := config.LoadForEdit(config.UserConfigPath())
	if cfg.Agent.MaxSteps != 37 || cfg.Agent.PlannerMaxSteps != 9 {
		t.Fatalf("saved config agent steps = max:%d planner:%d, want 37/9", cfg.Agent.MaxSteps, cfg.Agent.PlannerMaxSteps)
	}
	if cfg.Agent.Temperature != 0.35 || cfg.Agent.SystemPrompt != "custom system" {
		t.Fatalf("saved config did not preserve other agent params: %+v", cfg.Agent)
	}
}

func TestSetDesktopCheckUpdatesPersistsToUserConfig(t *testing.T) {
	isolateDesktopUserDirs(t)

	app := NewApp()
	if !app.Settings().CheckUpdates {
		t.Fatal("Settings().CheckUpdates default = false, want true")
	}
	if err := app.SetDesktopCheckUpdates(false); err != nil {
		t.Fatalf("SetDesktopCheckUpdates: %v", err)
	}
	view := app.Settings()
	if view.CheckUpdates {
		t.Fatal("Settings().CheckUpdates = true, want false")
	}
	cfg := config.LoadForEdit(config.UserConfigPath())
	if cfg.Desktop.CheckUpdates == nil || *cfg.Desktop.CheckUpdates {
		t.Fatalf("desktop.check_updates = %+v, want false", cfg.Desktop.CheckUpdates)
	}
	if cfg.DesktopCheckUpdates() {
		t.Fatal("DesktopCheckUpdates() = true, want false")
	}
}

func TestHotbarViewDefaults(t *testing.T) {
	v := defaultHotbarView()
	tests := []struct {
		key  string
		want string
	}{
		{"1", "palette"},
		{"2", "workspace"},
		{"3", "new"},
		{"4", "history"},
		{"5", "dock"},
		{"6", "sidebar"},
		{"7", "settings"},
	}
	for _, tt := range tests {
		got := hotbarField(v, tt.key)
		if got != tt.want {
			t.Errorf("default hotbar key %s = %q, want %q", tt.key, got, tt.want)
		}
	}
}

func TestHotbarViewFromConfig(t *testing.T) {
	tests := []struct {
		name string
		cfg  config.HotbarConfig
		want map[string]string
	}{
		{
			name: "all custom",
			cfg: config.HotbarConfig{
				Key1: "settings", Key2: "dock", Key3: "sidebar",
				Key4: "history", Key5: "new", Key6: "workspace", Key7: "palette",
			},
			want: map[string]string{
				"1": "settings", "2": "dock", "3": "sidebar",
				"4": "history", "5": "new", "6": "workspace", "7": "palette",
			},
		},
		{
			name: "partial with empty",
			cfg:  config.HotbarConfig{Key1: "palette", Key3: "new", Key5: "dock"},
			want: map[string]string{
				"1": "palette", "2": "", "3": "new",
				"4": "", "5": "dock", "6": "", "7": "",
			},
		},
		{
			name: "unbind all",
			cfg:  config.HotbarConfig{Key1: "", Key2: "", Key3: "", Key4: "", Key5: "", Key6: "", Key7: ""},
			want: map[string]string{
				"1": "", "2": "", "3": "", "4": "", "5": "", "6": "", "7": "",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := hotbarView(tt.cfg)
			for k, want := range tt.want {
				got := hotbarField(v, k)
				if got != want {
					t.Errorf("hotbar key %s = %q, want %q", k, got, want)
				}
			}
		})
	}
}

func TestHotbarViewValidation(t *testing.T) {
	// Unknown action names should not panic — they pass through silently
	// (the warning goes to stderr in production; tests don't check stderr).
	cfg := config.HotbarConfig{
		Key1: "garbage",
		Key2: "",
		Key3: "palette",
	}
	v := hotbarView(cfg)
	if v.Key1 != "garbage" {
		t.Errorf("unknown action should pass through, got %q", v.Key1)
	}
	if v.Key2 != "" {
		t.Errorf("empty should stay empty, got %q", v.Key2)
	}
	if v.Key3 != "palette" {
		t.Errorf("known action should pass through, got %q", v.Key3)
	}
}

// hotbarField returns the value for a numeric key from a HotbarView.
func hotbarField(v HotbarView, key string) string {
	switch key {
	case "1":
		return v.Key1
	case "2":
		return v.Key2
	case "3":
		return v.Key3
	case "4":
		return v.Key4
	case "5":
		return v.Key5
	case "6":
		return v.Key6
	case "7":
		return v.Key7
	default:
		return ""
	}
}
