package compress

import (
	"fmt"
	"strings"
	"testing"
)

// ── Benchmarks: Compress ─────────────────────────────────────────────────────

func BenchmarkCompressDisabled(b *testing.B) {
	c := New(false)
	data := strings.Repeat("line of output\n", 100)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Compress("bash", data)
	}
}

func BenchmarkCompressCacheHit(b *testing.B) {
	c := New(true)
	data := strings.Repeat("unique content for cache\n", 50)
	// Prime the cache.
	c.Compress("read_file", data)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Compress("read_file", data)
	}
}

func BenchmarkCompressCacheMiss(b *testing.B) {
	c := New(true)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data := fmt.Sprintf("turn-%d: %s", i, strings.Repeat("x", 200))
		c.Compress("read_file", data)
	}
}

func BenchmarkCompressBashNoRepeat(b *testing.B) {
	c := New(true)
	var lines []string
	for i := 0; i < 50; i++ {
		lines = append(lines, fmt.Sprintf("line-%d: some output", i))
	}
	data := strings.Join(lines, "\n")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Compress("bash", data)
	}
}

func BenchmarkCompressBashManyRepeats(b *testing.B) {
	c := New(true)
	// 100 lines, every 10th line repeats 10 times.
	var lines []string
	for i := 0; i < 100; i++ {
		line := fmt.Sprintf("line-%d", i/10)
		lines = append(lines, line)
	}
	data := strings.Join(lines, "\n")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Compress("bash", data)
	}
}

func BenchmarkCompressJSONSmall(b *testing.B) {
	c := New(true)
	data := `{"key1": "value1", "key2": null, "key3": "value3"}`
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Compress("web_fetch", data)
	}
}

func BenchmarkCompressJSONLarge(b *testing.B) {
	c := New(true)
	// 500-line JSON with 20% null fields.
	var lines []string
	lines = append(lines, "{")
	for i := 0; i < 500; i++ {
		val := fmt.Sprintf(`"value-%d"`, i)
		if i%5 == 0 {
			val = "null"
		}
		lines = append(lines, fmt.Sprintf(`  "key-%d": %s,`, i, val))
	}
	lines = append(lines, `  "last": "done"`)
	lines = append(lines, "}")
	data := strings.Join(lines, "\n")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Compress("web_fetch", data)
	}
}

func BenchmarkCompressErrorOutput(b *testing.B) {
	c := New(true)
	data := "panic: runtime error: invalid memory address\n\ngoroutine 1 [running]:\nmain.main()\n\t/path/to/main.go:10 +0x25\n"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Compress("bash", data)
	}
}

func BenchmarkCompressEmpty(b *testing.B) {
	c := New(true)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Compress("bash", "")
	}
}

// ── Benchmarks: SHA-256 ──────────────────────────────────────────────────────

func BenchmarkSHA256HexSmall(b *testing.B) {
	data := "hello world"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sha256Hex(data)
	}
}

func BenchmarkSHA256HexMedium(b *testing.B) {
	data := strings.Repeat("hello world ", 100) // ~1.2KB
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sha256Hex(data)
	}
}

func BenchmarkSHA256HexLarge(b *testing.B) {
	data := strings.Repeat("hello world ", 10000) // ~120KB
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sha256Hex(data)
	}
}

// ── Benchmarks: isErrorOutput ────────────────────────────────────────────────

func BenchmarkIsErrorOutputMatch(b *testing.B) {
	data := "Error: something failed\npanic: runtime error\n"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		isErrorOutput(data)
	}
}

func BenchmarkIsErrorOutputNoMatch(b *testing.B) {
	data := strings.Repeat("normal output line\n", 100)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		isErrorOutput(data)
	}
}

// ── Benchmarks: firstLine ────────────────────────────────────────────────────

func BenchmarkFirstLine(b *testing.B) {
	data := "first line\nsecond line\nthird line"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		firstLine(data, 100)
	}
}

// ── Benchmarks: tokenEstimate ────────────────────────────────────────────────

func BenchmarkTokenEstimate(b *testing.B) {
	data := strings.Repeat("hello world ", 1000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tokenEstimate(data)
	}
}
