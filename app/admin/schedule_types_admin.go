package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	khttp "github.com/go-kratos/kratos/v2/transport/http"
)

func registerScheduleTypesAdminRoutes(hsrv *khttp.Server, database *sql.DB) {
	if hsrv == nil || database == nil {
		return
	}

	hsrv.HandleFunc("/v1/admin/data/activity-types", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			methodNotAllowedJSON(w)
			return
		}
		if _, ok := requireAdminJWT(w, r); !ok {
			return
		}
		rows, err := database.Query(`SELECT code, full_name, description FROM activity_types ORDER BY code`)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "query failed"})
			return
		}
		defer rows.Close()
		list := []map[string]any{}
		for rows.Next() {
			var code, fullName, desc string
			if err := rows.Scan(&code, &fullName, &desc); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "scan failed"})
				return
			}
			list = append(list, map[string]any{"code": code, "fullName": fullName, "description": desc})
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": list})
	})

	hsrv.HandleFunc("/v1/admin/data/activity-types/upsert", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			methodNotAllowedJSON(w)
			return
		}
		actorID, ok := requireAdminJWT(w, r)
		if !ok {
			return
		}
		body, err := readJSONBody(r, adminMaxBodyBytes)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
			return
		}
		var payload struct {
			Code        string `json:"code"`
			FullName    string `json:"fullName"`
			Description string `json:"description"`
		}
		if err := json.Unmarshal(body, &payload); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "json parse failed"})
			return
		}
		payload.Code = strings.TrimSpace(payload.Code)
		payload.FullName = strings.TrimSpace(payload.FullName)
		payload.Description = strings.TrimSpace(payload.Description)
		if payload.Code == "" || payload.FullName == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "code and fullName required"})
			return
		}
		_, err = database.Exec(`
INSERT INTO activity_types (code, full_name, description)
VALUES (?, ?, ?)
ON CONFLICT(code) DO UPDATE SET full_name = excluded.full_name, description = excluded.description`,
			payload.Code, payload.FullName, payload.Description)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "upsert failed"})
			return
		}
		_, _ = database.Exec(`INSERT INTO admin_data_audit (actor_user_id, action, entity, sync_uid, detail) VALUES (?,?,?,?,?)`,
			actorID, "activity_type_upsert", "activity_types", nil, fmt.Sprintf(`{"code":%q}`, payload.Code))
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "code": payload.Code})
	})

	hsrv.HandleFunc("/v1/admin/data/shift-types", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			methodNotAllowedJSON(w)
			return
		}
		if _, ok := requireAdminJWT(w, r); !ok {
			return
		}
		rows, err := database.Query(`SELECT code, name, start_time, end_time, duration_minutes FROM shift_types ORDER BY code`)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "query failed"})
			return
		}
		defer rows.Close()
		list := []map[string]any{}
		for rows.Next() {
			var code, name, start, end string
			var dur int
			if err := rows.Scan(&code, &name, &start, &end, &dur); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "scan failed"})
				return
			}
			list = append(list, map[string]any{
				"code": code, "name": name, "startTime": start, "endTime": end, "durationMinutes": dur,
			})
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": list})
	})

	hsrv.HandleFunc("/v1/admin/data/shift-types/upsert", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			methodNotAllowedJSON(w)
			return
		}
		actorID, ok := requireAdminJWT(w, r)
		if !ok {
			return
		}
		body, err := readJSONBody(r, adminMaxBodyBytes)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
			return
		}
		var payload struct {
			Code             string `json:"code"`
			Name             string `json:"name"`
			StartTime        string `json:"startTime"`
			EndTime          string `json:"endTime"`
			DurationMinutes  int    `json:"durationMinutes"`
		}
		if err := json.Unmarshal(body, &payload); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "json parse failed"})
			return
		}
		payload.Code = strings.TrimSpace(payload.Code)
		payload.Name = strings.TrimSpace(payload.Name)
		payload.StartTime = strings.TrimSpace(payload.StartTime)
		payload.EndTime = strings.TrimSpace(payload.EndTime)
		if payload.Code == "" || payload.Name == "" || payload.StartTime == "" || payload.EndTime == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "code, name, startTime, endTime required"})
			return
		}
		if payload.DurationMinutes <= 0 {
			payload.DurationMinutes = 540
		}
		_, err = database.Exec(`
INSERT INTO shift_types (code, name, start_time, end_time, duration_minutes)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT(code) DO UPDATE SET
  name = excluded.name,
  start_time = excluded.start_time,
  end_time = excluded.end_time,
  duration_minutes = excluded.duration_minutes`,
			payload.Code, payload.Name, payload.StartTime, payload.EndTime, payload.DurationMinutes)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "upsert failed"})
			return
		}
		_, _ = database.Exec(`INSERT INTO admin_data_audit (actor_user_id, action, entity, sync_uid, detail) VALUES (?,?,?,?,?)`,
			actorID, "shift_type_upsert", "shift_types", nil, fmt.Sprintf(`{"code":%q}`, payload.Code))
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "code": payload.Code})
	})
}
