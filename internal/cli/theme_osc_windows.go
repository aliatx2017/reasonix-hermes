//go:build windows

package cli

import (
	"errors"
	"os"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/term"
)

const (
	terminalBGQueryTimeout  = 80 * time.Millisecond
	terminalBGQueryMaxBytes = 256
)

var (
	kernel32                = syscall.NewLazyDLL("kernel32.dll")
	procGetConsoleScreenBufferInfo = kernel32.NewProc("GetConsoleScreenBufferInfo")
)

type coord struct {
	X, Y int16
}

type smallRect struct {
	Left, Top, Right, Bottom int16
}

type consoleScreenBufferInfo struct {
	Size              coord
	CursorPosition    coord
	Attributes        uint16
	Window            smallRect
	MaximumWindowSize coord
}

func getConsoleScreenBufferInfo() (*consoleScreenBufferInfo, error) {
	var csbi consoleScreenBufferInfo
	handle := syscall.Handle(os.Stdout.Fd())
	r, _, err := procGetConsoleScreenBufferInfo.Call(uintptr(handle), uintptr(unsafe.Pointer(&csbi)))
	if r == 0 {
		return nil, err
	}
	return &csbi, nil
}

// attributeToRGB maps classic Windows console color attributes to approximate RGB.
// Values 0-15 use the standard console palette (matching the default cmd.exe colors);
// values above 15 use high-bit extraction to guess at modern terminals.
func attributeToRGB(attr uint16) terminalRGB {
	bg := (attr >> 4) & 0xF
	switch bg {
	case 0: // BLACK
		return terminalRGB{0, 0, 0}
	case 1: // DARK_BLUE
		return terminalRGB{0, 0, 128}
	case 2: // DARK_GREEN
		return terminalRGB{0, 128, 0}
	case 3: // DARK_CYAN
		return terminalRGB{0, 128, 128}
	case 4: // DARK_RED
		return terminalRGB{128, 0, 0}
	case 5: // DARK_MAGENTA
		return terminalRGB{128, 0, 128}
	case 6: // DARK_YELLOW (brown)
		return terminalRGB{128, 128, 0}
	case 7: // LIGHT_GRAY (default console bg)
		return terminalRGB{192, 192, 192}
	case 8: // DARK_GRAY
		return terminalRGB{128, 128, 128}
	case 9: // BLUE
		return terminalRGB{0, 0, 255}
	case 10: // GREEN
		return terminalRGB{0, 255, 0}
	case 11: // CYAN
		return terminalRGB{0, 255, 255}
	case 12: // RED
		return terminalRGB{255, 0, 0}
	case 13: // MAGENTA
		return terminalRGB{255, 0, 255}
	case 14: // YELLOW
		return terminalRGB{255, 255, 0}
	case 15: // WHITE
		return terminalRGB{255, 255, 255}
	default:
		return terminalRGB{0, 0, 0}
	}
}

func queryTerminalBackground() (terminalRGB, bool) {
	if !colorEnabled {
		return terminalRGB{}, false
	}
	inFd := int(os.Stdin.Fd())
	outFd := int(os.Stdout.Fd())

	// Try OSC 11 query first — Windows Terminal, WezTerm, and other modern
	// terminals support this.
	if term.IsTerminal(inFd) && term.IsTerminal(outFd) {
		if rgb, ok := queryTerminalBackgroundOSC(inFd, outFd); ok {
			return rgb, true
		}
	}

	// Fall back to Console API attributes for classic conhost.exe.
	csbi, err := getConsoleScreenBufferInfo()
	if err != nil {
		return terminalRGB{}, false
	}
	return attributeToRGB(csbi.Attributes), true
}

func queryTerminalBackgroundOSC(inFd, outFd int) (terminalRGB, bool) {
	oldState, err := term.MakeRaw(inFd)
	if err != nil {
		return terminalRGB{}, false
	}
	defer term.Restore(inFd, oldState)

	if _, err := os.Stdout.Write([]byte("\x1b]11;?\x07")); err != nil {
		return terminalRGB{}, false
	}

	deadline := time.Now().Add(terminalBGQueryTimeout)
	buf := make([]byte, 64)
	var response []byte
	for time.Now().Before(deadline) && len(response) < terminalBGQueryMaxBytes {
		n, err := syscall.Read(syscall.Handle(inFd), buf)
		if n > 0 {
			response = append(response, buf[:n]...)
			if rgb, ok := parseOSC11Response(string(response)); ok {
				return rgb, true
			}
			continue
		}
		if err == nil || errors.Is(err, syscall.Errno(0)) {
			continue
		}
		// EAGAIN is not a Windows concept; just sleep-retry.
		time.Sleep(5 * time.Millisecond)
	}
	return parseOSC11Response(string(response))
}
