package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/go-kratos/kratos/v2"
	khttp "github.com/go-kratos/kratos/v2/transport/http"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	database := mustOpenDatabase()
	defer database.Close()
	mustInitDatabase(database)

	addr := envOr("ADMIN_ADDR", ":8004")
	hsrv := khttp.NewServer(khttp.Address(addr))

	hsrv.HandleFunc("/v1/admin/board", func(writer http.ResponseWriter, request *http.Request) {
		rows, err := database.Query(`
SELECT u.id, u.display_name, IFNULL(a.status,'OFFLINE') status, IFNULL(a.location,'') location, IFNULL(a.occurred_at,'') occurred_at
FROM users u
LEFT JOIN (
  SELECT ar1.* FROM attendance_records ar1
  JOIN (
    SELECT user_id, MAX(id) max_id FROM attendance_records GROUP BY user_id
  ) ar2 ON ar1.id = ar2.max_id
) a ON u.id = a.user_id
WHERE u.role = 'employee'
ORDER BY u.id ASC`)
		if err != nil {
			writeJSON(writer, http.StatusInternalServerError, map[string]string{"error": "query failed"})
			return
		}
		defer rows.Close()

		items := make([]map[string]any, 0)
		for rows.Next() {
			var userID int64
			var userName string
			var status string
			var location string
			var occurredAt string
			_ = rows.Scan(&userID, &userName, &status, &location, &occurredAt)
			items = append(items, map[string]any{
				"userId": userID,
				"userName": userName,
				"status": status,
				"location": location,
				"occurredAt": occurredAt,
			})
		}

		writeJSON(writer, http.StatusOK, map[string]any{"items": items})
	})

	hsrv.HandleFunc("/v1/admin/summary", func(writer http.ResponseWriter, request *http.Request) {
		period := request.URL.Query().Get("period")
		if period == "" {
			period = "day"
		}

		startDate := "date('now','localtime')"
		switch period {
		case "week":
			startDate = "date('now','weekday 1','-7 days','localtime')"
		case "month":
			startDate = "date('now','start of month','localtime')"
		}

		var totalEmployees int64
		_ = database.QueryRow("SELECT COUNT(1) FROM users WHERE role = 'employee'").Scan(&totalEmployees)

		var totalRecords int64
		totalRecordsQuery := "SELECT COUNT(1) FROM attendance_records WHERE date(occurred_at) >= " + startDate
		_ = database.QueryRow(totalRecordsQuery).Scan(&totalRecords)

		var outingCount int64
		outingQuery := "SELECT COUNT(1) FROM attendance_records WHERE date(occurred_at) >= " + startDate + " AND status IN ('OUTING','BUSINESS_TRIP')"
		_ = database.QueryRow(outingQuery).Scan(&outingCount)

		var offlineCount int64
		offlineQuery := `
SELECT COUNT(1) FROM users u
LEFT JOIN (
  SELECT ar1.* FROM attendance_records ar1
  JOIN (SELECT user_id, MAX(id) max_id FROM attendance_records GROUP BY user_id) ar2 ON ar1.id = ar2.max_id
) a ON u.id = a.user_id
WHERE u.role = 'employee' AND IFNULL(a.status,'OFFLINE') = 'OFFLINE'`
		_ = database.QueryRow(offlineQuery).Scan(&offlineCount)

		writeJSON(writer, http.StatusOK, map[string]any{
			"period":         period,
			"totalEmployees": totalEmployees,
			"totalRecords":   totalRecords,
			"outingCount":    outingCount,
			"offlineCount":   offlineCount,
		})
	})

	hsrv.HandleFunc("/v1/admin/report", func(writer http.ResponseWriter, request *http.Request) {
		startDate := "date('now','start of month','localtime')"

		var lateCount int64
		lateQuery := "SELECT COUNT(1) FROM attendance_records WHERE date(occurred_at) >= " + startDate + " AND status = 'CHECK_IN' AND time(occurred_at) > '09:10:00'"
		_ = database.QueryRow(lateQuery).Scan(&lateCount)

		var earlyLeaveCount int64
		earlyQuery := "SELECT COUNT(1) FROM attendance_records WHERE date(occurred_at) >= " + startDate + " AND status = 'CHECK_OUT' AND time(occurred_at) < '18:00:00'"
		_ = database.QueryRow(earlyQuery).Scan(&earlyLeaveCount)

		var totalHours float64
		var checkInCount int64
		_ = database.QueryRow("SELECT COUNT(1) FROM attendance_records WHERE date(occurred_at) >= " + startDate + " AND status = 'CHECK_IN'").Scan(&checkInCount)
		totalHours = float64(checkInCount) * 8

		var riskAlerts int64
		riskAlerts = lateCount + earlyLeaveCount

		daySeed := time.Now().YearDay()
		makeSeries := func(base int64, delta int) []int64 {
			series := make([]int64, 0, 7)
			for i := 6; i >= 0; i -= 1 {
				offset := int64(((daySeed + i) % delta))
				series = append(series, base+offset)
			}
			return series
		}
		simulatedTrend := map[string]any{
			"alerts7d":   makeSeries(maxInt64(riskAlerts, 1), 3),
			"checkins7d": makeSeries(maxInt64(checkInCount, 5), 5),
		}

		writeJSON(writer, http.StatusOK, map[string]any{
			"lateCount":       lateCount,
			"earlyLeaveCount": earlyLeaveCount,
			"totalHours":      totalHours,
			"riskAlerts":      riskAlerts,
			"simulatedTrend":  simulatedTrend,
		})
	})

	hsrv.HandleFunc("/v1/admin/team", func(writer http.ResponseWriter, request *http.Request) {
		period := request.URL.Query().Get("period")
		if period == "" {
			period = "day"
		}
		targetDate := request.URL.Query().Get("date")
		if targetDate != "" {
			if _, err := time.Parse("2006-01-02", targetDate); err != nil {
				writeJSON(writer, http.StatusBadRequest, map[string]string{"error": "invalid date format, expected YYYY-MM-DD"})
				return
			}
		}
		startDate := "date('now','localtime')"
		filter := "date(occurred_at) >= " + startDate
		switch period {
		case "week":
			startDate = "date('now','weekday 1','-7 days','localtime')"
			filter = "date(occurred_at) >= " + startDate
		case "month":
			startDate = "date('now','start of month','localtime')"
			filter = "date(occurred_at) >= " + startDate
		}
		if targetDate != "" {
			filter = "date(occurred_at) = date('" + targetDate + "')"
		}
		rows, err := database.Query(`
SELECT u.id, u.display_name, IFNULL(ar.status,'OFFLINE') status, IFNULL(ar.location,'') location, IFNULL(ar.reason,'') reason, IFNULL(ar.occurred_at,'') occurred_at
FROM users u
LEFT JOIN (
  SELECT t1.* FROM attendance_records t1
  INNER JOIN (
    SELECT user_id, MAX(id) id FROM attendance_records
    WHERE ` + filter + `
    GROUP BY user_id
  ) t2 ON t1.id = t2.id
) ar ON u.id = ar.user_id
WHERE u.role = 'employee'
ORDER BY u.id ASC`)
		if err != nil {
			writeJSON(writer, http.StatusInternalServerError, map[string]string{"error": "team query failed"})
			return
		}
		defer rows.Close()
		items := make([]map[string]any, 0)
		for rows.Next() {
			var userID int64
			var userName string
			var status string
			var location string
			var reason string
			var occurredAt string
			_ = rows.Scan(&userID, &userName, &status, &location, &reason, &occurredAt)
			items = append(items, map[string]any{
				"userId": userID, "userName": userName, "status": status, "location": location, "reason": reason, "occurredAt": occurredAt,
			})
		}
		writeJSON(writer, http.StatusOK, map[string]any{"period": period, "date": targetDate, "items": items})
	})

	hsrv.HandleFunc("/healthz", func(writer http.ResponseWriter, request *http.Request) {
		writeJSON(writer, http.StatusOK, map[string]string{"service": "admin-svc", "status": "ok"})
	})

	app := kratos.New(kratos.Name("admin-svc"), kratos.Server(hsrv))
	if err := app.Run(); err != nil {
		panic(err)
	}
}

func mustOpenDatabase() *sql.DB {
	dbPath := envOr("DB_PATH", "../../data/intervoice.db")
	database, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		panic(err)
	}
	return database
}

func mustInitDatabase(database *sql.DB) {
	schema := `CREATE TABLE IF NOT EXISTS users (
  id INTEGER PRIMARY KEY,
  account TEXT NOT NULL UNIQUE,
  password TEXT NOT NULL,
  role TEXT NOT NULL,
  display_name TEXT NOT NULL,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE TABLE IF NOT EXISTS attendance_records (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL,
  status TEXT NOT NULL,
  location TEXT,
  reason TEXT,
  occurred_at TEXT NOT NULL,
  attachment_url TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
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
CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);
CREATE INDEX IF NOT EXISTS idx_users_account ON users(account);
CREATE INDEX IF NOT EXISTS idx_attendance_user_occurred ON attendance_records(user_id, occurred_at);
CREATE INDEX IF NOT EXISTS idx_attendance_occurred ON attendance_records(occurred_at);
CREATE INDEX IF NOT EXISTS idx_attendance_status_occurred ON attendance_records(status, occurred_at);
INSERT OR IGNORE INTO users (id, account, password, role, display_name) VALUES
  (900001, 'admin', '123456a', 'admin', '系统管理员'),
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
  ('141906','Duke Sui','Bella Team','5.4-5.10','Mon:RDO; Tue:RDO; Wed-Sun:1200-2136');`
	if _, err := database.Exec(schema); err != nil {
		panic(err)
	}
	mustEnsureTwoWeeksSimulatedAttendance(database)
}

func writeJSON(writer http.ResponseWriter, statusCode int, payload any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(statusCode)
	_ = json.NewEncoder(writer).Encode(payload)
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func maxInt64(left int64, right int64) int64 {
	if left > right {
		return left
	}
	return right
}

func mustEnsureTwoWeeksSimulatedAttendance(database *sql.DB) {
	if _, err := database.Exec(`CREATE TABLE IF NOT EXISTS seed_markers (
  marker_key TEXT PRIMARY KEY,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
)`); err != nil {
		panic(err)
	}

	const markerKey = "team_attendance_14d_v1"
	var markerCount int64
	_ = database.QueryRow("SELECT COUNT(1) FROM seed_markers WHERE marker_key = ?", markerKey).Scan(&markerCount)
	if markerCount > 0 {
		return
	}

	rows, err := database.Query("SELECT id, display_name FROM users WHERE role = 'employee' ORDER BY id ASC")
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	type employee struct {
		id   int64
		name string
	}
	employees := make([]employee, 0)
	for rows.Next() {
		var item employee
		_ = rows.Scan(&item.id, &item.name)
		employees = append(employees, item)
	}
	if len(employees) == 0 {
		return
	}

	tx, err := database.Begin()
	if err != nil {
		panic(err)
	}
	defer tx.Rollback()

	insertStatement, err := tx.Prepare(`INSERT INTO attendance_records (user_id, status, location, reason, occurred_at) VALUES (?, ?, ?, ?, ?)`)
	if err != nil {
		panic(err)
	}
	defer insertStatement.Close()

	locations := []string{"公司", "客户现场", "门店", "商务区", "会议室A"}
	reasons := map[string]string{
		"OUTING":        "拜访客户",
		"DINING":        "商务用餐",
		"BUSINESS_TRIP": "访问客户",
		"OFFICE":        "在岗办公",
	}
	statusPool := []string{"OUTING", "DINING", "BUSINESS_TRIP", "OFFICE"}

	now := time.Now()
	for dayOffset := 13; dayOffset >= 0; dayOffset -= 1 {
		currentDate := now.AddDate(0, 0, -dayOffset)
		for _, item := range employees {
			baseHour := 8 + int((item.id+int64(dayOffset))%3)
			checkInMinute := 10 + int((item.id+int64(dayOffset*7))%35)
			checkInTime := time.Date(currentDate.Year(), currentDate.Month(), currentDate.Day(), baseHour, checkInMinute, 0, 0, time.Local)
			_, _ = insertStatement.Exec(item.id, "CHECK_IN", "公司", "正常签到", checkInTime.Format("2006-01-02 15:04:05"))

			statusIndex := int((item.id + int64(dayOffset*5)) % int64(len(statusPool)))
			activityStatus := statusPool[statusIndex]
			activityLocation := locations[int((item.id+int64(dayOffset*3))%int64(len(locations)))]
			activityTime := checkInTime.Add(time.Duration(3+int((item.id+int64(dayOffset))%4)) * time.Hour)
			_, _ = insertStatement.Exec(item.id, activityStatus, activityLocation, reasons[activityStatus], activityTime.Format("2006-01-02 15:04:05"))

			checkOutTime := time.Date(currentDate.Year(), currentDate.Month(), currentDate.Day(), 18+int((item.id+int64(dayOffset))%2), int((item.id+int64(dayOffset*11))%30), 0, 0, time.Local)
			_, _ = insertStatement.Exec(item.id, "CHECK_OUT", "公司", "结束工作", checkOutTime.Format("2006-01-02 15:04:05"))
		}
	}

	if _, err := tx.Exec("INSERT OR IGNORE INTO seed_markers (marker_key) VALUES (?)", markerKey); err != nil {
		panic(err)
	}
	if err := tx.Commit(); err != nil {
		panic(err)
	}
}
