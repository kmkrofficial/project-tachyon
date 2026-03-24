package engine

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"project-tachyon/internal/storage"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// newTestAPIServer creates an APIServer with an in-memory DB for testing.
func newTestAPIServer(t *testing.T) (*APIServer, *storage.Storage) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	db.AutoMigrate(&storage.DownloadTask{}, &storage.DownloadLocation{}, &storage.DailyStat{}, &storage.AppSetting{})
	store := &storage.Storage{DB: db}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	engine := NewEngine(logger, store)

	srv := NewAPIServer(logger, engine, store)
	return srv, store
}

// --- extractDomain ---

func TestExtractDomain(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"https://example.com/path", "example.com"},
		{"http://sub.example.com:8080/file", "sub.example.com"},
		{"https://EXAMPLE.COM/", "example.com"},
		{"", ""},
		{"not-a-url", ""},
		{"://invalid", ""},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := extractDomain(tt.input)
			if got != tt.want {
				t.Errorf("extractDomain(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// --- matchesDomain ---

func TestMatchesDomain(t *testing.T) {
	tests := []struct {
		domain  string
		pattern string
		want    bool
	}{
		{"example.com", "example.com", true},
		{"EXAMPLE.COM", "example.com", true},
		{"sub.example.com", "example.com", true},
		{"deep.sub.example.com", "example.com", true},
		{"notexample.com", "example.com", false},
		{"example.com", "sub.example.com", false},
		{"other.com", "example.com", false},
	}
	for _, tt := range tests {
		t.Run(tt.domain+"_vs_"+tt.pattern, func(t *testing.T) {
			got := matchesDomain(tt.domain, tt.pattern)
			if got != tt.want {
				t.Errorf("matchesDomain(%q, %q) = %v, want %v", tt.domain, tt.pattern, got, tt.want)
			}
		})
	}
}

// --- generateRandomToken ---

func TestGenerateRandomToken(t *testing.T) {
	t1 := generateRandomToken()
	t2 := generateRandomToken()

	if len(t1) < 16 {
		t.Errorf("token too short: %s", t1)
	}
	if t1 == t2 {
		t.Error("two generated tokens should not be equal")
	}
}

// --- checkDomainFilters ---

func TestCheckDomainFilters_Blacklisted(t *testing.T) {
	srv, store := newTestAPIServer(t)
	store.SetStringList(KeyDomainBlacklist, []string{"ads.com", "spam.net"})

	result := srv.checkDomainFilters("https://ads.com/page", "")
	if result != "blocked" {
		t.Errorf("expected blocked, got %s", result)
	}
}

func TestCheckDomainFilters_SubdomainBlacklisted(t *testing.T) {
	srv, store := newTestAPIServer(t)
	store.SetStringList(KeyDomainBlacklist, []string{"ads.com"})

	result := srv.checkDomainFilters("https://tracker.ads.com/page", "")
	if result != "blocked" {
		t.Errorf("expected subdomain blocked, got %s", result)
	}
}

func TestCheckDomainFilters_Whitelisted(t *testing.T) {
	srv, store := newTestAPIServer(t)
	store.SetStringList(KeyDomainWhitelist, []string{"trusted.com"})

	result := srv.checkDomainFilters("https://trusted.com/file.zip", "")
	if result != "whitelisted" {
		t.Errorf("expected whitelisted, got %s", result)
	}
}

func TestCheckDomainFilters_SilentMode(t *testing.T) {
	srv, store := newTestAPIServer(t)
	store.SetString(KeySilentMode, "true")

	result := srv.checkDomainFilters("https://unknown-site.com/file", "")
	if result != "silent" {
		t.Errorf("expected silent, got %s", result)
	}
}

func TestCheckDomainFilters_AllowedDefault(t *testing.T) {
	srv, _ := newTestAPIServer(t)
	result := srv.checkDomainFilters("https://normal-site.com/file", "")
	if result != "allowed" {
		t.Errorf("expected allowed, got %s", result)
	}
}

func TestCheckDomainFilters_EmptyDomain(t *testing.T) {
	srv, _ := newTestAPIServer(t)
	result := srv.checkDomainFilters("", "")
	if result != "allowed" {
		t.Errorf("expected allowed for empty domain, got %s", result)
	}
}

func TestCheckDomainFilters_BlacklistPrecedence(t *testing.T) {
	srv, store := newTestAPIServer(t)
	// Both blacklisted and whitelisted — blacklist wins
	store.SetStringList(KeyDomainBlacklist, []string{"dual.com"})
	store.SetStringList(KeyDomainWhitelist, []string{"dual.com"})

	result := srv.checkDomainFilters("https://dual.com/file", "")
	if result != "blocked" {
		t.Errorf("blacklist should take precedence, got %s", result)
	}
}

// --- handleDownload auth ---

func TestHandleDownload_WrongMethod(t *testing.T) {
	srv, _ := newTestAPIServer(t)
	req := httptest.NewRequest("GET", "/api/v1/download", nil)
	rec := httptest.NewRecorder()
	srv.handleDownload(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rec.Code)
	}
}

func TestHandleDownload_InvalidToken(t *testing.T) {
	srv, _ := newTestAPIServer(t)
	body := `{"url":"https://example.com/file.zip"}`
	req := httptest.NewRequest("POST", "/api/v1/download", strings.NewReader(body))
	req.Header.Set("X-Tachyon-Token", "wrong-token")
	rec := httptest.NewRecorder()
	srv.handleDownload(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestHandleDownload_ValidToken(t *testing.T) {
	srv, _ := newTestAPIServer(t)
	body := `{"url":"https://example.com/file.zip"}`
	req := httptest.NewRequest("POST", "/api/v1/download", strings.NewReader(body))
	req.Header.Set("X-Tachyon-Token", srv.token)
	rec := httptest.NewRecorder()
	srv.handleDownload(rec, req)

	if rec.Code == http.StatusUnauthorized {
		t.Error("valid token should not return 401")
	}
}

func TestHandleDownload_InvalidBody(t *testing.T) {
	srv, _ := newTestAPIServer(t)
	req := httptest.NewRequest("POST", "/api/v1/download", strings.NewReader("not json"))
	req.Header.Set("X-Tachyon-Token", srv.token)
	rec := httptest.NewRecorder()
	srv.handleDownload(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestHandleDownload_BlacklistedDomain(t *testing.T) {
	srv, store := newTestAPIServer(t)
	store.SetStringList(KeyDomainBlacklist, []string{"evil.com"})

	body := `{"url":"https://evil.com/malware.exe"}`
	req := httptest.NewRequest("POST", "/api/v1/download", strings.NewReader(body))
	req.Header.Set("X-Tachyon-Token", srv.token)
	rec := httptest.NewRecorder()
	srv.handleDownload(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for blacklisted domain, got %d", rec.Code)
	}
}

func TestHandleDownload_SilentMode(t *testing.T) {
	srv, store := newTestAPIServer(t)
	store.SetString(KeySilentMode, "true")

	body := `{"url":"https://unknown-site.com/file.zip"}`
	req := httptest.NewRequest("POST", "/api/v1/download", strings.NewReader(body))
	req.Header.Set("X-Tachyon-Token", srv.token)
	rec := httptest.NewRecorder()
	srv.handleDownload(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for silent mode, got %d", rec.Code)
	}
	var resp map[string]string
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["status"] != "silent" {
		t.Errorf("expected status silent, got %s", resp["status"])
	}
}

// --- CORS middleware ---

func TestCorsMiddleware_Options(t *testing.T) {
	srv, _ := newTestAPIServer(t)
	handler := srv.corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("OPTIONS", "/api/v1/download", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("OPTIONS should return 200, got %d", rec.Code)
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("missing CORS Allow-Origin header")
	}
}

// --- getOrCreateToken ---

func TestGetOrCreateToken_Persistence(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&storage.AppSetting{})
	store := &storage.Storage{DB: db}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	token1 := getOrCreateToken(store, logger)
	token2 := getOrCreateToken(store, logger)

	if token1 != token2 {
		t.Error("token should be stable across calls when stored")
	}
	if len(token1) < 16 {
		t.Error("token too short")
	}
}
