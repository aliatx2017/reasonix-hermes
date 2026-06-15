package collab

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// --- Config / New ---

func TestNew_Disabled(t *testing.T) {
	t.Parallel()
	h := New(Config{Enabled: false}, nil, nil)
	if h != nil {
		t.Error("New() should return nil when disabled")
	}
}

func TestNew_NoListenAddrDefaults(t *testing.T) {
	t.Parallel()
	h := New(Config{Enabled: true, ListenAddr: ""}, nil, nil)
	if h == nil {
		t.Fatal("expected non-nil hub")
	}
	if h.server.Addr != ":9091" {
		t.Errorf("listen addr = %q, want :9091", h.server.Addr)
	}
}

func TestNew_UsesGivenAddr(t *testing.T) {
	t.Parallel()
	h := New(Config{Enabled: true, ListenAddr: ":9876"}, nil, nil)
	if h == nil {
		t.Fatal("expected non-nil hub")
	}
	if h.server.Addr != ":9876" {
		t.Errorf("listen addr = %q, want :9876", h.server.Addr)
	}
}

// --- Subscribe + Broadcast ---

func TestSubscribeAndBroadcast(t *testing.T) {
	t.Parallel()
	h := &Hub{
		peers:    make(map[*Peer]bool),
		sessions: make(map[string]*peerSet),
		upgrader: websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }},
		logger:   slog.Default(),
	}

	srv := httptest.NewServer(http.HandlerFunc(h.handleWS))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	// Subscribe as watcher
	sub := Message{Type: "subscribe", SessionID: "s1", Role: RoleWatcher}
	raw, _ := json.Marshal(sub)
	if err := conn.WriteMessage(websocket.TextMessage, raw); err != nil {
		t.Fatalf("write subscribe: %v", err)
	}
	time.Sleep(50 * time.Millisecond) // let hub process

	// Broadcast
	ev := Event{Kind: "message", SessionID: "s1", Text: "hello watcher"}
	h.Broadcast("s1", ev)

	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, raw, err = conn.ReadMessage()
	if err != nil {
		t.Fatalf("read event: %v", err)
	}

	var msg Message
	if err := json.Unmarshal(raw, &msg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if msg.Event == nil || msg.Event.Text != "hello watcher" {
		t.Errorf("event text = %v, want 'hello watcher'", msg.Event)
	}
}

// --- SessionWatchers ---

func TestSessionWatchers(t *testing.T) {
	t.Parallel()
	h := &Hub{
		peers:    make(map[*Peer]bool),
		sessions: make(map[string]*peerSet),
	}
	if got := h.SessionWatchers("nonexistent"); got != 0 {
		t.Errorf("watchers = %d, want 0", got)
	}

	p1 := &Peer{role: RoleWatcher}
	p2 := &Peer{role: RoleWatcher}
	h.sessions["s1"] = &peerSet{peers: map[*Peer]bool{p1: true, p2: true}}
	if got := h.SessionWatchers("s1"); got != 2 {
		t.Errorf("watchers = %d, want 2", got)
	}
}

// --- ActiveSessions ---

func TestActiveSessions(t *testing.T) {
	t.Parallel()
	h := &Hub{
		peers:    make(map[*Peer]bool),
		sessions: make(map[string]*peerSet),
	}
	p := &Peer{role: RoleWatcher}
	h.sessions["s1"] = &peerSet{peers: map[*Peer]bool{p: true}}
	h.sessions["s2"] = &peerSet{peers: map[*Peer]bool{}}

	active := h.ActiveSessions()
	if len(active) != 1 || active[0] != "s1" {
		t.Errorf("ActiveSessions = %v, want [s1]", active)
	}
}

// --- SteerCallback ---

func TestSteerCallback(t *testing.T) {
	t.Parallel()
	var steers []string
	h := New(Config{Enabled: true, ListenAddr: ":0"}, func(sid, text string) {
		steers = append(steers, sid+":"+text)
	}, nil)
	if h == nil {
		t.Fatal("hub is nil")
	}
	h.onSteer("s1", "check Go files")
	if len(steers) != 1 || steers[0] != "s1:check Go files" {
		t.Errorf("steers = %v, want [s1:check Go files]", steers)
	}
}

// --- Start / Stop ---

func TestStartStop(t *testing.T) {
	t.Parallel()
	h := New(Config{Enabled: true, ListenAddr: ":19998"}, nil, nil)
	if h == nil {
		t.Fatal("hub is nil")
	}
	if err := h.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	time.Sleep(50 * time.Millisecond)
	if err := h.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}
}

// --- Nil hub ---

func TestNilHub(t *testing.T) {
	t.Parallel()
	var h *Hub
	if err := h.Start(); err != nil {
		t.Errorf("nil.Start: %v", err)
	}
	if err := h.Stop(); err != nil {
		t.Errorf("nil.Stop: %v", err)
	}
	h.Broadcast("s1", Event{}) // must not panic
	if h.SessionWatchers("s1") != 0 {
		t.Error("nil hub reports 0 watchers")
	}
}

func TestRapidStartStop(t *testing.T) {
	t.Parallel()
	cfg := Config{Enabled: true, ListenAddr: "127.0.0.1:0"}
	h := New(cfg, nil, nil)
	for i := 0; i < 3; i++ {
		if err := h.Start(); err != nil {
			t.Fatalf("Start cycle %d: %v", i, err)
		}
		if err := h.Stop(); err != nil {
			t.Fatalf("Stop cycle %d: %v", i, err)
		}
	}
	// Should still be functional after cycles
	if err := h.Start(); err != nil {
		t.Fatalf("final Start: %v", err)
	}
	defer h.Stop()
	if h.SessionWatchers("any") != 0 {
		t.Error("no watchers expected after fresh start")
	}
}

func TestBroadcastNoWatchersNoPanic(t *testing.T) {
	t.Parallel()
	cfg := Config{Enabled: true, ListenAddr: "127.0.0.1:0"}
	h := New(cfg, nil, nil)
	if err := h.Start(); err != nil {
		t.Fatal(err)
	}
	defer h.Stop()
	// Broadcast to session with no watchers — must not panic
	h.Broadcast("no-watchers", Event{Kind: "orphan"})
}

func TestActiveSessionsEmpty(t *testing.T) {
	t.Parallel()
	cfg := Config{Enabled: true, ListenAddr: "127.0.0.1:0"}
	h := New(cfg, nil, nil)
	if err := h.Start(); err != nil {
		t.Fatal(err)
	}
	defer h.Stop()
	sessions := h.ActiveSessions()
	if len(sessions) != 0 {
		t.Errorf("expected 0 active sessions, got %d", len(sessions))
	}
}
