// Package mcputil provides a shared JSON-RPC / MCP server framework.
// Both the MCP bridge and the Hindsight memory server use this package
// to avoid duplicating protocol scaffolding.
package mcputil

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"reasonix/pkg/httputil"
)

// Tool is an MCP tool definition.
type Tool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

// Handler is called when a tool is invoked. Returns result text or error.
type Handler func(name string, arguments map[string]any) (string, error)

// Request is a JSON-RPC 2.0 request with raw ID (string, number, or null).
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// Response is a JSON-RPC 2.0 response with raw ID echoed verbatim.
type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

// RPCError is a JSON-RPC error object.
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Server implements MCP protocol over stdio or HTTP.
type Server struct {
	Name    string
	Version string
	Tools   []Tool
	Handle  Handler
}

// ServeStdio runs the server over stdin/stdout (MCP stdio transport).
func (s *Server) ServeStdio() error {
	reader := bufio.NewReader(os.Stdin)
	writer := os.Stdout
	log.SetOutput(os.Stderr)

	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("read stdin: %w", err)
		}

		resp := s.HandleMessage(line)
		if resp == nil {
			continue // notification — no response
		}
		resp = append(resp, '\n')
		if _, err := writer.Write(resp); err != nil {
			return fmt.Errorf("write stdout: %w", err)
		}
	}
}

// HandleHTTP is an http.HandlerFunc that reads the request body, processes it
// as an MCP message, and writes the JSON response. Use this to mount the MCP
// endpoint on a custom mux.
func (s *Server) HandleHTTP(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20) // 10 MB limit
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()
	resp := s.HandleMessage(body)
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(resp)
}

// ServeHTTP runs the server over HTTP with optional Bearer auth.
// authKeyEnv is the env var name for the API key (empty = no auth).
func (s *Server) ServeHTTP(addr, authKeyEnv string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/mcp", func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 10<<20) // 10 MB limit
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()
		resp := s.HandleMessage(body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(resp)
	})
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"ok","name":%q,"version":%q}`, s.Name, s.Version)
	})

	auth := &httputil.AuthMiddleware{
		APIKey: httputil.LoadAPIKey(authKeyEnv),
		KeyEnv: authKeyEnv,
	}
	handler := auth.Wrap(mux)

	srv := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		sc := make(chan os.Signal, 1)
		signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM)
		<-sc
		log.Printf("[%s] shutting down...", s.Name)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("[%s] shutdown error: %v", s.Name, err)
		}
	}()

	log.Printf("[%s] HTTP server listening on %s", s.Name, addr)
	return srv.ListenAndServe()
}

// HandleMessage processes a single JSON-RPC message and returns the response bytes.
// Returns nil for notifications (no response required).
func (s *Server) HandleMessage(data []byte) []byte {
	var req Request
	if err := json.Unmarshal(data, &req); err != nil {
		return s.ErrorResponse(json.RawMessage("null"), -32700, "Parse error: "+err.Error())
	}

	switch req.Method {
	case "initialize":
		return s.handleInitialize(req)
	case "notifications/initialized":
		return nil
	case "tools/list":
		return s.handleListTools(req)
	case "tools/call":
		return s.handleCallTool(req)
	default:
		return s.ErrorResponse(req.ID, -32601, "Method not found: "+req.Method)
	}
}

func (s *Server) handleInitialize(req Request) []byte {
	result := map[string]any{
		"protocolVersion": "2024-11-05",
		"serverInfo": map[string]string{
			"name":    s.Name,
			"version": s.Version,
		},
		"capabilities": map[string]any{
			"tools": map[string]bool{},
		},
	}
	return s.SuccessResponse(req.ID, result)
}

func (s *Server) handleListTools(req Request) []byte {
	result := map[string]any{"tools": s.Tools}
	return s.SuccessResponse(req.ID, result)
}

func (s *Server) handleCallTool(req Request) []byte {
	var call struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	}
	if err := json.Unmarshal(req.Params, &call); err != nil {
		return s.ErrorResponse(req.ID, -32602, "Invalid params: "+err.Error())
	}

	result, err := s.Handle(call.Name, call.Arguments)
	if err != nil {
		return s.ErrorResponse(req.ID, -32000, err.Error())
	}

	content := []map[string]string{
		{"type": "text", "text": result},
	}
	return s.SuccessResponse(req.ID, map[string]any{"content": content})
}

// SuccessResponse builds a JSON-RPC success response.
func (s *Server) SuccessResponse(id json.RawMessage, result any) []byte {
	r, _ := json.Marshal(result)
	resp := Response{
		JSONRPC: "2.0",
		ID:      id,
		Result:  r,
	}
	data, _ := json.Marshal(resp)
	return data
}

// ErrorResponse builds a JSON-RPC error response.
func (s *Server) ErrorResponse(id json.RawMessage, code int, message string) []byte {
	resp := Response{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &RPCError{Code: code, Message: message},
	}
	data, _ := json.Marshal(resp)
	return data
}
