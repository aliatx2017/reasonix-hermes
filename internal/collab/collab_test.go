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
	if h.server.Addr != "127.0.0.1:9091" {
		t.Errorf("listen addr = %q, want 127.0.0.1:9091", h.server.Addr)
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

// --- EchoWSHandler ---

func TestEchoWSHandler(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(EchoWSHandler())
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial echo: %v", err)
	}
	defer conn.Close()

	want := []byte("ping payload")
	if err := conn.WriteMessage(websocket.TextMessage, want); err != nil {
		t.Fatalf("write: %v", err)
	}
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, got, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read echo: %v", err)
	}
	if string(got) != string(want) {
		t.Errorf("echo = %q, want %q", got, want)
	}
}

// --- Token auth rejection ---

func TestHandleWS_TokenRejected(t *testing.T) {
	t.Parallel()
	h := &Hub{
		peers:    make(map[*Peer]bool),
		sessions: make(map[string]*peerSet),
		upgrader: websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }},
		logger:   slog.Default(),
		token:    "secret",
	}
	srv := httptest.NewServer(http.HandlerFunc(h.handleWS))
	defer srv.Close()

	// Upgrade without token — expect HTTP 401.
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/"
	_, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err == nil {
		t.Fatal("expected dial to fail with 401")
	}
	if resp == nil || resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %v, want 401", resp)
	}
}

// --- Bad JSON and steer message ---

func TestHandleWS_BadJSONIgnored(t *testing.T) {
	t.Parallel()
	h := &Hub{
		peers:    make(map[*Peer]bool),
		sessions: make(map[string]*peerSet),
		upgrader: websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }},
		logger:   slog.Default(),
	}
	srv := httptest.NewServer(http.HandlerFunc(h.handleWS))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	// Send malformed JSON — hub should log and continue, not close.
	if err := conn.WriteMessage(websocket.TextMessage, []byte("not json {")); err != nil {
		t.Fatalf("write bad json: %v", err)
	}
	time.Sleep(50 * time.Millisecond)

	// Send a valid subscribe to confirm connection is still alive.
	sub := Message{Type: "subscribe", SessionID: "alive", Role: RoleWatcher}
	raw, _ := json.Marshal(sub)
	if err := conn.WriteMessage(websocket.TextMessage, raw); err != nil {
		t.Fatalf("write after bad json: %v", err)
	}
	time.Sleep(50 * time.Millisecond)
	if h.SessionWatchers("alive") != 1 {
		t.Error("session should have 1 watcher after valid subscribe")
	}
}

func TestHandleWS_SteerCallback(t *testing.T) {
	t.Parallel()
	steered := make(chan string, 1)
	h := &Hub{
		peers:    make(map[*Peer]bool),
		sessions: make(map[string]*peerSet),
		upgrader: websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }},
		logger:   slog.Default(),
		onSteer: func(sid, text string) {
			steered <- sid + ":" + text
		},
	}
	srv := httptest.NewServer(http.HandlerFunc(h.handleWS))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	msg := Message{Type: "steer", SessionID: "s1", Text: "do something"}
	raw, _ := json.Marshal(msg)
	if err := conn.WriteMessage(websocket.TextMessage, raw); err != nil {
		t.Fatalf("write steer: %v", err)
	}

	select {
	case got := <-steered:
		if got != "s1:do something" {
			t.Errorf("steer = %q, want s1:do something", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("steer callback not called")
	}
}

// --- handleSubscribe with empty session ID ---

func TestHandleSubscribe_EmptySessionID(t *testing.T) {
	t.Parallel()
	h := &Hub{
		peers:    make(map[*Peer]bool),
		sessions: make(map[string]*peerSet),
		upgrader: websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }},
		logger:   slog.Default(),
	}
	srv := httptest.NewServer(http.HandlerFunc(h.handleWS))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	// Empty session ID subscribe — should be ignored.
	sub := Message{Type: "subscribe", SessionID: "   ", Role: RoleWatcher}
	raw, _ := json.Marshal(sub)
	if err := conn.WriteMessage(websocket.TextMessage, raw); err != nil {
		t.Fatalf("write: %v", err)
	}
	time.Sleep(50 * time.Millisecond)
	if len(h.sessions) != 0 {
		t.Errorf("expected 0 sessions, got %d", len(h.sessions))
	}
}

// --- Token authentication success ---

func TestHandleWS_TokenAccepted(t *testing.T) {
	t.Parallel()
	h := &Hub{
		peers:    make(map[*Peer]bool),
		sessions: make(map[string]*peerSet),
		upgrader: websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }},
		logger:   slog.Default(),
		token:    "mytoken",
	}
	srv := httptest.NewServer(http.HandlerFunc(h.handleWS))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/?token=mytoken"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial with valid token: %v", err)
	}
	defer conn.Close()

	sub := Message{Type: "subscribe", SessionID: "s-tok", Role: RoleWatcher}
	raw, _ := json.Marshal(sub)
	if err := conn.WriteMessage(websocket.TextMessage, raw); err != nil {
		t.Fatalf("write: %v", err)
	}
	time.Sleep(50 * time.Millisecond)
	if h.SessionWatchers("s-tok") != 1 {
		t.Error("expected 1 watcher after token-authenticated subscribe")
	}
}
