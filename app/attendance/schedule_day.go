package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	khttp "github.com/go-kratos/kratos/v2/transport/http"
)

func registerScheduleDayRoutes(hsrv *khttp.Server, database *sql.DB) {
	if hsrv == nil || database == nil {
		return
	}

	hsrv.HandleFunc("/v1/schedule/day-options", func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			writeJSON(writer, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
			return
		}
		shifts := []map[string]any{}
		srows, err := database.Query(`SELECT code, name, start_time, end_time, duration_minutes FROM shift_types ORDER BY code`)
		if err != nil {
			writeJSON(writer, http.StatusInternalServerError, map[string]string{"error": "shift_types_query_failed"})
			return
		}
		defer srows.Close()
		for srows.Next() {
			var code, name, start, end string
			var dur int
			if err := srows.Scan(&code, &name, &start, &end, &dur); err != nil {
				writeJSON(writer, http.StatusInternalServerError, map[string]string{"error": "shift_types_scan_failed"})
				return
			}
			shifts = append(shifts, map[string]any{
				"code": code, "name": name, "startTime": start, "endTime": end, "durationMinutes": dur,
			})
		}
		leaves := []map[string]any{}
		lrows, err := database.Query(`SELECT code, full_name, description FROM activity_types ORDER BY code`)
		if err != nil {
			writeJSON(writer, http.StatusInternalServerError, map[string]string{"error": "activity_types_query_failed"})
			return
		}
		defer lrows.Close()
		for lrows.Next() {
			var code, fullName, desc string
			if err := lrows.Scan(&code, &fullName, &desc); err != nil {
				writeJSON(writer, http.StatusInternalServerError, map[string]string{"error": "activity_types_scan_failed"})
				return
			}
			leaves = append(leaves, map[string]any{
				"code": code, "fullName": fullName, "description": desc,
			})
		}
		writeJSON(writer, http.StatusOK, map[string]any{"workShifts": shifts, "leaveTypes": leaves})
	})

	hsrv.HandleFunc("/v1/schedule/day", func(writer http.ResponseWriter, request *http.Request) {
		switch request.Method {
		case http.MethodGet:
			handleScheduleDayGet(writer, request, database)
		case http.MethodPost:
			handleScheduleDaySet(writer, request, database)
		default:
			writeJSON(writer, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		}
	})

	hsrv.HandleFunc("/v1/schedule/month", func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			writeJSON(writer, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
			return
		}
		userIDStr := request.URL.Query().Get("userId")
		yearMonth := strings.TrimSpace(request.URL.Query().Get("month"))
		if yearMonth == "" {
			now := time.Now()
			yearMonth = now.Format("2006-01")
		}
		if len(yearMonth) != 7 || yearMonth[4] != '-' {
			writeJSON(writer, http.StatusBadRequest, map[string]string{"error": "invalid month"})
			return
		}
		userID := int64(1)
		if userIDStr != "" {
			parsed, err := strconv.ParseInt(userIDStr, 10, 64)
			if err != nil || parsed < 1 {
				writeJSON(writer, http.StatusBadRequest, map[string]string{"error": "invalid userId"})
				return
			}
			userID = parsed
		}
		from := yearMonth + "-01"
		toTime, err := time.Parse("2006-01-02", from)
		if err != nil {
			writeJSON(writer, http.StatusBadRequest, map[string]string{"error": "invalid month"})
			return
		}
		to := time.Date(toTime.Year(), toTime.Month()+1, 0, 0, 0, 0, 0, toTime.Location()).Format("2006-01-02")
		rows, err := database.Query(`
SELECT work_date, mode, code FROM user_daily_schedule
WHERE user_id = ? AND work_date >= ? AND work_date <= ?
ORDER BY work_date ASC`, userID, from, to)
		if err != nil {
			writeJSON(writer, http.StatusInternalServerError, map[string]string{"error": "query_failed"})
			return
		}
		defer rows.Close()
		days := map[string]any{}
		for rows.Next() {
			var workDate, mode, code string
			if err := rows.Scan(&workDate, &mode, &code); err != nil {
				writeJSON(writer, http.StatusInternalServerError, map[string]string{"error": "scan_failed"})
				return
			}
			payload, err := resolveUserDailySchedule(database, userID, workDate)
			if err != nil {
				writeJSON(writer, http.StatusInternalServerError, map[string]string{"error": "resolve_failed"})
				return
			}
			short := scheduleShortLabel(payload)
			days[workDate] = map[string]any{
				"mode": payload["mode"], "code": payload["code"],
				"shortLabel": short, "calendarLabel": scheduleCalendarLabel(payload),
				"pillText": payload["pillText"],
				"startTime": payload["startTime"], "endTime": payload["endTime"],
			}
		}
		writeJSON(writer, http.StatusOK, map[string]any{"userId": userID, "month": yearMonth, "days": days})
	})
}

func scheduleShortLabel(payload map[string]any) string {
	return scheduleCalendarLabel(payload)
}

func scheduleCalendarLabel(payload map[string]any) string {
	mode, _ := payload["mode"].(string)
	code, _ := payload["code"].(string)
	code = strings.TrimSpace(code)
	if mode == "leave" {
		if code != "" {
			return code
		}
		return "休"
	}
	if code == "STANDBY24" {
		return "24h待命"
	}
	if code != "" {
		return code
	}
	label, _ := payload["label"].(string)
	if label != "" {
		return label
	}
	return "班"
}

func handleScheduleDayGet(writer http.ResponseWriter, request *http.Request, database *sql.DB) {
	userIDStr := request.URL.Query().Get("userId")
	dateStr := strings.TrimSpace(request.URL.Query().Get("date"))
	if dateStr == "" {
		writeJSON(writer, http.StatusBadRequest, map[string]string{"error": "date required"})
		return
	}
	if _, err := parseScheduleDate(dateStr); err != nil {
		writeJSON(writer, http.StatusBadRequest, map[string]string{"error": "invalid date"})
		return
	}
	userID := int64(1)
	if userIDStr != "" {
		parsed, err := strconv.ParseInt(userIDStr, 10, 64)
		if err != nil || parsed < 1 {
			writeJSON(writer, http.StatusBadRequest, map[string]string{"error": "invalid userId"})
			return
		}
		userID = parsed
	}
	payload, err := resolveUserDailySchedule(database, userID, dateStr)
	if err != nil {
		writeJSON(writer, http.StatusInternalServerError, map[string]string{"error": "schedule_resolve_failed"})
		return
	}
	writeJSON(writer, http.StatusOK, payload)
}

type scheduleDaySetBody struct {
	UserID int64  `json:"userId"`
	Date   string `json:"date"`
	Mode   string `json:"mode"`
	Code   string `json:"code"`
}

func handleScheduleDaySet(writer http.ResponseWriter, request *http.Request, database *sql.DB) {
	var body scheduleDaySetBody
	if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
		writeJSON(writer, http.StatusBadRequest, map[string]string{"error": "invalid payload"})
		return
	}
	if body.UserID < 1 {
		body.UserID = 1
	}
	body.Date = strings.TrimSpace(body.Date)
	body.Mode = strings.TrimSpace(strings.ToLower(body.Mode))
	body.Code = strings.TrimSpace(body.Code)
	if body.Date == "" {
		writeJSON(writer, http.StatusBadRequest, map[string]string{"error": "date required"})
		return
	}
	if _, err := parseScheduleDate(body.Date); err != nil {
		writeJSON(writer, http.StatusBadRequest, map[string]string{"error": "invalid date"})
		return
	}
	if body.Mode != "work" && body.Mode != "leave" {
		writeJSON(writer, http.StatusBadRequest, map[string]string{"error": "mode must be work or leave"})
		return
	}
	if body.Code == "" {
		writeJSON(writer, http.StatusBadRequest, map[string]string{"error": "code required"})
		return
	}
	if body.Mode == "work" {
		var n int
		if err := database.QueryRow(`SELECT COUNT(1) FROM shift_types WHERE code = ?`, body.Code).Scan(&n); err != nil || n == 0 {
			writeJSON(writer, http.StatusBadRequest, map[string]string{"error": "unknown shift code"})
			return
		}
	} else {
		var n int
		if err := database.QueryRow(`SELECT COUNT(1) FROM activity_types WHERE code = ?`, body.Code).Scan(&n); err != nil || n == 0 {
			writeJSON(writer, http.StatusBadRequest, map[string]string{"error": "unknown leave code"})
			return
		}
	}
	_, err := database.Exec(`
INSERT INTO user_daily_schedule (user_id, work_date, mode, code, updated_at)
VALUES (?, ?, ?, ?, datetime('now'))
ON CONFLICT(user_id, work_date) DO UPDATE SET mode = excluded.mode, code = excluded.code, updated_at = datetime('now')`,
		body.UserID, body.Date, body.Mode, body.Code)
	if err != nil {
		msg := strings.ToLower(err.Error())
		switch {
		case strings.Contains(msg, "no such table"):
			writeJSON(writer, http.StatusInternalServerError, map[string]string{"error": "schedule_db_pending"})
		case strings.Contains(msg, "foreign key constraint") || strings.Contains(msg, "constraint failed"):
			writeJSON(writer, http.StatusBadRequest, map[string]string{"error": "schedule_user_unknown"})
		default:
			writeJSON(writer, http.StatusInternalServerError, map[string]string{"error": "schedule_save_failed"})
		}
		return
	}
	payload, err := resolveUserDailySchedule(database, body.UserID, body.Date)
	if err != nil {
		writeJSON(writer, http.StatusInternalServerError, map[string]string{"error": "schedule_resolve_failed"})
		return
	}
	writeJSON(writer, http.StatusOK, payload)
}

func parseScheduleDate(s string) (string, error) {
	if len(s) != 10 || s[4] != '-' || s[7] != '-' {
		return "", errors.New("invalid date")
	}
	return s, nil
}

func resolveUserDailySchedule(database *sql.DB, userID int64, dateStr string) (map[string]any, error) {
	var mode, code string
	err := database.QueryRow(`
SELECT mode, code FROM user_daily_schedule WHERE user_id = ? AND work_date = ?`, userID, dateStr).Scan(&mode, &code)
	if err == sql.ErrNoRows {
		mode = "work"
		code = "OFFICE"
	} else if err != nil {
		return nil, err
	}
	mode = strings.TrimSpace(strings.ToLower(mode))
	code = strings.TrimSpace(code)
	if mode == "leave" {
		var fullName, desc string
		qerr := database.QueryRow(`SELECT full_name, description FROM activity_types WHERE code = ?`, code).Scan(&fullName, &desc)
		if qerr == sql.ErrNoRows {
			fullName, desc = code, ""
		} else if qerr != nil {
			return nil, qerr
		}
		tail := strings.TrimSpace(desc)
		if tail == "" {
			tail = strings.TrimSpace(fullName)
		}
		pill := strings.TrimSpace(code + " " + tail)
		return map[string]any{
			"userId": userID, "date": dateStr, "mode": "leave", "code": code,
			"label": code, "fullName": fullName, "description": desc,
			"pillText": "休假-" + strings.TrimSpace(pill),
			"startTime": "", "endTime": "",
		}, nil
	}
	storedWorkCode := code
	var name, start, end string
	var dur int
	qerr := database.QueryRow(`SELECT name, start_time, end_time, duration_minutes FROM shift_types WHERE code = ?`, code).Scan(&name, &start, &end, &dur)
	if qerr == sql.ErrNoRows {
		qerr = database.QueryRow(`SELECT name, start_time, end_time, duration_minutes FROM shift_types WHERE code = 'OFFICE'`).Scan(&name, &start, &end, &dur)
	}
	if qerr != nil {
		return nil, qerr
	}
	return map[string]any{
		"userId": userID, "date": dateStr, "mode": "work", "code": storedWorkCode,
		"label": name, "fullName": name, "description": "",
		"pillText": "常规班-" + name + " " + formatTimeRange(start, end),
		"startTime": start, "endTime": end, "durationMinutes": dur,
	}, nil
}

func formatTimeRange(start, end string) string {
	return start + "–" + end
}
