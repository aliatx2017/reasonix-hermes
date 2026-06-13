//go:build !windows

package sandbox

// IsAppContainer reports whether the Windows AppContainer sandbox is the
// active enforcement backend. Always false on non-Windows.
func IsAppContainer() bool { return false }

// ExecAppContainer is a no-op stub on non-Windows platforms.
func ExecAppContainer(spec Spec, sh Shell, command string, env []string, dir string) (string, error) {
	return "", nil
}
