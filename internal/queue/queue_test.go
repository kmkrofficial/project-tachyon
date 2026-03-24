package queue

import (
	"project-tachyon/internal/storage"
	"testing"
)

func TestDownloadQueue_PushPopOrder(t *testing.T) {
	q := NewDownloadQueue()

	q.Push(&storage.DownloadTask{ID: "a", QueueOrder: 3})
	q.Push(&storage.DownloadTask{ID: "b", QueueOrder: 1})
	q.Push(&storage.DownloadTask{ID: "c", QueueOrder: 2})

	if q.Len() != 3 {
		t.Fatalf("expected len 3, got %d", q.Len())
	}

	// Pop should return lowest QueueOrder first
	got := q.Pop()
	if got.ID != "b" {
		t.Fatalf("expected 'b' (order 1), got %s", got.ID)
	}
	got = q.Pop()
	if got.ID != "c" {
		t.Fatalf("expected 'c' (order 2), got %s", got.ID)
	}
	got = q.Pop()
	if got.ID != "a" {
		t.Fatalf("expected 'a' (order 3), got %s", got.ID)
	}
}

func TestDownloadQueue_Remove(t *testing.T) {
	q := NewDownloadQueue()
	q.Push(&storage.DownloadTask{ID: "x", QueueOrder: 1})
	q.Push(&storage.DownloadTask{ID: "y", QueueOrder: 2})

	if !q.Remove("x") {
		t.Fatal("expected Remove to return true for existing item")
	}
	if q.Len() != 1 {
		t.Fatalf("expected len 1 after remove, got %d", q.Len())
	}
	if q.Remove("x") {
		t.Fatal("expected Remove to return false for already removed item")
	}
}

func TestDownloadQueue_GetAll(t *testing.T) {
	q := NewDownloadQueue()
	q.Push(&storage.DownloadTask{ID: "a", QueueOrder: 1})
	q.Push(&storage.DownloadTask{ID: "b", QueueOrder: 2})

	all := q.GetAll()
	if len(all) != 2 {
		t.Fatalf("expected 2 items, got %d", len(all))
	}
	// GetAll should not remove items
	if q.Len() != 2 {
		t.Fatal("GetAll should not modify the queue")
	}
}

func TestDownloadQueue_GetNextOrder(t *testing.T) {
	q := NewDownloadQueue()
	if q.GetNextOrder() != 1 {
		t.Fatal("empty queue should return order 1")
	}

	q.Push(&storage.DownloadTask{ID: "a", QueueOrder: 5})
	if q.GetNextOrder() != 6 {
		t.Fatalf("expected next order 6, got %d", q.GetNextOrder())
	}
}

func TestDownloadQueue_LenEmpty(t *testing.T) {
	q := NewDownloadQueue()
	if q.Len() != 0 {
		t.Fatal("new queue should have length 0")
	}
}
