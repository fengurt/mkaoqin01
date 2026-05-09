package dbschema

import (
	"database/sql"
	_ "embed"
)

//go:embed bootstrap.sql
var bootstrapSQL string

//go:embed location_catalog.sql
var locationCatalogSQL string

//go:embed schedule_quick_sections.sql
var scheduleQuickSectionsSQL string

//go:embed user_daily_schedule.sql
var userDailyScheduleSQL string

// ApplySQLite runs canonical DDL + seeds, then additive migrations for older files.
func ApplySQLite(database *sql.DB) error {
	if _, err := database.Exec(bootstrapSQL); err != nil {
		return err
	}
	if _, err := database.Exec(locationCatalogSQL); err != nil {
		return err
	}
	if _, err := database.Exec(scheduleQuickSectionsSQL); err != nil {
		return err
	}
	if _, err := database.Exec(userDailyScheduleSQL); err != nil {
		return err
	}
	return MigrateSQLiteLegacy(database)
}
