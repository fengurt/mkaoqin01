#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
PORTS=(8010 8001 8002 8003 8004 5173)

stop_port_if_workspace_process() {
  local port="$1"
  while IFS= read -r pid; do
    [ -z "$pid" ] && continue
    local cwd
    cwd="$(lsof -a -p "$pid" -d cwd -Fn 2>/dev/null | sed -n 's/^n//p' | head -n 1 || true)"
    if [[ "$cwd" == "$ROOT_DIR"* ]]; then
      kill "$pid" || true
    fi
  done < <(lsof -nP -iTCP:"$port" -sTCP:LISTEN -t 2>/dev/null || true)
}

for port in "${PORTS[@]}"; do
  stop_port_if_workspace_process "$port"
done

mkdir -p "$ROOT_DIR/.run"

( cd "$ROOT_DIR/app/auth" && go run . > "$ROOT_DIR/.run/auth.log" 2>&1 ) &
( cd "$ROOT_DIR/app/attendance" && go run . > "$ROOT_DIR/.run/attendance.log" 2>&1 ) &
( cd "$ROOT_DIR/app/voice" && go run . > "$ROOT_DIR/.run/voice.log" 2>&1 ) &
( cd "$ROOT_DIR/app/admin" && go run . > "$ROOT_DIR/.run/admin.log" 2>&1 ) &
( cd "$ROOT_DIR/app/gateway" && go run . > "$ROOT_DIR/.run/gateway.log" 2>&1 ) &
( cd "$ROOT_DIR/frontend" && npm run dev > "$ROOT_DIR/.run/frontend.log" 2>&1 ) &

sleep 2

echo "Gateway:  http://localhost:8010"
echo "Frontend: http://localhost:5173"
open "http://localhost:5173" || true
