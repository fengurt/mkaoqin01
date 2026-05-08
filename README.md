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

## Quick start
1. Copy env:
   - `cp .env.example .env`
2. Start all:
   - `bash scripts/dev-up.sh`
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

## Demo accounts
- employee: `staff01 / 123456`
- admin: `admin01 / 123456`

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
