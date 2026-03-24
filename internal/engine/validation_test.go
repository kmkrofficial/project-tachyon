package engine

import (
	"strings"
	"testing"
)

func TestValidateURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr string
	}{
		{"empty", "", "empty URL"},
		{"valid http", "http://example.com/file.zip", ""},
		{"valid https", "https://example.com/file.zip", ""},
		{"ftp rejected", "ftp://example.com/file.zip", "unsupported URL scheme"},
		{"javascript rejected", "javascript:alert(1)", "unsupported URL scheme"},
		{"data rejected", "data:text/html,hello", "unsupported URL scheme"},
		{"file rejected", "file:///etc/passwd", "unsupported URL scheme"},
		{"no host", "http://", "URL has no host"},
		{"localhost blocked", "http://localhost/file.zip", "loopback"},
		{"127.0.0.1 blocked", "http://127.0.0.1/file.zip", "loopback"},
		{"::1 blocked", "http://[::1]/file.zip", "loopback"},
		{"0.0.0.0 blocked", "http://0.0.0.0/file.zip", "loopback"},
		{"too long", "https://example.com/" + strings.Repeat("a", maxURLLength), "exceeds maximum length"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateURL(tt.url)
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("ValidateURL(%q) unexpected error: %v", tt.url, err)
				}
			} else {
				if err == nil {
					t.Errorf("ValidateURL(%q) expected error containing %q, got nil", tt.url, tt.wantErr)
				} else if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("ValidateURL(%q) error = %q, want substring %q", tt.url, err.Error(), tt.wantErr)
				}
			}
		})
	}
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"empty", "", ""},
		{"normal", "file.zip", "file.zip"},
		{"traversal dots", "../../etc/passwd", "passwd"},
		{"backslash traversal", "..\\..\\system32\\config", "config"},
		{"absolute unix", "/etc/passwd", "passwd"},
		{"absolute win", "C:\\Windows\\System32\\cmd.exe", "cmd.exe"},
		{"hidden file", ".hidden", "hidden"},
		{"null bytes", "file\x00.zip", "file.zip"},
		{"control chars", "file\x01\x02.zip", "file.zip"},
		{"double dots in name", "fi..le.zip", "file.zip"},
		{"slashes replaced", "dir/file.zip", "file.zip"},
		{"empty after clean", "..", "download"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeFilename(tt.input)
			if got != tt.want {
				t.Errorf("SanitizeFilename(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestValidateHeaderKey(t *testing.T) {
	tests := []struct {
		key     string
		wantErr bool
	}{
		{"Referer", false},
		{"User-Agent", false},
		{"X-Custom", false},
		{"Host", true},
		{"Transfer-Encoding", true},
		{"Content-Length", true},
		{"Proxy-Authorization", true},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			err := ValidateHeaderKey(tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateHeaderKey(%q) err=%v, wantErr=%v", tt.key, err, tt.wantErr)
			}
		})
	}
}
