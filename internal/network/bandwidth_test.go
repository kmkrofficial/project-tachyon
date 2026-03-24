package network

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestBandwidthManager_UnlimitedByDefault(t *testing.T) {
	bm := NewBandwidthManager()
	ctx := context.Background()

	start := time.Now()
	for i := 0; i < 100; i++ {
		if err := bm.Wait(ctx, "task-1", 32768); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}
	elapsed := time.Since(start)
	if elapsed > 100*time.Millisecond {
		t.Fatalf("unlimited mode took too long: %v", elapsed)
	}
}

func TestBandwidthManager_SetLimitEnablesThrottle(t *testing.T) {
	bm := NewBandwidthManager()
	bm.SetLimit(1024) // 1KB/s

	if !bm.limitEnabled.Load() {
		t.Fatal("expected limit to be enabled")
	}

	bm.SetLimit(0) // Disable
	if bm.limitEnabled.Load() {
		t.Fatal("expected limit to be disabled after SetLimit(0)")
	}
}

func TestBandwidthManager_PriorityDefault(t *testing.T) {
	bm := NewBandwidthManager()
	bm.SetTaskPriority("t1", 3)

	bm.mu.RLock()
	p := bm.taskPriorities["t1"]
	bm.mu.RUnlock()

	if p != 3 {
		t.Fatalf("expected priority 3, got %d", p)
	}
}

func TestBandwidthManager_ContextCancellation(t *testing.T) {
	bm := NewBandwidthManager()
	bm.SetLimit(1) // Very slow — 1 byte/sec

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Should return immediately with context error
	err := bm.Wait(ctx, "task-1", 1024)
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
}

func TestBandwidthManager_LowPriorityDelay(t *testing.T) {
	bm := NewBandwidthManager()
	bm.SetLimit(1024 * 1024) // 1MB/s - fast enough
	bm.SetTaskPriority("low-task", 1)

	// Ensure the enabled flag is set
	var enabled atomic.Bool
	enabled.Store(bm.limitEnabled.Load())
	if !enabled.Load() {
		t.Fatal("limit should be enabled")
	}

	ctx := context.Background()
	start := time.Now()
	if err := bm.Wait(ctx, "low-task", 1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	elapsed := time.Since(start)

	// Low priority gets a 10ms artificial delay
	if elapsed < 5*time.Millisecond {
		t.Fatalf("expected low-priority delay, got %v", elapsed)
	}
}
