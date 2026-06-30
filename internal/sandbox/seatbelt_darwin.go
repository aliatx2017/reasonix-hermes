package sandbox

import (
	"fmt"
	"os/exec"
	"strings"
)

// Command returns the argv to run `command` through sh, wrapped in sandbox-exec
// when the spec enforces and the tool is available. The second return is whether
// wrapping happened; false means the command runs unconfined (sandbox off, or
// sandbox-exec missing — a graceful fallback rather than a hard failure, since
// the permission layer still gates the call).
func Command(spec Spec, sh Shell, command string) ([]string, bool) {
	if !spec.enforce() || !Available() {
		return sh.argv(command), false
	}
	return append([]string{"sandbox-exec", "-p", seatbeltProfile(spec)}, sh.argv(command)...), true
}

// CommandArgs is like Command but accepts the command as raw argv instead of a
// shell command string. The args are appended directly after the sandbox prefix
// without shell interpretation — suitable for direct binary invocations like
// ripgrep that don't need a shell wrapper.
func CommandArgs(spec Spec, args []string) ([]string, bool) {
	if !spec.enforce() || !Available() {
		return args, false
	}
	return append([]string{"sandbox-exec", "-p", seatbeltProfile(spec)}, args...), true
}

// Available reports whether sandbox-exec is on PATH (it ships with macOS).
func Available() bool {
	_, err := exec.LookPath("sandbox-exec")
	return err == nil
}

// seatbeltProfile builds an SBPL profile that allows everything, then denies
// all file writes and re-allows them only under the write-roots (workspace +
// temp + caches). Network is denied unless allowed. Forbid-read roots get
// individual deny-read rules. Reads elsewhere are left open so the
// toolchain (compilers reading GOROOT, git reading ~/.gitconfig, …) keeps
// working — the boundary this draws is "can't write outside the configured
// writable roots, and optionally can't talk to the network", which is the Phase
// 0 blast-radius made to also cover arbitrary shell commands.
func seatbeltProfile(spec Spec) string {
	var b strings.Builder
	b.WriteString("(version 1)\n(allow default)\n(deny file-write*)\n(allow file-write*\n")
	for _, p := range writeAllowDirs(spec.WriteRoots) {
		fmt.Fprintf(&b, "    (subpath %s)\n", sbplString(p))
	}
	b.WriteString(")\n")
	// Deny reads under forbid-read roots so even a permitted shell command
	// cannot peek at them through the OS sandbox. Each path gets its own deny
	// rule; (allow default) above keeps reads working everywhere else.
	for _, p := range forbidReadDirs(spec.ForbidReadRoots) {
		fmt.Fprintf(&b, "(deny file-read* (subpath %s))\n", sbplString(p))
	}
	if !spec.Network {
		b.WriteString("(deny network*)\n")
	}
	return b.String()
}

// sbplString quotes a path as an SBPL string literal, escaping backslash and
// double-quote so a path can't break out of the profile syntax.
func sbplString(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return `"` + s + `"`
}

// forbidReadDirs resolves forbid-read roots to absolute, symlink-free paths so
// Seatbelt matches the canonical on-disk location (e.g. /private/tmp for /tmp).
func forbidReadDirs(roots []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(roots))
	for _, d := range roots {
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
