#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
PORTS=(8010 8001 8002 8003 8004 5173)
GATEWAY_HEALTH_URL="${GATEWAY_HEALTH_URL:-http://127.0.0.1:8010/healthz}"

echo "== Intervoice dev stack =="
echo "Root: $ROOT_DIR"

port_listeners() {
  lsof -nP -iTCP:"$1" -sTCP:LISTEN 2>/dev/null || true
}

stop_port_if_workspace_process() {
  local port="$1"
  while IFS= read -r pid; do
    [ -z "$pid" ] && continue
    local cwd
    cwd="$(lsof -a -p "$pid" -d cwd -Fn 2>/dev/null | sed -n 's/^n//p' | head -n 1 || true)"
    if [[ "$cwd" == "$ROOT_DIR"* ]]; then
      echo "  Stopping PID $pid on port $port (cwd under workspace)"
      kill "$pid" || true
    fi
  done < <(lsof -nP -iTCP:"$port" -sTCP:LISTEN -t 2>/dev/null || true)
}

echo "-- Ports before cleanup --"
for port in "${PORTS[@]}"; do
  if port_listeners "$port" | grep -q .; then
    echo "[$port] busy:"
    port_listeners "$port" | sed 's/^/  /'
  else
    echo "[$port] free"
  fi
done

for port in "${PORTS[@]}"; do
  stop_port_if_workspace_process "$port"
done

sleep 1

echo "-- After workspace cleanup --"
BLOCKED=()
for port in "${PORTS[@]}"; do
  if port_listeners "$port" | grep -q .; then
    BLOCKED+=("$port")
    echo "[$port] STILL BUSY (not started from this repo — free it manually):"
    port_listeners "$port" | sed 's/^/  /'
  fi
done

if ((${#BLOCKED[@]} > 0)); then
  echo ""
  echo "Refusing to start: ports ${BLOCKED[*]} are still in use."
  echo "Stop those processes or close other apps using those ports, then re-run."
  exit 1
fi

mkdir -p "$ROOT_DIR/.run"

( cd "$ROOT_DIR/app/auth" && exec go run . > "$ROOT_DIR/.run/auth.log" 2>&1 ) &
( cd "$ROOT_DIR/app/attendance" && exec go run . > "$ROOT_DIR/.run/attendance.log" 2>&1 ) &
( cd "$ROOT_DIR/app/voice" && exec go run . > "$ROOT_DIR/.run/voice.log" 2>&1 ) &
( cd "$ROOT_DIR/app/admin" && exec go run . > "$ROOT_DIR/.run/admin.log" 2>&1 ) &
( cd "$ROOT_DIR/app/gateway" && exec go run . > "$ROOT_DIR/.run/gateway.log" 2>&1 ) &
( cd "$ROOT_DIR/frontend" && exec npm run dev > "$ROOT_DIR/.run/frontend.log" 2>&1 ) &

echo "Started auth :8001, attendance :8002, voice :8003, admin :8004, gateway :8010, Vite :5173"
echo "Logs: $ROOT_DIR/.run/*.log"

echo -n "Waiting for gateway "
READY=""
for _ in $(seq 1 60); do
  if curl -sf "$GATEWAY_HEALTH_URL" >/dev/null 2>&1; then
    READY="1"
    echo " OK"
    break
  fi
  echo -n "."
  sleep 0.5
done
if [[ -z "$READY" ]]; then
  echo ""
  echo "Gateway did not become ready in time. Check $ROOT_DIR/.run/gateway.log"
  exit 1
fi

echo ""
echo "Gateway:  http://localhost:8010/healthz"
echo "Frontend: http://localhost:5173"
echo "Tip: after changing gateway/admin routes, re-run this script so ports match this workspace."
open "http://localhost:5173" || true
