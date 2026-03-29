// Package network provides bandwidth management and congestion control
// for download operations.
package network

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/time/rate"
)

// Priority weights control bandwidth share ratios.
// High:Normal:Low = 6:3:1 — High gets 2× Normal, Normal gets 3× Low.
const (
	weightHigh   = 6
	weightNormal = 3
	weightLow    = 1
)

// BandwidthManager handles global speed limiting and priority-based
// weighted bandwidth allocation across concurrent downloads.
type BandwidthManager struct {
	globalLimiter *rate.Limiter
	limitEnabled  atomic.Bool
	mu            sync.RWMutex

	taskPriorities map[string]int
	activeTasks    map[string]bool // tracks currently downloading tasks
}

// NewBandwidthManager creates a new bandwidth manager with no limits
func NewBandwidthManager() *BandwidthManager {
	return &BandwidthManager{
		globalLimiter:  rate.NewLimiter(rate.Inf, 0),
		taskPriorities: make(map[string]int),
		activeTasks:    make(map[string]bool),
	}
}

// SetLimit updates the global speed limit in bytes per second.
// 0 means unlimited.
func (bm *BandwidthManager) SetLimit(bytesPerSec int) {
	if bytesPerSec <= 0 {
		bm.limitEnabled.Store(false)
		bm.globalLimiter.SetLimit(rate.Inf)
	} else {
		bm.limitEnabled.Store(true)
		bm.globalLimiter.SetLimit(rate.Limit(bytesPerSec))
		bm.globalLimiter.SetBurst(bytesPerSec)
	}
}

// SetTaskPriority sets the priority for a specific task
func (bm *BandwidthManager) SetTaskPriority(taskID string, priority int) {
	bm.mu.Lock()
	defer bm.mu.Unlock()
	bm.taskPriorities[taskID] = priority
}

// MarkActive registers a task as currently downloading.
func (bm *BandwidthManager) MarkActive(taskID string) {
	bm.mu.Lock()
	defer bm.mu.Unlock()
	bm.activeTasks[taskID] = true
}

// MarkInactive removes a task from the active set.
func (bm *BandwidthManager) MarkInactive(taskID string) {
	bm.mu.Lock()
	defer bm.mu.Unlock()
	delete(bm.activeTasks, taskID)
}

// Wait blocks until the requested bytes can be consumed.
// Implements weighted fair queuing: lower-priority tasks sleep longer
// per chunk so higher-priority tasks consume more bandwidth.
func (bm *BandwidthManager) Wait(ctx context.Context, taskID string, bytes int) error {
	// 1. Global rate limit (if configured)
	if bm.limitEnabled.Load() {
		if err := bm.globalLimiter.WaitN(ctx, bytes); err != nil {
			return err
		}
	}

	// 2. Priority-weighted delay
	bm.mu.RLock()
	priority, ok := bm.taskPriorities[taskID]
	if !ok {
		priority = 2
	}
	hasHigher := bm.hasHigherPriorityActive(taskID, priority)
	bm.mu.RUnlock()

	// Only apply throttling when a higher-priority task is competing
	if !hasHigher {
		return nil
	}

	delay := bm.priorityDelay(priority)
	if delay <= 0 {
		return nil
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(delay):
		return nil
	}
}

// hasHigherPriorityActive checks if any active task has strictly higher priority.
// Must be called with bm.mu held for reading.
func (bm *BandwidthManager) hasHigherPriorityActive(taskID string, myPriority int) bool {
	for id, active := range bm.activeTasks {
		if !active || id == taskID {
			continue
		}
		p, ok := bm.taskPriorities[id]
		if !ok {
			p = 2
		}
		if p > myPriority {
			return true
		}
	}
	return false
}

// priorityDelay returns the per-chunk delay for a given priority level.
// High (3) = 0, Normal (2) = 5ms, Low (1) = 20ms.
// These delays are per read-loop iteration (each ~1MB buffer), creating
// a bandwidth ratio of roughly High:Normal:Low ≈ 6:3:1 at full speed.
func (bm *BandwidthManager) priorityDelay(priority int) time.Duration {
	switch priority {
	case 3:
		return 0
	case 2:
		return 5 * time.Millisecond
	case 1:
		return 20 * time.Millisecond
	default:
		return 5 * time.Millisecond
	}
}
