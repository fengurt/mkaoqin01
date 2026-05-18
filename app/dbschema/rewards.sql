-- Rewards: streak counters + badge definitions + per-user awards (SQLite)

CREATE TABLE IF NOT EXISTS badge_definitions (
  code TEXT PRIMARY KEY,
  kind TEXT NOT NULL,
  tier TEXT NOT NULL DEFAULT 'BRONZE',
  title_i18n TEXT NOT NULL,
  description_i18n TEXT NOT NULL DEFAULT '{}',
  rule_type TEXT NOT NULL,
  rule_threshold INTEGER NOT NULL DEFAULT 0,
  rule_window TEXT NOT NULL DEFAULT 'ALL_TIME',
  reward_payload TEXT NOT NULL DEFAULT '{}',
  reissuable INTEGER NOT NULL DEFAULT 0,
  active INTEGER NOT NULL DEFAULT 1,
  sort_order INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS user_streaks (
  user_id INTEGER PRIMARY KEY,
  current_checkin_streak INTEGER NOT NULL DEFAULT 0,
  best_checkin_streak INTEGER NOT NULL DEFAULT 0,
  current_full_streak INTEGER NOT NULL DEFAULT 0,
  best_full_streak INTEGER NOT NULL DEFAULT 0,
  total_checkin_days INTEGER NOT NULL DEFAULT 0,
  total_full_days INTEGER NOT NULL DEFAULT 0,
  last_checkin_date TEXT,
  last_full_date TEXT,
  member_tier TEXT NOT NULL DEFAULT 'BRONZE',
  updated_at TEXT NOT NULL DEFAULT (datetime('now')),
  FOREIGN KEY(user_id) REFERENCES users(id)
);

CREATE TABLE IF NOT EXISTS user_badges (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL,
  badge_code TEXT NOT NULL,
  period_key TEXT NOT NULL DEFAULT 'ALL_TIME',
  awarded_at TEXT NOT NULL DEFAULT (datetime('now')),
  evidence_payload TEXT,
  acknowledged_at TEXT,
  sync_uid TEXT UNIQUE,
  FOREIGN KEY(user_id) REFERENCES users(id),
  FOREIGN KEY(badge_code) REFERENCES badge_definitions(code),
  UNIQUE(user_id, badge_code, period_key)
);

CREATE INDEX IF NOT EXISTS idx_user_badges_user ON user_badges(user_id, awarded_at DESC);
CREATE INDEX IF NOT EXISTS idx_user_badges_open ON user_badges(user_id) WHERE acknowledged_at IS NULL;

INSERT OR IGNORE INTO badge_definitions (code, kind, tier, title_i18n, description_i18n, rule_type, rule_threshold, rule_window, reissuable, sort_order) VALUES
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
   'TOTAL_FULL_DAYS', 30, 'ALL_TIME', 0, 50);
