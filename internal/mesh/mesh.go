// Package mesh provides agent-to-agent MCP delegation for Reasonix instances.
// Each instance can act as both a server (exposing mesh tools to peers) and a
// client (delegating tasks to other instances). Council mode broadcasts a task
// to N peers and synthesises their answers into a consensus.
//
// Architecture: mesh uses HTTP JSON-RPC (MCP protocol) to communicate between
// Reasonix instances. Each peer is a reachable MCP endpoint. The local instance
// can register as a peer in another instance's config.
//
// Config is driven by the [mesh] and [[mesh.peers]] TOML sections.
package mesh

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"reasonix/internal/netclient"
)

// PeerConfig describes a remote Reasonix instance reachable via HTTP MCP.
type PeerConfig struct {
	Name     string `toml:"name"`      // human-readable name
	URL      string `toml:"url"`       // MCP HTTP endpoint
	TokenEnv string `toml:"token_env"` // env var for bearer token
	Enabled  bool   `toml:"enabled"`   // whether this peer is active
}

// Config is the [mesh] TOML section.
type Config struct {
	Enabled bool        `toml:"enabled"`
	Peers   []PeerConfig `toml:"peers"`
}

// Peer is a connected remote instance.
type Peer struct {
	Name   string
	URL    string
	token  string
	client *http.Client
}

// DelegationResult is the result of a single delegate/broadcast operation.
type DelegationResult struct {
	Peer     string `json:"peer"`
	Success  bool   `json:"success"`
	Response string `json:"response"`
	Error    string `json:"error,omitempty"`
	Duration time.Duration `json:"durationMs"`
}

// Raw JSON-RPC structures used to call remote MCP tools.

type jsonrpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type jsonrpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type initializeResult struct {
	ProtocolVersion string `json:"protocolVersion"`
	ServerInfo      struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"serverInfo"`
}

type callToolParams struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

type callToolResult struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
}

// Mesh manages peer connections and delegation operations.
type Mesh struct {
	mu     sync.RWMutex
	peers  []*Peer
	active bool
}

// New creates a Mesh from config. Returns nil when no peers are configured
// or the mesh is disabled.
func New(cfg Config) *Mesh {
	if !cfg.Enabled || len(cfg.Peers) == 0 {
		return nil
	}
	m := &Mesh{active: true}
	for _, pc := range cfg.Peers {
		if !pc.Enabled {
			continue
		}
		url := strings.TrimSpace(pc.URL)
		if url == "" {
			continue
		}
		token := ""
		if pc.TokenEnv != "" {
			token = os.Getenv(pc.TokenEnv)
		}
		m.peers = append(m.peers, &Peer{
			Name:   pc.Name,
			URL:    url,
			token:  token,
			client: netclient.DefaultClient(),
		})
	}
	if len(m.peers) == 0 {
		return nil
	}
	return m
}

// Peers returns the configured peer names.
func (m *Mesh) Peers() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]string, len(m.peers))
	for i, p := range m.peers {
		out[i] = p.Name
	}
	return out
}

// Delegate sends a task to a single named peer and returns its answer.
func (m *Mesh) Delegate(ctx context.Context, peerName, task string) (*DelegationResult, error) {
	m.mu.RLock()
	peer := m.findPeer(peerName)
	m.mu.RUnlock()
	if peer == nil {
		return nil, fmt.Errorf("mesh: peer %q not found", peerName)
	}
	return m.executeTask(ctx, peer, task)
}

// Broadcast sends a task to every peer and returns all results.
func (m *Mesh) Broadcast(ctx context.Context, task string) ([]DelegationResult, error) {
	m.mu.RLock()
	peers := make([]*Peer, len(m.peers))
	copy(peers, m.peers)
	m.mu.RUnlock()

	var mu sync.Mutex
	var results []DelegationResult
	var wg sync.WaitGroup

	for _, p := range peers {
		wg.Add(1)
		go func(peer *Peer) {
			defer wg.Done()
			result, _ := m.executeTask(ctx, peer, task)
			if result != nil {
				mu.Lock()
				results = append(results, *result)
				mu.Unlock()
			}
		}(p)
	}
	wg.Wait()
	return results, nil
}

// Query sends a question to a peer using the mesh_query tool (if available).
func (m *Mesh) Query(ctx context.Context, peerName, question string) (*DelegationResult, error) {
	m.mu.RLock()
	peer := m.findPeer(peerName)
	m.mu.RUnlock()
	if peer == nil {
		return nil, fmt.Errorf("mesh: peer %q not found", peerName)
	}
	return m.executeQuery(ctx, peer, question)
}

// Status checks whether each peer is reachable.
func (m *Mesh) Status(ctx context.Context) map[string]bool {
	m.mu.RLock()
	peers := make([]*Peer, len(m.peers))
	copy(peers, m.peers)
	m.mu.RUnlock()

	out := make(map[string]bool)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, p := range peers {
		wg.Add(1)
		go func(peer *Peer) {
			defer wg.Done()
			alive := m.ping(ctx, peer)
			mu.Lock()
			out[peer.Name] = alive
			mu.Unlock()
		}(p)
	}
	wg.Wait()
	return out
}

func (m *Mesh) findPeer(name string) *Peer {
	for _, p := range m.peers {
		if p.Name == name {
			return p
		}
	}
	return nil
}

func (m *Mesh) executeTask(ctx context.Context, peer *Peer, task string) (*DelegationResult, error) {
	start := time.Now()
	r := &DelegationResult{Peer: peer.Name}

	// handshake: initialize + tools/list
	if err := m.initialize(ctx, peer); err != nil {
		r.Error = fmt.Sprintf("handshake: %v", err)
		r.Duration = time.Since(start)
		return r, nil
	}

	// call mesh_delegate tool if available
	params := callToolParams{
		Name:      "mesh_delegate",
		Arguments: map[string]any{"task": task},
	}
	paramsBytes, err := json.Marshal(params)
	if err != nil {
		r.Error = fmt.Sprintf("marshal params: %v", err)
		r.Duration = time.Since(start)
		return r, nil
	}

	resp, err := m.call(ctx, peer, "tools/call", paramsBytes)
	if err != nil {
		r.Error = fmt.Sprintf("tools/call: %v", err)
		r.Duration = time.Since(start)
		return r, nil
	}

	var result callToolResult
	if err := json.Unmarshal(resp, &result); err != nil {
		// might be a text-only error response
		r.Error = string(resp)
		r.Duration = time.Since(start)
		return r, nil
	}

	var text strings.Builder
	for _, c := range result.Content {
		text.WriteString(c.Text)
	}
	r.Success = true
	r.Response = text.String()
	r.Duration = time.Since(start)
	return r, nil
}

func (m *Mesh) executeQuery(ctx context.Context, peer *Peer, question string) (*DelegationResult, error) {
	start := time.Now()
	r := &DelegationResult{Peer: peer.Name}

	if err := m.initialize(ctx, peer); err != nil {
		r.Error = fmt.Sprintf("handshake: %v", err)
		r.Duration = time.Since(start)
		return r, nil
	}

	var text strings.Builder
	text.WriteString(question)

	params := callToolParams{
		Name:      "mesh_query",
		Arguments: map[string]any{"question": question},
	}
	paramsBytes, err := json.Marshal(params)
	if err != nil {
		r.Error = fmt.Sprintf("marshal params: %v", err)
		r.Duration = time.Since(start)
		return r, nil
	}

	resp, err := m.call(ctx, peer, "tools/call", paramsBytes)
	if err != nil {
		r.Error = fmt.Sprintf("tools/call: %v", err)
		r.Duration = time.Since(start)
		return r, nil
	}

	var result callToolResult
	if err := json.Unmarshal(resp, &result); err != nil {
		r.Error = string(resp)
		r.Duration = time.Since(start)
		return r, nil
	}
	for _, c := range result.Content {
		text.WriteString(c.Text)
	}
	r.Success = true
	r.Response = text.String()
	r.Duration = time.Since(start)
	return r, nil
}

func (m *Mesh) initialize(ctx context.Context, peer *Peer) error {
	initParams, _ := json.Marshal(map[string]any{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]any{},
		"clientInfo": map[string]any{
			"name":    "reasonix-mesh",
			"version": "1.0.0",
		},
	})

	resp, err := m.call(ctx, peer, "initialize", initParams)
	if err != nil {
		return fmt.Errorf("initialize: %w", err)
	}

	var initResult initializeResult
	if err := json.Unmarshal(resp, &initResult); err != nil {
		return fmt.Errorf("parse initialize result: %w", err)
	}

	// Send initialized notification (fire-and-forget)
	_ = m.notify(ctx, peer, "notifications/initialized", nil)
	return nil
}

func (m *Mesh) ping(ctx context.Context, peer *Peer) bool {
	initParams, _ := json.Marshal(map[string]any{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]any{},
		"clientInfo": map[string]any{
			"name":    "reasonix-mesh-ping",
			"version": "1.0.0",
		},
	})
	_, err := m.call(ctx, peer, "initialize", initParams)
	return err == nil
}

func (m *Mesh) call(ctx context.Context, peer *Peer, method string, params json.RawMessage) (json.RawMessage, error) {
	reqBody := jsonrpcRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  method,
		Params:  params,
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, peer.URL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if peer.token != "" {
		req.Header.Set("Authorization", "Bearer "+peer.token)
	}

	resp, err := peer.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024))
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var rpcResp jsonrpcResponse
	if err := json.Unmarshal(respBody, &rpcResp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	if rpcResp.Error != nil {
		return nil, fmt.Errorf("rpc error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}

	return rpcResp.Result, nil
}

func (m *Mesh) notify(ctx context.Context, peer *Peer, method string, params json.RawMessage) error {
	reqBody := jsonrpcRequest{
		JSONRPC: "2.0",
		ID:      0, // notification: no id expected
		Method:  method,
		Params:  params,
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, peer.URL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if peer.token != "" {
		req.Header.Set("Authorization", "Bearer "+peer.token)
	}

	resp, err := peer.client.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}
