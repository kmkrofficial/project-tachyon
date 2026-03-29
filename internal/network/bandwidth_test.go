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
