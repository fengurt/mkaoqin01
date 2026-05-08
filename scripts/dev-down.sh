#!/usr/bin/env bash
set -euo pipefail

PORTS=(8010 8001 8002 8003 8004 5173)
for port in "${PORTS[@]}"; do
  pids=$(lsof -nP -iTCP:"$port" -sTCP:LISTEN -t 2>/dev/null || true)
  if [[ -n "$pids" ]]; then
    echo "$pids" | xargs kill || true
  fi
done
