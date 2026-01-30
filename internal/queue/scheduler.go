package queue

import (
	"log/slog"
	"net/url"
	"project-tachyon/internal/storage"
	"sync"
	"time"
)

type SmartScheduler struct {
	logger        *slog.Logger
	queue         *DownloadQueue
	hostLimits    map[string]int // Domain -> Max Concurrent
	activePerHost map[string]int // Domain -> Current Activce
	mu            sync.Mutex
}

func NewSmartScheduler(logger *slog.Logger, queue *DownloadQueue) *SmartScheduler {
	return &SmartScheduler{
		logger:        logger,
		queue:         queue,
		hostLimits:    make(map[string]int),
		activePerHost: make(map[string]int),
	}
}

func (s *SmartScheduler) SetHostLimit(domain string, limit int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.hostLimits[domain] = limit
}

func (s *SmartScheduler) GetHostLimit(domain string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	if limit, ok := s.hostLimits[domain]; ok {
		return limit
	}
	return 0 // 0 means unlimited
}

// OnTaskStarted should be called by Engine when a task starts downloading
func (s *SmartScheduler) OnTaskStarted(task *storage.DownloadTask) {
	s.mu.Lock()
	defer s.mu.Unlock()
	domain := extractDomain(task.URL)
	s.activePerHost[domain]++
	task.Domain = domain // Update task domain if not set
}

// OnTaskCompleted should be called by Engine when a task stops/finishes
func (s *SmartScheduler) OnTaskCompleted(task *storage.DownloadTask) {
	s.mu.Lock()
	defer s.mu.Unlock()
	domain := extractDomain(task.URL)
	if s.activePerHost[domain] > 0 {
		s.activePerHost[domain]--
	}
	// Signal queue to wake up workers as a slot might have opened
	s.queue.Broadcast()
}

// GetNextTask returns the next eligible task from the queue
// regarding priority and host limits
func (s *SmartScheduler) GetNextTask(activeCount, maxConcurrent int) *storage.DownloadTask {
	// First check global concurrency
	if activeCount >= maxConcurrent {
		// Preemption Check?
		// Logic: If queue has High Priority and Running has Low Priority...
		// But GetNextTask usually assumes we have a slot.
		// If we don't have a slot, we shouldn't ask for a task unless we want to preempt?
		// Engine calls this loop.
		return nil
	}

	// Iterate queue and find first "Runnable" task
	// We need to peek/iterate the queue without popping until we find one.
	// Current DownloadQueue might need iteration support.

	// Assuming Queue exposes Items or Iterator.
	// We'll add this to DownloadQueue.

	candidates := s.queue.GetAll() // Snapshot
	for _, task := range candidates {
		// 1. Check Schedule
		if task.StartTime != "" {
			t, err := time.Parse(time.RFC3339, task.StartTime)
			if err == nil && time.Now().Before(t) {
				continue // Too early
			}
		}

		// 2. Check Host Limits
		domain := extractDomain(task.URL)
		limit := s.GetHostLimit(domain)

		s.mu.Lock()
		active := s.activePerHost[domain]
		s.mu.Unlock()

		if limit > 0 && active >= limit {
			continue // Host limit reached
		}

		// Found a candidate!
		// We need to Pop THIS SPECIFIC task from the queue.
		// Queue.Remove(id) ??
		// Or Queue.Pop() if it's the first one.
		// If it's not the first one, we are "Skipping" previous ones.
		// Users might be confused if order is 1, 2, 3 and 3 starts.
		// But that's "Smart Scheduling".

		removed := s.queue.Remove(task.ID)
		if removed {
			return task
		}
	}

	return nil
}

func extractDomain(urlStr string) string {
	u, err := url.Parse(urlStr)
	if err != nil {
		return ""
	}
	return u.Hostname()
}
