package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	khttp "github.com/go-kratos/kratos/v2/transport/http"
)

var defaultFortunePool = []string{
	"/fortune/___01-_____Medium-4ade1249-5ee2-4aa0-9ba4-3af93ea68817.png",
	"/fortune/___02-_____Medium-42139af7-913e-4f1b-a43b-6bf513e2cab3.png",
	"/fortune/___03-_____Medium-5a78044d-409a-44f7-914b-edab5b4ee6b4.png",
	"/fortune/___04-_____Medium-340c1d61-0cd5-49a2-a239-94130f251357.png",
	"/fortune/___05-______Medium-2918e36a-1bf9-4ba4-bcbe-609d7e127020.png",
	"/fortune/___06-_____Medium-a7e33781-73b4-493c-91b1-97632ebcf47c.png",
	"/fortune/___07-_____Medium-275f5415-8209-43d1-bce4-f172667bc1c8.png",
	"/fortune/___08-_____Medium-c737a7d6-13ef-4c13-b864-4bfc1e9a786d.png",
	"/fortune/___09-_____Medium-860fbd8b-73b5-4fa6-acbf-5e06eeea4b0f.png",
	"/fortune/___10-_____Medium-0f6b40c7-c5c0-4d26-b70e-96da90763da0.png",
}

var defaultFortuneCaptions = []string{
	"稳住节奏，今天适合把关键事往前推一小步。",
	"沟通比猜测更省力，开口就有转机。",
	"小事认真收尾，会换来一整天的轻松。",
	"留一点空白时间，反而更容易抓住重点。",
	"今日宜专注：一次只做一件要事。",
	"把目标写清楚，执行会顺很多。",
	"适当起身走动，思路会自己找上门。",
	"对同事多点耐心，效率会反弹回来。",
}

func fortuneUploadRoot() string {
	if root := strings.TrimSpace(os.Getenv("UPLOAD_ROOT")); root != "" {
		return root
	}
	return filepath.Clean(filepath.Join("..", "..", "data", "uploads"))
}

func registerFortuneRoutes(hsrv *khttp.Server, database *sql.DB) {
	uploadRoot := fortuneUploadRoot()
	_ = os.MkdirAll(filepath.Join(uploadRoot, "fortune"), 0o755)

	hsrv.HandleFunc("/uploads/fortune/", func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet && request.Method != http.MethodHead {
			writeJSON(writer, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
			return
		}
		rel := strings.TrimPrefix(request.URL.Path, "/uploads/fortune/")
		rel = filepath.Clean(rel)
		if rel == "." || strings.HasPrefix(rel, "..") || strings.Contains(rel, string(os.PathSeparator)) {
			http.NotFound(writer, request)
			return
		}
		full := filepath.Join(uploadRoot, "fortune", rel)
		http.ServeFile(writer, request, full)
	})

	hsrv.HandleFunc("/v1/fortune/day", func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			writeJSON(writer, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
			return
		}
		userID, err := parseFortuneUserID(request.URL.Query().Get("userId"))
		if err != nil {
			writeJSON(writer, http.StatusBadRequest, map[string]string{"error": "invalid userId"})
			return
		}
		dateStr := strings.TrimSpace(request.URL.Query().Get("date"))
		if dateStr == "" {
			dateStr = time.Now().Format("2006-01-02")
		}
		if _, err := parseScheduleDate(dateStr); err != nil {
			writeJSON(writer, http.StatusBadRequest, map[string]string{"error": "invalid date"})
			return
		}
		payload, err := resolveDailyFortune(database, userID, dateStr)
		if err != nil {
			writeJSON(writer, http.StatusInternalServerError, map[string]string{"error": "fortune_resolve_failed"})
			return
		}
		writeJSON(writer, http.StatusOK, payload)
	})

	hsrv.HandleFunc("/v1/fortune/month", func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			writeJSON(writer, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
			return
		}
		userID, err := parseFortuneUserID(request.URL.Query().Get("userId"))
		if err != nil {
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
		if _, err := parseScheduleDate(from); err != nil || func() error { _, e := parseScheduleDate(to); return e }() != nil {
			writeJSON(writer, http.StatusBadRequest, map[string]string{"error": "invalid date range"})
			return
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
			if err := rows.Scan(&dateStr, &imageURL, &caption, &updatedAt); err != nil {
				writeJSON(writer, http.StatusInternalServerError, map[string]string{"error": "scan_failed"})
				return
			}
			items = append(items, map[string]any{
				"date": dateStr, "imageUrl": imageURL, "caption": caption,
				"source": "assigned", "updatedAt": updatedAt,
			})
		}
		writeJSON(writer, http.StatusOK, map[string]any{"userId": userID, "from": from, "to": to, "items": items})
	})
}

func parseFortuneUserID(raw string) (int64, error) {
	if raw == "" {
		return 1, nil
	}
	return strconv.ParseInt(raw, 10, 64)
}

func resolveDailyFortune(database *sql.DB, userID int64, dateStr string) (map[string]any, error) {
	var imageURL, caption string
	err := database.QueryRow(`
SELECT image_url, IFNULL(caption,'') FROM user_daily_fortune WHERE user_id = ? AND fortune_date = ?`,
		userID, dateStr).Scan(&imageURL, &caption)
	if err == nil {
		return map[string]any{
			"userId": userID, "date": dateStr, "imageUrl": imageURL, "caption": caption, "source": "assigned",
		}, nil
	}
	if err != sql.ErrNoRows {
		return nil, err
	}
	img, cap := poolFortuneForUserDate(userID, dateStr)
	return map[string]any{
		"userId": userID, "date": dateStr, "imageUrl": img, "caption": cap, "source": "pool",
	}, nil
}

func poolFortuneForUserDate(userID int64, dateStr string) (string, string) {
	seed := sha256.Sum256([]byte(strconv.FormatInt(userID, 10) + "|" + dateStr))
	hexSeed := hex.EncodeToString(seed[:])
	n := int(hexSeed[0])<<8 | int(hexSeed[1])
	if len(defaultFortunePool) > 0 {
		img := defaultFortunePool[n%len(defaultFortunePool)]
		cap := defaultFortuneCaptions[n%len(defaultFortuneCaptions)]
		return img, cap
	}
	return "", ""
}
