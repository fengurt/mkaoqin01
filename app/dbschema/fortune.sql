-- 今日好运：按用户 + 日期存储海报图与话术（管理员可上传并同步）

CREATE TABLE IF NOT EXISTS user_daily_fortune (
  user_id INTEGER NOT NULL,
  fortune_date TEXT NOT NULL,
  image_url TEXT NOT NULL,
  caption TEXT,
  updated_at TEXT NOT NULL DEFAULT (datetime('now')),
  PRIMARY KEY (user_id, fortune_date),
  FOREIGN KEY(user_id) REFERENCES users(id)
);

CREATE INDEX IF NOT EXISTS idx_user_daily_fortune_date ON user_daily_fortune(fortune_date);
