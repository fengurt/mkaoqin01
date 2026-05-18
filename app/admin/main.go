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
	"intervoice/dbschema"
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
				"userId":     userID,
				"userName":   userName,
				"status":     status,
				"location":   location,
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

	registerAdminDataRoutes(hsrv, database)
	registerFortuneAdminRoutes(hsrv, database)
	registerScheduleGridAdminRoutes(hsrv, database)

	hsrv.HandleFunc("/v1/admin/leads", func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			writeJSON(writer, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
			return
		}
		rows, err := database.Query(`
SELECT ` + dbschema.LeadWideSelectColumns + `
FROM client_leads
ORDER BY updated_at DESC
LIMIT 200`)
		if err != nil {
			writeJSON(writer, http.StatusInternalServerError, map[string]string{"error": "query_failed"})
			return
		}
		defer rows.Close()
		items := []map[string]any{}
		for rows.Next() {
			row, err := dbschema.ScanClientLeadWideRow(rows)
			if err != nil {
				writeJSON(writer, http.StatusInternalServerError, map[string]string{"error": "scan_failed"})
				return
			}
			items = append(items, row)
		}
		writeJSON(writer, http.StatusOK, map[string]any{"items": items})
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
	database, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on&_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		panic(err)
	}
	return database
}

func mustInitDatabase(database *sql.DB) {
	if err := dbschema.ApplySQLite(database); err != nil {
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
