package core

import (
	"path/filepath"
	"testing"
)

func TestGetCategory(t *testing.T) {
	tests := []struct {
		filename string
		expected string
	}{
		{"movie.mp4", "Video"},
		{"song.mp3", "Music"},
		{"archive.zip", "Archives"},
		{"doc.pdf", "Documents"},
		{"setup.exe", "Programs"},
		{"random.xyz", "Others"},
	}

	for _, tt := range tests {
		cat := GetCategory(tt.filename)
		if cat != tt.expected {
			t.Errorf("GetCategory(%s) = %s; expected %s", tt.filename, cat, tt.expected)
		}
	}
}

func TestGetOrganizedPath(t *testing.T) {
	base := "C:\\Downloads"
	filename := "test.jpg"

	path, err := GetOrganizedPath(base, filename)
	if err != nil {
		t.Fatalf("GetOrganizedPath failed: %v", err)
	}

	expected := filepath.Join(base, "Images", filename)
	if path != expected {
		t.Errorf("Expected path %s, got %s", expected, path)
	}
}
