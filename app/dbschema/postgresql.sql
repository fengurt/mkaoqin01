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

-- Rewards / gamification (mirror SQLite semantics)
CREATE TABLE IF NOT EXISTS badge_definitions (
  code TEXT PRIMARY KEY,
  kind TEXT NOT NULL CHECK (kind IN ('USER_BADGE', 'USER_MEDAL', 'ACHIEVEMENT_BADGE', 'MEMBER_BADGE')),
  tier TEXT NOT NULL DEFAULT 'BRONZE' CHECK (tier IN ('BRONZE', 'SILVER', 'GOLD', 'PLATINUM')),
  title_i18n TEXT NOT NULL,
  description_i18n TEXT NOT NULL DEFAULT '{}',
  rule_type TEXT NOT NULL CHECK (rule_type IN ('CONSECUTIVE_CHECKIN', 'CONSECUTIVE_FULLDAY', 'TOTAL_FULL_DAYS')),
  rule_threshold INTEGER NOT NULL DEFAULT 0 CHECK (rule_threshold >= 0),
  rule_window TEXT NOT NULL DEFAULT 'ALL_TIME',
  reward_payload TEXT NOT NULL DEFAULT '{}',
  reissuable INTEGER NOT NULL DEFAULT 0,
  active INTEGER NOT NULL DEFAULT 1,
  sort_order INTEGER NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS user_streaks (
  user_id BIGINT PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
  current_checkin_streak INTEGER NOT NULL DEFAULT 0,
  best_checkin_streak INTEGER NOT NULL DEFAULT 0,
  current_full_streak INTEGER NOT NULL DEFAULT 0,
  best_full_streak INTEGER NOT NULL DEFAULT 0,
  total_checkin_days INTEGER NOT NULL DEFAULT 0,
  total_full_days INTEGER NOT NULL DEFAULT 0,
  last_checkin_date DATE,
  last_full_date DATE,
  member_tier TEXT NOT NULL DEFAULT 'BRONZE' CHECK (member_tier IN ('BRONZE', 'SILVER', 'GOLD', 'PLATINUM')),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT user_streaks_counts_nonneg CHECK (
    current_checkin_streak >= 0 AND best_checkin_streak >= 0 AND current_full_streak >= 0 AND best_full_streak >= 0
    AND total_checkin_days >= 0 AND total_full_days >= 0
  )
);

CREATE TABLE IF NOT EXISTS user_badges (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  badge_code TEXT NOT NULL REFERENCES badge_definitions (code),
  period_key TEXT NOT NULL DEFAULT 'ALL_TIME',
  awarded_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  evidence_payload TEXT,
  acknowledged_at TIMESTAMPTZ,
  sync_uid UUID UNIQUE,
  UNIQUE (user_id, badge_code, period_key)
);

CREATE INDEX IF NOT EXISTS idx_user_badges_user ON user_badges (user_id, awarded_at DESC);

CREATE TABLE IF NOT EXISTS client_leads (
  id BIGSERIAL PRIMARY KEY,
  source TEXT NOT NULL DEFAULT 'MANUAL',
  source_ref TEXT,
  lead_segment TEXT NOT NULL DEFAULT 'NEW_PURE' CHECK (lead_segment IN ('NEW_PURE', 'OLD_REACTIVATION')),
  approx_origin_region TEXT NOT NULL DEFAULT 'UNKNOWN' CHECK (approx_origin_region IN ('GBA', 'HK', 'MAINLAND', 'MACAU_LOCAL', 'SEA', 'INTL', 'UNKNOWN')),
  preferred_venue TEXT NOT NULL DEFAULT 'UNSPECIFIED' CHECK (preferred_venue IN ('MGM_PENINSULA', 'MGM_COTAI', 'OTHER_MACAU_IR', 'UNSPECIFIED')),
  language_pref TEXT NOT NULL DEFAULT 'zh-CN' CHECK (language_pref IN ('zh-CN', 'zh-TW', 'en', 'pt', 'OTHER')),
  estimated_party_size INTEGER CHECK (estimated_party_size IS NULL OR (estimated_party_size >= 0 AND estimated_party_size <= 50000)),
  event_occasion TEXT NOT NULL DEFAULT 'OTHER' CHECK (event_occasion IN ('WEDDING_BANQUET', 'VIP_TABLE', 'CONFERENCE', 'LEISURE', 'OTHER')),
  channel_touchpoint TEXT NOT NULL DEFAULT 'UNKNOWN' CHECK (channel_touchpoint IN ('WECHAT_OFFICIAL', 'HOST_REFERRAL', 'OTA', 'WALK_IN', 'PARTNER', 'UNKNOWN')),
  client_name TEXT NOT NULL,
  client_phone TEXT,
  client_company TEXT,
  intent TEXT NOT NULL DEFAULT 'VISIT',
  notes TEXT,
  priority TEXT NOT NULL DEFAULT 'NORMAL' CHECK (priority IN ('LOW', 'NORMAL', 'HIGH', 'URGENT')),
  status TEXT NOT NULL DEFAULT 'NEW' CHECK (status IN ('NEW', 'ASSIGNED', 'PICKED_UP', 'IN_FOLLOW_UP', 'WON', 'LOST', 'CANCELED')),
  assigned_user_id BIGINT REFERENCES users (id),
  picked_up_by BIGINT REFERENCES users (id),
  picked_up_at TIMESTAMPTZ,
  ref_last_visit_at TEXT,
  ref_last_property TEXT,
  ref_ltv_tier TEXT,
  ref_host_name TEXT,
  ref_member_id_masked TEXT,
  ref_notes TEXT,
  metadata TEXT,
  sync_uid UUID UNIQUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_client_leads_source_ref
  ON client_leads (source, source_ref) WHERE source_ref IS NOT NULL AND length(trim(source_ref)) > 0;

CREATE INDEX IF NOT EXISTS idx_client_leads_status ON client_leads (status);
CREATE INDEX IF NOT EXISTS idx_client_leads_assigned ON client_leads (assigned_user_id);
CREATE INDEX IF NOT EXISTS idx_client_leads_segment ON client_leads (lead_segment);
CREATE INDEX IF NOT EXISTS idx_client_leads_origin ON client_leads (approx_origin_region);
CREATE INDEX IF NOT EXISTS idx_client_leads_venue ON client_leads (preferred_venue);

CREATE TABLE IF NOT EXISTS client_lead_events (
  id BIGSERIAL PRIMARY KEY,
  lead_id BIGINT NOT NULL REFERENCES client_leads (id) ON DELETE CASCADE,
  user_id BIGINT REFERENCES users (id),
  event_type TEXT NOT NULL,
  payload TEXT,
  occurred_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  sync_uid UUID UNIQUE
);

CREATE INDEX IF NOT EXISTS idx_client_lead_events_lead ON client_lead_events (lead_id, occurred_at DESC);

INSERT INTO badge_definitions (code, kind, tier, title_i18n, description_i18n, rule_type, rule_threshold, rule_window, reissuable, sort_order)
VALUES
  ('STREAK_CI_3', 'USER_BADGE', 'BRONZE',
   '{"zh-CN":"三连签到","en":"3-day check-in"}',
   '{"zh-CN":"连续 3 个工作日完成上班签到","en":"Three consecutive workdays with check-in"}',
   'CONSECUTIVE_CHECKIN', 3, 'ALL_TIME', 0, 10),
  ('STREAK_CI_7', 'USER_BADGE', 'SILVER',
   '{"zh-CN":"七连签到","en":"7-day check-in"}',
   '{"zh-CN":"连续 7 个工作日完成上班签到","en":"Seven consecutive workdays with check-in"}',
   'CONSECUTIVE_CHECKIN', 7, 'ALL_TIME', 0, 20),
  ('FULL_PAIR_3', 'USER_MEDAL', 'GOLD',
   '{"zh-CN":"三日全勤闭环","en":"3-day full loop"}',
   '{"zh-CN":"连续 3 天完成签到+签退","en":"Three consecutive days with check-in and check-out"}',
   'CONSECUTIVE_FULLDAY', 3, 'ALL_TIME', 0, 30),
  ('MEMBER_SILVER', 'MEMBER_BADGE', 'SILVER',
   '{"zh-CN":"银级会员","en":"Silver member"}',
   '{"zh-CN":"累计 10 天闭环考勤","en":"10 cumulative full attendance days"}',
   'TOTAL_FULL_DAYS', 10, 'ALL_TIME', 0, 40),
  ('MEMBER_GOLD', 'MEMBER_BADGE', 'GOLD',
   '{"zh-CN":"金级会员","en":"Gold member"}',
   '{"zh-CN":"累计 30 天闭环考勤","en":"30 cumulative full attendance days"}',
   'TOTAL_FULL_DAYS', 30, 'ALL_TIME', 0, 50)
ON CONFLICT (code) DO NOTHING;
