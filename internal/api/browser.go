package api

import (
	"encoding/json"
	"net/http"
	"project-tachyon/internal/core"
)

type BrowserParams struct {
	URL       string `json:"url"`
	Cookies   string `json:"cookies"` // Raw string "a=b; c=d"
	UserAgent string `json:"user_agent"`
	Referer   string `json:"referer"`
	Filename  string `json:"filename"`
}

func (s *ControlServer) handleBrowserTrigger(w http.ResponseWriter, r *http.Request) {
	// Allow CORS for browser extension
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	var params BrowserParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if params.URL == "" {
		http.Error(w, "URL required", http.StatusBadRequest)
		return
	}

	// Parse Cookies strictly
	var cookieSlice []*http.Cookie
	if params.Cookies != "" {
		cookieSlice = ParseCookieString(params.Cookies)
	}

	// Prepare Options
	options := make(map[string]string)

	// Serialize cookies to JSON for storage
	if len(cookieSlice) > 0 {
		if b, err := json.Marshal(cookieSlice); err == nil {
			options["cookies_json"] = string(b)
		}
	} else if params.Cookies != "" {
		// Fallback to raw if parsing failed / empty (though helper shouldn't fail heavily)
		options["cookies_json"] = params.Cookies // Or store as raw? Engine needs to know.
		// Actually, let's trust ParseCookieString handles it or returns empty.
		// If empty, maybe raw was just bad.
	}

	// Headers Map
	headers := make(map[string]string)
	if params.UserAgent != "" {
		headers["User-Agent"] = params.UserAgent
	} else {
		headers["User-Agent"] = core.GenericUserAgent
	}
	if params.Referer != "" {
		headers["Referer"] = params.Referer
	}

	// Serialize headers
	if len(headers) > 0 {
		if b, err := json.Marshal(headers); err == nil {
			options["headers_json"] = string(b)
		}
	}

	// Determine Save Path
	defaultPath, err := core.GetDefaultDownloadPath()
	if err != nil {
		// Fallback
		defaultPath = "."
	}

	// Start Download
	id, err := s.engine.StartDownload(params.URL, defaultPath, params.Filename, options)
	if err != nil {
		s.audit.Log("127.0.0.1", r.UserAgent(), "POST /v1/browser/trigger", 500, err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.audit.Log("127.0.0.1", r.UserAgent(), "POST /v1/browser/trigger", 200, "Started "+id)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "started",
		"id":     id,
	})
}

// ParseCookieString helper parses a raw cookie string into a slice of http.Cookie
func ParseCookieString(raw string) []*http.Cookie {
	header := http.Header{}
	header.Add("Cookie", raw)
	req := http.Request{Header: header}
	return req.Cookies()
}
