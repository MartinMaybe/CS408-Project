// This file defines application route registration.
//
// This file maps HTTP paths to their handlers:
//   - Static asset serving
//   - HTML page routes (landing, session, statistics)
//   - API endpoints for sessions, nodes, ports, and history
package main

import "net/http"

func registerRoutes() {
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	http.HandleFunc("/", landingHandler)
	http.HandleFunc("/session", sessionHandler)
	http.HandleFunc("/statistics", statisticsHandler)
	http.HandleFunc("/api/sessions", sessionsAPIHandler)
	http.HandleFunc("/api/session", sessionAPIHandler)
	http.HandleFunc("/api/session/history", sessionHistoryAPIHandler)
	http.HandleFunc("/api/port", portAPIHandler)
	http.HandleFunc("/api/node", nodeAPIHandler)
}
