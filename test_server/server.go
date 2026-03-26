package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

// TestServer serves deterministic files of various types for E2E testing.
type TestServer struct {
	requestCount atomic.Int64
	bytesServed  atomic.Int64
	log          *slog.Logger
}

// NewTestServer creates a new test server instance.
func NewTestServer(logger *slog.Logger) *TestServer {
	return &TestServer{log: logger}
}

// Handler returns the HTTP handler for the test server.
func (ts *TestServer) Handler() http.Handler {
	mux := http.NewServeMux()

	// File endpoints — deterministic binary content
	mux.HandleFunc("/file/", ts.handleFile)      // /file/10mb — fast download
	mux.HandleFunc("/slow/", ts.handleSlow)      // /slow/5mb?speed=100kb — throttled
	mux.HandleFunc("/fail-at/", ts.handleFailAt) // /fail-at/10mb?at=4mb — breaks mid-stream
	mux.HandleFunc("/flaky/", ts.handleFlaky)    // /flaky/5mb?fail_rate=0.3 — random failures

	// Multimedia endpoints — valid file headers + padding
	mux.HandleFunc("/media/image/", ts.handleMediaImage)  // /media/image/photo.png?size=1mb
	mux.HandleFunc("/media/video/", ts.handleMediaVideo)  // /media/video/clip.mp4?size=5mb
	mux.HandleFunc("/media/audio/", ts.handleMediaAudio)  // /media/audio/track.mp3?size=2mb
	mux.HandleFunc("/media/document/", ts.handleMediaDoc) // /media/document/report.pdf?size=500kb
	mux.HandleFunc("/media/archive/", ts.handleMediaZip)  // /media/archive/data.zip?size=3mb

	// Control endpoints
	mux.HandleFunc("/auth/", ts.handleAuth)         // /auth/5mb — requires Basic auth
	mux.HandleFunc("/redirect/", ts.handleRedirect) // /redirect/3/5mb — N redirects then serve
	mux.HandleFunc("/stats", ts.handleStats)        // Server statistics

	return mux
}

// handleFile serves a deterministic binary file with range support.
func (ts *TestServer) handleFile(w http.ResponseWriter, r *http.Request) {
	size, err := parseSizeFromPath(r.URL.Path, "/file/")
	if err != nil {
		ts.log.Warn("bad request", "handler", "file", "path", r.URL.Path, "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	ts.log.Info("serving file", "handler", "file", "size", size, "range", r.Header.Get("Range"), "remote", r.RemoteAddr)
	ts.serveRangeContent(w, r, size, "application/octet-stream", "")
}

// handleSlow serves content at a limited speed.
func (ts *TestServer) handleSlow(w http.ResponseWriter, r *http.Request) {
	size, err := parseSizeFromPath(r.URL.Path, "/slow/")
	if err != nil {
		ts.log.Warn("bad request", "handler", "slow", "path", r.URL.Path, "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	speedStr := r.URL.Query().Get("speed")
	if speedStr == "" {
		speedStr = "100kb"
	}
	speed, err := parseSize(speedStr)
	if err != nil {
		ts.log.Warn("invalid speed param", "handler", "slow", "speed", speedStr, "error", err)
		http.Error(w, "invalid speed", http.StatusBadRequest)
		return
	}
	ts.log.Info("serving throttled", "handler", "slow", "size", size, "speed_bps", speed, "remote", r.RemoteAddr)
	ts.serveThrottled(w, size, speed)
}

// handleFailAt serves content but aborts after a certain number of bytes.
func (ts *TestServer) handleFailAt(w http.ResponseWriter, r *http.Request) {
	size, err := parseSizeFromPath(r.URL.Path, "/fail-at/")
	if err != nil {
		ts.log.Warn("bad request", "handler", "fail-at", "path", r.URL.Path, "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	atStr := r.URL.Query().Get("at")
	if atStr == "" {
		atStr = fmt.Sprintf("%d", size/2)
	}
	failAt, err := parseSize(atStr)
	if err != nil {
		ts.log.Warn("invalid at param", "handler", "fail-at", "at", atStr, "error", err)
		http.Error(w, "invalid at param", http.StatusBadRequest)
		return
	}

	ts.log.Info("serving fail-at", "handler", "fail-at", "size", size, "fail_at", failAt, "remote", r.RemoteAddr)
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", strconv.FormatInt(size, 10))
	w.Header().Set("Accept-Ranges", "bytes")

	written := writePatternData(w, min(failAt, size))
	ts.recordBytes(written)
	ts.log.Debug("fail-at complete", "handler", "fail-at", "written", written, "intended_fail", failAt)
}

// handleFlaky randomly fails some percentage of requests.
func (ts *TestServer) handleFlaky(w http.ResponseWriter, r *http.Request) {
	size, err := parseSizeFromPath(r.URL.Path, "/flaky/")
	if err != nil {
		ts.log.Warn("bad request", "handler", "flaky", "path", r.URL.Path, "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	rateStr := r.URL.Query().Get("fail_rate")
	rate := 0.3
	if rateStr != "" {
		if v, err := strconv.ParseFloat(rateStr, 64); err == nil {
			rate = v
		}
	}
	count := ts.requestCount.Add(1)
	if float64(count%10)/10.0 < rate {
		ts.log.Warn("flaky: simulated failure", "handler", "flaky", "count", count, "rate", rate, "remote", r.RemoteAddr)
		http.Error(w, "simulated server error", http.StatusInternalServerError)
		return
	}
	ts.log.Info("flaky: serving normally", "handler", "flaky", "size", size, "count", count, "remote", r.RemoteAddr)
	ts.serveRangeContent(w, r, size, "application/octet-stream", "")
}

// handleAuth requires Basic auth (user: test, pass: test).
func (ts *TestServer) handleAuth(w http.ResponseWriter, r *http.Request) {
	user, pass, ok := r.BasicAuth()
	if !ok || user != "test" || pass != "test" {
		ts.log.Warn("auth rejected", "handler", "auth", "user", user, "has_creds", ok, "remote", r.RemoteAddr)
		w.Header().Set("WWW-Authenticate", `Basic realm="test"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	size, err := parseSizeFromPath(r.URL.Path, "/auth/")
	if err != nil {
		ts.log.Warn("bad request", "handler", "auth", "path", r.URL.Path, "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	ts.log.Info("auth accepted", "handler", "auth", "user", user, "size", size, "remote", r.RemoteAddr)
	ts.serveRangeContent(w, r, size, "application/octet-stream", "")
}

// handleRedirect redirects N times then serves the file.
func (ts *TestServer) handleRedirect(w http.ResponseWriter, r *http.Request) {
	trimmed := strings.TrimPrefix(r.URL.Path, "/redirect/")
	parts := strings.SplitN(trimmed, "/", 2)
	if len(parts) != 2 {
		ts.log.Warn("bad redirect path", "handler", "redirect", "path", r.URL.Path)
		http.Error(w, "expected /redirect/{n}/{size}", http.StatusBadRequest)
		return
	}
	n, err := strconv.Atoi(parts[0])
	if err != nil || n < 0 {
		ts.log.Warn("invalid redirect count", "handler", "redirect", "n", parts[0])
		http.Error(w, "invalid redirect count", http.StatusBadRequest)
		return
	}
	if n > 0 {
		next := fmt.Sprintf("/redirect/%d/%s", n-1, parts[1])
		ts.log.Debug("redirecting", "handler", "redirect", "remaining", n, "next", next, "remote", r.RemoteAddr)
		http.Redirect(w, r, next, http.StatusFound)
		return
	}
	size, err := parseSize(parts[1])
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	ts.log.Info("redirect complete, serving", "handler", "redirect", "size", size, "remote", r.RemoteAddr)
	ts.serveRangeContent(w, r, size, "application/octet-stream", "")
}

// handleStats returns server statistics.
func (ts *TestServer) handleStats(w http.ResponseWriter, _ *http.Request) {
	reqs := ts.requestCount.Load()
	bytes := ts.bytesServed.Load()
	ts.log.Debug("stats requested", "requests", reqs, "bytes_served", bytes)
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"requests":%d,"bytes_served":%d,"uptime":"%s"}`,
		reqs, bytes, time.Since(time.Now()).String())
}

// recordBytes records bytes served for stats.
func (ts *TestServer) recordBytes(n int64) {
	ts.requestCount.Add(1)
	ts.bytesServed.Add(n)
}
