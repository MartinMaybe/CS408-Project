package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"testing"
)

func TestPostSessionAndGetSessionHandlers(t *testing.T) {
	setupTestDatabaseForAPI(t)

	createRequest := httptest.NewRequest(http.MethodPost, "/api/sessions", nil)
	createResponse := httptest.NewRecorder()

	sessionsAPIHandler(createResponse, createRequest)

	if createResponse.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, createResponse.Code)
	}

	var createdSession CreateSessionResponse
	if err := json.NewDecoder(createResponse.Body).Decode(&createdSession); err != nil {
		t.Fatalf("decode created session response: %v", err)
	}

	if createdSession.SessionID == 0 {
		t.Fatalf("expected created session ID to be non-zero")
	}

	getRequest := httptest.NewRequest(
		http.MethodGet,
		"/api/session?session_id="+strconv.Itoa(createdSession.SessionID),
		nil,
	)
	getResponse := httptest.NewRecorder()

	sessionAPIHandler(getResponse, getRequest)

	if getResponse.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, getResponse.Code)
	}

	var sessionRecord SessionResponse
	if err := json.NewDecoder(getResponse.Body).Decode(&sessionRecord); err != nil {
		t.Fatalf("decode session record: %v", err)
	}

	if sessionRecord.ID != createdSession.SessionID {
		t.Fatalf("expected session ID %d, got %d", createdSession.SessionID, sessionRecord.ID)
	}
	if sessionRecord.CurrentNodeID != 1 {
		t.Fatalf("expected current node ID 1, got %d", sessionRecord.CurrentNodeID)
	}
}

func TestPostSessionHandlerUpdatesCurrentNode(t *testing.T) {
	setupTestDatabaseForAPI(t)

	sessionID, err := createSession()
	if err != nil {
		t.Fatalf("create test session: %v", err)
	}

	portID, err := getPortIDByNodeAndKey(rootNodeID, "yes")
	if err != nil {
		t.Fatalf("get yes port ID: %v", err)
	}

	requestBody, err := json.Marshal(SessionAdvanceRequest{
		SessionID: sessionID,
		PortID:    portID,
	})
	if err != nil {
		t.Fatalf("marshal update request: %v", err)
	}

	updateRequest := httptest.NewRequest(
		http.MethodPost,
		"/api/session",
		bytes.NewReader(requestBody),
	)
	updateResponse := httptest.NewRecorder()

	sessionAPIHandler(updateResponse, updateRequest)

	if updateResponse.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, updateResponse.Code)
	}

	var updatedSession SessionStatusResponse
	if err := json.NewDecoder(updateResponse.Body).Decode(&updatedSession); err != nil {
		t.Fatalf("decode updated session response: %v", err)
	}

	if updatedSession.Status != "ok" {
		t.Fatalf("expected status %q, got %q", "ok", updatedSession.Status)
	}

	getRequest := httptest.NewRequest(
		http.MethodGet,
		"/api/session?session_id="+strconv.Itoa(sessionID),
		nil,
	)
	getResponse := httptest.NewRecorder()

	sessionAPIHandler(getResponse, getRequest)

	if getResponse.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, getResponse.Code)
	}

	var sessionRecord SessionResponse
	if err := json.NewDecoder(getResponse.Body).Decode(&sessionRecord); err != nil {
		t.Fatalf("decode session record: %v", err)
	}

	if sessionRecord.CurrentNodeID != 2 {
		t.Fatalf("expected current node ID 2, got %d", sessionRecord.CurrentNodeID)
	}
	if sessionRecord.PathLength != 1 {
		t.Fatalf("expected path length 1, got %d", sessionRecord.PathLength)
	}
}

func TestGetNodeHandlerReturnsSeededNode(t *testing.T) {
	setupTestDatabaseForAPI(t)

	request := httptest.NewRequest(http.MethodGet, "/api/node?node_id=1", nil)
	response := httptest.NewRecorder()

	nodeAPIHandler(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}

	var nodeRecord NodeResponse
	if err := json.NewDecoder(response.Body).Decode(&nodeRecord); err != nil {
		t.Fatalf("decode node record: %v", err)
	}

	if nodeRecord.ID != 1 {
		t.Fatalf("expected node ID 1, got %d", nodeRecord.ID)
	}
	if nodeRecord.Prompt != "Is it an animal?" {
		t.Fatalf("expected prompt %q, got %q", "Is it an animal?", nodeRecord.Prompt)
	}
}

func TestGetNodePortHandlerReturnsPortID(t *testing.T) {
	setupTestDatabaseForAPI(t)

	request := httptest.NewRequest(http.MethodGet, "/api/port?node_id=1&port_key=yes", nil)
	response := httptest.NewRecorder()

	portAPIHandler(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}

	var portResponse PortLookupResponse
	if err := json.NewDecoder(response.Body).Decode(&portResponse); err != nil {
		t.Fatalf("decode port response: %v", err)
	}

	if portResponse.PortID == 0 {
		t.Fatalf("expected non-zero port ID")
	}
}

func TestPostNodeHandlerCreatesNode(t *testing.T) {
	setupTestDatabaseForAPI(t)

	requestBody, err := json.Marshal(CreateNodeRequest{
		Kind:   "yesno",
		Prompt: "Is it imaginary?",
		JSON:   "{}",
	})
	if err != nil {
		t.Fatalf("marshal create node request: %v", err)
	}

	request := httptest.NewRequest(http.MethodPost, "/api/node", bytes.NewReader(requestBody))
	response := httptest.NewRecorder()

	nodeAPIHandler(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, response.Code)
	}

	var createdNode CreateNodeResponse
	if err := json.NewDecoder(response.Body).Decode(&createdNode); err != nil {
		t.Fatalf("decode create node response: %v", err)
	}

	if createdNode.NodeID == 0 {
		t.Fatalf("expected created node ID to be non-zero")
	}

	nodeRecord, err := getNodeByID(createdNode.NodeID)
	if err != nil {
		t.Fatalf("get created node by ID: %v", err)
	}

	if nodeRecord.Prompt != "Is it imaginary?" {
		t.Fatalf("expected prompt %q, got %q", "Is it imaginary?", nodeRecord.Prompt)
	}
}

func setupTestDatabaseForAPI(t *testing.T) {
	t.Helper()

	oldDB := appDB
	oldSession := currentSession

	dbPath := filepath.Join(t.TempDir(), "api-test.db")
	db, err := openDatabase(dbPath)
	if err != nil {
		t.Fatalf("openDatabase returned error: %v", err)
	}

	appDB = db
	currentSession = Session{}

	t.Cleanup(func() {
		currentSession = oldSession
		appDB = oldDB
		db.Close()
	})
}
