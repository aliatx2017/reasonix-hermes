//go:build !darwin && !windows

package sandbox

import (
	"os"
	"os/exec"
	"path/filepath"
)

<<<<<<< HEAD
// Command runs the command wrapped in bubblewrap (bwrap) when spec.Mode is
// "enforce" and bwrap is available on PATH. The sandbox profile mirrors the
// macOS Seatbelt implementation: reads are open across the whole filesystem,
// writes are confined to WriteRoots plus temp dirs and toolchain caches,
// and network is denied unless spec.Network is true.
//
// When bwrap is unavailable the command runs unconfined — boot and acp warn
// about this once at startup. The permission layer still gates every call.
=======
// When spec.Mode is "enforce" and bubblewrap (bwrap) is available on PATH,
// the command is wrapped in a bubblewrap sandbox with a profile analogous to
// macOS Seatbelt: writes confined to WriteRoots, network denied unless
// spec.Network is true. When bwrap is unavailable the command runs unconfined
// (boot and acp warn about this once at startup).
>>>>>>> upstream/main-v2
func Command(spec Spec, sh Shell, command string) ([]string, bool) {
	if !spec.enforce() {
		return sh.argv(command), false
	}
	if bwrap, err := exec.LookPath("bwrap"); err == nil {
		return append([]string{bwrap}, bwrapArgs(spec, sh, command)...), true
	}
	return sh.argv(command), false
}

<<<<<<< HEAD
// Available reports whether an OS sandbox is available. On Linux this checks
// for bubblewrap (bwrap) on PATH. Install with: apt install bubblewrap.
=======
// CommandArgs is like Command but accepts the command as raw argv instead of a
// shell command string. The args are appended directly after the bwrap sandbox
// prefix without shell interpretation — suitable for direct binary invocations
// like ripgrep that don't need a shell wrapper.
func CommandArgs(spec Spec, args []string) ([]string, bool) {
	if !spec.enforce() {
		return args, false
	}
	if bwrap, err := exec.LookPath("bwrap"); err == nil {
		argv := append([]string{bwrap}, bwrapArgsForArgs(spec, args)...)
		return argv, true
	}
	return args, false
}

// Available reports whether an OS sandbox is available on this platform.
// On Linux, this checks for bubblewrap (bwrap) on PATH.
>>>>>>> upstream/main-v2
func Available() bool {
	_, err := exec.LookPath("bwrap")
	return err == nil
}

<<<<<<< HEAD
// bwrapArgs builds the bubblewrap command-line arguments for the given spec.
// The sandbox profile matches the macOS Seatbelt behaviour:
//   - Entire filesystem read-only by default (--ro-bind / /)
//   - /dev, /proc mounted for process introspection
//   - Write roots from spec + temp dirs + toolchain caches bind-mounted read-write
//   - Network namespace isolated (--unshare-net) unless spec.Network is true
=======
// bwrapArgs builds the bubblewrap command-line arguments that confine the
// shell command to the write roots, deny network unless allowed, and overlay
// forbid-read directories with tmpfs so they appear empty. The rest of the
// filesystem is mounted read-only (matching the macOS Seatbelt profile).
>>>>>>> upstream/main-v2
func bwrapArgs(spec Spec, sh Shell, command string) []string {
	args := []string{
		"--unshare-net",
		"--ro-bind", "/", "/",
		"--dev", "/dev",
		"--proc", "/proc",
	}

	for _, root := range writeAllowDirs(spec.WriteRoots) {
		// Bind-mount the directory read-write. bwrap creates the mount
		// point inside the namespace automatically.
		args = append(args, "--bind", root, root)
	}
<<<<<<< HEAD

	if spec.Network {
		// Remove --unshare-net (it's always first in args).
		args = args[1:]
	}

=======
	for _, root := range linuxWriteDirs() {
		args = append(args, "--bind", root, root)
	}
	for _, root := range spec.ForbidReadRoots {
		args = append(args, "--tmpfs", root)
	}
>>>>>>> upstream/main-v2
	return append(args, sh.argv(command)...)
}

// bwrapArgsForArgs is like bwrapArgs but accepts raw argv instead of a shell
// command string. It builds the same sandbox prefix and appends the caller's
// argv directly — no shell interpreter wrapping.
func bwrapArgsForArgs(spec Spec, args []string) []string {
	out := []string{
		"--unshare-net", // deny network by default
		"--ro-bind", "/", "/",
		"--dev", "/dev",
		"--proc", "/proc",
		"--tmpfs", "/tmp",
	}
	if spec.Network {
		// Re-allow network by removing the network namespace.
		out = out[1:] // drop --unshare-net
	}
	for _, root := range spec.WriteRoots {
		out = append(out, "--bind", root, root)
	}
	for _, root := range linuxWriteDirs() {
		out = append(out, "--bind", root, root)
	}
	for _, root := range spec.ForbidReadRoots {
		out = append(out, "--tmpfs", root)
	}
	return append(out, args...)
}

func linuxWriteDirs() []string {
	dirs := []string{}
	if td := os.TempDir(); td != "" && td != "/tmp" {
		dirs = append(dirs, td)
	}
	if home, err := os.UserHomeDir(); err == nil {
		for _, sub := range []string{".cache", ".cargo", ".npm", "go"} {
			dirs = append(dirs, filepath.Join(home, sub))
		}
	}
	seen := map[string]bool{}
	out := make([]string, 0, len(dirs))
	for _, d := range dirs {
		abs, err := filepath.Abs(d)
		if err != nil {
			continue
		}
		if real, err := filepath.EvalSymlinks(abs); err == nil {
			abs = real
		}
		if abs == "/tmp" || seen[abs] || !dirExists(abs) {
			continue
		}
		seen[abs] = true
		out = append(out, abs)
	}
	return out
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
