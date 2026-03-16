package main

import (
	"html/template"
	"log"
	"net/http"
	"time"
)

type Page struct {
	Title string
	Time string
}

var templates = template.Must(template.ParseFiles(
	"templates/landing.html",
	"templates/session.html",
	"templates/statistics.html",
))

func sessionHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("sessionHandler was called")

	err := templates.ExecuteTemplate(w, "session.html", nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func landingHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("landingHandler was called")

	now := time.Now()
	formatted := now.Format("2006-01-02 15:04")
	
	p := &Page{
		Title: "Public Decision Tree",
		Time:  formatted,
	}

	err := templates.ExecuteTemplate(w, "landing.html", p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func statisticsHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("statisticsHandler was called")

	err := templates.ExecuteTemplate(w, "statistics.html", nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func main() {
	http.HandleFunc("/", landingHandler)
	http.HandleFunc("/session", sessionHandler)
	http.HandleFunc("/statistics", statisticsHandler)

	log.Println("Starting server on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
