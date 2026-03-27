package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

const defaultDBPath = "data/public_decision_tree.db"

var appDB *sql.DB

func databasePath() string {
	path := strings.TrimSpace(os.Getenv("DB_PATH")) //Optional OS path ENV opption

	if path != "" {
		return path
	}

	return defaultDBPath
}

func initializeDatabase() (*sql.DB, error) {
	return openDatabase(databasePath())
}

func openDatabase(dbPath string) (*sql.DB, error) {
	if dbPath != ":memory:" {
		dir := filepath.Dir(dbPath)
		if dir != "." {
			//Create full directory structure if it doesn't exist (with unix permission code 755)
			err := os.MkdirAll(dir, 0o755)
			if err != nil {
				return nil, fmt.Errorf("create database directory: %w", err)
			}
		}
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	db.SetMaxOpenConns(1)

	//Enable foreign key enforcement
	_, err = db.Exec("PRAGMA foreign_keys = ON")
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}

	//Initialize tables
	err = ensureSchema(db)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("initialize schema: %w", err)
	}

	return db, nil
}

func ensureSchema(db *sql.DB) error {
	//Create a transaction
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	//Rollback will undo work that isn't committed (cleans transaction on erorr or crash)
	defer tx.Rollback()

	//This should be exactly to spec of DBML given in plan document
	statements := []string{
		`CREATE TABLE IF NOT EXISTS nodes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			kind TEXT NOT NULL CHECK (kind IN ('yesno', 'multichoice', 'roulette', 'conditional')),
			prompt TEXT NOT NULL,
			json TEXT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS ports (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			from_node_id INTEGER NOT NULL,
			to_node_id INTEGER,
			port_key TEXT NOT NULL,
			taken_count INTEGER NOT NULL DEFAULT 0,
			FOREIGN KEY (from_node_id) REFERENCES nodes(id),
			FOREIGN KEY (to_node_id) REFERENCES nodes(id)
		)`,
		`CREATE TABLE IF NOT EXISTS sessions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			path_fingerprint TEXT NOT NULL DEFAULT '',
			path_length INTEGER NOT NULL DEFAULT 0,
			current_node_id INTEGER,
			random_seed INTEGER NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			session_text TEXT NOT NULL DEFAULT '',
			FOREIGN KEY (current_node_id) REFERENCES nodes(id)
		)`,
		`CREATE TABLE IF NOT EXISTS session_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			session_id INTEGER NOT NULL,
			session_index INTEGER NOT NULL,
			node_id INTEGER NOT NULL,
			port_id INTEGER NOT NULL,
			text_delta TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (session_id) REFERENCES sessions(id),
			FOREIGN KEY (node_id) REFERENCES nodes(id),
			FOREIGN KEY (port_id) REFERENCES ports(id)
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_ports_from_node_port_key
			ON ports (from_node_id, port_key)`,
		`CREATE INDEX IF NOT EXISTS idx_ports_from_node_id
			ON ports (from_node_id)`,
		`CREATE INDEX IF NOT EXISTS idx_ports_to_node_id
			ON ports (to_node_id)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_session_history_session_index
			ON session_history (session_id, session_index)`,
		`CREATE INDEX IF NOT EXISTS idx_session_history_session_id
			ON session_history (session_id)`,
		`CREATE INDEX IF NOT EXISTS idx_session_history_port_id
			ON session_history (port_id)`,
	}

	for _, stmt := range statements {
		_, err = tx.Exec(stmt)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}
