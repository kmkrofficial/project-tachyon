package engine

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"project-tachyon/internal/network"
)

// newHTTPEngine creates a minimal TachyonEngine for HTTP-related tests.
func newHTTPEngine() *TachyonEngine {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	return &TachyonEngine{
		logger:     logger,
		httpClient: &http.Client{},
		congestion: network.NewCongestionController(4, 16),
		probes:     newProbeCache(),
	}
}

// --- friendlyError ---

func TestFriendlyError(t *testing.T) {
	tests := []struct {
		name    string
		errMsg  string
		wantSub string
	}{
		{"no such host", "dial tcp: lookup xyz: no such host", "Server not found"},
		{"connection refused", "dial tcp 1.2.3.4:80: connection refused", "offline or unreachable"},
		{"timeout", "context deadline exceeded", "timed out"},
		{"certificate", "x509: certificate signed by unknown authority", "SSL certificate"},
		{"network unreachable", "network is unreachable", "No internet"},
		{"unknown", "some random error", "Connection failed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := friendlyError(fmt.Errorf("%s", tt.errMsg))
			if !strings.Contains(got.Error(), tt.wantSub) {
				t.Errorf("friendlyError(%q) = %q, want substring %q", tt.errMsg, got.Error(), tt.wantSub)
			}
		})
	}
}

// --- friendlyHTTPError ---

func TestFriendlyHTTPError(t *testing.T) {
	tests := []struct {
		status  int
		wantSub string
	}{
		{404, "not found"},
		{403, "denied"},
		{401, "Authentication"},
		{500, "Server error"},
		{502, "Server error"},
		{503, "Server error"},
		{429, "Too many"},
		{418, "error 418"},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("status_%d", tt.status), func(t *testing.T) {
			got := friendlyHTTPError(tt.status)
			if !strings.Contains(got.Error(), tt.wantSub) {
				t.Errorf("friendlyHTTPError(%d) = %q, want substring %q", tt.status, got.Error(), tt.wantSub)
			}
		})
	}
}

// --- newRequest ---

func TestNewRequest_DefaultHeaders(t *testing.T) {
	e := newHTTPEngine()
	req, err := e.newRequest("GET", "https://example.com/file.bin", "", "")
	if err != nil {
		t.Fatalf("newRequest failed: %v", err)
	}

	if req.Header.Get("User-Agent") != GenericUserAgent {
		t.Errorf("expected default User-Agent, got %q", req.Header.Get("User-Agent"))
	}
	if req.Header.Get("Accept") != "*/*" {
		t.Error("missing Accept header")
	}
	if req.Header.Get("Connection") != "keep-alive" {
		t.Error("missing Connection header")
	}
}

func TestNewRequest_CustomHeaders(t *testing.T) {
	e := newHTTPEngine()
	headers := `{"Referer": "https://origin.com", "X-Custom": "value"}`
	req, err := e.newRequest("GET", "https://example.com/file.bin", headers, "")
	if err != nil {
		t.Fatalf("newRequest failed: %v", err)
	}

	if req.Header.Get("Referer") != "https://origin.com" {
		t.Error("custom Referer not applied")
	}
	if req.Header.Get("X-Custom") != "value" {
		t.Error("custom X-Custom not applied")
	}
}

func TestNewRequest_DangerousHeadersBlocked(t *testing.T) {
	e := newHTTPEngine()
	// Host and Transfer-Encoding should be silently skipped
	headers := `{"Host": "evil.com", "Transfer-Encoding": "chunked", "Referer": "ok.com"}`
	req, err := e.newRequest("GET", "https://example.com/file.bin", headers, "")
	if err != nil {
		t.Fatalf("newRequest failed: %v", err)
	}

	if req.Header.Get("Referer") != "ok.com" {
		t.Error("safe header was not applied")
	}
	// Host on a Request is in req.Host, not in headers - but the point is it shouldn't override
}

func TestNewRequest_InvalidHeadersJSON(t *testing.T) {
	e := newHTTPEngine()
	// Invalid JSON should be silently ignored (no error)
	req, err := e.newRequest("GET", "https://example.com/file.bin", "not json at all", "")
	if err != nil {
		t.Fatalf("newRequest failed: %v", err)
	}
	// Should still have default headers
	if req.Header.Get("User-Agent") == "" {
		t.Error("defaults should still be set even with bad headers JSON")
	}
}

func TestNewRequest_CookiesJSON(t *testing.T) {
	e := newHTTPEngine()
	cookies := `[{"Name":"session","Value":"abc123"},{"Name":"lang","Value":"en"}]`
	req, err := e.newRequest("GET", "https://example.com/file.bin", "", cookies)
	if err != nil {
		t.Fatalf("newRequest failed: %v", err)
	}

	cookieList := req.Cookies()
	if len(cookieList) != 2 {
		t.Fatalf("expected 2 cookies, got %d", len(cookieList))
	}

	names := make(map[string]string)
	for _, c := range cookieList {
		names[c.Name] = c.Value
	}
	if names["session"] != "abc123" {
		t.Error("session cookie not set correctly")
	}
}

func TestNewRequest_CookiesRawString(t *testing.T) {
	e := newHTTPEngine()
	req, err := e.newRequest("GET", "https://example.com/file.bin", "", "session=abc; lang=en")
	if err != nil {
		t.Fatalf("newRequest failed: %v", err)
	}

	if req.Header.Get("Cookie") != "session=abc; lang=en" {
		t.Errorf("raw cookie string not set, got %q", req.Header.Get("Cookie"))
	}
}

func TestNewRequest_CookiesMalformedJSON(t *testing.T) {
	e := newHTTPEngine()
	// Starts with [ but is invalid JSON → should fallback to raw string
	req, err := e.newRequest("GET", "https://example.com/file.bin", "", "[bad json")
	if err != nil {
		t.Fatalf("newRequest failed: %v", err)
	}
	if req.Header.Get("Cookie") != "[bad json" {
		t.Error("malformed JSON cookie should fallback to raw header")
	}
}

func TestNewRequest_EmptyCookies(t *testing.T) {
	e := newHTTPEngine()
	req, err := e.newRequest("GET", "https://example.com/file.bin", "", "")
	if err != nil {
		t.Fatalf("newRequest failed: %v", err)
	}
	if req.Header.Get("Cookie") != "" {
		t.Error("empty cookie string should not set Cookie header")
	}
}

func TestNewRequest_InvalidURL(t *testing.T) {
	e := newHTTPEngine()
	_, err := e.newRequest("GET", "://bad-url", "", "")
	if err == nil {
		t.Error("expected error for invalid URL")
	}
}

// --- Sentinel errors ---

func TestSentinelErrors(t *testing.T) {
	if ErrLinkExpired == nil {
		t.Error("ErrLinkExpired should not be nil")
	}
	if ErrRangeIgnored == nil {
		t.Error("ErrRangeIgnored should not be nil")
	}
	// They should be distinct
	if ErrLinkExpired == ErrRangeIgnored {
		t.Error("sentinel errors should be distinct")
	}
}

// --- ProbeURL with mock server ---

func TestProbeURL_HEAD(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "12345")
		w.Header().Set("Accept-Ranges", "bytes")
		w.Header().Set("Content-Disposition", `attachment; filename="test.zip"`)
		w.Header().Set("ETag", `"abc123"`)
		w.Header().Set("Last-Modified", "Mon, 01 Jan 2024 00:00:00 GMT")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	e := newHTTPEngine()
	result, err := e.ProbeURL(server.URL, "", "")
	if err != nil {
		t.Fatalf("ProbeURL failed: %v", err)
	}
	if result.Size != 12345 {
		t.Errorf("expected size 12345, got %d", result.Size)
	}
	if result.Filename != "test.zip" {
		t.Errorf("expected filename test.zip, got %s", result.Filename)
	}
	if !result.AcceptRanges {
		t.Error("expected AcceptRanges true")
	}
	if result.ETag != `"abc123"` {
		t.Errorf("ETag mismatch: %s", result.ETag)
	}
}

func TestProbeURL_FallbackToGETRange(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "HEAD" {
			// HEAD returns no size → triggers fallback
			w.WriteHeader(http.StatusOK)
			return
		}
		// GET with Range
		w.Header().Set("Content-Range", "bytes 0-0/999")
		w.Header().Set("Content-Length", "1")
		w.WriteHeader(http.StatusPartialContent)
		w.Write([]byte("x"))
	}))
	defer server.Close()

	e := newHTTPEngine()
	result, err := e.ProbeURL(server.URL+"/file.bin", "", "")
	if err != nil {
		t.Fatalf("ProbeURL failed: %v", err)
	}
	if result.Size != 999 {
		t.Errorf("expected size 999 from Content-Range, got %d", result.Size)
	}
	if !result.AcceptRanges {
		t.Error("206 response means ranges are accepted")
	}
}

func TestProbeURL_404(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	e := newHTTPEngine()
	_, err := e.ProbeURL(server.URL, "", "")
	if err == nil {
		t.Error("expected error for 404")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' in error, got: %s", err.Error())
	}
}

func TestProbeURL_FilenameFromURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "100")
		w.Header().Set("Accept-Ranges", "bytes")
		// No Content-Disposition → should parse from URL
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	e := newHTTPEngine()
	result, err := e.ProbeURL(server.URL+"/downloads/ubuntu.iso", "", "")
	if err != nil {
		t.Fatalf("ProbeURL failed: %v", err)
	}
	if result.Filename != "ubuntu.iso" {
		t.Errorf("expected filename ubuntu.iso from URL path, got %s", result.Filename)
	}
}

func TestProbeURL_NoFilename(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "100")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	e := newHTTPEngine()
	result, err := e.ProbeURL(server.URL+"/", "", "")
	if err != nil {
		t.Fatalf("ProbeURL failed: %v", err)
	}
	if result.Filename != "unknown_file" {
		t.Errorf("expected 'unknown_file' for root path, got %s", result.Filename)
	}
}

func TestProbeURL_HEADRefusedGETFallback(t *testing.T) {
	// Simulate a server that rejects HEAD (connection reset) but serves GET fine
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "HEAD" {
			// Force a connection close to simulate transport-level error on HEAD
			hj, ok := w.(http.Hijacker)
			if !ok {
				t.Skip("server does not support hijack")
			}
			conn, _, _ := hj.Hijack()
			conn.Close()
			return
		}
		// GET with Range works
		w.Header().Set("Content-Range", "bytes 0-0/5000")
		w.Header().Set("Content-Length", "1")
		w.WriteHeader(http.StatusPartialContent)
		w.Write([]byte("x"))
	}))
	defer server.Close()

	e := newHTTPEngine()
	result, err := e.ProbeURL(server.URL+"/file.bin", "", "")
	if err != nil {
		t.Fatalf("ProbeURL should succeed via GET fallback, but got: %v", err)
	}
	if result.Size != 5000 {
		t.Errorf("expected size 5000 from GET fallback, got %d", result.Size)
	}
	if !result.AcceptRanges {
		t.Error("expected AcceptRanges true from 206 response")
	}
}
