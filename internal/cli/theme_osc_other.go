//go:build !darwin && !dragonfly && !freebsd && !linux && !netbsd && !openbsd && !solaris && !windows

package cli

func queryTerminalBackground() (terminalRGB, bool) {
	return terminalRGB{}, false
}
