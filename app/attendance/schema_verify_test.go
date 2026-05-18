package main

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"intervoice/dbschema"
)

func TestApplySQLiteForeignKeysAndQuickCheck(t *testing.T) {
	t.Parallel()
	database, err := sql.Open("sqlite3", "file:mem_schema_integrity?mode=memory&cache=shared&_foreign_keys=on")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	if err := dbschema.ApplySQLite(database); err != nil {
		t.Fatalf("ApplySQLite: %v", err)
	}
}
