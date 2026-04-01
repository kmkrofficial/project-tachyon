package api

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"project-tachyon/internal/engine"
	"project-tachyon/internal/filesystem"
)

// GrabRequest is the payload from the extension when downloading a captured stream.
type GrabRequest struct {
	URL            string            `json:"url"`
	PageURL        string            `json:"page_url"`
	Cookies        string            `json:"cookies"`
	UserAgent      string            `json:"user_agent"`
	Referer        string            `json:"referer"`
	Filename       string            `json:"filename"`
	ContentType    string            `json:"content_type"`
	Size           int64             `json:"size"`
	Quality        string            `json:"quality"`
	RequestHeaders map[string]string `json:"request_headers"`
}

// ResolveRequest is the payload for resolving HLS/DASH manifests.
type ResolveRequest struct {
	URL       string `json:"url"`
	PageURL   string `json:"page_url"`
	Cookies   string `json:"cookies"`
	UserAgent string `json:"user_agent"`
	Referer   string `json:"referer"`
}

// StreamVariant describes one quality level from a resolved manifest.
type StreamVariant struct {
	URL         string `json:"url"`
	Quality     string `json:"quality"`
	Resolution  string `json:"resolution"`
	Bandwidth   int    `json:"bandwidth"`
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
}

// ResolveResponse is the response for manifest resolution.
type ResolveResponse struct {
	Type     string          `json:"type"` // "hls" or "dash"
	Variants []StreamVariant `json:"variants"`
}

func (s *ControlServer) handleGrabDownload(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	var req GrabRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxRequestBody)).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.URL == "" {
		http.Error(w, "URL required", http.StatusBadRequest)
		return
	}

	if err := engine.ValidateURL(req.URL); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	filename := engine.SanitizeFilename(req.Filename)

	// Build options from captured request context
	options := make(map[string]string)

	if req.Cookies != "" {
		cookieSlice := ParseCookieString(req.Cookies)
		if len(cookieSlice) > 0 {
			if b, err := json.Marshal(cookieSlice); err == nil {
				options["cookies_json"] = string(b)
			}
		}
	}

	headers := make(map[string]string)
	if req.UserAgent != "" {
		headers["User-Agent"] = req.UserAgent
	} else {
		headers["User-Agent"] = engine.GenericUserAgent
	}
	if req.Referer != "" {
		headers["Referer"] = req.Referer
	} else if req.PageURL != "" {
		headers["Referer"] = req.PageURL
	}

	// Set Origin for YouTube/Google Video URLs (required for auth)
	if strings.Contains(req.URL, "googlevideo.com") {
		headers["Origin"] = "https://www.youtube.com"
		if headers["Referer"] == "" {
			headers["Referer"] = "https://www.youtube.com/"
		}
	}

	// Merge captured request headers (only safe ones)
	for k, v := range req.RequestHeaders {
		kl := strings.ToLower(k)
		// Forward only safe headers—skip hop-by-hop and security-sensitive ones
		if kl == "origin" || kl == "accept" || kl == "accept-language" ||
			kl == "accept-encoding" || kl == "range" {
			headers[k] = v
		}
	}

	if len(headers) > 0 {
		if b, err := json.Marshal(headers); err == nil {
			options["headers_json"] = string(b)
		}
	}

	if req.Quality != "" {
		options["quality"] = req.Quality
	}

	// Pass extension-provided size as a hint (e.g. from YouTube contentLength)
	if req.Size > 0 {
		options["size_hint"] = strconv.FormatInt(req.Size, 10)
	}

	defaultPath, err := filesystem.GetDefaultDownloadPath()
	if err != nil {
		defaultPath = "."
	}

	id, err := s.engine.StartDownload(req.URL, defaultPath, filename, options)
	if err != nil {
		s.audit.Log("127.0.0.1", r.UserAgent(), "POST /v1/grab/download", 500, err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.audit.Log("127.0.0.1", r.UserAgent(), "POST /v1/grab/download", 200, "Started "+id)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "started",
		"id":     id,
	})
}

func (s *ControlServer) handleGrabResolve(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	var req ResolveRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxRequestBody)).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.URL == "" {
		http.Error(w, "URL required", http.StatusBadRequest)
		return
	}

	if err := engine.ValidateURL(req.URL); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Fetch the manifest
	httpReq, err := http.NewRequest("GET", req.URL, nil)
	if err != nil {
		http.Error(w, "Invalid manifest URL", http.StatusBadRequest)
		return
	}

	ua := req.UserAgent
	if ua == "" {
		ua = engine.GenericUserAgent
	}
	httpReq.Header.Set("User-Agent", ua)
	if req.Referer != "" {
		httpReq.Header.Set("Referer", req.Referer)
	} else if req.PageURL != "" {
		httpReq.Header.Set("Referer", req.PageURL)
	}
	if req.Cookies != "" {
		httpReq.Header.Set("Cookie", req.Cookies)
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		http.Error(w, "Failed to fetch manifest: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024)) // 2MB max manifest
	if err != nil {
		http.Error(w, "Failed to read manifest", http.StatusBadGateway)
		return
	}

	content := string(body)
	ct := strings.ToLower(resp.Header.Get("Content-Type"))

	var result ResolveResponse

	if strings.Contains(ct, "mpegurl") || strings.Contains(req.URL, ".m3u8") ||
		strings.HasPrefix(strings.TrimSpace(content), "#EXTM3U") {
		result = resolveHLS(content, req.URL)
	} else if strings.Contains(ct, "dash") || strings.Contains(req.URL, ".mpd") ||
		strings.Contains(content, "<MPD") {
		result = resolveDASH(content, req.URL)
	} else {
		http.Error(w, "Unknown manifest format", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func setCORSHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Tachyon-Token")
}

// ─── HLS Manifest Parsing ─────────────────────────────────────────────────────

var hlsBandwidthRe = regexp.MustCompile(`BANDWIDTH=(\d+)`)
var hlsResolutionRe = regexp.MustCompile(`RESOLUTION=(\d+x\d+)`)
var hlsNameRe = regexp.MustCompile(`NAME="([^"]+)"`)

func resolveHLS(content, manifestURL string) ResolveResponse {
	result := ResolveResponse{Type: "hls"}
	scanner := bufio.NewScanner(strings.NewReader(content))

	var currentTag string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.HasPrefix(line, "#EXT-X-STREAM-INF:") {
			currentTag = line
			continue
		}

		// Lines after #EXT-X-STREAM-INF are the variant URLs
		if currentTag != "" && !strings.HasPrefix(line, "#") && line != "" {
			variant := parseHLSVariant(currentTag, line, manifestURL)
			result.Variants = append(result.Variants, variant)
			currentTag = ""
		}
	}

	// If no multi-variant found, the manifest itself is the stream
	if len(result.Variants) == 0 {
		result.Variants = append(result.Variants, StreamVariant{
			URL:         manifestURL,
			Quality:     "Default",
			ContentType: "application/x-mpegurl",
			Filename:    filenameFromURL(manifestURL),
		})
	}

	return result
}

func parseHLSVariant(tag, streamURL, manifestURL string) StreamVariant {
	v := StreamVariant{ContentType: "application/x-mpegurl"}

	// Resolve relative URL
	v.URL = resolveURL(manifestURL, streamURL)

	// Parse BANDWIDTH
	if m := hlsBandwidthRe.FindStringSubmatch(tag); len(m) == 2 {
		v.Bandwidth, _ = strconv.Atoi(m[1])
	}

	// Parse RESOLUTION
	if m := hlsResolutionRe.FindStringSubmatch(tag); len(m) == 2 {
		v.Resolution = m[1]
		v.Quality = resolutionToLabel(m[1])
	}

	// Parse NAME
	if m := hlsNameRe.FindStringSubmatch(tag); len(m) == 2 {
		if v.Quality == "" {
			v.Quality = m[1]
		}
	}

	// Fallback quality from bandwidth
	if v.Quality == "" && v.Bandwidth > 0 {
		v.Quality = fmt.Sprintf("%d kbps", v.Bandwidth/1000)
	}

	v.Filename = filenameFromURL(v.URL)
	return v
}

// ─── DASH Manifest Parsing ────────────────────────────────────────────────────

var dashBandwidthRe = regexp.MustCompile(`bandwidth="(\d+)"`)
var dashWidthRe = regexp.MustCompile(`width="(\d+)"`)
var dashHeightRe = regexp.MustCompile(`height="(\d+)"`)
var dashMimeRe = regexp.MustCompile(`mimeType="([^"]+)"`)
var dashBaseURLRe = regexp.MustCompile(`<BaseURL>([^<]+)</BaseURL>`)
var dashInitRe = regexp.MustCompile(`initialization="([^"]+)"`)

func resolveDASH(content, manifestURL string) ResolveResponse {
	result := ResolveResponse{Type: "dash"}

	// Split into AdaptationSet blocks
	adaptationSets := splitTag(content, "AdaptationSet")

	for _, as := range adaptationSets {
		asMime := extractAttr(as, dashMimeRe)

		// Split into Representation blocks
		representations := splitTag(as, "Representation")

		for _, rep := range representations {
			v := StreamVariant{}

			// Bandwidth
			if m := dashBandwidthRe.FindStringSubmatch(rep); len(m) == 2 {
				v.Bandwidth, _ = strconv.Atoi(m[1])
			}

			// Resolution
			width := extractAttr(rep, dashWidthRe)
			height := extractAttr(rep, dashHeightRe)
			if width != "" && height != "" {
				v.Resolution = width + "x" + height
				v.Quality = resolutionToLabel(v.Resolution)
			}

			// Mime type
			mime := extractAttr(rep, dashMimeRe)
			if mime == "" {
				mime = asMime
			}
			v.ContentType = mime

			// URL — BaseURL or SegmentTemplate
			if m := dashBaseURLRe.FindStringSubmatch(rep); len(m) == 2 {
				v.URL = resolveURL(manifestURL, strings.TrimSpace(m[1]))
			} else if m := dashInitRe.FindStringSubmatch(rep); len(m) == 2 {
				v.URL = resolveURL(manifestURL, strings.TrimSpace(m[1]))
			}

			if v.URL == "" {
				v.URL = manifestURL
			}

			if v.Quality == "" && v.Bandwidth > 0 {
				v.Quality = fmt.Sprintf("%d kbps", v.Bandwidth/1000)
			}

			v.Filename = filenameFromURL(v.URL)
			result.Variants = append(result.Variants, v)
		}
	}

	if len(result.Variants) == 0 {
		result.Variants = append(result.Variants, StreamVariant{
			URL:         manifestURL,
			Quality:     "Default",
			ContentType: "application/dash+xml",
			Filename:    filenameFromURL(manifestURL),
		})
	}

	return result
}

func splitTag(content, tagName string) []string {
	var blocks []string
	openTag := "<" + tagName
	closeTag := "</" + tagName + ">"

	remaining := content
	for {
		start := strings.Index(remaining, openTag)
		if start == -1 {
			break
		}
		end := strings.Index(remaining[start:], closeTag)
		if end == -1 {
			// Self-closing or malformed — take rest
			blocks = append(blocks, remaining[start:])
			break
		}
		blocks = append(blocks, remaining[start:start+end+len(closeTag)])
		remaining = remaining[start+end+len(closeTag):]
	}
	return blocks
}

func extractAttr(block string, re *regexp.Regexp) string {
	if m := re.FindStringSubmatch(block); len(m) == 2 {
		return m[1]
	}
	return ""
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func resolveURL(base, ref string) string {
	if strings.HasPrefix(ref, "http://") || strings.HasPrefix(ref, "https://") {
		return ref
	}
	baseURL, err := url.Parse(base)
	if err != nil {
		return ref
	}
	refURL, err := url.Parse(ref)
	if err != nil {
		return ref
	}
	return baseURL.ResolveReference(refURL).String()
}

func resolutionToLabel(res string) string {
	parts := strings.Split(res, "x")
	if len(parts) != 2 {
		return res
	}
	height, err := strconv.Atoi(parts[1])
	if err != nil {
		return res
	}
	switch {
	case height >= 2160:
		return "4K"
	case height >= 1440:
		return "1440p"
	case height >= 1080:
		return "1080p"
	case height >= 720:
		return "720p"
	case height >= 480:
		return "480p"
	case height >= 360:
		return "360p"
	case height >= 240:
		return "240p"
	default:
		return fmt.Sprintf("%dp", height)
	}
}

func filenameFromURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "media"
	}
	base := path.Base(u.Path)
	if base == "" || base == "." || base == "/" {
		return "media"
	}
	// Remove query params from filename
	if idx := strings.Index(base, "?"); idx != -1 {
		base = base[:idx]
	}
	return base
}
