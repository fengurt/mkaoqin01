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
	return nil
}
