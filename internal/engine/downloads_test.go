package engine

import (
	"log/slog"
	"os"
	"testing"
	"time"

	"project-tachyon/internal/storage"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func createDownloadsTestDB(t *testing.T) *storage.Storage {
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

func TestGetHistory_Empty(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	s := createDownloadsTestDB(t)
	e := NewEngine(logger, s)

	tasks, err := e.GetHistory()
	if err != nil {
		t.Fatalf("GetHistory() error: %v", err)
	}
	if len(tasks) != 0 {
		t.Errorf("Expected 0 tasks, got %d", len(tasks))
	}
}

func TestGetHistory_WithTasks(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	s := createDownloadsTestDB(t)
	e := NewEngine(logger, s)

	// Insert tasks directly into DB
	s.SaveTask(storage.DownloadTask{
		ID:       "t1",
		URL:      "http://example.com/file1.zip",
		Filename: "file1.zip",
		Status:   "completed",
	})
	s.SaveTask(storage.DownloadTask{
		ID:       "t2",
		URL:      "http://example.com/file2.zip",
		Filename: "file2.zip",
		Status:   "paused",
	})

	tasks, err := e.GetHistory()
	if err != nil {
		t.Fatalf("GetHistory() error: %v", err)
	}
	if len(tasks) != 2 {
		t.Errorf("Expected 2 tasks, got %d", len(tasks))
	}
}

func TestGetTask(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	s := createDownloadsTestDB(t)
	e := NewEngine(logger, s)

	s.SaveTask(storage.DownloadTask{
		ID:       "task-abc",
		URL:      "http://example.com/test.zip",
		Filename: "test.zip",
		Status:   "pending",
	})

	task, err := e.GetTask("task-abc")
	if err != nil {
		t.Fatalf("GetTask() error: %v", err)
	}
	if task.Filename != "test.zip" {
		t.Errorf("Filename = %q, want %q", task.Filename, "test.zip")
	}
}

func TestGetTask_NotFound(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	s := createDownloadsTestDB(t)
	e := NewEngine(logger, s)

	_, err := e.GetTask("nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent task")
	}
}

func TestPauseDownload_NotActive(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	s := createDownloadsTestDB(t)
	e := NewEngine(logger, s)

	// Save a pending task
	s.SaveTask(storage.DownloadTask{
		ID:       "pause-test",
		URL:      "http://example.com/file.zip",
		Filename: "file.zip",
		Status:   "pending",
	})

	err := e.PauseDownload("pause-test")
	if err != nil {
		t.Fatalf("PauseDownload() error: %v", err)
	}

	// Verify status was updated
	task, _ := s.GetTask("pause-test")
	if task.Status != "paused" {
		t.Errorf("Status = %q, want %q", task.Status, "paused")
	}
}

func TestResumeDownload(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	s := createDownloadsTestDB(t)
	e := NewEngine(logger, s)

	s.SaveTask(storage.DownloadTask{
		ID:       "resume-test",
		URL:      "http://example.com/file.zip",
		Filename: "file.zip",
		Status:   "paused",
	})

	err := e.ResumeDownload("resume-test")
	if err != nil {
		t.Fatalf("ResumeDownload() error: %v", err)
	}
}

func TestResumeDownload_NotFound(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	s := createDownloadsTestDB(t)
	e := NewEngine(logger, s)

	err := e.ResumeDownload("nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent task")
	}
}

func TestResumeDownload_CompletedCannotResume(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	s := createDownloadsTestDB(t)
	e := NewEngine(logger, s)

	s.SaveTask(storage.DownloadTask{
		ID:       "completed-task",
		URL:      "http://example.com/file.zip",
		Filename: "file.zip",
		Status:   "completed",
	})

	err := e.ResumeDownload("completed-task")
	if err == nil {
		t.Error("Expected error when resuming completed task")
	}
}

func TestStopDownload(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	s := createDownloadsTestDB(t)
	e := NewEngine(logger, s)

	s.SaveTask(storage.DownloadTask{
		ID:       "stop-test",
		URL:      "http://example.com/file.zip",
		Filename: "file.zip",
		Status:   "downloading",
	})

	err := e.StopDownload("stop-test")
	if err != nil {
		t.Fatalf("StopDownload() error: %v", err)
	}

	task, _ := s.GetTask("stop-test")
	if task.Status != "stopped" {
		t.Errorf("Status = %q, want %q", task.Status, "stopped")
	}
}

func TestPauseAllDownloads(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	s := createDownloadsTestDB(t)
	e := NewEngine(logger, s)

	// Add pending tasks
	s.SaveTask(storage.DownloadTask{
		ID:       "p1",
		URL:      "http://example.com/1.zip",
		Filename: "1.zip",
		Status:   "pending",
	})
	s.SaveTask(storage.DownloadTask{
		ID:       "p2",
		URL:      "http://example.com/2.zip",
		Filename: "2.zip",
		Status:   "pending",
	})

	e.PauseAllDownloads()

	// Both should be paused
	t1, _ := s.GetTask("p1")
	t2, _ := s.GetTask("p2")
	if t1.Status != "paused" {
		t.Errorf("Task p1 status = %q, want paused", t1.Status)
	}
	if t2.Status != "paused" {
		t.Errorf("Task p2 status = %q, want paused", t2.Status)
	}
}

func TestStartDownload(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	s := createDownloadsTestDB(t)
	e := NewEngine(logger, s)

	id, err := e.StartDownload("https://example.com/testfile.zip", os.TempDir(), "", map[string]string{})
	if err != nil {
		t.Fatalf("StartDownload() error: %v", err)
	}
	if id == "" {
		t.Error("StartDownload() returned empty ID")
	}

	// Verify task exists in DB
	task, err := s.GetTask(id)
	if err != nil {
		t.Fatalf("Task not found in DB: %v", err)
	}
	if task.Status != "pending" {
		t.Errorf("Status = %q, want pending", task.Status)
	}
}

func TestStartDownload_InvalidURL(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	s := createDownloadsTestDB(t)
	e := NewEngine(logger, s)

	_, err := e.StartDownload("ftp://bad/url", os.TempDir(), "", map[string]string{})
	if err == nil {
		t.Error("Expected error for invalid URL scheme")
	}
}

func TestStartDownload_ScheduledStart(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	s := createDownloadsTestDB(t)
	e := NewEngine(logger, s)

	future := time.Now().Add(1 * time.Hour).Format(time.RFC3339)
	id, err := e.StartDownload("https://example.com/scheduled.zip", os.TempDir(), "", map[string]string{
		"start_time": future,
	})
	if err != nil {
		t.Fatalf("StartDownload() error: %v", err)
	}

	task, _ := s.GetTask(id)
	if task.Status != "scheduled" {
		t.Errorf("Status = %q, want scheduled", task.Status)
	}
}

func TestStartDownload_CustomFilename(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	s := createDownloadsTestDB(t)
	e := NewEngine(logger, s)

	id, err := e.StartDownload("https://example.com/file.zip", os.TempDir(), "custom_name.zip", map[string]string{})
	if err != nil {
		t.Fatalf("StartDownload() error: %v", err)
	}

	task, _ := s.GetTask(id)
	// Filename should be based on the custom name (may be in a subdirectory)
	if task.Filename == "" {
		t.Error("Filename should not be empty")
	}
}
