package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLandingHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	landingHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}
}

func TestSessionHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/session", nil)
	w := httptest.NewRecorder()

	sessionHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}
}

func TestStatisticsHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/statistics", nil)
	w := httptest.NewRecorder()

	statisticsHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}
}
