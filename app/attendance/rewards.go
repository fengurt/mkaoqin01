package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	khttp "github.com/go-kratos/kratos/v2/transport/http"
)

func genSyncUID() string {
	var buf [16]byte
	_, _ = rand.Read(buf[:])
	return hex.EncodeToString(buf[:])
}

func parseLocalYmd(occurredAt string) (string, error) {
	layouts := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05",
		"2006-01-02T15:04:05Z07:00",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, occurredAt); err == nil {
			return t.In(time.Local).Format("2006-01-02"), nil
		}
	}
	if t, err := time.ParseInLocation("2006-01-02T15:04:05", occurredAt, time.Local); err == nil {
		return t.Format("2006-01-02"), nil
	}
	if t, err := time.ParseInLocation("2006-01-02", occurredAt, time.Local); err == nil {
		return t.Format("2006-01-02"), nil
	}
	return "", fmt.Errorf("cannot parse occurredAt")
}

func nextCalendarDay(ymd string) (string, error) {
	t, err := time.ParseInLocation("2006-01-02", ymd, time.Local)
	if err != nil {
		return "", err
	}
	return t.AddDate(0, 0, 1).Format("2006-01-02"), nil
}

func memberTierFromTotals(totalFull int64) string {
	if totalFull >= 30 {
		return "GOLD"
	}
	if totalFull >= 10 {
		return "SILVER"
	}
	return "BRONZE"
}

// processRewardsAfterSubmit updates streak counters for CHECK_IN / CHECK_OUT and inserts newly earned badges.
func processRewardsAfterSubmit(db *sql.DB, userID int64, status string, occurredAt string) ([]map[string]any, error) {
	if status != "CHECK_IN" && status != "CHECK_OUT" {
		return nil, nil
	}
	day, err := parseLocalYmd(occurredAt)
	if err != nil {
		return nil, err
	}
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.Exec(`INSERT OR IGNORE INTO user_streaks (user_id) VALUES (?)`, userID); err != nil {
		return nil, err
	}

	var lastCIStr, lastFDStr string
	var curCI, bestCI, curFD, bestFD, totalCI, totalFD int64
	var memberTier string
	err = tx.QueryRow(`
SELECT IFNULL(last_checkin_date,''), IFNULL(last_full_date,''),
       current_checkin_streak, best_checkin_streak, current_full_streak, best_full_streak,
       total_checkin_days, total_full_days, IFNULL(member_tier,'BRONZE')
FROM user_streaks WHERE user_id = ?`, userID).Scan(&lastCIStr, &lastFDStr, &curCI, &bestCI, &curFD, &bestFD, &totalCI, &totalFD, &memberTier)
	if err != nil {
		return nil, err
	}

	newCI := curCI
	newFD := curFD
	newBestCI := bestCI
	newBestFD := bestFD
	newTotalCI := totalCI
	newTotalFD := totalFD
	newLastCI := lastCIStr
	newLastFD := lastFDStr

	if status == "CHECK_IN" {
		if lastCIStr == "" {
			newCI = 1
		} else if lastCIStr == day {
			newCI = curCI
		} else {
			nd, err := nextCalendarDay(lastCIStr)
			if err != nil {
				return nil, err
			}
			if nd == day {
				newCI = curCI + 1
			} else {
				newCI = 1
			}
		}
		newBestCI = max64(bestCI, newCI)
		if lastCIStr != day {
			newTotalCI++
		}
		newLastCI = day
	}

	if status == "CHECK_OUT" {
		var ciCount, coCount int64
		if err := tx.QueryRow(`
SELECT
  (SELECT COUNT(1) FROM attendance_records WHERE user_id = ? AND date(occurred_at,'localtime') = date(?) AND status = 'CHECK_IN'),
  (SELECT COUNT(1) FROM attendance_records WHERE user_id = ? AND date(occurred_at,'localtime') = date(?) AND status = 'CHECK_OUT')`,
			userID, day, userID, day).Scan(&ciCount, &coCount); err != nil {
			return nil, err
		}
		if ciCount >= 1 && coCount >= 1 && lastFDStr != day {
			if lastFDStr == "" {
				newFD = 1
			} else {
				nd, err := nextCalendarDay(lastFDStr)
				if err != nil {
					return nil, err
				}
				if nd == day {
					newFD = curFD + 1
				} else {
					newFD = 1
				}
			}
			newBestFD = max64(bestFD, newFD)
			newTotalFD++
			newLastFD = day
		}
	}

	newMember := memberTierFromTotals(newTotalFD)

	_, err = tx.Exec(`
UPDATE user_streaks SET
  current_checkin_streak = ?, best_checkin_streak = ?, current_full_streak = ?, best_full_streak = ?,
  total_checkin_days = ?, total_full_days = ?,
  last_checkin_date = NULLIF(?, ''), last_full_date = NULLIF(?, ''),
  member_tier = ?, updated_at = datetime('now')
WHERE user_id = ?`,
		newCI, newBestCI, newFD, newBestFD, newTotalCI, newTotalFD,
		nullStringArg(newLastCI), nullStringArg(newLastFD),
		newMember, userID,
	)
	if err != nil {
		return nil, err
	}

	awarded, err := evaluateAndInsertBadges(tx, userID, newCI, newFD, newTotalFD)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return awarded, nil
}

func nullStringArg(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func max64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func evaluateAndInsertBadges(tx *sql.Tx, userID int64, curCI, curFD, totalFD int64) ([]map[string]any, error) {
	rows, err := tx.Query(`
SELECT code, kind, tier, title_i18n, rule_type, rule_threshold
FROM badge_definitions WHERE active = 1 ORDER BY sort_order ASC, rule_threshold ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	awarded := make([]map[string]any, 0)
	for rows.Next() {
		var code, kind, tier, titleI18n, ruleType string
		var threshold int64
		if err := rows.Scan(&code, &kind, &tier, &titleI18n, &ruleType, &threshold); err != nil {
			return nil, err
		}
		ok := false
		switch ruleType {
		case "CONSECUTIVE_CHECKIN":
			ok = curCI >= threshold
		case "CONSECUTIVE_FULLDAY":
			ok = curFD >= threshold
		case "TOTAL_FULL_DAYS":
			ok = totalFD >= threshold
		default:
			// Unknown catalog rule_type — do not award (data integrity).
			continue
		}
		if !ok {
			continue
		}
		evidence, _ := json.Marshal(map[string]any{
			"currentCheckInStreak": curCI, "currentFullStreak": curFD, "totalFullDays": totalFD,
		})
		sync := genSyncUID()
		res, err := tx.Exec(`
INSERT OR IGNORE INTO user_badges (user_id, badge_code, period_key, evidence_payload, sync_uid)
VALUES (?, ?, 'ALL_TIME', ?, ?)`,
			userID, code, string(evidence), sync)
		if err != nil {
			return nil, err
		}
		n, _ := res.RowsAffected()
		if n != 1 {
			continue
		}
		var id int64
		if err := tx.QueryRow(`SELECT id FROM user_badges WHERE user_id = ? AND badge_code = ? AND period_key = 'ALL_TIME'`, userID, code).Scan(&id); err != nil {
			return nil, err
		}
		var titleParsed any
		_ = json.Unmarshal([]byte(titleI18n), &titleParsed)
		awarded = append(awarded, map[string]any{
			"id": id, "code": code, "kind": kind, "tier": tier, "titleI18n": titleParsed, "titleI18nRaw": titleI18n,
		})
	}
	return awarded, nil
}

func registerRewardRoutes(hsrv *khttp.Server, database *sql.DB) {
	hsrv.HandleFunc("/v1/rewards/me", func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			writeJSON(writer, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
			return
		}
		userID := request.URL.Query().Get("userId")
		if userID == "" {
			userID = "1"
		}
		var curCI, bestCI, curFD, bestFD, totalCI, totalFD int64
		var lastCI, lastFD, memberTier sql.NullString
		err := database.QueryRow(`
SELECT IFNULL(current_checkin_streak,0), IFNULL(best_checkin_streak,0),
       IFNULL(current_full_streak,0), IFNULL(best_full_streak,0),
       IFNULL(total_checkin_days,0), IFNULL(total_full_days,0),
       last_checkin_date, last_full_date, IFNULL(member_tier,'BRONZE')
FROM user_streaks WHERE user_id = ?`, userID).Scan(&curCI, &bestCI, &curFD, &bestFD, &totalCI, &totalFD, &lastCI, &lastFD, &memberTier)
		if err == sql.ErrNoRows {
			writeJSON(writer, http.StatusOK, map[string]any{
				"streak": map[string]any{
					"checkInCurrent": 0, "checkInBest": 0, "fullDayCurrent": 0, "fullDayBest": 0,
					"totalCheckInDays": 0, "totalFullDays": 0, "memberTier": "BRONZE",
				},
				"badges": []any{},
			})
			return
		}
		if err != nil {
			writeJSON(writer, http.StatusInternalServerError, map[string]string{"error": "query_failed"})
			return
		}
		badgeRows, err := database.Query(`
SELECT ub.id, ub.badge_code, ub.awarded_at, ub.acknowledged_at, bd.kind, bd.tier, bd.title_i18n
FROM user_badges ub
JOIN badge_definitions bd ON bd.code = ub.badge_code
WHERE ub.user_id = ?
ORDER BY ub.awarded_at DESC`, userID)
		if err != nil {
			writeJSON(writer, http.StatusInternalServerError, map[string]string{"error": "badges_query_failed"})
			return
		}
		defer badgeRows.Close()
		badges := []map[string]any{}
		for badgeRows.Next() {
			var id int64
			var code, awardedAt string
			var ack sql.NullString
			var kind, tier, titleI18n string
			_ = badgeRows.Scan(&id, &code, &awardedAt, &ack, &kind, &tier, &titleI18n)
			var titleParsed any
			_ = json.Unmarshal([]byte(titleI18n), &titleParsed)
			badges = append(badges, map[string]any{
				"id": id, "code": code, "kind": kind, "tier": tier,
				"titleI18n": titleParsed, "awardedAt": awardedAt, "acknowledgedAt": nullStringPtr(ack),
			})
		}
		writeJSON(writer, http.StatusOK, map[string]any{
			"streak": map[string]any{
				"checkInCurrent":   curCI,
				"checkInBest":      bestCI,
				"fullDayCurrent":   curFD,
				"fullDayBest":      bestFD,
				"totalCheckInDays": totalCI,
				"totalFullDays":    totalFD,
				"lastCheckInDate":  nullStringPtr(lastCI),
				"lastFullDate":     nullStringPtr(lastFD),
				"memberTier":       memberTier.String,
			},
			"badges": badges,
		})
	})

	hsrv.HandleFunc("/v1/rewards/ack", func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			writeJSON(writer, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
			return
		}
		var body struct {
			UserID   int64   `json:"userId"`
			BadgeIDs []int64 `json:"badgeIds"`
		}
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			writeJSON(writer, http.StatusBadRequest, map[string]string{"error": "invalid_payload"})
			return
		}
		if body.UserID == 0 {
			body.UserID = 1
		}
		if len(body.BadgeIDs) == 0 {
			writeJSON(writer, http.StatusBadRequest, map[string]string{"error": "badge_ids_required"})
			return
		}
		for _, bid := range body.BadgeIDs {
			if _, err := database.Exec(`
UPDATE user_badges SET acknowledged_at = datetime('now')
WHERE id = ? AND user_id = ? AND acknowledged_at IS NULL`, bid, body.UserID); err != nil {
				writeJSON(writer, http.StatusInternalServerError, map[string]string{"error": "update_failed"})
				return
			}
		}
		writeJSON(writer, http.StatusOK, map[string]string{"status": "ok"})
	})
}

func nullStringPtr(ns sql.NullString) any {
	if !ns.Valid {
		return nil
	}
	if ns.String == "" {
		return nil
	}
	return ns.String
}
