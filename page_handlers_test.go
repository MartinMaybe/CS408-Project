package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLandingHandlerRendersLandingPage(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	landingHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status code %d, got %d", http.StatusOK, w.Code)
	}
	if !strings.Contains(w.Body.String(), "Welcome to the Public Decision Tree") {
		t.Fatalf("expected landing page to include welcome heading")
	}
}

func TestSessionHandlerRendersSessionShell(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/session", nil)
	w := httptest.NewRecorder()

	sessionHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status code %d, got %d", http.StatusOK, w.Code)
	}
	if !strings.Contains(w.Body.String(), "Loading session...") {
		t.Fatalf("expected session page to include loading placeholder")
	}
}

func TestStatisticsHandlerRendersStatisticsPage(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/statistics", nil)
	w := httptest.NewRecorder()

	statisticsHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status code %d, got %d", http.StatusOK, w.Code)
	}
	if !strings.Contains(w.Body.String(), "Global Statistics") {
		t.Fatalf("expected statistics page to include statistics heading")
	}
}
