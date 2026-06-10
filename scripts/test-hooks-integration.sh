#!/usr/bin/env bash
# test-hooks-integration.sh — Integration tests for retain-hook.sh and reflect-hook.sh
#
# Spins up a fake Hindsight server (python3 HTTP), runs each hook against it,
# verifies the server received the expected JSON-RPC request.
#
# Usage: bash scripts/test-hooks-integration.sh

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
RETAIN_HOOK="$SCRIPT_DIR/retain-hook.sh"
REFLECT_HOOK="$SCRIPT_DIR/reflect-hook.sh"
PASS=0
FAIL=0
TOTAL=0

assert() {
  local desc="$1" actual="$2" expected="$3"
  TOTAL=$((TOTAL + 1))
  if [[ "$actual" == "$expected" ]]; then
    echo "  PASS: $desc"
    PASS=$((PASS + 1))
  else
    echo "  FAIL: $desc"
    echo "    expected: $expected"
    echo "    actual:   $actual"
    FAIL=$((FAIL + 1))
  fi
}

assert_contains() {
  local desc="$1" haystack="$2" needle="$3"
  TOTAL=$((TOTAL + 1))
  if [[ "$haystack" == *"$needle"* ]]; then
    echo "  PASS: $desc"
    PASS=$((PASS + 1))
  else
    echo "  FAIL: $desc"
    echo "    expected to contain: $needle"
    echo "    actual: $haystack"
    FAIL=$((FAIL + 1))
  fi
}

# ── Fake Hindsight server ─────────────────────────────────────────

FAKE_PORT=18765
FAKE_URL="http://127.0.0.1:$FAKE_PORT"
RECEIVED_FILE=""
FAKE_PID=""

start_fake_server() {
  RECEIVED_FILE=$(mktemp)
  : > "$RECEIVED_FILE"
  python3 -c "
import http.server, json, sys, os

class Handler(http.server.BaseHTTPRequestHandler):
    def do_POST(self):
        length = int(self.headers.get('Content-Length', 0))
        body = self.rfile.read(length).decode() if length else ''
        with open('$RECEIVED_FILE', 'a') as f:
            f.write(body + '\n')
        self.send_response(200)
        self.end_headers()
        self.wfile.write(b'{\"jsonrpc\":\"2.0\",\"id\":1,\"result\":{\"content\":[{\"type\":\"text\",\"text\":\"ok\"}]}}')
    def do_GET(self):
        self.send_response(200)
        self.end_headers()
        self.wfile.write(b'ok')
    def log_message(self, *a):
        pass  # suppress logs

srv = http.server.HTTPServer(('127.0.0.1', $FAKE_PORT), Handler)
srv.timeout = 5
# Handle up to 10 requests before shutting down
for _ in range(10):
    srv.handle_request()
" &
  FAKE_PID=$!
  # Wait for server to be ready
  for i in $(seq 1 30); do
    if curl -sf --max-time 1 "$FAKE_URL" &>/dev/null; then break; fi
    sleep 0.1
  done 2>/dev/null || true
}

stop_fake_server() {
  if [[ -n "$FAKE_PID" ]]; then
    kill "$FAKE_PID" 2>/dev/null || true
    wait "$FAKE_PID" 2>/dev/null || true
    FAKE_PID=""
  fi
}

get_received() {
  if [[ -f "$RECEIVED_FILE" ]]; then
    cat "$RECEIVED_FILE"
  else
    echo ""
  fi
}

# ── Test: retain-hook.sh ─────────────────────────────────────────

echo ""
echo "=== retain-hook.sh tests ==="
echo ""

# 1. Skip noise tools (read_file)
echo "--- Test: skip noise tools ---"
start_fake_server
HINDSIGHT_URL="$FAKE_URL" bash "$RETAIN_HOOK" <<< '{"tool_name":"read_file","tool_input":{"path":"main.go"}}' 2>&1 || true
sleep 0.2
received=$(get_received)
assert "noise tool skipped — no request sent" "$received" ""
stop_fake_server

# 2. Empty tool name
echo "--- Test: empty tool name ---"
start_fake_server
HINDSIGHT_URL="$FAKE_URL" bash "$RETAIN_HOOK" <<< '{"tool_name":"","tool_input":{}}' 2>&1 || true
sleep 0.2
received=$(get_received)
assert "empty tool name skipped — no request sent" "$received" ""
stop_fake_server

# 3. Meaningful tool sends retain request
echo "--- Test: meaningful tool sends retain ---"
start_fake_server
HINDSIGHT_URL="$FAKE_URL" bash "$RETAIN_HOOK" <<< '{"tool_name":"write_code","tool_input":{"file":"main.go","content":"package main"}}' 2>&1 || true
sleep 0.3
received=$(get_received)
assert_contains "retain request contains hindsight_retain" "$received" "hindsight_retain"
assert_contains "retain request contains write_code tag" "$received" "write_code"
stop_fake_server

# 4. Server unreachable — should skip gracefully with warning
echo "--- Test: server unreachable ---"
output=$(HINDSIGHT_URL="http://127.0.0.1:1" HINDSIGHT_TIMEOUT="1" bash "$RETAIN_HOOK" <<< '{"tool_name":"write_code","tool_input":{}}' 2>&1) || true
assert_contains "unreachable server prints warning" "$output" "failed to reach memory server"

# 5. Auth header sent when HINDSIGHT_KEY is set
echo "--- Test: auth header with key ---"
AUTH_RECEIVED=$(mktemp)
: > "$AUTH_RECEIVED"
AUTH_PORT=18766
python3 -c "
import http.server

class Handler(http.server.BaseHTTPRequestHandler):
    def do_POST(self):
        auth = self.headers.get('Authorization', '')
        length = int(self.headers.get('Content-Length', 0))
        body = self.rfile.read(length).decode() if length else ''
        with open('$AUTH_RECEIVED', 'a') as f:
            f.write(auth + '\n---\n' + body)
        self.send_response(200)
        self.end_headers()
        self.wfile.write(b'{\"jsonrpc\":\"2.0\",\"id\":1,\"result\":{\"content\":[{\"type\":\"text\",\"text\":\"ok\"}]}}')
    def log_message(self, *a):
        pass

srv = http.server.HTTPServer(('127.0.0.1', $AUTH_PORT), Handler)
srv.timeout = 5
for _ in range(5):
    srv.handle_request()
" &
  AUTH_PID=$!
  sleep 0.3
  HINDSIGHT_URL="http://127.0.0.1:$AUTH_PORT" HINDSIGHT_KEY="test-key-123" bash "$RETAIN_HOOK" <<< '{"tool_name":"write_code","tool_input":{}}' 2>&1 || true
  sleep 0.3
  auth_content=$(cat "$AUTH_RECEIVED")
  assert_contains "auth header present" "$auth_content" "Bearer test-key-123"
  kill "$AUTH_PID" 2>/dev/null || true
  wait "$AUTH_PID" 2>/dev/null || true

# 6. Malformed JSON payload
echo "--- Test: malformed JSON payload ---"
start_fake_server
output=$(HINDSIGHT_URL="$FAKE_URL" bash "$RETAIN_HOOK" <<< 'not-json-at-all' 2>&1) || true
sleep 0.2
received=$(get_received)
assert "malformed JSON skipped — no request sent" "$received" ""
stop_fake_server

# ── Test: reflect-hook.sh ─────────────────────────────────────────

echo ""
echo "=== reflect-hook.sh tests ==="
echo ""

# 1. Sends reflect request with session ID
echo "--- Test: sends reflect request ---"
start_fake_server
HINDSIGHT_URL="$FAKE_URL" bash "$REFLECT_HOOK" <<< '{"session_id":"sess-xyz"}' 2>&1 || true
sleep 0.3
received=$(get_received)
assert_contains "reflect request contains hindsight_reflect" "$received" "hindsight_reflect"
assert_contains "reflect request contains session id" "$received" "sess-xyz"
stop_fake_server

# 2. Server unreachable
echo "--- Test: server unreachable ---"
output=$(HINDSIGHT_URL="http://127.0.0.1:1" HINDSIGHT_TIMEOUT="1" bash "$REFLECT_HOOK" <<< '{"session_id":"abc"}' 2>&1) || true
assert_contains "unreachable server prints warning" "$output" "failed to reach memory server"

# 3. Empty session ID defaults to "latest"
echo "--- Test: empty session defaults to latest ---"
start_fake_server
HINDSIGHT_URL="$FAKE_URL" bash "$REFLECT_HOOK" <<< '{"session_id":""}' 2>&1 || true
sleep 0.3
received=$(get_received)
assert_contains "empty session uses 'latest'" "$received" "latest"
stop_fake_server

# 4. Malformed JSON payload — reflect-hook gracefully degrades to "latest"
echo "--- Test: malformed JSON defaults to latest ---"
start_fake_server
output=$(HINDSIGHT_URL="$FAKE_URL" bash "$REFLECT_HOOK" <<< 'garbage' 2>&1) || true
sleep 0.2
received=$(get_received)
assert_contains "malformed JSON defaults to latest" "$received" "latest"
stop_fake_server

# ── Summary ────────────────────────────────────────────────────────

echo ""
echo "=== Results: $PASS passed, $FAIL failed, $TOTAL total ==="

if [[ "$FAIL" -gt 0 ]]; then
  exit 1
fi
exit 0