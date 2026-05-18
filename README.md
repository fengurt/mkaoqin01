# Intervoice - Kratos Microservices + H5 MVP

## Stack
- Backend: Go + Kratos microservices
- Frontend: Vue 3 + Vite + Vant 4
- DB: SQLite3
- AI: Volcengine OpenSpeech ASR + Doubao Ark LLM

## Services
- gateway: HTTP entry (`:8010`)
- auth-svc: auth APIs (`:8001`)
- attendance-svc: attendance APIs (`:8002`)
- voice-svc: ASR + NLU (`:8003`)
- admin-svc: dashboard APIs (`:8004`)
- frontend: H5 (`:5173`)

### Gateway: rewards & client leads (JWT `Authorization: Bearer`)

| Method | Path | Notes |
|--------|------|--------|
| GET | `/v1/rewards/me` | Query `userId` |
| POST | `/v1/rewards/ack` | JSON `{ "userId", "badgeIds": [] }` |
| GET | `/v1/leads/feed` | Query `userId` |
| GET | `/v1/leads/detail` | Query `userId`, `leadId` |
| POST | `/v1/leads/pick-up` | JSON `{ "leadId", "userId" }` |
| POST | `/v1/leads/follow-up` | JSON `{ "leadId", "userId", "note", "statusTo?" }` — server allows update only if `userId` matches `assigned_user_id` or `picked_up_by` |
| GET | `/v1/admin/leads` | **Admin** JWT only; list leads |

SQLite 每次服务启动会执行 `dbschema.ApplySQLite`：迁移后跑 **`PRAGMA foreign_key_check`** 与 **`PRAGMA quick_check`**；详见 [`data/DATA_SYNC.md`](data/DATA_SYNC.md)「Post-apply integrity」。

线索 **高价值潜力雷达**（六维规则评分 + SVG 雷达图）由前端 [`frontend/src/lib/leadValuePotential.js`](frontend/src/lib/leadValuePotential.js) 计算，可在后续与 CRM 规则对齐后迁到服务端。

## Quick start
1. Copy env:
   - `cp .env.example .env`
2. Start all:
   - `bash scripts/dev-up.sh`
   - 若新增或修改了 gateway/admin 路由后接口返回 **404**，请再执行一次 `bash scripts/dev-down.sh` 后 `bash scripts/dev-up.sh`（各服务使用 `go run`，需重启方能加载最新代码）。
3. Stop all:
   - `bash scripts/dev-down.sh`
4. Voice API smoke:
   - `bash scripts/test-voice.sh`

## Docker deployment
1. Prepare env:
   - `cp .env.example .env`
   - Fill `ARK_API_KEY` / `SPEECH_API_KEY` if using real AI APIs.
2. Build and start:
   - `docker compose up --build -d`
3. Check services:
   - `docker compose ps`
4. Access:
   - Frontend: `http://localhost:5173`
   - Gateway: `http://localhost:8010`
5. Stop:
   - `docker compose down`

## Docker production profile (AMD64 Linux server)
See **[deploy/DEPLOY_AMD.md](deploy/DEPLOY_AMD.md)** for full server steps.

1. Prepare env:
   - `cp .env.example .env`
   - Set `JWT_SECRET` (and AI keys). Optional: `HTTP_PORT=80`
2. On server after `git clone`:
   - `bash scripts/deploy-amd-prod.sh`
   - Or: `docker compose -f docker-compose.prod.yml up --build -d`
3. Check:
   - `docker compose -f docker-compose.prod.yml ps`
4. Access:
   - `http://<server-ip>:${HTTP_PORT:-80}`
5. Stop:
   - `docker compose -f docker-compose.prod.yml down`

### SQLite sync (local ↔ server)
One file per deployment: `data/intervoice.db`. See **[data/DATA_SYNC.md](data/DATA_SYNC.md)**.

- **Full file copy** (simplest): stop stack, copy DB, start stack:
  - `bash scripts/db-sync.sh push user@host:/opt/mkaoqin01`
  - `bash scripts/db-sync.sh pull user@host:/opt/mkaoqin01`
- **Schedules only**: export/import JSON in 团队考勤 → 班次导入/导出 (admin), or `GET/POST /v1/admin/schedule/grid/export|import`.
- **Selective rows**: admin JWT + `/v1/admin/data/*` or MCP (`scripts/mcp-intervoice-admin`).

## Coolify deployment (recommended bundle)
Use a single Docker Compose stack in Coolify with:
- compose file: `docker-compose.coolify.yml`
- public service: `app` (nginx)
- private services: `auth`, `attendance`, `voice`, `admin`, `gateway`, `frontend`

Why this is better than one mega-container:
- keeps service isolation and easier debugging/restarts
- avoids process supervisor complexity in one image
- matches Coolify's native compose-stack model

Notes:
- Do not define custom networks; Coolify manages networking.
- Inter-service calls use service names (`http://auth:8001`, etc.).
- Persisted data is mounted from `./data` (SQLite file survives redeploys).

## Demo accounts
- employee: `Staff ID / 123456a` (e.g. `132369 / 123456a`, `118919 / 123456a`)
- admin: `admin / 123456a` (compatible: `admin01 / 123456a`)

## Notes
- Wechat login is mocked in MVP.
- If speech credentials are unavailable, voice service uses deterministic mock parse fallback.

## Publish to GitHub
- Create remote repository:
  - `gh repo create fengurt/mkaoqin01 --public --source . --remote origin --push`
- Or push to existing remote:
  - `git push -u origin main`

## Real Ark API Configuration
- Set real credentials in local `.env` (do not commit secrets):
  - `ARK_API_KEY=<your_ark_api_key>`
  - `ARK_ENDPOINT_ID=6030fa1e-eaeb-4664-b330-1d81b08fd526`
  - Preferred ASR auth (new): `SPEECH_API_KEY=<your_openspeech_api_key>`
  - Optional legacy ASR auth: `SPEECH_APP_ID=<your_openspeech_app_id>`, `SPEECH_TOKEN=<your_openspeech_token>`
  - Optional resource override: `SPEECH_RESOURCE_ID=volc.seedasr.auc`
- `voice-svc` now returns `asrMode` and `nluMode` in `/v1/voice/recognize`:
  - Real mode expected: `asrMode=real_openspeech_bigmodel_apikey` (or `real_openspeech`), `nluMode=real_ark_chat_completions`
  - Fallback mode means credentials missing or upstream failed.
