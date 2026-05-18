# AMD Linux server deployment

Target: x86_64 Linux (AMD64) with Docker Engine + Compose v2.

## 1. Server prerequisites

```bash
sudo apt-get update
sudo apt-get install -y git docker.io docker-compose-plugin
sudo usermod -aG docker "$USER"
# log out and back in so docker group applies
```

## 2. Clone and configure

```bash
git clone https://github.com/fengurt/mkaoqin01.git
cd mkaoqin01
cp .env.example .env
```

Edit `.env` on the server (never commit):

| Variable | Notes |
|----------|--------|
| `JWT_SECRET` | Strong random string, same across all services |
| `ARK_API_KEY` / `SPEECH_API_KEY` | Production AI keys (optional; voice falls back without them) |
| `HTTP_PORT` | Default `80` in `docker-compose.prod.yml` |

## 3. Database on first deploy

- Compose mounts `./data` → `/app/data` for every backend service.
- If `data/intervoice.db` is missing, services create schema via `dbschema.ApplySQLite` on startup.
- To **seed from your Mac** (users, schedules, attendance), copy the SQLite file **before** first start or while stack is stopped (see [DATA_SYNC.md](../data/DATA_SYNC.md) and `scripts/db-sync.sh`).

```bash
mkdir -p data
# from your laptop (replace host):
# scp data/intervoice.db deploy@YOUR_SERVER:/path/to/mkaoqin01/data/
```

## 4. Build and run (production)

```bash
docker compose -f docker-compose.prod.yml up --build -d
docker compose -f docker-compose.prod.yml ps
curl -sS "http://127.0.0.1:${HTTP_PORT:-80}/healthz" || curl -sS http://127.0.0.1:8010/healthz
```

Public URL: `http://<server-ip>/` (nginx serves frontend and proxies `/v1` to gateway).

## 5. Updates after `git pull`

```bash
git pull origin main
docker compose -f docker-compose.prod.yml up --build -d
```

Migrations run automatically on service start (`ApplySQLite` + legacy migrators).

## 6. Firewall

Open HTTP (and HTTPS if you terminate TLS in front):

```bash
sudo ufw allow 80/tcp
# optional: sudo ufw allow 443/tcp
```

## 7. Coolify (optional)

Use `docker-compose.coolify.yml` instead; mount persistent volume on `./data` in Coolify UI.
