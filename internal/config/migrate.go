package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// legacyConfig is the subset of the v0.x (~/.reasonix/config.json) schema this
// import carries forward. Fields absent here are dropped on purpose: desktop tab
// state is frontend-owned, and skills already live in the shared ~/.reasonix/skills
// root that v1+ also scans, so they need no migration.
type legacyConfig struct {
	APIKey      string                       `json:"apiKey"`
	BaseURL     string                       `json:"baseUrl"`
	Model       string                       `json:"model"`
	Lang        string                       `json:"lang"`
	MCP         []string                     `json:"mcp"` // pre-mcpServers `--mcp`-format strings
	MCPServers  map[string]legacyMCPServer   `json:"mcpServers"`
	MCPEnv      map[string]map[string]string `json:"mcpEnv"`
	MCPDisabled []string                     `json:"mcpDisabled"`
	QQ          legacyQQConfig               `json:"qq"`
}

type legacyMCPServer struct {
	Command   string            `json:"command"`
	Args      []string          `json:"args"`
	Env       map[string]string `json:"env"`
	Transport string            `json:"transport"`
	Type      string            `json:"type"`
	URL       string            `json:"url"`
	Headers   map[string]string `json:"headers"`
	Disabled  bool              `json:"disabled"`
}

type legacyQQConfig struct {
	AppID       string   `json:"appId"`
	AppSecret   string   `json:"appSecret"`
	Sandbox     bool     `json:"sandbox"`
	Enabled     bool     `json:"enabled"`
	OwnerOpenID string   `json:"ownerOpenId"`
	Allowlist   []string `json:"allowlist"`
}

// MigrationResult summarizes a one-time legacy import for the boot-time notice.
type MigrationResult struct {
	From     string
	To       string
	KeyToEnv bool
	Plugins  int
	Warnings []string
}

func (r *MigrationResult) Notice() string {
	var b strings.Builder
	fmt.Fprintf(&b, "migrated your previous configuration: %s → %s", r.From, r.To)
	if r.Plugins > 0 {
		fmt.Fprintf(&b, " (%d MCP server(s))", r.Plugins)
	}
	if r.KeyToEnv {
		b.WriteString("; API key saved to reasonix's credentials store")
	}
	b.WriteString(". The old files were left untouched.")
	for _, w := range r.Warnings {
		b.WriteString("\n  note: " + w)
	}
	return b.String()
}

// MigrateLegacyIfNeeded performs a one-time, non-destructive import of older
// installs into the current user config when the latter does not exist yet. It
// checks v1-era TOML first, then v0.5/v0.x ~/.reasonix/config.json, and never
// modifies or deletes the legacy files. Returns nil when there is nothing to
// migrate, or when the current user config already exists.
func MigrateLegacyIfNeeded() (*MigrationResult, error) {
	credErr := migrateLegacyCredentialsIfNeeded()
	dest := userConfigPath()
	if dest == "" {
		return nil, credErr
	}
	if _, err := os.Stat(dest); err == nil {
		return nil, credErr
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, credErr
	}
	if res, err := migrateLegacyTOMLIfNeeded(dest, home); res != nil || err != nil {
		if err == nil {
			err = credErr
		}
		return res, err
	}
	src := filepath.Join(home, ".reasonix", "config.json")
	data, err := os.ReadFile(src)
	if err != nil {
		return nil, nil
	}
	var legacy legacyConfig
	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF}) // tolerate a UTF-8 BOM (some editors add one)
	if err := json.Unmarshal(data, &legacy); err != nil {
		return nil, fmt.Errorf("parse legacy config %s: %w", src, err)
	}

	cfg := Default()
	res := &MigrationResult{From: src, To: dest}
	if legacy.Lang != "" {
		cfg.Language = legacy.Lang
		_ = cfg.SetDesktopLanguage(legacy.Lang)
	}
	if legacy.Model != "" {
		if entry, ok := cfg.ResolveModel(legacy.Model); ok {
			cfg.DefaultModel = entry.Name + "/" + entry.Model
		} else {
			cfg.DefaultModel = legacy.Model
		}
	}
	migrateLegacyBaseURL(cfg, legacy.BaseURL)
	cfg.Plugins = legacyPlugins(legacy)
	res.Plugins = len(cfg.Plugins)

	var envLines []string
	if key := strings.TrimSpace(legacy.APIKey); key != "" {
		envLines = append(envLines, "DEEPSEEK_API_KEY="+key)
		res.KeyToEnv = true
		if base := strings.TrimSpace(legacy.BaseURL); base != "" && !strings.Contains(base, "deepseek.com") {
			res.Warnings = append(res.Warnings, "your previous base_url was "+base+
				" — it was applied to the built-in DeepSeek providers; verify models if this endpoint is not DeepSeek-compatible")
		}
	}
	if qqSecret := strings.TrimSpace(legacy.QQ.AppSecret); qqSecret != "" {
		envLines = append(envLines, "QQ_BOT_APP_SECRET="+qqSecret)
		res.Warnings = append(res.Warnings, "your previous QQ Bot App Secret was saved to reasonix's credentials store")
	}
	migrateLegacyQQConfig(cfg, legacy.QQ)

	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return nil, fmt.Errorf("create config dir: %w", err)
	}
	if err := cfg.WriteFile(dest); err != nil {
		return nil, fmt.Errorf("write %s: %w", dest, err)
	}
	if len(envLines) > 0 {
		if err := writeCredentialsEnv(home, envLines); err != nil {
			return res, fmt.Errorf("write credentials: %w", err)
		}
	}
	return res, credErr
}

func migrateLegacyCredentialsIfNeeded() error {
	missing := map[string]string{}
	for _, src := range legacyCredentialsPaths() {
		if src == "" {
			continue
		}
		data, err := os.ReadFile(src)
		if err != nil {
			continue
		}
		assignments := parseCredentialLines(strings.Split(string(data), "\n"))
		for key, value := range assignments {
			if _, exists := missing[key]; !exists && !credentialCurrentStoreHasKey(key) {
				missing[key] = value
			}
		}
	}
	if len(missing) == 0 {
		return nil
	}
	_, err := StoreCredentialLines(credentialLines(missing))
	return err
}

func credentialLines(assignments map[string]string) []string {
	keys := make([]string, 0, len(assignments))
	for key := range assignments {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	lines := make([]string, 0, len(keys))
	for _, key := range keys {
		lines = append(lines, key+"="+assignments[key])
	}
	return lines
}

func migrateLegacyQQConfig(cfg *Config, legacy legacyQQConfig) {
	if cfg == nil || !legacyQQConfigured(legacy) {
		return
	}
	cfg.Bot.Enabled = cfg.Bot.Enabled || legacy.Enabled
	cfg.Bot.QQ.Enabled = legacy.Enabled
	cfg.Bot.QQ.AppID = strings.TrimSpace(legacy.AppID)
	cfg.Bot.QQ.AppSecretEnv = "QQ_BOT_APP_SECRET"
	cfg.Bot.QQ.Sandbox = legacy.Sandbox
	cfg.Bot.Allowlist.Enabled = true
	cfg.Bot.Allowlist.QQUsers = mergeUniqueTrimmed(cfg.Bot.Allowlist.QQUsers, legacy.OwnerOpenID)
	cfg.Bot.Allowlist.QQUsers = mergeUniqueTrimmed(cfg.Bot.Allowlist.QQUsers, legacy.Allowlist...)
}

func legacyQQConfigured(legacy legacyQQConfig) bool {
	return legacy.Enabled ||
		strings.TrimSpace(legacy.AppID) != "" ||
		strings.TrimSpace(legacy.AppSecret) != "" ||
		strings.TrimSpace(legacy.OwnerOpenID) != "" ||
		len(legacy.Allowlist) > 0 ||
		legacy.Sandbox
}

func mergeUniqueTrimmed(base []string, values ...string) []string {
	seen := make(map[string]bool, len(base)+len(values))
	out := make([]string, 0, len(base)+len(values))
	for _, value := range append(base, values...) {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func migrateLegacyTOMLIfNeeded(dest, home string) (*MigrationResult, error) {
	for _, src := range legacyTOMLPaths(dest, home) {
		if src == "" || filepath.Clean(src) == filepath.Clean(dest) {
			continue
		}
		if _, err := os.Stat(src); err != nil {
			continue
		}
		cfg := Default()
		if err := mergeFile(cfg, src); err != nil {
			return nil, fmt.Errorf("parse legacy config %s: %w", src, err)
		}
		cfg.ConfigVersion = Default().ConfigVersion
		if strings.TrimSpace(cfg.Desktop.CloseBehavior) == "" && strings.TrimSpace(cfg.UI.CloseBehavior) != "" {
			cfg.Desktop.CloseBehavior = cfg.DesktopCloseBehavior()
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return nil, fmt.Errorf("create config dir: %w", err)
		}
		if err := cfg.WriteFile(dest); err != nil {
			return nil, fmt.Errorf("write %s: %w", dest, err)
		}
		return &MigrationResult{From: src, To: dest, Plugins: len(cfg.Plugins)}, nil
	}
	return nil, nil
}

func legacyTOMLPaths(dest, home string) []string {
	seen := map[string]bool{}
	var paths []string
	add := func(path string) {
		if path == "" {
			return
		}
		path = filepath.Clean(path)
		if seen[path] {
			return
		}
		seen[path] = true
		paths = append(paths, path)
	}
	if legacy := legacyUserConfigPath(); legacy != "" {
		add(legacy)
	}
	for _, legacy := range legacyXDGConfigPaths() {
		add(legacy)
		add(filepath.Join(filepath.Dir(legacy), "reasonix.toml"))
	}
	add(filepath.Join(filepath.Dir(dest), "reasonix.toml"))
	if home != "" {
		add(filepath.Join(home, ".reasonix", "reasonix.toml"))
	}
	return paths
}

func migrateLegacyBaseURL(cfg *Config, baseURL string) {
	baseURL = strings.TrimSpace(baseURL)
	if cfg == nil || baseURL == "" {
		return
	}
	for i := range cfg.Providers {
		if cfg.Providers[i].APIKeyEnv == "DEEPSEEK_API_KEY" {
			cfg.Providers[i].BaseURL = baseURL
		}
	}
}

func legacyPlugins(legacy legacyConfig) []PluginEntry {
	disabled := make(map[string]bool, len(legacy.MCPDisabled))
	for _, n := range legacy.MCPDisabled {
		disabled[n] = true
	}
	var out []PluginEntry
	index := make(map[string]int)
	add := func(pe PluginEntry, off bool) {
		if off {
			v := false
			pe.AutoStart = &v
		}
		pe, _ = NormalizePluginCommandLine(pe)
		if j, dup := index[pe.Name]; dup {
			out[j] = pe // mcpServers overrides the `mcp` list on a name collision, matching v0.x
			return
		}
		index[pe.Name] = len(out)
		out = append(out, pe)
	}
	for i, raw := range legacy.MCP {
		pe, ok := parseLegacyMCPSpec(raw)
		if !ok {
			continue
		}
		if pe.Name == "" {
			pe.Name = anonymousMCPName(i)
		} else if pe.Command != "" {
			pe.Env = mergeEnv(nil, legacy.MCPEnv[pe.Name])
		}
		add(pe, disabled[pe.Name])
	}
	names := make([]string, 0, len(legacy.MCPServers))
	for n := range legacy.MCPServers {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, name := range names {
		s := legacy.MCPServers[name]
		pe := PluginEntry{
			Name:    name,
			Type:    normalizeTransport(firstNonEmpty(s.Type, s.Transport)),
			Command: s.Command,
			Args:    s.Args,
			Env:     mergeEnv(s.Env, legacy.MCPEnv[name]),
			URL:     s.URL,
			Headers: s.Headers,
		}
		add(pe, s.Disabled || disabled[name])
	}
	return out
}

// normalizeTransport maps the v0.x transport names to v1+ plugin types. stdio is
// the default, so it returns "" (RenderTOML then omits the field).
func normalizeTransport(t string) string {
	switch strings.ToLower(strings.TrimSpace(t)) {
	case "http", "streamable-http":
		return "http"
	case "sse":
		return "sse"
	default:
		return ""
	}
}

func firstNonEmpty(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}

// mergeEnv overlays the per-server env map onto the spec's own env (overlay wins,
// matching v0.x mcpEnv precedence). Returns nil when both are empty.
func mergeEnv(base, overlay map[string]string) map[string]string {
	if len(base) == 0 && len(overlay) == 0 {
		return nil
	}
	out := make(map[string]string, len(base)+len(overlay))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range overlay {
		out[k] = v
	}
	return out
}

// writeCredentialsEnv merges lines into the configured global credential store
// and pins them into the current process env so the just-built session resolves
// the key without a restart. Falls back to ~/.env only when Reasonix home can't
// be resolved — never a project .env, so a migration keeps secrets out of the
// user's project tree.
func writeCredentialsEnv(home string, lines []string) error {
	if _, err := StoreCredentialLines(lines); err != nil {
		if UserCredentialsPath() == "" && home != "" {
			return os.WriteFile(filepath.Join(home, ".env"), []byte(strings.Join(lines, "\n")+"\n"), 0o600)
		}
		return err
	}
	return nil
}
