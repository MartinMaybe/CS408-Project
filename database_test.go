package main

import (
	"path/filepath"
	"testing"
)

/*
* Test database is opened, tables are created, and foreign keys are enabled
 */
func TestOpenDatabaseCreatesSchema(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "checkpoint2.db")

	db, err := openDatabase(dbPath)
	if err != nil {
		t.Fatalf("openDatabase returned error: %v", err)
	}
	t.Cleanup(func() {
		db.Close()
	})

	expectedTables := []string{"nodes", "ports", "sessions", "session_history"}
	for _, table := range expectedTables {
		var name string
		err := db.QueryRow(
			`SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?`,
			table,
		).Scan(&name)
		if err != nil {
			t.Fatalf("expected table %q to exist: %v", table, err)
		}
	}

	var foreignKeysEnabled int
	if err := db.QueryRow(`PRAGMA foreign_keys`).Scan(&foreignKeysEnabled); err != nil {
		t.Fatalf("check foreign key pragma: %v", err)
	}
	if foreignKeysEnabled != 1 {
		t.Fatalf("expected foreign keys to be enabled, got %d", foreignKeysEnabled)
	}
}
