-- Target PostgreSQL / compatible-with-MySQL (types adjusted per engine) — logical mirror of bootstrap.sql.
-- Use sync_uid as the global identity for replication; local bigint id is optional per deployment.

CREATE TABLE IF NOT EXISTS users (
  id BIGSERIAL PRIMARY KEY,
  account TEXT NOT NULL UNIQUE,
  password TEXT NOT NULL,
  role TEXT NOT NULL,
  display_name TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  sync_uid UUID UNIQUE
);

CREATE TABLE IF NOT EXISTS attendance_records (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
  status TEXT NOT NULL,
  location TEXT,
  reason TEXT,
  occurred_at TIMESTAMPTZ NOT NULL,
  attachment_url TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  sync_uid UUID UNIQUE
);

CREATE TABLE IF NOT EXISTS activity_types (
  code TEXT PRIMARY KEY,
  full_name TEXT NOT NULL,
  description TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS shift_types (
  code TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  start_time TEXT NOT NULL,
  end_time TEXT NOT NULL,
  duration_minutes INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS employee_schedules (
  id BIGSERIAL PRIMARY KEY,
  staff_id TEXT NOT NULL,
  staff_name TEXT NOT NULL,
  team_name TEXT NOT NULL,
  week_range TEXT NOT NULL,
  schedule_text TEXT NOT NULL,
  UNIQUE (staff_id, team_name, week_range)
);

-- Demo/dev only: do not replicate to production central DB unless you want synthetic rows.
CREATE TABLE IF NOT EXISTS seed_markers (
  marker_key TEXT PRIMARY KEY,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_users_role ON users (role);
CREATE INDEX IF NOT EXISTS idx_users_account ON users (account);
CREATE INDEX IF NOT EXISTS idx_attendance_user_occurred ON attendance_records (user_id, occurred_at);
CREATE INDEX IF NOT EXISTS idx_attendance_occurred ON attendance_records (occurred_at);
CREATE INDEX IF NOT EXISTS idx_attendance_status_occurred ON attendance_records (status, occurred_at);

CREATE TABLE IF NOT EXISTS admin_data_audit (
  id BIGSERIAL PRIMARY KEY,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  actor_user_id BIGINT NOT NULL,
  action TEXT NOT NULL,
  entity TEXT NOT NULL,
  sync_uid UUID,
  detail TEXT
);

CREATE TABLE IF NOT EXISTS location_catalog (
  id BIGSERIAL PRIMARY KEY,
  slug TEXT NOT NULL UNIQUE,
  category TEXT NOT NULL,
  region TEXT NOT NULL DEFAULT '',
  title TEXT NOT NULL,
  subtitle TEXT NOT NULL DEFAULT '',
  detail TEXT NOT NULL DEFAULT '',
  sort_order INTEGER NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  sync_uid UUID
);

CREATE INDEX IF NOT EXISTS idx_location_catalog_category ON location_catalog (category);
CREATE INDEX IF NOT EXISTS idx_location_catalog_region ON location_catalog (region);

CREATE TABLE IF NOT EXISTS schedule_quick_section (
  id BIGSERIAL PRIMARY KEY,
  sort_order INTEGER NOT NULL DEFAULT 0,
  section_label TEXT NOT NULL,
  item_category TEXT NOT NULL,
  item_region TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS ux_schedule_quick_section_natural ON schedule_quick_section (section_label, item_category, item_region);
CREATE INDEX IF NOT EXISTS idx_schedule_quick_section_sort ON schedule_quick_section (sort_order);

CREATE TABLE IF NOT EXISTS user_daily_schedule (
  user_id BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  work_date DATE NOT NULL,
  mode TEXT NOT NULL,
  code TEXT NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (user_id, work_date)
);

CREATE INDEX IF NOT EXISTS idx_user_daily_schedule_date ON user_daily_schedule (work_date);

INSERT INTO shift_types (code, name, start_time, end_time, duration_minutes)
VALUES ('OFFICE', '标准办公', '09:00', '18:00', 540)
ON CONFLICT (code) DO NOTHING;
