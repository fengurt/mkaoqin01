package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	khttp "github.com/go-kratos/kratos/v2/transport/http"
	"intervoice/dbschema"
)

// Closed set for client_leads.status (application + PostgreSQL CHECK mirror).
var allowedLeadStatuses = map[string]struct{}{
	"NEW": {}, "ASSIGNED": {}, "PICKED_UP": {}, "IN_FOLLOW_UP": {}, "WON": {}, "LOST": {}, "CANCELED": {},
}

func validLeadStatus(status string) bool {
	_, ok := allowedLeadStatuses[status]
	return ok
}

func registerLeadRoutes(hsrv *khttp.Server, database *sql.DB) {
	hsrv.HandleFunc("/v1/leads/feed", func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			writeJSON(writer, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
			return
		}
		userID := request.URL.Query().Get("userId")
		if userID == "" {
			userID = "1"
		}
		rows, err := database.Query(`
SELECT `+dbschema.LeadWideSelectColumns+`
FROM client_leads
WHERE status NOT IN ('WON','LOST','CANCELED')
  AND (assigned_user_id = ? OR picked_up_by = ? OR status = 'NEW')
ORDER BY updated_at DESC
LIMIT 30`, userID, userID)
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

	hsrv.HandleFunc("/v1/leads/pick-up", func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			writeJSON(writer, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
			return
		}
		var body struct {
			LeadID int64 `json:"leadId"`
			UserID int64 `json:"userId"`
		}
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			writeJSON(writer, http.StatusBadRequest, map[string]string{"error": "invalid_payload"})
			return
		}
		if body.UserID == 0 {
			body.UserID = 1
		}
		if body.LeadID == 0 {
			writeJSON(writer, http.StatusBadRequest, map[string]string{"error": "lead_id_required"})
			return
		}
		now := time.Now().Format(time.RFC3339)
		res, err := database.Exec(`
UPDATE client_leads SET status = 'PICKED_UP', picked_up_by = ?, picked_up_at = ?, updated_at = datetime('now')
WHERE id = ? AND status IN ('NEW','ASSIGNED')`, body.UserID, now, body.LeadID)
		if err != nil {
			writeJSON(writer, http.StatusInternalServerError, map[string]string{"error": "update_failed"})
			return
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			writeJSON(writer, http.StatusConflict, map[string]string{"error": "lead_not_available"})
			return
		}
		payload, _ := json.Marshal(map[string]any{"userId": body.UserID})
		_, _ = database.Exec(`INSERT INTO client_lead_events (lead_id, user_id, event_type, payload) VALUES (?, ?, 'PICK_UP', ?)`,
			body.LeadID, body.UserID, string(payload))
		writeJSON(writer, http.StatusOK, map[string]string{"status": "ok"})
	})

	hsrv.HandleFunc("/v1/leads/follow-up", func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			writeJSON(writer, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
			return
		}
		var body struct {
			LeadID   int64  `json:"leadId"`
			UserID   int64  `json:"userId"`
			Note     string `json:"note"`
			StatusTo string `json:"statusTo"`
		}
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			writeJSON(writer, http.StatusBadRequest, map[string]string{"error": "invalid_payload"})
			return
		}
		if body.UserID == 0 {
			body.UserID = 1
		}
		if body.LeadID == 0 || strings.TrimSpace(body.Note) == "" {
			writeJSON(writer, http.StatusBadRequest, map[string]string{"error": "invalid_fields"})
			return
		}
		if body.StatusTo == "" {
			body.StatusTo = "IN_FOLLOW_UP"
		}
		if !validLeadStatus(body.StatusTo) {
			writeJSON(writer, http.StatusBadRequest, map[string]string{"error": "invalid_lead_status"})
			return
		}
		res, err := database.Exec(`
UPDATE client_leads SET status = ?, updated_at = datetime('now')
WHERE id = ? AND (picked_up_by = ? OR assigned_user_id = ?)`,
			body.StatusTo, body.LeadID, body.UserID, body.UserID)
		if err != nil {
			writeJSON(writer, http.StatusInternalServerError, map[string]string{"error": "update_failed"})
			return
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			writeJSON(writer, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}
		payload, _ := json.Marshal(map[string]any{"note": body.Note, "statusTo": body.StatusTo})
		_, _ = database.Exec(`INSERT INTO client_lead_events (lead_id, user_id, event_type, payload) VALUES (?, ?, 'FOLLOWUP', ?)`,
			body.LeadID, body.UserID, string(payload))
		writeJSON(writer, http.StatusOK, map[string]string{"status": "ok"})
	})

	hsrv.HandleFunc("/v1/leads/detail", func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			writeJSON(writer, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
			return
		}
		leadID := request.URL.Query().Get("leadId")
		userID := request.URL.Query().Get("userId")
		if userID == "" {
			userID = "1"
		}
		lid, err := strconv.ParseInt(leadID, 10, 64)
		if err != nil || lid == 0 {
			writeJSON(writer, http.StatusBadRequest, map[string]string{"error": "lead_id_required"})
			return
		}
		r := database.QueryRow(`
SELECT `+dbschema.LeadWideSelectColumns+`
FROM client_leads WHERE id = ?`, lid)
		leadMap, err := dbschema.ScanClientLeadWideRow(r)
		if err == sql.ErrNoRows {
			writeJSON(writer, http.StatusNotFound, map[string]string{"error": "not_found"})
			return
		}
		if err != nil {
			writeJSON(writer, http.StatusInternalServerError, map[string]string{"error": "query_failed"})
			return
		}
		uid, _ := strconv.ParseInt(userID, 10, 64)
		status, _ := leadMap["status"].(string)
		assigned, _ := leadMap["assignedUserId"].(int64)
		pickedBy, _ := leadMap["pickedUpBy"].(int64)
		if status != "NEW" && assigned != uid && pickedBy != uid {
			writeJSON(writer, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}
		evRows, err := database.Query(`
SELECT id, user_id, event_type, payload, occurred_at FROM client_lead_events WHERE lead_id = ? ORDER BY occurred_at ASC`, lid)
		if err != nil {
			writeJSON(writer, http.StatusInternalServerError, map[string]string{"error": "events_failed"})
			return
		}
		defer evRows.Close()
		events := []map[string]any{}
		for evRows.Next() {
			var eid int64
			var euid sql.NullInt64
			var etype, epayload, eat string
			_ = evRows.Scan(&eid, &euid, &etype, &epayload, &eat)
			events = append(events, map[string]any{
				"id": eid, "userId": euid.Int64, "type": etype, "payload": epayload, "occurredAt": eat,
			})
		}
		writeJSON(writer, http.StatusOK, map[string]any{
			"lead":   leadMap,
			"events": events,
		})
	})
}
