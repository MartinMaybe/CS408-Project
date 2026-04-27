package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

func TestSessionsAPIHandlerCreatesSession(t *testing.T) {
	setupTestAppDB(t)

	sessionID := createSessionViaAPI(t)
	sessionRecord := getSessionViaAPI(t, sessionID)

	if sessionRecord.ID != sessionID {
		t.Fatalf("expected session ID %d, got %d", sessionID, sessionRecord.ID)
	}
	if sessionRecord.CurrentNodeID != rootNodeID {
		t.Fatalf("expected current node ID %d, got %d", rootNodeID, sessionRecord.CurrentNodeID)
	}
	if sessionRecord.PathLength != 0 {
		t.Fatalf("expected path length 0, got %d", sessionRecord.PathLength)
	}
}

func TestSessionsAPIHandlerRejectsGet(t *testing.T) {
	setupTestAppDB(t)

	request := httptest.NewRequest(http.MethodGet, "/api/sessions", nil)
	response := httptest.NewRecorder()

	sessionsAPIHandler(response, request)

	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, response.Code)
	}
	if allowHeader := response.Header().Get("Allow"); allowHeader != http.MethodPost {
		t.Fatalf("expected Allow header %q, got %q", http.MethodPost, allowHeader)
	}
}

func TestSessionAPIHandlerRequiresSessionIDForGet(t *testing.T) {
	setupTestAppDB(t)

	request := httptest.NewRequest(http.MethodGet, "/api/session", nil)
	response := httptest.NewRecorder()

	sessionAPIHandler(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, response.Code)
	}
}

func TestSessionAPIHandlerAdvancesSessionByPort(t *testing.T) {
	setupTestAppDB(t)

	sessionID := createSessionViaAPI(t)
	portID := getPortViaAPI(t, 1, "yes")
	status := advanceSessionViaAPI(t, sessionID, portID)

	if status.Status != "ok" {
		t.Fatalf("expected status %q, got %q", "ok", status.Status)
	}

	sessionRecord := getSessionViaAPI(t, sessionID)
	if sessionRecord.CurrentNodeID != 2 {
		t.Fatalf("expected current node ID 2, got %d", sessionRecord.CurrentNodeID)
	}
	if sessionRecord.PathLength != 1 {
		t.Fatalf("expected path length 1, got %d", sessionRecord.PathLength)
	}
}

func TestSessionAPIHandlerReturnsCompleteForTerminalPort(t *testing.T) {
	setupTestAppDB(t)

	sessionID := createSessionViaAPI(t)

	status := advanceSessionViaAPI(t, sessionID, getPortViaAPI(t, 1, "yes"))
	if status.Status != "ok" {
		t.Fatalf("expected first status %q, got %q", "ok", status.Status)
	}

	status = advanceSessionViaAPI(t, sessionID, getPortViaAPI(t, 2, "yes"))
	if status.Status != "ok" {
		t.Fatalf("expected second status %q, got %q", "ok", status.Status)
	}

	status = advanceSessionViaAPI(t, sessionID, getPortViaAPI(t, 4, "yes"))
	if status.Status != "complete" {
		t.Fatalf("expected final status %q, got %q", "complete", status.Status)
	}

	sessionRecord := getSessionViaAPI(t, sessionID)
	if sessionRecord.CurrentNodeID != 4 {
		t.Fatalf("expected current node ID 4, got %d", sessionRecord.CurrentNodeID)
	}
	if sessionRecord.PathLength != 3 {
		t.Fatalf("expected path length 3, got %d", sessionRecord.PathLength)
	}
}

func TestSessionAPIHandlerRejectsPortFromDifferentNode(t *testing.T) {
	setupTestAppDB(t)

	sessionID := createSessionViaAPI(t)
	portID := getPortViaAPI(t, 2, "yes")

	requestBody, err := json.Marshal(SessionAdvanceRequest{
		SessionID: sessionID,
		PortID:    portID,
	})
	if err != nil {
		t.Fatalf("marshal advance request: %v", err)
	}

	request := httptest.NewRequest(http.MethodPost, "/api/session", bytes.NewReader(requestBody))
	response := httptest.NewRecorder()

	sessionAPIHandler(response, request)

	if response.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d", http.StatusConflict, response.Code)
	}
}

func TestSessionHistoryAPIHandlerReturnsHistory(t *testing.T) {
	setupTestAppDB(t)

	sessionID := createSessionViaAPI(t)
	rootYesPortID := getPortViaAPI(t, 1, "yes")
	advanceSessionViaAPI(t, sessionID, rootYesPortID)
	nodeTwoNoPortID := getPortViaAPI(t, 2, "no")
	advanceSessionViaAPI(t, sessionID, nodeTwoNoPortID)

	history := getSessionHistoryViaAPI(t, sessionID)
	if len(history) != 2 {
		t.Fatalf("expected 2 history rows, got %d", len(history))
	}

	firstStep := history[0]
	if firstStep.SessionIndex != 0 || firstStep.NodeID != 1 || firstStep.NodePrompt != "Is it an animal?" || firstStep.PortID != rootYesPortID || firstStep.PortKey != "yes" {
		t.Fatalf("unexpected first history step: %+v", firstStep)
	}

	secondStep := history[1]
	if secondStep.SessionIndex != 1 || secondStep.NodeID != 2 || secondStep.NodePrompt != "Does it have fur?" || secondStep.PortID != nodeTwoNoPortID || secondStep.PortKey != "no" {
		t.Fatalf("unexpected second history step: %+v", secondStep)
	}
}

func TestSessionHistoryAPIHandlerReturnsEmptyHistoryForNewSession(t *testing.T) {
	setupTestAppDB(t)

	sessionID := createSessionViaAPI(t)
	history := getSessionHistoryViaAPI(t, sessionID)
	if len(history) != 0 {
		t.Fatalf("expected empty history, got %d rows", len(history))
	}
}

func TestSessionHistoryAPIHandlerRequiresSessionID(t *testing.T) {
	setupTestAppDB(t)

	request := httptest.NewRequest(http.MethodGet, "/api/session/history", nil)
	response := httptest.NewRecorder()

	sessionHistoryAPIHandler(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, response.Code)
	}
}

func TestSessionHistoryAPIHandlerReturnsNotFound(t *testing.T) {
	setupTestAppDB(t)

	request := httptest.NewRequest(http.MethodGet, "/api/session/history?session_id=999", nil)
	response := httptest.NewRecorder()

	sessionHistoryAPIHandler(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, response.Code)
	}
}

func TestSessionHistoryAPIHandlerRejectsPost(t *testing.T) {
	setupTestAppDB(t)

	request := httptest.NewRequest(http.MethodPost, "/api/session/history?session_id=1", nil)
	response := httptest.NewRecorder()

	sessionHistoryAPIHandler(response, request)

	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, response.Code)
	}
	if allowHeader := response.Header().Get("Allow"); allowHeader != http.MethodGet {
		t.Fatalf("expected Allow header %q, got %q", http.MethodGet, allowHeader)
	}
}

func TestPortAPIHandlerReturnsPortID(t *testing.T) {
	setupTestAppDB(t)

	portID := getPortViaAPI(t, 1, "yes")
	if portID == 0 {
		t.Fatalf("expected non-zero port ID")
	}
}

func TestPortAPIHandlerPostAttachesDanglingPort(t *testing.T) {
	setupTestAppDB(t)

	nodeRequestBody, err := json.Marshal(CreateNodeRequest{
		Kind:   "yesno",
		Prompt: "Is it imaginary?",
		JSON:   "{}",
	})
	if err != nil {
		t.Fatalf("marshal create node request: %v", err)
	}

	nodeRequest := httptest.NewRequest(http.MethodPost, "/api/node", bytes.NewReader(nodeRequestBody))
	nodeResponse := httptest.NewRecorder()

	nodeAPIHandler(nodeResponse, nodeRequest)

	if nodeResponse.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, nodeResponse.Code)
	}

	var createdNode CreateNodeResponse
	if err := json.NewDecoder(nodeResponse.Body).Decode(&createdNode); err != nil {
		t.Fatalf("decode create node response: %v", err)
	}

	portID := getPortViaAPI(t, 4, "yes")

	attachRequestBody, err := json.Marshal(PortAttachRequest{
		PortID:   portID,
		ToNodeID: createdNode.NodeID,
	})
	if err != nil {
		t.Fatalf("marshal attach port request: %v", err)
	}

	attachRequest := httptest.NewRequest(http.MethodPost, "/api/port", bytes.NewReader(attachRequestBody))
	attachResponse := httptest.NewRecorder()

	portAPIHandler(attachResponse, attachRequest)

	if attachResponse.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, attachResponse.Code)
	}

	var attachStatus PortStatusResponse
	if err := json.NewDecoder(attachResponse.Body).Decode(&attachStatus); err != nil {
		t.Fatalf("decode attach status response: %v", err)
	}
	if attachStatus.Status != "ok" {
		t.Fatalf("expected status %q, got %q", "ok", attachStatus.Status)
	}

	sessionID := createSessionViaAPI(t)
	status := advanceSessionViaAPI(t, sessionID, getPortViaAPI(t, 1, "yes"))
	if status.Status != "ok" {
		t.Fatalf("expected first advance status %q, got %q", "ok", status.Status)
	}

	status = advanceSessionViaAPI(t, sessionID, getPortViaAPI(t, 2, "yes"))
	if status.Status != "ok" {
		t.Fatalf("expected second advance status %q, got %q", "ok", status.Status)
	}

	status = advanceSessionViaAPI(t, sessionID, portID)
	if status.Status != "ok" {
		t.Fatalf("expected attached branch status %q, got %q", "ok", status.Status)
	}

	sessionRecord := getSessionViaAPI(t, sessionID)
	if sessionRecord.CurrentNodeID != createdNode.NodeID {
		t.Fatalf("expected current node ID %d, got %d", createdNode.NodeID, sessionRecord.CurrentNodeID)
	}
}

func TestPortAPIHandlerRequiresPortKey(t *testing.T) {
	setupTestAppDB(t)

	request := httptest.NewRequest(http.MethodGet, "/api/port?node_id=1", nil)
	response := httptest.NewRecorder()

	portAPIHandler(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, response.Code)
	}
}

func TestNodeAPIHandlerGetReturnsSeededNode(t *testing.T) {
	setupTestAppDB(t)

	request := httptest.NewRequest(http.MethodGet, "/api/node?node_id=1", nil)
	response := httptest.NewRecorder()

	nodeAPIHandler(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}

	var nodeRecord NodeResponse
	if err := json.NewDecoder(response.Body).Decode(&nodeRecord); err != nil {
		t.Fatalf("decode node response: %v", err)
	}

	if nodeRecord.ID != 1 {
		t.Fatalf("expected node ID 1, got %d", nodeRecord.ID)
	}
	if nodeRecord.Prompt != "Is it an animal?" {
		t.Fatalf("expected prompt %q, got %q", "Is it an animal?", nodeRecord.Prompt)
	}
}

func TestNodeAPIHandlerPostCreatesNodeWithDefaultConfig(t *testing.T) {
	setupTestAppDB(t)

	requestBody, err := json.Marshal(CreateNodeRequest{
		Kind:   "yesno",
		Prompt: "Is it imaginary?",
		JSON:   "{}",
	})
	if err != nil {
		t.Fatalf("marshal create node request: %v", err)
	}

	postRequest := httptest.NewRequest(http.MethodPost, "/api/node", bytes.NewReader(requestBody))
	postResponse := httptest.NewRecorder()

	nodeAPIHandler(postResponse, postRequest)

	if postResponse.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, postResponse.Code)
	}

	var createdNode CreateNodeResponse
	if err := json.NewDecoder(postResponse.Body).Decode(&createdNode); err != nil {
		t.Fatalf("decode create node response: %v", err)
	}

	getRequest := httptest.NewRequest(
		http.MethodGet,
		"/api/node?node_id="+strconv.Itoa(createdNode.NodeID),
		nil,
	)
	getResponse := httptest.NewRecorder()

	nodeAPIHandler(getResponse, getRequest)

	if getResponse.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, getResponse.Code)
	}

	var nodeRecord NodeResponse
	if err := json.NewDecoder(getResponse.Body).Decode(&nodeRecord); err != nil {
		t.Fatalf("decode created node response: %v", err)
	}

	if nodeRecord.Prompt != "Is it imaginary?" {
		t.Fatalf("expected prompt %q, got %q", "Is it imaginary?", nodeRecord.Prompt)
	}

	yesPortID := getPortViaAPI(t, createdNode.NodeID, "yes")
	if yesPortID == 0 {
		t.Fatalf("expected non-zero yes port ID for created node")
	}
}

func createSessionViaAPI(t *testing.T) int {
	t.Helper()

	request := httptest.NewRequest(http.MethodPost, "/api/sessions", nil)
	response := httptest.NewRecorder()

	sessionsAPIHandler(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, response.Code)
	}

	var createdSession CreateSessionResponse
	if err := json.NewDecoder(response.Body).Decode(&createdSession); err != nil {
		t.Fatalf("decode created session response: %v", err)
	}

	if createdSession.SessionID == 0 {
		t.Fatalf("expected non-zero session ID")
	}

	return createdSession.SessionID
}

func getSessionViaAPI(t *testing.T, sessionID int) SessionResponse {
	t.Helper()

	request := httptest.NewRequest(
		http.MethodGet,
		"/api/session?session_id="+strconv.Itoa(sessionID),
		nil,
	)
	response := httptest.NewRecorder()

	sessionAPIHandler(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}

	var sessionRecord SessionResponse
	if err := json.NewDecoder(response.Body).Decode(&sessionRecord); err != nil {
		t.Fatalf("decode session response: %v", err)
	}

	return sessionRecord
}

func getPortViaAPI(t *testing.T, nodeID int, portKey string) int {
	t.Helper()

	request := httptest.NewRequest(
		http.MethodGet,
		"/api/port?node_id="+strconv.Itoa(nodeID)+"&port_key="+portKey,
		nil,
	)
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

	return portResponse.PortID
}

func advanceSessionViaAPI(t *testing.T, sessionID int, portID int) SessionStatusResponse {
	t.Helper()

	requestBody, err := json.Marshal(SessionAdvanceRequest{
		SessionID: sessionID,
		PortID:    portID,
	})
	if err != nil {
		t.Fatalf("marshal advance request: %v", err)
	}

	request := httptest.NewRequest(http.MethodPost, "/api/session", bytes.NewReader(requestBody))
	response := httptest.NewRecorder()

	sessionAPIHandler(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}

	var statusResponse SessionStatusResponse
	if err := json.NewDecoder(response.Body).Decode(&statusResponse); err != nil {
		t.Fatalf("decode session status response: %v", err)
	}

	return statusResponse
}

func getSessionHistoryViaAPI(t *testing.T, sessionID int) []SessionPathStepResponse {
	t.Helper()

	request := httptest.NewRequest(
		http.MethodGet,
		"/api/session/history?session_id="+strconv.Itoa(sessionID),
		nil,
	)
	response := httptest.NewRecorder()

	sessionHistoryAPIHandler(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}

	var history SessionHistoryResponse
	if err := json.NewDecoder(response.Body).Decode(&history); err != nil {
		t.Fatalf("decode session history response: %v", err)
	}
	if history.SessionID != sessionID {
		t.Fatalf("expected session ID %d, got %d", sessionID, history.SessionID)
	}

	return history.Steps
}
