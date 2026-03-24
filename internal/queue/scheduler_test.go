package queue

import (
	"log/slog"
	"os"
	"project-tachyon/internal/storage"
	"testing"
	"time"
)

func newTestScheduler() (*SmartScheduler, *DownloadQueue) {
	q := NewDownloadQueue()
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	return NewSmartScheduler(logger, q), q
}

func TestSmartScheduler_BasicDispatch(t *testing.T) {
	sched, q := newTestScheduler()

	q.Push(&storage.DownloadTask{ID: "t1", URL: "https://example.com/file", QueueOrder: 1})

	task := sched.GetNextTask(0, 5)
	if task == nil || task.ID != "t1" {
		t.Fatal("expected task t1 to be dispatched")
	}
	if q.Len() != 0 {
		t.Fatal("task should be removed from queue after dispatch")
	}
}

func TestSmartScheduler_RespectsGlobalLimit(t *testing.T) {
	sched, q := newTestScheduler()
	q.Push(&storage.DownloadTask{ID: "t1", URL: "https://example.com/file", QueueOrder: 1})

	// At capacity
	task := sched.GetNextTask(5, 5)
	if task != nil {
		t.Fatal("should not dispatch when at global limit")
	}
}

func TestSmartScheduler_RespectsHostLimit(t *testing.T) {
	sched, q := newTestScheduler()
	sched.SetHostLimit("example.com", 1)

	q.Push(&storage.DownloadTask{ID: "t1", URL: "https://example.com/a", QueueOrder: 1})
	q.Push(&storage.DownloadTask{ID: "t2", URL: "https://other.com/b", QueueOrder: 2})

	// Start t1
	t1 := sched.GetNextTask(0, 5)
	if t1 == nil || t1.ID != "t1" {
		t.Fatal("expected t1")
	}
	sched.OnTaskStarted(t1)

	// t2 should still be runnable (different host)
	t2 := sched.GetNextTask(1, 5)
	if t2 == nil || t2.ID != "t2" {
		t.Fatal("expected t2 from other.com to be dispatched")
	}
}

func TestSmartScheduler_SkipsScheduledFuture(t *testing.T) {
	sched, q := newTestScheduler()

	future := time.Now().Add(1 * time.Hour).Format(time.RFC3339)
	q.Push(&storage.DownloadTask{ID: "t1", URL: "https://example.com/file", QueueOrder: 1, StartTime: future})

	task := sched.GetNextTask(0, 5)
	if task != nil {
		t.Fatal("should not dispatch task scheduled in the future")
	}
}

func TestSmartScheduler_SetGetHostLimit(t *testing.T) {
	sched, _ := newTestScheduler()
	sched.SetHostLimit("example.com", 3)

	if sched.GetHostLimit("example.com") != 3 {
		t.Fatal("expected host limit 3")
	}
	if sched.GetHostLimit("other.com") != 0 {
		t.Fatal("expected 0 (unlimited) for unset host")
	}
}

func TestSmartScheduler_OnTaskCompletedDecrementsCount(t *testing.T) {
	sched, _ := newTestScheduler()
	task := &storage.DownloadTask{ID: "t1", URL: "https://example.com/file"}

	sched.OnTaskStarted(task)
	sched.mu.Lock()
	before := sched.activePerHost["example.com"]
	sched.mu.Unlock()

	sched.OnTaskCompleted(task)
	sched.mu.Lock()
	after := sched.activePerHost["example.com"]
	sched.mu.Unlock()

	if after >= before {
		t.Fatalf("expected count to decrease: before=%d, after=%d", before, after)
	}
}

func TestExtractDomain(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{"https://example.com/file.zip", "example.com"},
		{"http://sub.domain.org:8080/path", "sub.domain.org"},
		{"not-a-url", ""},
	}
	for _, tt := range tests {
		got := extractDomain(tt.url)
		if got != tt.want {
			t.Errorf("extractDomain(%q) = %q, want %q", tt.url, got, tt.want)
		}
	}
}
