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
	setupTestAppDB(t)
	createSessionWithChoices(t, "yes", "no")

	req := httptest.NewRequest(http.MethodGet, "/statistics", nil)
	w := httptest.NewRecorder()

	statisticsHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status code %d, got %d", http.StatusOK, w.Code)
	}
	if !strings.Contains(w.Body.String(), "Global Statistics") {
		t.Fatalf("expected statistics page to include statistics heading")
	}
	if !strings.Contains(w.Body.String(), "Total decisions") {
		t.Fatalf("expected statistics page to include total decisions")
	}
	if !strings.Contains(w.Body.String(), ">2<") {
		t.Fatalf("expected statistics page to render real decision count")
	}
	if strings.Contains(w.Body.String(), "1,284") || strings.Contains(w.Body.String(), "Do you like movies?") {
		t.Fatalf("expected statistics page to replace placeholder content")
	}
}
