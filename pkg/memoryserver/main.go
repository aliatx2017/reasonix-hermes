// Hindsight-Reasonix Memory MCP Server — provides cross-session persistent
// memory via MCP tools: hindsight_recall, hindsight_retain, hindsight_reflect.
//
// Usage:
//
//	go run ./pkg/memoryserver [--port 8080]
//
// Can be connected to Reasonix as an MCP plugin:
//
//	[[plugins]]
//	name    = "hindsight"
//	command = "python"
//	args    = ["/path/to/hindsight_mcp.py"]
//
// Or via HTTP:
//
//	[[plugins]]
//	name    = "hindsight"
//	type    = "http"
//	url     = "http://localhost:8080/mcp"
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// MemoryEntry is a single stored memory.
type MemoryEntry struct {
	ID        string    `json:"id"`
	SessionID string    `json:"session_id"`
	Content   string    `json:"content"`
	Tags      []string  `json:"tags,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	AccessCount int     `json:"access_count"`
}

// MemoryStore persists memories to disk.
type MemoryStore struct {
	mu      sync.RWMutex
	dir     string
	entries []MemoryEntry
}

func NewMemoryStore(dir string) (*MemoryStore, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	ms := &MemoryStore{dir: dir}
	ms.load()
	return ms, nil
}

func (ms *MemoryStore) load() {
	path := filepath.Join(ms.dir, "memories.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	json.Unmarshal(data, &ms.entries)
}

func (ms *MemoryStore) save() error {
	data, err := json.MarshalIndent(ms.entries, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(ms.dir, "memories.json"), data, 0644)
}

func (ms *MemoryStore) Retain(sessionID, content string, tags []string) (*MemoryEntry, error) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	entry := MemoryEntry{
		ID:        fmt.Sprintf("mem-%d-%d", len(ms.entries)+1, time.Now().Unix()),
		SessionID: sessionID,
		Content:   content,
		Tags:      tags,
		CreatedAt: time.Now(),
	}
	ms.entries = append(ms.entries, entry)
	if err := ms.save(); err != nil {
		return nil, err
	}
	return &entry, nil
}

func (ms *MemoryStore) Recall(sessionID, query string, limit int) []MemoryEntry {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	var results []MemoryEntry
	lower := strings.ToLower(query)

	for i := range ms.entries {
		e := &ms.entries[i]
		match := query == "" ||
			strings.Contains(strings.ToLower(e.Content), lower) ||
			(e.SessionID == sessionID)

		// Also match tags
		if !match {
			for _, tag := range e.Tags {
				if strings.Contains(strings.ToLower(tag), lower) {
					match = true
					break
				}
			}
		}

		if match {
			e.AccessCount++
			results = append(results, *e)
		}
	}

	ms.save() // persist access counts

	sort.Slice(results, func(i, j int) bool {
		return results[i].CreatedAt.After(results[j].CreatedAt)
	})

	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}

	return results
}

func (ms *MemoryStore) Reflect(sessionID string) string {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	var sessionMemories []MemoryEntry
	for _, e := range ms.entries {
		if e.SessionID == sessionID {
			sessionMemories = append(sessionMemories, e)
		}
	}

	if len(sessionMemories) == 0 {
		return "No memories found for this session."
	}

	var out strings.Builder
	out.WriteString(fmt.Sprintf("# Session Reflection: %s\n\n", sessionID))
	out.WriteString(fmt.Sprintf("%d memories retained:\n\n", len(sessionMemories)))
	for _, e := range sessionMemories {
		out.WriteString(fmt.Sprintf("- [%s] %s\n", e.CreatedAt.Format("Jan 2 15:04"), truncateStr(e.Content, 100)))
	}
	return out.String()
}

func truncateStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-3] + "..."
}

// ── MCP Server ────────────────────────────────────────────────────

type MCPServer struct {
	store *MemoryStore
	mu    sync.Mutex
}

func NewMCPServer(store *MemoryStore) *MCPServer {
	return &MCPServer{store: store}
}

type jsonRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *jsonRPCError   `json:"error,omitempty"`
}

type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (s *MCPServer) ServeStdio() error {
	reader := bufio.NewReader(os.Stdin)
	log.SetOutput(os.Stderr)

	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		resp := s.handleMessage(line)
		if resp != nil {
			resp = append(resp, '\n')
			os.Stdout.Write(resp)
		}
	}
}

func (s *MCPServer) ServeHTTP(addr string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/mcp", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		defer r.Body.Close()
		resp := s.handleMessage(body)
		w.Header().Set("Content-Type", "application/json")
		w.Write(resp)
	})
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"ok","name":"hindsight-reasonix"}`)
	})

	log.Printf("Hindsight Memory MCP server listening on %s", addr)
	return http.ListenAndServe(addr, mux)
}

func (s *MCPServer) handleMessage(data []byte) []byte {
	var req jsonRPCRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return s.errorResp(0, -32700, "Parse error")
	}

	switch req.Method {
	case "initialize":
		return s.successResp(req.ID, map[string]any{
			"protocolVersion": "2024-11-05",
			"serverInfo":      map[string]string{"name": "hindsight-reasonix", "version": "1.0.0"},
			"capabilities":    map[string]any{"tools": map[string]bool{}},
		})
	case "notifications/initialized":
		return nil
	case "tools/list":
		return s.listTools(req.ID)
	case "tools/call":
		return s.callTool(req.ID, req.Params)
	default:
		return s.errorResp(req.ID, -32601, "Method not found")
	}
}

func (s *MCPServer) listTools(id int) []byte {
	tools := []map[string]any{
		{
			"name":        "hindsight_retain",
			"description": "Store a new memory fact for later recall. Use after important decisions or discoveries.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"session_id": map[string]any{"type": "string", "description": "Current session identifier"},
					"content":    map[string]any{"type": "string", "description": "The memory content to store"},
					"tags":       map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Optional tags for categorization"},
				},
				"required": []string{"content"},
			},
		},
		{
			"name":        "hindsight_recall",
			"description": "Search and retrieve memories by keyword, session, or tags.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"session_id": map[string]any{"type": "string", "description": "Filter by session ID"},
					"query":      map[string]any{"type": "string", "description": "Search keyword (empty = all)"},
					"limit":      map[string]any{"type": "integer", "description": "Max results (default 10)"},
				},
			},
		},
		{
			"name":        "hindsight_reflect",
			"description": "Reflect on all memories from a session. Summarizes what was learned and retained.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"session_id": map[string]any{"type": "string", "description": "Session to reflect on"},
				},
				"required": []string{"session_id"},
			},
		},
	}
	return s.successResp(id, map[string]any{"tools": tools})
}

func (s *MCPServer) callTool(id int, params json.RawMessage) []byte {
	var call struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	}
	if err := json.Unmarshal(params, &call); err != nil {
		return s.errorResp(id, -32602, "Invalid params")
	}

	var result string
	var err error

	switch call.Name {
	case "hindsight_retain":
		sessionID, _ := call.Arguments["session_id"].(string)
		content, _ := call.Arguments["content"].(string)
		var tags []string
		if t, ok := call.Arguments["tags"].([]interface{}); ok {
			for _, tag := range t {
				if ts, ok := tag.(string); ok {
					tags = append(tags, ts)
				}
			}
		}
		entry, e := s.store.Retain(sessionID, content, tags)
		if e != nil {
			err = e
		} else {
			result = fmt.Sprintf("Memory retained: %s", entry.ID)
		}

	case "hindsight_recall":
		sessionID, _ := call.Arguments["session_id"].(string)
		query, _ := call.Arguments["query"].(string)
		limit := 10
		if l, ok := call.Arguments["limit"].(float64); ok {
			limit = int(l)
		}
		entries := s.store.Recall(sessionID, query, limit)
		if len(entries) == 0 {
			result = "No matching memories found."
		} else {
			var sb strings.Builder
			sb.WriteString(fmt.Sprintf("# Found %d memories:\n\n", len(entries)))
			for _, e := range entries {
				sb.WriteString(fmt.Sprintf("- [%s] %s\n", e.ID, e.Content))
			}
			result = sb.String()
		}

	case "hindsight_reflect":
		sessionID, _ := call.Arguments["session_id"].(string)
		result = s.store.Reflect(sessionID)

	default:
		return s.errorResp(id, -32601, "Unknown tool: "+call.Name)
	}

	if err != nil {
		return s.errorResp(id, -32000, err.Error())
	}

	content := []map[string]string{{"type": "text", "text": result}}
	return s.successResp(id, map[string]any{"content": content})
}

func (s *MCPServer) successResp(id int, result any) []byte {
	r, _ := json.Marshal(result)
	resp := jsonRPCResponse{JSONRPC: "2.0", ID: id, Result: r}
	data, _ := json.Marshal(resp)
	return data
}

func (s *MCPServer) errorResp(id, code int, message string) []byte {
	resp := jsonRPCResponse{JSONRPC: "2.0", ID: id, Error: &jsonRPCError{Code: code, Message: message}}
	data, _ := json.Marshal(resp)
	return data
}

// ── Main ──────────────────────────────────────────────────────────

func main() {
	storeDir := ".reasonix/hindsight-memory"
	if home, err := os.UserHomeDir(); err == nil {
		storeDir = filepath.Join(home, ".reasonix", "hindsight-memory")
	}

	store, err := NewMemoryStore(storeDir)
	if err != nil {
		log.Fatalf("Failed to create memory store: %v", err)
	}

	server := NewMCPServer(store)

	if len(os.Args) > 1 && os.Args[1] == "--http" {
		port := "8080"
		if len(os.Args) > 3 && os.Args[2] == "--port" {
			port = os.Args[3]
		}
		log.Fatal(server.ServeHTTP(":" + port))
	}

	log.SetPrefix("[hindsight] ")
	log.Println("Starting in stdio mode (MCP)...")
	if err := server.ServeStdio(); err != nil {
		log.Fatal(err)
	}
}
