-- 员工按日选择的排班：工作日引用 shift_types，休假引用 activity_types
CREATE TABLE IF NOT EXISTS user_daily_schedule (
  user_id INTEGER NOT NULL,
  work_date TEXT NOT NULL,
  mode TEXT NOT NULL,
  code TEXT NOT NULL,
  updated_at TEXT NOT NULL DEFAULT (datetime('now')),
  PRIMARY KEY (user_id, work_date),
  FOREIGN KEY (user_id) REFERENCES users(id)
);
CREATE INDEX IF NOT EXISTS idx_user_daily_schedule_date ON user_daily_schedule(work_date);

INSERT OR IGNORE INTO shift_types (code, name, start_time, end_time, duration_minutes) VALUES
('OFFICE','标准办公','09:00','18:00',540);
