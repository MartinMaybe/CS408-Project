package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
)

/* requests and response structs */
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
}

type CreateSessionResponse struct {
	SessionID int `json:"session_id"`
}

type SessionAdvanceRequest struct {
	SessionID int `json:"session_id"`
	PortID    int `json:"port_id"`
}

type SessionStatusResponse struct {
	Status string `json:"status"`
}

type PortLookupResponse struct {
	PortID int `json:"port_id"`
}

type PortAttachRequest struct {
	PortID   int `json:"port_id"`
	ToNodeID int `json:"to_node_id"`
}

type PortStatusResponse struct {
	Status string `json:"status"`
}

type CreateNodeRequest struct {
	Kind   string `json:"kind"`
	Prompt string `json:"prompt"`
	JSON   string `json:"json"`
}

type CreateNodeResponse struct {
	NodeID int `json:"node_id"`
}

/* Broad route handlers*/
func sessionsAPIHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		postSessionsAPIHandler(w, r)
	default:
		methodNotAllowed(w, http.MethodPost)
	}
}

func sessionAPIHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		getSessionAPIHandler(w, r)
	case http.MethodPost:
		postSessionAPIHandler(w, r)
	default:
		methodNotAllowed(w, "GET, POST")
	}
}

func portAPIHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		getPortAPIHandler(w, r)
	case http.MethodPost:
		postPortAPIHandler(w, r)
	default:
		methodNotAllowed(w, "GET, POST")
	}
}

func nodeAPIHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		getNodeAPIHandler(w, r)
	case http.MethodPost:
		postNodeAPIHandler(w, r)
	default:
		methodNotAllowed(w, "GET, POST")
	}
}

/* post endpoint handlers */
func postNodeAPIHandler(w http.ResponseWriter, r *http.Request) {
	var request CreateNodeRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	request.Kind = strings.TrimSpace(request.Kind)
	request.Prompt = strings.TrimSpace(request.Prompt)
	request.JSON = strings.TrimSpace(request.JSON)
	if request.Kind == "" {
		http.Error(w, "kind is required", http.StatusBadRequest)
		return
	}
	if request.Prompt == "" {
		http.Error(w, "prompt is required", http.StatusBadRequest)
		return
	}

	nodeID, err := createNode(request.Kind, request.Prompt, request.JSON)
	if err != nil {
		log.Println("create node error:", err)
		http.Error(w, "DB insert error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, CreateNodeResponse{NodeID: nodeID})
}

func postSessionsAPIHandler(w http.ResponseWriter, r *http.Request) {
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

func postSessionAPIHandler(w http.ResponseWriter, r *http.Request) {
	var request SessionAdvanceRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	if request.SessionID <= 0 {
		http.Error(w, "session_id must be a positive integer", http.StatusBadRequest)
		return
	}
	if request.PortID <= 0 {
		http.Error(w, "port_id must be a positive integer", http.StatusBadRequest)
		return
	}

	status, err := advanceSessionByPort(request.SessionID, request.PortID)
	if errors.Is(err, ErrSessionNotFound) {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}
	if errors.Is(err, ErrPortNotFound) {
		http.Error(w, "Port not found", http.StatusNotFound)
		return
	}
	if errors.Is(err, ErrPortDoesNotBelongToSession) {
		http.Error(w, "Port does not belong to current session node", http.StatusConflict)
		return
	}
	if errors.Is(err, ErrNullCurrentNode) {
		http.Error(w, "Session has no current node", http.StatusConflict)
		return
	}
	if err != nil {
		log.Println("advance session by port error:", err)
		http.Error(w, "DB update error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, SessionStatusResponse{Status: status})
}

func postPortAPIHandler(w http.ResponseWriter, r *http.Request) {
	var request PortAttachRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	if request.PortID <= 0 {
		http.Error(w, "port_id must be a positive integer", http.StatusBadRequest)
		return
	}
	if request.ToNodeID <= 0 {
		http.Error(w, "to_node_id must be a positive integer", http.StatusBadRequest)
		return
	}

	err := attachPortToNode(request.PortID, request.ToNodeID)
	if errors.Is(err, ErrPortNotFound) {
		http.Error(w, "Port not found", http.StatusNotFound)
		return
	}
	if errors.Is(err, ErrNodeNotFound) {
		http.Error(w, "Node not found", http.StatusNotFound)
		return
	}
	if errors.Is(err, ErrPortAlreadyConnected) {
		http.Error(w, "Port already connected", http.StatusConflict)
		return
	}
	if err != nil {
		log.Println("attach port error:", err)
		http.Error(w, "DB update error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, PortStatusResponse{Status: "ok"})
}

/* get endpoint handlers */
func getSessionAPIHandler(w http.ResponseWriter, r *http.Request) {
	sessionID, err := requiredQueryInt(r, "session_id")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

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

func getPortAPIHandler(w http.ResponseWriter, r *http.Request) {
	nodeID, err := requiredQueryInt(r, "node_id")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	portKey := strings.TrimSpace(r.URL.Query().Get("port_key"))
	if portKey == "" {
		http.Error(w, "port_key is required", http.StatusBadRequest)
		return
	}

	portID, err := getPortIDByNodeAndKey(nodeID, portKey)
	if errors.Is(err, ErrPortNotFound) {
		http.Error(w, "Port not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Println("get port error:", err)
		http.Error(w, "DB query error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, PortLookupResponse{PortID: portID})
}

func getNodeAPIHandler(w http.ResponseWriter, r *http.Request) {
	nodeID, err := requiredQueryInt(r, "node_id")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

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

/* helpers */
func writeJSON(w http.ResponseWriter, statusCode int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(payload)
}

func newCreateSessionResponse(row sessionRow) CreateSessionResponse {
	return CreateSessionResponse{
		SessionID: row.ID,
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
	}
}

func methodNotAllowed(w http.ResponseWriter, allowedMethod string) {
	w.Header().Set("Allow", allowedMethod)
	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func requiredQueryInt(r *http.Request, key string) (int, error) {
	value := strings.TrimSpace(r.URL.Query().Get(key))
	if value == "" {
		return 0, errors.New(key + " is required")
	}

	parsedValue, err := strconv.Atoi(value)
	if err != nil {
		return 0, errors.New(key + " must be an integer")
	}
	if parsedValue <= 0 {
		return 0, errors.New(key + " must be a positive integer")
	}

	return parsedValue, nil
}
