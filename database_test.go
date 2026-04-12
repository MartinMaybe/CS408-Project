package main

import (
	"errors"
	"path/filepath"
	"testing"
)

/*
* Test database is opened, tables are created, and foreign keys are enabled
 */
func TestOpenDatabaseCreatesSchemaAndSeedData(t *testing.T) {
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

	var nodeCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM nodes`).Scan(&nodeCount); err != nil {
		t.Fatalf("count seeded nodes: %v", err)
	}
	if nodeCount != 7 {
		t.Fatalf("expected 7 seeded nodes, got %d", nodeCount)
	}

	var portCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM ports`).Scan(&portCount); err != nil {
		t.Fatalf("count seeded ports: %v", err)
	}
	if portCount != 14 {
		t.Fatalf("expected 14 seeded ports, got %d", portCount)
	}
}

func TestCreateSessionStartsAtRootNode(t *testing.T) {
	setupTestAppDB(t)

	sessionID, err := createSession()
	if err != nil {
		t.Fatalf("createSession returned error: %v", err)
	}

	sessionRecord, err := getSessionByID(sessionID)
	if err != nil {
		t.Fatalf("getSessionByID returned error: %v", err)
	}

	if sessionRecord.ID != sessionID {
		t.Fatalf("expected session ID %d, got %d", sessionID, sessionRecord.ID)
	}
	if sessionRecord.CurrentNodeID != rootNodeID {
		t.Fatalf("expected current node ID %d, got %d", rootNodeID, sessionRecord.CurrentNodeID)
	}
	if sessionRecord.PathLength != 0 {
		t.Fatalf("expected path length 0, got %d", sessionRecord.PathLength)
	}
	if sessionRecord.RandomSeed == 0 {
		t.Fatalf("expected non-zero random seed")
	}
}

func TestCreateNodePersistsNode(t *testing.T) {
	setupTestAppDB(t)

	nodeID, err := createNode("yesno", "Is it imaginary?", "{}")
	if err != nil {
		t.Fatalf("createNode returned error: %v", err)
	}

	nodeRecord, err := getNodeByID(nodeID)
	if err != nil {
		t.Fatalf("getNodeByID returned error: %v", err)
	}

	if nodeRecord.ID != nodeID {
		t.Fatalf("expected node ID %d, got %d", nodeID, nodeRecord.ID)
	}
	if nodeRecord.Kind != "yesno" {
		t.Fatalf("expected kind %q, got %q", "yesno", nodeRecord.Kind)
	}
	if nodeRecord.Prompt != "Is it imaginary?" {
		t.Fatalf("expected prompt %q, got %q", "Is it imaginary?", nodeRecord.Prompt)
	}
	if nodeRecord.JSON != "{}" {
		t.Fatalf("expected json %q, got %q", "{}", nodeRecord.JSON)
	}
}

func TestGetPortIDByNodeAndKeyReturnsSeededPort(t *testing.T) {
	setupTestAppDB(t)

	portID, err := getPortIDByNodeAndKey(1, "yes")
	if err != nil {
		t.Fatalf("getPortIDByNodeAndKey returned error: %v", err)
	}

	if portID == 0 {
		t.Fatalf("expected non-zero port ID")
	}
}

func TestGetPortIDByNodeAndKeyReturnsNotFound(t *testing.T) {
	setupTestAppDB(t)

	_, err := getPortIDByNodeAndKey(1, "maybe")
	if !errors.Is(err, ErrPortNotFound) {
		t.Fatalf("expected ErrPortNotFound, got %v", err)
	}
}

func TestAdvanceSessionByPortMovesSessionAndIncrementsTakenCount(t *testing.T) {
	setupTestAppDB(t)

	sessionID, err := createSession()
	if err != nil {
		t.Fatalf("createSession returned error: %v", err)
	}

	portID, err := getPortIDByNodeAndKey(1, "yes")
	if err != nil {
		t.Fatalf("getPortIDByNodeAndKey returned error: %v", err)
	}

	status, err := advanceSessionByPort(sessionID, portID)
	if err != nil {
		t.Fatalf("advanceSessionByPort returned error: %v", err)
	}
	if status != "ok" {
		t.Fatalf("expected status %q, got %q", "ok", status)
	}

	sessionRecord, err := getSessionByID(sessionID)
	if err != nil {
		t.Fatalf("getSessionByID returned error: %v", err)
	}
	if sessionRecord.CurrentNodeID != 2 {
		t.Fatalf("expected current node ID 2, got %d", sessionRecord.CurrentNodeID)
	}
	if sessionRecord.PathLength != 1 {
		t.Fatalf("expected path length 1, got %d", sessionRecord.PathLength)
	}

	var takenCount int
	if err := appDB.QueryRow(`SELECT taken_count FROM ports WHERE id = ?`, portID).Scan(&takenCount); err != nil {
		t.Fatalf("query port taken_count: %v", err)
	}
	if takenCount != 1 {
		t.Fatalf("expected taken_count 1, got %d", takenCount)
	}
}

func TestAdvanceSessionByPortCompletesDanglingBranch(t *testing.T) {
	setupTestAppDB(t)

	sessionID, err := createSession()
	if err != nil {
		t.Fatalf("createSession returned error: %v", err)
	}

	rootYesPortID, err := getPortIDByNodeAndKey(1, "yes")
	if err != nil {
		t.Fatalf("get root yes port: %v", err)
	}
	if status, err := advanceSessionByPort(sessionID, rootYesPortID); err != nil || status != "ok" {
		t.Fatalf("expected first advance to return ok, got status=%q err=%v", status, err)
	}

	nodeTwoYesPortID, err := getPortIDByNodeAndKey(2, "yes")
	if err != nil {
		t.Fatalf("get node 2 yes port: %v", err)
	}
	if status, err := advanceSessionByPort(sessionID, nodeTwoYesPortID); err != nil || status != "ok" {
		t.Fatalf("expected second advance to return ok, got status=%q err=%v", status, err)
	}

	terminalPortID, err := getPortIDByNodeAndKey(4, "yes")
	if err != nil {
		t.Fatalf("get terminal port: %v", err)
	}

	status, err := advanceSessionByPort(sessionID, terminalPortID)
	if err != nil {
		t.Fatalf("advanceSessionByPort returned error: %v", err)
	}
	if status != "complete" {
		t.Fatalf("expected status %q, got %q", "complete", status)
	}

	sessionRecord, err := getSessionByID(sessionID)
	if err != nil {
		t.Fatalf("getSessionByID returned error: %v", err)
	}
	if sessionRecord.CurrentNodeID != 4 {
		t.Fatalf("expected current node ID 4, got %d", sessionRecord.CurrentNodeID)
	}
	if sessionRecord.PathLength != 3 {
		t.Fatalf("expected path length 3, got %d", sessionRecord.PathLength)
	}

	var takenCount int
	if err := appDB.QueryRow(`SELECT taken_count FROM ports WHERE id = ?`, terminalPortID).Scan(&takenCount); err != nil {
		t.Fatalf("query terminal port taken_count: %v", err)
	}
	if takenCount != 1 {
		t.Fatalf("expected terminal taken_count 1, got %d", takenCount)
	}
}

func TestAdvanceSessionByPortRejectsPortFromDifferentNode(t *testing.T) {
	setupTestAppDB(t)

	sessionID, err := createSession()
	if err != nil {
		t.Fatalf("createSession returned error: %v", err)
	}

	portID, err := getPortIDByNodeAndKey(2, "yes")
	if err != nil {
		t.Fatalf("getPortIDByNodeAndKey returned error: %v", err)
	}

	status, err := advanceSessionByPort(sessionID, portID)
	if !errors.Is(err, ErrPortDoesNotBelongToSession) {
		t.Fatalf("expected ErrPortDoesNotBelongToSession, got status=%q err=%v", status, err)
	}

	sessionRecord, err := getSessionByID(sessionID)
	if err != nil {
		t.Fatalf("getSessionByID returned error: %v", err)
	}
	if sessionRecord.CurrentNodeID != rootNodeID {
		t.Fatalf("expected current node ID %d, got %d", rootNodeID, sessionRecord.CurrentNodeID)
	}
	if sessionRecord.PathLength != 0 {
		t.Fatalf("expected path length 0, got %d", sessionRecord.PathLength)
	}
}

func TestGetSessionByIDReturnsNullCurrentNodeError(t *testing.T) {
	setupTestAppDB(t)

	result, err := appDB.Exec(`
		INSERT INTO sessions (current_node_id, random_seed)
		VALUES (NULL, ?)`,
		12345,
	)
	if err != nil {
		t.Fatalf("insert null current_node_id session: %v", err)
	}

	sessionID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("read inserted session ID: %v", err)
	}

	_, err = getSessionByID(int(sessionID))
	if !errors.Is(err, ErrNullCurrentNode) {
		t.Fatalf("expected ErrNullCurrentNode, got %v", err)
	}
}
