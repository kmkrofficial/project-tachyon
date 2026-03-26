package app

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"project-tachyon/internal/config"
	"project-tachyon/internal/engine"
	"project-tachyon/internal/security"
	"project-tachyon/internal/storage"
)

// newTestApp creates a minimal App with real Storage and Engine for testing.
// It returns the App, a cleanup function, and the temp directory used.
func newTestApp(t *testing.T) (*App, func()) {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	store, err := storage.NewStorageWithPath(dbPath)
	if err != nil {
		t.Fatal("failed to create storage:", err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	eng := engine.NewEngine(logger, store)
	cfg := config.NewConfigManager(store)
	audit := security.NewAuditLogger(logger)

	app := NewApp(logger, eng, nil, cfg, audit)
	app.ctx = context.Background()

	// Note: engine context is NOT set to avoid Wails runtime.EventsEmit panics.
	// Methods that need to emit events will check for nil ctx.

	cleanup := func() {
		audit.Close()
		eng.Shutdown()
		store.Close()
	}
	return app, cleanup
}

func TestNewApp(t *testing.T) {
	a, cleanup := newTestApp(t)
	defer cleanup()

	if a == nil {
		t.Fatal("NewApp returned nil")
	}
	if a.engine == nil {
		t.Error("engine is nil")
	}
	if a.cfg == nil {
		t.Error("config manager is nil")
	}
	if a.audit == nil {
		t.Error("audit logger is nil")
	}
	if a.isQuitting {
		t.Error("isQuitting should be false initially")
	}
}

func TestStartup_SetsContext(t *testing.T) {
	a, cleanup := newTestApp(t)
	defer cleanup()

	// Reset context to verify Startup sets it
	a.ctx = nil
	ctx := context.Background()
	a.Startup(ctx)

	if a.ctx != ctx {
		t.Error("Startup did not set context")
	}
}

func TestVerifyFileExists_EmptyPath(t *testing.T) {
	a, cleanup := newTestApp(t)
	defer cleanup()

	if a.VerifyFileExists("") {
		t.Error("empty path should return false")
	}
}

func TestVerifyFileExists_NonExistent(t *testing.T) {
	a, cleanup := newTestApp(t)
	defer cleanup()

	if a.VerifyFileExists(filepath.Join(t.TempDir(), "nonexistent.file")) {
		t.Error("nonexistent file should return false")
	}
}

func TestVerifyFileExists_ExistingFile(t *testing.T) {
	a, cleanup := newTestApp(t)
	defer cleanup()

	f := filepath.Join(t.TempDir(), "exists.txt")
	os.WriteFile(f, []byte("test"), 0644)

	if !a.VerifyFileExists(f) {
		t.Error("existing file should return true")
	}
}

func TestGetQueuedDownloads_Empty(t *testing.T) {
	a, cleanup := newTestApp(t)
	defer cleanup()

	items := a.GetQueuedDownloads()
	if len(items) != 0 {
		t.Errorf("expected 0 queued downloads, got %d", len(items))
	}
}

func TestGetTasks_Empty(t *testing.T) {
	a, cleanup := newTestApp(t)
	defer cleanup()

	tasks := a.GetTasks()
	if len(tasks) != 0 {
		t.Errorf("expected 0 tasks, got %d", len(tasks))
	}
}

func TestGetNetworkHealth_NoActiveDownloads(t *testing.T) {
	a, cleanup := newTestApp(t)
	defer cleanup()

	health := a.GetNetworkHealth()
	if health.Level != "normal" {
		t.Errorf("expected 'normal' with no active downloads, got %q", health.Level)
	}
}

// --- Security bridge methods ---

func TestGetSetEnableAI(t *testing.T) {
	a, cleanup := newTestApp(t)
	defer cleanup()

	// Default should be false
	if a.GetEnableAI() {
		t.Error("AI should be disabled by default")
	}

	a.SetEnableAI(true)
	if !a.GetEnableAI() {
		t.Error("AI should be enabled after SetEnableAI(true)")
	}

	a.SetEnableAI(false)
	if a.GetEnableAI() {
		t.Error("AI should be disabled after SetEnableAI(false)")
	}
}

func TestGetSetAIPort(t *testing.T) {
	a, cleanup := newTestApp(t)
	defer cleanup()

	a.SetAIPort(9999)
	if got := a.GetAIPort(); got != 9999 {
		t.Errorf("expected port 9999, got %d", got)
	}
}

func TestGetSetAIMaxConcurrent(t *testing.T) {
	a, cleanup := newTestApp(t)
	defer cleanup()

	a.SetAIMaxConcurrent(10)
	if got := a.GetAIMaxConcurrent(); got != 10 {
		t.Errorf("expected 10, got %d", got)
	}
}

func TestGetRecentAuditLogs_Empty(t *testing.T) {
	a, cleanup := newTestApp(t)
	defer cleanup()

	logs := a.GetRecentAuditLogs()
	if logs == nil {
		t.Error("audit logs should not be nil")
	}
}

func TestGetRecentAuditLogs_NilAudit(t *testing.T) {
	a, cleanup := newTestApp(t)
	defer cleanup()
	a.audit = nil

	logs := a.GetRecentAuditLogs()
	if logs == nil {
		t.Error("should return empty slice, not nil")
	}
	if len(logs) != 0 {
		t.Errorf("expected 0 logs, got %d", len(logs))
	}
}

// --- Engine delegation tests ---

func TestSetGetUserAgent(t *testing.T) {
	a, cleanup := newTestApp(t)
	defer cleanup()

	a.SetUserAgent("TestAgent/1.0")
	if got := a.GetUserAgent(); got != "TestAgent/1.0" {
		t.Errorf("expected TestAgent/1.0, got %q", got)
	}
}

func TestSetGetHostLimit(t *testing.T) {
	a, cleanup := newTestApp(t)
	defer cleanup()

	a.SetHostLimit("example.com", 5)
	if got := a.GetHostLimit("example.com"); got != 5 {
		t.Errorf("expected 5, got %d", got)
	}
}

func TestSetGlobalSpeedLimit(t *testing.T) {
	a, cleanup := newTestApp(t)
	defer cleanup()

	// Should not panic
	a.SetGlobalSpeedLimit(1024 * 1024)
}

func TestSetMaxConcurrentDownloads(t *testing.T) {
	a, cleanup := newTestApp(t)
	defer cleanup()

	// Should not panic
	a.SetMaxConcurrentDownloads(3)
}

func TestGetDownloadLocations_Empty(t *testing.T) {
	a, cleanup := newTestApp(t)
	defer cleanup()

	locs := a.GetDownloadLocations()
	if locs == nil {
		t.Error("locations should not be nil")
	}
}

func TestAddDownloadLocation(t *testing.T) {
	a, cleanup := newTestApp(t)
	defer cleanup()

	a.AddDownloadLocation(t.TempDir(), "test-location")
	locs := a.GetDownloadLocations()
	if len(locs) != 1 {
		t.Fatalf("expected 1 location, got %d", len(locs))
	}
	if locs[0].Nickname != "test-location" {
		t.Errorf("expected nickname 'test-location', got %q", locs[0].Nickname)
	}
}

// --- Download operations ---

func TestAddDownload_InvalidURL(t *testing.T) {
	a, cleanup := newTestApp(t)
	defer cleanup()

	result := a.AddDownload("not-a-valid-url")
	if result == "" {
		t.Error("expected non-empty result for invalid URL")
	}
}

func TestPauseDownload_NonExistent(t *testing.T) {
	a, cleanup := newTestApp(t)
	defer cleanup()

	// Should not panic when pausing non-existent download
	a.PauseDownload("nonexistent-id")
}

func TestResumeDownload_NonExistent(t *testing.T) {
	a, cleanup := newTestApp(t)
	defer cleanup()

	err := a.ResumeDownload("nonexistent-id")
	if err == nil {
		t.Error("expected error when resuming non-existent download")
	}
}

func TestStopDownload_NonExistent(t *testing.T) {
	a, cleanup := newTestApp(t)
	defer cleanup()

	// Should not panic
	a.StopDownload("nonexistent-id")
}

func TestDeleteDownload_NonExistent(t *testing.T) {
	a, cleanup := newTestApp(t)
	defer cleanup()

	// Should not panic
	a.DeleteDownload("nonexistent-id", false)
}

func TestPauseAllDownloads(t *testing.T) {
	a, cleanup := newTestApp(t)
	defer cleanup()

	// Should not panic with no active downloads
	a.PauseAllDownloads()
}

func TestResumeAllDownloads(t *testing.T) {
	a, cleanup := newTestApp(t)
	defer cleanup()

	// Should not panic with no active downloads
	a.ResumeAllDownloads()
}

func TestGetLifetimeStats(t *testing.T) {
	a, cleanup := newTestApp(t)
	defer cleanup()

	stats := a.GetLifetimeStats()
	if stats < 0 {
		t.Errorf("lifetime stats should be >= 0, got %d", stats)
	}
}

func TestGetAnalytics(t *testing.T) {
	a, cleanup := newTestApp(t)
	defer cleanup()

	analytics := a.GetAnalytics()
	// Should return a valid struct with no panic
	_ = analytics
}

func TestGetDiskUsage(t *testing.T) {
	a, cleanup := newTestApp(t)
	defer cleanup()

	disk := a.GetDiskUsage()
	// TotalGB should be > 0 on any real system
	if disk.TotalGB == 0 {
		t.Log("disk TotalGB is 0 (may be expected in some CI environments)")
	}
}

func TestGetSpeedTestHistory_Empty(t *testing.T) {
	a, cleanup := newTestApp(t)
	defer cleanup()

	history := a.GetSpeedTestHistory()
	if history == nil {
		t.Error("speed test history should not be nil")
	}
	if len(history) != 0 {
		t.Errorf("expected 0 history entries, got %d", len(history))
	}
}

func TestUpdateSettings(t *testing.T) {
	a, cleanup := newTestApp(t)
	defer cleanup()

	a.UpdateSettings(`{"theme":"dark","maxConcurrent":"5"}`)
	// Verify settings were persisted
	val, err := a.engine.GetStorage().GetString("theme")
	if err != nil {
		t.Fatal("failed to get setting:", err)
	}
	if val != "dark" {
		t.Errorf("expected 'dark', got %q", val)
	}
}

func TestUpdateSettings_InvalidJSON(t *testing.T) {
	a, cleanup := newTestApp(t)
	defer cleanup()

	// Should not panic on invalid JSON
	a.UpdateSettings("not json")
}

func TestUpdateSettings_BoolAndNumericValues(t *testing.T) {
	a, cleanup := newTestApp(t)
	defer cleanup()

	a.UpdateSettings(`{"enabled":true,"count":42}`)

	val, _ := a.engine.GetStorage().GetString("enabled")
	if val != "true" {
		t.Errorf("expected 'true', got %q", val)
	}
	val, _ = a.engine.GetStorage().GetString("count")
	if val != "42" {
		t.Errorf("expected '42', got %q", val)
	}
}
