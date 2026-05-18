# Data layer: single source & sync readiness

## Canonical schema (no duplicate DDL)

- **SQLite DDL + seeds**: [`app/dbschema/bootstrap.sql`](../app/dbschema/bootstrap.sql) plus embedded fragments [`location_catalog.sql`](../app/dbschema/location_catalog.sql), [`schedule_quick_sections.sql`](../app/dbschema/schedule_quick_sections.sql), [`user_daily_schedule.sql`](../app/dbschema/user_daily_schedule.sql), [`rewards.sql`](../app/dbschema/rewards.sql), [`leads.sql`](../app/dbschema/leads.sql) — all applied by [`intervoice/dbschema`](../app/dbschema) via `dbschema.ApplySQLite` on auth, attendance, and admin startup.
- **PostgreSQL target** (logical mirror): [`app/dbschema/postgresql.sql`](../app/dbschema/postgresql.sql). Adjust types (`UUID` → `CHAR(36)`) if you standardise on MySQL. New installs include **CHECK** constraints on `badge_definitions`, `user_streaks` counts, and `client_leads` (status, priority, lead segmentation / venue / channel enums where applicable). SQLite runtime uses the same enums enforced in Go + `PRAGMA foreign_key_check`.
- **Legacy file** [`schema.sql`](schema.sql) is only a short pointer; do not edit it as a second source of truth.

Additive upgrades for databases created before `bootstrap.sql` gained `updated_at` / `sync_uid` run via `dbschema.MigrateSQLiteLegacy` (idempotent `ALTER TABLE … ADD COLUMN`, ignores duplicate column). The same path runs **`migrateClientLeadColumns`** when `client_leads` already exists without the newer Macau-oriented columns.

## Runtime storage rules

- **One physical SQLite file per deployment** (`DB_PATH`, default `data/intervoice.db`). Auth, attendance, and admin share it; connection flags enable **`foreign_keys`** and **`journal_mode=WAL`** plus **`busy_timeout`** to reduce multi-process locking issues.
- **Authoritative business rows** for the app API are **`users`** and **`attendance_records`**. Reference data: **`activity_types`**, **`shift_types`**, **`employee_schedules`**.
- **Gamification** (attendance-svc): **`badge_definitions`** (catalog), **`user_streaks`** (per-user counters), **`user_badges`** (earned rows; `sync_uid` for replication). Streaks update on **`CHECK_IN`** / **`CHECK_OUT`** only.
- **Client leads** (attendance-svc + admin list): **`client_leads`**, **`client_lead_events`** (append-only timeline). Segmentation fields include **`lead_segment`** (`NEW_PURE` vs `OLD_REACTIVATION`), **`approx_origin_region`**, **`preferred_venue`** (MGM 半岛/氹仔等), **`language_pref`**, **`estimated_party_size`**, **`event_occasion`**, **`channel_touchpoint`**, plus optional **`ref_*`** columns for reactivation reference data. Demo rows (`source='DEMO'`, `source_ref` in `seed-macau-new-guest-cotai` / `seed-macau-reactivate-peninsula`) are inserted by **`migrateClientLeadColumns`** (not `leads.sql`) so legacy databases migrate columns before seed `INSERT` — exclude from production sync if undesired.
- **`seed_markers`** is **dev/demo bookkeeping** (e.g. simulated 14-day attendance). Exclude it from production central sync unless you explicitly want synthetic rows replicated.

## Post-apply integrity (SQLite)

After `dbschema.ApplySQLite` finishes DDL + `MigrateSQLiteLegacy`:

1. **`migrateClientLeadColumns`** adds missing segmentation / reference columns on `client_leads`, creates supporting indexes, and runs idempotent **`INSERT OR IGNORE`** demo rows for the two `DEMO` `source_ref` values above (skipped entirely if the `client_leads` table does not exist).
2. **`migrateRewardsLeadsReferentialSanity`** (inside migrate) drops `user_badges` / `user_streaks` rows whose `user_id` is missing from `users`, trims orphan `client_lead_events`, nulls invalid `client_leads` assignee/picker references, and clamps negative streak counters to zero.
3. **`VerifySQLitePostApply`** runs **`PRAGMA foreign_key_check`** (must be empty) and **`PRAGMA quick_check`** (must report `ok`). Connections must keep **`_foreign_keys=on`** (see `DB_PATH` DSN in auth/attendance/admin).

If startup fails with a foreign-key violation, inspect the printed `child_table` / `rowid` from the error, repair data, or restore from backup.

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
| GET | `/v1/admin/leads` | All client leads (newest first, cap 200 rows) |

Writes are **audited** in **`admin_data_audit`**. Request bodies are capped at **1 MiB**.

**MCP (Cursor agents):** run [`scripts/mcp-intervoice-admin/server.mjs`](../scripts/mcp-intervoice-admin/server.mjs) over stdio; set **`INTERVOICE_GATEWAY_URL`** and **`INTERVOICE_ADMIN_TOKEN`**. See [`scripts/mcp-intervoice-admin/cursor-mcp-config.example.json`](../scripts/mcp-intervoice-admin/cursor-mcp-config.example.json).
