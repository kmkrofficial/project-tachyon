package storage

import (
	"os"
	"testing"
	"time"

	"github.com/dgraph-io/badger/v4"
)

func TestStorage_Integration(t *testing.T) {
	// Setup Temp Dir
	tempDir, err := os.MkdirTemp("", "tachyon_test_db")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Override Config Dir for Test (Mocking via Env or just using NewStorage pointing to temp if refactored)
	// Since NewStorage uses os.UserConfigDir, it's hard to mock without refactoring.
	// REFACTOR SUCGESTION: Change NewStorage to accept a path?
	// For this test, let's create a specialized OpenDB function or just use NewStorage logic inline to test methods.

	// Option B: Inline logic to reuse code isn't great.
	// Best approach: Refactor NewStorage to accept (path string) and have a helper NewDefaultStorage().
	// But to avoid changing `db.go` signature excessively given user instructions, I'll modify NewStorage validation slightly
	// or assume I can run it locally.

	// Actually, modifying `db.go` to Open(path) is better.
	// Let's modify db.go first to make it testable?
	// Or just test the logic by copying `badger.Open` here?

	// I will instantiate Storage struct directly with a custom DB.
	opts := badger.DefaultOptions(tempDir)
	opts.Logger = nil
	db, err := badger.Open(opts)
	if err != nil {
		t.Fatal(err)
	}

	store := &Storage{db: db}
	defer store.Close()

	// 1. Test SaveTask
	task := Task{
		ID:        "test-1",
		URL:       "http://example.com/test",
		Filename:  "test.file",
		Status:    "downloading",
		CreatedAt: time.Now(),
	}

	if err := store.SaveTask(task); err != nil {
		t.Fatalf("SaveTask failed: %v", err)
	}

	// 2. Test GetTasks
	tasks, err := store.GetAllTasks()
	if err != nil {
		t.Fatalf("GetAllTasks failed: %v", err)
	}

	if len(tasks) != 1 {
		t.Errorf("Expected 1 task, got %d", len(tasks))
	}
	if tasks[0].ID != "test-1" {
		t.Errorf("Expected task ID test-1, got %s", tasks[0].ID)
	}

	// 3. Test DeleteTask
	if err := store.DeleteTask("test-1"); err != nil {
		t.Fatalf("DeleteTask failed: %v", err)
	}

	tasks, err = store.GetAllTasks()
	if err != nil {
		t.Fatalf("GetAllTasks after delete failed: %v", err)
	}
	if len(tasks) != 0 {
		t.Errorf("Expected 0 tasks after delete, got %d", len(tasks))
	}
}
