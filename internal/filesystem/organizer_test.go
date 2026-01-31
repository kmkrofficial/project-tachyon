package filesystem

import (
	"os"
	"path/filepath"
	"project-tachyon/internal/storage"
	"testing"
)

func TestOrganizer(t *testing.T) {
	// Setup Temp Dir
	tmpDir, _ := os.MkdirTemp("", "tachyon_organizer_test")
	defer os.RemoveAll(tmpDir)

	organizer := NewSmartOrganizer()

	// Test Cases
	tests := []struct {
		filename string
		category string
	}{
		{"pic.jpg", "Images"},
		{"song.mp3", "Music"},
		{"doc.pdf", "Documents"},
		{"installer.exe", "Software"},
		{"movie.mp4", "Videos"},
		{"archive.zip", "Archives"},
		{"unknown.xyz", "Others"},
	}

	for _, tt := range tests {
		// Mock Task
		originalPath := filepath.Join(tmpDir, tt.filename)
		// Create dummy file
		os.WriteFile(originalPath, []byte("dummy"), 0644)

		task := &storage.DownloadTask{
			ID:       "1",
			Filename: tt.filename,
			SavePath: originalPath,
		}

		newPath, err := organizer.OrganizeFile(task)
		if err != nil {
			t.Errorf("OrganizeFile(%s) failed: %v", tt.filename, err)
			continue
		}

		expectedDir := filepath.Join(tmpDir, tt.category)
		expectedPath := filepath.Join(expectedDir, tt.filename)

		if newPath != expectedPath {
			t.Errorf("Expected path %s, got %s", expectedPath, newPath)
		}

		// Verify file moved
		if _, err := os.Stat(newPath); os.IsNotExist(err) {
			t.Errorf("File not found at new path: %s", newPath)
		}
	}
}

func TestCollisionHandling(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "tachyon_collision_test")
	defer os.RemoveAll(tmpDir)

	organizer := NewSmartOrganizer()

	filename := "test.jpg"
	category := "Images"

	// Create "Images" dir explicitly
	imgDir := filepath.Join(tmpDir, category)
	os.MkdirAll(imgDir, 0755)

	// Create EXISTING file in target
	targetPath := filepath.Join(imgDir, filename)
	os.WriteFile(targetPath, []byte("existing"), 0644)

	// Create source file
	sourcePath := filepath.Join(tmpDir, filename)
	os.WriteFile(sourcePath, []byte("new"), 0644)

	task := &storage.DownloadTask{
		ID:       "2",
		Filename: filename,
		SavePath: sourcePath,
	}

	newPath, err := organizer.OrganizeFile(task)
	if err != nil {
		t.Fatalf("OrganizeFile failed: %v", err)
	}

	// Expect rename to test (1).jpg
	expectedPath := filepath.Join(imgDir, "test (1).jpg")
	if newPath != expectedPath {
		t.Errorf("Expected collision handling to %s, got %s", expectedPath, newPath)
	}
}
