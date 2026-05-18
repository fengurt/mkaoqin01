package dbschema

import (
	"database/sql"
	"fmt"
	"strings"
)

// VerifySQLitePostApply runs lightweight integrity checks after DDL + migrations.
// Requires the connection to have foreign keys enabled (e.g. DSN `_foreign_keys=on`).
func VerifySQLitePostApply(database *sql.DB) error {
	if err := verifyForeignKeys(database); err != nil {
		return err
	}
	return verifyQuickCheck(database)
}

func verifyForeignKeys(database *sql.DB) error {
	rows, err := database.Query(`PRAGMA foreign_key_check`)
	if err != nil {
		return fmt.Errorf("pragma foreign_key_check: %w", err)
	}
	defer rows.Close()
	var violations []string
	for rows.Next() {
		var tableName, rowID, parent, fkIndex sql.NullString
		if scanErr := rows.Scan(&tableName, &rowID, &parent, &fkIndex); scanErr != nil {
			return fmt.Errorf("foreign_key_check scan: %w", scanErr)
		}
		violations = append(violations, fmt.Sprintf(
			"child_table=%s rowid=%s parent=%s fk_index=%s",
			nullStringForLog(tableName),
			nullStringForLog(rowID),
			nullStringForLog(parent),
			nullStringForLog(fkIndex),
		))
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if len(violations) > 0 {
		return fmt.Errorf("sqlite foreign key violations (%d): %s", len(violations), strings.Join(violations, " | "))
	}
	return nil
}

func verifyQuickCheck(database *sql.DB) error {
	rows, err := database.Query(`PRAGMA quick_check`)
	if err != nil {
		return fmt.Errorf("pragma quick_check: %w", err)
	}
	defer rows.Close()
	var messages []string
	for rows.Next() {
		var message string
		if scanErr := rows.Scan(&message); scanErr != nil {
			return fmt.Errorf("quick_check scan: %w", scanErr)
		}
		messages = append(messages, strings.TrimSpace(message))
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, m := range messages {
		if m != "" && !strings.EqualFold(m, "ok") {
			return fmt.Errorf("sqlite quick_check: %s", strings.Join(messages, " | "))
		}
	}
	return nil
}

func nullStringForLog(ns sql.NullString) string {
	if !ns.Valid {
		return "?"
	}
	return ns.String
}
