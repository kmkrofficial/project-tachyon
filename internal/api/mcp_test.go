package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newRequest creates an httptest request for API tests.
func newRequest(t *testing.T, method, path, body string) *http.Request {
	t.Helper()
	return httptest.NewRequest(method, path, strings.NewReader(body))
}

// serve runs a handler and returns the recorder.
func serve(h http.Handler, req *http.Request) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// --- JSON-RPC types ---

func TestJsonRpcRequest_Unmarshal(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		method string
		id     interface{}
	}{
		{
			"download request",
			`{"jsonrpc":"2.0","method":"tachyon_download","params":{"url":"https://example.com/file.zip"},"id":1}`,
			"tachyon_download",
			float64(1), // JSON numbers decode to float64
		},
		{
			"list request",
			`{"jsonrpc":"2.0","method":"tachyon_list","params":{},"id":"abc"}`,
			"tachyon_list",
			"abc",
		},
		{
			"tools/list",
			`{"jsonrpc":"2.0","method":"tools/list","id":42}`,
			"tools/list",
			float64(42),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req JsonRpcRequest
			if err := json.Unmarshal([]byte(tt.input), &req); err != nil {
				t.Fatalf("unmarshal failed: %v", err)
			}
			if req.Method != tt.method {
				t.Errorf("method = %q, want %q", req.Method, tt.method)
			}
			if req.JSONRPC != "2.0" {
				t.Error("jsonrpc should be 2.0")
			}
		})
	}
}

func TestJsonRpcRequest_InvalidJSON(t *testing.T) {
	var req JsonRpcRequest
	err := json.Unmarshal([]byte("not json"), &req)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestJsonRpcResponse_Marshal(t *testing.T) {
	resp := JsonRpcResponse{
		JSONRPC: "2.0",
		Result:  map[string]string{"status": "ok"},
		ID:      1,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded map[string]interface{}
	json.Unmarshal(data, &decoded)

	if decoded["jsonrpc"] != "2.0" {
		t.Error("missing jsonrpc field")
	}
	result := decoded["result"].(map[string]interface{})
	if result["status"] != "ok" {
		t.Error("result mismatch")
	}
}

func TestJsonRpcResponse_ErrorMarshal(t *testing.T) {
	resp := JsonRpcResponse{
		JSONRPC: "2.0",
		Error:   &RpcError{Code: -32601, Message: "Method not found"},
		ID:      "req-1",
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded map[string]interface{}
	json.Unmarshal(data, &decoded)

	errObj := decoded["error"].(map[string]interface{})
	if errObj["code"].(float64) != -32601 {
		t.Error("error code mismatch")
	}
	if errObj["message"] != "Method not found" {
		t.Error("error message mismatch")
	}
	if decoded["result"] != nil {
		t.Error("result should be omitted on error")
	}
}

// --- DownloadParams ---

func TestDownloadParams_Unmarshal(t *testing.T) {
	input := `{"url":"https://example.com/file.zip","path":"/downloads","filename":"custom.zip"}`
	var params DownloadParams
	if err := json.Unmarshal([]byte(input), &params); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if params.URL != "https://example.com/file.zip" {
		t.Errorf("URL = %q", params.URL)
	}
	if params.Path != "/downloads" {
		t.Errorf("Path = %q", params.Path)
	}
	if params.Filename != "custom.zip" {
		t.Errorf("Filename = %q", params.Filename)
	}
}

func TestDownloadParams_MinimalInput(t *testing.T) {
	input := `{"url":"https://example.com/file.zip"}`
	var params DownloadParams
	if err := json.Unmarshal([]byte(input), &params); err != nil {
		t.Fatal(err)
	}
	if params.URL == "" {
		t.Error("URL should be set")
	}
	if params.Path != "" || params.Filename != "" {
		t.Error("optional fields should be empty")
	}
}

// --- BrowserParams ---

func TestBrowserParams_Unmarshal(t *testing.T) {
	input := `{"url":"https://cdn.example.com/file.bin","cookies":"session=abc","user_agent":"MyUA","referer":"https://origin.com","filename":"download.bin"}`
	var params BrowserParams
	if err := json.Unmarshal([]byte(input), &params); err != nil {
		t.Fatal(err)
	}
	if params.URL != "https://cdn.example.com/file.bin" {
		t.Errorf("URL = %q", params.URL)
	}
	if params.Cookies != "session=abc" {
		t.Errorf("Cookies = %q", params.Cookies)
	}
	if params.UserAgent != "MyUA" {
		t.Errorf("UserAgent = %q", params.UserAgent)
	}
	if params.Referer != "https://origin.com" {
		t.Errorf("Referer = %q", params.Referer)
	}
	if params.Filename != "download.bin" {
		t.Errorf("Filename = %q", params.Filename)
	}
}

func TestBrowserParams_EmptyFields(t *testing.T) {
	input := `{"url":"https://example.com/file"}`
	var params BrowserParams
	if err := json.Unmarshal([]byte(input), &params); err != nil {
		t.Fatal(err)
	}
	if params.Cookies != "" || params.UserAgent != "" || params.Referer != "" || params.Filename != "" {
		t.Error("optional fields should be empty")
	}
}

// --- handleBrowserTrigger ---

func TestHandleBrowserTrigger_CORS_Preflight(t *testing.T) {
	s := newTestControlServer(t)
	handler := http.HandlerFunc(s.handleBrowserTrigger)

	req := newRequest(t, "OPTIONS", "/v1/browser/trigger", "")
	rec := serve(handler, req)

	if rec.Code != 200 {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("missing CORS header")
	}
}

func TestHandleBrowserTrigger_EmptyURL(t *testing.T) {
	s := newTestControlServer(t)
	handler := http.HandlerFunc(s.handleBrowserTrigger)

	req := newRequest(t, "POST", "/v1/browser/trigger", `{"url":""}`)
	rec := serve(handler, req)

	if rec.Code != 400 {
		t.Errorf("expected 400 for empty URL, got %d", rec.Code)
	}
}

func TestHandleBrowserTrigger_InvalidJSON(t *testing.T) {
	s := newTestControlServer(t)
	handler := http.HandlerFunc(s.handleBrowserTrigger)

	req := newRequest(t, "POST", "/v1/browser/trigger", "not json")
	rec := serve(handler, req)

	if rec.Code != 400 {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestHandleBrowserTrigger_InvalidURL(t *testing.T) {
	s := newTestControlServer(t)
	handler := http.HandlerFunc(s.handleBrowserTrigger)

	req := newRequest(t, "POST", "/v1/browser/trigger", `{"url":"ftp://bad-scheme.com/file"}`)
	rec := serve(handler, req)

	if rec.Code != 400 {
		t.Errorf("expected 400 for invalid URL scheme, got %d", rec.Code)
	}
}

// --- ParseCookieString additional tests ---

func TestParseCookieString_Whitespace(t *testing.T) {
	cookies := ParseCookieString("  a=1 ;  b=2  ")
	if len(cookies) != 2 {
		t.Fatalf("expected 2 cookies, got %d", len(cookies))
	}
}

func TestParseCookieString_ValueWithEquals(t *testing.T) {
	cookies := ParseCookieString("token=abc=def==")
	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(cookies))
	}
	if cookies[0].Value != "abc=def==" {
		t.Errorf("expected value with = signs preserved, got %q", cookies[0].Value)
	}
}

func TestParseCookieString_NoValue(t *testing.T) {
	cookies := ParseCookieString("orphan")
	// Should handle gracefully (either skip or set empty value)
	t.Logf("parsed %d cookies from bare key", len(cookies))
}

func TestParseCookieString_ManyPairs(t *testing.T) {
	raw := "a=1; b=2; c=3; d=4; e=5; f=6; g=7; h=8; i=9; j=10"
	cookies := ParseCookieString(raw)
	if len(cookies) != 10 {
		t.Errorf("expected 10 cookies, got %d", len(cookies))
	}
}
