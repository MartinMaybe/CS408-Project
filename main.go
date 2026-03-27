package main

import (
	"html/template"
	"log"
	"net/http"
	"time"
)

type Page struct {
	Title string
	Time  string
}

var templates = template.Must(template.ParseFiles(
	"templates/landing.html",
	"templates/session.html",
	"templates/statistics.html",
))

func newPage() *Page {
	return &Page{
		Title: "Public Decision Tree",
		Time:  time.Now().Format("2006-01-02 15:04"),
	}
}

// #region Request Handlers

func sessionHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("sessionHandler was called")

	err := templates.ExecuteTemplate(w, "session.html", newPage())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
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

//#endregion

func main() {
	var err error
	appDB, err = initializeDatabase()

	if err != nil {
		log.Fatal(err)
	}

	defer appDB.Close()

	http.HandleFunc("/", landingHandler)
	http.HandleFunc("/session", sessionHandler)
	http.HandleFunc("/statistics", statisticsHandler)

	log.Println("Starting server on http://localhost:8080")

	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal(err)
	}
}
