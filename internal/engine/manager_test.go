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
	if DownloadChunkSize != 1*1024*1024 {
		t.Errorf("DownloadChunkSize = %d, want %d", DownloadChunkSize, 1*1024*1024)
	}
	if BufferSize != 256*1024 {
		t.Errorf("BufferSize = %d, want %d", BufferSize, 256*1024)
	}
	if MaxWorkersPerTask != 24 {
		t.Errorf("MaxWorkersPerTask = %d, want 24", MaxWorkersPerTask)
	}
	if GenericUserAgent == "" {
		t.Error("GenericUserAgent should not be empty")
	}
}
