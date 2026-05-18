#!/usr/bin/env bash
# Sync SQLite between local machine and remote AMD server.
# Usage:
#   ./scripts/db-sync.sh push user@host:/path/to/mkaoqin01
#   ./scripts/db-sync.sh pull user@host:/path/to/mkaoqin01
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DB_LOCAL="${DB_LOCAL:-$ROOT/data/intervoice.db}"
ACTION="${1:-}"
REMOTE_SPEC="${2:-}"

usage() {
  echo "Usage: $0 push|pull user@host:/path/to/mkaoqin01"
  exit 1
}

[[ -n "$ACTION" && -n "$REMOTE_SPEC" ]] || usage
[[ "$ACTION" == push || "$ACTION" == pull" ]] || usage

if [[ "$REMOTE_SPEC" =~ ^([^@]+@[^:]+):(.+)$ ]]; then
  SSH_TARGET="${BASH_REMATCH[1]}"
  REMOTE_DIR="${BASH_REMATCH[2]}"
else
  echo "REMOTE_PATH must be user@host:/absolute/path"
  exit 1
fi

REMOTE_DB="${REMOTE_DIR%/}/data/intervoice.db"
STAMP="$(date +%Y%m%d-%H%M%S)"

if [[ "$ACTION" == push ]]; then
  [[ -f "$DB_LOCAL" ]] || { echo "Missing local DB: $DB_LOCAL"; exit 1; }
  echo "==> Backup remote DB (if exists)"
  ssh "$SSH_TARGET" "mkdir -p '${REMOTE_DIR}/data' && test -f '${REMOTE_DB}' && cp '${REMOTE_DB}' '${REMOTE_DB}.bak-${STAMP}' || true"
  echo "==> Stop remote stack"
  ssh "$SSH_TARGET" "cd '${REMOTE_DIR}' && docker compose -f docker-compose.prod.yml down 2>/dev/null || true"
  echo "==> Upload $DB_LOCAL"
  scp "$DB_LOCAL" "${SSH_TARGET}:${REMOTE_DB}"
  echo "==> Start remote stack"
  ssh "$SSH_TARGET" "cd '${REMOTE_DIR}' && docker compose -f docker-compose.prod.yml up -d"
  echo "Done."
elif [[ "$ACTION" == pull ]]; then
  mkdir -p "$(dirname "$DB_LOCAL")"
  [[ -f "$DB_LOCAL" ]] && cp "$DB_LOCAL" "${DB_LOCAL}.bak-${STAMP}"
  echo "==> Download from server"
  scp "${SSH_TARGET}:${REMOTE_DB}" "$DB_LOCAL"
  echo "Done. Local backup: ${DB_LOCAL}.bak-${STAMP} (if existed)"
fi
