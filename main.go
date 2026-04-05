package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"
)

type Page struct {
	Title    string
	Time     string
	Question string
}

type Session struct {
	ID          int
	CurrentNode int
	Decisions   int
}

var currentSession = Session{}

type Node struct {
	ID       int
	Question string
	YesNext  int
	NoNext   int
}

var nodes = map[int]Node{
	1: {ID: 1, Question: "Is it an animal?", YesNext: 2, NoNext: 3},
	2: {ID: 2, Question: "Does it have fur?", YesNext: 4, NoNext: 5},
	3: {ID: 3, Question: "Is it a plant?", YesNext: 6, NoNext: 7},
	4: {ID: 4, Question: "Is it a dog?", YesNext: 0, NoNext: 0},
	5: {ID: 5, Question: "Is it a reptile?", YesNext: 0, NoNext: 0},
	6: {ID: 6, Question: "Is it a tree?", YesNext: 0, NoNext: 0},
	7: {ID: 7, Question: "Is it a flower?", YesNext: 0, NoNext: 0},
}

type SessionPage struct {
	Title       string
	Time        string
	Question    string
	DecisionNum int
}

type SessionResultPage struct {
	Title       string
	Time        string
	Decisions   int
	SummaryText string
}

var templates = template.Must(template.ParseFiles(
	"templates/landing.html",
	"templates/session.html",
	"templates/statistics.html",
	"templates/results.html",
))

func newPage() *Page {
	question := ""
	if node, ok := nodes[currentSession.CurrentNode]; ok {
		question = node.Question
	}

	return &Page{
		Title:    "Public Decision Tree",
		Time:     time.Now().Format("2006-01-02 15:04"),
		Question: question,
	}
}

// #region Request Handlers

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

	//update session in DB
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

	// currentSession = Session{
	// 	ID:          1,
	// 	CurrentNode: 1,
	// 	Decisions:   0,
	// }
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

	err := templates.ExecuteTemplate(w, "landing.html", newPage())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func statisticsHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("statisticsHandler was called")

	err := templates.ExecuteTemplate(w, "statistics.html", newPage())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func sessionResultsHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("sessionResultsHandler was called")

	// Summary text
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

//#endregion

func main() {
	var err error
	appDB, err = initializeDatabase()

	if err != nil {
		log.Fatal("Database initialization error:", err)
	}

	defer appDB.Close()

	http.HandleFunc("/", landingHandler)
	http.HandleFunc("/session", sessionHandler)
	http.HandleFunc("/session/choose", chooseHandler)
	http.HandleFunc("/statistics", statisticsHandler)
	http.HandleFunc("/api/start", startSessionHandler)
	http.HandleFunc("/session/results", sessionResultsHandler)
	http.HandleFunc("/api/session/current", apiCurrentHandler)
	http.HandleFunc("/api/session/choose", apiChooseHandler)

	log.Println("Starting server on http://localhost:8080")

	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal(err)
	}
}
