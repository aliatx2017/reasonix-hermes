// Package collab provides live collaborative session sharing between Reasonix
// instances via WebSocket. An instance can share its session transcript in
// real-time, letting peers watch tool calls, turn results, and messages.
// Connected peers can optionally steer or contribute commands.
//
// Architecture: a Hub manages WebSocket connections. A session can have one
// owner (the active agent running turns) and multiple watchers. Watchers receive
// a stream of session events and can send steer commands back.
//
// Transport is a simple JSON message protocol over WebSocket:
//   → subscribe {sessionID, role: "watcher"|"owner"}
//   → event {sessionID, kind: "message"|"turn"|"tool_call"|"tool_result"|"steer"}
//   → steer {sessionID, text}
//
// Config is at [collab] in reasonix.toml.
package collab

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"reasonix/internal/netclient"
)

// Config is the [collab] TOML section.
type Config struct {
	Enabled    bool   `toml:"enabled"`
	ListenAddr string `toml:"listen_addr"` // e.g. ":9091"
}

// Role is the participant role.
type Role string

const (
	RoleOwner   Role = "owner"
	RoleWatcher Role = "watcher"
)

// Message is a protocol message over WebSocket.
type Message struct {
	Type      string `json:"type"` // subscribe, event, steer
	SessionID string `json:"sessionId,omitempty"`
	Role      Role   `json:"role,omitempty"`
	Event     *Event `json:"event,omitempty"`
	Text      string `json:"text,omitempty"`
}

// Event is a session event broadcast to watchers.
type Event struct {
	Kind      string `json:"kind"` // message, turn_start, turn_end, tool_call, tool_result, steer
	SessionID string `json:"sessionId"`
	Turn      int    `json:"turn,omitempty"`
	Tool      string `json:"tool,omitempty"`
	ToolArgs  string `json:"toolArgs,omitempty"`
	Text      string `json:"text,omitempty"`
	Sender    string `json:"sender,omitempty"`
	Timestamp int64  `json:"timestamp"`
}

// Peer is a connected remote instance.
type Peer struct {
	conn     *websocket.Conn
	role     Role
	sessions map[string]bool
	mu       sync.Mutex
}

// SteerCallback is called when a watcher sends a steer command.
// The controller wires this to inject the text as a mid-turn steer.
type SteerCallback func(sessionID, text string)

// Hub manages WebSocket connections and event broadcasting.
type Hub struct {
	mu       sync.RWMutex
	peers    map[*Peer]bool
	sessions map[string]*peerSet // sessionID → connected peers
	onSteer  SteerCallback
	server   *http.Server
	logger   *slog.Logger
	upgrader websocket.Upgrader
}

type peerSet struct {
	peers map[*Peer]bool
}

// New creates a Hub. Returns nil when disabled.
func New(cfg Config, onSteer SteerCallback, logger *slog.Logger) *Hub {
	if !cfg.Enabled {
		return nil
	}
	if cfg.ListenAddr == "" {
		cfg.ListenAddr = ":9091"
	}
	if logger == nil {
		logger = slog.Default()
	}
	h := &Hub{
		peers:    make(map[*Peer]bool),
		sessions: make(map[string]*peerSet),
		onSteer:  onSteer,
		logger:   logger.With("component", "collab"),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", h.handleWS)
	h.server = &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	return h
}

// Start begins listening for WebSocket connections.
func (h *Hub) Start() error {
	if h == nil {
		return nil
	}
	h.logger.Info("collaboration hub starting", "addr", h.server.Addr)
	go func() {
		if err := h.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			h.logger.Error("collab hub listen error", "err", err)
		}
	}()
	return nil
}

// Stop shuts down the hub.
func (h *Hub) Stop() error {
	if h == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return h.server.Shutdown(ctx)
}

// Broadcast sends an event to all watchers of a session.
func (h *Hub) Broadcast(sessionID string, ev Event) {
	if h == nil {
		return
	}
	h.mu.RLock()
	ps := h.sessions[sessionID]
	h.mu.RUnlock()
	if ps == nil {
		return
	}

	ev.Timestamp = time.Now().UnixMilli()
	msg := Message{Type: "event", SessionID: sessionID, Event: &ev}
	raw, err := json.Marshal(msg)
	if err != nil {
		return
	}

	// Collect peers under lock
	h.mu.RLock()
	var targets []*Peer
	for p := range ps.peers {
		targets = append(targets, p)
	}
	h.mu.RUnlock()

	for _, p := range targets {
		p.mu.Lock()
		if p.conn != nil {
			_ = p.conn.WriteMessage(websocket.TextMessage, raw)
		}
		p.mu.Unlock()
	}
}

// SessionWatchers returns the number of watchers for a session.
func (h *Hub) SessionWatchers(sessionID string) int {
	if h == nil {
		return 0
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	ps := h.sessions[sessionID]
	if ps == nil {
		return 0
	}
	return len(ps.peers)
}

// ActiveSessions returns the sessions with at least one watcher.
func (h *Hub) ActiveSessions() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	var out []string
	for id, ps := range h.sessions {
		if len(ps.peers) > 0 {
			out = append(out, id)
		}
	}
	return out
}

func (h *Hub) handleWS(w http.ResponseWriter, r *http.Request) {
	if h.logger == nil {
		h.logger = slog.Default()
	}
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Warn("collab: upgrade failed", "err", err)
		return
	}

	peer := &Peer{
		conn:     conn,
		sessions: make(map[string]bool),
	}
	h.mu.Lock()
	h.peers[peer] = true
	h.mu.Unlock()

	defer func() {
		h.removePeer(peer)
		conn.Close()
	}()

	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				h.logger.Warn("collab: read error", "err", err)
			}
			return
		}

		var msg Message
		if err := json.Unmarshal(raw, &msg); err != nil {
			h.logger.Warn("collab: bad message", "err", err)
			continue
		}

		switch msg.Type {
		case "subscribe":
			h.handleSubscribe(peer, msg)
		case "steer":
			if h.onSteer != nil {
				h.onSteer(msg.SessionID, msg.Text)
			}
		}
	}
}

func (h *Hub) handleSubscribe(peer *Peer, msg Message) {
	sid := strings.TrimSpace(msg.SessionID)
	if sid == "" {
		return
	}

	peer.mu.Lock()
	peer.role = msg.Role
	peer.sessions[sid] = true
	peer.mu.Unlock()

	h.mu.Lock()
	if h.sessions[sid] == nil {
		h.sessions[sid] = &peerSet{peers: make(map[*Peer]bool)}
	}
	h.sessions[sid].peers[peer] = true
	h.mu.Unlock()

	h.logger.Info("collab: peer subscribed", "session", sid, "role", msg.Role)
}

func (h *Hub) removePeer(peer *Peer) {
	h.mu.Lock()
	delete(h.peers, peer)
	for _, ps := range h.sessions {
		delete(ps.peers, peer)
	}
	h.mu.Unlock()
}

// EchoWSHandler returns an http.Handler that upgrades to WebSocket and echoes
// messages back — useful for testing and health checks.
func EchoWSHandler() http.Handler {
	up := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		for {
			mt, raw, err := conn.ReadMessage()
			if err != nil {
				return
			}
			_ = conn.WriteMessage(mt, raw)
		}
	})
}

// Ensure netclient import is used.
var _ = netclient.DefaultClient
