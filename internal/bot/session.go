package bot

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// BuildSessionKey generates a stable session key following the Hermes model:
//   - DM: isolated by chat (shared history within the same DM conversation)
//   - Group: isolated by user (each person gets their own session)
//   - Thread: shared (everyone in the thread shares context)
func BuildSessionKey(src SessionSource) string {
	var scope string
	switch src.ChatType {
	case ChatDM:
		scope = fmt.Sprintf("%s:dm:%s", src.Platform, src.ChatID)
	case ChatGroup:
		scope = fmt.Sprintf("%s:group:%s:%s", src.Platform, src.ChatID, src.UserID)
	case ChatGuild:
		scope = fmt.Sprintf("%s:guild:%s:%s", src.Platform, src.ChatID, src.UserID)
	case ChatDirect:
		scope = fmt.Sprintf("%s:direct:%s", src.Platform, src.ChatID)
	case ChatThread:
		threadID := src.ThreadID
		if threadID == "" {
			threadID = src.ChatID
		}
		scope = fmt.Sprintf("%s:thread:%s", src.Platform, threadID)
	default:
		scope = fmt.Sprintf("%s:%s:%s:%s", src.Platform, src.ChatType, src.ChatID, src.UserID)
	}
	h := sha256.Sum256([]byte(scope))
	return hex.EncodeToString(h[:])[:16]
}

// slashCommands is the set of slash commands that bypass the busy queue and
// are handled immediately even when a session is actively running a turn.
var slashCommands = map[string]bool{
	"/stop":    true,
	"/new":     true,
	"/reset":   true,
	"/approve": true,
	"/deny":    true,
	"/answer":  true,
	"/status":  true,
	"/goal":    true,
	"/model":   true,
	"/help":    true,
}

// IsSlashBypass returns whether the message text is a slash command that
// should bypass the session's busy queue.
func IsSlashBypass(text string) bool {
	if len(text) == 0 {
		return false
	}
	cmd := text
	for i, r := range text {
		if r == ' ' {
			cmd = text[:i]
			break
		}
	}
	return slashCommands[cmd]
}

// pendingTurn is a queued turn waiting to execute.
type pendingTurn struct {
	msg       InboundMessage
	timestamp time.Time
}

// SessionManager provides session-level concurrency control: at most one
// task runs per session at a time. Messages for a busy session are queued
// and merged within the debounce window.
type SessionManager struct {
	mu       sync.Mutex
	active   map[string]bool          // session key -> currently running
	pending  map[string][]pendingTurn // session key -> queued turns
	debounce time.Duration
}

// NewSessionManager creates a new session manager. debounce is the window
// within which consecutive messages from the same session are merged.
func NewSessionManager(debounce time.Duration) *SessionManager {
	if debounce <= 0 {
		debounce = 1500 * time.Millisecond
	}
	return &SessionManager{
		active:   make(map[string]bool),
		pending:  make(map[string][]pendingTurn),
		debounce: debounce,
	}
}

// TryAcquire attempts to acquire the session lock. If the session is busy
// and the message is not a bypass command, it is queued and TryAcquire
// returns (false, true) — merged is true when the message was coalesced
// with the last queued message within the debounce window.
func (sm *SessionManager) TryAcquire(key string, msg InboundMessage) (acquired bool, merged bool) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.active[key] {
		// Bypass commands are handled immediately (caller processes directly).
		if IsSlashBypass(msg.Text) {
			return true, false
		}
		// Queue the message. If it arrives within the debounce window
		// after the last queued message from the same session, merge
		// their text (consecutive input combined into one turn).
		queue := sm.pending[key]
		if len(queue) > 0 {
			last := &queue[len(queue)-1]
			if msg.Text != "" && time.Since(last.timestamp) < sm.debounce {
				// Merge: replace the last message's text.
				if last.msg.Text != "" {
					last.msg.Text = last.msg.Text + "\n" + msg.Text
				} else {
					last.msg.Text = msg.Text
				}
				last.timestamp = time.Now()
				return false, true
			}
		}
		queue = append(queue, pendingTurn{msg: msg, timestamp: time.Now()})
		sm.pending[key] = queue
		return false, true
	}

	sm.active[key] = true
	return true, false
}

// Release releases the session lock and returns the next merged message
// from the queue. If the queue is empty the session is fully released.
func (sm *SessionManager) Release(key string) *InboundMessage {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	queue := sm.pending[key]
	if len(queue) == 0 {
		delete(sm.active, key)
		delete(sm.pending, key)
		return nil
	}

	// Drain the queue, merging all messages into one.
	var merged *InboundMessage
	for i := range queue {
		if merged == nil {
			m := queue[i].msg
			merged = &m
		} else {
			if queue[i].msg.Text != "" {
				merged.Text = merged.Text + "\n" + queue[i].msg.Text
			}
		}
	}
	delete(sm.pending, key)
	// active stays true — the caller will start a new turn immediately
	// with the merged message.
	return merged
}

// IsActive returns whether the session has a running task.
func (sm *SessionManager) IsActive(key string) bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	return sm.active[key]
}

// ActiveCount returns the number of currently active sessions.
func (sm *SessionManager) ActiveCount() int {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	return len(sm.active)
}

// ForceRelease forcefully releases the session (for session close or error
// recovery).
func (sm *SessionManager) ForceRelease(key string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.active, key)
	delete(sm.pending, key)
}
