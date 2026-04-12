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
	if createdSession.CurrentNodeID != 1 {
		t.Fatalf("expected created session current node to be 1, got %d", createdSession.CurrentNodeID)
	}

	getRequest := httptest.NewRequest(http.MethodGet, "/api/sessions/"+strconv.Itoa(createdSession.SessionID), nil)
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

func TestPostSessionCurrentHandlerUpdatesCurrentNode(t *testing.T) {
	setupTestDatabaseForAPI(t)

	sessionID, err := createSession()
	if err != nil {
		t.Fatalf("create test session: %v", err)
	}

	requestBody, err := json.Marshal(UpdateSessionCurrentRequest{NodeID: 2})
	if err != nil {
		t.Fatalf("marshal update request: %v", err)
	}

	updateRequest := httptest.NewRequest(
		http.MethodPost,
		"/api/sessions/"+strconv.Itoa(sessionID)+"/current",
		bytes.NewReader(requestBody),
	)
	updateResponse := httptest.NewRecorder()

	sessionAPIHandler(updateResponse, updateRequest)

	if updateResponse.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, updateResponse.Code)
	}

	var updatedSession UpdateSessionCurrentResponse
	if err := json.NewDecoder(updateResponse.Body).Decode(&updatedSession); err != nil {
		t.Fatalf("decode updated session response: %v", err)
	}

	if updatedSession.CurrentNodeID != 2 {
		t.Fatalf("expected updated current node ID 2, got %d", updatedSession.CurrentNodeID)
	}

	getRequest := httptest.NewRequest(http.MethodGet, "/api/sessions/"+strconv.Itoa(sessionID), nil)
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

	request := httptest.NewRequest(http.MethodGet, "/api/nodes/1", nil)
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

	request := httptest.NewRequest(http.MethodGet, "/api/nodes/1/ports/yes", nil)
	response := httptest.NewRecorder()

	nodeAPIHandler(response, request)

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
