package engine

import (
	"log/slog"
	"os"
	"testing"

	"project-tachyon/internal/storage"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func createTestDB(t *testing.T) *storage.Storage {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open in-memory db: %v", err)
	}
	if err := db.AutoMigrate(
		&storage.DownloadTask{},
		&storage.DownloadLocation{},
		&storage.DailyStat{},
		&storage.AppSetting{},
		&storage.SpeedTestHistory{},
	); err != nil {
		t.Fatalf("Migration failed: %v", err)
	}
	return &storage.Storage{DB: db}
}

func TestNewEngine(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	s := createTestDB(t)

	e := NewEngine(logger, s)
	if e == nil {
		t.Fatal("NewEngine returned nil")
	}

	if e.maxConcurrent != 5 {
		t.Errorf("maxConcurrent = %d, want 5", e.maxConcurrent)
	}
	if e.runningDownloads != 0 {
		t.Errorf("runningDownloads = %d, want 0", e.runningDownloads)
	}
	if e.maxWorkersPerTask != MaxWorkersPerTask {
		t.Errorf("maxWorkersPerTask = %d, want %d", e.maxWorkersPerTask, MaxWorkersPerTask)
	}
}

func TestSetDownloadTuning(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	s := createTestDB(t)
	e := NewEngine(logger, s)

	// Normal tuning
	e.SetDownloadTuning(8, 1024*1024)
	if e.maxWorkersPerTask != 8 {
		t.Errorf("maxWorkersPerTask after tuning = %d, want 8", e.maxWorkersPerTask)
	}
	if e.baseChunkSize != 1024*1024 {
		t.Errorf("baseChunkSize = %d, want %d", e.baseChunkSize, 1024*1024)
	}

	// Clamp to minimum
	e.SetDownloadTuning(0, -1)
	if e.maxWorkersPerTask != 1 {
		t.Errorf("maxWorkersPerTask should clamp to 1, got %d", e.maxWorkersPerTask)
	}
	if e.baseChunkSize != 0 {
		t.Errorf("baseChunkSize should clamp to 0, got %d", e.baseChunkSize)
	}

	// Clamp to maximum
	e.SetDownloadTuning(100, 0)
	if e.maxWorkersPerTask != 64 {
		t.Errorf("maxWorkersPerTask should clamp to 64, got %d", e.maxWorkersPerTask)
	}
}

func TestConstants(t *testing.T) {
	if DownloadChunkSize != 4*1024*1024 {
		t.Errorf("DownloadChunkSize = %d, want %d", DownloadChunkSize, 4*1024*1024)
	}
	if BufferSize != 1*1024*1024 {
		t.Errorf("BufferSize = %d, want %d", BufferSize, 1*1024*1024)
	}
	if MaxWorkersPerTask != 24 {
		t.Errorf("MaxWorkersPerTask = %d, want 24", MaxWorkersPerTask)
	}
	if GenericUserAgent == "" {
		t.Error("GenericUserAgent should not be empty")
	}
}

func TestRecoverInterruptedDownloads_AbruptClose(t *testing.T) {
	s := createTestDB(t)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	e := NewEngine(logger, s)

	// Simulate abrupt close: downloads stuck in "downloading" and "pending"
	s.SaveTask(storage.DownloadTask{ID: "d1", Status: "downloading", Downloaded: 5000, Progress: 50})
	s.SaveTask(storage.DownloadTask{ID: "d2", Status: "pending"})
	s.SaveTask(storage.DownloadTask{ID: "d3", Status: "paused"})    // manually paused
	s.SaveTask(storage.DownloadTask{ID: "d4", Status: "error"})     // was in error
	s.SaveTask(storage.DownloadTask{ID: "d5", Status: "completed"}) // done

	e.RecoverInterruptedDownloads()

	// d1 and d2 should be moved to paused first (then queued for resume via goroutine)
	t1, _ := s.GetTask("d1")
	t2, _ := s.GetTask("d2")
	if t1.Status != "paused" {
		t.Errorf("d1 status = %q, want paused", t1.Status)
	}
	// d1 should retain its downloaded bytes
	if t1.Downloaded != 5000 {
		t.Errorf("d1 downloaded = %d, want 5000", t1.Downloaded)
	}
	if t2.Status != "paused" {
		t.Errorf("d2 status = %q, want paused", t2.Status)
	}

	// d3/d4/d5 should NOT be touched
	t3, _ := s.GetTask("d3")
	t4, _ := s.GetTask("d4")
	t5, _ := s.GetTask("d5")
	if t3.Status != "paused" {
		t.Errorf("d3 status = %q, want paused (unchanged)", t3.Status)
	}
	if t4.Status != "error" {
		t.Errorf("d4 status = %q, want error (unchanged)", t4.Status)
	}
	if t5.Status != "completed" {
		t.Errorf("d5 status = %q, want completed (unchanged)", t5.Status)
	}
}

func TestRecoverInterruptedDownloads_GracefulShutdown(t *testing.T) {
	s := createTestDB(t)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	e := NewEngine(logger, s)

	// Simulate graceful shutdown: downloads saved as paused, auto_resume_ids set
	s.SaveTask(storage.DownloadTask{ID: "g1", Status: "paused", Downloaded: 8000, Progress: 80})
	s.SaveTask(storage.DownloadTask{ID: "g2", Status: "paused"}) // was manually paused before shutdown
	s.SetString("auto_resume_ids", "g1")

	e.RecoverInterruptedDownloads()

	// auto_resume_ids should be cleared
	val, _ := s.GetString("auto_resume_ids")
	if val != "" {
		t.Errorf("auto_resume_ids should be cleared, got %q", val)
	}

	// g2 should stay paused (not in auto_resume set)
	t2, _ := s.GetTask("g2")
	if t2.Status != "paused" {
		t.Errorf("g2 status = %q, want paused (unchanged)", t2.Status)
	}
}

func TestRecoverInterruptedDownloads_NoAutoResumeForStoppedOrError(t *testing.T) {
	s := createTestDB(t)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	e := NewEngine(logger, s)

	s.SaveTask(storage.DownloadTask{ID: "s1", Status: "stopped"})
	s.SaveTask(storage.DownloadTask{ID: "e1", Status: "error"})

	e.RecoverInterruptedDownloads()

	t1, _ := s.GetTask("s1")
	t2, _ := s.GetTask("e1")
	if t1.Status != "stopped" {
		t.Errorf("s1 status = %q, want stopped (unchanged)", t1.Status)
	}
	if t2.Status != "error" {
		t.Errorf("e1 status = %q, want error (unchanged)", t2.Status)
	}
}

func TestJoinSplitIDs(t *testing.T) {
	ids := []string{"abc", "def", "ghi"}
	joined := joinIDs(ids)
	if joined != "abc,def,ghi" {
		t.Errorf("joinIDs = %q, want %q", joined, "abc,def,ghi")
	}

	split := splitIDs(joined)
	if len(split) != 3 || split[0] != "abc" || split[1] != "def" || split[2] != "ghi" {
		t.Errorf("splitIDs = %v, want [abc def ghi]", split)
	}

	if splitIDs("") != nil {
		t.Errorf("splitIDs empty should return nil")
	}
}
