package network

import (
	"context"
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
	bm.SetTaskPriority("low-task", 1)
	bm.SetTaskPriority("high-task", 3)
	bm.MarkActive("low-task")
	bm.MarkActive("high-task")

	ctx := context.Background()
	start := time.Now()
	if err := bm.Wait(ctx, "low-task", 1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	elapsed := time.Since(start)

	// Low priority gets a 20ms delay when high priority is active
	if elapsed < 15*time.Millisecond {
		t.Fatalf("expected low-priority delay, got %v", elapsed)
	}
}

func TestBandwidthManager_NoPriorityDelayWhenAlone(t *testing.T) {
	bm := NewBandwidthManager()
	bm.SetTaskPriority("low-task", 1)
	bm.MarkActive("low-task")

	ctx := context.Background()
	start := time.Now()
	for i := 0; i < 10; i++ {
		if err := bm.Wait(ctx, "low-task", 1024); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}
	elapsed := time.Since(start)

	// No delay when no higher-priority task is competing
	if elapsed > 50*time.Millisecond {
		t.Fatalf("low-priority alone should not be delayed, got %v", elapsed)
	}
}

func TestBandwidthManager_HighPriorityNoDelay(t *testing.T) {
	bm := NewBandwidthManager()
	bm.SetTaskPriority("high-task", 3)
	bm.SetTaskPriority("low-task", 1)
	bm.MarkActive("high-task")
	bm.MarkActive("low-task")

	ctx := context.Background()
	start := time.Now()
	for i := 0; i < 10; i++ {
		if err := bm.Wait(ctx, "high-task", 1024); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}
	elapsed := time.Since(start)

	// High priority never delayed
	if elapsed > 50*time.Millisecond {
		t.Fatalf("high-priority should not be delayed, got %v", elapsed)
	}
}

func TestBandwidthManager_MarkInactive(t *testing.T) {
	bm := NewBandwidthManager()
	bm.SetTaskPriority("low-task", 1)
	bm.SetTaskPriority("high-task", 3)
	bm.MarkActive("low-task")
	bm.MarkActive("high-task")

	// Remove high-priority task
	bm.MarkInactive("high-task")

	ctx := context.Background()
	start := time.Now()
	if err := bm.Wait(ctx, "low-task", 1024); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	elapsed := time.Since(start)

	// No delay since higher priority is no longer active
	if elapsed > 10*time.Millisecond {
		t.Fatalf("expected no delay after high-priority removed, got %v", elapsed)
	}
}
