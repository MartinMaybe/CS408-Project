package main

import (
	"database/sql"
	"path/filepath"
	"testing"
)

func setupTestAppDB(t *testing.T) *sql.DB {
	t.Helper()

	oldDB := appDB

	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := openDatabase(dbPath)
	if err != nil {
		t.Fatalf("openDatabase returned error: %v", err)
	}

	appDB = db

	t.Cleanup(func() {
		appDB = oldDB
		db.Close()
	})

	return db
}
