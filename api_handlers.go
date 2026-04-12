package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
)

type SessionResponse struct {
	ID              int    `json:"id"`
	CurrentNodeID   int    `json:"current_node_id"`
	PathLength      int    `json:"path_length"`
	RandomSeed      int64  `json:"random_seed"`
	PathFingerprint string `json:"path_fingerprint"`
	SessionText     string `json:"session_text"`
}

type NodeResponse struct {
	ID     int    `json:"id"`
	Kind   string `json:"kind"`
	Prompt string `json:"prompt"`
	JSON   string `json:"json"`
}

type CreateSessionResponse struct {
	SessionID     int `json:"session_id"`
	CurrentNodeID int `json:"current_node_id"`
}

type UpdateSessionCurrentRequest struct {
	NodeID int `json:"node_id"`
}

type UpdateSessionCurrentResponse struct {
	SessionID     int `json:"session_id"`
	CurrentNodeID int `json:"current_node_id"`
}

type PortLookupResponse struct {
	PortID int `json:"port_id"`
}

func sessionsAPIHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}

	postSessionHandler(w, r)
}

func sessionAPIHandler(w http.ResponseWriter, r *http.Request) {
	segments := apiPathSegments(r.URL.Path, "/api/sessions/")
	if len(segments) == 0 {
		http.NotFound(w, r)
		return
	}

	sessionID, err := strconv.Atoi(segments[0])
	if err != nil {
		http.Error(w, "Invalid session ID", http.StatusBadRequest)
		return
	}

	switch {
	case len(segments) == 1 && r.Method == http.MethodGet:
		getSessionHandler(w, r, sessionID)
	case len(segments) == 2 && segments[1] == "current" && r.Method == http.MethodPost:
		postSessionCurrentHandler(w, r, sessionID)
	case len(segments) == 1:
		methodNotAllowed(w, http.MethodGet)
	case len(segments) == 2 && segments[1] == "current":
		methodNotAllowed(w, http.MethodPost)
	default:
		http.NotFound(w, r)
	}
}

func nodeAPIHandler(w http.ResponseWriter, r *http.Request) {
	segments := apiPathSegments(r.URL.Path, "/api/nodes/")
	if len(segments) == 0 {
		http.NotFound(w, r)
		return
	}

	nodeID, err := strconv.Atoi(segments[0])
	if err != nil {
		http.Error(w, "Invalid node ID", http.StatusBadRequest)
		return
	}

	switch {
	case len(segments) == 1 && r.Method == http.MethodGet:
		getNodeHandler(w, r, nodeID)
	case len(segments) == 3 && segments[1] == "ports" && r.Method == http.MethodGet:
		getNodePortHandler(w, r, nodeID, segments[2])
	case len(segments) == 1:
		methodNotAllowed(w, http.MethodGet)
	case len(segments) == 3 && segments[1] == "ports":
		methodNotAllowed(w, http.MethodGet)
	default:
		http.NotFound(w, r)
	}
}

func postSessionHandler(w http.ResponseWriter, r *http.Request) {
	sessionID, err := createSession()
	if err != nil {
		log.Println("create session error:", err)
		http.Error(w, "DB Error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	createdSession, err := getSessionByID(sessionID)
	if err != nil {
		log.Println("get created session error:", err)
		http.Error(w, "DB query error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, newCreateSessionResponse(createdSession))
}

func postSessionCurrentHandler(w http.ResponseWriter, r *http.Request, sessionID int) {
	var request UpdateSessionCurrentRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	if request.NodeID <= 0 {
		http.Error(w, "node_id must be a positive integer", http.StatusBadRequest)
		return
	}

	err := updateSessionCurrentNode(sessionID, request.NodeID)
	if errors.Is(err, ErrNodeNotFound) {
		http.Error(w, "Node not found", http.StatusNotFound)
		return
	}
	if errors.Is(err, ErrSessionNotFound) {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Println("update session current node error:", err)
		http.Error(w, "DB update error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, UpdateSessionCurrentResponse{
		SessionID:     sessionID,
		CurrentNodeID: request.NodeID,
	})
}

func getSessionHandler(w http.ResponseWriter, r *http.Request, sessionID int) {
	record, err := getSessionByID(sessionID)
	if errors.Is(err, ErrSessionNotFound) {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Println("get session error:", err)
		http.Error(w, "DB query error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, newSessionResponse(record))
}

func getNodeHandler(w http.ResponseWriter, r *http.Request, nodeID int) {
	record, err := getNodeByID(nodeID)
	if errors.Is(err, ErrNodeNotFound) {
		http.Error(w, "Node not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Println("get node error:", err)
		http.Error(w, "DB query error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, newNodeResponse(record))
}

func getNodePortHandler(w http.ResponseWriter, r *http.Request, nodeID int, portKey string) {
	portID, err := getPortIDByNodeAndKey(nodeID, portKey)
	if errors.Is(err, ErrPortNotFound) {
		http.Error(w, "Port not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Println("get node port error:", err)
		http.Error(w, "DB query error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, PortLookupResponse{PortID: portID})
}

func writeJSON(w http.ResponseWriter, statusCode int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(payload)
}

func newCreateSessionResponse(row sessionRow) CreateSessionResponse {
	return CreateSessionResponse{
		SessionID:     row.ID,
		CurrentNodeID: row.CurrentNodeID,
	}
}

func newSessionResponse(row sessionRow) SessionResponse {
	return SessionResponse{
		ID:              row.ID,
		CurrentNodeID:   row.CurrentNodeID,
		PathLength:      row.PathLength,
		RandomSeed:      row.RandomSeed,
		PathFingerprint: row.PathFingerprint,
		SessionText:     row.SessionText,
	}
}

func newNodeResponse(row nodeRow) NodeResponse {
	return NodeResponse{
		ID:     row.ID,
		Kind:   row.Kind,
		Prompt: row.Prompt,
		JSON:   row.JSON,
	}
}

func methodNotAllowed(w http.ResponseWriter, allowedMethod string) {
	w.Header().Set("Allow", allowedMethod)
	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func apiPathSegments(path string, prefix string) []string {
	trimmedPath := strings.TrimPrefix(path, prefix)
	trimmedPath = strings.Trim(trimmedPath, "/")
	if trimmedPath == "" {
		return nil
	}

	return strings.Split(trimmedPath, "/")
}
