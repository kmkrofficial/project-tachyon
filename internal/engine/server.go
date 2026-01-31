package engine

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"project-tachyon/internal/storage"
	"strings"
	"time"
)

type APIServer struct {
	logger  *slog.Logger
	engine  *TachyonEngine
	storage *storage.Storage
	server  *http.Server
	token   string
}

type DownloadRequest struct {
	URL       string `json:"url"`
	Cookies   string `json:"cookies"`
	UserAgent string `json:"userAgent"`
	Referer   string `json:"referer"`
}

// Storage keys for filtering
const (
	KeyDomainWhitelist = "settings_domain_whitelist"
	KeyDomainBlacklist = "settings_domain_blacklist"
	KeySilentMode      = "settings_silent_mode"
	KeyAPIToken        = "settings_api_token"
)

func NewAPIServer(logger *slog.Logger, engine *TachyonEngine, store *storage.Storage) *APIServer {
	// Load or generate API token
	token := getOrCreateToken(store, logger)

	return &APIServer{
		logger:  logger,
		engine:  engine,
		storage: store,
		token:   token,
	}
}

// getOrCreateToken loads existing token from storage or generates a new one
func getOrCreateToken(store *storage.Storage, logger *slog.Logger) string {
	token, err := store.GetString(KeyAPIToken)
	if err == nil && token != "" {
		logger.Info("Loaded existing API token")
		return token
	}

	// Generate new random token
	token = generateRandomToken()
	if err := store.SetString(KeyAPIToken, token); err != nil {
		logger.Error("Failed to save API token", "error", err)
	} else {
		logger.Info("Generated new API token")
	}
	return token
}

// generateRandomToken creates a secure random token
func generateRandomToken() string {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		// Fallback to uuid if crypto/rand fails
		return "tachyon-" + fmt.Sprintf("%x", time.Now().UnixNano())
	}
	return fmt.Sprintf("%x", b)
}

func (s *APIServer) Start(port int) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/download", s.handleDownload)

	addr := fmt.Sprintf(":%d", port)
	s.server = &http.Server{
		Addr:    addr,
		Handler: s.corsMiddleware(mux),
	}

	go func() {
		s.logger.Info("API Server starting", "addr", addr)
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("API Server failed", "error", err)
		}
	}()
}

func (s *APIServer) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.server.Shutdown(ctx)
}

func (s *APIServer) handleDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Auth Check
	token := r.Header.Get("X-Tachyon-Token")
	if token != s.token {
		if token == "" {
			s.logger.Warn("API Request missing token")
		} else if token != s.token {
			s.logger.Warn("API Request invalid token", "token", token)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
	}

	var req DownloadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}

	// Domain Filtering Logic
	filterResult := s.checkDomainFilters(req.Referer, req.URL)
	switch filterResult {
	case "blocked":
		s.logger.Info("Download blocked by domain filter", "url", req.URL, "referer", req.Referer)
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{"status": "blocked", "reason": "Domain is blacklisted"})
		return
	case "whitelisted":
		s.logger.Info("Download auto-approved by whitelist", "url", req.URL, "referer", req.Referer)
		// Continue to download
	case "silent":
		// Silent mode - don't auto-download unknown domains
		s.logger.Info("Download skipped in silent mode", "url", req.URL, "referer", req.Referer)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "silent", "reason": "Domain not whitelisted in silent mode"})
		return
	default:
		// "allowed" - neither blacklisted nor in whitelist, proceed
	}

	// Default path
	homeDir, _ := os.UserHomeDir()
	defaultPath := filepath.Join(homeDir, "Downloads")

	// Start Download
	id, err := s.engine.StartDownload(req.URL, defaultPath, "", nil)
	if err != nil {
		s.logger.Error("API failed to start download", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"id": id, "status": "started"})
}

// checkDomainFilters checks if the request should be blocked, allowed, or handled by silent mode
// Returns: "blocked", "whitelisted", "silent", or "allowed"
func (s *APIServer) checkDomainFilters(referer, downloadURL string) string {
	// Extract domain from referer or download URL
	domain := extractDomain(referer)
	if domain == "" {
		domain = extractDomain(downloadURL)
	}
	if domain == "" {
		return "allowed" // Can't determine domain, allow by default
	}

	// Check blacklist first
	blacklist, _ := s.storage.GetStringList(KeyDomainBlacklist)
	for _, blocked := range blacklist {
		if matchesDomain(domain, blocked) {
			return "blocked"
		}
	}

	// Check whitelist
	whitelist, _ := s.storage.GetStringList(KeyDomainWhitelist)
	for _, allowed := range whitelist {
		if matchesDomain(domain, allowed) {
			return "whitelisted"
		}
	}

	// Check silent mode
	silentMode, _ := s.storage.GetString(KeySilentMode)
	if silentMode == "true" {
		return "silent"
	}

	return "allowed"
}

// extractDomain extracts the hostname from a URL
func extractDomain(rawURL string) string {
	if rawURL == "" {
		return ""
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return strings.ToLower(parsed.Hostname())
}

// matchesDomain checks if domain matches a pattern (supports subdomain matching)
// e.g., "video.example.com" matches "example.com"
func matchesDomain(domain, pattern string) bool {
	pattern = strings.ToLower(pattern)
	domain = strings.ToLower(domain)

	if domain == pattern {
		return true
	}
	// Check if domain is a subdomain of pattern
	if strings.HasSuffix(domain, "."+pattern) {
		return true
	}
	return false
}

func (s *APIServer) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*") // Restrict in prod
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Tachyon-Token")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
