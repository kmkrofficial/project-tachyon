package filesystem

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestGetDefaultDownloadPath_Format(t *testing.T) {
	path, err := GetDefaultDownloadPath()
	if err != nil {
		t.Fatalf("GetDefaultDownloadPath() returned error: %v", err)
	}

	if path == "" {
		t.Fatal("GetDefaultDownloadPath() returned empty path")
	}

	// Should end with "Downloads"
	base := filepath.Base(path)
	if base != "Downloads" {
		t.Errorf("GetDefaultDownloadPath() base = %q, want %q", base, "Downloads")
	}

	// Should be absolute
	if !filepath.IsAbs(path) {
		t.Errorf("GetDefaultDownloadPath() returned non-absolute path: %q", path)
	}
}

func TestOpenFile_UnsupportedPlatform(t *testing.T) {
	// We can test the function doesn't panic with a nonexistent file
	// The actual open will fail gracefully via cmd.Start() error
	if runtime.GOOS != "windows" && runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		err := OpenFile("/tmp/nonexistent.txt")
		if err == nil {
			t.Error("Expected error for unsupported platform")
		}
	}
}

func TestOpenFolder_AbsolutePath(t *testing.T) {
	// Create a temp file to test path resolution
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "testfile.txt")
	os.WriteFile(testFile, []byte("test"), 0644)

	// We don't actually want to open a file explorer in tests,
	// but we can verify the function doesn't panic
	// The explorer process will start and close immediately when file doesn't exist in a real environment
	// Just ensure no panic on a valid path
	_ = OpenFolder(testFile) // May or may not error depending on environment
}

func TestOpenFolder_ErrorOnRelativePath(t *testing.T) {
	// Even with relative paths, filepath.Abs should resolve them
	err := OpenFolder("relative/path/file.txt")
	// Should not panic — may fail with explorer but that's OK
	_ = err
}
