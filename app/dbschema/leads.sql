-- Client leads (SQLite): 销售线索 — 纯新客 / 老客激活 + 澳门综合度假村场景字段

CREATE TABLE IF NOT EXISTS client_leads (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  source TEXT NOT NULL DEFAULT 'MANUAL',
  source_ref TEXT,
  lead_segment TEXT NOT NULL DEFAULT 'NEW_PURE',
  approx_origin_region TEXT NOT NULL DEFAULT 'UNKNOWN',
  preferred_venue TEXT NOT NULL DEFAULT 'UNSPECIFIED',
  language_pref TEXT NOT NULL DEFAULT 'zh-CN',
  estimated_party_size INTEGER,
  event_occasion TEXT NOT NULL DEFAULT 'OTHER',
  channel_touchpoint TEXT NOT NULL DEFAULT 'UNKNOWN',
  client_name TEXT NOT NULL,
  client_phone TEXT,
  client_company TEXT,
  intent TEXT NOT NULL DEFAULT 'VISIT',
  notes TEXT,
  priority TEXT NOT NULL DEFAULT 'NORMAL',
  status TEXT NOT NULL DEFAULT 'NEW',
  assigned_user_id INTEGER,
  picked_up_by INTEGER,
  picked_up_at TEXT,
  ref_last_visit_at TEXT,
  ref_last_property TEXT,
  ref_ltv_tier TEXT,
  ref_host_name TEXT,
  ref_member_id_masked TEXT,
  ref_notes TEXT,
  metadata TEXT,
  sync_uid TEXT UNIQUE,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at TEXT NOT NULL DEFAULT (datetime('now')),
  FOREIGN KEY(assigned_user_id) REFERENCES users(id),
  FOREIGN KEY(picked_up_by) REFERENCES users(id)
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_client_leads_source_ref
  ON client_leads(source, source_ref) WHERE source_ref IS NOT NULL AND length(trim(source_ref)) > 0;

CREATE INDEX IF NOT EXISTS idx_client_leads_status ON client_leads(status);
CREATE INDEX IF NOT EXISTS idx_client_leads_assigned ON client_leads(assigned_user_id);
CREATE INDEX IF NOT EXISTS idx_client_leads_updated ON client_leads(updated_at DESC);
-- Indexes on lead_segment / approx_origin_region / preferred_venue are created in
-- MigrateSQLiteLegacy (migrateClientLeadColumns) so older DBs get columns before indexing.

CREATE TABLE IF NOT EXISTS client_lead_events (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  lead_id INTEGER NOT NULL,
  user_id INTEGER,
  event_type TEXT NOT NULL,
  payload TEXT,
  occurred_at TEXT NOT NULL DEFAULT (datetime('now')),
  sync_uid TEXT UNIQUE,
  FOREIGN KEY(lead_id) REFERENCES client_leads(id),
  FOREIGN KEY(user_id) REFERENCES users(id)
);

CREATE INDEX IF NOT EXISTS idx_client_lead_events_lead ON client_lead_events(lead_id, occurred_at DESC);

-- Demo rows: see migrateClientLeadColumns in migrate.go (runs after ALTER on legacy DBs).
