# Data layer: single source & sync readiness

## Canonical schema (no duplicate DDL)

- **SQLite DDL + seeds**: [`app/dbschema/bootstrap.sql`](../app/dbschema/bootstrap.sql) — embedded by [`intervoice/dbschema`](../app/dbschema) and applied by auth, attendance, and admin on startup (`dbschema.ApplySQLite`).
- **PostgreSQL target** (logical mirror): [`app/dbschema/postgresql.sql`](../app/dbschema/postgresql.sql). Adjust types (`UUID` → `CHAR(36)`) if you standardise on MySQL.
- **Legacy file** [`schema.sql`](schema.sql) is only a short pointer; do not edit it as a second source of truth.

Additive upgrades for databases created before `bootstrap.sql` gained `updated_at` / `sync_uid` run via `dbschema.MigrateSQLiteLegacy` (idempotent `ALTER TABLE … ADD COLUMN`, ignores duplicate column).

## Runtime storage rules

- **One physical SQLite file per deployment** (`DB_PATH`, default `data/intervoice.db`). Auth, attendance, and admin share it; connection flags enable **`foreign_keys`** and **`journal_mode=WAL`** plus **`busy_timeout`** to reduce multi-process locking issues.
- **Authoritative business rows** for the app API are **`users`** and **`attendance_records`**. Reference data: **`activity_types`**, **`shift_types`**, **`employee_schedules`**.
- **`seed_markers`** is **dev/demo bookkeeping** (e.g. simulated 14-day attendance). Exclude it from production central sync unless you explicitly want synthetic rows replicated.

## Client / edge

- **Browser `localStorage`** (session token, cached `user` JSON) is **not** part of server sync; treat as ephemeral cache only.

## Two-way sync with PostgreSQL / MySQL (recommended model)

True peer-to-peer **two-way** replication between SQLite and a central server is easy to get wrong. Safer patterns:

1. **Single system of record (recommended)**  
   Production writes go to **PostgreSQL (or MySQL)** only. Edge SQLite is **read-through cache** or **offline queue** drained into the server; conflicts resolved by server rules.

2. **If you must reconcile two writable stores**  
   - Every replicated row needs a **stable global id**: column **`sync_uid`** (UUID text in SQLite; `UUID` in PostgreSQL). Integer **`id`** stays local or is mapped per replica; do **not** treat integer `id` as global.
   - Carry **`updated_at`** on mutable entities; use **last-write-wins** or explicit **version** / conflict queues.
   - Replicate **append-only** `attendance_records` by **`sync_uid`** `INSERT … ON CONFLICT DO NOTHING/UPDATE`; detect duplicates by natural key only if you define one (e.g. same `sync_uid`).
   - Map **`user_id`** via **`users.sync_uid`** when merging across databases so FKs stay consistent.

## Integrity checklist before enabling sync

- [ ] Backfill **`sync_uid`** for existing rows (UUID v4) where NULL.
- [ ] Ensure all writers set **`updated_at`** on UPDATE (app layer or triggers).
- [ ] Decide whether **`employee_schedules`** / **`activity_types`** are owned by HQ only (push to edge) or editable offline.
- [ ] Exclude **`seed_markers`** (and optionally demo seeds) from replication jobs.

## Admin data API (remote / agent access)

All routes require **`Authorization: Bearer <JWT>`** with **`role: admin`**. Proxied through the gateway at `/v1/admin/data/…` (same origin as other `/v1` calls).

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/v1/admin/data/meta` | Counts + sync hints |
| GET | `/v1/admin/data/schema` | Table/column reference |
| GET | `/v1/admin/data/users` | `limit`, `offset` (max 500) |
| GET | `/v1/admin/data/attendance` | `userId`, `updatedSince`, `limit`, `offset` |
| POST | `/v1/admin/data/attendance/upsert` | Body: `syncUid`, `userId`, `status`, … `ifMatchUpdatedAt` for optimistic concurrency (409 on conflict) |
| POST | `/v1/admin/data/users/patch` | Body: `id` or `syncUid`, `displayName` / `role`, `ifMatchUpdatedAt` — **no passwords** |

Writes are **audited** in **`admin_data_audit`**. Request bodies are capped at **1 MiB**.

**MCP (Cursor agents):** run [`scripts/mcp-intervoice-admin/server.mjs`](../scripts/mcp-intervoice-admin/server.mjs) over stdio; set **`INTERVOICE_GATEWAY_URL`** and **`INTERVOICE_ADMIN_TOKEN`**. See [`scripts/mcp-intervoice-admin/cursor-mcp-config.example.json`](../scripts/mcp-intervoice-admin/cursor-mcp-config.example.json).
