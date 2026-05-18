package dbschema

import (
	"database/sql"
	"fmt"
	"strings"
)

func execAlterIgnoreDuplicate(database *sql.DB, ddl string) error {
	_, err := database.Exec(ddl)
	if err == nil {
		return nil
	}
	message := strings.ToLower(err.Error())
	if strings.Contains(message, "duplicate column") {
		return nil
	}
	return err
}

// MigrateSQLiteLegacy adds columns and tables introduced after older deployments.
// Safe to run after every bootstrap; ignores duplicate-column errors.
func MigrateSQLiteLegacy(database *sql.DB) error {
	// SQLite rejects ALTER ADD … DEFAULT (datetime('now')) ("non-constant default").
	// Use nullable columns + UPDATE backfill so older intervoice.db files still migrate.
	alters := []string{
		"ALTER TABLE users ADD COLUMN updated_at TEXT",
		"ALTER TABLE users ADD COLUMN sync_uid TEXT",
		"ALTER TABLE attendance_records ADD COLUMN updated_at TEXT",
		"ALTER TABLE attendance_records ADD COLUMN sync_uid TEXT",
	}
	for _, ddl := range alters {
		if err := execAlterIgnoreDuplicate(database, ddl); err != nil {
			return fmt.Errorf("migrate sqlite: %w", err)
		}
	}
	if _, err := database.Exec(`UPDATE users SET updated_at = datetime('now') WHERE updated_at IS NULL OR trim(COALESCE(updated_at,'')) = ''`); err != nil {
		return fmt.Errorf("migrate sqlite users updated_at backfill: %w", err)
	}
	if _, err := database.Exec(`UPDATE attendance_records SET updated_at = datetime('now') WHERE updated_at IS NULL OR trim(COALESCE(updated_at,'')) = ''`); err != nil {
		return fmt.Errorf("migrate sqlite attendance updated_at backfill: %w", err)
	}
	if _, err := database.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS ux_users_sync_uid ON users(sync_uid) WHERE sync_uid IS NOT NULL AND length(trim(sync_uid)) > 0`); err != nil {
		return fmt.Errorf("migrate sqlite users sync_uid index: %w", err)
	}
	if _, err := database.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS ux_attendance_sync_uid ON attendance_records(sync_uid) WHERE sync_uid IS NOT NULL AND length(trim(sync_uid)) > 0`); err != nil {
		return fmt.Errorf("migrate sqlite attendance sync_uid index: %w", err)
	}
	_, err := database.Exec(`
CREATE TABLE IF NOT EXISTS seed_markers (
  marker_key TEXT PRIMARY KEY,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
)`)
	if err != nil {
		return fmt.Errorf("migrate sqlite seed_markers: %w", err)
	}
	_, err = database.Exec(`
CREATE TABLE IF NOT EXISTS admin_data_audit (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  actor_user_id INTEGER NOT NULL,
  action TEXT NOT NULL,
  entity TEXT NOT NULL,
  sync_uid TEXT,
  detail TEXT
)`)
	if err != nil {
		return fmt.Errorf("migrate sqlite admin_data_audit: %w", err)
	}
	// Older deployments may lack user_daily_schedule if they never ran a build that embeds it;
	// safe to repeat (CREATE IF NOT EXISTS + INSERT OR IGNORE OFFICE).
	if _, err := database.Exec(userDailyScheduleSQL); err != nil {
		return fmt.Errorf("migrate sqlite user_daily_schedule: %w", err)
	}
	if err := migrateClientLeadColumns(database); err != nil {
		return err
	}
	if err := migrateFortuneTable(database); err != nil {
		return err
	}
	if err := migrateRewardsLeadsReferentialSanity(database); err != nil {
		return err
	}
	return nil
}

func migrateFortuneTable(database *sql.DB) error {
	var tableCount int
	if err := database.QueryRow(`SELECT COUNT(1) FROM sqlite_master WHERE type='table' AND name='user_daily_fortune'`).Scan(&tableCount); err != nil {
		return fmt.Errorf("migrate sqlite fortune table probe: %w", err)
	}
	if tableCount > 0 {
		return nil
	}
	if _, err := database.Exec(fortuneSQL); err != nil {
		return fmt.Errorf("migrate sqlite fortune ddl: %w", err)
	}
	return nil
}

// migrateClientLeadColumns adds Macau-oriented lead segmentation fields for older DBs
// (CREATE IF NOT EXISTS skips when client_leads already exists) and idempotent demo rows.
func migrateClientLeadColumns(database *sql.DB) error {
	var tableCount int
	if err := database.QueryRow(`SELECT COUNT(1) FROM sqlite_master WHERE type='table' AND name='client_leads'`).Scan(&tableCount); err != nil {
		return fmt.Errorf("migrate sqlite client_leads table probe: %w", err)
	}
	if tableCount == 0 {
		return nil
	}
	alters := []string{
		`ALTER TABLE client_leads ADD COLUMN lead_segment TEXT NOT NULL DEFAULT 'NEW_PURE'`,
		`ALTER TABLE client_leads ADD COLUMN approx_origin_region TEXT NOT NULL DEFAULT 'UNKNOWN'`,
		`ALTER TABLE client_leads ADD COLUMN preferred_venue TEXT NOT NULL DEFAULT 'UNSPECIFIED'`,
		`ALTER TABLE client_leads ADD COLUMN language_pref TEXT NOT NULL DEFAULT 'zh-CN'`,
		`ALTER TABLE client_leads ADD COLUMN estimated_party_size INTEGER`,
		`ALTER TABLE client_leads ADD COLUMN event_occasion TEXT NOT NULL DEFAULT 'OTHER'`,
		`ALTER TABLE client_leads ADD COLUMN channel_touchpoint TEXT NOT NULL DEFAULT 'UNKNOWN'`,
		`ALTER TABLE client_leads ADD COLUMN ref_last_visit_at TEXT`,
		`ALTER TABLE client_leads ADD COLUMN ref_last_property TEXT`,
		`ALTER TABLE client_leads ADD COLUMN ref_ltv_tier TEXT`,
		`ALTER TABLE client_leads ADD COLUMN ref_host_name TEXT`,
		`ALTER TABLE client_leads ADD COLUMN ref_member_id_masked TEXT`,
		`ALTER TABLE client_leads ADD COLUMN ref_notes TEXT`,
	}
	for _, ddl := range alters {
		if err := execAlterIgnoreDuplicate(database, ddl); err != nil {
			return fmt.Errorf("migrate sqlite client_leads columns: %w", err)
		}
	}
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_client_leads_segment ON client_leads(lead_segment)`,
		`CREATE INDEX IF NOT EXISTS idx_client_leads_origin ON client_leads(approx_origin_region)`,
		`CREATE INDEX IF NOT EXISTS idx_client_leads_venue ON client_leads(preferred_venue)`,
	}
	for _, ddl := range indexes {
		if _, err := database.Exec(ddl); err != nil {
			return fmt.Errorf("migrate sqlite client_leads indexes: %w", err)
		}
	}
	seedStatements := []string{
		`INSERT OR IGNORE INTO client_leads (
  source, source_ref, lead_segment, approx_origin_region, preferred_venue, language_pref,
  estimated_party_size, event_occasion, channel_touchpoint,
  client_name, client_phone, client_company, intent, notes, priority, status, assigned_user_id, metadata
) VALUES (
  'DEMO', 'seed-macau-new-guest-cotai', 'NEW_PURE', 'MAINLAND', 'MGM_COTAI', 'zh-CN',
  12, 'WEDDING_BANQUET', 'WECHAT_OFFICIAL',
  '李女士 · 深圳', '+86-138****2288', '科技公司市场总监', 'VISIT',
  '希望了解美狮宴会厅与餐饮配套，偏好普通话接待', 'HIGH', 'NEW', NULL,
  '{"zh-CN":"纯新客演示：来源地+偏好场地+渠道","en":"New guest demo"}'
)`,
		`INSERT OR IGNORE INTO client_leads (
  source, source_ref, lead_segment, approx_origin_region, preferred_venue, language_pref,
  estimated_party_size, event_occasion, channel_touchpoint,
  client_name, client_phone, client_company, intent, notes, priority, status, assigned_user_id,
  ref_last_visit_at, ref_last_property, ref_ltv_tier, ref_host_name, ref_member_id_masked, ref_notes, metadata
) VALUES (
  'DEMO', 'seed-macau-reactivate-peninsula', 'OLD_REACTIVATION', 'HK', 'MGM_PENINSULA', 'zh-TW',
  4, 'VIP_TABLE', 'HOST_REFERRAL',
  '陈先生（熟客）', '+852-9xxx-xxxx', '私募基金合伙人', 'VISIT',
  '两年未到访半岛场，希望安排贵宾厅与餐饮体验', 'URGENT', 'NEW', NULL,
  '2023-11-18', 'MGM_PENINSULA', 'GOLD', 'Amy Ho', 'MG****2188',
  '历史偏好：粤菜、忌海鲜过敏；曾参与私人晚宴',
  '{"zh-CN":"老客激活演示：参考到访/层级/主机","en":"Reactivation demo with reference fields"}'
)`,
	}
	for _, sqlText := range seedStatements {
		if _, err := database.Exec(sqlText); err != nil {
			return fmt.Errorf("migrate sqlite client_leads demo seed: %w", err)
		}
	}
	return nil
}

// migrateRewardsLeadsReferentialSanity removes rows that violate FK targets when
// historical data predates constraints or was loaded with FKs temporarily off.
func migrateRewardsLeadsReferentialSanity(database *sql.DB) error {
	steps := []string{
		`DELETE FROM user_badges WHERE user_id NOT IN (SELECT id FROM users)`,
		`DELETE FROM user_streaks WHERE user_id NOT IN (SELECT id FROM users)`,
		`DELETE FROM client_lead_events WHERE lead_id NOT IN (SELECT id FROM client_leads)`,
		`DELETE FROM client_lead_events WHERE user_id IS NOT NULL AND user_id NOT IN (SELECT id FROM users)`,
		`UPDATE client_leads SET assigned_user_id = NULL
		 WHERE assigned_user_id IS NOT NULL AND assigned_user_id NOT IN (SELECT id FROM users)`,
		`UPDATE client_leads SET picked_up_by = NULL, picked_up_at = NULL,
		      status = CASE WHEN status IN ('PICKED_UP','IN_FOLLOW_UP') THEN 'NEW' ELSE status END
		 WHERE picked_up_by IS NOT NULL AND picked_up_by NOT IN (SELECT id FROM users)`,
		`UPDATE user_streaks SET
		      current_checkin_streak = MAX(0, current_checkin_streak),
		      best_checkin_streak = MAX(0, best_checkin_streak),
		      current_full_streak = MAX(0, current_full_streak),
		      best_full_streak = MAX(0, best_full_streak),
		      total_checkin_days = MAX(0, total_checkin_days),
		      total_full_days = MAX(0, total_full_days)
		 WHERE current_checkin_streak < 0 OR best_checkin_streak < 0 OR current_full_streak < 0
		    OR best_full_streak < 0 OR total_checkin_days < 0 OR total_full_days < 0`,
	}
	for _, ddl := range steps {
		if _, err := database.Exec(ddl); err != nil {
			return fmt.Errorf("migrate rewards/leads referential sanity: %w", err)
		}
	}
	return nil
}
