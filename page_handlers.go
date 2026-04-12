package main

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

func sessionHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("sessionHandler was called")

	node, ok := nodes[currentSession.CurrentNode]
	if !ok {
		http.Redirect(w, r, "/session/results", http.StatusSeeOther)
		return
	}

	page := SessionPage{
		Title:       "Public Decision Tree - Session",
		Time:        time.Now().Format("2006-01-02 15:04"),
		Question:    node.Question,
		DecisionNum: currentSession.Decisions,
	}

	err := templates.ExecuteTemplate(w, "session.html", page)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func chooseHandler(w http.ResponseWriter, r *http.Request) {
	decision := r.URL.Query().Get("decision")
	currentNode, ok := nodes[currentSession.CurrentNode]
	if !ok {
		http.Error(w, "Node not found", http.StatusNotFound)
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

	// Check that next node exists
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

		http.Redirect(w, r, "/session/results", http.StatusSeeOther)
		return
	}

	currentSession.CurrentNode = nextNodeID
	currentSession.Decisions++

	// Update session in DB
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

	http.Redirect(w, r, "/session", http.StatusSeeOther)
}

func landingHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("landingHandler was called")

	err := templates.ExecuteTemplate(w, "landing.html", newPage("Public Decision Tree"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func statisticsHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("statisticsHandler was called")

	err := templates.ExecuteTemplate(w, "statistics.html", newPage("Statistics"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func sessionResultsHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("sessionResultsHandler was called")

	summary := "This session has completed. Decisions made: " +
		fmt.Sprintf("%d", currentSession.Decisions)

	page := SessionResultPage{
		Title:       "Session Results",
		Time:        time.Now().Format("2006-01-02 15:04"),
		Decisions:   currentSession.Decisions,
		SummaryText: summary,
	}

	err := templates.ExecuteTemplate(w, "results.html", page)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
