//go:build !darwin && !windows

package sandbox

import "os/exec"

// Command runs the command wrapped in bubblewrap (bwrap) when spec.Mode is
// "enforce" and bwrap is available on PATH. The sandbox profile mirrors the
// macOS Seatbelt implementation: reads are open across the whole filesystem,
// writes are confined to WriteRoots plus temp dirs and toolchain caches,
// and network is denied unless spec.Network is true.
//
// When bwrap is unavailable the command runs unconfined — boot and acp warn
// about this once at startup. The permission layer still gates every call.
func Command(spec Spec, sh Shell, command string) ([]string, bool) {
	if !spec.enforce() {
		return sh.argv(command), false
	}
	if bwrap, err := exec.LookPath("bwrap"); err == nil {
		return append([]string{bwrap}, bwrapArgs(spec, sh, command)...), true
	}
	return sh.argv(command), false
}

// Available reports whether an OS sandbox is available. On Linux this checks
// for bubblewrap (bwrap) on PATH. Install with: apt install bubblewrap.
func Available() bool {
	_, err := exec.LookPath("bwrap")
	return err == nil
}

// bwrapArgs builds the bubblewrap command-line arguments for the given spec.
// The sandbox profile matches the macOS Seatbelt behaviour:
//   - Entire filesystem read-only by default (--ro-bind / /)
//   - /dev, /proc mounted for process introspection
//   - Write roots from spec + temp dirs + toolchain caches bind-mounted read-write
//   - Network namespace isolated (--unshare-net) unless spec.Network is true
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

	if spec.Network {
		// Remove --unshare-net (it's always first in args).
		args = args[1:]
	}

	return append(args, sh.argv(command)...)
}
