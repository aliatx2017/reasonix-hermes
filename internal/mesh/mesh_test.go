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
	t.Parallel()
	m := New(Config{Enabled: false, Peers: []PeerConfig{{Name: "p1", URL: "http://x", Enabled: true}}})
	if m != nil {
		t.Error("New() should return nil when disabled")
	}
}

func TestNew_NoPeers(t *testing.T) {
	t.Parallel()
	m := New(Config{Enabled: true})
	if m != nil {
		t.Error("New() should return nil with no peers")
	}
}

func TestNew_AllPeersDisabled(t *testing.T) {
	t.Parallel()
	m := New(Config{Enabled: true, Peers: []PeerConfig{{Name: "p1", URL: "http://x", Enabled: false}}})
	if m != nil {
		t.Error("New() should return nil when all peers disabled")
	}
}

func TestNew_ValidPeers(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
	m := New(Config{Enabled: true, Peers: []PeerConfig{{Name: "p1", URL: "", Enabled: true}}})
	if m != nil {
		t.Error("New() should skip peer with empty URL")
	}
}

// --- Peers() ---

func TestPeers(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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

// --- Council Judge ---

func TestCouncil_Judge_StructuredJSON(t *testing.T) {
	t.Parallel()

	// Simulate proposals from two "peers".
	council := &Council{
		proposals: []DelegationResult{
			{Peer: "model-a", Success: true, Response: "The answer is 42. Edge case: negative numbers should return 0."},
			{Peer: "model-b", Success: true, Response: "Result is 42. Also consider overflow for large inputs."},
		},
	}

	// Judge function that returns valid JSON.
	judgeJSON := func(ctx context.Context, prompt string) (string, error) {
		return `{
  "consensus": "Both models agree the answer is 42.",
  "contradictions": ["Model A says negative numbers return 0, Model B focuses on overflow"],
  "coverage_gaps": ["Model A covered negative numbers, Model B covered overflow"],
  "unique_insights": ["Model A: detailed edge-case handling for negatives"],
  "blind_spots": ["Neither model discussed floating-point errors"]
}`, nil
	}

	ctx := context.Background()
	if err := council.Judge(ctx, judgeJSON); err != nil {
		t.Fatalf("Judge() error: %v", err)
	}

	j := council.Judgment()
	if j == nil {
		t.Fatal("Judgment() returned nil after successful Judge()")
	}
	if j.Consensus != "Both models agree the answer is 42." {
		t.Errorf("Consensus = %q", j.Consensus)
	}
	if len(j.Contradictions) != 1 {
		t.Errorf("Contradictions = %d items, want 1", len(j.Contradictions))
	}
	if len(j.CoverageGaps) != 1 {
		t.Errorf("CoverageGaps = %d items, want 1", len(j.CoverageGaps))
	}
	if len(j.UniqueInsights) != 1 {
		t.Errorf("UniqueInsights = %d items, want 1", len(j.UniqueInsights))
	}
	if len(j.BlindSpots) != 1 {
		t.Errorf("BlindSpots = %d items, want 1", len(j.BlindSpots))
	}
	if j.Raw != "" {
		t.Error("Raw should be empty after successful parse")
	}
}

func TestCouncil_Judge_MarkdownWrappedJSON(t *testing.T) {
	t.Parallel()

	council := &Council{
		proposals: []DelegationResult{
			{Peer: "peer-x", Success: true, Response: "Use approach A."},
		},
	}

	// Simulate a judge that wraps JSON in ``` fences (common LLM behavior).
	judgeFenced := func(ctx context.Context, prompt string) (string, error) {
		return "```json\n{\"consensus\":\"Approach A is recommended.\",\"contradictions\":[],\"coverage_gaps\":[],\"unique_insights\":[],\"blind_spots\":[\"No alternative considered\"]}\n```", nil
	}

	if err := council.Judge(context.Background(), judgeFenced); err != nil {
		t.Fatalf("Judge() with fenced JSON: %v", err)
	}

	j := council.Judgment()
	if j.Consensus != "Approach A is recommended." {
		t.Errorf("Consensus = %q", j.Consensus)
	}
	if len(j.BlindSpots) != 1 || j.BlindSpots[0] != "No alternative considered" {
		t.Errorf("BlindSpots = %v", j.BlindSpots)
	}
}

func TestCouncil_Judge_InvalidJSON_Fallback(t *testing.T) {
	t.Parallel()

	council := &Council{
		proposals: []DelegationResult{
			{Peer: "p", Success: true, Response: "Raw answer here."},
		},
	}

	// Judge returns non-JSON text. Judge() should succeed and store raw.
	judgeText := func(ctx context.Context, prompt string) (string, error) {
		return "I compared the proposals and here is my analysis...", nil
	}

	if err := council.Judge(context.Background(), judgeText); err != nil {
		t.Fatalf("Judge() should not error on unparseable JSON: %v", err)
	}

	j := council.Judgment()
	if j == nil {
		t.Fatal("Judgment() returned nil")
	}
	if j.Raw == "" {
		t.Error("Raw should be populated when JSON parsing fails")
	}
	if j.Consensus != "" {
		t.Error("Consensus should be empty when parsing fails")
	}
}

func TestCouncil_Judge_NoProposals(t *testing.T) {
	t.Parallel()

	council := &Council{
		proposals: []DelegationResult{
			{Peer: "dead", Success: false, Response: "", Error: "timeout"},
		},
	}

	judge := func(ctx context.Context, prompt string) (string, error) { return "{}", nil }
	err := council.Judge(context.Background(), judge)
	if err == nil {
		t.Error("expected error for no successful proposals")
	}
}

func TestCouncil_Judge_NilFunc(t *testing.T) {
	t.Parallel()

	council := &Council{
		proposals: []DelegationResult{
			{Peer: "p", Success: true, Response: "ok"},
		},
	}

	err := council.Judge(context.Background(), nil)
	if err == nil {
		t.Error("expected error for nil judge function")
	}
}

func TestCouncil_Judge_EmptyResponseSkipped(t *testing.T) {
	t.Parallel()

	council := &Council{
		proposals: []DelegationResult{
			{Peer: "empty", Success: true, Response: ""},
			{Peer: "good", Success: true, Response: "Valid response."},
		},
	}

	judge := func(ctx context.Context, prompt string) (string, error) {
		// Verify prompt only contains the "good" peer.
		if !strings.Contains(prompt, "good") {
			t.Error("judge prompt should contain 'good' peer")
		}
		if strings.Contains(prompt, "empty") {
			t.Error("judge prompt should NOT contain 'empty' peer (response was empty)")
		}
		return `{"consensus":"ok","contradictions":[],"coverage_gaps":[],"unique_insights":[],"blind_spots":[]}`, nil
	}

	if err := council.Judge(context.Background(), judge); err != nil {
		t.Fatalf("Judge(): %v", err)
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

func TestDelegate_PeerUnreachable(t *testing.T) {
	t.Parallel()
	m := New(Config{Enabled: true, Peers: []PeerConfig{
		{Name: "gone", URL: "http://127.0.0.1:19999", Enabled: true},
	}})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	result, err := m.Delegate(ctx, "gone", "do something")
	if err != nil {
		t.Fatalf("Delegate returned error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Error == "" {
		t.Error("expected error in result for unreachable peer")
	}
}

func TestBroadcast_PartialFailure(t *testing.T) {
	t.Parallel()
	srv1 := newMockMCPServer(t, "peer-ok", func(method string, params json.RawMessage) (json.RawMessage, error) {
		if method == "initialize" {
			return json.Marshal(map[string]any{
				"protocolVersion": "2024-11-05",
				"serverInfo":      map[string]string{"name": "peer-ok", "version": "1.0"},
			})
		}
		if method == "tools/list" {
			return json.Marshal(map[string]any{"tools": []map[string]any{
				{"name": "mesh_delegate", "inputSchema": map[string]any{}},
			}})
		}
		return json.Marshal(map[string]any{
			"content": []map[string]string{{"type": "text", "text": "ok"}},
		})
	})
	defer srv1.Close()

	m := New(Config{Enabled: true, Peers: []PeerConfig{
		{Name: "peer-ok", URL: srv1.URL, Enabled: true},
		{Name: "peer-gone", URL: "http://127.0.0.1:19998", Enabled: true},
	}})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	results, err := m.Broadcast(ctx, "my-task")
	if err != nil {
		t.Fatalf("Broadcast error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d: %+v", len(results), results)
	}
	var successCount, failCount int
	for _, r := range results {
		if r.Success {
			successCount++
		} else {
			failCount++
		}
	}
	if successCount == 0 {
		t.Error("expected at least one successful peer")
	}
	if failCount == 0 {
		t.Error("expected at least one failed peer (unreachable)")
	}
}
