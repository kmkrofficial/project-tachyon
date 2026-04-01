package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleHealth_ReturnsOK(t *testing.T) {
	s := newTestControlServer(t)

	req := httptest.NewRequest(http.MethodGet, "/v1/health", nil)
	rec := httptest.NewRecorder()

	s.handleHealth(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}

	if body["status"] != "ok" {
		t.Errorf("status = %q, want %q", body["status"], "ok")
	}
	if body["version"] == "" {
		t.Error("version should not be empty")
	}
}

func TestHandleHealth_CORS(t *testing.T) {
	s := newTestControlServer(t)

	req := httptest.NewRequest(http.MethodOptions, "/v1/health", nil)
	rec := httptest.NewRecorder()

	s.handleHealth(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("CORS preflight status = %d, want 200", rec.Code)
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("missing CORS Allow-Origin header")
	}
	if rec.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Error("missing CORS Allow-Methods header")
	}
}

func TestHandleHealth_NoAuthRequired(t *testing.T) {
	// The health endpoint should respond without any X-Tachyon-Token header.
	s := newTestControlServer(t)

	req := httptest.NewRequest(http.MethodGet, "/v1/health", nil)
	// Explicitly no auth header
	rec := httptest.NewRecorder()

	s.handleHealth(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("health should not require auth, got status %d", rec.Code)
	}
}
