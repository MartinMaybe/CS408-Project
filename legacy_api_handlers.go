package main

import (
	"encoding/json"
	"log"
	"net/http"
)

func apiCurrentHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("apiCurrentHandler was called")

	node, ok := nodes[currentSession.CurrentNode]
	if !ok {
		http.Error(w, "No active session", http.StatusNotFound)
		return
	}

	response := map[string]interface{}{
		"question":  node.Question,
		"node_id":   node.ID,
		"decisions": currentSession.Decisions,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func apiChooseHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("apiChooseHandler was called")

	decision := r.URL.Query().Get("decision")

	currentNode, ok := nodes[currentSession.CurrentNode]
	if !ok {
		http.Error(w, "Invalid session", http.StatusNotFound)
		return
	}

	var nextNodeID int
	if decision == "yes" {
		nextNodeID = currentNode.YesNext
	} else if decision == "no" {
		nextNodeID = currentNode.NoNext
	} else {
		http.Error(w, "Invalid decision", http.StatusBadRequest)
		return
	}

	if _, ok := nodes[nextNodeID]; !ok {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "complete",
		})
		return
	}

	currentSession.CurrentNode = nextNodeID
	currentSession.Decisions++

	err := updateSessionCurrentNode(currentSession.ID, currentSession.CurrentNode)
	if err != nil {
		log.Println("DB update error:", err)
		http.Error(w, "DB update error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
	})
}

func startSessionHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("startSessionHandler was called")

	sessionID, err := createSession()
	if err != nil {
		log.Println("DB insert error:", err)
		http.Error(w, "DB Error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	currentSession = Session{
		ID:          sessionID,
		CurrentNode: rootNodeID,
		Decisions:   0,
	}

	http.Redirect(w, r, "/session", http.StatusSeeOther)
}
