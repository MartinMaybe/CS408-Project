package main

import (
	"log"
	"net/http"
)

func sessionHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("sessionHandler was called")

	err := templates.ExecuteTemplate(w, "session.html", newPage("Public Decision Tree - Session"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
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
