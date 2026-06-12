package constitution

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadMissingFile(t *testing.T) {
	dir := t.TempDir()
	_, ok := Load(dir)
	if ok {
		t.Error("expected ok=false for missing constitution.json")
	}
}

func TestLoadValid(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, Dir), 0o755)
	writeFile(t, filepath.Join(dir, Dir, File), `{
		"version": 1,
		"conventions": {"language": "Go", "framework": "Wails"},
		"rules": [
			{"id": "no-global-state", "description": "Avoid package-level mutable state", "severity": "error", "scope": "*.go"},
			{"id": "prefer-context", "description": "Functions that do I/O must accept context.Context as first arg", "severity": "warning"}
		],
		"principles": ["One controller behind every frontend", "Cache-first system prompt"],
		"constraints": ["Never commit secrets", "No external network calls without explicit permission"]
	}`)

	d, ok := Load(dir)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if d.Version != 1 {
		t.Errorf("version = %d, want 1", d.Version)
	}
	if d.Conventions["language"] != "Go" {
		t.Errorf("conventions.language = %v, want Go", d.Conventions["language"])
	}
	if len(d.Rules) != 2 {
		t.Errorf("rules count = %d, want 2", len(d.Rules))
	}
	if len(d.Principles) != 2 {
		t.Errorf("principles count = %d, want 2", len(d.Principles))
	}
	if len(d.Constraints) != 2 {
		t.Errorf("constraints count = %d, want 2", len(d.Constraints))
	}
}

func TestFormatFull(t *testing.T) {
	d := Doc{
		Version: 1,
		Conventions: map[string]any{
			"language": "Go",
			"testing":  "go test",
		},
		Rules: []Rule{
			{ID: "r1", Description: "No global state", Severity: "error", Scope: "*.go"},
		},
		Principles:  []string{"Keep it simple"},
		Constraints: []string{"No external calls without permission"},
	}
	block := Format(d)
	if block == "" {
		t.Fatal("expected non-empty block")
	}
	if !strings.Contains(block, "# Project Constitution") {
		t.Errorf("missing Constitution header")
	}
	if !strings.Contains(block, "**[ERROR]**") {
		t.Errorf("missing error severity marker")
	}
	if !strings.Contains(block, "No global state") {
		t.Errorf("missing rule description")
	}
	if !strings.Contains(block, "Go") {
		t.Errorf("missing convention value")
	}
}

func TestFormatEmpty(t *testing.T) {
	block := Format(Doc{})
	if block != "" {
		t.Errorf("expected empty block, got:\n%s", block)
	}
}

func TestFormatEmptyRulesOnly(t *testing.T) {
	d := Doc{Rules: []Rule{}}
	block := Format(d)
	if block != "" {
		t.Errorf("expected empty block for empty rules slice, got:\n%s", block)
	}
}

func TestFormatDefaults(t *testing.T) {
	// Version 0 → defaults to 1. Severity "" → "info".
	d := Doc{
		Rules: []Rule{
			{ID: "r1", Description: "Something", Severity: ""},
		},
	}
	block := Format(d)
	if !strings.Contains(block, "**[INFO]**") {
		t.Errorf("expected default severity 'info', got:\n%s", block)
	}
}

func TestLoadMalformedJSON(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, Dir), 0o755)
	writeFile(t, filepath.Join(dir, Dir, File), `{invalid}`)
	_, ok := Load(dir)
	if ok {
		t.Error("expected ok=false for malformed JSON")
	}
}

func TestLoadVersionDefaults(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, Dir), 0o755)
	writeFile(t, filepath.Join(dir, Dir, File), `{"rules":[{"id":"r1","description":"x"}]}`)
	d, ok := Load(dir)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if d.Version != 1 {
		t.Errorf("version = %d, want 1 (default)", d.Version)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
