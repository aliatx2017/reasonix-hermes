//go:build !darwin

package sandbox

import (
	"runtime"
	"testing"
)

func TestCommandNonDarwinBwrap(t *testing.T) {
	if runtime.GOOS == "darwin" {
		t.Skip("testing non-darwin path")
	}
	// With no bwrap on PATH, enforce should fall back to unconfined.
	spec := Spec{Mode: "enforce", WriteRoots: []string{"/tmp"}}
	cmd, wrapped := Command(spec, Shell{Kind: ShellBash, Path: "sh"}, "echo hi")
	if !Available() && wrapped {
		t.Error("enforce without bwrap should NOT wrap, fall back to unconfined")
	}
	if !Available() {
		// No bwrap: command should be plain sh -c "echo hi"
		if len(cmd) != 3 || cmd[0] != "sh" || cmd[1] != "-c" || cmd[2] != "echo hi" {
			t.Errorf("unexpected cmd: %v", cmd)
		}
	}
}

func TestBwrapArgs(t *testing.T) {
	spec := Spec{
		Mode:       "enforce",
		WriteRoots: []string{"/workspace"},
		Network:    false,
	}
	args := bwrapArgs(spec, Shell{Kind: ShellBash, Path: "bash"}, "make build")

	// Should start with bubblewrap isolation flags.
	if args[0] != "--unshare-net" {
		t.Errorf("first arg = %q, want --unshare-net", args[0])
	}
	if args[1] != "--ro-bind" || args[2] != "/" || args[3] != "/" {
		t.Error("missing --ro-bind / /")
	}
	if args[4] != "--dev" || args[5] != "/dev" {
		t.Error("missing --dev /dev")
	}
	if args[6] != "--proc" || args[7] != "/proc" {
		t.Error("missing --proc /proc")
	}

	// Should include the workspace as a write root.
	found := false
	for i, a := range args {
		if a == "--bind" && i+1 < len(args) && args[i+1] == "/workspace" {
			found = true
			break
		}
	}
	if !found {
		t.Error("workspace write root not found in bwrap args")
	}

	// Should end with the shell command.
	end := args[len(args)-3:]
	if end[0] != "bash" || end[1] != "-c" || end[2] != "make build" {
		t.Errorf("shell command tail = %v, want [bash -c 'make build']", end)
	}
}

func TestBwrapArgsNetwork(t *testing.T) {
	spec := Spec{Mode: "enforce", WriteRoots: []string{"/ws"}, Network: true}
	args := bwrapArgs(spec, Shell{Kind: ShellBash, Path: "sh"}, "echo hi")

	// With Network=true, --unshare-net should be absent.
	for _, a := range args {
		if a == "--unshare-net" {
			t.Error("--unshare-net should be absent when Network=true")
		}
	}
}
