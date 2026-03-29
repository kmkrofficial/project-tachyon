package engine

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
	"sync/atomic"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// --- Helper Functions ---

// createTempDB creates a file-backed SQLite DB for testing.
// Using a temp file instead of :memory: so that all goroutines share the same tables.
func createTempDB(t *testing.T) *storage.Storage {
	dir, err := os.MkdirTemp("", "tachyon_testdb_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	dbPath := filepath.Join(dir, "test.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open test db: %v", err)
	}
	// Migrate all tables the engine may touch
	if err := db.AutoMigrate(&storage.DownloadTask{}, &storage.DownloadLocation{}, &storage.DailyStat{}, &storage.AppSetting{}, &storage.SpeedTestHistory{}); err != nil {
		t.Fatalf("Migration failed: %v", err)
	}
	s := &storage.Storage{DB: db}
	t.Cleanup(func() {
		s.Close()
		os.RemoveAll(dir)
	})
	return s
}

// spawnRangeServer creates a mock HTTP server supporting Range requests
func spawnRangeServer(_ *testing.T, content []byte, errorEveryN int) *httptest.Server {
	var requestCount atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := int(requestCount.Add(1))

		// Simulate Random Failures
		if errorEveryN > 0 && count%errorEveryN == 0 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// Handle HEAD (Probe) - Note: Our engine uses GET with Range for probing
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

			w.Header().Set("Accept-Ranges", "bytes")
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
		w.Header().Set("Accept-Ranges", "bytes")
		w.WriteHeader(http.StatusOK)
		w.Write(content)
	}))

	return server
}

// spawnThrottledRangeServer creates a mock HTTP server that adds a delay per write
// chunk so that downloads take long enough to be paused mid-flight.
func spawnThrottledRangeServer(_ *testing.T, content []byte, chunkDelay time.Duration) *httptest.Server {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "HEAD" {
			w.Header().Set("Content-Length", strconv.Itoa(len(content)))
			w.Header().Set("Accept-Ranges", "bytes")
			w.WriteHeader(http.StatusOK)
			return
		}

		rangeHeader := r.Header.Get("Range")
		if rangeHeader != "" {
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

			w.Header().Set("Accept-Ranges", "bytes")
			w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, len(content)))
			w.Header().Set("Content-Length", strconv.Itoa(end-start+1))
			w.WriteHeader(http.StatusPartialContent)

			// Write in 64KB chunks with delay between each
			data := content[start : end+1]
			chunkSize := 64 * 1024
			for i := 0; i < len(data); i += chunkSize {
				e := i + chunkSize
				if e > len(data) {
					e = len(data)
				}
				w.Write(data[i:e])
				if f, ok := w.(http.Flusher); ok {
					f.Flush()
				}
				time.Sleep(chunkDelay)
			}
			return
		}

		w.Header().Set("Content-Length", strconv.Itoa(len(content)))
		w.Header().Set("Accept-Ranges", "bytes")
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
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

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
	engine.allowLoopback = true
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
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	size := 5 * 1024 * 1024
	content := generateDummyContent(size)
	expectedHashStr := hex.EncodeToString(md5.New().Sum(nil)) // Placeholder, computed below
	expectedHash := md5.Sum(content)
	expectedHashStr = hex.EncodeToString(expectedHash[:])

	server := spawnRangeServer(t, content, 0)
	defer server.Close()

	tmpDir, _ := os.MkdirTemp("", "tachyon_resume_test")
	defer os.RemoveAll(tmpDir)

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	store := createTempDB(t)
	engine := NewEngine(logger, store)
	engine.allowLoopback = true

	// 1. Start
	id, _ := engine.StartDownload(server.URL, tmpDir, "resume.bin", nil)

	// 2. Pause immediately (may race with completion for fast downloads)
	time.Sleep(20 * time.Millisecond)
	engine.PauseDownload(id)
	time.Sleep(200 * time.Millisecond)

	task, _ := store.GetTask(id)

	// 3. Verify: either paused or already completed
	if task.Status == "completed" {
		t.Log("Download completed before pause — verifying integrity")
		diskHash, err := calculateMD5(task.SavePath)
		if err != nil {
			t.Fatalf("Failed to read file: %v", err)
		}
		if diskHash != expectedHashStr {
			t.Error("Completed file has wrong hash")
		}
		return
	}

	if task.Status != "paused" {
		t.Fatalf("Expected paused or completed, got %s", task.Status)
	}

	// 4. Resume by re-queuing
	task.Status = "pending"
	engine.queue.Push(&task)

	// Wait for completion
	timeout := time.After(15 * time.Second)
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

	// 5. Verify integrity
	task, _ = store.GetTask(id)
	diskHash, err := calculateMD5(task.SavePath)
	if err != nil {
		t.Fatalf("Failed to read final file: %v", err)
	}
	if diskHash != expectedHashStr {
		fi, _ := os.Stat(task.SavePath)
		t.Errorf("Resume resulted in corrupted file: expected size %d, got %d", size, fi.Size())
	}
}

func TestNetworkFailureAndRetry(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

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
	engine.allowLoopback = true

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
	diskHash, err := calculateMD5(task.SavePath)
	if err != nil {
		t.Fatalf("Failed to read final file: %v", err)
	}
	expectedHash := md5.Sum(content)
	if diskHash != hex.EncodeToString(expectedHash[:]) {
		fi, _ := os.Stat(task.SavePath)
		t.Errorf("File corrupted after retries: expected size %d, got %d, path %s", len(content), fi.Size(), task.SavePath)
	}
}

func TestServerNoRanges(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	content := []byte("Simulated Single Thread Content")

	// Mock Server that sends 200 OK and ignores ranges
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Always set Content-Length header for size detection
		w.Header().Set("Content-Length", strconv.Itoa(len(content)))
		// NO Accept-Ranges header - simulate server that doesn't support ranges
		w.WriteHeader(http.StatusOK)
		if r.Method != "HEAD" {
			w.Write(content)
		}
	}))
	defer server.Close()

	tmpDir, _ := os.MkdirTemp("", "tachyon_norange_test")
	defer os.RemoveAll(tmpDir)

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	store := createTempDB(t)
	engine := NewEngine(logger, store)
	engine.allowLoopback = true

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
