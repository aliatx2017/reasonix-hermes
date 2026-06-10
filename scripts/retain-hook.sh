#!/usr/bin/env bash
# retain-hook.sh — PreToolUse hook that sends context to the Hindsight memory server.
#
# Called by Reasonix before each tool use. Reads the tool name and input from
# stdin (JSON payload from the hook runner) and POSTs a retain request to the
# memory server if the tool is worth remembering.
#
# Environment:
#   HINDSIGHT_URL  — memory server URL (default: http://localhost:8080)
#   HINDSIGHT_KEY  — API key if MEMORY_API_KEY is set on the server
#   HINDSIGHT_TIMEOUT — curl timeout in seconds (default: 5)
#
# Install in .reasonix/settings.json:
#   {
#     "hooks": {
#       "PreToolUse": [
#         {
#           "match": ".*",
#           "command": "bash /path/to/retain-hook.sh",
#           "description": "Auto-retain tool context in Hindsight memory"
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
  echo "[retain-hook] curl not found in PATH — skipping" >&2
  exit 0
fi

if ! command -v python3 &>/dev/null; then
  echo "[retain-hook] python3 not found in PATH — skipping" >&2
  exit 0
fi

# ── Read hook payload from stdin ──────────────────────────────────

PAYLOAD=$(cat)

# Extract tool name from payload
TOOL_NAME=$(echo "$PAYLOAD" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('tool_name',''))" 2>/dev/null || echo "")

# Skip noise — only retain meaningful tools
case "$TOOL_NAME" in
  read_file|write_file|edit_file|bash|search|glob) exit 0 ;;
  "") exit 0 ;;
esac

# ── Build retain request ─────────────────────────────────────────

REQUEST=$(python3 -c "
import sys, json
try:
    d = json.load(sys.stdin)
    tool = d.get('tool_name', 'unknown')
    inp = d.get('tool_input', {})
    content = f'{tool}: {json.dumps(inp)}' if inp else f'{tool} (no input)'
    print(json.dumps({
        'jsonrpc': '2.0',
        'id': 1,
        'method': 'tools/call',
        'params': {
            'name': 'hindsight_retain',
            'arguments': {
                'content': content,
                'tags': ['tool_use', tool]
            }
        }
    }))
except Exception:
    sys.exit(1)
" <<< "$PAYLOAD" 2>/dev/null) || {
  echo "[retain-hook] failed to build retain request — skipping" >&2
  exit 0
}

# ── Send to memory server ─────────────────────────────────────────

HEADERS=(-H "Content-Type: application/json")
if [[ -n "$HINDSIGHT_KEY" ]]; then
  HEADERS+=(-H "Authorization: Bearer $HINDSIGHT_KEY")
fi

curl -sf --max-time "$HINDSIGHT_TIMEOUT" -X POST "$HINDSIGHT_URL/mcp" "${HEADERS[@]}" -d "$REQUEST" >/dev/null 2>&1 || {
  echo "[retain-hook] failed to reach memory server at $HINDSIGHT_URL — skipping" >&2
  exit 0
}
exit 0