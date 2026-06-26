package scheduler

import (
	"testing"
	"time"
)

// FuzzParseCron verifies that parseCron never panics on arbitrary input.
// Invalid expressions should return an error, not a panic.
func FuzzParseCron(f *testing.F) {
	f.Add("* * * * *")
	f.Add("0 2 * * *")
	f.Add("*/15 * * * *")
	f.Add("0 9-17 * * 1-5")
	f.Add("0,30 0,12 * * *")
	f.Add("0 0-23/3 * * *")
	f.Add("0 0 1 1 *")
	f.Add("invalid")
	f.Add("")
	f.Add("* * *")
	f.Add("* * * * * *")
	f.Add("60 * * * *")
	f.Add("-1 * * * *")
	f.Add("abc * * * *")
	f.Add("0 0 31 2 *")
	f.Add("*/0 * * * *")
	f.Add("1-abc * * * *")

	f.Fuzz(func(t *testing.T, expr string) {
		// Must not panic — invalid expressions return an error.
		_, _ = parseCron(expr)
	})
}

// FuzzNextAfter verifies that NextAfter never panics on arbitrary cron
// expressions. Invalid expressions return an error; valid ones return a time.
func FuzzNextAfter(f *testing.F) {
	f.Add("* * * * *")
	f.Add("0 2 * * *")
	f.Add("*/15 * * * *")
	f.Add("0 9-17 * * 1-5")
	f.Add("0 0 31 2 *")
	f.Add("invalid")
	f.Add("")
	f.Add("60 24 32 13 8")

	now := time.Date(2026, 6, 26, 12, 0, 0, 0, time.UTC)

	f.Fuzz(func(t *testing.T, expr string) {
		// Must not panic — any expression is safe to pass.
		_, _ = NextAfter(expr, now)
	})
}
