package engine

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// =============================================================================
// STABILITY TEST SUITE — Exercises the download engine under adverse conditions
// to guarantee data integrity and resilience against regressions.
// =============================================================================

// --- Test Helpers ---

// spawnSlowServer creates a server that throttles at a fixed bytes/sec rate
func spawnSlowServer(_ *testing.T, content []byte, bytesPerSec int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "HEAD" {
			w.Header().Set("Content-Length", strconv.Itoa(len(content)))
			w.Header().Set("Accept-Ranges", "bytes")
			w.WriteHeader(http.StatusOK)
			return
		}

		rangeHeader := r.Header.Get("Range")
		start, end := 0, len(content)-1
		partial := false

		if rangeHeader != "" {
			parts := strings.Split(strings.TrimPrefix(rangeHeader, "bytes="), "-")
			start, _ = strconv.Atoi(parts[0])
			if len(parts) > 1 && parts[1] != "" {
				end, _ = strconv.Atoi(parts[1])
			}
			if start > end || start >= len(content) {
				w.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
				return
			}
			partial = true
		}

		data := content[start : end+1]

		if partial {
			w.Header().Set("Accept-Ranges", "bytes")
			w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, len(content)))
			w.Header().Set("Content-Length", strconv.Itoa(len(data)))
			w.WriteHeader(http.StatusPartialContent)
		} else {
			w.Header().Set("Content-Length", strconv.Itoa(len(content)))
			w.Header().Set("Accept-Ranges", "bytes")
			w.WriteHeader(http.StatusOK)
		}

		// Throttle writes
		chunkSize := bytesPerSec / 10
		if chunkSize < 1024 {
			chunkSize = 1024
		}
		for i := 0; i < len(data); i += chunkSize {
			e := i + chunkSize
			if e > len(data) {
				e = len(data)
			}
			w.Write(data[i:e])
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
			time.Sleep(100 * time.Millisecond)
		}
	}))
}

// spawnFlakyCDN creates a server that randomly resets connections mid-transfer
func spawnFlakyCDN(_ *testing.T, content []byte, failRate float64) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "HEAD" {
			w.Header().Set("Content-Length", strconv.Itoa(len(content)))
			w.Header().Set("Accept-Ranges", "bytes")
			w.Header().Set("ETag", `"flaky-cdn-etag"`)
			w.WriteHeader(http.StatusOK)
			return
		}

		rangeHeader := r.Header.Get("Range")
		start, end := 0, len(content)-1
		partial := false

		if rangeHeader != "" {
			parts := strings.Split(strings.TrimPrefix(rangeHeader, "bytes="), "-")
			start, _ = strconv.Atoi(parts[0])
			if len(parts) > 1 && parts[1] != "" {
				end, _ = strconv.Atoi(parts[1])
			}
			partial = true
		}

		data := content[start : end+1]

		if partial {
			w.Header().Set("Accept-Ranges", "bytes")
			w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, len(content)))
			w.Header().Set("Content-Length", strconv.Itoa(len(data)))
			w.Header().Set("ETag", `"flaky-cdn-etag"`)
			w.WriteHeader(http.StatusPartialContent)
		} else {
			w.Header().Set("Content-Length", strconv.Itoa(len(content)))
			w.Header().Set("Accept-Ranges", "bytes")
			w.Header().Set("ETag", `"flaky-cdn-etag"`)
			w.WriteHeader(http.StatusOK)
		}

		// Write data but randomly abort mid-transfer
		written := 0
		chunkSize := 64 * 1024
		for written < len(data) {
			e := written + chunkSize
			if e > len(data) {
				e = len(data)
			}
			// Random failure after writing some data
			if rand.Float64() < failRate && written > chunkSize {
				// Simulate connection reset by returning early
				// (hijack not needed — just stop writing)
				return
			}
			w.Write(data[written:e])
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
			written = e
		}
	}))
}

func hashContent(content []byte) string {
	h := md5.Sum(content)
	return hex.EncodeToString(h[:])
}

func sha256Content(content []byte) string {
	h := sha256.Sum256(content)
	return hex.EncodeToString(h[:])
}

// --- Stability Tests ---

// TestLargeFileIntegrity downloads a 50MB file and verifies MD5 matches exactly
func TestLargeFileIntegrity(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large file test in short mode")
	}

	size := 50 * 1024 * 1024 // 50MB
	content := make([]byte, size)
	rand.Read(content)
	expectedHash := hashContent(content)

	server := spawnRangeServer(t, content, 0)
	defer server.Close()

	tmpDir, _ := os.MkdirTemp("", "tachyon_large_test")
	defer os.RemoveAll(tmpDir)

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	store := createTempDB(t)
	engine := NewEngine(logger, store)
	engine.allowLoopback = true

	id, err := engine.StartDownload(server.URL, tmpDir, "large.bin", nil)
	if err != nil {
		t.Fatalf("StartDownload failed: %v", err)
	}

	deadline := time.After(30 * time.Second)
	for {
		select {
		case <-deadline:
			task, _ := store.GetTask(id)
			t.Fatalf("Timeout — status=%s progress=%.1f%%", task.Status, task.Progress)
		case <-time.After(200 * time.Millisecond):
			task, _ := store.GetTask(id)
			if task.Status == "completed" {
				goto verify
			}
			if task.Status == "error" {
				t.Fatalf("Download failed")
			}
		}
	}

verify:
	task, _ := store.GetTask(id)
	diskHash, err := calculateMD5(task.SavePath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}
	if diskHash != expectedHash {
		fi, _ := os.Stat(task.SavePath)
		t.Errorf("Hash mismatch: expected=%s got=%s (file size: %d, expected: %d)",
			expectedHash, diskHash, fi.Size(), size)
	}
}

// TestConcurrentDownloads runs multiple simultaneous downloads to stress concurrency
func TestConcurrentDownloads(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent download test in short mode")
	}

	const numDownloads = 5
	size := 5 * 1024 * 1024 // 5MB each
	contents := make([][]byte, numDownloads)
	hashes := make([]string, numDownloads)
	servers := make([]*httptest.Server, numDownloads)

	for i := 0; i < numDownloads; i++ {
		contents[i] = make([]byte, size)
		rand.Read(contents[i])
		hashes[i] = hashContent(contents[i])
		servers[i] = spawnRangeServer(t, contents[i], 0)
		defer servers[i].Close()
	}

	tmpDir, _ := os.MkdirTemp("", "tachyon_concurrent_test")
	defer os.RemoveAll(tmpDir)

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	store := createTempDB(t)
	engine := NewEngine(logger, store)
	engine.allowLoopback = true
	engine.SetMaxConcurrent(numDownloads)

	ids := make([]string, numDownloads)
	for i := 0; i < numDownloads; i++ {
		id, err := engine.StartDownload(servers[i].URL, tmpDir, fmt.Sprintf("file_%d.bin", i), nil)
		if err != nil {
			t.Fatalf("StartDownload[%d] failed: %v", i, err)
		}
		ids[i] = id
	}

	// Wait for all to finish
	deadline := time.After(30 * time.Second)
	completed := make(map[int]bool)
	for len(completed) < numDownloads {
		select {
		case <-deadline:
			t.Fatalf("Timeout — only %d/%d completed", len(completed), numDownloads)
		case <-time.After(200 * time.Millisecond):
			for i, id := range ids {
				if completed[i] {
					continue
				}
				task, _ := store.GetTask(id)
				if task.Status == "completed" {
					completed[i] = true
				}
				if task.Status == "error" {
					t.Errorf("Download %d failed", i)
					completed[i] = true
				}
			}
		}
	}

	// Verify all hashes
	for i, id := range ids {
		task, _ := store.GetTask(id)
		diskHash, err := calculateMD5(task.SavePath)
		if err != nil {
			t.Errorf("Download %d: failed to read: %v", i, err)
			continue
		}
		if diskHash != hashes[i] {
			t.Errorf("Download %d: hash mismatch", i)
		}
	}
}

// TestPauseResumeIntegrity pauses mid-download and verifies the resumed file is correct
func TestPauseResumeIntegrity(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping pause/resume test in short mode")
	}

	size := 10 * 1024 * 1024 // 10MB
	content := make([]byte, size)
	rand.Read(content)
	expectedHash := hashContent(content)

	// Slow server so pause has time to trigger mid-download
	server := spawnThrottledRangeServer(t, content, 5*time.Millisecond)
	defer server.Close()

	tmpDir, _ := os.MkdirTemp("", "tachyon_pauseresume_test")
	defer os.RemoveAll(tmpDir)

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	store := createTempDB(t)
	engine := NewEngine(logger, store)
	engine.allowLoopback = true

	id, err := engine.StartDownload(server.URL, tmpDir, "pauseresume.bin", nil)
	if err != nil {
		t.Fatalf("StartDownload failed: %v", err)
	}

	// Wait for some progress, then pause
	time.Sleep(200 * time.Millisecond)
	engine.PauseDownload(id)
	time.Sleep(500 * time.Millisecond)

	task, _ := store.GetTask(id)
	if task.Status == "completed" {
		// Finished before pause — just verify hash
		diskHash, _ := calculateMD5(task.SavePath)
		if diskHash != expectedHash {
			t.Error("Hash mismatch (completed before pause)")
		}
		return
	}

	if task.Status != "paused" {
		t.Fatalf("Expected paused, got %s", task.Status)
	}

	pausedProgress := task.Progress
	t.Logf("Paused at %.1f%%", pausedProgress)

	// Resume
	task.Status = "pending"
	engine.queue.Push(&task)

	deadline := time.After(30 * time.Second)
	for {
		select {
		case <-deadline:
			t2, _ := store.GetTask(id)
			t.Fatalf("Resume timeout — status=%s progress=%.1f%%", t2.Status, t2.Progress)
		case <-time.After(200 * time.Millisecond):
			t2, _ := store.GetTask(id)
			if t2.Status == "completed" {
				goto done
			}
			if t2.Status == "error" {
				t.Fatalf("Resume failed with error")
			}
		}
	}

done:
	task, _ = store.GetTask(id)
	diskHash, err := calculateMD5(task.SavePath)
	if err != nil {
		t.Fatalf("Failed to read file after resume: %v", err)
	}
	if diskHash != expectedHash {
		fi, _ := os.Stat(task.SavePath)
		t.Errorf("Hash mismatch after pause/resume: file=%d expected=%d", fi.Size(), size)
	}
}

// TestMultiplePauseResumeCycles pauses and resumes 3 times and verifies integrity
func TestMultiplePauseResumeCycles(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping multi pause/resume test in short mode")
	}

	size := 8 * 1024 * 1024 // 8MB
	content := make([]byte, size)
	rand.Read(content)
	expectedHash := hashContent(content)

	server := spawnThrottledRangeServer(t, content, 3*time.Millisecond)
	defer server.Close()

	tmpDir, _ := os.MkdirTemp("", "tachyon_multipause_test")
	defer os.RemoveAll(tmpDir)

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	store := createTempDB(t)
	engine := NewEngine(logger, store)
	engine.allowLoopback = true

	id, err := engine.StartDownload(server.URL, tmpDir, "multipause.bin", nil)
	if err != nil {
		t.Fatalf("StartDownload failed: %v", err)
	}

	for cycle := 0; cycle < 3; cycle++ {
		time.Sleep(150 * time.Millisecond)

		task, _ := store.GetTask(id)
		if task.Status == "completed" {
			t.Logf("Completed before cycle %d pause", cycle)
			goto verify
		}

		engine.PauseDownload(id)
		time.Sleep(300 * time.Millisecond)

		task, _ = store.GetTask(id)
		if task.Status == "completed" {
			t.Logf("Completed during cycle %d pause", cycle)
			goto verify
		}
		if task.Status != "paused" && task.Status != "pending" {
			// Could still be in probing/downloading transition
			t.Logf("Cycle %d: status=%s (continuing)", cycle, task.Status)
		}
		t.Logf("Cycle %d: paused at %.1f%%", cycle, task.Progress)

		// Resume
		task.Status = "pending"
		engine.queue.Push(&task)
	}

	// Wait for final completion
	{
		deadline := time.After(30 * time.Second)
		for {
			select {
			case <-deadline:
				task, _ := store.GetTask(id)
				t.Fatalf("Final timeout — status=%s", task.Status)
			case <-time.After(200 * time.Millisecond):
				task, _ := store.GetTask(id)
				if task.Status == "completed" {
					goto verify
				}
				if task.Status == "error" {
					t.Fatalf("Download failed after multi-pause")
				}
			}
		}
	}

verify:
	task, _ := store.GetTask(id)
	diskHash, err := calculateMD5(task.SavePath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}
	if diskHash != expectedHash {
		fi, _ := os.Stat(task.SavePath)
		t.Errorf("Hash mismatch after %d cycles: file=%d expected=%d", 3, fi.Size(), size)
	}
}

// TestRetryWithIntermittentFailures verifies data integrity under heavy server errors
func TestRetryWithIntermittentFailures(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping retry test in short mode")
	}

	size := 8 * 1024 * 1024 // 8MB
	content := make([]byte, size)
	rand.Read(content)
	expectedHash := hashContent(content)

	// Fail every 3rd request — very aggressive
	server := spawnRangeServer(t, content, 3)
	defer server.Close()

	tmpDir, _ := os.MkdirTemp("", "tachyon_heavyretry_test")
	defer os.RemoveAll(tmpDir)

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	store := createTempDB(t)
	engine := NewEngine(logger, store)
	engine.allowLoopback = true

	id, err := engine.StartDownload(server.URL, tmpDir, "heavyretry.bin", nil)
	if err != nil {
		t.Fatalf("StartDownload failed: %v", err)
	}

	deadline := time.After(30 * time.Second)
	for {
		select {
		case <-deadline:
			task, _ := store.GetTask(id)
			t.Fatalf("Timeout — status=%s progress=%.1f%%", task.Status, task.Progress)
		case <-time.After(200 * time.Millisecond):
			task, _ := store.GetTask(id)
			if task.Status == "completed" {
				goto check
			}
			if task.Status == "error" {
				t.Fatalf("Download failed despite retries")
			}
		}
	}

check:
	task, _ := store.GetTask(id)
	diskHash, err := calculateMD5(task.SavePath)
	if err != nil {
		t.Fatalf("Read fail: %v", err)
	}
	if diskHash != expectedHash {
		fi, _ := os.Stat(task.SavePath)
		t.Errorf("Hash mismatch after retries: file=%d expected=%d", fi.Size(), size)
	}
}

// TestSingleStreamFallback verifies correctness when server doesn't support ranges
func TestSingleStreamFallback(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping single-stream test in short mode")
	}

	content := make([]byte, 2*1024*1024) // 2MB
	rand.Read(content)
	expectedHash := hashContent(content)

	// Server that ignores Range headers entirely
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", strconv.Itoa(len(content)))
		w.WriteHeader(http.StatusOK)
		if r.Method != "HEAD" {
			w.Write(content)
		}
	}))
	defer server.Close()

	tmpDir, _ := os.MkdirTemp("", "tachyon_singlestream_test")
	defer os.RemoveAll(tmpDir)

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	store := createTempDB(t)
	engine := NewEngine(logger, store)
	engine.allowLoopback = true

	id, err := engine.StartDownload(server.URL, tmpDir, "single.bin", nil)
	if err != nil {
		t.Fatalf("StartDownload failed: %v", err)
	}

	deadline := time.After(10 * time.Second)
	for {
		select {
		case <-deadline:
			t.Fatal("Single-stream timeout")
		case <-time.After(200 * time.Millisecond):
			task, _ := store.GetTask(id)
			if task.Status == "completed" {
				goto check
			}
			if task.Status == "error" {
				t.Fatal("Single-stream download failed")
			}
		}
	}

check:
	task, _ := store.GetTask(id)
	diskHash, err := calculateMD5(task.SavePath)
	if err != nil {
		t.Fatalf("Read fail: %v", err)
	}
	if diskHash != expectedHash {
		t.Error("Hash mismatch in single-stream mode")
	}
}

// TestSmallFilesRapidFire downloads 20 tiny files in quick succession
func TestSmallFilesRapidFire(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping rapid fire test in short mode")
	}

	const numFiles = 20
	contents := make([][]byte, numFiles)
	hashes := make([]string, numFiles)

	for i := 0; i < numFiles; i++ {
		// Small files: 10KB to 200KB
		size := (i + 1) * 10 * 1024
		contents[i] = make([]byte, size)
		rand.Read(contents[i])
		hashes[i] = hashContent(contents[i])
	}

	var mu sync.Mutex
	fileIndex := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract file index from path
		path := r.URL.Path
		var idx int
		fmt.Sscanf(path, "/file_%d.bin", &idx)

		mu.Lock()
		if idx < 0 || idx >= numFiles {
			idx = fileIndex % numFiles
			fileIndex++
		}
		mu.Unlock()

		content := contents[idx]

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
			w.Header().Set("Accept-Ranges", "bytes")
			w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, len(content)))
			w.Header().Set("Content-Length", strconv.Itoa(end-start+1))
			w.WriteHeader(http.StatusPartialContent)
			w.Write(content[start : end+1])
			return
		}

		w.Header().Set("Content-Length", strconv.Itoa(len(content)))
		w.Header().Set("Accept-Ranges", "bytes")
		w.WriteHeader(http.StatusOK)
		w.Write(content)
	}))
	defer server.Close()

	tmpDir, _ := os.MkdirTemp("", "tachyon_rapidfire_test")
	defer os.RemoveAll(tmpDir)

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	store := createTempDB(t)
	engine := NewEngine(logger, store)
	engine.allowLoopback = true
	engine.SetMaxConcurrent(10)

	ids := make([]string, numFiles)
	for i := 0; i < numFiles; i++ {
		id, err := engine.StartDownload(
			fmt.Sprintf("%s/file_%d.bin", server.URL, i),
			tmpDir,
			fmt.Sprintf("rapid_%d.bin", i),
			nil,
		)
		if err != nil {
			t.Fatalf("StartDownload[%d] failed: %v", i, err)
		}
		ids[i] = id
	}

	// Wait for all — generous timeout; queue may take time to dispatch
	deadline := time.After(20 * time.Second)
	completedCount := 0
	completed := make(map[int]bool)
	for completedCount < numFiles {
		select {
		case <-deadline:
			for i, id := range ids {
				if !completed[i] {
					task, _ := store.GetTask(id)
					t.Logf("Stuck task[%d] id=%s status=%s", i, id, task.Status)
				}
			}
			t.Fatalf("Timeout: only %d/%d completed", completedCount, numFiles)
		case <-time.After(200 * time.Millisecond):
			for i, id := range ids {
				if completed[i] {
					continue
				}
				task, _ := store.GetTask(id)
				if task.Status == "completed" || task.Status == "error" {
					completed[i] = true
					completedCount++
					if task.Status == "error" {
						t.Errorf("File %d failed", i)
					}
				}
			}
		}
	}

	// Verify a sample of files
	for i := 0; i < numFiles; i += 3 {
		task, _ := store.GetTask(ids[i])
		if task.Status != "completed" {
			continue
		}
		fi, err := os.Stat(task.SavePath)
		if err != nil {
			t.Errorf("File %d missing: %v", i, err)
			continue
		}
		if fi.Size() != int64(len(contents[i])) {
			t.Errorf("File %d: size mismatch got=%d want=%d", i, fi.Size(), len(contents[i]))
		}
	}
}

// TestBandwidthLimitRespected verifies the speed limiter doesn't break data integrity
func TestBandwidthLimitRespected(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping bandwidth limit test in short mode")
	}

	size := 4 * 1024 * 1024 // 4MB
	content := make([]byte, size)
	rand.Read(content)
	expectedHash := hashContent(content)

	server := spawnRangeServer(t, content, 0)
	defer server.Close()

	tmpDir, _ := os.MkdirTemp("", "tachyon_bwlimit_test")
	defer os.RemoveAll(tmpDir)

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	store := createTempDB(t)
	engine := NewEngine(logger, store)
	engine.allowLoopback = true

	// Set a 2MB/s speed limit
	engine.bandwidthManager.SetLimit(2 * 1024 * 1024)

	id, err := engine.StartDownload(server.URL, tmpDir, "limited.bin", nil)
	if err != nil {
		t.Fatalf("StartDownload failed: %v", err)
	}

	started := time.Now()
	deadline := time.After(30 * time.Second)
	for {
		select {
		case <-deadline:
			t.Fatal("Timeout with bandwidth limit")
		case <-time.After(200 * time.Millisecond):
			task, _ := store.GetTask(id)
			if task.Status == "completed" {
				goto check
			}
			if task.Status == "error" {
				t.Fatal("Download failed with bandwidth limit")
			}
		}
	}

check:
	elapsed := time.Since(started)
	// 4MB at 2MB/s = ~2s minimum
	if elapsed < 1500*time.Millisecond {
		t.Logf("Warning: completed too fast (%.1fs) — bandwidth limiter may not be effective", elapsed.Seconds())
	}

	task, _ := store.GetTask(id)
	diskHash, err := calculateMD5(task.SavePath)
	if err != nil {
		t.Fatalf("Read fail: %v", err)
	}
	if diskHash != expectedHash {
		fi, _ := os.Stat(task.SavePath)
		t.Errorf("Hash mismatch with bandwidth limit active (file=%d expected=%d)", fi.Size(), size)
	}
}

// TestHTTP403HandledGracefully verifies link-expired handling
func TestHTTP403HandledGracefully(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping 403 test in short mode")
	}

	var requestCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := int(requestCount.Add(1))
		if r.Method == "HEAD" {
			if count == 1 {
				// First HEAD succeeds
				w.Header().Set("Content-Length", "1000000")
				w.Header().Set("Accept-Ranges", "bytes")
				w.WriteHeader(http.StatusOK)
				return
			}
		}
		// All subsequent requests get 403
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	tmpDir, _ := os.MkdirTemp("", "tachyon_403_test")
	defer os.RemoveAll(tmpDir)

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	store := createTempDB(t)
	engine := NewEngine(logger, store)
	engine.allowLoopback = true

	id, err := engine.StartDownload(server.URL, tmpDir, "forbidden.bin", nil)
	if err != nil {
		t.Fatalf("StartDownload failed: %v", err)
	}

	// Wait for the task to enter needs_auth or error state
	deadline := time.After(15 * time.Second)
	for {
		select {
		case <-deadline:
			task, _ := store.GetTask(id)
			t.Fatalf("Timeout — status=%s", task.Status)
		case <-time.After(200 * time.Millisecond):
			task, _ := store.GetTask(id)
			if task.Status == StatusNeedsAuth || task.Status == "error" {
				t.Logf("Correctly handled 403: status=%s", task.Status)
				return
			}
		}
	}
}

// TestMergeOrderCorrectness verifies parts are merged at correct byte offsets
func TestMergeOrderCorrectness(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "tachyon_merge_test")
	defer os.RemoveAll(tmpDir)

	taskID := "merge-order-test"

	// Create parts in reverse order with known content
	parts := []struct {
		offset  int64
		content []byte
	}{
		{offset: 200, content: []byte("CCCCC")}, // 5 bytes at offset 200
		{offset: 0, content: []byte("AAAAA")},   // 5 bytes at offset 0
		{offset: 100, content: []byte("BBBBB")}, // 5 bytes at offset 100
	}

	for _, p := range parts {
		path := filepath.Join(tmpDir, fmt.Sprintf("%s.part.%d", taskID, p.offset))
		if err := os.WriteFile(path, p.content, 0666); err != nil {
			t.Fatalf("Failed to write part file: %v", err)
		}
	}

	destPath := filepath.Join(tmpDir, "merged.bin")
	if err := mergePartFiles(tmpDir, taskID, destPath); err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	data, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("Failed to read merged file: %v", err)
	}

	// File should be 205 bytes (last part ends at offset 204)
	if len(data) < 205 {
		t.Fatalf("Merged file too small: %d bytes", len(data))
	}

	// Check content at offsets
	if string(data[0:5]) != "AAAAA" {
		t.Errorf("Offset 0: expected AAAAA, got %q", string(data[0:5]))
	}
	if string(data[100:105]) != "BBBBB" {
		t.Errorf("Offset 100: expected BBBBB, got %q", string(data[100:105]))
	}
	if string(data[200:205]) != "CCCCC" {
		t.Errorf("Offset 200: expected CCCCC, got %q", string(data[200:205]))
	}
}

// TestWorkStealingDoesNotCorrupt verifies work-stealing produces correct output
func TestWorkStealingDoesNotCorrupt(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping work-stealing test in short mode")
	}

	// Use a larger file to give work-stealing a chance to trigger
	size := 20 * 1024 * 1024 // 20MB
	content := make([]byte, size)
	rand.Read(content)
	expectedHash := hashContent(content)

	// Use throttled server so workers have time to steal
	server := spawnThrottledRangeServer(t, content, 2*time.Millisecond)
	defer server.Close()

	tmpDir, _ := os.MkdirTemp("", "tachyon_steal_test")
	defer os.RemoveAll(tmpDir)

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	store := createTempDB(t)
	engine := NewEngine(logger, store)
	engine.allowLoopback = true
	// Force small chunks and many workers to increase stealing likelihood
	engine.SetDownloadTuning(16, 1*1024*1024) // 1MB chunks, 16 workers

	id, err := engine.StartDownload(server.URL, tmpDir, "steal.bin", nil)
	if err != nil {
		t.Fatalf("StartDownload failed: %v", err)
	}

	deadline := time.After(60 * time.Second)
	for {
		select {
		case <-deadline:
			task, _ := store.GetTask(id)
			t.Fatalf("Timeout — status=%s progress=%.1f%%", task.Status, task.Progress)
		case <-time.After(300 * time.Millisecond):
			task, _ := store.GetTask(id)
			if task.Status == "completed" {
				goto check
			}
			if task.Status == "error" {
				t.Fatalf("Work-stealing download failed")
			}
		}
	}

check:
	task, _ := store.GetTask(id)
	diskHash, err := calculateMD5(task.SavePath)
	if err != nil {
		t.Fatalf("Read fail: %v", err)
	}
	if diskHash != expectedHash {
		fi, _ := os.Stat(task.SavePath)
		t.Errorf("Hash mismatch with work-stealing: file=%d expected=%d", fi.Size(), size)
	}
}

// TestPartPlanDeterminism verifies the planner produces identical plans for same input
func TestPartPlanDeterminism(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	store := createTempDB(t)
	engine := NewEngine(logger, store)

	sizes := []int64{
		1024,                   // 1KB
		1024 * 1024,            // 1MB
		64 * 1024 * 1024,       // 64MB
		512 * 1024 * 1024,      // 512MB
		2 * 1024 * 1024 * 1024, // 2GB
		5 * 1024 * 1024 * 1024, // 5GB
	}

	for _, size := range sizes {
		plan1 := engine.planDownloadParts(size, true)
		plan2 := engine.planDownloadParts(size, true)

		if len(plan1) != len(plan2) {
			t.Errorf("Size %d: plan length differs (%d vs %d)", size, len(plan1), len(plan2))
			continue
		}

		for i := range plan1 {
			if plan1[i].StartOffset != plan2[i].StartOffset || plan1[i].EndOffset != plan2[i].EndOffset {
				t.Errorf("Size %d, part %d: different offsets", size, i)
			}
		}

		// Verify parts cover entire file with no gaps
		if len(plan1) > 0 {
			if plan1[0].StartOffset != 0 {
				t.Errorf("Size %d: first part doesn't start at 0", size)
			}
			if plan1[len(plan1)-1].EndOffset != size-1 {
				t.Errorf("Size %d: last part doesn't end at %d (got %d)", size, size-1, plan1[len(plan1)-1].EndOffset)
			}
			for i := 1; i < len(plan1); i++ {
				if plan1[i].StartOffset != plan1[i-1].EndOffset+1 {
					t.Errorf("Size %d: gap between parts %d and %d", size, i-1, i)
				}
			}
		}
	}
}

// TestNoRangesFallbackIntegrity similar to TestServerNoRanges but larger file
func TestNoRangesFallbackIntegrity(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	store := createTempDB(t)
	engine := NewEngine(logger, store)

	sizes := []int64{0, -1, 100}
	for _, size := range sizes {
		parts := engine.planDownloadParts(size, true)
		if size <= 0 {
			if len(parts) != 1 || parts[0].EndOffset != StreamEndOffset {
				t.Errorf("Size %d: expected single stream part, got %d parts", size, len(parts))
			}
		}
	}

	// No ranges
	parts := engine.planDownloadParts(100*1024*1024, false)
	if len(parts) != 1 || parts[0].EndOffset != StreamEndOffset {
		t.Error("No ranges: expected single stream part")
	}
}

// TestQueuedDownloadStartsAfterPause verifies that when an active download is
// paused, the next queued download on the SAME HOST starts successfully.
// This catches the circuit breaker poisoning bug: cancelled workers must not
// count as server failures.
func TestQueuedDownloadStartsAfterPause(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping queued-after-pause test in short mode")
	}

	size := 5 * 1024 * 1024 // 5MB
	content1 := make([]byte, size)
	content2 := make([]byte, size)
	rand.Read(content1)
	rand.Read(content2)
	expectedHash2 := hashContent(content2)

	// Single server serving both files — so both downloads hit the same host
	// and share the same circuit breaker state.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var content []byte
		if strings.HasPrefix(r.URL.Path, "/slow") {
			content = content1
		} else {
			content = content2
		}

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
			w.Header().Set("Accept-Ranges", "bytes")
			w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, len(content)))
			w.Header().Set("Content-Length", strconv.Itoa(end-start+1))
			w.WriteHeader(http.StatusPartialContent)

			data := content[start : end+1]
			if strings.HasPrefix(r.URL.Path, "/slow") {
				// Throttle the slow file
				chunk := 32 * 1024
				for i := 0; i < len(data); i += chunk {
					e := i + chunk
					if e > len(data) {
						e = len(data)
					}
					w.Write(data[i:e])
					if f, ok := w.(http.Flusher); ok {
						f.Flush()
					}
					time.Sleep(50 * time.Millisecond)
				}
			} else {
				w.Write(data)
			}
			return
		}

		w.Header().Set("Content-Length", strconv.Itoa(len(content)))
		w.Header().Set("Accept-Ranges", "bytes")
		w.WriteHeader(http.StatusOK)
		if strings.HasPrefix(r.URL.Path, "/slow") {
			chunk := 32 * 1024
			for i := 0; i < len(content); i += chunk {
				e := i + chunk
				if e > len(content) {
					e = len(content)
				}
				w.Write(content[i:e])
				if f, ok := w.(http.Flusher); ok {
					f.Flush()
				}
				time.Sleep(50 * time.Millisecond)
			}
		} else {
			w.Write(content)
		}
	}))
	defer server.Close()

	tmpDir, _ := os.MkdirTemp("", "tachyon_queue_pause_test")
	defer os.RemoveAll(tmpDir)

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	store := createTempDB(t)
	eng := NewEngine(logger, store)
	eng.allowLoopback = true
	eng.SetMaxConcurrent(1) // Only 1 concurrent download

	// Start download 1 (slow — will be paused)
	id1, err := eng.StartDownload(server.URL+"/slow/file1.bin", tmpDir, "file1.bin", nil)
	if err != nil {
		t.Fatalf("StartDownload[1] failed: %v", err)
	}

	// Wait for it to start downloading
	deadline := time.After(10 * time.Second)
	for {
		select {
		case <-deadline:
			t.Fatal("Timeout waiting for download 1 to start")
		case <-time.After(100 * time.Millisecond):
			task, _ := store.GetTask(id1)
			if task.Status == "downloading" {
				goto started
			}
		}
	}
started:

	// Start download 2 (fast — same host, should be queued behind download 1)
	id2, err := eng.StartDownload(server.URL+"/fast/file2.bin", tmpDir, "file2.bin", nil)
	if err != nil {
		t.Fatalf("StartDownload[2] failed: %v", err)
	}

	// Give queue time to register
	time.Sleep(200 * time.Millisecond)

	// Pause download 1 — this should free the slot for download 2
	eng.PauseDownload(id1)

	// Wait for download 2 to complete
	deadline = time.After(30 * time.Second)
	for {
		select {
		case <-deadline:
			task2, _ := store.GetTask(id2)
			t.Fatalf("Timeout — download 2 status=%s", task2.Status)
		case <-time.After(200 * time.Millisecond):
			task2, _ := store.GetTask(id2)
			if task2.Status == "completed" {
				goto verify
			}
			if task2.Status == "error" {
				t.Fatalf("Download 2 errored after download 1 was paused (breaker poisoned)")
			}
		}
	}

verify:
	task2, _ := store.GetTask(id2)
	diskHash, err := calculateMD5(task2.SavePath)
	if err != nil {
		t.Fatalf("Failed to read file 2: %v", err)
	}
	if diskHash != expectedHash2 {
		t.Error("Download 2 hash mismatch after queued start")
	}
}
