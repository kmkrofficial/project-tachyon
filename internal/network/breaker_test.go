package network

import (
	"testing"
	"time"
)

func TestCircuitBreaker_AllowClosedByDefault(t *testing.T) {
	cb := NewCircuitBreaker(3, 10*time.Second)
	if err := cb.Allow("example.com"); err != nil {
		t.Fatalf("expected allow on fresh host, got: %v", err)
	}
}

func TestCircuitBreaker_OpensAfterThreshold(t *testing.T) {
	cb := NewCircuitBreaker(3, 10*time.Second)

	cb.RecordFailure("example.com")
	cb.RecordFailure("example.com")
	if err := cb.Allow("example.com"); err != nil {
		t.Fatal("should still be closed after 2 failures")
	}

	cb.RecordFailure("example.com") // 3rd — trips
	if err := cb.Allow("example.com"); err == nil {
		t.Fatal("expected breaker to be open after 3 consecutive failures")
	}
	if cb.State("example.com") != BreakerOpen {
		t.Fatalf("expected BreakerOpen, got %d", cb.State("example.com"))
	}
}

func TestCircuitBreaker_SuccessResetsFailures(t *testing.T) {
	cb := NewCircuitBreaker(3, 10*time.Second)
	cb.RecordFailure("example.com")
	cb.RecordFailure("example.com")
	cb.RecordSuccess("example.com") // Reset
	cb.RecordFailure("example.com") // Only 1 now

	if err := cb.Allow("example.com"); err != nil {
		t.Fatal("breaker should still be closed after success reset")
	}
}

func TestCircuitBreaker_HalfOpenAfterCooldown(t *testing.T) {
	cb := NewCircuitBreaker(2, 50*time.Millisecond)
	cb.RecordFailure("example.com")
	cb.RecordFailure("example.com")

	if err := cb.Allow("example.com"); err == nil {
		t.Fatal("should be open immediately after tripping")
	}

	time.Sleep(60 * time.Millisecond)

	if err := cb.Allow("example.com"); err != nil {
		t.Fatalf("expected half-open after cooldown, got: %v", err)
	}
	if cb.State("example.com") != BreakerHalfOpen {
		t.Fatalf("expected BreakerHalfOpen, got %d", cb.State("example.com"))
	}
}

func TestCircuitBreaker_HalfOpenClosesOnSuccess(t *testing.T) {
	cb := NewCircuitBreaker(2, 50*time.Millisecond)
	cb.RecordFailure("example.com")
	cb.RecordFailure("example.com")
	time.Sleep(60 * time.Millisecond)

	cb.Allow("example.com") // Transitions to half-open
	cb.RecordSuccess("example.com")
	cb.RecordSuccess("example.com") // 2 successes closes it

	if cb.State("example.com") != BreakerClosed {
		t.Fatalf("expected BreakerClosed after 2 half-open successes, got %d", cb.State("example.com"))
	}
}

func TestCircuitBreaker_HalfOpenReopensOnFailure(t *testing.T) {
	cb := NewCircuitBreaker(2, 50*time.Millisecond)
	cb.RecordFailure("example.com")
	cb.RecordFailure("example.com")
	time.Sleep(60 * time.Millisecond)

	cb.Allow("example.com") // half-open
	cb.RecordFailure("example.com")

	if cb.State("example.com") != BreakerOpen {
		t.Fatalf("expected BreakerOpen after half-open failure, got %d", cb.State("example.com"))
	}
}

func TestCircuitBreaker_IndependentPerHost(t *testing.T) {
	cb := NewCircuitBreaker(2, 10*time.Second)
	cb.RecordFailure("bad.com")
	cb.RecordFailure("bad.com")

	if err := cb.Allow("bad.com"); err == nil {
		t.Fatal("bad.com should be open")
	}
	if err := cb.Allow("good.com"); err != nil {
		t.Fatal("good.com should be unaffected")
	}
}

func TestCircuitBreaker_StateDefaultClosed(t *testing.T) {
	cb := NewCircuitBreaker(5, 30*time.Second)
	if cb.State("unknown.com") != BreakerClosed {
		t.Fatal("unknown host should report closed")
	}
}
