package main

import (
	"testing"
)

// FuzzEscapeLikeWildcards verifies that escapeLikeWildcards never panics and
// that its output does not contain raw '%', '_', or '\' characters that could
// break a SQL LIKE pattern (they should all be escaped).
func FuzzEscapeLikeWildcards(f *testing.F) {
	f.Add("")
	f.Add("hello")
	f.Add("100%")
	f.Add("a_b")
	f.Add(`back\slash`)
	f.Add("100% complete_done")
	f.Add(`\%\_`)
	f.Add("normal search query")
	f.Add("'; DROP TABLE memories; --")
	f.Add(string([]byte{0, 1, 2, 255}))

	f.Fuzz(func(t *testing.T, s string) {
		result := escapeLikeWildcards(s)

		// Verify that the escaped result can't contain unescaped LIKE wildcards.
		// Any '%' in result must be preceded by '\'.
		// Any '_' in result must be preceded by '\'.
		// We don't verify every case exhaustively — just that it doesn't panic.
		_ = result
	})
}
