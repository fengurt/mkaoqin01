package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2"
	khttp "github.com/go-kratos/kratos/v2/transport/http"
	_ "github.com/mattn/go-sqlite3"
	"intervoice/dbschema"
)

type submitStatusRequest struct {
	UserID       int64  `json:"userId"`
	Status       string `json:"status"`
	Location     string `json:"location"`
	Reason       string `json:"reason"`
	OccurredAt   string `json:"occurredAt"`
	AttachmentURL string `json:"attachmentUrl"`
}

var allowedAttendanceStatuses = map[string]bool{
	"CHECK_IN":      true,
	"OFFICE":        true,
	"OUTING":        true,
	"DINING":        true,
	"BUSINESS_TRIP": true,
	"CHECK_OUT":     true,
}

func main() {
	database := mustOpenDatabase()
	defer database.Close()
	mustInitDatabase(database)

	addr := envOr("ATTENDANCE_ADDR", ":8002")
	hsrv := khttp.NewServer(khttp.Address(addr))

	hsrv.HandleFunc("/v1/attendance/submit", func(writer http.ResponseWriter, request *http.Request) {
		var payload submitStatusRequest
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			writeJSON(writer, http.StatusBadRequest, map[string]string{"error": "invalid payload"})
			return
		}
		if payload.UserID == 0 {
			payload.UserID = 1
		}
		if payload.Status == "" {
			writeJSON(writer, http.StatusBadRequest, map[string]string{"error": "status required"})
			return
		}
		if !allowedAttendanceStatuses[payload.Status] {
			writeJSON(writer, http.StatusBadRequest, map[string]string{"error": "invalid status"})
			return
		}
		if payload.OccurredAt == "" {
			payload.OccurredAt = time.Now().Format(time.RFC3339)
		}

		result, err := database.Exec(
			"INSERT INTO attendance_records (user_id, status, location, reason, occurred_at, attachment_url) VALUES (?, ?, ?, ?, ?, ?)",
			payload.UserID, payload.Status, payload.Location, payload.Reason, payload.OccurredAt, payload.AttachmentURL,
		)
		if err != nil {
			writeJSON(writer, http.StatusInternalServerError, map[string]string{"error": "insert failed"})
			return
		}
		recordID, _ := result.LastInsertId()
		response := map[string]any{"id": recordID}
		if earned, rewardErr := processRewardsAfterSubmit(database, payload.UserID, payload.Status, payload.OccurredAt); rewardErr != nil {
			response["rewardError"] = "streak_update_failed"
		} else if len(earned) > 0 {
			response["newlyEarnedBadges"] = earned
		}
		writeJSON(writer, http.StatusOK, response)
	})

	hsrv.HandleFunc("/v1/attendance/today", func(writer http.ResponseWriter, request *http.Request) {
		userID := request.URL.Query().Get("userId")
		if userID == "" {
			userID = "1"
		}
		rows, err := database.Query(
			"SELECT id, status, location, reason, occurred_at FROM attendance_records WHERE user_id = ? AND date(occurred_at) = date('now','localtime') ORDER BY occurred_at DESC",
			userID,
		)
		if err != nil {
			writeJSON(writer, http.StatusInternalServerError, map[string]string{"error": "query failed"})
			return
		}
		defer rows.Close()

		records := make([]map[string]any, 0)
		for rows.Next() {
			var id int64
			var status string
			var location string
			var reason string
			var occurredAt string
			_ = rows.Scan(&id, &status, &location, &reason, &occurredAt)
			records = append(records, map[string]any{
				"id": id, "status": status, "location": location, "reason": reason, "occurredAt": occurredAt,
			})
		}
		writeJSON(writer, http.StatusOK, map[string]any{"records": records})
	})

	hsrv.HandleFunc("/v1/attendance/summary", func(writer http.ResponseWriter, request *http.Request) {
		userID := request.URL.Query().Get("userId")
		if userID == "" {
			userID = "1"
		}
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

		totalRecordsQuery := "SELECT COUNT(1) FROM attendance_records WHERE user_id = ? AND date(occurred_at) >= " + startDate
		var totalRecords int64
		if err := database.QueryRow(totalRecordsQuery, userID).Scan(&totalRecords); err != nil {
			writeJSON(writer, http.StatusInternalServerError, map[string]string{"error": "summary query failed"})
			return
		}

		var outingCount int64
		outingQuery := "SELECT COUNT(1) FROM attendance_records WHERE user_id = ? AND date(occurred_at) >= " + startDate + " AND status IN ('OUTING','BUSINESS_TRIP')"
		_ = database.QueryRow(outingQuery, userID).Scan(&outingCount)

		var diningCount int64
		diningQuery := "SELECT COUNT(1) FROM attendance_records WHERE user_id = ? AND date(occurred_at) >= " + startDate + " AND status = 'DINING'"
		_ = database.QueryRow(diningQuery, userID).Scan(&diningCount)

		var overtimeHours float64
		if period == "week" || period == "month" {
			var checkoutCount int64
			checkoutQuery := "SELECT COUNT(1) FROM attendance_records WHERE user_id = ? AND date(occurred_at) >= " + startDate + " AND status = 'CHECK_OUT'"
			_ = database.QueryRow(checkoutQuery, userID).Scan(&checkoutCount)
			overtimeHours = float64(checkoutCount) * 1.5
		}

		writeJSON(writer, http.StatusOK, map[string]any{
			"period":        period,
			"totalRecords":  totalRecords,
			"outingCount":   outingCount,
			"diningCount":   diningCount,
			"overtimeHours": overtimeHours,
		})
	})

	hsrv.HandleFunc("/v1/attendance/list", func(writer http.ResponseWriter, request *http.Request) {
		userID := request.URL.Query().Get("userId")
		if userID == "" {
			userID = "1"
		}
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
		rows, err := database.Query(
			"SELECT id, user_id, status, location, reason, occurred_at FROM attendance_records WHERE user_id = ? AND date(occurred_at) >= "+startDate+" ORDER BY occurred_at DESC",
			userID,
		)
		if err != nil {
			writeJSON(writer, http.StatusInternalServerError, map[string]string{"error": "list query failed"})
			return
		}
		defer rows.Close()
		records := make([]map[string]any, 0)
		for rows.Next() {
			var id int64
			var queryUserID int64
			var status string
			var location string
			var reason string
			var occurredAt string
			_ = rows.Scan(&id, &queryUserID, &status, &location, &reason, &occurredAt)
			records = append(records, map[string]any{
				"id": id, "userId": queryUserID, "status": status, "location": location, "reason": reason, "occurredAt": occurredAt,
			})
		}
		writeJSON(writer, http.StatusOK, map[string]any{"period": period, "records": records})
	})

	hsrv.HandleFunc("/v1/attendance/by-date", func(writer http.ResponseWriter, request *http.Request) {
		userID := request.URL.Query().Get("userId")
		if userID == "" {
			userID = "1"
		}
		dateText := request.URL.Query().Get("date")
		if dateText == "" {
			dateText = time.Now().Format("2006-01-02")
		}
		rows, err := database.Query(
			"SELECT id, user_id, status, location, reason, occurred_at FROM attendance_records WHERE user_id = ? AND date(occurred_at) = date(?) ORDER BY occurred_at ASC",
			userID, dateText,
		)
		if err != nil {
			writeJSON(writer, http.StatusInternalServerError, map[string]string{"error": "by-date query failed"})
			return
		}
		defer rows.Close()
		records := make([]map[string]any, 0)
		for rows.Next() {
			var id int64
			var queryUserID int64
			var status string
			var location string
			var reason string
			var occurredAt string
			_ = rows.Scan(&id, &queryUserID, &status, &location, &reason, &occurredAt)
			records = append(records, map[string]any{
				"id": id, "userId": queryUserID, "status": status, "location": location, "reason": reason, "occurredAt": occurredAt,
			})
		}
		writeJSON(writer, http.StatusOK, map[string]any{"date": dateText, "records": records})
	})

	hsrv.HandleFunc("/v1/catalog/locations", func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			writeJSON(writer, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
			return
		}
		rows, err := database.Query(`
SELECT slug, category, region, title, subtitle, detail, sort_order
FROM location_catalog
ORDER BY category, region, sort_order, title`)
		if err != nil {
			writeJSON(writer, http.StatusInternalServerError, map[string]string{"error": "catalog_query_failed"})
			return
		}
		defer rows.Close()
		items := []map[string]any{}
		for rows.Next() {
			var slug, category, region, title, subtitle, detail string
			var sortOrder int
			if err := rows.Scan(&slug, &category, &region, &title, &subtitle, &detail, &sortOrder); err != nil {
				writeJSON(writer, http.StatusInternalServerError, map[string]string{"error": "catalog_scan_failed"})
				return
			}
			items = append(items, map[string]any{
				"slug": slug, "category": category, "region": region,
				"title": title, "subtitle": subtitle, "detail": detail, "sortOrder": sortOrder,
			})
		}
		writeJSON(writer, http.StatusOK, map[string]any{"items": items})
	})

	hsrv.HandleFunc("/v1/catalog/schedule-quick", func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			writeJSON(writer, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
			return
		}
		secRows, err := database.Query(`
SELECT id, sort_order, section_label, item_category, IFNULL(item_region,'')
FROM schedule_quick_section
ORDER BY sort_order ASC, id ASC`)
		if err != nil {
			writeJSON(writer, http.StatusInternalServerError, map[string]string{"error": "schedule_quick_query_failed"})
			return
		}
		defer secRows.Close()
		sections := []map[string]any{}
		for secRows.Next() {
			var sectionID int64
			var sortOrder int
			var sectionLabel, itemCategory, itemRegion string
			if err := secRows.Scan(&sectionID, &sortOrder, &sectionLabel, &itemCategory, &itemRegion); err != nil {
				writeJSON(writer, http.StatusInternalServerError, map[string]string{"error": "schedule_quick_scan_failed"})
				return
			}
			itemRegion = strings.TrimSpace(itemRegion)
			itemRows, err := database.Query(`
SELECT slug, category, region, title, subtitle, detail, sort_order
FROM location_catalog
WHERE category = ? AND ((? = '') OR (region = ?))
ORDER BY sort_order ASC, title ASC`, itemCategory, itemRegion, itemRegion)
			if err != nil {
				writeJSON(writer, http.StatusInternalServerError, map[string]string{"error": "catalog_items_query_failed"})
				return
			}
			items := []map[string]any{}
			for itemRows.Next() {
				var slug, category, region, title, subtitle, detail string
				var itemSort int
				if err := itemRows.Scan(&slug, &category, &region, &title, &subtitle, &detail, &itemSort); err != nil {
					itemRows.Close()
					writeJSON(writer, http.StatusInternalServerError, map[string]string{"error": "catalog_items_scan_failed"})
					return
				}
				items = append(items, map[string]any{
					"slug": slug, "category": category, "region": region,
					"title": title, "subtitle": subtitle, "detail": detail, "sortOrder": itemSort,
				})
			}
			itemRows.Close()
			sections = append(sections, map[string]any{
				"sectionId":    sectionID,
				"sortOrder":    sortOrder,
				"label":        sectionLabel,
				"itemCategory": itemCategory,
				"itemRegion":   itemRegion,
				"items":        items,
			})
		}
		writeJSON(writer, http.StatusOK, map[string]any{"sections": sections})
	})

	registerScheduleDayRoutes(hsrv, database)
	registerRewardRoutes(hsrv, database)
	registerLeadRoutes(hsrv, database)
	registerFortuneRoutes(hsrv, database)

	hsrv.HandleFunc("/healthz", func(writer http.ResponseWriter, request *http.Request) {
		writeJSON(writer, http.StatusOK, map[string]string{"service": "attendance-svc", "status": "ok"})
	})

	app := kratos.New(kratos.Name("attendance-svc"), kratos.Server(hsrv))
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
