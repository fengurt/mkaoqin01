package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	khttp "github.com/go-kratos/kratos/v2/transport/http"
	"github.com/google/uuid"
)

const (
	adminMaxListLimit   = 500
	adminMaxBodyBytes   = 1 << 20 // 1 MiB
	adminSchemaVersion  = "2026-05-09-v1"
)

var allowedAttendanceStatusesAdmin = map[string]bool{
	"CHECK_IN": true, "OFFICE": true, "OUTING": true, "DINING": true,
	"BUSINESS_TRIP": true, "CHECK_OUT": true,
}

var allowedUserRolesAdmin = map[string]bool{
	"admin": true, "employee": true,
}

func registerAdminDataRoutes(hsrv *khttp.Server, database *sql.DB) {
	if hsrv == nil || database == nil {
		return
	}

	hsrv.HandleFunc("/v1/admin/data/meta", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			methodNotAllowedJSON(w)
			return
		}
		actorID, ok := requireAdminJWT(w, r)
		if !ok {
			return
		}
		handleDataMeta(w, database, actorID)
	})

	hsrv.HandleFunc("/v1/admin/data/schema", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			methodNotAllowedJSON(w)
			return
		}
		if _, ok := requireAdminJWT(w, r); !ok {
			return
		}
		handleDataSchema(w)
	})

	hsrv.HandleFunc("/v1/admin/data/users", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			methodNotAllowedJSON(w)
			return
		}
		if _, ok := requireAdminJWT(w, r); !ok {
			return
		}
		handleListUsers(w, r, database)
	})

	hsrv.HandleFunc("/v1/admin/data/attendance", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			methodNotAllowedJSON(w)
			return
		}
		if _, ok := requireAdminJWT(w, r); !ok {
			return
		}
		handleListAttendance(w, r, database)
	})

	hsrv.HandleFunc("/v1/admin/data/attendance/upsert", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			methodNotAllowedJSON(w)
			return
		}
		actorID, ok := requireAdminJWT(w, r)
		if !ok {
			return
		}
		handleUpsertAttendance(w, r, database, actorID)
	})

	hsrv.HandleFunc("/v1/admin/data/users/patch", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			methodNotAllowedJSON(w)
			return
		}
		actorID, ok := requireAdminJWT(w, r)
		if !ok {
			return
		}
		handlePatchUser(w, r, database, actorID)
	})

	registerCatalogAdminRoutes(hsrv, database)
	registerScheduleTypesAdminRoutes(hsrv, database)
}

func methodNotAllowedJSON(w http.ResponseWriter) {
	writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
}

func handleDataMeta(w http.ResponseWriter, database *sql.DB, actorID int64) {
	var userCount, attendanceCount int64
	if err := database.QueryRow("SELECT COUNT(1) FROM users").Scan(&userCount); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "count users failed"})
		return
	}
	if err := database.QueryRow("SELECT COUNT(1) FROM attendance_records").Scan(&attendanceCount); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "count attendance failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"schemaVersion":      adminSchemaVersion,
		"userCount":          userCount,
		"attendanceCount":    attendanceCount,
		"optimisticLocking":  true,
		"syncUidPrimaryKey":  true,
		"authenticatedActor": actorID,
	})
}

func handleDataSchema(w http.ResponseWriter) {
	writeJSON(w, http.StatusOK, map[string]any{
		"schemaVersion": adminSchemaVersion,
		"tables": []map[string]any{
			{
				"name": "users",
				"columns": []string{
					"id", "account", "password", "role", "display_name",
					"created_at", "updated_at", "sync_uid",
				},
				"notes": "Password is never returned by admin data APIs. Use auth password endpoints.",
			},
			{
				"name": "attendance_records",
				"columns": []string{
					"id", "user_id", "status", "location", "reason", "occurred_at",
					"attachment_url", "created_at", "updated_at", "sync_uid",
				},
			},
			{"name": "admin_data_audit", "columns": []string{"id", "created_at", "actor_user_id", "action", "entity", "sync_uid", "detail"}},
			{"name": "location_catalog", "columns": []string{"id", "slug", "category", "region", "title", "subtitle", "detail", "sort_order", "created_at", "updated_at", "sync_uid"}},
			{"name": "schedule_quick_section", "columns": []string{"id", "sort_order", "section_label", "item_category", "item_region", "created_at", "updated_at"}},
			{"name": "user_daily_schedule", "columns": []string{"user_id", "work_date", "mode", "code", "updated_at"}},
			{"name": "activity_types", "columns": []string{"code", "full_name", "description"}},
			{"name": "shift_types", "columns": []string{"code", "name", "start_time", "end_time", "duration_minutes"}},
		},
	})
}

func handleListUsers(w http.ResponseWriter, r *http.Request, database *sql.DB) {
	limit, offset := parseLimitOffset(r, adminMaxListLimit)
	var total int64
	if err := database.QueryRow("SELECT COUNT(1) FROM users").Scan(&total); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "count failed"})
		return
	}
	rows, err := database.Query(`
SELECT id, account, role, display_name, created_at, IFNULL(updated_at,''), IFNULL(sync_uid,'')
FROM users ORDER BY id ASC LIMIT ? OFFSET ?`, limit, offset)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "query failed"})
		return
	}
	defer rows.Close()
	list := make([]map[string]any, 0)
	for rows.Next() {
		var id int64
		var account, role, displayName, createdAt, updatedAt, syncUID sql.NullString
		if err := rows.Scan(&id, &account, &role, &displayName, &createdAt, &updatedAt, &syncUID); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "scan failed"})
			return
		}
		list = append(list, map[string]any{
			"id": id, "account": account.String, "role": role.String,
			"displayName": displayName.String, "createdAt": createdAt.String,
			"updatedAt": nullStr(updatedAt), "syncUid": nullStr(syncUID),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"total": total, "limit": limit, "offset": offset, "users": list})
}

func handleListAttendance(w http.ResponseWriter, r *http.Request, database *sql.DB) {
	limit, offset := parseLimitOffset(r, adminMaxListLimit)
	userIDStr := r.URL.Query().Get("userId")
	updatedSince := r.URL.Query().Get("updatedSince")

	where := "1=1"
	args := make([]any, 0)
	if userIDStr != "" {
		uid, err := strconv.ParseInt(userIDStr, 10, 64)
		if err != nil || uid < 1 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid userId"})
			return
		}
		where += " AND user_id = ?"
		args = append(args, uid)
	}
	if strings.TrimSpace(updatedSince) != "" {
		where += " AND IFNULL(updated_at, created_at) >= ?"
		args = append(args, strings.TrimSpace(updatedSince))
	}

	countQuery := "SELECT COUNT(1) FROM attendance_records WHERE " + where
	var total int64
	if err := database.QueryRow(countQuery, args...).Scan(&total); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "count failed"})
		return
	}

	args = append(args, limit, offset)
	query := `
SELECT id, user_id, status, location, reason, occurred_at, IFNULL(attachment_url,''),
       created_at, IFNULL(updated_at,''), IFNULL(sync_uid,'')
FROM attendance_records WHERE ` + where + ` ORDER BY id ASC LIMIT ? OFFSET ?`
	rows, err := database.Query(query, args...)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "query failed"})
		return
	}
	defer rows.Close()
	list := make([]map[string]any, 0)
	for rows.Next() {
		var id, userID int64
		var status, location, reason, occurredAt, attachment, createdAt, updatedAt, syncUID string
		if err := rows.Scan(&id, &userID, &status, &location, &reason, &occurredAt, &attachment, &createdAt, &updatedAt, &syncUID); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "scan failed"})
			return
		}
		list = append(list, map[string]any{
			"id": id, "userId": userID, "status": status, "location": location, "reason": reason,
			"occurredAt": occurredAt, "attachmentUrl": attachment,
			"createdAt": createdAt, "updatedAt": updatedAt, "syncUid": syncUID,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"total": total, "limit": limit, "offset": offset, "records": list})
}

type attendanceUpsertBody struct {
	SyncUID          string `json:"syncUid"`
	ID               int64  `json:"id"`
	UserID           int64  `json:"userId"`
	Status           string `json:"status"`
	Location         string `json:"location"`
	Reason           string `json:"reason"`
	OccurredAt       string `json:"occurredAt"`
	AttachmentURL    string `json:"attachmentUrl"`
	IfMatchUpdatedAt string `json:"ifMatchUpdatedAt"`
}

func handleUpsertAttendance(w http.ResponseWriter, r *http.Request, database *sql.DB, actorID int64) {
	body, err := readJSONBody(r, adminMaxBodyBytes)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	var payload attendanceUpsertBody
	if err := json.Unmarshal(body, &payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "json parse failed"})
		return
	}
	payload.Status = strings.TrimSpace(payload.Status)
	if !allowedAttendanceStatusesAdmin[payload.Status] {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid status"})
		return
	}
	if payload.UserID < 1 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "userId required"})
		return
	}
	var userExists int64
	if err := database.QueryRow("SELECT COUNT(1) FROM users WHERE id = ?", payload.UserID).Scan(&userExists); err != nil || userExists == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "userId not found"})
		return
	}

	occurred := strings.TrimSpace(payload.OccurredAt)
	if occurred == "" {
		occurred = time.Now().Format(time.RFC3339)
	} else if _, err := time.Parse(time.RFC3339, occurred); err != nil {
		if _, err2 := time.Parse("2006-01-02 15:04:05", occurred); err2 != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "occurredAt must be RFC3339 or YYYY-MM-DD HH:MM:SS"})
			return
		}
	}

	syncUID := strings.TrimSpace(strings.ToLower(payload.SyncUID))
	if syncUID != "" {
		if _, err := uuid.Parse(syncUID); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "syncUid must be a UUID"})
			return
		}
	}

	tx, err := database.Begin()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "tx begin failed"})
		return
	}
	defer tx.Rollback()

	// Legacy row: id + optional first-time sync_uid assignment
	if payload.ID > 0 && syncUID == "" {
		var dbUpdated, existingSync string
		err := tx.QueryRow(`SELECT IFNULL(updated_at,''), IFNULL(sync_uid,'') FROM attendance_records WHERE id = ?`, payload.ID).Scan(&dbUpdated, &existingSync)
		if err == sql.ErrNoRows {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "record id not found"})
			return
		}
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "lookup failed"})
			return
		}
		if payload.IfMatchUpdatedAt != "" && dbUpdated != payload.IfMatchUpdatedAt {
			writeJSON(w, http.StatusConflict, map[string]any{"error": "conflict", "serverUpdatedAt": dbUpdated, "syncUid": existingSync})
			return
		}
		newSync := existingSync
		if existingSync == "" {
			newSync = uuid.New().String()
		}
		_, err = tx.Exec(`
UPDATE attendance_records SET user_id=?, status=?, location=?, reason=?, occurred_at=?, attachment_url=?,
  sync_uid=?, updated_at=datetime('now')
WHERE id=?`,
			payload.UserID, payload.Status, payload.Location, payload.Reason, occurred, payload.AttachmentURL, newSync, payload.ID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "update failed"})
			return
		}
		auditDetail := fmt.Sprintf(`{"id":%d,"syncUid":"%s"}`, payload.ID, newSync)
		_, _ = tx.Exec(`INSERT INTO admin_data_audit (actor_user_id, action, entity, sync_uid, detail) VALUES (?,?,?,?,?)`,
			actorID, "attendance_upsert", "attendance_records", newSync, auditDetail)
		if err := tx.Commit(); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "commit failed"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "id": payload.ID, "syncUid": newSync})
		return
	}

	if syncUID != "" {
		var existingID int64
		var dbUpdated string
		err := tx.QueryRow(`SELECT id, IFNULL(updated_at,'') FROM attendance_records WHERE sync_uid = ?`, syncUID).Scan(&existingID, &dbUpdated)
		if err != nil && err != sql.ErrNoRows {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "lookup failed"})
			return
		}
		if err == nil {
			if payload.IfMatchUpdatedAt != "" && dbUpdated != payload.IfMatchUpdatedAt {
				writeJSON(w, http.StatusConflict, map[string]any{"error": "conflict", "serverUpdatedAt": dbUpdated, "syncUid": syncUID, "id": existingID})
				return
			}
			_, err = tx.Exec(`
UPDATE attendance_records SET user_id=?, status=?, location=?, reason=?, occurred_at=?, attachment_url=?,
  updated_at=datetime('now') WHERE sync_uid=?`,
				payload.UserID, payload.Status, payload.Location, payload.Reason, occurred, payload.AttachmentURL, syncUID)
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "update failed"})
				return
			}
			_, _ = tx.Exec(`INSERT INTO admin_data_audit (actor_user_id, action, entity, sync_uid, detail) VALUES (?,?,?,?,?)`,
				actorID, "attendance_upsert", "attendance_records", syncUID, `{"op":"update"}`)
			if err := tx.Commit(); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "commit failed"})
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"ok": true, "id": existingID, "syncUid": syncUID})
			return
		}
	}

	newSync := syncUID
	if newSync == "" {
		newSync = uuid.New().String()
	}
	res, err := tx.Exec(`
INSERT INTO attendance_records (user_id, status, location, reason, occurred_at, attachment_url, sync_uid, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, datetime('now'))`,
		payload.UserID, payload.Status, payload.Location, payload.Reason, occurred, payload.AttachmentURL, newSync)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "insert failed"})
		return
	}
	newID, _ := res.LastInsertId()
	_, _ = tx.Exec(`INSERT INTO admin_data_audit (actor_user_id, action, entity, sync_uid, detail) VALUES (?,?,?,?,?)`,
		actorID, "attendance_upsert", "attendance_records", newSync, `{"op":"insert"}`)
	if err := tx.Commit(); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "commit failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "id": newID, "syncUid": newSync})
}

type userPatchBody struct {
	ID               int64  `json:"id"`
	SyncUID          string `json:"syncUid"`
	DisplayName      string `json:"displayName"`
	Role             string `json:"role"`
	IfMatchUpdatedAt string `json:"ifMatchUpdatedAt"`
}

func handlePatchUser(w http.ResponseWriter, r *http.Request, database *sql.DB, actorID int64) {
	body, err := readJSONBody(r, adminMaxBodyBytes)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	var payload userPatchBody
	if err := json.Unmarshal(body, &payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "json parse failed"})
		return
	}
	payload.DisplayName = strings.TrimSpace(payload.DisplayName)
	payload.Role = strings.TrimSpace(payload.Role)
	payload.SyncUID = strings.TrimSpace(strings.ToLower(payload.SyncUID))

	if payload.ID < 1 && payload.SyncUID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "id or syncUid required"})
		return
	}
	if payload.DisplayName == "" && payload.Role == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "displayName or role required"})
		return
	}
	if payload.Role != "" && !allowedUserRolesAdmin[payload.Role] {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid role"})
		return
	}

	var rowID int64
	var dbUpdated, existingSync string
	var qerr error
	if payload.SyncUID != "" {
		qerr = database.QueryRow(`SELECT id, IFNULL(updated_at,''), IFNULL(sync_uid,'') FROM users WHERE sync_uid = ?`, payload.SyncUID).Scan(&rowID, &dbUpdated, &existingSync)
	} else {
		qerr = database.QueryRow(`SELECT id, IFNULL(updated_at,''), IFNULL(sync_uid,'') FROM users WHERE id = ?`, payload.ID).Scan(&rowID, &dbUpdated, &existingSync)
	}
	if qerr == sql.ErrNoRows {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "user not found"})
		return
	}
	if qerr != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "lookup failed"})
		return
	}
	if payload.IfMatchUpdatedAt != "" && dbUpdated != payload.IfMatchUpdatedAt {
		writeJSON(w, http.StatusConflict, map[string]any{"error": "conflict", "serverUpdatedAt": dbUpdated})
		return
	}

	newSync := existingSync
	if existingSync == "" {
		newSync = uuid.New().String()
	}

	sets := []string{"sync_uid = ?", "updated_at = datetime('now')"}
	args := []any{newSync}
	if payload.DisplayName != "" {
		sets = append(sets, "display_name = ?")
		args = append(args, payload.DisplayName)
	}
	if payload.Role != "" {
		sets = append(sets, "role = ?")
		args = append(args, payload.Role)
	}
	args = append(args, rowID)
	query := "UPDATE users SET " + strings.Join(sets, ", ") + " WHERE id = ?"
	_, execErr := database.Exec(query, args...)
	if execErr != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "update failed"})
		return
	}
	_, _ = database.Exec(`INSERT INTO admin_data_audit (actor_user_id, action, entity, sync_uid, detail) VALUES (?,?,?,?,?)`,
		actorID, "user_patch", "users", newSync, fmt.Sprintf(`{"userId":%d}`, rowID))
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "id": rowID, "syncUid": newSync})
}

func readJSONBody(r *http.Request, maxBytes int64) ([]byte, error) {
	defer r.Body.Close()
	lr := io.LimitReader(r.Body, maxBytes+1)
	b, err := io.ReadAll(lr)
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > maxBytes {
		return nil, fmt.Errorf("too large")
	}
	return b, nil
}

func parseLimitOffset(r *http.Request, maxCap int) (limit int, offset int) {
	limit = 100
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	if limit > maxCap {
		limit = maxCap
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}
	return limit, offset
}

func nullStr(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}
