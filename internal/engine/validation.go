package engine

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
)

const maxURLLength = 8192

// dangerousHeaders that must not be set via user input
var dangerousHeaders = map[string]bool{
	"host":                true,
	"transfer-encoding":   true,
	"content-length":      true,
	"proxy-authorization": true,
	"proxy-connection":    true,
	"te":                  true,
	"upgrade":             true,
}

// ValidateURL checks that a URL is safe to download from.
func ValidateURL(rawURL string) error {
	if rawURL == "" {
		return fmt.Errorf("empty URL")
	}
	if len(rawURL) > maxURLLength {
		return fmt.Errorf("URL exceeds maximum length of %d characters", maxURLLength)
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return fmt.Errorf("unsupported URL scheme %q: only http and https are allowed", u.Scheme)
	}

	if u.Host == "" {
		return fmt.Errorf("URL has no host")
	}

	// Block localhost/loopback to prevent SSRF
	host := strings.ToLower(u.Hostname())
	if host == "localhost" || host == "127.0.0.1" || host == "::1" || host == "0.0.0.0" {
		return fmt.Errorf("downloading from loopback addresses is not allowed")
	}

	return nil
}

// ValidateURLAllowLoopback validates a URL like ValidateURL but permits loopback
// addresses. Used only during integration tests with local test servers.
func ValidateURLAllowLoopback(rawURL string) error {
	if rawURL == "" {
		return fmt.Errorf("empty URL")
	}
	if len(rawURL) > maxURLLength {
		return fmt.Errorf("URL exceeds maximum length of %d characters", maxURLLength)
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return fmt.Errorf("unsupported URL scheme %q: only http and https are allowed", u.Scheme)
	}

	if u.Host == "" {
		return fmt.Errorf("URL has no host")
	}

	return nil
}

// SanitizeFilename removes path traversal components and dangerous characters
// from a user-supplied filename, returning a safe basename.
func SanitizeFilename(name string) string {
	if name == "" {
		return ""
	}

	// Normalize separators and take only the final component
	name = strings.ReplaceAll(name, "\\", "/")
	name = filepath.Base(name)

	// Strip leading dots to prevent hidden files / traversal
	name = strings.TrimLeft(name, ".")

	// Remove null bytes and control characters
	cleaned := strings.Map(func(r rune) rune {
		if r < 32 || r == 127 {
			return -1
		}
		return r
	}, name)

	// Replace remaining dangerous path characters
	replacer := strings.NewReplacer(
		"..", "",
		"/", "_",
		"\\", "_",
		":", "_",
	)
	cleaned = replacer.Replace(cleaned)

	if cleaned == "" || cleaned == "." {
		return "download"
	}

	// Enforce a reasonable filename length (255 bytes is the OS limit)
	if len(cleaned) > 200 {
		ext := filepath.Ext(cleaned)
		cleaned = cleaned[:200-len(ext)] + ext
	}

	return cleaned
}

// ValidateHeaderKey checks that a custom header key is safe to apply.
func ValidateHeaderKey(key string) error {
	lower := strings.ToLower(strings.TrimSpace(key))
	if dangerousHeaders[lower] {
		return fmt.Errorf("header %q is not allowed as a custom header", key)
	}
	return nil
}
