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

	// End of tree
	if _, ok := nodes[nextNodeID]; !ok {

		_, err := appDB.Exec(`
			UPDATE sessions
			SET current_node_id = ?, path_length = ?
			WHERE id = ?`,
			currentSession.CurrentNode,
			currentSession.Decisions,
			currentSession.ID,
		)

		if err != nil {
			log.Println("DB update error:", err)
			http.Error(w, "Session completed but failed to update DB", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "complete",
		})
		return
	}

	currentSession.CurrentNode = nextNodeID
	currentSession.Decisions++

	// update session in DB
	_, err := appDB.Exec(`
		UPDATE sessions
		SET current_node_id = ?, path_length = ?
		WHERE id = ?`,
		currentSession.CurrentNode,
		currentSession.Decisions,
		currentSession.ID,
	)

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

	result, err := appDB.Exec(`
		INSERT INTO sessions (current_node_id, random_seed)
		VALUES (?, ?)`, 1, 12345)

	if err != nil {
		log.Println("DB Insert Error:", err)
		http.Error(w, "DB Error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	id, _ := result.LastInsertId()

	currentSession = Session{
		ID:          int(id),
		CurrentNode: 1,
		Decisions:   0,
	}

	http.Redirect(w, r, "/session", http.StatusSeeOther)
}
