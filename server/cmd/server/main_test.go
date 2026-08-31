package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestHealthHandler verifies the minimal process health contract.
func TestHealthHandler(t *testing.T) {
	recording := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	buildRouter().ServeHTTP(recording, request)
	if recording.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recording.Code, http.StatusOK)
	}
}
