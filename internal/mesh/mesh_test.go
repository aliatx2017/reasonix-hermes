package mesh

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

// --- Config/New ---

func TestNew_Disabled(t *testing.T) {
	m := New(Config{Enabled: false, Peers: []PeerConfig{{Name: "p1", URL: "http://x", Enabled: true}}})
	if m != nil {
		t.Error("New() should return nil when disabled")
	}
}

func TestNew_NoPeers(t *testing.T) {
	m := New(Config{Enabled: true})
	if m != nil {
		t.Error("New() should return nil with no peers")
	}
}

func TestNew_AllPeersDisabled(t *testing.T) {
	m := New(Config{Enabled: true, Peers: []PeerConfig{{Name: "p1", URL: "http://x", Enabled: false}}})
	if m != nil {
		t.Error("New() should return nil when all peers disabled")
	}
}

func TestNew_ValidPeers(t *testing.T) {
	os.Setenv("TEST_MESH_TOKEN", "test-token")
	defer os.Unsetenv("TEST_MESH_TOKEN")

	m := New(Config{
		Enabled: true,
		Peers: []PeerConfig{
			{Name: "alpha", URL: "http://localhost:9000", TokenEnv: "TEST_MESH_TOKEN", Enabled: true},
			{Name: "beta", URL: "http://localhost:9001", Enabled: true},
		},
	})
	if m == nil {
		t.Fatal("expected non-nil mesh")
	}
	peers := m.Peers()
	if len(peers) != 2 {
		t.Errorf("Peers() = %d, want 2", len(peers))
	}
}

func TestNew_SkipsEmptyURL(t *testing.T) {
	m := New(Config{Enabled: true, Peers: []PeerConfig{{Name: "p1", URL: "", Enabled: true}}})
	if m != nil {
		t.Error("New() should skip peer with empty URL")
	}
}

// --- Peers() ---

func TestPeers(t *testing.T) {
	m := New(Config{Enabled: true, Peers: []PeerConfig{
		{Name: "one", URL: "http://one", Enabled: true},
		{Name: "two", URL: "http://two", Enabled: true},
	}})
	peers := m.Peers()
	if len(peers) != 2 {
		t.Fatalf("Peers() = %d, want 2", len(peers))
	}
	if peers[0] != "one" || peers[1] != "two" {
		t.Errorf("Peers() = %v, want [one two]", peers)
	}
}

// --- Delegate with mock server ---

func TestDelegate_Success(t *testing.T) {
	srv := newMockMCPServer(t, "peer-test", func(method string, params json.RawMessage) (json.RawMessage, error) {
		if method == "initialize" {
			return json.Marshal(map[string]any{
				"protocolVersion": "2024-11-05",
				"serverInfo":      map[string]string{"name": "peer-test", "version": "1.0"},
			})
		}
		if method == "tools/call" {
			return json.Marshal(map[string]any{
				"content": []map[string]string{{"type": "text", "text": "task done by peer-test"}},
			})
		}
		return nil, nil
	})
	defer srv.Close()

	m := New(Config{Enabled: true, Peers: []PeerConfig{
		{Name: "peer-test", URL: srv.URL, Enabled: true},
	}})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := m.Delegate(ctx, "peer-test", "test task")
	if err != nil {
		t.Fatalf("Delegate() error: %v", err)
	}
	if !result.Success {
		t.Errorf("expected success, got error: %s", result.Error)
	}
	if !strings.Contains(result.Response, "task done") {
		t.Errorf("response = %q, want contains 'task done'", result.Response)
	}
}

func TestDelegate_PeerNotFound(t *testing.T) {
	m := New(Config{Enabled: true, Peers: []PeerConfig{
		{Name: "alpha", URL: "http://alpha", Enabled: true},
	}})
	_, err := m.Delegate(context.Background(), "nonexistent", "task")
	if err == nil {
		t.Error("expected error for unknown peer")
	}
}

// --- Broadcast with mock servers ---

func TestBroadcast(t *testing.T) {
	srv1 := newMockMCPServer(t, "p1", func(method string, params json.RawMessage) (json.RawMessage, error) {
		if method == "initialize" {
			return json.Marshal(map[string]any{"protocolVersion": "2024-11-05", "serverInfo": map[string]string{}})
		}
		return json.Marshal(map[string]any{"content": []map[string]string{{"type": "text", "text": "result from p1"}}})
	})
	defer srv1.Close()

	srv2 := newMockMCPServer(t, "p2", func(method string, params json.RawMessage) (json.RawMessage, error) {
		if method == "initialize" {
			return json.Marshal(map[string]any{"protocolVersion": "2024-11-05", "serverInfo": map[string]string{}})
		}
		return json.Marshal(map[string]any{"content": []map[string]string{{"type": "text", "text": "result from p2"}}})
	})
	defer srv2.Close()

	m := New(Config{Enabled: true, Peers: []PeerConfig{
		{Name: "p1", URL: srv1.URL, Enabled: true},
		{Name: "p2", URL: srv2.URL, Enabled: true},
	}})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	results, err := m.Broadcast(ctx, "broadcast task")
	if err != nil {
		t.Fatalf("Broadcast() error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	for _, r := range results {
		if !r.Success {
			t.Errorf("peer %s failed: %s", r.Peer, r.Error)
		}
	}
}

// --- Status ---

func TestStatus(t *testing.T) {
	srv := newMockMCPServer(t, "alive", func(method string, params json.RawMessage) (json.RawMessage, error) {
		return json.Marshal(map[string]any{"protocolVersion": "2024-11-05", "serverInfo": map[string]string{}})
	})
	defer srv.Close()

	m := New(Config{Enabled: true, Peers: []PeerConfig{
		{Name: "alive", URL: srv.URL, Enabled: true},
		{Name: "dead", URL: "http://127.0.0.1:19999", Enabled: true},
	}})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	status := m.Status(ctx)
	if !status["alive"] {
		t.Error("'alive' peer should be reachable")
	}
	// 'dead' should be false (connection refused or timeout)
	if status["dead"] {
		t.Log("'dead' peer unexpectedly reachable (maybe port 19999 is open)")
	}
}

// --- Query ---

func TestQuery(t *testing.T) {
	srv := newMockMCPServer(t, "peer-qry", func(method string, params json.RawMessage) (json.RawMessage, error) {
		if method == "initialize" {
			return json.Marshal(map[string]any{"protocolVersion": "2024-11-05", "serverInfo": map[string]string{}})
		}
		return json.Marshal(map[string]any{"content": []map[string]string{{"type": "text", "text": "query answer"}}})
	})
	defer srv.Close()

	m := New(Config{Enabled: true, Peers: []PeerConfig{
		{Name: "peer-qry", URL: srv.URL, Enabled: true},
	}})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := m.Query(ctx, "peer-qry", "what's the status?")
	if err != nil {
		t.Fatalf("Query() error: %v", err)
	}
	if !result.Success {
		t.Errorf("expected success, got error: %s", result.Error)
	}
}

// --- Council ---

func TestCouncil_ConveneAndConsensus(t *testing.T) {
	srv1 := newMockMCPServer(t, "c1", func(method string, params json.RawMessage) (json.RawMessage, error) {
		if method == "initialize" {
			return json.Marshal(map[string]any{"protocolVersion": "2024-11-05", "serverInfo": map[string]string{}})
		}
		return json.Marshal(map[string]any{"content": []map[string]string{{"type": "text", "text": "proposal from c1"}}})
	})
	defer srv1.Close()

	srv2 := newMockMCPServer(t, "c2", func(method string, params json.RawMessage) (json.RawMessage, error) {
		if method == "initialize" {
			return json.Marshal(map[string]any{"protocolVersion": "2024-11-05", "serverInfo": map[string]string{}})
		}
		return json.Marshal(map[string]any{"content": []map[string]string{{"type": "text", "text": "proposal from c2"}}})
	})
	defer srv2.Close()

	m := New(Config{Enabled: true, Peers: []PeerConfig{
		{Name: "c1", URL: srv1.URL, Enabled: true},
		{Name: "c2", URL: srv2.URL, Enabled: true},
	}})

	council := NewCouncil(m)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := council.Convene(ctx, "council task"); err != nil {
		t.Fatalf("Convene() error: %v", err)
	}

	consensus := council.Consensus()
	if !strings.Contains(consensus, "Council Results") {
		t.Error("consensus should contain 'Council Results'")
	}
	if !strings.Contains(consensus, "c1") || !strings.Contains(consensus, "c2") {
		t.Error("consensus should mention both peers")
	}

	merge := council.Merge()
	if !strings.Contains(merge, "Synthesise") {
		t.Error("merge should contain 'Synthesise'")
	}
}

func TestCouncil_Empty(t *testing.T) {
	m := New(Config{Enabled: true, Peers: []PeerConfig{
		{Name: "only", URL: "http://localhost:1", Enabled: true},
	}})
	council := NewCouncil(m)
	// Don't convene — test empty state
	consensus := council.Consensus()
	if !strings.Contains(consensus, "No peers") {
		t.Errorf("empty consensus = %q, want 'No peers responded'", consensus)
	}
	merge := council.Merge()
	if !strings.Contains(merge, "No proposals") {
		t.Errorf("empty merge = %q", merge)
	}
}

// --- mock MCP server ---

func newMockMCPServer(t *testing.T, name string, handler func(method string, params json.RawMessage) (json.RawMessage, error)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req jsonrpcRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", 400)
			return
		}

		result, err := handler(req.Method, req.Params)
		if err != nil {
			resp := jsonrpcResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error:   &RPCError{Code: -32603, Message: err.Error()},
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		resp := jsonrpcResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  result,
		}
		json.NewEncoder(w).Encode(resp)
	}))
}
