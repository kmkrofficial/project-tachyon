package main

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestServerAndHTTP() (*TestServer, *httptest.Server) {
	ts := NewTestServer(slog.New(slog.NewTextHandler(io.Discard, nil)))
	srv := httptest.NewServer(ts.Handler())
	return ts, srv
}

// --- /file/ endpoint ---

func TestFileEndpoint_BasicDownload(t *testing.T) {
	_, srv := newTestServerAndHTTP()
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/file/1kb")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if len(body) != 1024 {
		t.Fatalf("expected 1024 bytes, got %d", len(body))
	}
	if resp.Header.Get("Accept-Ranges") != "bytes" {
		t.Error("missing Accept-Ranges header")
	}
}

func TestFileEndpoint_LargerFile(t *testing.T) {
	_, srv := newTestServerAndHTTP()
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/file/1mb")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if len(body) != 1024*1024 {
		t.Fatalf("expected %d bytes, got %d", 1024*1024, len(body))
	}
}

func TestFileEndpoint_InvalidSize(t *testing.T) {
	_, srv := newTestServerAndHTTP()
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/file/notasize")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 400 {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestFileEndpoint_MissingSize(t *testing.T) {
	_, srv := newTestServerAndHTTP()
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/file/")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 400 {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

// --- Range request support ---

func TestFileEndpoint_RangeRequest(t *testing.T) {
	_, srv := newTestServerAndHTTP()
	defer srv.Close()

	req, _ := http.NewRequest("GET", srv.URL+"/file/10kb", nil)
	req.Header.Set("Range", "bytes=0-99")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 206 {
		t.Fatalf("expected 206, got %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if len(body) != 100 {
		t.Fatalf("expected 100 bytes, got %d", len(body))
	}
	cr := resp.Header.Get("Content-Range")
	if !strings.HasPrefix(cr, "bytes 0-99/") {
		t.Fatalf("unexpected Content-Range: %s", cr)
	}
}

func TestFileEndpoint_RangeMiddle(t *testing.T) {
	_, srv := newTestServerAndHTTP()
	defer srv.Close()

	req, _ := http.NewRequest("GET", srv.URL+"/file/10kb", nil)
	req.Header.Set("Range", "bytes=100-199")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 206 {
		t.Fatalf("expected 206, got %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if len(body) != 100 {
		t.Fatalf("expected 100 bytes, got %d", len(body))
	}
}

func TestFileEndpoint_RangeOpenEnd(t *testing.T) {
	_, srv := newTestServerAndHTTP()
	defer srv.Close()

	req, _ := http.NewRequest("GET", srv.URL+"/file/1kb", nil)
	req.Header.Set("Range", "bytes=512-")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 206 {
		t.Fatalf("expected 206, got %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if len(body) != 512 {
		t.Fatalf("expected 512 bytes, got %d", len(body))
	}
}

// --- Deterministic content ---

func TestFileEndpoint_DeterministicContent(t *testing.T) {
	_, srv := newTestServerAndHTTP()
	defer srv.Close()

	// Two requests to the same endpoint should produce identical content
	get := func() []byte {
		resp, err := http.Get(srv.URL + "/file/1kb")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		return body
	}

	body1 := get()
	body2 := get()
	if len(body1) != len(body2) {
		t.Fatal("different lengths")
	}
	for i := range body1 {
		if body1[i] != body2[i] {
			t.Fatalf("mismatch at offset %d", i)
		}
	}
}

// --- /slow/ endpoint ---

func TestSlowEndpoint_ServesContent(t *testing.T) {
	_, srv := newTestServerAndHTTP()
	defer srv.Close()

	// Use a very small file with fast speed to keep the test quick
	resp, err := http.Get(srv.URL + "/slow/1kb?speed=1mb")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if len(body) != 1024 {
		t.Fatalf("expected 1024 bytes, got %d", len(body))
	}
}

func TestSlowEndpoint_InvalidSpeed(t *testing.T) {
	_, srv := newTestServerAndHTTP()
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/slow/1kb?speed=invalid")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 400 {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

// --- /fail-at/ endpoint ---

func TestFailAtEndpoint_Truncates(t *testing.T) {
	_, srv := newTestServerAndHTTP()
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/fail-at/10kb?at=2kb")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if len(body) != 2048 {
		t.Fatalf("expected 2048 bytes (fail at 2kb), got %d", len(body))
	}
}

func TestFailAtEndpoint_DefaultHalfway(t *testing.T) {
	_, srv := newTestServerAndHTTP()
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/fail-at/4kb")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if len(body) != 2048 {
		t.Fatalf("expected 2048 bytes (halfway), got %d", len(body))
	}
}

// --- /flaky/ endpoint ---

func TestFlakyEndpoint_SomeRequestsSucceed(t *testing.T) {
	_, srv := newTestServerAndHTTP()
	defer srv.Close()

	successes := 0
	failures := 0
	for i := 0; i < 20; i++ {
		resp, err := http.Get(srv.URL + "/flaky/1kb?fail_rate=0.3")
		if err != nil {
			t.Fatal(err)
		}
		switch resp.StatusCode {
		case 200:
			successes++
		case 500:
			failures++
		}
		resp.Body.Close()
	}
	if successes == 0 {
		t.Error("no requests succeeded out of 20")
	}
	if failures == 0 {
		t.Error("no requests failed out of 20 (expected some with 30% fail rate)")
	}
}

// --- /auth/ endpoint ---

func TestAuthEndpoint_RejectsNoCredentials(t *testing.T) {
	_, srv := newTestServerAndHTTP()
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/auth/1kb")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 401 {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
	if resp.Header.Get("WWW-Authenticate") == "" {
		t.Error("missing WWW-Authenticate header")
	}
}

func TestAuthEndpoint_RejectsWrongCredentials(t *testing.T) {
	_, srv := newTestServerAndHTTP()
	defer srv.Close()

	req, _ := http.NewRequest("GET", srv.URL+"/auth/1kb", nil)
	req.SetBasicAuth("wrong", "creds")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 401 {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

func TestAuthEndpoint_AcceptsValidCredentials(t *testing.T) {
	_, srv := newTestServerAndHTTP()
	defer srv.Close()

	req, _ := http.NewRequest("GET", srv.URL+"/auth/1kb", nil)
	req.SetBasicAuth("test", "test")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if len(body) != 1024 {
		t.Fatalf("expected 1024 bytes, got %d", len(body))
	}
}

// --- /redirect/ endpoint ---

func TestRedirectEndpoint_FollowsRedirects(t *testing.T) {
	_, srv := newTestServerAndHTTP()
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/redirect/3/1kb")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if len(body) != 1024 {
		t.Fatalf("expected 1024 bytes, got %d", len(body))
	}
}

func TestRedirectEndpoint_ZeroRedirects(t *testing.T) {
	_, srv := newTestServerAndHTTP()
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/redirect/0/1kb")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestRedirectEndpoint_SingleRedirect(t *testing.T) {
	_, srv := newTestServerAndHTTP()
	defer srv.Close()

	// Don't follow redirects to verify the 302
	client := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}}
	resp, err := client.Get(srv.URL + "/redirect/2/1kb")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 302 {
		t.Fatalf("expected 302, got %d", resp.StatusCode)
	}
	loc := resp.Header.Get("Location")
	if loc != "/redirect/1/1kb" {
		t.Fatalf("expected redirect to /redirect/1/1kb, got %s", loc)
	}
}

func TestRedirectEndpoint_InvalidPath(t *testing.T) {
	_, srv := newTestServerAndHTTP()
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/redirect/abc")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 400 {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

// --- /stats endpoint ---

func TestStatsEndpoint_ReturnsJSON(t *testing.T) {
	_, srv := newTestServerAndHTTP()
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/stats")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected application/json, got %s", ct)
	}
	var stats map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		t.Fatal("failed to decode stats JSON:", err)
	}
	if _, ok := stats["requests"]; !ok {
		t.Error("missing 'requests' field in stats")
	}
	if _, ok := stats["bytes_served"]; !ok {
		t.Error("missing 'bytes_served' field in stats")
	}
}

// --- Media endpoints ---

func TestMediaImage_PNG_HasValidHeader(t *testing.T) {
	_, srv := newTestServerAndHTTP()
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/media/image/test.png?size=1kb")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if len(body) != 1024 {
		t.Fatalf("expected 1024 bytes, got %d", len(body))
	}
	// PNG signature: 0x89 0x50 0x4E 0x47
	if body[0] != 0x89 || body[1] != 0x50 || body[2] != 0x4E || body[3] != 0x47 {
		t.Fatalf("invalid PNG header: %x %x %x %x", body[0], body[1], body[2], body[3])
	}
	if resp.Header.Get("Content-Type") != "image/png" {
		t.Errorf("expected image/png content type, got %s", resp.Header.Get("Content-Type"))
	}
}

func TestMediaImage_JPEG_HasValidHeader(t *testing.T) {
	_, srv := newTestServerAndHTTP()
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/media/image/photo.jpg?size=2kb")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if body[0] != 0xFF || body[1] != 0xD8 {
		t.Fatalf("invalid JPEG header: %x %x", body[0], body[1])
	}
	if resp.Header.Get("Content-Type") != "image/jpeg" {
		t.Errorf("expected image/jpeg, got %s", resp.Header.Get("Content-Type"))
	}
}

func TestMediaVideo_MP4_HasValidHeader(t *testing.T) {
	_, srv := newTestServerAndHTTP()
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/media/video/clip.mp4?size=2kb")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	// MP4 ftyp box: "ftyp" at offset 4
	if string(body[4:8]) != "ftyp" {
		t.Fatalf("missing ftyp box, got: %x", body[4:8])
	}
	if resp.Header.Get("Content-Type") != "video/mp4" {
		t.Errorf("expected video/mp4, got %s", resp.Header.Get("Content-Type"))
	}
}

func TestMediaAudio_MP3_HasValidHeader(t *testing.T) {
	_, srv := newTestServerAndHTTP()
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/media/audio/track.mp3?size=2kb")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	// ID3 tag
	if string(body[0:3]) != "ID3" {
		t.Fatalf("missing ID3 tag, got: %x", body[0:3])
	}
	if resp.Header.Get("Content-Type") != "audio/mpeg" {
		t.Errorf("expected audio/mpeg, got %s", resp.Header.Get("Content-Type"))
	}
}

func TestMediaDocument_PDF_HasValidHeader(t *testing.T) {
	_, srv := newTestServerAndHTTP()
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/media/document/report.pdf?size=2kb")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if !strings.HasPrefix(string(body), "%PDF") {
		t.Fatalf("missing PDF header, first bytes: %x", body[0:4])
	}
	if resp.Header.Get("Content-Type") != "application/pdf" {
		t.Errorf("expected application/pdf, got %s", resp.Header.Get("Content-Type"))
	}
}

func TestMediaArchive_ZIP_HasValidHeader(t *testing.T) {
	_, srv := newTestServerAndHTTP()
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/media/archive/data.zip?size=2kb")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	// ZIP magic: PK (0x50 0x4B)
	if body[0] != 0x50 || body[1] != 0x4B {
		t.Fatalf("invalid ZIP header: %x %x", body[0], body[1])
	}
	if resp.Header.Get("Content-Type") != "application/zip" {
		t.Errorf("expected application/zip, got %s", resp.Header.Get("Content-Type"))
	}
}

func TestMediaImage_ContentDisposition(t *testing.T) {
	_, srv := newTestServerAndHTTP()
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/media/image/myfile.png?size=1kb")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	cd := resp.Header.Get("Content-Disposition")
	if !strings.Contains(cd, "myfile.png") {
		t.Errorf("expected Content-Disposition with filename, got: %s", cd)
	}
}

func TestMediaImage_DefaultFilename(t *testing.T) {
	_, srv := newTestServerAndHTTP()
	defer srv.Close()

	// Empty filename path should default to "image.png"
	resp, err := http.Get(srv.URL + "/media/image/?size=1kb")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

// --- Helper function tests ---

func TestParseSize(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
		wantErr  bool
	}{
		{"1kb", 1024, false},
		{"1mb", 1024 * 1024, false},
		{"10mb", 10 * 1024 * 1024, false},
		{"1gb", 1024 * 1024 * 1024, false},
		{"500b", 500, false},
		{"1024", 1024, false},
		{"0.5mb", 512 * 1024, false},
		{"invalid", 0, true},
		{"", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseSize(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseSize(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.expected {
				t.Fatalf("parseSize(%q) = %d, want %d", tt.input, got, tt.expected)
			}
		})
	}
}

func TestParseSizeFromPath(t *testing.T) {
	got, err := parseSizeFromPath("/file/5mb", "/file/")
	if err != nil {
		t.Fatal(err)
	}
	if got != 5*1024*1024 {
		t.Fatalf("expected %d, got %d", 5*1024*1024, got)
	}

	_, err = parseSizeFromPath("/file/", "/file/")
	if err == nil {
		t.Fatal("expected error for empty size")
	}
}

func TestExtOf(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"photo.png", ".png"},
		{"video.MP4", ".mp4"},
		{"file.tar.gz", ".gz"},
		{"noext", ""},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := extOf(tt.input)
			if got != tt.expected {
				t.Fatalf("extOf(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestFillPattern_Deterministic(t *testing.T) {
	buf1 := make([]byte, 256)
	buf2 := make([]byte, 256)
	fillPattern(buf1, 0)
	fillPattern(buf2, 0)
	for i := range buf1 {
		if buf1[i] != buf2[i] {
			t.Fatalf("pattern not deterministic at idx %d", i)
		}
	}
}

func TestFillPattern_OffsetMatters(t *testing.T) {
	buf1 := make([]byte, 100)
	buf2 := make([]byte, 100)
	fillPattern(buf1, 0)
	fillPattern(buf2, 100)
	differs := false
	for i := range buf1 {
		if buf1[i] != buf2[i] {
			differs = true
			break
		}
	}
	if !differs {
		t.Error("pattern should differ for different offsets")
	}
}

// --- Header function tests ---

func TestHeaders_PNGSignature(t *testing.T) {
	h := pngHeader()
	if len(h) < 8 {
		t.Fatal("PNG header too short")
	}
	if h[0] != 0x89 || h[1] != 0x50 || h[2] != 0x4E || h[3] != 0x47 {
		t.Fatal("invalid PNG signature")
	}
}

func TestHeaders_JPEGSignature(t *testing.T) {
	h := jpegHeader()
	if h[0] != 0xFF || h[1] != 0xD8 {
		t.Fatal("invalid JPEG SOI")
	}
}

func TestHeaders_MP4Signature(t *testing.T) {
	h := mp4Header()
	if string(h[4:8]) != "ftyp" {
		t.Fatal("invalid MP4 ftyp box")
	}
}

func TestHeaders_ZIPSignature(t *testing.T) {
	h := zipHeader()
	if h[0] != 0x50 || h[1] != 0x4B {
		t.Fatal("invalid ZIP magic")
	}
}

func TestHeaders_PDFSignature(t *testing.T) {
	h := pdfHeader()
	if !strings.HasPrefix(string(h), "%PDF") {
		t.Fatal("invalid PDF header")
	}
}

func TestHeaders_WAVSignature(t *testing.T) {
	h := wavHeader(44100)
	if string(h[0:4]) != "RIFF" {
		t.Fatal("invalid WAV RIFF header")
	}
	if string(h[8:12]) != "WAVE" {
		t.Fatal("invalid WAV format")
	}
}

func TestHeaders_FLACSignature(t *testing.T) {
	h := flacHeader()
	if string(h[0:4]) != "fLaC" {
		t.Fatal("invalid FLAC header")
	}
}

func TestHeaders_MP3ID3(t *testing.T) {
	h := mp3Header()
	if string(h[0:3]) != "ID3" {
		t.Fatal("invalid MP3 ID3 tag")
	}
}

func TestHeaders_OGGSignature(t *testing.T) {
	h := oggHeader()
	if string(h[0:4]) != "OggS" {
		t.Fatal("invalid OGG header")
	}
}

// --- parseRangeHeader ---

func TestParseRangeHeader(t *testing.T) {
	tests := []struct {
		header    string
		totalSize int64
		wantStart int64
		wantEnd   int64
		wantErr   bool
	}{
		{"bytes=0-99", 1000, 0, 99, false},
		{"bytes=100-199", 1000, 100, 199, false},
		{"bytes=500-", 1000, 500, 999, false},
		{"bytes=0-9999", 1000, 0, 999, false}, // clamp to total-1
		{"bytes=1000-", 1000, 0, 0, true},     // start >= total
		{"invalid", 1000, 0, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.header, func(t *testing.T) {
			start, end, err := parseRangeHeader(tt.header, tt.totalSize)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				if start != tt.wantStart || end != tt.wantEnd {
					t.Fatalf("got [%d, %d], want [%d, %d]", start, end, tt.wantStart, tt.wantEnd)
				}
			}
		})
	}
}
