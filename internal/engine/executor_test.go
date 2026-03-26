package engine

import (
	"log/slog"
	"os"
	"testing"

	"project-tachyon/internal/storage"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func createExecutorTestDB(t *testing.T) *storage.Storage {
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

func TestFailTask_SetsErrorStatus(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	s := createExecutorTestDB(t)
	e := NewEngine(logger, s)

	task := &storage.DownloadTask{
		ID:     "fail-1",
		URL:    "http://example.com/file.zip",
		Status: "downloading",
	}
	s.SaveTask(*task)

	e.failTask(task, "Probe failed: connection reset")

	updated, err := s.GetTask("fail-1")
	if err != nil {
		t.Fatalf("GetTask error: %v", err)
	}
	if updated.Status != "error" {
		t.Errorf("Status = %q, want %q", updated.Status, "error")
	}
}

func TestFailTask_StoresErrorStatus(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	s := createExecutorTestDB(t)
	e := NewEngine(logger, s)

	task := &storage.DownloadTask{
		ID:     "fail-2",
		URL:    "http://example.com/file.zip",
		Status: "downloading",
	}
	s.SaveTask(*task)

	e.failTask(task, "Allocation failed: not enough space")

	updated, _ := s.GetTask("fail-2")
	if updated.Status != "error" {
		t.Errorf("Status = %q, want %q after failTask", updated.Status, "error")
	}
}

func TestStatusNeedsAuthConstant(t *testing.T) {
	if StatusNeedsAuth != "needs_auth" {
		t.Errorf("StatusNeedsAuth = %q, want %q", StatusNeedsAuth, "needs_auth")
	}
}

func TestConfigConstants(t *testing.T) {
	if DownloadChunkSize <= 0 {
		t.Errorf("DownloadChunkSize = %d, want > 0", DownloadChunkSize)
	}
	if BufferSize <= 0 {
		t.Errorf("BufferSize = %d, want > 0", BufferSize)
	}
	if MaxWorkersPerTask <= 0 {
		t.Errorf("MaxWorkersPerTask = %d, want > 0", MaxWorkersPerTask)
	}
	if GenericUserAgent == "" {
		t.Error("GenericUserAgent should not be empty")
	}
}
