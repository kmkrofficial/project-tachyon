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

// BandwidthManager handles global speed limiting with zero overhead when disabled
type BandwidthManager struct {
	globalLimiter *rate.Limiter
	limitEnabled  atomic.Bool
	mu            sync.RWMutex

	// Map of TaskID -> Priority Level (1=Low, 2=Normal, 3=High)
	taskPriorities map[string]int
}

// NewBandwidthManager creates a new bandwidth manager with no limits
func NewBandwidthManager() *BandwidthManager {
	return &BandwidthManager{
		// Default to strict limit initially, but enabled=false bypasses it
		globalLimiter:  rate.NewLimiter(rate.Inf, 0),
		taskPriorities: make(map[string]int),
	}
}

// SetLimit updates the global speed limit in bytes per second
// 0 means unlimited
func (bm *BandwidthManager) SetLimit(bytesPerSec int) {
	if bytesPerSec <= 0 {
		bm.limitEnabled.Store(false)
		bm.globalLimiter.SetLimit(rate.Inf)
	} else {
		bm.limitEnabled.Store(true)
		bm.globalLimiter.SetLimit(rate.Limit(bytesPerSec))
		bm.globalLimiter.SetBurst(bytesPerSec) // Allow 1s burst
	}
}

// SetTaskPriority sets the priority for a specific task
func (bm *BandwidthManager) SetTaskPriority(taskID string, priority int) {
	bm.mu.Lock()
	defer bm.mu.Unlock()
	bm.taskPriorities[taskID] = priority
}

// Wait blocks until the requested bytes can be consumed
// Returns fast if limit is disabled
func (bm *BandwidthManager) Wait(ctx context.Context, taskID string, bytes int) error {
	// 1. FAST PATH: Zero overhead check
	if !bm.limitEnabled.Load() {
		return nil
	}

	// 2. Priority Logic
	bm.mu.RLock()
	priority, ok := bm.taskPriorities[taskID]
	if !ok {
		priority = 2 // Default Normal
	}
	bm.mu.RUnlock()

	// High Priority (3): Just wait
	// Normal Priority (2): Wait
	// Low Priority (1): Wait + Micro-sleep if constrained

	err := bm.globalLimiter.WaitN(ctx, bytes)
	if err != nil {
		return err
	}

	if priority == 1 {
		// Artificial delay for low priority tasks to yield to high priority ones
		time.Sleep(10 * time.Millisecond)
	}

	return nil
}
