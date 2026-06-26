package config

import (
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
)

// FuzzConfigTOMLDecode verifies that TOML decoding into Config never panics on
// arbitrary input. Malformed TOML should return a parse error, not a panic.
func FuzzConfigTOMLDecode(f *testing.F) {
	f.Add(``)
	f.Add(`default_model = "deepseek/deepseek-reasoner"`)
	f.Add(`
[agent]
max_steps = 50
temperature = 0.7
`)
	f.Add(`
[[providers]]
name = "test"
model = "gpt-4"
api_key_env = "TEST_KEY"

[[plugins]]
name = "test-mcp"
command = "test"
`)
	f.Add(`not valid toml = = = `)
	f.Add(`[incomplete`)
	f.Add("\x00\x01\x02invalid")
	f.Add(`default_model = "` + strings.Repeat("x", 10000) + `"`)
	f.Add(`
[billing]
currency = "CNY"
auto_exchange_rate = true
`)

	f.Fuzz(func(t *testing.T, data string) {
		cfg := Default()
		// toml.Decode should always return an error or succeed — never panic.
		_, _ = toml.Decode(data, cfg)
	})
}
