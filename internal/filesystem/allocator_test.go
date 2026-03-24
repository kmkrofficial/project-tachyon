package filesystem

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAllocateFile_CreatesFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test_alloc.bin")

	a := NewAllocator()
	err := a.AllocateFile(path, 1024)
	if err != nil {
		t.Fatalf("AllocateFile failed: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("file not created: %v", err)
	}
	if info.Size() != 1024 {
		t.Errorf("expected size 1024, got %d", info.Size())
	}
}

func TestAllocateFile_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "subdir", "nested", "file.bin")

	a := NewAllocator()
	err := a.AllocateFile(path, 512)
	if err != nil {
		t.Fatalf("AllocateFile failed: %v", err)
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("file should exist after allocation")
	}
}

func TestAllocateFile_ZeroSize(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "zero.bin")

	a := NewAllocator()
	err := a.AllocateFile(path, 0)
	if err != nil {
		t.Fatalf("AllocateFile with 0 size failed: %v", err)
	}

	info, _ := os.Stat(path)
	if info.Size() != 0 {
		t.Errorf("expected 0-byte file, got %d", info.Size())
	}
}

func TestAllocateFile_OverwritesSmallerFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "grow.bin")

	a := NewAllocator()
	a.AllocateFile(path, 100)
	a.AllocateFile(path, 2048) // Grow

	info, _ := os.Stat(path)
	if info.Size() != 2048 {
		t.Errorf("expected 2048 after re-allocation, got %d", info.Size())
	}
}

func TestAllocateFile_ShrinkFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "shrink.bin")

	a := NewAllocator()
	a.AllocateFile(path, 4096)
	a.AllocateFile(path, 100) // Shrink

	info, _ := os.Stat(path)
	if info.Size() != 100 {
		t.Errorf("expected 100 after shrink, got %d", info.Size())
	}
}

func TestAllocateFile_LargeFile(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping large allocation in short mode")
	}

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "large.bin")

	a := NewAllocator()
	// 100MB - should succeed on most test machines
	err := a.AllocateFile(path, 100*1024*1024)
	if err != nil {
		t.Fatalf("AllocateFile 100MB failed: %v", err)
	}

	info, _ := os.Stat(path)
	if info.Size() != 100*1024*1024 {
		t.Errorf("expected 100MB, got %d", info.Size())
	}
}

func TestNewAllocator(t *testing.T) {
	a := NewAllocator()
	if a == nil {
		t.Fatal("NewAllocator returned nil")
	}
}

// --- GetDefaultDownloadPath ---

func TestGetDefaultDownloadPath(t *testing.T) {
	path, err := GetDefaultDownloadPath()
	if err != nil {
		t.Fatalf("GetDefaultDownloadPath failed: %v", err)
	}
	if path == "" {
		t.Error("expected non-empty path")
	}
	if !filepath.IsAbs(path) {
		t.Error("expected absolute path")
	}
}

// --- GetCategory ---

func TestGetCategory(t *testing.T) {
	tests := []struct {
		filename string
		want     string
	}{
		{"photo.jpg", "Images"},
		{"pic.png", "Images"},
		{"song.mp3", "Music"},
		{"track.flac", "Music"},
		{"movie.mp4", "Videos"},
		{"clip.mkv", "Videos"},
		{"doc.pdf", "Documents"},
		{"data.xlsx", "Documents"},
		{"app.exe", "Software"},
		{"setup.msi", "Software"},
		{"archive.zip", "Archives"},
		{"backup.tar.gz", "Archives"},
		{"unknown.xyz", "Others"},
		{"noext", "Others"},
	}
	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			got := GetCategory(tt.filename)
			if got != tt.want {
				t.Errorf("GetCategory(%q) = %q, want %q", tt.filename, got, tt.want)
			}
		})
	}
}

// --- GetOrganizedPath ---

func TestGetOrganizedPath(t *testing.T) {
	tmpDir := t.TempDir()

	path, err := GetOrganizedPath(tmpDir, "photo.jpg")
	if err != nil {
		t.Fatalf("GetOrganizedPath failed: %v", err)
	}

	expected := filepath.Join(tmpDir, "Images", "photo.jpg")
	if path != expected {
		t.Errorf("expected %s, got %s", expected, path)
	}
}

// --- FindAvailablePath ---

func TestFindAvailablePath_NoConflict(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "newfile.txt")

	result := FindAvailablePath(path)
	if result != path {
		t.Errorf("no conflict: expected %s, got %s", path, result)
	}
}

func TestFindAvailablePath_WithConflict(t *testing.T) {
	tmpDir := t.TempDir()
	original := filepath.Join(tmpDir, "exists.txt")
	os.WriteFile(original, []byte("data"), 0644)

	result := FindAvailablePath(original)
	expected := filepath.Join(tmpDir, "exists_2.txt")
	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

func TestFindAvailablePath_MultipleConflicts(t *testing.T) {
	tmpDir := t.TempDir()
	base := filepath.Join(tmpDir, "file.txt")
	os.WriteFile(base, []byte("1"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "file_2.txt"), []byte("2"), 0644)

	result := FindAvailablePath(base)
	expected := filepath.Join(tmpDir, "file_3.txt")
	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}
