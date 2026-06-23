//go:build !windows

package proc

import (
	"os/exec"
	"syscall"
	"testing"
)

func TestLowPriorityStarted(t *testing.T) {
	// LowPriorityStarted renices a running process to priority 10.
	cmd := exec.Command("sleep", "1")
	if err := cmd.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer cmd.Wait()

	LowPriorityStarted(cmd) // should not panic
}

func TestLowPriorityStartedNilProcess(t *testing.T) {
	// cmd.Process is nil before Start — should not panic.
	cmd := exec.Command("true")
	LowPriorityStarted(cmd) // should be a no-op
}

func TestLowPriorityStartedAfterWait(t *testing.T) {
	// cmd.Process is set but the process has exited — should not panic.
	cmd := exec.Command("true")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Run: %v", err)
	}
	// Process is still non-nil after Run, but the process is dead.
	// Setpriority will fail but that's fine — we don't check the error.
	LowPriorityStarted(cmd)
}

func TestPrepareShellPATHProbe(t *testing.T) {
	cmd := exec.Command("true")
	PrepareShellPATHProbe(cmd)

	if cmd.SysProcAttr == nil {
		t.Fatal("SysProcAttr should be set")
	}
	if !cmd.SysProcAttr.Setsid {
		t.Error("Setsid should be true")
	}
}

func TestPrepareShellPATHProbeExistingAttr(t *testing.T) {
	cmd := &exec.Cmd{
		SysProcAttr: &syscall.SysProcAttr{},
	}
	PrepareShellPATHProbe(cmd)

	if !cmd.SysProcAttr.Setsid {
		t.Error("Setsid should be true even with existing SysProcAttr")
	}
}

func TestSetProcessGroupKillExistingAttr(t *testing.T) {
	// Already covered by other tests; ensure existing attrs are preserved.
	cmd := &exec.Cmd{
		SysProcAttr: &syscall.SysProcAttr{Noctty: true},
	}
	SetProcessGroupKill(cmd)

	if !cmd.SysProcAttr.Setpgid {
		t.Error("Setpgid should be true")
	}
	if !cmd.SysProcAttr.Noctty {
		t.Error("existing Noctty should be preserved")
	}
}

func TestKillTreeNilCmd(t *testing.T) {
	// Must not panic.
	KillTree(nil)
}

func TestKillTreeNilProcess(t *testing.T) {
	// Must not panic — cmd is non-nil but Process is nil (not started).
	cmd := exec.Command("true")
	KillTree(cmd)
}
