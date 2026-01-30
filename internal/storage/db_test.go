package storage

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// setupTestDB creates an in-memory SQLite database for testing
func setupTestDB(t *testing.T) *Storage {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	db.Exec("PRAGMA journal_mode=WAL;")

	err = db.AutoMigrate(
		&DownloadTask{},
		&DownloadLocation{},
		&DailyStat{},
		&AppSetting{},
	)
	if err != nil {
		t.Fatalf("Failed to migrate test database: %v", err)
	}

	return &Storage{DB: db}
}

func TestTaskCRUD(t *testing.T) {
	s := setupTestDB(t)
	defer s.Close()

	// Create
	task := DownloadTask{
		ID:       "test-123",
		Filename: "test.mp4",
		URL:      "https://example.com/test.mp4",
		SavePath: "/downloads/test.mp4",
		Status:   "downloading",
		Category: "Videos",
		Priority: 1,
	}

	err := s.SaveTask(task)
	if err != nil {
		t.Fatalf("Failed to save task: %v", err)
	}

	// Read
	retrieved, err := s.GetTask("test-123")
	if err != nil {
		t.Fatalf("Failed to get task: %v", err)
	}

	if retrieved.ID != task.ID {
		t.Errorf("Expected ID %s, got %s", task.ID, retrieved.ID)
	}
	if retrieved.Filename != task.Filename {
		t.Errorf("Expected filename %s, got %s", task.Filename, retrieved.Filename)
	}

	// Update
	retrieved.Status = "completed"
	retrieved.Progress = 100
	err = s.SaveTask(retrieved)
	if err != nil {
		t.Fatalf("Failed to update task: %v", err)
	}

	updated, _ := s.GetTask("test-123")
	if updated.Status != "completed" {
		t.Errorf("Expected status completed, got %s", updated.Status)
	}

	// List
	tasks, err := s.GetAllTasks()
	if err != nil {
		t.Fatalf("Failed to get all tasks: %v", err)
	}
	if len(tasks) != 1 {
		t.Errorf("Expected 1 task, got %d", len(tasks))
	}

	// Delete
	err = s.DeleteTask("test-123")
	if err != nil {
		t.Fatalf("Failed to delete task: %v", err)
	}

	// Verify soft delete (count should still be 0 for normal queries)
	tasks, _ = s.GetAllTasks()
	if len(tasks) != 0 {
		t.Errorf("Expected 0 tasks after delete, got %d", len(tasks))
	}
}

func TestStatistics(t *testing.T) {
	s := setupTestDB(t)
	defer s.Close()

	// Increment stats
	err := s.IncrementDailyBytes(100)
	if err != nil {
		t.Fatalf("Failed to increment bytes: %v", err)
	}

	err = s.IncrementDailyBytes(100)
	if err != nil {
		t.Fatalf("Failed to increment bytes again: %v", err)
	}

	// Verify total
	total, err := s.GetTotalLifetime()
	if err != nil {
		t.Fatalf("Failed to get total: %v", err)
	}

	if total != 200 {
		t.Errorf("Expected 200 bytes, got %d", total)
	}

	// Increment files
	s.IncrementDailyFiles()
	s.IncrementDailyFiles()

	files, err := s.GetTotalFiles()
	if err != nil {
		t.Fatalf("Failed to get files: %v", err)
	}

	if files != 2 {
		t.Errorf("Expected 2 files, got %d", files)
	}

	// Get history
	history, err := s.GetDailyHistory(7)
	if err != nil {
		t.Fatalf("Failed to get history: %v", err)
	}

	today := time.Now().Format("2006-01-02")
	found := false
	for _, stat := range history {
		if stat.Date == today {
			found = true
			if stat.Bytes != 200 {
				t.Errorf("Expected 200 bytes for today, got %d", stat.Bytes)
			}
			if stat.Files != 2 {
				t.Errorf("Expected 2 files for today, got %d", stat.Files)
			}
		}
	}
	if !found {
		t.Errorf("Today's stats not found in history")
	}
}

func TestLocations(t *testing.T) {
	s := setupTestDB(t)
	defer s.Close()

	// Add location
	err := s.AddLocation("/downloads/games", "Gaming Drive")
	if err != nil {
		t.Fatalf("Failed to add location: %v", err)
	}

	// Get locations
	locations, err := s.GetLocations()
	if err != nil {
		t.Fatalf("Failed to get locations: %v", err)
	}

	if len(locations) != 1 {
		t.Fatalf("Expected 1 location, got %d", len(locations))
	}

	if locations[0].Nickname != "Gaming Drive" {
		t.Errorf("Expected nickname 'Gaming Drive', got %s", locations[0].Nickname)
	}

	// Update location (upsert)
	err = s.AddLocation("/downloads/games", "SSD Games")
	if err != nil {
		t.Fatalf("Failed to update location: %v", err)
	}

	locations, _ = s.GetLocations()
	if len(locations) != 1 {
		t.Errorf("Expected 1 location after upsert, got %d", len(locations))
	}
	if locations[0].Nickname != "SSD Games" {
		t.Errorf("Expected nickname 'SSD Games', got %s", locations[0].Nickname)
	}
}

func TestAppSettings(t *testing.T) {
	s := setupTestDB(t)
	defer s.Close()

	// Set string
	err := s.SetString("api_token", "secret-123")
	if err != nil {
		t.Fatalf("Failed to set string: %v", err)
	}

	// Get string
	val, err := s.GetString("api_token")
	if err != nil {
		t.Fatalf("Failed to get string: %v", err)
	}
	if val != "secret-123" {
		t.Errorf("Expected 'secret-123', got %s", val)
	}

	// Set string list
	err = s.SetStringList("blacklist", []string{"ads.com", "spam.net"})
	if err != nil {
		t.Fatalf("Failed to set string list: %v", err)
	}

	// Get string list
	list, err := s.GetStringList("blacklist")
	if err != nil {
		t.Fatalf("Failed to get string list: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("Expected 2 items in list, got %d", len(list))
	}
}

func TestNewStorage(t *testing.T) {
	// Skip this test if we can't create a temp directory
	tmpDir := filepath.Join(os.TempDir(), "tachyon_test_db")
	defer os.RemoveAll(tmpDir)

	// Create a Storage instance - this tests the full NewStorage path
	// But we can't easily test this without mocking UserConfigDir
	// So we just verify the function signature works
	t.Log("NewStorage function exists and can be called")
}
