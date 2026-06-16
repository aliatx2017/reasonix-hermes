package config

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// loadDotEnv loads KEY=value files into the process environment without
// overriding variables that are already set (first file to set a key wins).
// Order: a project ./.env (read-only back-compat, so a manual project override
// takes precedence), then the configured Reasonix credential store (where
// `reasonix setup` writes keys, so they resolve from any directory without ever
// touching a project's own .env), then ~/.env as a legacy fallback. Existing
// environment variables always win over all credential sources.
func loadDotEnv() {
	loadDotEnvForRoot(".")
}

// loadDotEnvForRoot loads a root's .env file (if present) before the home .env
// fallback. When root is "." it behaves like loadDotEnv().
func loadDotEnvForRoot(root string) {
	dotEnvPath := ".env"
	if root != "" && root != "." {
		dotEnvPath = filepath.Join(root, ".env")
	}
	loadDotEnvFile(dotEnvPath)
	loadCredentialStoreForRoot(root)
	if home, err := os.UserHomeDir(); err == nil {
		loadDotEnvFile(filepath.Join(home, ".env"))
	}
}

func legacyCredentialsPaths() []string {
	current := UserCredentialsPath()
	seen := map[string]bool{}
	var paths []string
	add := func(path string) {
		if path == "" {
			return
		}
		path = filepath.Clean(path)
		if current != "" && samePath(path, current) {
			return
		}
		if seen[path] {
			return
		}
		seen[path] = true
		paths = append(paths, path)
	}
	if dir := legacyOSSupportDir(); dir != "" {
		add(filepath.Join(dir, "credentials"))
	}
	for _, cfg := range legacyXDGConfigPaths() {
		add(filepath.Join(filepath.Dir(cfg), "credentials"))
	}
	return paths
}

// loadDotEnvFile reads one .env file (if present) and sets any keys not already
// present in the environment. Lenient, zero-dependency parsing.
func loadDotEnvFile(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.Trim(strings.TrimSpace(val), `"'`)
		if key == "" {
			continue
		}
		if _, exists := os.LookupEnv(key); !exists {
			os.Setenv(key, val)
		}
	}
}
