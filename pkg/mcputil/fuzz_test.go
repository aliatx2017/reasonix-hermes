package mcputil

import (
	"testing"
)

// FuzzHandleMessage verifies that HandleMessage never panics on arbitrary input.
// The JSON-RPC parse path uses json.Unmarshal on untrusted bytes, so any
// malformed input should return an error response rather than panicking.
func FuzzHandleMessage(f *testing.F) {
	f.Add([]byte(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`))
	f.Add([]byte(`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`))
	f.Add([]byte(`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"noop","arguments":{}}}`))
	f.Add([]byte(`{"jsonrpc":"2.0","method":"notifications/initialized"}`))
	f.Add([]byte(`{"jsonrpc":"2.0","id":null,"method":"unknown"}`))
	f.Add([]byte(`{}`))
	f.Add([]byte(`not-json`))
	f.Add([]byte(nil))
	f.Add([]byte(`{"jsonrpc":"2.0","id":"str-id","method":"tools/call","params":{"name":"noop","arguments":{"x":1}}}`))
	f.Add([]byte("\x00\x01\x02"))

	s := &Server{
		Name:    "fuzz",
		Version: "0.0.1",
		Tools: []Tool{
			{Name: "noop", Description: "no-op", InputSchema: map[string]any{"type": "object", "properties": map[string]any{}}},
		},
		Handle: func(name string, arguments map[string]any) (string, error) {
			return "ok", nil
		},
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		// Must not panic — any input is valid at the protocol boundary.
		_ = s.HandleMessage(data)
	})
}
