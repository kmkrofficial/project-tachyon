package api

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"project-tachyon/internal/security"
)

// Ensure imports are used
var _ = time.Now

func newTestAudit(t *testing.T) *security.AuditLogger {
	t.Helper()
	return security.NewAuditLogger(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})))
}

func newTestControlServer(t *testing.T) *ControlServer {
	t.Helper()
	return &ControlServer{
		rateHits: make(map[string][]time.Time),
		audit:    newTestAudit(t),
	}
}

func TestParseCookieString_Basic(t *testing.T) {
	cookies := ParseCookieString("session=abc123; lang=en")
	if len(cookies) != 2 {
		t.Fatalf("expected 2 cookies, got %d", len(cookies))
	}

	names := map[string]string{}
	for _, c := range cookies {
		names[c.Name] = c.Value
	}
	if names["session"] != "abc123" {
		t.Errorf("expected session=abc123, got %s", names["session"])
	}
	if names["lang"] != "en" {
		t.Errorf("expected lang=en, got %s", names["lang"])
	}
}

func TestParseCookieString_Empty(t *testing.T) {
	cookies := ParseCookieString("")
	if len(cookies) != 0 {
		t.Fatalf("expected 0 cookies for empty string, got %d", len(cookies))
	}
}

func TestParseCookieString_Single(t *testing.T) {
	cookies := ParseCookieString("token=xyz")
	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(cookies))
	}
	if cookies[0].Name != "token" || cookies[0].Value != "xyz" {
		t.Errorf("unexpected cookie: %+v", cookies[0])
	}
}

func TestRateLimitMiddleware_AllowsUnderLimit(t *testing.T) {
	s := newTestControlServer(t)

	handler := s.rateLimitMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Fire requests under the limit
	for i := 0; i < rateLimit; i++ {
		req := httptest.NewRequest("GET", "/v1/status", nil)
		req.RemoteAddr = "127.0.0.1:12345"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("request %d: expected 200, got %d", i, rec.Code)
		}
	}
}

func TestRateLimitMiddleware_BlocksOverLimit(t *testing.T) {
	s := newTestControlServer(t)

	handler := s.rateLimitMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Fill up the rate limit
	for i := 0; i < rateLimit; i++ {
		req := httptest.NewRequest("GET", "/v1/status", nil)
		req.RemoteAddr = "127.0.0.1:12345"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}

	// Next request should be blocked
	req := httptest.NewRequest("GET", "/v1/status", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", rec.Code)
	}
}

func TestRateLimitMiddleware_DifferentIPsIndependent(t *testing.T) {
	s := newTestControlServer(t)

	handler := s.rateLimitMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Fill up rate limit for IP1
	for i := 0; i < rateLimit; i++ {
		req := httptest.NewRequest("GET", "/v1/status", nil)
		req.RemoteAddr = "127.0.0.1:12345"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}

	// IP2 should still be allowed
	req := httptest.NewRequest("GET", "/v1/status", nil)
	req.RemoteAddr = "192.168.1.1:54321"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for different IP, got %d", rec.Code)
	}
}
