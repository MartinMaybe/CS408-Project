package main

import "net/http"

func registerRoutes() {
	http.HandleFunc("/", landingHandler)
	http.HandleFunc("/session", sessionHandler)
	http.HandleFunc("/session/choose", chooseHandler)
	http.HandleFunc("/statistics", statisticsHandler)
	http.HandleFunc("/session/results", sessionResultsHandler)
	http.HandleFunc("/api/sessions", sessionsAPIHandler)
	http.HandleFunc("/api/session", sessionAPIHandler)
	http.HandleFunc("/api/port", portAPIHandler)
	http.HandleFunc("/api/node", nodeAPIHandler)
}
