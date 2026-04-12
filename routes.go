package main

import "net/http"

func registerRoutes() {
	http.HandleFunc("/", landingHandler)
	http.HandleFunc("/session", sessionHandler)
	http.HandleFunc("/statistics", statisticsHandler)
	http.HandleFunc("/api/sessions", sessionsAPIHandler)
	http.HandleFunc("/api/session", sessionAPIHandler)
	http.HandleFunc("/api/port", portAPIHandler)
	http.HandleFunc("/api/node", nodeAPIHandler)
}
