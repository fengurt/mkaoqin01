#!/usr/bin/env bash
# Production deploy on AMD64 Linux (run on the server inside repo root).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

if [[ ! -f .env ]]; then
  echo "Copy .env.example to .env and set JWT_SECRET (and AI keys) first."
  exit 1
fi

mkdir -p data

echo "==> Pull latest main"
git pull origin main

echo "==> Build and start production stack"
docker compose -f docker-compose.prod.yml up --build -d

echo "==> Status"
docker compose -f docker-compose.prod.yml ps

PORT="${HTTP_PORT:-80}"
if curl -sf "http://127.0.0.1:${PORT}/healthz" >/dev/null 2>&1; then
  echo "OK: http://127.0.0.1:${PORT}/healthz"
elif curl -sf "http://127.0.0.1:8010/healthz" >/dev/null 2>&1; then
  echo "OK: gateway http://127.0.0.1:8010/healthz"
else
  echo "WARN: health check failed — check docker compose logs"
fi
