// Package sandbox wraps a shell command in an OS-level jail so the model's
// `bash` calls are confined: it may read freely but write only inside the
// workspace (plus temp and toolchain caches) and reach the network only when
// allowed. This is the *enforcement* layer beneath the permission rules
// (*policy*): a permitted command still cannot escape the box.
//
// Only macOS (Seatbelt via sandbox-exec) is implemented; on every other OS, or
// when the OS tooling is missing, Command falls back to running the command
// unwrapped (see Available). Confining the in-process file-writer built-ins is
// handled separately, in package tool/builtin.
package sandbox

import (
	"os"
	"path/filepath"
	"runtime"
)

// Spec describes how to confine one command. The zero value (Mode == "") does
// not enforce, so an unconfigured caller runs commands unchanged.
type Spec struct {
	// Mode is "enforce" to wrap the command, "remote" to send to a remote
	// sandbox API (e.g. OpenSandbox), anything else (incl. "off" and "")
	// to run it unwrapped.
	Mode string
	// WriteRoots are directories the command may write to (the workspace root
	// plus any configured extras). Temp dirs and common toolchain caches are
	// added automatically so builds and package managers keep working.
	WriteRoots []string
	// Network allows network egress from inside the sandbox. Off blocks it so a
	// command cannot exfiltrate or fetch; many dev commands (module/package
	// downloads) need it, so it defaults on at the config layer.
	Network bool
	// RemoteURL is the API endpoint for a remote sandbox backend (OpenSandbox).
	// Only used when Mode == "remote".
	RemoteURL string
	// RemoteToken is the bearer auth token for the remote sandbox API.
	RemoteToken string
}

// enforce reports whether the spec asks for local confinement.
func (s Spec) enforce() bool { return s.Mode == "enforce" }

// remote reports whether the spec asks for remote execution.
func (s Spec) remote() bool { return s.Mode == "remote" }

// Run executes a shell command through the sandbox backend specified by the Spec.
// When Mode is "remote", the command is sent to the remote sandbox API and the
// combined output (stdout+stderr) is returned. For local backends ("enforce" or
// off), Run returns ("", false, nil) — the caller executes the command via the
// argv returned by Command() as usual.
func Run(spec Spec, command string) (output string, handled bool, err error) {
	if spec.remote() {
		output, err := commandRemote(spec, command)
		return output, true, err
	}
	return "", false, nil
}

// writeAllowDirs is the deduplicated, symlink-resolved set of directories the
// sandbox permits writes to: the caller's roots plus temp dirs, /dev, and the
// common toolchain caches. Used by both macOS Seatbelt and Linux bubblewrap.
func writeAllowDirs(roots []string) []string {
	dirs := append([]string{}, roots...)
	dirs = append(dirs, "/dev", "/tmp", os.TempDir())
	if home, err := os.UserHomeDir(); err == nil {
		for _, sub := range []string{".cache", ".npm", ".cargo", "go"} {
			dirs = append(dirs, filepath.Join(home, sub))
		}
	}
	// macOS-specific paths added on Darwin only.
	if runtime.GOOS == "darwin" {
		dirs = append(dirs, "/private/tmp", "/private/var/folders")
		if home, err := os.UserHomeDir(); err == nil {
			dirs = append(dirs, filepath.Join(home, "Library/Caches"))
		}
	}
	seen := map[string]bool{}
	out := make([]string, 0, len(dirs))
	for _, d := range dirs {
		if d == "" {
			continue
		}
		abs, err := filepath.Abs(d)
		if err != nil {
			continue
		}
		if real, err := filepath.EvalSymlinks(abs); err == nil {
			abs = real
		}
		if !seen[abs] {
			seen[abs] = true
			out = append(out, abs)
		}
	}
	return out
}
