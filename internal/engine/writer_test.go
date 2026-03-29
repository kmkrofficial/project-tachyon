package engine

import (
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
)

func TestPartWriter_BasicWrite(t *testing.T) {
	tmpDir := t.TempDir()
	var downloaded int64

	pw, err := newPartWriter(tmpDir, "test-task", 0, &downloaded)
	if err != nil {
		t.Fatal(err)
	}

	data := []byte("hello, tachyon!")
	if err := pw.Write(data); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	pw.Close()

	content, err := os.ReadFile(pw.Path())
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "hello, tachyon!" {
		t.Errorf("expected 'hello, tachyon!', got %q", string(content))
	}
	if atomic.LoadInt64(&downloaded) != int64(len(data)) {
		t.Errorf("expected downloaded=%d, got %d", len(data), downloaded)
	}
}

func TestPartWriter_MultipleWrites(t *testing.T) {
	tmpDir := t.TempDir()
	var downloaded int64

	pw, err := newPartWriter(tmpDir, "test-task", 1, &downloaded)
	if err != nil {
		t.Fatal(err)
	}

	pw.Write([]byte("part1-"))
	pw.Write([]byte("part2-"))
	pw.Write([]byte("part3"))
	pw.Close()

	content, err := os.ReadFile(pw.Path())
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "part1-part2-part3" {
		t.Errorf("expected 'part1-part2-part3', got %q", string(content))
	}
	if pw.Written() != 17 {
		t.Errorf("expected Written()=17, got %d", pw.Written())
	}
}

func TestMergePartFiles(t *testing.T) {
	tmpDir := t.TempDir()
	var downloaded int64

	// Create 3 part files
	for i := 0; i < 3; i++ {
		pw, err := newPartWriter(tmpDir, "merge-task", i, &downloaded)
		if err != nil {
			t.Fatal(err)
		}
		data := []byte{'A' + byte(i)}
		for j := 0; j < 100; j++ {
			pw.Write(data)
		}
		pw.Close()
	}

	destPath := filepath.Join(tmpDir, "merged.bin")
	if err := mergePartFiles(tmpDir, "merge-task", destPath); err != nil {
		t.Fatalf("merge failed: %v", err)
	}

	content, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(content) != 300 {
		t.Fatalf("expected 300 bytes, got %d", len(content))
	}
	// First 100 bytes should be 'A', next 100 'B', last 100 'C'
	for i := 0; i < 100; i++ {
		if content[i] != 'A' {
			t.Errorf("byte %d: expected 'A', got %c", i, content[i])
			break
		}
	}
	for i := 100; i < 200; i++ {
		if content[i] != 'B' {
			t.Errorf("byte %d: expected 'B', got %c", i, content[i])
			break
		}
	}
	for i := 200; i < 300; i++ {
		if content[i] != 'C' {
			t.Errorf("byte %d: expected 'C', got %c", i, content[i])
			break
		}
	}

	// Verify part files were cleaned up
	matches, _ := filepath.Glob(filepath.Join(tmpDir, "merge-task.part.*"))
	if len(matches) != 0 {
		t.Errorf("expected part files to be deleted, found %d", len(matches))
	}
}

func TestPartFileExists(t *testing.T) {
	tmpDir := t.TempDir()
	var downloaded int64

	pw, err := newPartWriter(tmpDir, "exist-task", 5, &downloaded)
	if err != nil {
		t.Fatal(err)
	}
	pw.Write([]byte("12345"))
	pw.Close()

	if !partFileExists(tmpDir, "exist-task", 5, 5) {
		t.Error("expected partFileExists to return true")
	}
	if partFileExists(tmpDir, "exist-task", 5, 10) {
		t.Error("expected partFileExists to return false for wrong size")
	}
	if partFileExists(tmpDir, "exist-task", 99, 5) {
		t.Error("expected partFileExists to return false for missing part")
	}
}
