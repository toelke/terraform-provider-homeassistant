#!/usr/bin/env bash
# Implement tickets one after another, each in a fresh headless Claude session.
# Usage: scripts/ticket-loop.sh            (run 2-3 copies in parallel, started a few seconds apart)
# Env:   IDLE_SLEEP  seconds to wait when no ticket is ready (default 1800)
#        PERMISSION_MODE  claude --permission-mode (default auto)
set -euo pipefail

cd "$(git -C "$(dirname "$0")" rev-parse --show-toplevel)"
idle_sleep="${IDLE_SLEEP:-1800}"
mode="${PERMISSION_MODE:-auto}"

while true; do
  git fetch -q origin
  echo "=== $(date -Is) starting a session"
  out=$(claude -p --permission-mode "$mode" \
    "Take the next ticket in todo and implement it, following AGENTS.md.
If no ticket is ready, reply with exactly NO_TICKET_READY and do nothing else.") || true
  echo "$out"
  if grep -q NO_TICKET_READY <<<"$out"; then
    echo "=== no ticket ready; sleeping ${idle_sleep}s (merge PRs to unblock)"
    sleep "$idle_sleep"
  fi
done
