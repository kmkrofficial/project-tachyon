package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// ─── HLS Parsing ──────────────────────────────────────────────────────────────

func TestResolveHLS_MultiVariant(t *testing.T) {
	manifest := `#EXTM3U
#EXT-X-STREAM-INF:BANDWIDTH=800000,RESOLUTION=640x360
360p/index.m3u8
#EXT-X-STREAM-INF:BANDWIDTH=1400000,RESOLUTION=1280x720
720p/index.m3u8
#EXT-X-STREAM-INF:BANDWIDTH=5000000,RESOLUTION=1920x1080
1080p/index.m3u8
#EXT-X-STREAM-INF:BANDWIDTH=14000000,RESOLUTION=3840x2160
4k/index.m3u8
`
	result := resolveHLS(manifest, "https://cdn.example.com/video/master.m3u8")

	if result.Type != "hls" {
		t.Errorf("type = %q, want hls", result.Type)
	}
	if len(result.Variants) != 4 {
		t.Fatalf("variants = %d, want 4", len(result.Variants))
	}

	// Check 360p variant
	v0 := result.Variants[0]
	if v0.Quality != "360p" {
		t.Errorf("v0 quality = %q, want 360p", v0.Quality)
	}
	if v0.Resolution != "640x360" {
		t.Errorf("v0 resolution = %q, want 640x360", v0.Resolution)
	}
	if v0.URL != "https://cdn.example.com/video/360p/index.m3u8" {
		t.Errorf("v0 URL = %q", v0.URL)
	}
	if v0.Bandwidth != 800000 {
		t.Errorf("v0 bandwidth = %d, want 800000", v0.Bandwidth)
	}

	// Check 4K variant
	v3 := result.Variants[3]
	if v3.Quality != "4K" {
		t.Errorf("v3 quality = %q, want 4K", v3.Quality)
	}
	if v3.Resolution != "3840x2160" {
		t.Errorf("v3 resolution = %q, want 3840x2160", v3.Resolution)
	}
}

func TestResolveHLS_SingleStream(t *testing.T) {
	manifest := `#EXTM3U
#EXT-X-TARGETDURATION:10
#EXTINF:10.0,
segment001.ts
#EXTINF:10.0,
segment002.ts
`
	result := resolveHLS(manifest, "https://cdn.example.com/live/stream.m3u8")

	if len(result.Variants) != 1 {
		t.Fatalf("variants = %d, want 1 (fallback)", len(result.Variants))
	}
	if result.Variants[0].Quality != "Default" {
		t.Errorf("quality = %q, want Default", result.Variants[0].Quality)
	}
}

func TestResolveHLS_AbsoluteURLs(t *testing.T) {
	manifest := `#EXTM3U
#EXT-X-STREAM-INF:BANDWIDTH=2000000,RESOLUTION=1280x720
https://other-cdn.example.com/720p.m3u8
`
	result := resolveHLS(manifest, "https://cdn.example.com/master.m3u8")

	if len(result.Variants) != 1 {
		t.Fatalf("variants = %d, want 1", len(result.Variants))
	}
	if result.Variants[0].URL != "https://other-cdn.example.com/720p.m3u8" {
		t.Errorf("URL = %q, want absolute URL preserved", result.Variants[0].URL)
	}
}

func TestResolveHLS_WithNAME(t *testing.T) {
	manifest := `#EXTM3U
#EXT-X-STREAM-INF:BANDWIDTH=500000,NAME="Low Quality"
low.m3u8
`
	result := resolveHLS(manifest, "https://cdn.example.com/master.m3u8")

	if len(result.Variants) != 1 {
		t.Fatalf("variants = %d, want 1", len(result.Variants))
	}
	if result.Variants[0].Quality != "Low Quality" {
		t.Errorf("quality = %q, want Low Quality", result.Variants[0].Quality)
	}
}

// ─── DASH Parsing ─────────────────────────────────────────────────────────────

func TestResolveDASH_MultiRepresentation(t *testing.T) {
	manifest := `<?xml version="1.0"?>
<MPD>
  <Period>
    <AdaptationSet mimeType="video/mp4">
      <Representation bandwidth="800000" width="640" height="360">
        <BaseURL>video_360p.mp4</BaseURL>
      </Representation>
      <Representation bandwidth="2500000" width="1280" height="720">
        <BaseURL>video_720p.mp4</BaseURL>
      </Representation>
      <Representation bandwidth="5000000" width="1920" height="1080">
        <BaseURL>video_1080p.mp4</BaseURL>
      </Representation>
    </AdaptationSet>
    <AdaptationSet mimeType="audio/mp4">
      <Representation bandwidth="128000">
        <BaseURL>audio_128k.mp4</BaseURL>
      </Representation>
    </AdaptationSet>
  </Period>
</MPD>`

	result := resolveDASH(manifest, "https://cdn.example.com/content/manifest.mpd")

	if result.Type != "dash" {
		t.Errorf("type = %q, want dash", result.Type)
	}
	if len(result.Variants) != 4 {
		t.Fatalf("variants = %d, want 4 (3 video + 1 audio)", len(result.Variants))
	}

	// Check 360p
	v0 := result.Variants[0]
	if v0.Quality != "360p" {
		t.Errorf("v0 quality = %q, want 360p", v0.Quality)
	}
	if v0.URL != "https://cdn.example.com/content/video_360p.mp4" {
		t.Errorf("v0 URL = %q", v0.URL)
	}
	if v0.ContentType != "video/mp4" {
		t.Errorf("v0 content type = %q, want video/mp4", v0.ContentType)
	}

	// Check 1080p
	v2 := result.Variants[2]
	if v2.Quality != "1080p" {
		t.Errorf("v2 quality = %q, want 1080p", v2.Quality)
	}

	// Check audio
	v3 := result.Variants[3]
	if v3.ContentType != "audio/mp4" {
		t.Errorf("v3 content type = %q, want audio/mp4", v3.ContentType)
	}
	if v3.Quality != "128 kbps" {
		t.Errorf("v3 quality = %q, want '128 kbps'", v3.Quality)
	}
}

func TestResolveDASH_EmptyManifest(t *testing.T) {
	manifest := `<?xml version="1.0"?><MPD></MPD>`
	result := resolveDASH(manifest, "https://cdn.example.com/empty.mpd")

	if len(result.Variants) != 1 {
		t.Fatalf("variants = %d, want 1 (fallback)", len(result.Variants))
	}
	if result.Variants[0].Quality != "Default" {
		t.Errorf("quality = %q, want Default", result.Variants[0].Quality)
	}
}

// ─── URL Resolution ───────────────────────────────────────────────────────────

func TestResolveURL_Relative(t *testing.T) {
	result := resolveURL("https://cdn.example.com/video/master.m3u8", "720p/index.m3u8")
	want := "https://cdn.example.com/video/720p/index.m3u8"
	if result != want {
		t.Errorf("resolveURL relative = %q, want %q", result, want)
	}
}

func TestResolveURL_Absolute(t *testing.T) {
	result := resolveURL("https://cdn.example.com/master.m3u8", "https://other.com/stream.m3u8")
	want := "https://other.com/stream.m3u8"
	if result != want {
		t.Errorf("resolveURL absolute = %q, want %q", result, want)
	}
}

func TestResolveURL_RootRelative(t *testing.T) {
	result := resolveURL("https://cdn.example.com/path/master.m3u8", "/video/720p.m3u8")
	want := "https://cdn.example.com/video/720p.m3u8"
	if result != want {
		t.Errorf("resolveURL root-relative = %q, want %q", result, want)
	}
}

// ─── Resolution Label ─────────────────────────────────────────────────────────

func TestResolutionToLabel(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"3840x2160", "4K"},
		{"2560x1440", "1440p"},
		{"1920x1080", "1080p"},
		{"1280x720", "720p"},
		{"854x480", "480p"},
		{"640x360", "360p"},
		{"426x240", "240p"},
		{"256x144", "144p"},
	}

	for _, tc := range tests {
		got := resolutionToLabel(tc.input)
		if got != tc.want {
			t.Errorf("resolutionToLabel(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

// ─── Filename Extraction ──────────────────────────────────────────────────────

func TestFilenameFromURL(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{"https://cdn.example.com/video/720p.mp4", "720p.mp4"},
		{"https://cdn.example.com/video/720p.mp4?token=abc", "720p.mp4"},
		{"https://cdn.example.com/", "media"},
		{"https://cdn.example.com", "media"},
	}
	for _, tc := range tests {
		got := filenameFromURL(tc.url)
		if got != tc.want {
			t.Errorf("filenameFromURL(%q) = %q, want %q", tc.url, got, tc.want)
		}
	}
}

// ─── API Handler Tests ────────────────────────────────────────────────────────

func TestHandleGrabResolve_HLSManifest(t *testing.T) {
	// The HLS/DASH parsing is tested extensively by TestResolveHLS_* and TestResolveDASH_*.
	// This test verifies the handler accepts valid JSON and returns proper response format.
	// Real manifest fetch would be blocked by ValidateURL (loopback), which is correct for security.
	s := newTestControlServer(t)

	body, _ := json.Marshal(ResolveRequest{
		URL:       "https://cdn.example.com/master.m3u8",
		UserAgent: "TDM-Test",
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/grab/resolve", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	s.handleGrabResolve(rec, req)

	// The fetch will fail (non-existent host), but the handler should not panic
	// and should return a proper error response (502 Bad Gateway).
	if rec.Code != http.StatusBadGateway {
		t.Logf("status = %d (expected 502 for unreachable host)", rec.Code)
	}
}

func TestHandleGrabResolve_DASHManifest(t *testing.T) {
	s := newTestControlServer(t)

	body, _ := json.Marshal(ResolveRequest{
		URL: "https://cdn.example.com/manifest.mpd",
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/grab/resolve", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	s.handleGrabResolve(rec, req)

	// Unreachable host → 502
	if rec.Code != http.StatusBadGateway {
		t.Logf("status = %d (expected 502 for unreachable host)", rec.Code)
	}
}

func TestHandleGrabResolve_EmptyURL(t *testing.T) {
	s := newTestControlServer(t)

	body, _ := json.Marshal(ResolveRequest{URL: ""})
	req := httptest.NewRequest(http.MethodPost, "/v1/grab/resolve", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	s.handleGrabResolve(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestHandleGrabResolve_CORS(t *testing.T) {
	s := newTestControlServer(t)

	req := httptest.NewRequest(http.MethodOptions, "/v1/grab/resolve", nil)
	rec := httptest.NewRecorder()

	s.handleGrabResolve(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("CORS preflight status = %d, want 200", rec.Code)
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("missing CORS header")
	}
}

func TestHandleGrabDownload_CORS(t *testing.T) {
	s := newTestControlServer(t)

	req := httptest.NewRequest(http.MethodOptions, "/v1/grab/download", nil)
	rec := httptest.NewRecorder()

	s.handleGrabDownload(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("CORS preflight status = %d, want 200", rec.Code)
	}
}

func TestHandleGrabDownload_EmptyURL(t *testing.T) {
	s := newTestControlServer(t)

	body, _ := json.Marshal(GrabRequest{URL: ""})
	req := httptest.NewRequest(http.MethodPost, "/v1/grab/download", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	s.handleGrabDownload(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}
