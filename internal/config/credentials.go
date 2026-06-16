package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/zalando/go-keyring"

	"reasonix/internal/fileutil"
)

const (
	CredentialsStoreAuto    = "auto"
	CredentialsStoreKeyring = "keyring"
	CredentialsStoreFile    = "file"

	credentialsKeyringService = "reasonix"
)

func normalizeCredentialsStore(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case CredentialsStoreKeyring:
		return CredentialsStoreKeyring
	case CredentialsStoreFile:
		return CredentialsStoreFile
	default:
		return CredentialsStoreAuto
	}
}

func credentialsStoreMode() string {
	if mode := strings.TrimSpace(os.Getenv("REASONIX_CREDENTIALS_STORE")); mode != "" {
		return normalizeCredentialsStore(mode)
	}
	var partial struct {
		CredentialsStore string `toml:"credentials_store"`
	}
	if path := userConfigLoadPath(); path != "" {
		_, _ = toml.DecodeFile(path, &partial)
	}
	return normalizeCredentialsStore(partial.CredentialsStore)
}

func credentialEnvNamesForRoot(root string) []string {
	root = resolveRoot(root)
	cfg := Default()

	projectTOML := "reasonix.toml"
	if root != "." {
		projectTOML = filepath.Join(root, "reasonix.toml")
	}
	if uc := userConfigLoadPath(); uc != "" {
		_ = mergeFile(cfg, uc)
	}
	_ = mergeFile(cfg, projectTOML)
	var tomlSources []string
	if uc := userConfigLoadPath(); uc != "" {
		tomlSources = append(tomlSources, uc)
	}
	tomlSources = append(tomlSources, projectTOML)
	if providers, _, ok, err := mergeTOMLProviders(tomlSources); err == nil && ok {
		cfg.Providers = providers
	}

	seen := map[string]bool{}
	var out []string
	add := func(name string) {
		name = strings.TrimSpace(name)
		if name == "" || seen[name] {
			return
		}
		seen[name] = true
		out = append(out, name)
	}
	for _, p := range cfg.Providers {
		add(p.APIKeyEnv)
	}
	add(cfg.Bot.QQ.AppSecretEnv)
	add(cfg.Bot.Feishu.AppSecretEnv)
	add(cfg.Bot.Weixin.TokenEnv)
	for _, conn := range cfg.Bot.Connections {
		add(conn.Credential.AppSecretEnv)
		add(conn.Credential.TokenEnv)
	}
	sort.Strings(out)
	return out
}

func loadCredentialStoreForRoot(root string) {
	names := credentialEnvNamesForRoot(root)
	if len(names) == 0 {
		return
	}
	mode := credentialsStoreMode()
	if mode == CredentialsStoreAuto || mode == CredentialsStoreKeyring {
		for _, name := range names {
			if _, exists := os.LookupEnv(name); exists {
				continue
			}
			value, err := keyring.Get(credentialsKeyringService, name)
			if err == nil && value != "" {
				_ = os.Setenv(name, value)
			}
		}
	}
	if mode == CredentialsStoreAuto || mode == CredentialsStoreFile {
		if p := UserCredentialsPath(); p != "" {
			loadDotEnvFile(p)
		}
		for _, p := range legacyCredentialsPaths() {
			loadDotEnvFile(p)
		}
	}
}

// StoreCredentialLines stores KEY=value assignments in the configured user
// credential store and pins them into the current process environment.
func StoreCredentialLines(lines []string) (string, error) {
	assignments := parseCredentialLines(lines)
	if len(assignments) == 0 {
		return CredentialsTargetDescription(), nil
	}
	mode := credentialsStoreMode()
	if mode == CredentialsStoreAuto || mode == CredentialsStoreKeyring {
		if err := storeCredentialsInKeyring(assignments); err == nil {
			pinCredentialAssignments(assignments)
			return "system credential store", nil
		} else if mode == CredentialsStoreKeyring {
			return "", err
		}
	}
	if err := storeCredentialsInFile(UserCredentialsPath(), assignments); err != nil {
		return "", err
	}
	pinCredentialAssignments(assignments)
	return UserCredentialsPath(), nil
}

func SetCredential(key, value string) (string, error) {
	key = strings.TrimSpace(key)
	if !isCredentialKey(key) {
		return "", fmt.Errorf("invalid credential key %q", key)
	}
	if strings.ContainsAny(value, "\r\n") {
		return "", fmt.Errorf("credential value for %s contains a newline", key)
	}
	return StoreCredentialLines([]string{key + "=" + value})
}

func RemoveCredential(key string) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil
	}
	mode := credentialsStoreMode()
	if mode == CredentialsStoreAuto || mode == CredentialsStoreKeyring {
		err := keyring.Delete(credentialsKeyringService, key)
		if err != nil && !errors.Is(err, keyring.ErrNotFound) && mode == CredentialsStoreKeyring {
			return err
		}
	}
	if mode == CredentialsStoreAuto || mode == CredentialsStoreFile {
		if path := UserCredentialsPath(); path != "" {
			if err := removeCredentialFromFile(path, key); err != nil {
				return err
			}
		}
	}
	return os.Unsetenv(key)
}

func CredentialIsSet(key string) bool {
	key = strings.TrimSpace(key)
	if key == "" {
		return false
	}
	if os.Getenv(key) != "" {
		return true
	}
	return CredentialStored(key)
}

func CredentialStored(key string) bool {
	key = strings.TrimSpace(key)
	if key == "" {
		return false
	}
	mode := credentialsStoreMode()
	if mode == CredentialsStoreAuto || mode == CredentialsStoreKeyring {
		if value, err := keyring.Get(credentialsKeyringService, key); err == nil && value != "" {
			return true
		}
	}
	if mode == CredentialsStoreAuto || mode == CredentialsStoreFile {
		for _, path := range append([]string{UserCredentialsPath()}, legacyCredentialsPaths()...) {
			if envFileHasKey(path, key) {
				return true
			}
		}
	}
	return false
}

func credentialCurrentStoreHasKey(key string) bool {
	key = strings.TrimSpace(key)
	if key == "" {
		return false
	}
	mode := credentialsStoreMode()
	if mode == CredentialsStoreAuto || mode == CredentialsStoreKeyring {
		if value, err := keyring.Get(credentialsKeyringService, key); err == nil && value != "" {
			return true
		}
	}
	if mode == CredentialsStoreAuto || mode == CredentialsStoreFile {
		return envFileHasKey(UserCredentialsPath(), key)
	}
	return false
}

func CredentialsTargetDescription() string {
	switch credentialsStoreMode() {
	case CredentialsStoreKeyring:
		return "system credential store"
	case CredentialsStoreFile:
		return UserCredentialsPath()
	default:
		return "system credential store or " + UserCredentialsPath()
	}
}

func parseCredentialLines(lines []string) map[string]string {
	out := map[string]string{}
	for _, raw := range lines {
		line := strings.TrimPrefix(strings.TrimSpace(raw), "export ")
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		if !ok || !isCredentialKey(key) || strings.ContainsAny(value, "\r\n") {
			continue
		}
		out[key] = value
	}
	return out
}

func pinCredentialAssignments(assignments map[string]string) {
	for key, value := range assignments {
		_ = os.Setenv(key, value)
	}
}

func storeCredentialsInKeyring(assignments map[string]string) error {
	for key, value := range assignments {
		if err := keyring.Set(credentialsKeyringService, key, value); err != nil {
			return err
		}
	}
	return nil
}

func storeCredentialsInFile(path string, assignments map[string]string) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("credentials store unavailable")
	}
	lines, err := readCredentialFileLines(path)
	if err != nil {
		return err
	}
	replaced := map[string]bool{}
	for i, line := range lines {
		key, ok := credentialLineKey(line)
		if !ok {
			continue
		}
		if value, hit := assignments[key]; hit {
			lines[i] = key + "=" + value
			replaced[key] = true
		}
	}
	keys := make([]string, 0, len(assignments))
	for key := range assignments {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if !replaced[key] {
			lines = append(lines, key+"="+assignments[key])
		}
	}
	return writeCredentialFileLines(path, lines)
}

func removeCredentialFromFile(path, key string) error {
	lines, err := readCredentialFileLines(path)
	if err != nil {
		return err
	}
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if k, ok := credentialLineKey(line); ok && k == key {
			continue
		}
		out = append(out, line)
	}
	return writeCredentialFileLines(path, out)
}

func readCredentialFileLines(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	text := strings.TrimRight(string(data), "\n")
	if text == "" {
		return nil, nil
	}
	return strings.Split(text, "\n"), nil
}

func writeCredentialFileLines(path string, lines []string) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("credentials store unavailable")
	}
	out := ""
	if len(lines) > 0 {
		out = strings.Join(lines, "\n") + "\n"
	}
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	tmp, err := os.CreateTemp(dir, "credentials.*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	if _, err := tmp.WriteString(out); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}
	if err := os.Chmod(tmpPath, 0o600); err != nil {
		os.Remove(tmpPath)
		return err
	}
	if err := fileutil.ReplaceFile(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return err
	}
	return nil
}

func credentialLineKey(line string) (string, bool) {
	trimmed := strings.TrimPrefix(strings.TrimSpace(line), "export ")
	if trimmed == "" || strings.HasPrefix(trimmed, "#") {
		return "", false
	}
	key, _, ok := strings.Cut(trimmed, "=")
	key = strings.TrimSpace(key)
	return key, ok && isCredentialKey(key)
}

func isCredentialKey(key string) bool {
	if key == "" {
		return false
	}
	for i, r := range key {
		if r == '_' || r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || i > 0 && r >= '0' && r <= '9' {
			continue
		}
		return false
	}
	return true
}

func envFileHasKey(path, key string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	lines, err := readCredentialFileLines(path)
	if err != nil {
		return false
	}
	for _, line := range lines {
		if k, ok := credentialLineKey(line); ok && k == key {
			return true
		}
	}
	return false
}
