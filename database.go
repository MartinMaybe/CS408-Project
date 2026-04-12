package main

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

const defaultDBPath = "data/public_decision_tree.db"
const rootNodeID = 1

var appDB *sql.DB

var (
	ErrSessionNotFound            = errors.New("session not found")
	ErrNodeNotFound               = errors.New("node not found")
	ErrPortNotFound               = errors.New("port not found")
	ErrNullCurrentNode            = errors.New("session current_node_id is null")
	ErrPortDoesNotBelongToSession = errors.New("port does not belong to current session node")
)

type sessionRow struct {
	ID              int
	CurrentNodeID   int
	PathLength      int
	RandomSeed      int64
	PathFingerprint string
	SessionText     string
}

type nodeRow struct {
	ID     int
	Kind   string
	Prompt string
	JSON   string
}

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

	// Prepopulate nodes
	_, err = tx.Exec(`
	INSERT OR IGNORE INTO nodes (id, kind, prompt, json) VALUES
		(1, 'yesno', 'Is it an animal?', '{}'),
		(2, 'yesno', 'Does it have fur?', '{}'),
		(3, 'yesno', 'Is it a plant?', '{}'),
		(4, 'yesno', 'Is it a dog?', '{}'),
		(5, 'yesno', 'Is it a reptile?', '{}'),
		(6, 'yesno', 'Is it a tree?', '{}'),
		(7, 'yesno', 'Is it a flower?', '{}')
	`)

	if err != nil {
		return err
	}

	_, err = tx.Exec(`
	INSERT OR IGNORE INTO ports (from_node_id, to_node_id, port_key) VALUES
		(1, 2, 'yes'),
		(1, 3, 'no'),
		(2, 4, 'yes'),
		(2, 5, 'no'),
		(3, 6, 'yes'),
		(3, 7, 'no'),
		(4, NULL, 'yes'),
		(4, NULL, 'no'),
		(5, NULL, 'yes'),
		(5, NULL, 'no'),
		(6, NULL, 'yes'),
		(6, NULL, 'no'),
		(7, NULL, 'yes'),
		(7, NULL, 'no')
	`)

	if err != nil {
		return err
	}

	return tx.Commit()
}

func createSession() (int, error) {
	randomSeed := time.Now().UnixNano()

	result, err := appDB.Exec(`
		INSERT INTO sessions (current_node_id, random_seed)
		VALUES (?, ?)`,
		rootNodeID,
		randomSeed,
	)
	if err != nil {
		return 0, fmt.Errorf("create session: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("read created session ID: %w", err)
	}

	return int(id), nil
}

func createNode(kind string, prompt string, jsonText string) (int, error) {
	result, err := appDB.Exec(`
		INSERT INTO nodes (kind, prompt, json)
		VALUES (?, ?, ?)`,
		kind,
		prompt,
		jsonText,
	)
	if err != nil {
		return 0, fmt.Errorf("create node: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("read created node ID: %w", err)
	}

	return int(id), nil
}

func advanceSessionByPort(sessionID int, portID int) (string, error) {
	tx, err := appDB.Begin()
	if err != nil {
		return "", fmt.Errorf("begin session advance transaction: %w", err)
	}
	defer tx.Rollback()

	var currentNodeID sql.NullInt64
	err = tx.QueryRow(`
		SELECT current_node_id
		FROM sessions
		WHERE id = ?`,
		sessionID,
	).Scan(&currentNodeID)
	if err == sql.ErrNoRows {
		return "", ErrSessionNotFound
	}
	if err != nil {
		return "", fmt.Errorf("get session current node: %w", err)
	}
	if !currentNodeID.Valid {
		return "", ErrNullCurrentNode
	}

	var (
		fromNodeID int
		toNodeID   sql.NullInt64
	)
	err = tx.QueryRow(`
		SELECT from_node_id, to_node_id
		FROM ports
		WHERE id = ?`,
		portID,
	).Scan(&fromNodeID, &toNodeID)
	if err == sql.ErrNoRows {
		return "", ErrPortNotFound
	}
	if err != nil {
		return "", fmt.Errorf("get port by ID: %w", err)
	}

	if fromNodeID != int(currentNodeID.Int64) {
		return "", ErrPortDoesNotBelongToSession
	}

	_, err = tx.Exec(`
		UPDATE ports
		SET taken_count = taken_count + 1
		WHERE id = ?`,
		portID,
	)
	if err != nil {
		return "", fmt.Errorf("increment port taken_count: %w", err)
	}

	if !toNodeID.Valid {
		_, err = tx.Exec(`
			UPDATE sessions
			SET path_length = path_length + 1, updated_at = CURRENT_TIMESTAMP
			WHERE id = ?`,
			sessionID,
		)
		if err != nil {
			return "", fmt.Errorf("update terminal session progress: %w", err)
		}
		if err := tx.Commit(); err != nil {
			return "", fmt.Errorf("commit terminal session advance: %w", err)
		}
		return "complete", nil
	}

	_, err = tx.Exec(`
		UPDATE sessions
		SET current_node_id = ?, path_length = path_length + 1, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`,
		int(toNodeID.Int64),
		sessionID,
	)
	if err != nil {
		return "", fmt.Errorf("advance session current node by port: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("commit session advance: %w", err)
	}

	return "ok", nil
}

func updateSessionCurrentNode(sessionID int, nodeID int) error {
	exists, err := nodeExists(nodeID)
	if err != nil {
		return err
	}
	if !exists {
		return ErrNodeNotFound
	}

	result, err := appDB.Exec(`
		UPDATE sessions
		SET current_node_id = ?, path_length = path_length + 1, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`,
		nodeID,
		sessionID,
	)
	if err != nil {
		return fmt.Errorf("update session current node: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read updated session rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrSessionNotFound
	}

	return nil
}

func getSessionByID(sessionID int) (sessionRow, error) {
	var (
		row           sessionRow
		currentNodeID sql.NullInt64
	)

	err := appDB.QueryRow(`
		SELECT id, current_node_id, path_length, random_seed, path_fingerprint, session_text
		FROM sessions
		WHERE id = ?`,
		sessionID,
	).Scan(
		&row.ID,
		&currentNodeID,
		&row.PathLength,
		&row.RandomSeed,
		&row.PathFingerprint,
		&row.SessionText,
	)
	if err == sql.ErrNoRows {
		return sessionRow{}, ErrSessionNotFound
	}
	if err != nil {
		return sessionRow{}, fmt.Errorf("get session by ID: %w", err)
	}

	if !currentNodeID.Valid {
		return sessionRow{}, ErrNullCurrentNode
	}

	row.CurrentNodeID = int(currentNodeID.Int64)
	return row, nil
}

func getNodeByID(nodeID int) (nodeRow, error) {
	var row nodeRow

	err := appDB.QueryRow(`
		SELECT id, kind, prompt, json
		FROM nodes
		WHERE id = ?`,
		nodeID,
	).Scan(&row.ID, &row.Kind, &row.Prompt, &row.JSON)
	if err == sql.ErrNoRows {
		return nodeRow{}, ErrNodeNotFound
	}
	if err != nil {
		return nodeRow{}, fmt.Errorf("get node by ID: %w", err)
	}

	return row, nil
}

func getPortIDByNodeAndKey(nodeID int, portKey string) (int, error) {
	var portID int

	err := appDB.QueryRow(`
		SELECT id
		FROM ports
		WHERE from_node_id = ? AND port_key = ?`,
		nodeID,
		portKey,
	).Scan(&portID)
	if err == sql.ErrNoRows {
		return 0, ErrPortNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("get port by node and key: %w", err)
	}

	return portID, nil
}

func nodeExists(nodeID int) (bool, error) {
	var exists int
	err := appDB.QueryRow(`SELECT EXISTS(SELECT 1 FROM nodes WHERE id = ?)`, nodeID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check node exists: %w", err)
	}

	return exists == 1, nil
}
