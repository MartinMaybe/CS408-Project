package main

import "net/http"

func registerRoutes() {
	http.HandleFunc("/", landingHandler)
	http.HandleFunc("/session", sessionHandler)
	http.HandleFunc("/session/choose", chooseHandler)
	http.HandleFunc("/statistics", statisticsHandler)
	http.HandleFunc("/api/start", startSessionHandler)
	http.HandleFunc("/session/results", sessionResultsHandler)
	http.HandleFunc("/api/session/current", apiCurrentHandler)
	http.HandleFunc("/api/session/choose", apiChooseHandler)
}
