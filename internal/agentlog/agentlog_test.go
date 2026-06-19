package agentlog

import (
	"os"
	"path/filepath"
	"testing"

	"reasonix/internal/config"
)

func TestInit(t *testing.T) {
	tmp := t.TempDir()
	logPath := filepath.Join(tmp, "agent.log")
	t.Setenv("AGENT_LOG", logPath)

	Init(config.AgentLogConfig{})

	// Verify the file was created (may be empty — first log entry comes from boot).
	if _, err := os.Stat(logPath); err != nil {
		t.Fatalf("agent.log not created: %v", err)
	}
}

func TestInitDisabled(t *testing.T) {
	tmp := t.TempDir()
	logPath := filepath.Join(tmp, "agent.log")
	t.Setenv("AGENT_LOG", logPath)

	// Zero-value AgentLogConfig means "not configured" → logging is enabled by
	// default. To disable logging, set a non-zero MaxBackups with Enabled=false.
	Init(config.AgentLogConfig{Enabled: false, MaxBackups: 1})

	// File must NOT be created when explicitly disabled (non-zero config).
	if _, err := os.Stat(logPath); err == nil {
		t.Fatalf("agent.log was created despite enabled=false with explicit config")
	}
}

func TestRotateLogUnderThreshold(t *testing.T) {
	tmp := t.TempDir()
	logPath := filepath.Join(tmp, "agent.log")

	// Write a small file.
	if err := os.WriteFile(logPath, []byte("small file"), 0o600); err != nil {
		t.Fatal(err)
	}

	// Threshold is 1 MB — file is well under.
	rotateLog(logPath, 1<<20, 3)

	// File should still exist.
	if _, err := os.Stat(logPath); err != nil {
		t.Fatalf("agent.log disappeared: %v", err)
	}
	// No backups should exist.
	for i := 1; i <= 3; i++ {
		backup := logPath + "." + string(rune('0'+i))
		if _, err := os.Stat(backup); err == nil {
			t.Fatalf("unexpected backup: %s", backup)
		}
	}
}

func TestRotateLogOverThreshold(t *testing.T) {
	tmp := t.TempDir()
	logPath := filepath.Join(tmp, "agent.log")

	// Write a file that exceeds the threshold.
	payload := make([]byte, 100)
	if err := os.WriteFile(logPath, payload, 0o600); err != nil {
		t.Fatal(err)
	}

	// Threshold is 50 bytes — file exceeds it.
	rotateLog(logPath, 50, 3)

	// Original should be gone (moved to .1).
	if _, err := os.Stat(logPath); err == nil {
		t.Fatal("agent.log should have been rotated away")
	}

	// Backup .1 should exist with the old content.
	backup1 := logPath + ".1"
	data, err := os.ReadFile(backup1)
	if err != nil {
		t.Fatalf("backup .1 not found: %v", err)
	}
	if len(data) != 100 {
		t.Fatalf("backup .1 size = %d, want 100", len(data))
	}

	// Higher backups should not exist.
	for i := 2; i <= 3; i++ {
		backup := logPath + "." + string(rune('0'+i))
		if _, err := os.Stat(backup); err == nil {
			t.Fatalf("unexpected backup: %s", backup)
		}
	}
}

func TestRotateLogShiftBackups(t *testing.T) {
	tmp := t.TempDir()
	logPath := filepath.Join(tmp, "agent.log")

	// Create existing backups.
	os.WriteFile(logPath+".1", []byte("first"), 0o600)
	os.WriteFile(logPath+".2", []byte("second"), 0o600)

	// Current file exceeds threshold.
	if err := os.WriteFile(logPath, []byte("current"), 0o600); err != nil {
		t.Fatal(err)
	}

	rotateLog(logPath, 1, 3)

	// Original → .1
	data, _ := os.ReadFile(logPath + ".1")
	if string(data) != "current" {
		t.Fatalf(".1 = %q, want %q", data, "current")
	}

	// Old .1 → .2
	data, _ = os.ReadFile(logPath + ".2")
	if string(data) != "first" {
		t.Fatalf(".2 = %q, want %q", data, "first")
	}

	// Old .2 → .3
	data, _ = os.ReadFile(logPath + ".3")
	if string(data) != "second" {
		t.Fatalf(".3 = %q, want %q", data, "second")
	}
}

func TestRotateLogDeletesOldest(t *testing.T) {
	tmp := t.TempDir()
	logPath := filepath.Join(tmp, "agent.log")

	// Fill all backup slots.
	os.WriteFile(logPath+".1", []byte("1"), 0o600)
	os.WriteFile(logPath+".2", []byte("2"), 0o600)
	os.WriteFile(logPath+".3", []byte("3"), 0o600)

	// Current exceeds threshold.
	if err := os.WriteFile(logPath, []byte("current"), 0o600); err != nil {
		t.Fatal(err)
	}

	rotateLog(logPath, 1, 3)

	// Old .3 ("3") is deleted, then old .2 ("2") shifts into .3.
	data, err := os.ReadFile(logPath + ".3")
	if err != nil {
		t.Fatalf(".3 missing after rotation: %v", err)
	}
	if string(data) != "2" {
		t.Fatalf(".3 = %q, want %q", data, "2")
	}
}

func TestRotateLogMissingFile(t *testing.T) {
	tmp := t.TempDir()
	logPath := filepath.Join(tmp, "nonexistent", "agent.log")

	// Should not panic.
	rotateLog(logPath, 1, 5)
}
