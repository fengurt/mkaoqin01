package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	khttp "github.com/go-kratos/kratos/v2/transport/http"
	"github.com/google/uuid"
)

func registerFortuneAdminRoutes(hsrv *khttp.Server, database *sql.DB) {
	uploadRoot := fortuneUploadRoot()
	_ = os.MkdirAll(filepath.Join(uploadRoot, "fortune"), 0o755)

	hsrv.HandleFunc("/v1/admin/fortune/month", func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			writeJSON(writer, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
			return
		}
		userID, err := strconv.ParseInt(request.URL.Query().Get("userId"), 10, 64)
		if err != nil || userID < 1 {
			writeJSON(writer, http.StatusBadRequest, map[string]string{"error": "invalid userId"})
			return
		}
		from := strings.TrimSpace(request.URL.Query().Get("from"))
		to := strings.TrimSpace(request.URL.Query().Get("to"))
		if from == "" || to == "" {
			now := time.Now()
			from = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).Format("2006-01-02")
			to = time.Date(now.Year(), now.Month()+1, 0, 0, 0, 0, 0, now.Location()).Format("2006-01-02")
		}
		rows, err := database.Query(`
SELECT fortune_date, image_url, IFNULL(caption,''), updated_at
FROM user_daily_fortune
WHERE user_id = ? AND fortune_date >= ? AND fortune_date <= ?
ORDER BY fortune_date ASC`, userID, from, to)
		if err != nil {
			writeJSON(writer, http.StatusInternalServerError, map[string]string{"error": "query_failed"})
			return
		}
		defer rows.Close()
		items := []map[string]any{}
		for rows.Next() {
			var dateStr, imageURL, caption, updatedAt string
			_ = rows.Scan(&dateStr, &imageURL, &caption, &updatedAt)
			items = append(items, map[string]any{
				"date": dateStr, "imageUrl": imageURL, "caption": caption, "updatedAt": updatedAt,
			})
		}
		writeJSON(writer, http.StatusOK, map[string]any{"userId": userID, "from": from, "to": to, "items": items})
	})

	hsrv.HandleFunc("/v1/admin/fortune/upload", func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			writeJSON(writer, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
			return
		}
		if err := request.ParseMultipartForm(12 << 20); err != nil {
			writeJSON(writer, http.StatusBadRequest, map[string]string{"error": "invalid_multipart"})
			return
		}
		userID, _ := strconv.ParseInt(request.FormValue("userId"), 10, 64)
		dateStr := strings.TrimSpace(request.FormValue("date"))
		if userID < 1 || dateStr == "" {
			writeJSON(writer, http.StatusBadRequest, map[string]string{"error": "userId and date required"})
			return
		}
		file, header, err := request.FormFile("image")
		if err != nil {
			writeJSON(writer, http.StatusBadRequest, map[string]string{"error": "image_required"})
			return
		}
		defer file.Close()
		ext := strings.ToLower(filepath.Ext(header.Filename))
		if ext == "" {
			ext = ".jpg"
		}
		if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".webp" && ext != ".gif" {
			writeJSON(writer, http.StatusBadRequest, map[string]string{"error": "unsupported_image_type"})
			return
		}
		safeName := fmt.Sprintf("u%d_%s_%s%s", userID, strings.ReplaceAll(dateStr, "-", ""), uuid.NewString()[:8], ext)
		destPath := filepath.Join(uploadRoot, "fortune", safeName)
		out, err := os.Create(destPath)
		if err != nil {
			writeJSON(writer, http.StatusInternalServerError, map[string]string{"error": "save_failed"})
			return
		}
		_, err = io.Copy(out, file)
		_ = out.Close()
		if err != nil {
			writeJSON(writer, http.StatusInternalServerError, map[string]string{"error": "save_failed"})
			return
		}
		imageURL := "/uploads/fortune/" + safeName
		caption := strings.TrimSpace(request.FormValue("caption"))
		if err := upsertUserFortune(database, userID, dateStr, imageURL, caption); err != nil {
			writeJSON(writer, http.StatusInternalServerError, map[string]string{"error": "db_save_failed"})
			return
		}
		writeJSON(writer, http.StatusOK, map[string]any{
			"userId": userID, "date": dateStr, "imageUrl": imageURL, "caption": caption,
		})
	})

	hsrv.HandleFunc("/v1/admin/fortune/assign", func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			writeJSON(writer, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
			return
		}
		var body struct {
			UserID   int64  `json:"userId"`
			Date     string `json:"date"`
			ImageURL string `json:"imageUrl"`
			Caption  string `json:"caption"`
		}
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			writeJSON(writer, http.StatusBadRequest, map[string]string{"error": "invalid_payload"})
			return
		}
		if body.UserID < 1 || strings.TrimSpace(body.Date) == "" || strings.TrimSpace(body.ImageURL) == "" {
			writeJSON(writer, http.StatusBadRequest, map[string]string{"error": "invalid_fields"})
			return
		}
		if err := upsertUserFortune(database, body.UserID, strings.TrimSpace(body.Date), strings.TrimSpace(body.ImageURL), strings.TrimSpace(body.Caption)); err != nil {
			writeJSON(writer, http.StatusInternalServerError, map[string]string{"error": "db_save_failed"})
			return
		}
		writeJSON(writer, http.StatusOK, map[string]string{"status": "ok"})
	})

	hsrv.HandleFunc("/v1/admin/fortune/sync", func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			writeJSON(writer, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
			return
		}
		var body struct {
			SourceUserID int64   `json:"sourceUserId"`
			Date         string  `json:"date"`
			TargetUserIDs []int64 `json:"targetUserIds"`
		}
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			writeJSON(writer, http.StatusBadRequest, map[string]string{"error": "invalid_payload"})
			return
		}
		body.Date = strings.TrimSpace(body.Date)
		if body.SourceUserID < 1 || body.Date == "" || len(body.TargetUserIDs) == 0 {
			writeJSON(writer, http.StatusBadRequest, map[string]string{"error": "invalid_fields"})
			return
		}
		var imageURL, caption string
		err := database.QueryRow(`
SELECT image_url, IFNULL(caption,'') FROM user_daily_fortune WHERE user_id = ? AND fortune_date = ?`,
			body.SourceUserID, body.Date).Scan(&imageURL, &caption)
		if err == sql.ErrNoRows {
			writeJSON(writer, http.StatusNotFound, map[string]string{"error": "source_fortune_not_found"})
			return
		}
		if err != nil {
			writeJSON(writer, http.StatusInternalServerError, map[string]string{"error": "query_failed"})
			return
		}
		synced := 0
		for _, targetID := range body.TargetUserIDs {
			if targetID < 1 || targetID == body.SourceUserID {
				continue
			}
			if err := upsertUserFortune(database, targetID, body.Date, imageURL, caption); err != nil {
				continue
			}
			synced++
		}
		writeJSON(writer, http.StatusOK, map[string]any{"synced": synced, "imageUrl": imageURL, "caption": caption})
	})
}

func fortuneUploadRoot() string {
	if root := strings.TrimSpace(os.Getenv("UPLOAD_ROOT")); root != "" {
		return root
	}
	return filepath.Clean(filepath.Join("..", "..", "data", "uploads"))
}

func upsertUserFortune(database *sql.DB, userID int64, dateStr, imageURL, caption string) error {
	_, err := database.Exec(`
INSERT INTO user_daily_fortune (user_id, fortune_date, image_url, caption, updated_at)
VALUES (?, ?, ?, ?, datetime('now'))
ON CONFLICT(user_id, fortune_date) DO UPDATE SET
  image_url = excluded.image_url,
  caption = excluded.caption,
  updated_at = datetime('now')`,
		userID, dateStr, imageURL, caption)
	return err
}
