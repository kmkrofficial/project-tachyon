package network

import (
	"testing"
	"time"
)

func TestCongestionController_SlowStart(t *testing.T) {
	cc := NewCongestionController(4, 24)
	// Unknown host starts at minWorkers
	ideal := cc.GetIdealConcurrency("new-host.com")
	if ideal != 4 {
		t.Fatalf("expected slow start at 4, got %d", ideal)
	}
}

func TestCongestionController_AdditiveIncrease(t *testing.T) {
	cc := NewCongestionController(4, 24)

	// Record enough successes to trigger AI
	for i := 0; i < 6; i++ {
		cc.RecordOutcome("host.com", 50*time.Millisecond, nil)
	}
	ideal := cc.GetIdealConcurrency("host.com")
	if ideal <= 4 {
		t.Fatalf("expected concurrency to increase from slow-start, got %d", ideal)
	}
}

func TestCongestionController_MultiplicativeDecrease(t *testing.T) {
	cc := NewCongestionController(4, 24)

	// Build up some concurrency
	for i := 0; i < 20; i++ {
		cc.RecordOutcome("host.com", 50*time.Millisecond, nil)
	}
	beforeError := cc.GetIdealConcurrency("host.com")

	// Record an error
	cc.RecordOutcome("host.com", 50*time.Millisecond, errTestSentinel)
	afterError := cc.GetIdealConcurrency("host.com")

	if afterError >= beforeError {
		t.Fatalf("expected MD after error: before=%d, after=%d", beforeError, afterError)
	}
}

func TestCongestionController_RespectsBounds(t *testing.T) {
	cc := NewCongestionController(2, 8)

	// Push up concurrency
	for i := 0; i < 200; i++ {
		cc.RecordOutcome("host.com", 10*time.Millisecond, nil)
		ideal := cc.GetIdealConcurrency("host.com")
		if ideal > 8 {
			t.Fatalf("exceeded max bound: %d", ideal)
		}
	}
}

func TestCongestionController_GetHostStats(t *testing.T) {
	cc := NewCongestionController(4, 24)
	if cc.GetHostStats("ghost.com") != nil {
		t.Fatal("expected nil stats for unknown host")
	}

	cc.RecordOutcome("real.com", 100*time.Millisecond, nil)
	stats := cc.GetHostStats("real.com")
	if stats == nil {
		t.Fatal("expected non-nil stats after recording outcome")
	}
	if stats.SuccessCount != 1 {
		t.Fatalf("expected 1 success, got %d", stats.SuccessCount)
	}
}

var errTestSentinel = errForTest("test error")

type errForTest string

func (e errForTest) Error() string { return string(e) }
