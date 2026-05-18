# AMD deployment notes (Linux x86_64)

Production stack: **Docker Compose** (`docker-compose.prod.yml`) — nginx → frontend + gateway → auth / attendance / voice / admin.  
SQLite: **`./data/intervoice.db`** (single file, shared volume).  
Quick deploy script: **`scripts/deploy-amd-prod.sh`**.

---

## Pre-deploy checklist

| Item | Action |
|------|--------|
| OS | Ubuntu/Debian AMD64, Docker Engine + Compose v2 |
| Repo | `git clone https://github.com/fengurt/mkaoqin01.git` |
| Secrets | `cp .env.example .env` — set **`JWT_SECRET`** (required); optional `ARK_API_KEY`, `SPEECH_API_KEY` |
| Data dir | `mkdir -p data data/uploads/fortune` |
| DB seed | Empty DB auto-migrates on first start; or copy local DB (see § Database sync) |
| Firewall | Allow `${HTTP_PORT:-80}` (and 443 if TLS in front) |
| Git | Deploy from **`main`** after merge; on server run `git pull origin main` before rebuild |

**Do not commit:** `.env`, `data/*.db`, `data/uploads/*`, local Go binaries under `app/*/`.

---

## 1. Server prerequisites

```bash
sudo apt-get update
sudo apt-get install -y git docker.io docker-compose-plugin curl
sudo usermod -aG docker "$USER"
# log out and back in so docker group applies
docker compose version
```

---

## 2. First-time install

```bash
git clone https://github.com/fengurt/mkaoqin01.git
cd mkaoqin01
git checkout main
cp .env.example .env
# edit .env — at minimum set JWT_SECRET to a long random string

mkdir -p data data/uploads/fortune
bash scripts/deploy-amd-prod.sh
```

Manual equivalent:

```bash
docker compose -f docker-compose.prod.yml up --build -d
docker compose -f docker-compose.prod.yml ps
curl -sS "http://127.0.0.1:${HTTP_PORT:-80}/healthz"
```

**Public URL:** `http://<server-ip>/`  
**API:** same origin `/v1/...` (proxied by nginx to gateway).

---

## 3. Environment variables (`.env`)

| Variable | Required | Notes |
|----------|----------|--------|
| `JWT_SECRET` | Yes | Same value for all services; changing it invalidates existing tokens |
| `HTTP_PORT` | No | Host port for nginx (default `80`) |
| `ARK_API_KEY` | No | Doubao / Ark LLM for voice NLU |
| `SPEECH_API_KEY` | No | OpenSpeech ASR (preferred over legacy `SPEECH_APP_ID` + `SPEECH_TOKEN`) |
| `DB_PATH` | — | Set inside compose to `/app/data/intervoice.db` (do not override unless you know paths) |
| `UPLOAD_ROOT` | — | Set in compose to `/app/data/uploads` for fortune images |

---

## 4. Persistence

| Path on host | Purpose |
|--------------|---------|
| `data/intervoice.db` | All app data (users, attendance, schedules, leads, badges, fortune metadata) |
| `data/uploads/fortune/` | Uploaded fortune poster images (`/uploads/fortune/...` via gateway) |

Compose mounts **`./data` → `/app/data`** on auth, attendance, voice, admin.  
**Back up `data/`** before major upgrades or DB sync.

---

## 5. Database sync (local Mac ↔ AMD server)

There is **one SQLite file per environment**, not live two-way replication. Details: [data/DATA_SYNC.md](../data/DATA_SYNC.md).

### Full database copy (recommended for initial prod seed)

**Always stop the stack before replacing the DB file.**

From your laptop (repo root):

```bash
# Push local DB → server
bash scripts/db-sync.sh push deploy@YOUR_SERVER:/opt/mkaoqin01

# Pull server DB → local
bash scripts/db-sync.sh pull deploy@YOUR_SERVER:/opt/mkaoqin01
```

Manual:

```bash
# Server
cd /opt/mkaoqin01 && docker compose -f docker-compose.prod.yml down
# Laptop
scp data/intervoice.db deploy@YOUR_SERVER:/opt/mkaoqin01/data/
# Server
docker compose -f docker-compose.prod.yml up -d
```

### Schedules only (no full DB overwrite)

1. Log in as **admin** on local or server.
2. **团队考勤** → **班次导出** (date range) → JSON file.
3. On target environment: **班次导入** the same JSON.  
   API: `GET /v1/admin/schedule/grid/export`, `POST /v1/admin/schedule/grid/import` (admin JWT).  
   Guide: `importdata/SCHEDULE_GRID_AGENT_GUIDE.md`.

### Selective row sync

- Admin JWT routes: `/v1/admin/data/users`, `/v1/admin/data/attendance`, upsert endpoints.
- Cursor MCP: `scripts/mcp-intervoice-admin` with `INTERVOICE_GATEWAY_URL` + `INTERVOICE_ADMIN_TOKEN`.

### What not to sync to production

- `seed_markers` (dev simulation) unless intentional.
- Demo `client_leads` with `source='DEMO'` if you excluded them locally.

---

## 6. Updates (after merge to `main`)

On the server:

```bash
cd /opt/mkaoqin01   # your clone path
git pull origin main
docker compose -f docker-compose.prod.yml up --build -d
```

Or: `bash scripts/deploy-amd-prod.sh`

Schema migrations run on each service start (`dbschema.ApplySQLite` + legacy migrators).  
If startup fails with **foreign key** errors, see DATA_SYNC.md § Post-apply integrity.

---

## 7. Post-deploy smoke test

| Check | How |
|-------|-----|
| Health | `curl -sS http://127.0.0.1/healthz` |
| Login | Admin `admin` / `123456a` (change in prod!) or employee `132369` / `123456a` |
| Team schedule | **我的** → 团队考勤 → tap **当前周** → roster grid with employees |
| Schedule I/O | 团队考勤 → 班次导出 (admin) returns JSON with `users` + `employees` |
| Gateway routes | After code changes, **rebuild all images** (`up --build`); stale gateway → **404** on new APIs |

---

## 8. Troubleshooting

| Symptom | Fix |
|---------|-----|
| New API returns **404** | `docker compose -f docker-compose.prod.yml up --build -d` (rebuild gateway + admin + attendance) |
| Export shows **员工 0** | Old admin binary; rebuild admin + gateway; or export merges users from `/v1/auth/users` in frontend |
| Fortune images broken | Ensure nginx has `/uploads/` proxy; `data/uploads/fortune` exists; `UPLOAD_ROOT=/app/data/uploads` on attendance/admin |
| SQLite locked | Only one writer; stop duplicate bare-metal `go run` on same `data/intervoice.db` while Docker is up |
| Login works locally but not server | DB not seeded or different `JWT_SECRET`; sync DB or reset passwords via admin |

Logs:

```bash
docker compose -f docker-compose.prod.yml logs -f gateway admin attendance
```

---

## 9. Firewall

```bash
sudo ufw allow 80/tcp
# optional: sudo ufw allow 443/tcp
```

Put TLS (Caddy, Certbot, or cloud LB) in front of nginx if exposing to the internet.

---

## 10. Coolify (optional)

- Compose file: `docker-compose.coolify.yml`
- Public service: `app` (nginx)
- Mount persistent volume on **`./data`**
- Do not define custom networks (Coolify manages them)

---

## 11. Demo accounts (change before public internet)

| Role | Account | Password |
|------|---------|----------|
| Admin | `admin` or `admin01` | `123456a` |
| Employee | staff ID e.g. `132369`, `118919` | `123456a` |

See [default-accounts.md](../default-accounts.md) for full list.
