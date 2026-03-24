package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Sentinel errors
var (
	// ErrLinkExpired indicates the download URL has expired (HTTP 403)
	ErrLinkExpired = errors.New("link expired or access denied (403)")
	// ErrRangeIgnored indicates the server ignored byte range requests.
	ErrRangeIgnored = errors.New("server ignored range request")
)

// ProbeResult contains metadata from a URL probe
type ProbeResult struct {
	Size         int64  `json:"size"`
	Filename     string `json:"filename"`
	Status       int    `json:"status"`
	AcceptRanges bool   `json:"accept_ranges"`
	ETag         string `json:"etag"`
	LastModified string `json:"last_modified"`
}

// newRequest creates an HTTP request with configured headers
func (e *TachyonEngine) newRequest(method, urlStr string, headersStr string, cookiesStr string) (*http.Request, error) {
	req, err := http.NewRequest(method, urlStr, nil)
	if err != nil {
		return nil, err
	}

	// Use custom User-Agent if set, otherwise use default
	userAgent := e.GetUserAgent()
	if userAgent == "" {
		userAgent = GenericUserAgent
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Connection", "keep-alive")

	// Apply custom headers (with injection prevention)
	if headersStr != "" {
		var headers map[string]string
		if err := json.Unmarshal([]byte(headersStr), &headers); err == nil {
			for k, v := range headers {
				if err := ValidateHeaderKey(k); err != nil {
					continue // Skip dangerous headers silently
				}
				req.Header.Set(k, v)
			}
		}
	}

	// Apply cookies
	if cookiesStr != "" {
		// Detect JSON array
		if strings.HasPrefix(strings.TrimSpace(cookiesStr), "[") {
			var cookies []*http.Cookie
			if err := json.Unmarshal([]byte(cookiesStr), &cookies); err == nil {
				for _, c := range cookies {
					req.AddCookie(c)
				}
			} else {
				// JSON parse failed, fallback to raw string
				req.Header.Set("Cookie", cookiesStr)
			}
		} else {
			// Raw String
			req.Header.Set("Cookie", cookiesStr)
		}
	}

	return req, nil
}

// ProbeURL checks the URL using HEAD first, falling back to GET+Range if needed
func (e *TachyonEngine) ProbeURL(urlStr string, headersStr string, cookiesStr string) (*ProbeResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 1. Try HEAD first (fast, no body transfer)
	result, err := e.probeHEAD(ctx, urlStr, headersStr, cookiesStr)
	if err == nil && result.Size > 0 {
		return result, nil
	}

	// 2. Fallback to GET+Range for servers that don't support HEAD properly
	e.logger.Info("HEAD probe insufficient, falling back to GET+Range", "url", urlStr)
	return e.probeGETRange(ctx, urlStr, headersStr, cookiesStr)
}

// probeHEAD performs a lightweight HEAD request to gather file metadata
func (e *TachyonEngine) probeHEAD(ctx context.Context, urlStr string, headersStr string, cookiesStr string) (*ProbeResult, error) {
	req, err := e.newRequest("HEAD", urlStr, headersStr, cookiesStr)
	if err != nil {
		return nil, friendlyError(err)
	}
	req = req.WithContext(ctx)

	resp, err := e.httpClient.Do(req)
	if err != nil {
		e.logger.Error("HEAD probe failed", "url", urlStr, "error", err)
		return nil, friendlyError(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return &ProbeResult{Status: resp.StatusCode}, friendlyHTTPError(resp.StatusCode)
	}

	return e.parseProbeResponse(resp), nil
}

// probeGETRange performs a GET request with Range: bytes=0-0 as a fallback probe
func (e *TachyonEngine) probeGETRange(ctx context.Context, urlStr string, headersStr string, cookiesStr string) (*ProbeResult, error) {
	req, err := e.newRequest("GET", urlStr, headersStr, cookiesStr)
	if err != nil {
		return nil, friendlyError(err)
	}
	req = req.WithContext(ctx)
	req.Header.Set("Range", "bytes=0-0")

	resp, err := e.httpClient.Do(req)
	if err != nil {
		e.logger.Error("GET range probe failed", "url", urlStr, "error", err)
		return nil, friendlyError(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 && resp.StatusCode != http.StatusPartialContent {
		return &ProbeResult{Status: resp.StatusCode}, friendlyHTTPError(resp.StatusCode)
	}

	return e.parseProbeResponse(resp), nil
}

// parseProbeResponse extracts metadata from an HTTP response
func (e *TachyonEngine) parseProbeResponse(resp *http.Response) *ProbeResult {
	filename := ""
	cd := resp.Header.Get("Content-Disposition")
	if cd != "" {
		if _, params, err := mime.ParseMediaType(cd); err == nil {
			filename = params["filename"]
		}
	}
	if filename == "" {
		filename = filepath.Base(resp.Request.URL.Path)
		if filename == "." || filename == "/" {
			filename = "unknown_file"
		}
	}

	acceptRanges := resp.Header.Get("Accept-Ranges") == "bytes"

	// Size determination
	size := resp.ContentLength

	// If response is 206 Partial Content, parse total size from Content-Range
	if resp.StatusCode == http.StatusPartialContent {
		acceptRanges = true
		cr := resp.Header.Get("Content-Range")
		if cr != "" {
			if parts := strings.Split(cr, "/"); len(parts) == 2 {
				if total, err := strconv.ParseInt(parts[1], 10, 64); err == nil {
					size = total
				}
			}
		}
	}

	return &ProbeResult{
		Size:         size,
		Filename:     filename,
		Status:       resp.StatusCode,
		AcceptRanges: acceptRanges,
		ETag:         resp.Header.Get("ETag"),
		LastModified: resp.Header.Get("Last-Modified"),
	}
}

// friendlyError converts technical errors to user-friendly messages
func friendlyError(err error) error {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "no such host"):
		return fmt.Errorf("Server not found. Check the URL is correct.")
	case strings.Contains(msg, "connection refused"):
		return fmt.Errorf("Server is offline or unreachable.")
	case strings.Contains(msg, "timeout") || strings.Contains(msg, "deadline exceeded"):
		return fmt.Errorf("Connection timed out. Try again later.")
	case strings.Contains(msg, "certificate"):
		return fmt.Errorf("SSL certificate error. The website may not be secure.")
	case strings.Contains(msg, "network is unreachable"):
		return fmt.Errorf("No internet connection.")
	default:
		return fmt.Errorf("Connection failed. Check your internet.")
	}
}

// friendlyHTTPError converts HTTP status codes to user-friendly messages
func friendlyHTTPError(status int) error {
	switch status {
	case 404:
		return fmt.Errorf("File not found on server (404)")
	case 403:
		return fmt.Errorf("Access denied by server (403)")
	case 401:
		return fmt.Errorf("Authentication required (401)")
	case 500, 502, 503:
		return fmt.Errorf("Server error. Try again later (%d)", status)
	case 429:
		return fmt.Errorf("Too many requests. Wait and try again.")
	default:
		return fmt.Errorf("Server returned error %d", status)
	}
}
