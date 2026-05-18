package main

import (
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode"

	khttp "github.com/go-kratos/kratos/v2/transport/http"
)

const scheduleGridSchema = "intervoice.scheduleGrid.v1"
const scheduleGridSchemaVersion = 2

var leaveTagNames = map[string]bool{
	"RDO": true, "AL": true, "PH": true, "PHCL": true, "RDOC": true, "RV": true,
}

func registerScheduleGridAdminRoutes(hsrv *khttp.Server, database *sql.DB) {
	hsrv.HandleFunc("/v1/admin/schedule/grid/export", func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			writeJSON(writer, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
			return
		}
		from := strings.TrimSpace(request.URL.Query().Get("from"))
		to := strings.TrimSpace(request.URL.Query().Get("to"))
		if from == "" || to == "" {
			now := time.Now()
			from = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).Format("2006-01-02")
			to = time.Date(now.Year(), now.Month()+1, 0, 0, 0, 0, 0, now.Location()).Format("2006-01-02")
		}
		payload, err := buildScheduleGridExport(database, from, to)
		if err != nil {
			writeJSON(writer, http.StatusInternalServerError, map[string]string{"error": "export_failed: " + err.Error()})
			return
		}
		writeJSON(writer, http.StatusOK, payload)
	})

	hsrv.HandleFunc("/v1/admin/schedule/grid/import", func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			writeJSON(writer, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
			return
		}
		bodyBytes, err := io.ReadAll(request.Body)
		if err != nil {
			writeJSON(writer, http.StatusBadRequest, map[string]string{"error": "read_failed"})
			return
		}
		var raw any
		if err := json.Unmarshal(bodyBytes, &raw); err != nil {
			writeJSON(writer, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
			return
		}
		result, err := importScheduleGrid(database, raw)
		if err != nil {
			writeJSON(writer, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(writer, http.StatusOK, result)
	})
}

func buildScheduleGridExport(database *sql.DB, from, to string) (map[string]any, error) {
	allUsers, err := listExportUsers(database)
	if err != nil {
		return nil, err
	}
	employees := filterRosterUsers(allUsers)
	roster := buildChronoscapeRoster(employees)

	rows, err := database.Query(`
SELECT u.id, u.account, u.display_name, s.work_date, s.mode, s.code
FROM user_daily_schedule s
JOIN users u ON u.id = s.user_id
WHERE s.work_date >= ? AND s.work_date <= ?
ORDER BY u.id, s.work_date`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	assignments := []map[string]any{}
	for rows.Next() {
		var userID int64
		var account, displayName, workDate, mode, code string
		if err := rows.Scan(&userID, &account, &displayName, &workDate, &mode, &code); err != nil {
			return nil, err
		}
		tagItemName := lookupTagItemName(database, mode, code)
		assignments = append(assignments, map[string]any{
			"userId": userID, "account": account, "displayName": displayName,
			"objectName": displayName,
			"date": workDate, "mode": mode, "code": code, "tagItemName": tagItemName,
		})
	}
	shiftCatalog, err := listShiftTypes(database)
	if err != nil {
		return nil, err
	}
	leaveCatalog, err := listLeaveTypes(database)
	if err != nil {
		return nil, err
	}
	activeTags := buildActiveTagsCatalog(shiftCatalog, leaveCatalog)

	return map[string]any{
		"schema":           scheduleGridSchema,
		"schemaVersion":    scheduleGridSchemaVersion,
		"exportedAt":       time.Now().UTC().Format(time.RFC3339),
		"dateRange":        map[string]string{"start": from, "end": to},
		"users":            allUsers,
		"employees":        employees,
		"roster":           roster,
		"shiftTypeCatalog": shiftCatalog,
		"leaveTypeCatalog": leaveCatalog,
		"activeTags":       activeTags,
		"assignments":      assignments,
		"importHints": map[string]any{
			"matchUserBy":      []string{"userId", "account", "displayName", "objectName", "normalizedName"},
			"matchShiftBy":     []string{"tagItemName", "code", "mode+code"},
			"chronoscapeCells": "cells[] with objectName + date + tagItemName also supported on import",
		},
	}, nil
}

func listExportUsers(database *sql.DB) ([]map[string]any, error) {
	rows, err := database.Query(`
SELECT id, account, role, display_name FROM users ORDER BY role ASC, display_name ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id int64
		var account, role, displayName string
		if err := rows.Scan(&id, &account, &role, &displayName); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{
			"userId":         id,
			"account":        account,
			"role":           role,
			"displayName":    displayName,
			"normalizedName": normalizePersonName(displayName),
		})
	}
	return out, nil
}

func filterRosterUsers(users []map[string]any) []map[string]any {
	out := []map[string]any{}
	for _, u := range users {
		role, _ := u["role"].(string)
		if strings.EqualFold(strings.TrimSpace(role), "admin") {
			continue
		}
		out = append(out, u)
	}
	return out
}

func buildChronoscapeRoster(employees []map[string]any) []map[string]any {
	roster := make([]map[string]any, 0, len(employees))
	for i, u := range employees {
		userID := toInt64(u["userId"])
		displayName, _ := u["displayName"].(string)
		account, _ := u["account"].(string)
		roster = append(roster, map[string]any{
			"simObjectId": 70000 + userID,
			"objectId":    userID,
			"objectName":  displayName,
			"account":     account,
			"sortOrder":   i + 1,
			"isActive":    true,
		})
	}
	return roster
}

func buildActiveTagsCatalog(shiftCatalog, leaveCatalog []map[string]any) []map[string]any {
	tags := []map[string]any{}
	tagItemID := 2001
	for _, row := range leaveCatalog {
		code, _ := row["code"].(string)
		if code == "" {
			continue
		}
		tags = append(tags, map[string]any{
			"tagId": 20, "tagItemId": tagItemID, "isSystem": false,
			"tagName": "排班类型", "tagItemName": code,
			"mode": "leave", "code": code,
		})
		tagItemID++
	}
	for _, row := range shiftCatalog {
		code, _ := row["code"].(string)
		name, _ := row["name"].(string)
		tagItemName := exportTagItemNameForShift(code, name)
		if tagItemName == "" {
			continue
		}
		tags = append(tags, map[string]any{
			"tagId": 20, "tagItemId": tagItemID, "isSystem": false,
			"tagName": "排班类型", "tagItemName": tagItemName,
			"mode": "work", "code": code,
		})
		tagItemID++
	}
	return tags
}

func exportTagItemNameForShift(code, name string) string {
	if code == "STANDBY24" {
		return "24 hours available on mobile"
	}
	compact := strings.ReplaceAll(strings.TrimSpace(name), " ", "")
	if shiftTimePattern.MatchString(compact) {
		return compact
	}
	if strings.TrimSpace(name) != "" {
		return strings.TrimSpace(name)
	}
	return strings.TrimSpace(code)
}

func lookupTagItemName(database *sql.DB, mode, code string) string {
	mode = strings.TrimSpace(strings.ToLower(mode))
	code = strings.TrimSpace(code)
	if mode == "leave" {
		return code
	}
	if code == "STANDBY24" {
		return "24 hours available on mobile"
	}
	var name string
	err := database.QueryRow(`SELECT name FROM shift_types WHERE code = ?`, code).Scan(&name)
	if err == nil {
		return exportTagItemNameForShift(code, name)
	}
	return code
}

func listShiftTypes(database *sql.DB) ([]map[string]any, error) {
	rows, err := database.Query(`SELECT code, name, start_time, end_time, duration_minutes FROM shift_types ORDER BY code`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var code, name, start, end string
		var dur int
		_ = rows.Scan(&code, &name, &start, &end, &dur)
		out = append(out, map[string]any{"code": code, "name": name, "startTime": start, "endTime": end, "durationMinutes": dur})
	}
	return out, nil
}

func listLeaveTypes(database *sql.DB) ([]map[string]any, error) {
	rows, err := database.Query(`SELECT code, full_name, description FROM activity_types ORDER BY code`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var code, fullName, desc string
		_ = rows.Scan(&code, &fullName, &desc)
		out = append(out, map[string]any{"code": code, "fullName": fullName, "description": desc})
	}
	return out, nil
}

func importScheduleGrid(database *sql.DB, raw any) (map[string]any, error) {
	root, ok := raw.(map[string]any)
	if !ok {
		return nil, errImport("root must be object")
	}
	schema, _ := root["schema"].(string)
	assignments := []importAssignment{}
	unmatched := []map[string]any{}

	if schema == "chronoscape.simulationProjectData" || root["cells"] != nil {
		cells, _ := root["cells"].([]any)
		for _, cell := range cells {
			m, ok := cell.(map[string]any)
			if !ok {
				continue
			}
			name, _ := firstString(m, "objectName", "displayName")
			date, _ := m["date"].(string)
			tag, _ := firstString(m, "tagItemName", "displayName")
			assignments = append(assignments, importAssignment{displayName: name, date: date, tagName: tag})
		}
	} else {
		items, _ := root["assignments"].([]any)
		for _, item := range items {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			assignments = append(assignments, importAssignment{
				userID:      toInt64(m["userId"]),
				account:     str(m["account"]),
				displayName: str(m["displayName"]),
				date:        str(m["date"]),
				mode:        str(m["mode"]),
				code:        str(m["code"]),
				tagName:     str(m["code"]),
			})
		}
	}

	applied := 0
	skipped := 0
	for _, row := range assignments {
		if row.date == "" {
			skipped++
			continue
		}
		userID := row.userID
		if userID < 1 {
			var err error
			userID, err = resolveUserIDByName(database, row.displayName, row.account)
			if err != nil {
				unmatched = append(unmatched, map[string]any{"displayName": row.displayName, "date": row.date, "reason": err.Error()})
				skipped++
				continue
			}
		}
		mode, code, err := mapTagToSchedule(database, row.mode, row.code, row.tagName)
		if err != nil {
			unmatched = append(unmatched, map[string]any{"userId": userID, "date": row.date, "tag": row.tagName, "reason": err.Error()})
			skipped++
			continue
		}
		_, err = database.Exec(`
INSERT INTO user_daily_schedule (user_id, work_date, mode, code, updated_at)
VALUES (?, ?, ?, ?, datetime('now'))
ON CONFLICT(user_id, work_date) DO UPDATE SET mode = excluded.mode, code = excluded.code, updated_at = datetime('now')`,
			userID, row.date, mode, code)
		if err != nil {
			skipped++
			continue
		}
		applied++
	}
	return map[string]any{
		"applied": applied, "skipped": skipped, "unmatched": unmatched,
	}, nil
}

type importAssignment struct {
	userID      int64
	account     string
	displayName string
	date        string
	mode        string
	code        string
	tagName     string
}

func errImport(msg string) error {
	return &importError{msg: msg}
}

type importError struct{ msg string }

func (e *importError) Error() string { return e.msg }

func firstString(m map[string]any, keys ...string) (string, bool) {
	for _, k := range keys {
		if v, ok := m[k].(string); ok && strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v), true
		}
	}
	return "", false
}

func str(v any) string {
	if s, ok := v.(string); ok {
		return strings.TrimSpace(s)
	}
	return ""
}

func toInt64(v any) int64 {
	switch t := v.(type) {
	case float64:
		return int64(t)
	case int64:
		return t
	case int:
		return int64(t)
	default:
		return 0
	}
}

func resolveUserIDByName(database *sql.DB, displayName, account string) (int64, error) {
	displayName = strings.TrimSpace(displayName)
	account = strings.TrimSpace(account)
	if account != "" {
		var id int64
		err := database.QueryRow(`SELECT id FROM users WHERE account = ?`, account).Scan(&id)
		if err == nil {
			return id, nil
		}
	}
	if displayName == "" {
		return 0, errImport("missing user identity")
	}
	var id int64
	err := database.QueryRow(`SELECT id FROM users WHERE lower(trim(display_name)) = lower(trim(?))`, displayName).Scan(&id)
	if err == nil {
		return id, nil
	}
	rows, err := database.Query(`SELECT id, display_name FROM users WHERE role = 'employee'`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	normTarget := normalizePersonName(displayName)
	for rows.Next() {
		var uid int64
		var name string
		_ = rows.Scan(&uid, &name)
		if normalizePersonName(name) == normTarget {
			return uid, nil
		}
	}
	return 0, errImport("user not found: " + displayName)
}

func normalizePersonName(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

var shiftTimePattern = regexp.MustCompile(`^(\d{1,2})[:.]?(\d{2})-(\d{1,2})[:.]?(\d{2})$`)

func mapTagToSchedule(database *sql.DB, mode, code, tagName string) (string, string, error) {
	mode = strings.ToLower(strings.TrimSpace(mode))
	code = strings.TrimSpace(code)
	tagName = strings.TrimSpace(tagName)
	if mode == "work" && code != "" {
		return "work", code, nil
	}
	if mode == "leave" && code != "" {
		return "leave", code, nil
	}
	label := tagName
	if label == "" {
		label = code
	}
	if label == "" {
		return "", "", errImport("empty shift tag")
	}
	upper := strings.ToUpper(label)
	if leaveTagNames[upper] || leaveTagNames[label] {
		if err := ensureLeaveType(database, upper); err != nil {
			return "", "", err
		}
		return "leave", upper, nil
	}
	if strings.Contains(strings.ToLower(label), "24 hour") || strings.Contains(strings.ToLower(label), "24 hours") {
		if err := ensureShiftType(database, "STANDBY24", "24小时手机待命", "00:00", "23:59", 1440); err != nil {
			return "", "", err
		}
		return "work", "STANDBY24", nil
	}
	if m := shiftTimePattern.FindStringSubmatch(strings.ReplaceAll(label, " ", "")); len(m) == 5 {
		start := padTime(m[1], m[2])
		end := padTime(m[3], m[4])
		shiftCode := strings.ReplaceAll(label, " ", "")
		if err := ensureShiftType(database, shiftCode, label, start, end, 576); err != nil {
			return "", "", err
		}
		return "work", shiftCode, nil
	}
	if err := ensureShiftType(database, shiftCodeSafe(label), label, "09:00", "18:00", 540); err != nil {
		return "", "", err
	}
	return "work", shiftCodeSafe(label), nil
}

func padTime(hh, mm string) string {
	if len(hh) == 1 {
		hh = "0" + hh
	}
	if len(mm) == 1 {
		mm = "0" + mm
	}
	return hh + ":" + mm
}

func shiftCodeSafe(label string) string {
	code := strings.ToUpper(strings.ReplaceAll(label, " ", "_"))
	code = strings.ReplaceAll(code, ":", "")
	code = strings.ReplaceAll(code, "-", "_")
	if len(code) > 48 {
		code = code[:48]
	}
	return code
}

func ensureLeaveType(database *sql.DB, code string) error {
	_, err := database.Exec(`
INSERT OR IGNORE INTO activity_types (code, full_name, description) VALUES (?, ?, ?)`,
		code, code, "导入自动创建")
	return err
}

func ensureShiftType(database *sql.DB, code, name, start, end string, dur int) error {
	_, err := database.Exec(`
INSERT OR IGNORE INTO shift_types (code, name, start_time, end_time, duration_minutes) VALUES (?, ?, ?, ?, ?)`,
		code, name, start, end, dur)
	return err
}
