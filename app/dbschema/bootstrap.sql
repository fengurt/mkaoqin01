-- Canonical SQLite DDL + seeds (single source). See data/DATA_SYNC.md

CREATE TABLE IF NOT EXISTS users (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  account TEXT NOT NULL UNIQUE,
  password TEXT NOT NULL,
  role TEXT NOT NULL,
  display_name TEXT NOT NULL,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at TEXT NOT NULL DEFAULT (datetime('now')),
  sync_uid TEXT UNIQUE
);
CREATE TABLE IF NOT EXISTS attendance_records (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL,
  status TEXT NOT NULL,
  location TEXT,
  reason TEXT,
  occurred_at TEXT NOT NULL,
  attachment_url TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at TEXT NOT NULL DEFAULT (datetime('now')),
  sync_uid TEXT UNIQUE,
  FOREIGN KEY(user_id) REFERENCES users(id)
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
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  staff_id TEXT NOT NULL,
  staff_name TEXT NOT NULL,
  team_name TEXT NOT NULL,
  week_range TEXT NOT NULL,
  schedule_text TEXT NOT NULL,
  UNIQUE(staff_id, team_name, week_range)
);

CREATE TABLE IF NOT EXISTS seed_markers (
  marker_key TEXT PRIMARY KEY,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);
CREATE INDEX IF NOT EXISTS idx_users_account ON users(account);
CREATE INDEX IF NOT EXISTS idx_attendance_user_occurred ON attendance_records(user_id, occurred_at);
CREATE INDEX IF NOT EXISTS idx_attendance_occurred ON attendance_records(occurred_at);
CREATE INDEX IF NOT EXISTS idx_attendance_status_occurred ON attendance_records(status, occurred_at);
-- id=1: matches /v1/auth/wechat demo JWT + frontend default localStorage user; required for FK (attendance_records, user_daily_schedule).
INSERT OR IGNORE INTO users (id, account, password, role, display_name) VALUES
  (1, 'wechat_demo', '123456a', 'employee', '微信员工演示账号');
INSERT OR IGNORE INTO users (id, account, password, role, display_name) VALUES
  (900001, 'admin', '123456a', 'admin', '系统管理员'),
  (900002, 'admin01', '123456a', 'admin', '系统管理员-兼容账号'),
  (900101, 'staff01', '123456a', 'employee', '员工演示账号'),
  (118919, '118919', '123456a', 'employee', 'Justin Lu'),
  (132369, '132369', '123456a', 'employee', 'Albee Liu'),
  (132387, '132387', '123456a', 'employee', 'Betty Zhang'),
  (132920, '132920', '123456a', 'employee', 'Heather Zou'),
  (135320, '135320', '123456a', 'employee', 'Simon Kok'),
  (111071, '111071', '123456a', 'employee', 'Sonia Song'),
  (142035, '142035', '123456a', 'employee', 'Ashley Lei'),
  (132922, '132922', '123456a', 'employee', 'Isaac Su'),
  (128943, '128943', '123456a', 'employee', 'Kalei Kong'),
  (134846, '134846', '123456a', 'employee', 'Simon Wu'),
  (137291, '137291', '123456a', 'employee', 'Emily Li'),
  (137420, '137420', '123456a', 'employee', 'Max Wang'),
  (139407, '139407', '123456a', 'employee', 'Stacey Pong'),
  (140957, '140957', '123456a', 'employee', 'Owen Liang'),
  (132921, '132921', '123456a', 'employee', 'Bella Guo'),
  (132386, '132386', '123456a', 'employee', 'Elva Ao'),
  (132923, '132923', '123456a', 'employee', 'Sky Wang'),
  (132924, '132924', '123456a', 'employee', 'William Chen'),
  (139436, '139436', '123456a', 'employee', 'Joyce Yi'),
  (141898, '141898', '123456a', 'employee', 'Leah Zhou'),
  (142133, '142133', '123456a', 'employee', 'Jeremy Cai'),
  (132945, '132945', '123456a', 'employee', 'Vicky Yue'),
  (140672, '140672', '123456a', 'employee', 'SiSi Sou'),
  (141780, '141780', '123456a', 'employee', 'Sammi Xian'),
  (141906, '141906', '123456a', 'employee', 'Duke Sui');
INSERT OR IGNORE INTO activity_types (code, full_name, description) VALUES
  ('RDO','Rest Day Off','常规休息日'),
  ('AL','Annual Leave','带薪年假'),
  ('PH','Public Holiday','公共假期'),
  ('PHCL','Public Holiday Compensatory Leave','公共假期加班补休'),
  ('RDOC','Rest Day Off Compensatory','休息日加班补休'),
  ('RV','Rest Vacation','其他类型带薪休假');
INSERT OR IGNORE INTO shift_types (code, name, start_time, end_time, duration_minutes) VALUES
  ('EARLY','早班','09:30','19:06',576),
  ('EARLY_MID','早中班','10:30','20:06',576),
  ('MID','中班','11:00','20:36',576),
  ('MID_LATE','中晚班','12:00','21:36',576),
  ('LATE','晚班','13:30','23:06',576),
  ('NIGHT','深夜班','14:00','23:36',576),
  ('STANDBY24','24小时手机待命','00:00','23:59',1440);
INSERT OR IGNORE INTO employee_schedules (staff_id, staff_name, team_name, week_range, schedule_text) VALUES
  ('118919','Justin Lu','Isaac Team','4.27-5.3','Mon:RDO; Tue-Sat:24H_STANDBY; Sun:RDO'),
  ('118919','Justin Lu','Bella Team','5.4-5.10','Mon:RDO; Tue-Sat:24H_STANDBY; Sun:RDO'),
  ('132369','Albee Liu','Isaac Team','4.27-5.3','Mon:RDO; Tue:RDO; Wed-Sun:1330-2306'),
  ('132369','Albee Liu','Bella Team','5.4-5.10','Mon-Thu:1330-2306; Fri:RDOC; Sat:RDO; Sun:RDO'),
  ('132387','Betty Zhang','Isaac Team','4.27-5.3','Mon:RDO; Tue:RDO; Wed-Sun:1330-2306'),
  ('132387','Betty Zhang','Bella Team','5.4-5.10','Mon:RDO; Tue:RDO; Wed-Sun:1330-2306'),
  ('132920','Heather Zou','Isaac Team','4.27-5.3','Mon:1030-2006; Tue:RDO; Wed-Thu:1030-2006; Fri:PH; Sat:RDO; Sun:1030-2006'),
  ('132920','Heather Zou','Bella Team','5.4-5.10','Mon-Thu:1030-2006; Fri:RDO; Sat:RDO; Sun:RDO'),
  ('135320','Simon Kok','Isaac Team','4.27-5.3','Mon:RDO; Tue-Sat:1330-2306; Sun:RDO'),
  ('135320','Simon Kok','Bella Team','5.4-5.10','Mon:RDO; Tue-Sat:1330-2306; Sun:RDO'),
  ('111071','Sonia Song','Isaac Team','4.27-5.3','Mon:RDO; Tue-Thu:1330-2306; Fri:PH; Sat:RDO; Sun:1330-2306'),
  ('111071','Sonia Song','Bella Team','5.4-5.10','Mon-Thu:1330-2306; Fri:RDO; Sat:RDO; Sun:1330-2306'),
  ('142035','Ashley Lei','Isaac Team','4.27-5.3','Mon-Tue:1330-2306; Wed:RDO; Thu:RDO; Fri-Sun:1330-2306'),
  ('142035','Ashley Lei','Bella Team','5.4-5.10','Mon-Wed:1330-2306; Thu:RDO; Fri:RDO; Sat-Sun:1330-2306'),
  ('132922','Isaac Su','Isaac Team','4.27-5.3','Mon:RDO; Tue-Sat:1330-2306; Sun:RDO'),
  ('132922','Isaac Su','Bella Team','5.4-5.10','Mon:RDO; Tue-Sat:1330-2306; Sun:RDO'),
  ('128943','Kalei Kong','Isaac Team','4.27-5.3','Mon:1100-2036; Tue:RDO; Wed:RDO; Thu-Sun:1100-2036'),
  ('128943','Kalei Kong','Bella Team','5.4-5.10','Mon-Fri:1100-2036; Sat:RDO; Sun:RDO'),
  ('134846','Simon Wu','Isaac Team','4.27-5.3','Mon-Tue:1400-2336; Wed:1100-2036; Thu:0930-1906; Fri:PH; Sat:RDO; Sun:RDO'),
  ('134846','Simon Wu','Bella Team','5.4-5.10','Mon:RDO; Tue:RDO; Wed:AL; Thu:AL; Fri:RDO; Sat:RDO; Sun:1400-2336'),
  ('137291','Emily Li','Isaac Team','4.27-5.3','Mon-Tue:1100-2036; Wed:RDO; Thu:RDO; Fri-Sun:1100-2036'),
  ('137291','Emily Li','Bella Team','5.4-5.10','Mon:1100-2036; Tue:RDO; Wed:RDO; Thu-Sun:1100-2036'),
  ('137420','Max Wang','Isaac Team','4.27-5.3','Mon:RDO; Tue:RDO; Wed:PHCL; Thu:PHCL; Fri:PH; Sat-Sun:1400-2336'),
  ('137420','Max Wang','Bella Team','5.4-5.10','Mon:RDO; Tue:RDO; Wed-Sun:1400-2336'),
  ('139407','Stacey Pong','Isaac Team','4.27-5.3','Mon:RDO; Tue:RDO; Wed-Sun:1100-2036'),
  ('139407','Stacey Pong','Bella Team','5.4-5.10','Mon:RDO; Tue:RDO; Wed-Thu:1100-2036; Fri-Sat:1400-2336; Sun:1100-2036'),
  ('140957','Owen Liang','Isaac Team','4.27-5.3','Mon:PHCL; Tue:RDO; Wed:RDO; Thu:1400-2336; Fri-Sun:1100-2036'),
  ('140957','Owen Liang','Bella Team','5.4-5.10','Mon:1100-2036; Tue:RDO; Wed:RDO; Thu:1100-2036; Fri-Sat:1400-2336; Sun:1100-2036'),
  ('132921','Bella Guo','Isaac Team','4.27-5.3','Mon:RDO; Tue:RDO; Wed:PHCL; Thu-Sun:1330-2306'),
  ('132921','Bella Guo','Bella Team','5.4-5.10','Mon:1330-2306; Tue:RDO; Wed-Sat:1330-2306; Sun:RDO'),
  ('132386','Elva Ao','Isaac Team','4.27-5.3','Mon-Thu:1330-2306; Fri:PH; Sat:RDO; Sun:RDO'),
  ('132386','Elva Ao','Bella Team','5.4-5.10','Mon:RDO; Tue:RDO; Wed:RDOC; Thu-Sun:1330-2306'),
  ('132923','Sky Wang','Isaac Team','4.27-5.3','Mon-Thu:AL; Fri:PH; Sat:RDO; Sun:RDO'),
  ('132923','Sky Wang','Bella Team','5.4-5.10','Mon-Fri:1030-2006; Sat:RDO; Sun:RDO'),
  ('132924','William Chen','Isaac Team','4.27-5.3','Mon-Thu:1330-2306; Fri:PH; Sat:RDO; Sun:RDO'),
  ('132924','William Chen','Bella Team','5.4-5.10','Mon:RDO; Tue:RDO; Wed-Sun:1330-2306'),
  ('139436','Joyce Yi','Isaac Team','4.27-5.3','Mon:RDO; Tue:RDO; Wed:AL; Thu-Sun:1330-2306'),
  ('139436','Joyce Yi','Bella Team','5.4-5.10','Mon:RDO; Tue:RDO; Wed-Sun:1330-2306'),
  ('141898','Leah Zhou','Isaac Team','4.27-5.3','Mon-Thu:1330-2306; Fri:PH; Sat:RDO; Sun:RDO'),
  ('141898','Leah Zhou','Bella Team','5.4-5.10','Mon:RDO; Tue-Sat:1330-2306; Sun:RDO'),
  ('142133','Jeremy Cai','Isaac Team','4.27-5.3','Mon:RDO; Tue-Sat:1330-2306; Sun:RDO'),
  ('142133','Jeremy Cai','Bella Team','5.4-5.10','Mon:RDO; Tue-Sat:1330-2306; Sun:RDO'),
  ('132945','Vicky Yue','Isaac Team','4.27-5.3','Mon-Thu:1200-2136; Fri:PH; Sat:RDO; Sun:RDO'),
  ('132945','Vicky Yue','Bella Team','5.4-5.10','Mon-Fri:1200-2136; Sat:RDO; Sun:RDO'),
  ('140672','SiSi Sou','Isaac Team','4.27-5.3','Mon:1400-2336; Tue:RDO; Wed-Thu:1400-2336; Fri:PH; Sat-Sun:1400-2336'),
  ('140672','SiSi Sou','Bella Team','5.4-5.10','Mon-Fri:1400-2336; Sat:RDO; Sun:RDO'),
  ('141780','Sammi Xian','Isaac Team','4.27-5.3','Mon:1100-2036; Tue:RDO; Wed:RV; Thu:RV; Fri:PH; Sat-Sun:1100-2036'),
  ('141780','Sammi Xian','Bella Team','5.4-5.10','Mon:1100-2036; Tue:RDO; Wed:RDO; Thu-Sun:1100-2036'),
  ('141906','Duke Sui','Isaac Team','4.27-5.3','Mon:RDO; Tue:RDO; Wed-Thu:1200-2136; Fri:PH; Sat-Sun:1200-2136'),
  ('141906','Duke Sui','Bella Team','5.4-5.10','Mon:RDO; Tue:RDO; Wed-Sun:1200-2136');
INSERT OR IGNORE INTO users (id, account, password, role, display_name)
SELECT CAST(staff_id AS INTEGER), staff_id, '123456a', 'employee', staff_name
FROM employee_schedules
GROUP BY staff_id, staff_name;