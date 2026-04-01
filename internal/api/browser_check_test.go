package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleBrowserCheck_CORSPreflight(t *testing.T) {
	req := httptest.NewRequest(http.MethodOptions, "/v1/browser/check", nil)
	w := httptest.NewRecorder()

	s := &ControlServer{}
	s.handleBrowserCheck(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("missing CORS header")
	}
}

func TestHandleBrowserCheck_MissingURL(t *testing.T) {
	body, _ := json.Marshal(CheckRequest{URL: ""})
	req := httptest.NewRequest(http.MethodPost, "/v1/browser/check", bytes.NewReader(body))
	w := httptest.NewRecorder()

	s := &ControlServer{}
	s.handleBrowserCheck(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestHandleBrowserCheck_InvalidJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/v1/browser/check", bytes.NewReader([]byte("not json")))
	w := httptest.NewRecorder()

	s := &ControlServer{}
	s.handleBrowserCheck(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestCheckRequest_JSON(t *testing.T) {
	cr := CheckRequest{
		URL:      "https://example.com/file.zip",
		Filename: "file.zip",
	}

	data, err := json.Marshal(cr)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded CheckRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.URL != cr.URL {
		t.Errorf("URL = %q, want %q", decoded.URL, cr.URL)
	}
	if decoded.Filename != cr.Filename {
		t.Errorf("Filename = %q, want %q", decoded.Filename, cr.Filename)
	}
}

func TestCheckResponse_ClearJSON(t *testing.T) {
	resp := CheckResponse{Status: "clear"}
	data, _ := json.Marshal(resp)

	var decoded CheckResponse
	json.Unmarshal(data, &decoded)

	if decoded.Status != "clear" {
		t.Errorf("Status = %q, want clear", decoded.Status)
	}
	if decoded.TaskID != "" {
		t.Errorf("TaskID should be empty for clear, got %q", decoded.TaskID)
	}
}

func TestCheckResponse_DownloadingJSON(t *testing.T) {
	resp := CheckResponse{
		Status:   "downloading",
		TaskID:   "abc-123",
		Filename: "file.zip",
		Progress: 42.5,
		Size:     1024000,
	}
	data, _ := json.Marshal(resp)

	var decoded CheckResponse
	json.Unmarshal(data, &decoded)

	if decoded.Status != "downloading" {
		t.Errorf("Status = %q, want downloading", decoded.Status)
	}
	if decoded.Progress != 42.5 {
		t.Errorf("Progress = %f, want 42.5", decoded.Progress)
	}
}

func TestCheckResponse_CompletedJSON(t *testing.T) {
	resp := CheckResponse{
		Status:   "completed",
		TaskID:   "xyz-789",
		Filename: "video.mp4",
		SavePath: "/downloads/video.mp4",
		Size:     5242880,
	}
	data, _ := json.Marshal(resp)

	var decoded CheckResponse
	json.Unmarshal(data, &decoded)

	if decoded.Status != "completed" {
		t.Errorf("Status = %q, want completed", decoded.Status)
	}
	if decoded.SavePath != "/downloads/video.mp4" {
		t.Errorf("SavePath = %q, want /downloads/video.mp4", decoded.SavePath)
	}
}

func TestCheckResponse_ExistsJSON(t *testing.T) {
	resp := CheckResponse{
		Status:   "exists",
		Filename: "report.pdf",
		SavePath: "/downloads/Documents/report.pdf",
		Size:     102400,
	}
	data, _ := json.Marshal(resp)

	var decoded CheckResponse
	json.Unmarshal(data, &decoded)

	if decoded.Status != "exists" {
		t.Errorf("Status = %q, want exists", decoded.Status)
	}
}
