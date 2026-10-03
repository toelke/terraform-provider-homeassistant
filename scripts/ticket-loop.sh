#!/usr/bin/env bash
# Implement tickets one after another, each in a fresh headless Claude session.
# Streams a readable trace (agent text + tool calls) to stdout; the full JSON
# stream of each session is kept in .loop-logs/.
# Usage: scripts/ticket-loop.sh            (run 2-3 copies in parallel, started a few seconds apart)
# Env:   IDLE_SLEEP       seconds to wait when no ticket is ready (default 1800)
#        PERMISSION_MODE  claude --permission-mode (default auto)
set -euo pipefail

cd "$(git -C "$(dirname "$0")" rev-parse --show-toplevel)"
idle_sleep="${IDLE_SLEEP:-1800}"
mode="${PERMISSION_MODE:-auto}"
mkdir -p .loop-logs

pretty='
  if .type == "assistant" then
    (.message.content[]? |
      if .type == "text" then .text + "\n"
      elif .type == "tool_use" then
        "  → \(.name) \((.input.command // .input.file_path // .input.description // .input.pattern // "") | tostring | gsub("\n"; " ") | .[0:140])\n"
      else empty end)
  elif .type == "result" then
    "=== result: \(.subtype) · \(.num_turns // "?") turns · $\(.total_cost_usd // "?")\n"
  else empty end'

while true; do
  git fetch -q origin
  log=".loop-logs/$(date +%Y%m%d-%H%M%S)-$$.jsonl"
  echo "=== $(date -Is) starting a session (log: $log)"
  claude -p --permission-mode "$mode" --output-format stream-json --verbose \
    "Follow AGENTS.md: first tend every open PR (rebase onto origin/main, resolve conflicts,
address all review comments, get CI green), then take the next ticket in todo and implement it.
You cannot ask the human in this session: put questions as PR comments instead.
If no new ticket is ready after tending the PRs, end your reply with exactly NO_TICKET_READY." \
    | tee "$log" | jq -rj --unbuffered "$pretty" || true
  if jq -r 'select(.type == "result") | .result // ""' "$log" | grep -q NO_TICKET_READY; then
    echo "=== no ticket ready; sleeping ${idle_sleep}s (merge PRs to unblock)"
    sleep "$idle_sleep"
  fi
done
