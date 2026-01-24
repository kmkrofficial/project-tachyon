package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetCategory(t *testing.T) {
	tests := []struct {
		filename string
		expected string
	}{
		{"movie.mp4", "Videos"},
		{"song.mp3", "Music"},
		{"archive.zip", "Archives"},
		{"doc.pdf", "Documents"},
		{"setup.exe", "Software"},
		{"random.xyz", "Others"},
		{"image.jpg", "Images"},
		{"video.mkv", "Videos"},
		{"app.dmg", "Software"},
	}

	for _, tt := range tests {
		cat := GetCategory(tt.filename)
		if cat != tt.expected {
			t.Errorf("GetCategory(%s) = %s; expected %s", tt.filename, cat, tt.expected)
		}
	}
}

func TestGetOrganizedPath(t *testing.T) {
	// Use temp directory for cross-platform compatibility
	base := os.TempDir()
	filename := "test.jpg"

	path, err := GetOrganizedPath(base, filename)
	if err != nil {
		t.Fatalf("GetOrganizedPath failed: %v", err)
	}

	expected := filepath.Join(base, "Images", filename)
	if path != expected {
		t.Errorf("Expected path %s, got %s", expected, path)
	}

	// Verify folder was created
	categoryDir := filepath.Join(base, "Images")
	if _, err := os.Stat(categoryDir); os.IsNotExist(err) {
		t.Errorf("Category folder was not created: %s", categoryDir)
	}
}

func TestGetDefaultDownloadPath(t *testing.T) {
	path, err := GetDefaultDownloadPath()
	if err != nil {
		t.Fatalf("GetDefaultDownloadPath failed: %v", err)
	}

	// Verify path ends with Tachyon Downloads
	if !strings.HasSuffix(path, TachyonRootFolder) {
		t.Errorf("Expected path to end with '%s', got %s", TachyonRootFolder, path)
	}

	// Verify folder was created
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("Tachyon root folder was not created: %s", path)
	}
}

func TestEnsureCategoryFolders(t *testing.T) {
	// Use temp directory
	base := filepath.Join(os.TempDir(), "tachyon_test")
	defer os.RemoveAll(base)

	err := EnsureCategoryFolders(base)
	if err != nil {
		t.Fatalf("EnsureCategoryFolders failed: %v", err)
	}

	// Check all categories exist
	categories := []string{"Videos", "Music", "Images", "Archives", "Documents", "Software", "Others"}
	for _, cat := range categories {
		catPath := filepath.Join(base, cat)
		if _, err := os.Stat(catPath); os.IsNotExist(err) {
			t.Errorf("Category folder not created: %s", cat)
		}
	}
}
