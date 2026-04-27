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

	yesPortID, err := getPortIDByNodeAndKey(nodeID, "yes")
	if err != nil {
		t.Fatalf("get yes port for created node: %v", err)
	}
	if yesPortID == 0 {
		t.Fatalf("expected non-zero yes port ID for created node")
	}

	noPortID, err := getPortIDByNodeAndKey(nodeID, "no")
	if err != nil {
		t.Fatalf("get no port for created node: %v", err)
	}
	if noPortID == 0 {
		t.Fatalf("expected non-zero no port ID for created node")
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

func TestAdvanceSessionByPortRecordsHistoryWithZeroBasedIndexes(t *testing.T) {
	setupTestAppDB(t)

	sessionID, err := createSession()
	if err != nil {
		t.Fatalf("createSession returned error: %v", err)
	}

	rootYesPortID, err := getPortIDByNodeAndKey(1, "yes")
	if err != nil {
		t.Fatalf("get root yes port: %v", err)
	}
	if _, err := advanceSessionByPort(sessionID, rootYesPortID); err != nil {
		t.Fatalf("advance root yes: %v", err)
	}

	nodeTwoNoPortID, err := getPortIDByNodeAndKey(2, "no")
	if err != nil {
		t.Fatalf("get node 2 no port: %v", err)
	}
	if _, err := advanceSessionByPort(sessionID, nodeTwoNoPortID); err != nil {
		t.Fatalf("advance node 2 no: %v", err)
	}

	history, err := reconstructSessionPath(sessionID)
	if err != nil {
		t.Fatalf("reconstructSessionPath returned error: %v", err)
	}
	if len(history) != 2 {
		t.Fatalf("expected 2 history rows, got %d", len(history))
	}

	if history[0].SessionIndex != 0 || history[0].NodeID != 1 || history[0].PortID != rootYesPortID || history[0].PortKey != "yes" {
		t.Fatalf("unexpected first history row: %+v", history[0])
	}
	if history[1].SessionIndex != 1 || history[1].NodeID != 2 || history[1].PortID != nodeTwoNoPortID || history[1].PortKey != "no" {
		t.Fatalf("unexpected second history row: %+v", history[1])
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

func TestAdvanceSessionByPortRecordsTerminalHistory(t *testing.T) {
	setupTestAppDB(t)

	sessionID, err := createSession()
	if err != nil {
		t.Fatalf("createSession returned error: %v", err)
	}

	rootYesPortID, err := getPortIDByNodeAndKey(1, "yes")
	if err != nil {
		t.Fatalf("get root yes port: %v", err)
	}
	if _, err := advanceSessionByPort(sessionID, rootYesPortID); err != nil {
		t.Fatalf("advance root yes: %v", err)
	}

	nodeTwoYesPortID, err := getPortIDByNodeAndKey(2, "yes")
	if err != nil {
		t.Fatalf("get node 2 yes port: %v", err)
	}
	if _, err := advanceSessionByPort(sessionID, nodeTwoYesPortID); err != nil {
		t.Fatalf("advance node 2 yes: %v", err)
	}

	terminalPortID, err := getPortIDByNodeAndKey(4, "yes")
	if err != nil {
		t.Fatalf("get node 4 yes port: %v", err)
	}
	if status, err := advanceSessionByPort(sessionID, terminalPortID); err != nil || status != "complete" {
		t.Fatalf("expected terminal status complete, got status=%q err=%v", status, err)
	}

	history, err := reconstructSessionPath(sessionID)
	if err != nil {
		t.Fatalf("reconstructSessionPath returned error: %v", err)
	}
	if len(history) != 3 {
		t.Fatalf("expected 3 history rows, got %d", len(history))
	}

	lastStep := history[2]
	if lastStep.SessionIndex != 2 || lastStep.NodeID != 4 || lastStep.PortID != terminalPortID || lastStep.PortKey != "yes" {
		t.Fatalf("unexpected terminal history row: %+v", lastStep)
	}
}

func TestAdvanceSessionByPortUpdatesPathFingerprint(t *testing.T) {
	setupTestAppDB(t)

	sessionID, err := createSession()
	if err != nil {
		t.Fatalf("createSession returned error: %v", err)
	}

	sessionRecord, err := getSessionByID(sessionID)
	if err != nil {
		t.Fatalf("getSessionByID returned error: %v", err)
	}
	if sessionRecord.PathFingerprint != "" {
		t.Fatalf("expected empty initial fingerprint, got %q", sessionRecord.PathFingerprint)
	}

	rootYesPortID, err := getPortIDByNodeAndKey(1, "yes")
	if err != nil {
		t.Fatalf("get root yes port: %v", err)
	}
	if _, err := advanceSessionByPort(sessionID, rootYesPortID); err != nil {
		t.Fatalf("advance root yes: %v", err)
	}

	sessionRecord, err = getSessionByID(sessionID)
	if err != nil {
		t.Fatalf("getSessionByID after first advance returned error: %v", err)
	}
	expectedFingerprint := expectedPathFingerprint(rootYesPortID)
	if sessionRecord.PathFingerprint != expectedFingerprint {
		t.Fatalf("expected fingerprint %q, got %q", expectedFingerprint, sessionRecord.PathFingerprint)
	}

	nodeTwoNoPortID, err := getPortIDByNodeAndKey(2, "no")
	if err != nil {
		t.Fatalf("get node 2 no port: %v", err)
	}
	if _, err := advanceSessionByPort(sessionID, nodeTwoNoPortID); err != nil {
		t.Fatalf("advance node 2 no: %v", err)
	}

	sessionRecord, err = getSessionByID(sessionID)
	if err != nil {
		t.Fatalf("getSessionByID after second advance returned error: %v", err)
	}
	expectedFingerprint = expectedPathFingerprint(rootYesPortID, nodeTwoNoPortID)
	if sessionRecord.PathFingerprint != expectedFingerprint {
		t.Fatalf("expected fingerprint %q, got %q", expectedFingerprint, sessionRecord.PathFingerprint)
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

func TestAttachPortToNodeConnectsDanglingPort(t *testing.T) {
	setupTestAppDB(t)

	nodeID, err := createNode("yesno", "Is it imaginary?", "{}")
	if err != nil {
		t.Fatalf("createNode returned error: %v", err)
	}

	portID, err := getPortIDByNodeAndKey(4, "yes")
	if err != nil {
		t.Fatalf("get dangling port by key: %v", err)
	}

	if err := attachPortToNode(portID, nodeID); err != nil {
		t.Fatalf("attachPortToNode returned error: %v", err)
	}

	var attachedNodeID int
	if err := appDB.QueryRow(`SELECT to_node_id FROM ports WHERE id = ?`, portID).Scan(&attachedNodeID); err != nil {
		t.Fatalf("query attached port target: %v", err)
	}
	if attachedNodeID != nodeID {
		t.Fatalf("expected attached node ID %d, got %d", nodeID, attachedNodeID)
	}
}

func TestAttachPortToNodeRejectsAlreadyConnectedPort(t *testing.T) {
	setupTestAppDB(t)

	nodeID, err := createNode("yesno", "Is it imaginary?", "{}")
	if err != nil {
		t.Fatalf("createNode returned error: %v", err)
	}

	portID, err := getPortIDByNodeAndKey(1, "yes")
	if err != nil {
		t.Fatalf("get connected port by key: %v", err)
	}

	err = attachPortToNode(portID, nodeID)
	if !errors.Is(err, ErrPortAlreadyConnected) {
		t.Fatalf("expected ErrPortAlreadyConnected, got %v", err)
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

func expectedPathFingerprint(portIDs ...int) string {
	fingerprint := ""
	for _, portID := range portIDs {
		fingerprint = rollFingerprint(fingerprint, portID)
	}

	return fingerprint
}
