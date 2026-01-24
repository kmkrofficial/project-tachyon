package core

import (
	"project-tachyon/internal/storage"
	"testing"
	"time"
)

func TestPriorityQueue(t *testing.T) {
	dq := NewDownloadQueue()

	t1 := &storage.DownloadTask{ID: "1", Priority: 0, CreatedAt: time.Now()} // Low
	t2 := &storage.DownloadTask{ID: "2", Priority: 2, CreatedAt: time.Now()} // High
	t3 := &storage.DownloadTask{ID: "3", Priority: 1, CreatedAt: time.Now()} // Normal

	dq.Push(t1)
	dq.Push(t2)
	dq.Push(t3)

	// Pop order should be High (2) -> Normal (3) -> Low (1)
	p1 := dq.Pop()
	if p1.ID != "2" {
		t.Errorf("Expected first pop to be ID 2 (High), got %s", p1.ID)
	}

	p2 := dq.Pop()
	if p2.ID != "3" {
		t.Errorf("Expected second pop to be ID 3 (Normal), got %s", p2.ID)
	}

	p3 := dq.Pop()
	if p3.ID != "1" {
		t.Errorf("Expected third pop to be ID 1 (Low), got %s", p3.ID)
	}
}
