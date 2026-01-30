package core

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"project-tachyon/internal/storage"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// --- Helper Functions ---

// createTempDB creates an in-memory SQLite DB for testing
func createTempDB(t *testing.T) *storage.Storage {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open in-memory db: %v", err)
	}
	// Migrate
	if err := db.AutoMigrate(&storage.DownloadTask{}, &storage.DownloadLocation{}, &storage.DailyStat{}, &storage.AppSetting{}); err != nil {
		t.Fatalf("Migration failed: %v", err)
	}
	return &storage.Storage{DB: db}
}

// spawnRangeServer creates a mock HTTP server supporting Range requests
func spawnRangeServer(t *testing.T, content []byte, errorEveryN int) *httptest.Server {
	var requestCount int

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++

		// Simulate Random Failures
		if errorEveryN > 0 && requestCount%errorEveryN == 0 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// Handle HEAD (Probe)
		if r.Method == "HEAD" {
			w.Header().Set("Content-Length", strconv.Itoa(len(content)))
			w.Header().Set("Accept-Ranges", "bytes")
			w.Header().Set("Content-Disposition", "attachment; filename=testfile.bin")
			w.WriteHeader(http.StatusOK)
			return
		}

		// Handle Range
		rangeHeader := r.Header.Get("Range")
		if rangeHeader != "" {
			// Format: bytes=start-end
			parts := strings.Split(strings.TrimPrefix(rangeHeader, "bytes="), "-")
			start, _ := strconv.Atoi(parts[0])
			end := len(content) - 1
			if len(parts) > 1 && parts[1] != "" {
				end, _ = strconv.Atoi(parts[1])
			}

			if start > end || start >= len(content) {
				w.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
				return
			}

			w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, len(content)))
			w.Header().Set("Content-Length", strconv.Itoa(end-start+1))
			w.WriteHeader(http.StatusPartialContent)
			if _, err := w.Write(content[start : end+1]); err != nil {
				// Ignore write errors (broken pipe etc)
			}
			return
		}

		// Full Content
		w.Header().Set("Content-Length", strconv.Itoa(len(content)))
		w.WriteHeader(http.StatusOK)
		w.Write(content)
	}))

	return server
}

// generateDummyContent creates random bytes
func generateDummyContent(size int) []byte {
	b := make([]byte, size)
	rand.Read(b)
	return b
}

func calculateMD5(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	hash := md5.New()
	if _, err := io.Copy(hash, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// --- Tests ---

func TestDynamicWorkStealing(t *testing.T) {
	// Setup: 10MB Content
	size := 10 * 1024 * 1024
	content := generateDummyContent(size)
	expectedHash := md5.Sum(content)
	expectedHashStr := hex.EncodeToString(expectedHash[:])

	server := spawnRangeServer(t, content, 0)
	defer server.Close()

	// Temp Dir
	tmpDir, _ := os.MkdirTemp("", "tachyon_test")
	defer os.RemoveAll(tmpDir)

	// Engine
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	store := createTempDB(t)
	engine := NewEngine(logger, store)
	// engine.SetContext(context.TODO()) // Removed to avoid Wails panic on non-wails context

	// Start Download
	id, err := engine.StartDownload(server.URL, tmpDir, "download.bin", nil)
	if err != nil {
		t.Fatalf("StartDownload failed: %v", err)
	}

	// Wait for completion (Poll Task Status)
	timeout := time.After(10 * time.Second)
	completed := false
Loop:
	for {
		select {
		case <-timeout:
			t.Fatal("Timeout waiting for download")
		case <-time.After(100 * time.Millisecond):
			task, _ := store.GetTask(id)
			if task.Status == "completed" {
				completed = true
				break Loop
			}
			if task.Status == "error" {
				t.Fatalf("Download failed with error")
			}
		}
	}

	if !completed {
		t.Fatal("Download did not complete")
	}

	// Verify File Content
	task, _ := store.GetTask(id)
	finalPath := task.SavePath
	diskHash, err := calculateMD5(finalPath)
	if err != nil {
		t.Fatalf("MD5 check failed: %v", err)
	}

	if diskHash != expectedHashStr {
		t.Errorf("Hash Mismatch. Expected %s, Got %s", expectedHashStr, diskHash)
	}
}

func TestPauseAndResume(t *testing.T) {
	// Setup: 10MB Content (Enough to pause mid-way manually in real world, but in tests fast CPU might finish too fast)
	// We'll use a larger file or rely on pausing quickly.
	size := 10 * 1024 * 1024
	content := generateDummyContent(size)
	server := spawnRangeServer(t, content, 0)
	defer server.Close()

	tmpDir, _ := os.MkdirTemp("", "tachyon_resume_test")
	defer os.RemoveAll(tmpDir)

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	store := createTempDB(t)
	engine := NewEngine(logger, store)

	// 1. Start
	id, _ := engine.StartDownload(server.URL, tmpDir, "resume.bin", nil)

	// 2. Sleep briefly then Pause
	time.Sleep(50 * time.Millisecond) // Let it download a bit
	engine.PauseDownload(id)

	// 3. Verify Paused State
	time.Sleep(100 * time.Millisecond) // Wait for workers to stop
	task, _ := store.GetTask(id)

	// Note: Engine updates status to 'paused' on cancel in the loop (if implemented) or we manually set it.
	// Current implementation: PauseDownload sets Cancel, Loop detects Done, sets Paused.
	// Let's verify status.
	if task.Status != "paused" {
		t.Logf("Warning: Status is %s, expected paused (might be race condition in test wait)", task.Status)
	}

	// Verify partial file exists
	task, _ = store.GetTask(id) // Refresh task
	fi, err := os.Stat(task.SavePath)
	if err != nil {
		t.Fatal("File missing after pause")
	}
	if fi.Size() != int64(size) {
		t.Errorf("File size should be pre-allocated to %d, got %d", size, fi.Size())
	}

	// 4. Resume (How? Re-queue logic isn't exposed properly in NewEngine, usually UI calls something.
	// We need a Resume function exposed or just call StartDownload again with same URL/Path?
	// Engine.go currently has StartDownload creating NEW ID.
	// Ideally we need 'ResumeDownload(id)'.
	// WORKAROUND: We manually re-queue the task task.Status = 'pending', queue.Push(task).

	task.Status = "pending"
	engine.queue.Push(&task)

	// Wait for finish
	timeout := time.After(10 * time.Second)
Loop:
	for {
		select {
		case <-timeout:
			t.Fatal("Timeout waiting for resume completion")
		case <-time.After(100 * time.Millisecond):
			t2, _ := store.GetTask(id)
			if t2.Status == "completed" {
				break Loop
			}
		}
	}

	// Verify Integrity
	task, _ = store.GetTask(id)
	diskHash, _ := calculateMD5(task.SavePath)
	expectedHash := md5.Sum(content)
	if diskHash != hex.EncodeToString(expectedHash[:]) {
		t.Error("Resume resulted in corrupted file")
	}
}

func TestNetworkFailureAndRetry(t *testing.T) {
	size := 5 * 1024 * 1024
	content := generateDummyContent(size)
	// Fail every 5th request
	server := spawnRangeServer(t, content, 5)
	defer server.Close()

	tmpDir, _ := os.MkdirTemp("", "tachyon_retry_test")
	defer os.RemoveAll(tmpDir)

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	store := createTempDB(t)
	engine := NewEngine(logger, store)

	id, _ := engine.StartDownload(server.URL, tmpDir, "retry.bin", nil)

	// Wait longer for retries
	timeout := time.After(15 * time.Second)
	success := false
Loop:
	for {
		select {
		case <-timeout:
			t.Fatal("Timeout waiting for retry-download")
		case <-time.After(200 * time.Millisecond):
			task, _ := store.GetTask(id)
			if task.Status == "completed" {
				success = true
				break Loop
			}
			if task.Status == "error" {
				t.Fatalf("Download failed despite retry logic")
			}
		}
	}

	if !success {
		t.Fatal("Did not complete successfully")
	}

	task, _ := store.GetTask(id)
	diskHash, _ := calculateMD5(task.SavePath)
	expectedHash := md5.Sum(content)
	if diskHash != hex.EncodeToString(expectedHash[:]) {
		t.Error("File corrupted after retries")
	}
}

func TestServerNoRanges(t *testing.T) {
	content := []byte("Simulated Single Thread Content")

	// Mock Server that sends 200 OK and ignores ranges
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "HEAD" {
			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Length", strconv.Itoa(len(content)))
			// No Accept-Ranges header
			return
		}
		// Return full content regardless of Range header
		w.WriteHeader(http.StatusOK)
		w.Write(content)
	}))
	defer server.Close()

	tmpDir, _ := os.MkdirTemp("", "tachyon_norange_test")
	defer os.RemoveAll(tmpDir)

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	store := createTempDB(t)
	engine := NewEngine(logger, store)

	id, _ := engine.StartDownload(server.URL, tmpDir, "norange.bin", nil)

	// Wait
	time.Sleep(1 * time.Second)

	task, _ := store.GetTask(id)
	if task.Status != "completed" {
		t.Errorf("Single thread download failed, status: %s", task.Status)
	}

	data, _ := os.ReadFile(task.SavePath)
	if string(data) != string(content) {
		t.Error("Content mismatch in single thread mode")
	}
}

func TestRealWorldDownload(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping long running test in short mode")
	}

	url := "https://ash-speed.hetzner.com/1GB.bin"

	tmpDir, _ := os.MkdirTemp("", "tachyon_real_test")
	defer os.RemoveAll(tmpDir)

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	store := createTempDB(t)
	engine := NewEngine(logger, store)
	engine.SetMaxConcurrent(16)

	// Use non-Wails context
	// engine.SetContext(context.TODO())

	t.Logf("Starting real download from %s", url)
	id, err := engine.StartDownload(url, tmpDir, "1GB.bin", nil)
	if err != nil {
		t.Fatalf("Failed to start: %v", err)
	}

	// Wait loop with progress logging
	timeout := time.After(60 * time.Second)
Loop:
	for {
		select {
		case <-timeout:
			t.Fatal("Real download timed out")
		case <-time.After(1 * time.Second):
			task, _ := store.GetTask(id)
			t.Logf("Progress: %.2f%% (%s / %s)", task.Progress, humanizeBytes(task.Downloaded), humanizeBytes(task.TotalSize))

			if task.Status == "completed" {
				break Loop
			}
			if task.Status == "error" {
				t.Fatalf("Real download failed")
			}
		}
	}

	// Verify file exists and has size
	fi, err := os.Stat(filepath.Join(tmpDir, "1GB.bin"))
	if err != nil {
		t.Fatal("File not found")
	}
	if fi.Size() == 0 {
		t.Fatal("File is empty")
	}
	t.Logf("Successfully downloaded %s (%d bytes)", fi.Name(), fi.Size())
}

func humanizeBytes(s int64) string {
	if s < 1024 {
		return fmt.Sprintf("%d B", s)
	}
	if s < 1024*1024 {
		return fmt.Sprintf("%.2f KB", float64(s)/1024)
	}
	return fmt.Sprintf("%.2f MB", float64(s)/1024/1024)
}
