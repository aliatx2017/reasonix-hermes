#!/usr/bin/env bash
# reflect-hook.sh — Stop hook that triggers session reflection in Hindsight.
#
# Called by Reasonix when a session ends. Sends a reflect request summarizing
# what happened during the session.
#
# Environment:
#   HINDSIGHT_URL  — memory server URL (default: http://localhost:8080)
#   HINDSIGHT_KEY  — API key if MEMORY_API_KEY is set on the server
#   HINDSIGHT_TIMEOUT — curl timeout in seconds (default: 5)
#
# Install in .reasonix/settings.json:
#   {
#     "hooks": {
#       "Stop": [
#         {
#           "command": "bash /path/to/reflect-hook.sh",
#           "description": "Auto-reflect on session in Hindsight memory"
#         }
#       ]
#     }
#   }

set -euo pipefail

HINDSIGHT_URL="${HINDSIGHT_URL:-http://localhost:8080}"
HINDSIGHT_KEY="${HINDSIGHT_KEY:-}"
HINDSIGHT_TIMEOUT="${HINDSIGHT_TIMEOUT:-5}"

# ── Dependency checks ────────────────────────────────────────────

if ! command -v curl &>/dev/null; then
  echo "[reflect-hook] curl not found in PATH — skipping" >&2
  exit 0
fi

if ! command -v python3 &>/dev/null; then
  echo "[reflect-hook] python3 not found in PATH — skipping" >&2
  exit 0
fi

# ── Read hook payload from stdin ──────────────────────────────────

PAYLOAD=$(cat)

# Extract session ID or transcript hint
SESSION_HINT=$(echo "$PAYLOAD" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('session_id',''))" 2>/dev/null || echo "")

# ── Build reflect request ─────────────────────────────────────────

REQUEST=$(python3 -c "
import sys, json
try:
    sid = sys.argv[1] if len(sys.argv) > 1 else ''
    print(json.dumps({
        'jsonrpc': '2.0',
        'id': 1,
        'method': 'tools/call',
        'params': {
            'name': 'hindsight_reflect',
            'arguments': {
                'session': sid or 'latest',
                'query': 'session summary'
            }
        }
    }))
except Exception:
    sys.exit(1)
" "$SESSION_HINT" 2>/dev/null) || {
  echo "[reflect-hook] failed to build reflect request — skipping" >&2
  exit 0
}

# ── Send to memory server ─────────────────────────────────────────

HEADERS=(-H "Content-Type: application/json")
if [[ -n "$HINDSIGHT_KEY" ]]; then
  HEADERS+=(-H "Authorization: Bearer $HINDSIGHT_KEY")
fi

curl -sf --max-time "$HINDSIGHT_TIMEOUT" -X POST "$HINDSIGHT_URL/mcp" "${HEADERS[@]}" -d "$REQUEST" >/dev/null 2>&1 || {
  echo "[reflect-hook] failed to reach memory server at $HINDSIGHT_URL — skipping" >&2
  exit 0
}
exit 0