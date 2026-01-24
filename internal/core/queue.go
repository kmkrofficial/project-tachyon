package core

import (
	"container/heap"
	"project-tachyon/internal/storage"
	"sync"
)

// Item wraps a DownloadTask for the PriorityQueue
type Item struct {
	Task     *storage.DownloadTask
	Priority int // 0=Low, 1=Normal, 2=High
	Index    int // Index in the heap
}

// PriorityQueue implements heap.Interface and holds Items.
type PriorityQueue []*Item

func (pq PriorityQueue) Len() int { return len(pq) }

func (pq PriorityQueue) Less(i, j int) bool {
	// We want Pop to give us the highest, not lowest, priority so we use greater than here.
	// If priorities are equal, FIFO (using CreatedAt could be secondary sort)
	if pq[i].Priority == pq[j].Priority {
		return pq[i].Task.CreatedAt.Before(pq[j].Task.CreatedAt) // Oldest first for same priority
	}
	return pq[i].Priority > pq[j].Priority
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].Index = i
	pq[j].Index = j
}

func (pq *PriorityQueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*Item)
	item.Index = n
	*pq = append(*pq, item)
}

func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil  // avoid memory leak
	item.Index = -1 // for safety
	*pq = old[0 : n-1]
	return item
}

// DownloadQueue manages safe access to the PriorityQueue
type DownloadQueue struct {
	pq    PriorityQueue
	mutex sync.Mutex
	cond  *sync.Cond
}

func NewDownloadQueue() *DownloadQueue {
	dq := &DownloadQueue{}
	dq.cond = sync.NewCond(&dq.mutex)
	heap.Init(&dq.pq)
	return dq
}

func (dq *DownloadQueue) Push(task *storage.DownloadTask) {
	dq.mutex.Lock()
	defer dq.mutex.Unlock()

	item := &Item{
		Task:     task,
		Priority: task.Priority,
	}
	heap.Push(&dq.pq, item)
	dq.cond.Signal()
}

func (dq *DownloadQueue) Pop() *storage.DownloadTask {
	dq.mutex.Lock()
	defer dq.mutex.Unlock()

	for dq.pq.Len() == 0 {
		dq.cond.Wait()
	}

	item := heap.Pop(&dq.pq).(*Item)
	return item.Task
}

func (dq *DownloadQueue) Len() int {
	dq.mutex.Lock()
	defer dq.mutex.Unlock()
	return dq.pq.Len()
}
