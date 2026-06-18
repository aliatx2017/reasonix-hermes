package agentlog

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInit(t *testing.T) {
	tmp := t.TempDir()
	logPath := filepath.Join(tmp, "agent.log")
	t.Setenv("AGENT_LOG", logPath)

	Init()

	// Verify the file was created (may be empty — first log entry comes from boot).
	if _, err := os.Stat(logPath); err != nil {
		t.Fatalf("agent.log not created: %v", err)
	}
}
