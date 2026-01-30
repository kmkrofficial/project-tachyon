package queue

import (
	"project-tachyon/internal/storage"
	"sort"
	"sync"
)

// DownloadQueue manages ordered queue of downloads
type DownloadQueue struct {
	items []*storage.DownloadTask
	mutex sync.Mutex
	cond  *sync.Cond
}

func NewDownloadQueue() *DownloadQueue {
	dq := &DownloadQueue{
		items: make([]*storage.DownloadTask, 0),
	}
	dq.cond = sync.NewCond(&dq.mutex)
	return dq
}

// Push adds a task to the queue, sorted by QueueOrder
func (dq *DownloadQueue) Push(task *storage.DownloadTask) {
	dq.mutex.Lock()
	defer dq.mutex.Unlock()

	dq.items = append(dq.items, task)
	// Sort by QueueOrder (lowest first)
	sort.Slice(dq.items, func(i, j int) bool {
		return dq.items[i].QueueOrder < dq.items[j].QueueOrder
	})
	dq.cond.Signal()
}

// Pop removes and returns the first task (lowest QueueOrder)
func (dq *DownloadQueue) Pop() *storage.DownloadTask {
	dq.mutex.Lock()
	defer dq.mutex.Unlock()

	for len(dq.items) == 0 {
		dq.cond.Wait()
	}

	task := dq.items[0]
	dq.items = dq.items[1:]
	return task
}

// PopSpecific removes a specific task by ID (for SmartScheduler picking)
func (dq *DownloadQueue) Remove(id string) bool {
	dq.mutex.Lock()
	defer dq.mutex.Unlock()

	for i, item := range dq.items {
		if item.ID == id {
			dq.items = append(dq.items[:i], dq.items[i+1:]...)
			return true
		}
	}
	return false
}

// Len returns the number of items in the queue
func (dq *DownloadQueue) Len() int {
	dq.mutex.Lock()
	defer dq.mutex.Unlock()
	return len(dq.items)
}

// GetAll returns a copy of all queued items
func (dq *DownloadQueue) GetAll() []*storage.DownloadTask {
	dq.mutex.Lock()
	defer dq.mutex.Unlock()

	result := make([]*storage.DownloadTask, len(dq.items))
	copy(result, dq.items)
	return result
}

// GetNextOrder returns the next available QueueOrder value
func (dq *DownloadQueue) GetNextOrder() int {
	dq.mutex.Lock()
	defer dq.mutex.Unlock()

	if len(dq.items) == 0 {
		return 1
	}
	maxOrder := 0
	for _, item := range dq.items {
		if item.QueueOrder > maxOrder {
			maxOrder = item.QueueOrder
		}
	}
	return maxOrder + 1
}

// Wait blocks until a signal is received
func (dq *DownloadQueue) Wait() {
	dq.mutex.Lock()
	defer dq.mutex.Unlock()
	dq.cond.Wait()
}

// Signal wakes one waiter
func (dq *DownloadQueue) Signal() {
	dq.cond.Signal()
}

// Broadcast wakes all waiters
func (dq *DownloadQueue) Broadcast() {
	dq.cond.Broadcast()
}

// MoveToFirst, Prev, Next, Last - implementation identical to core/queue.go
// Adding them for completeness... will omit detailed body mostly unless needed.
// Actually Engine relied on them. I must include them.

func (dq *DownloadQueue) moveTask(id string, op func(idx int) int) bool {
	// Helper to avoid repetition
	return false
}

// Full implementation required for Engine to work
func (dq *DownloadQueue) MoveToFirst(id string) bool {
	dq.mutex.Lock()
	defer dq.mutex.Unlock()
	idx := dq.findIndex(id)
	if idx <= 0 {
		return false
	}
	task := dq.items[idx]
	dq.items = append(dq.items[:idx], dq.items[idx+1:]...)
	dq.items = append([]*storage.DownloadTask{task}, dq.items...)
	dq.reorderSequential()
	return true
}

func (dq *DownloadQueue) MoveToPrev(id string) bool {
	dq.mutex.Lock()
	defer dq.mutex.Unlock()
	idx := dq.findIndex(id)
	if idx <= 0 {
		return false
	}
	dq.items[idx], dq.items[idx-1] = dq.items[idx-1], dq.items[idx]
	dq.reorderSequential()
	return true
}

func (dq *DownloadQueue) MoveToNext(id string) bool {
	dq.mutex.Lock()
	defer dq.mutex.Unlock()
	idx := dq.findIndex(id)
	if idx < 0 || idx >= len(dq.items)-1 {
		return false
	}
	dq.items[idx], dq.items[idx+1] = dq.items[idx+1], dq.items[idx]
	dq.reorderSequential()
	return true
}

func (dq *DownloadQueue) MoveToLast(id string) bool {
	dq.mutex.Lock()
	defer dq.mutex.Unlock()
	idx := dq.findIndex(id)
	if idx < 0 || idx >= len(dq.items)-1 {
		return false
	}
	task := dq.items[idx]
	dq.items = append(dq.items[:idx], dq.items[idx+1:]...)
	dq.items = append(dq.items, task)
	dq.reorderSequential()
	return true
}

func (dq *DownloadQueue) findIndex(id string) int {
	for i, item := range dq.items {
		if item.ID == id {
			return i
		}
	}
	return -1
}

func (dq *DownloadQueue) reorderSequential() {
	for i, item := range dq.items {
		item.QueueOrder = i + 1
	}
}
