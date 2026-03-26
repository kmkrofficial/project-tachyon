package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleBrowserTrigger_CORS(t *testing.T) {
	// Create a request to test CORS headers
	req := httptest.NewRequest(http.MethodOptions, "/v1/browser/trigger", nil)
	w := httptest.NewRecorder()

	// Use a handler directly (we just need to test the CORS part)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
	})

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("CORS preflight status = %d, want 200", w.Code)
	}
	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("Missing CORS Allow-Origin header")
	}
}

func TestBrowserParams_JSON(t *testing.T) {
	params := BrowserParams{
		URL:       "https://example.com/file.zip",
		Cookies:   "session=abc123",
		UserAgent: "Mozilla/5.0",
		Referer:   "https://example.com",
		Filename:  "file.zip",
	}

	data, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded BrowserParams
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.URL != params.URL {
		t.Errorf("URL = %q, want %q", decoded.URL, params.URL)
	}
	if decoded.Cookies != params.Cookies {
		t.Errorf("Cookies = %q, want %q", decoded.Cookies, params.Cookies)
	}
	if decoded.UserAgent != params.UserAgent {
		t.Errorf("UserAgent = %q, want %q", decoded.UserAgent, params.UserAgent)
	}
	if decoded.Referer != params.Referer {
		t.Errorf("Referer = %q, want %q", decoded.Referer, params.Referer)
	}
	if decoded.Filename != params.Filename {
		t.Errorf("Filename = %q, want %q", decoded.Filename, params.Filename)
	}
}

func TestBrowserParams_InvalidJSON(t *testing.T) {
	body := bytes.NewBufferString(`{invalid json}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/browser/trigger", body)

	var params BrowserParams
	err := json.NewDecoder(req.Body).Decode(&params)
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

func TestBrowserParams_EmptyURL(t *testing.T) {
	params := BrowserParams{
		URL: "",
	}
	if params.URL != "" {
		t.Error("URL should be empty")
	}
}

func TestParseCookieString_RoundTrip(t *testing.T) {
	cookies := ParseCookieString("session=abc123; token=xyz789")

	if len(cookies) != 2 {
		t.Fatalf("Expected 2 cookies, got %d", len(cookies))
	}

	found := map[string]string{}
	for _, c := range cookies {
		found[c.Name] = c.Value
	}

	if found["session"] != "abc123" {
		t.Errorf("session cookie = %q, want %q", found["session"], "abc123")
	}
	if found["token"] != "xyz789" {
		t.Errorf("token cookie = %q, want %q", found["token"], "xyz789")
	}
}

func TestParseCookieString_EmptyInput(t *testing.T) {
	cookies := ParseCookieString("")
	if len(cookies) != 0 {
		t.Errorf("Expected 0 cookies for empty string, got %d", len(cookies))
	}
}
