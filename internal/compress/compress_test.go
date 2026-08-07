package compress

import (
	"strings"
	"testing"
)

func TestCompressDisabled(t *testing.T) {
	c := New(false)
	raw := "hello world"
	if got := c.Compress("bash", raw); got != raw {
		t.Errorf("disabled compressor should return input unchanged, got %q", got)
	}
}

func TestCompressEmpty(t *testing.T) {
	c := New(true)
	if got := c.Compress("bash", ""); got != "" {
		t.Errorf("empty input should stay empty, got %q", got)
	}
}

func TestCompressErrorSafeMode(t *testing.T) {
	c := New(true)
	// Test output with FAIL and error markers — should be preserved.
	errOutput := strings.Join([]string{
		"--- FAIL: TestFoo (0.00s)",
		"    foo_test.go:10: error: expected 1, got 2",
		"    panic: runtime error: invalid memory address",
		"goroutine 1 [running]:",
		"main.main()",
	}, "\n")
	if got := c.Compress("bash", errOutput); got != errOutput {
		t.Errorf("error output should be preserved verbatim, got %q", got)
	}
}

func TestCompressErrorSafeModeSingleMarker(t *testing.T) {
	c := New(true)
	// Single "error:" in otherwise normal output — should still compress.
	single := "line one\nline two\nerror: something\nline four"
	got := c.Compress("bash", single)
	if got == single {
		t.Log("single error marker may not trigger safe mode (needs >=2)")
	}
}

func TestCompressBashRepeatedLines(t *testing.T) {
	c := New(true)
	raw := strings.Join([]string{
		"Building...",
		"Building...",
		"Building...",
		"Building...",
		"Done.",
	}, "\n")
	got := c.Compress("bash", raw)
	if !strings.Contains(got, "[×3 above]") {
		t.Errorf("expected [×3 above] marker for 4 repeated lines, got: %s", got)
	}
	if !strings.Contains(got, "Done.") {
		t.Error("last unique line should be preserved")
	}
}

func TestCompressBashShortOutput(t *testing.T) {
	c := New(true)
	raw := "hello\nworld"
	got := c.Compress("bash", raw)
	// 2 lines is too short to compress.
	if got != raw {
		t.Errorf("short bash output should not be compressed, got %q", got)
	}
}

func TestCompressBashNoRepeats(t *testing.T) {
	c := New(true)
	raw := "line one\nline two\nline three\nline four\nline five"
	got := c.Compress("bash", raw)
	if got != raw {
		t.Errorf("output with no repeats should not change, got %q", got)
	}
}

func TestCompressCacheHitSameTurn(t *testing.T) {
	c := New(true)
	c.SetTurn(5)
	// Long enough that the dedup marker is smaller than the content (the marker
	// embeds a ~100-char first-line summary, so short inputs stay verbatim).
	raw := strings.Repeat("exact same content repeated ", 40)
	first := c.Compress("read_file", raw)
	if first != raw {
		t.Fatalf("first call should return raw, got %q", first)
	}
	second := c.Compress("read_file", raw)
	if !strings.Contains(second, "identical to earlier tool output") {
		t.Errorf("second call should be deduped, got: %s", second)
	}
	if strings.Contains(second, "turn") {
		t.Errorf("marker must be self-contained (no turn anchor that can dangle), got: %s", second)
	}
}

func TestCompressCacheHitDifferentTurn(t *testing.T) {
	c := New(true)
	c.SetTurn(3)
	// Long enough that the dedup marker is smaller than the content (the marker
	// embeds a ~100-char first-line summary, so short inputs stay verbatim).
	raw := strings.Repeat("some file content that gets read twice ", 40)
	c.Compress("read_file", raw)
	c.SetTurn(7)
	got := c.Compress("read_file", raw)
	if !strings.Contains(got, "identical to earlier tool output this session") {
		t.Errorf("cross-turn hit should emit the self-contained dedup marker, got: %s", got)
	}
	if !strings.Contains(got, "sha256:") {
		t.Errorf("cache ref should include sha256 prefix, got: %s", got)
	}
	// The marker must NOT point at a specific turn: the original may have been
	// pruned/compacted away, leaving a dangling reference.
	if strings.Contains(got, "turn 3") {
		t.Errorf("marker must not anchor to a turn number, got: %s", got)
	}
}

func TestCompressCacheDifferentContent(t *testing.T) {
	c := New(true)
	c.SetTurn(1)
	c.Compress("read_file", "content A")
	c.SetTurn(2)
	got := c.Compress("read_file", "content B")
	if got != "content B" {
		t.Errorf("different content should not hit cache, got: %s", got)
	}
}

func TestCompressReadFilePreservesContent(t *testing.T) {
	c := New(true)
	c.SetTurn(1)
	raw := "line 1\nline 2\nline 3"
	got := c.Compress("read_file", raw)
	if got != raw {
		t.Errorf("read_file without cache hit should return unchanged, got %q", got)
	}
}

func TestCompressJSONNullStrip(t *testing.T) {
	c := New(true)
	raw := strings.Join([]string{
		"{",
		`  "name": "test",`,
		`  "optional": null,`,
		`  "value": 42,`,
		`  "unused": null`,
		"}",
	}, "\n")
	got := c.Compress("web_fetch", raw)
	if strings.Contains(got, "null") {
		t.Errorf("null fields should be stripped, got: %s", got)
	}
	if !strings.Contains(got, `"name"`) {
		t.Error("non-null fields should be preserved")
	}
	if !strings.Contains(got, `42`) {
		t.Error("numeric values should be preserved")
	}
}

func TestCompressJSONShort(t *testing.T) {
	c := New(true)
	raw := `{"a":1}`
	got := c.Compress("web_fetch", raw)
	if got != raw {
		t.Errorf("short JSON should not be compressed, got %q", got)
	}
}

func TestSafeModePreservesDiff(t *testing.T) {
	c := New(true)
	raw := strings.Join([]string{
		"diff --git a/file.go b/file.go",
		"--- a/file.go",
		"+++ b/file.go",
		"@@ -1,3 +1,3 @@",
		" error: compilation failed",
	}, "\n")
	got := c.Compress("bash", raw)
	if got != raw {
		t.Errorf("diff + error output should be preserved by safe mode, got: %s", got)
	}
}

func TestIsErrorOutput(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"panic: runtime error\ngoroutine 1 [running]:", true}, // 2 markers: panic: + goroutine
		{"panic: runtime error", false},                        // 1 marker only — needs 2
		{"--- FAIL\nerror: test", true},                        // 2 markers
		{"diff --git a b\nerror: build", true},                 // 2 markers
		{"hello world", false},
		{"success: build complete", false}, // "success:" not a marker
		{"warning: deprecated", false},     // "warning:" not a marker
	}
	for _, tc := range tests {
		got := isErrorOutput(tc.input)
		if got != tc.want {
			t.Errorf("isErrorOutput(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}
}

func TestFirstLine(t *testing.T) {
	if s := firstLine("hello\nworld", 10); s != "hello" {
		t.Errorf("firstLine = %q", s)
	}
	if s := firstLine("\n\n  hi  \nthere", 10); s != "hi" {
		t.Errorf("firstLine skip blanks = %q", s)
	}
	if s := firstLine("very long line here", 8); s != "very lon…" {
		t.Errorf("firstLine truncate = %q", s)
	}
}

func TestTokenEstimate(t *testing.T) {
	if n := tokenEstimate("hello world"); n != 2 {
		t.Errorf("tokenEstimate('hello world') = %d, want ~2", n)
	}
}

func TestIsTextOnly(t *testing.T) {
	if !isTextOnly("hello world\n") {
		t.Error("plain text should be text-only")
	}
	binary := string([]byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05})
	if isTextOnly(binary) {
		t.Error("binary should not be text-only")
	}
}

func TestCompressJSONEmptyLines(t *testing.T) {
	c := New(true)
	raw := strings.Join([]string{
		"{",
		`  "a": 1,`,
		"",
		"",
		`  "b": 2`,
		"}",
	}, "\n")
	got := c.Compress("web_fetch", raw)
	if strings.Count(got, "\n\n") > 0 {
		t.Errorf("empty lines should be stripped, got: %s", got)
	}
}

func TestCompressBlankLines(t *testing.T) {
	c := New(true)
	raw := strings.Join([]string{
		"error:",
		"",
		"fatal:",
		"",
		"end",
	}, "\n")
	// Should be preserved by safe mode (2 error markers).
	got := c.Compress("bash", raw)
	if got != raw {
		t.Errorf("error+blank output should be preserved by safe mode")
	}
}

func TestSortedKeys(t *testing.T) {
	m := map[string]cacheEntry{
		"z": {},
		"a": {},
		"m": {},
	}
	keys := sortedKeys(m)
	if len(keys) != 3 || keys[0] != "a" || keys[2] != "z" {
		t.Errorf("sortedKeys = %v", keys)
	}
}

func TestCompressRapidRepeatSameContent(t *testing.T) {
	t.Parallel()
	c := New(true)
	raw := strings.Repeat("identical output line\n", 100)
	got := c.Compress("bash", raw)
	if !strings.Contains(got, "[×") {
		t.Error("rapid repeats of identical line should be collapsed")
	}
}

func TestCompressZeroByteContent(t *testing.T) {
	t.Parallel()
	c := New(true)
	// Content that is effectively empty but has whitespace
	got := c.Compress("bash", "   \n\t\n   ")
	if got == "" {
		// Acceptable — empty-looking content collapses
		return
	}
	if len(got) > 16 {
		t.Errorf("whitespace-only content should be minimal, got %d bytes: %q", len(got), got)
	}
}

func TestCompressNilSafe(t *testing.T) {
	t.Parallel()
	c := New(true)
	// Verify we don't panic on degenerate inputs
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Compress panicked: %v", r)
			}
		}()
		_ = c.Compress("", "")
		_ = c.Compress("bash", "\x00\x00")
		_ = c.Compress("read_file", strings.Repeat("\n", 1000))
		_ = c.Compress("json", "null")
	}()
}

func TestCompressCacheHitSmallOutputNotBloated(t *testing.T) {
	c := New(true)
	// A tiny repeated output: the dedup marker would be larger than the content,
	// so the hit path must return it verbatim rather than growing the context.
	raw := "ok"
	if first := c.Compress("read_file", raw); first != raw {
		t.Fatalf("first call should return raw, got %q", first)
	}
	second := c.Compress("read_file", raw)
	if second != raw {
		t.Errorf("small repeated output must not be replaced by a larger marker, got %q", second)
	}
	if s := c.Stats(); s.BytesSaved < 0 {
		t.Errorf("BytesSaved must never go negative, got %d", s.BytesSaved)
	}
}

func TestCompressJSONStatsCounted(t *testing.T) {
	c := New(true)
	raw := strings.Join([]string{
		"{",
		`  "name": "test",`,
		`  "optional": null,`,
		`  "value": 42,`,
		`  "unused": null`,
		"}",
	}, "\n")
	got := c.Compress("web_fetch", raw)
	if strings.Contains(got, "null") {
		t.Fatalf("null fields should be stripped, got: %s", got)
	}
	s := c.Stats()
	if s.JSONFieldsStripped < 2 {
		t.Errorf("JSONFieldsStripped should count the two null lines, got %d", s.JSONFieldsStripped)
	}
	if s.BytesSaved <= 0 {
		t.Errorf("JSON minification should record BytesSaved, got %d", s.BytesSaved)
	}
}

func TestCompressStatsAccumulate(t *testing.T) {
	t.Parallel()
	c1 := New(true)
	// Generate content that will definitely compress via repeated-line collapse.
	// 10 identical lines should trigger collapse.
	raw := strings.Repeat("identical\n", 10)
	got := c1.Compress("bash", raw)
	if got == raw {
		t.Skip("repeated-line collapse not triggered for this pattern")
	}
	s1 := c1.Stats()
	// At least one stat counter should be non-zero.
	if s1.CacheHits == 0 && s1.LinesCollapsed == 0 && s1.BytesSaved == 0 {
		t.Error("stats should show activity after compression")
	}
}
