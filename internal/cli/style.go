package cli

import (
	"bytes"
	"fmt"
	"math"
	"os"
	"os/exec"
	"time"

	"golang.org/x/term"
)

// colorEnabled is decided once at startup: only colorize when writing to a real
// terminal and the user hasn't opted out via NO_COLOR (https://no-color.org) or
// a dumb TERM. Piped/redirected output and CI stay plain so scripts aren't broken.
var colorEnabled = detectColor()

func detectColor() bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	if os.Getenv("TERM") == "dumb" {
		return false
	}
	return term.IsTerminal(int(os.Stdout.Fd()))
}

const (
	ansiReset   = "\033[0m"
	ansiBold    = "\033[1m"
	ansiDim     = "\033[2m"
	ansiGreen   = "\033[32m"
	ansiRed     = "\033[31m"
	ansiYellow  = "\033[33m"
	ansiBlue    = "\033[38;5;39m"
	ansiCyan    = "\033[38;5;44m"
	ansiMagenta = "\033[38;5;176m"
	ansiReverse = "\033[7m"
	// ansiAccent is the dark theme fallback for Reasonix's warm copper brand
	// colour. accent() uses the active CLI theme, but tests and legacy callers can
	// still refer to this concrete escape sequence.
	ansiAccent = "\033[38;5;173m"

)

func sgr(code, s string) string {
	if !colorEnabled {
		return s
	}
	return code + s + ansiReset
}

func bold(s string) string    { return sgr(ansiBold, s) }
func dim(s string) string     { return themeFg(activeCLITheme.faint, s) }
func green(s string) string   { return themeFg(activeCLITheme.success, s) }
func red(s string) string     { return themeFg(activeCLITheme.err, s) }
func yellow(s string) string  { return themeFg(activeCLITheme.warn, s) }
func accent(s string) string  { return themeFg(activeCLITheme.accent, s) }
func reverse(s string) string { return sgr(ansiReverse, s) }

// logoGradient renders text in a crossfading cycle through the Diamond Wing logo
// palette: indigo → cyan → pink → indigo over an 8-second period.
func logoGradient(s string) string {
	if !colorEnabled {
		return s
	}
	pos := float64(time.Now().UnixMilli()%8000) / 8000.0
	r, g, b := logoBlend(pos)
	return fmt.Sprintf("\033[38;2;%d;%d;%dm%s\033[0m", r, g, b, s)
}

// logoBlend maps a 0..1 position to an RGB color in the logo cycle.
func logoBlend(pos float64) (r, g, b int) {
	type rgb struct{ r, g, b int }
	indigo := rgb{99, 102, 241}  // #6366f1
	cyan := rgb{6, 182, 212}     // #06b6d4
	pink := rgb{217, 70, 239}    // #d946ef

	var a, c rgb
	var t float64
	switch {
	case pos < 1.0/3.0:
		a, c = indigo, cyan
		t = pos * 3
	case pos < 2.0/3.0:
		a, c = cyan, pink
		t = (pos - 1.0/3.0) * 3
	default:
		a, c = pink, indigo
		t = (pos - 2.0/3.0) * 3
	}
	// Cosine ease in/out for smooth blend.
	ease := (1 - math.Cos(t*math.Pi)) / 2
	r = a.r + int(float64(c.r-a.r)*ease)
	g = a.g + int(float64(c.g-a.g)*ease)
	b = a.b + int(float64(c.b-a.b)*ease)
	return
}

// resolveVersion returns the current build version. When BuildVersion was set
// via ldflags (production build), it returns that. Otherwise it tries git
// describe for a dynamic version tag, falling back to a hardcoded default only
// when git is unavailable.
func resolveVersion() string {
	if BuildVersion != "dev" && BuildVersion != "" {
		return BuildVersion
	}
	cmd := exec.Command("git", "describe", "--tags", "--match", "v*")
	cmd.Stderr = nil
	out, err := cmd.Output()
	if err == nil {
		return string(bytes.TrimSpace(out))
	}
	// Last resort — keep this updated when cutting a new major/minor tag.
	return "v1.10.0"
}
