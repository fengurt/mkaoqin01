package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	khttp "github.com/go-kratos/kratos/v2/transport/http"
)

var allowedCatalogCategories = map[string]bool{
	"hotel_intro":         true,
	"dining_restaurant":   true,
	"schedule_chip":       true,
}

type locationCatalogUpsertBody struct {
	Slug        string `json:"slug"`
	Category    string `json:"category"`
	Region      string `json:"region"`
	Title       string `json:"title"`
	Subtitle    string `json:"subtitle"`
	Detail      string `json:"detail"`
	SortOrder   int    `json:"sortOrder"`
}

type scheduleSectionUpsertBody struct {
	ID            int64  `json:"id"`
	SortOrder     int    `json:"sortOrder"`
	SectionLabel  string `json:"sectionLabel"`
	ItemCategory  string `json:"itemCategory"`
	ItemRegion    string `json:"itemRegion"`
}

type slugDeleteBody struct {
	Slug string `json:"slug"`
}

type scheduleSectionDeleteBody struct {
	ID int64 `json:"id"`
}

func registerCatalogAdminRoutes(hsrv *khttp.Server, database *sql.DB) {
	if hsrv == nil || database == nil {
		return
	}

	hsrv.HandleFunc("/v1/admin/data/location-catalog", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			methodNotAllowedJSON(w)
			return
		}
		if _, ok := requireAdminJWT(w, r); !ok {
			return
		}
		category := strings.TrimSpace(r.URL.Query().Get("category"))
		var rows *sql.Rows
		var err error
		if category != "" {
			rows, err = database.Query(`
SELECT id, slug, category, IFNULL(region,''), title, IFNULL(subtitle,''), detail, sort_order,
       created_at, IFNULL(updated_at,'')
FROM location_catalog WHERE category = ? ORDER BY region, sort_order, title`, category)
		} else {
			rows, err = database.Query(`
SELECT id, slug, category, IFNULL(region,''), title, IFNULL(subtitle,''), detail, sort_order,
       created_at, IFNULL(updated_at,'')
FROM location_catalog ORDER BY category, region, sort_order, title`)
		}
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "query failed"})
			return
		}
		defer rows.Close()
		list := make([]map[string]any, 0)
		for rows.Next() {
			var id int64
			var slug, cat, region, title, subtitle, detail, createdAt, updatedAt string
			var sortOrder int
			if err := rows.Scan(&id, &slug, &cat, &region, &title, &subtitle, &detail, &sortOrder, &createdAt, &updatedAt); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "scan failed"})
				return
			}
			list = append(list, map[string]any{
				"id": id, "slug": slug, "category": cat, "region": region,
				"title": title, "subtitle": subtitle, "detail": detail, "sortOrder": sortOrder,
				"createdAt": createdAt, "updatedAt": updatedAt,
			})
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": list})
	})

	hsrv.HandleFunc("/v1/admin/data/location-catalog/upsert", func(w http.ResponseWriter, r *http.Request) {
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
		var payload locationCatalogUpsertBody
		if err := json.Unmarshal(body, &payload); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "json parse failed"})
			return
		}
		payload.Slug = strings.TrimSpace(payload.Slug)
		payload.Category = strings.TrimSpace(payload.Category)
		payload.Region = strings.TrimSpace(payload.Region)
		payload.Title = strings.TrimSpace(payload.Title)
		if payload.Slug == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "slug required"})
			return
		}
		if payload.Title == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "title required"})
			return
		}
		if !allowedCatalogCategories[payload.Category] {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid category"})
			return
		}
		_, err = database.Exec(`
INSERT INTO location_catalog (slug, category, region, title, subtitle, detail, sort_order, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, datetime('now'))
ON CONFLICT(slug) DO UPDATE SET
  category = excluded.category,
  region = excluded.region,
  title = excluded.title,
  subtitle = excluded.subtitle,
  detail = excluded.detail,
  sort_order = excluded.sort_order,
  updated_at = datetime('now')`,
			payload.Slug, payload.Category, payload.Region, payload.Title, payload.Subtitle, payload.Detail, payload.SortOrder)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "upsert failed"})
			return
		}
		detailAudit := fmt.Sprintf(`{"slug":%q,"category":%q}`, payload.Slug, payload.Category)
		_, _ = database.Exec(`INSERT INTO admin_data_audit (actor_user_id, action, entity, sync_uid, detail) VALUES (?,?,?,?,?)`,
			actorID, "location_catalog_upsert", "location_catalog", nil, detailAudit)
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "slug": payload.Slug})
	})

	hsrv.HandleFunc("/v1/admin/data/location-catalog/delete", func(w http.ResponseWriter, r *http.Request) {
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
		var payload slugDeleteBody
		if err := json.Unmarshal(body, &payload); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "json parse failed"})
			return
		}
		slug := strings.TrimSpace(payload.Slug)
		if slug == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "slug required"})
			return
		}
		res, err := database.Exec(`DELETE FROM location_catalog WHERE slug = ?`, slug)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "delete failed"})
			return
		}
		n, _ := res.RowsAffected()
		_, _ = database.Exec(`INSERT INTO admin_data_audit (actor_user_id, action, entity, sync_uid, detail) VALUES (?,?,?,?,?)`,
			actorID, "location_catalog_delete", "location_catalog", nil, fmt.Sprintf(`{"slug":%q,"rows":%d}`, slug, n))
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "deleted": n})
	})

	hsrv.HandleFunc("/v1/admin/data/schedule-quick-sections", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			methodNotAllowedJSON(w)
			return
		}
		if _, ok := requireAdminJWT(w, r); !ok {
			return
		}
		rows, err := database.Query(`
SELECT id, sort_order, section_label, item_category, IFNULL(item_region,''),
       created_at, IFNULL(updated_at,'')
FROM schedule_quick_section
ORDER BY sort_order ASC, id ASC`)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "query failed"})
			return
		}
		defer rows.Close()
		list := make([]map[string]any, 0)
		for rows.Next() {
			var id int64
			var sortOrder int
			var label, cat, region, createdAt, updatedAt string
			if err := rows.Scan(&id, &sortOrder, &label, &cat, &region, &createdAt, &updatedAt); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "scan failed"})
				return
			}
			list = append(list, map[string]any{
				"id": id, "sortOrder": sortOrder, "sectionLabel": label,
				"itemCategory": cat, "itemRegion": region,
				"createdAt": createdAt, "updatedAt": updatedAt,
			})
		}
		writeJSON(w, http.StatusOK, map[string]any{"sections": list})
	})

	hsrv.HandleFunc("/v1/admin/data/schedule-quick-sections/upsert", func(w http.ResponseWriter, r *http.Request) {
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
		var payload scheduleSectionUpsertBody
		if err := json.Unmarshal(body, &payload); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "json parse failed"})
			return
		}
		payload.SectionLabel = strings.TrimSpace(payload.SectionLabel)
		payload.ItemCategory = strings.TrimSpace(payload.ItemCategory)
		payload.ItemRegion = strings.TrimSpace(payload.ItemRegion)
		if payload.SectionLabel == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "sectionLabel required"})
			return
		}
		if !allowedCatalogCategories[payload.ItemCategory] {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid itemCategory"})
			return
		}
		if payload.ID > 0 {
			_, err = database.Exec(`
UPDATE schedule_quick_section SET sort_order = ?, section_label = ?, item_category = ?, item_region = ?, updated_at = datetime('now')
WHERE id = ?`,
				payload.SortOrder, payload.SectionLabel, payload.ItemCategory, payload.ItemRegion, payload.ID)
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "update failed"})
				return
			}
			_, _ = database.Exec(`INSERT INTO admin_data_audit (actor_user_id, action, entity, sync_uid, detail) VALUES (?,?,?,?,?)`,
				actorID, "schedule_quick_section_upsert", "schedule_quick_section", nil,
				fmt.Sprintf(`{"id":%d}`, payload.ID))
			writeJSON(w, http.StatusOK, map[string]any{"ok": true, "id": payload.ID})
			return
		}
		res, err := database.Exec(`
INSERT INTO schedule_quick_section (sort_order, section_label, item_category, item_region, updated_at)
VALUES (?, ?, ?, ?, datetime('now'))`,
			payload.SortOrder, payload.SectionLabel, payload.ItemCategory, payload.ItemRegion)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "insert failed"})
			return
		}
		newID, _ := res.LastInsertId()
		_, _ = database.Exec(`INSERT INTO admin_data_audit (actor_user_id, action, entity, sync_uid, detail) VALUES (?,?,?,?,?)`,
			actorID, "schedule_quick_section_upsert", "schedule_quick_section", nil,
			fmt.Sprintf(`{"id":%d}`, newID))
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "id": newID})
	})

	hsrv.HandleFunc("/v1/admin/data/schedule-quick-sections/delete", func(w http.ResponseWriter, r *http.Request) {
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
		var payload scheduleSectionDeleteBody
		if err := json.Unmarshal(body, &payload); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "json parse failed"})
			return
		}
		if payload.ID < 1 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "id required"})
			return
		}
		res, err := database.Exec(`DELETE FROM schedule_quick_section WHERE id = ?`, payload.ID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "delete failed"})
			return
		}
		n, _ := res.RowsAffected()
		_, _ = database.Exec(`INSERT INTO admin_data_audit (actor_user_id, action, entity, sync_uid, detail) VALUES (?,?,?,?,?)`,
			actorID, "schedule_quick_section_delete", "schedule_quick_section", nil, fmt.Sprintf(`{"id":%d,"rows":%d}`, payload.ID, n))
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "deleted": n})
	})
}
