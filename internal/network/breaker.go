package network

import (
	"fmt"
	"sync"
	"time"
)

// BreakerState represents the current state of a circuit breaker.
type BreakerState int

const (
	BreakerClosed   BreakerState = iota // Normal operation
	BreakerOpen                         // Failing, reject requests
	BreakerHalfOpen                     // Testing with limited traffic
)

// CircuitBreaker implements per-host circuit breaking to prevent
// workers from hammering a failing server.
type CircuitBreaker struct {
	mu    sync.Mutex
	hosts map[string]*hostBreaker

	// Thresholds (configurable at construction)
	failThreshold int           // consecutive failures to trip
	cooldown      time.Duration // time before half-open probe
}

type hostBreaker struct {
	state       BreakerState
	failures    int       // consecutive failure count
	lastFailure time.Time // timestamp of last failure (for cooldown)
	successes   int       // consecutive successes in half-open state
}

// NewCircuitBreaker creates a breaker with sensible defaults for download workloads.
func NewCircuitBreaker(failThreshold int, cooldown time.Duration) *CircuitBreaker {
	if failThreshold < 1 {
		failThreshold = 5
	}
	if cooldown < time.Second {
		cooldown = 30 * time.Second
	}
	return &CircuitBreaker{
		hosts:         make(map[string]*hostBreaker),
		failThreshold: failThreshold,
		cooldown:      cooldown,
	}
}

// Allow checks whether a request to host should proceed.
// Returns an error when the breaker is open.
func (cb *CircuitBreaker) Allow(host string) error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	hb, ok := cb.hosts[host]
	if !ok {
		return nil // No state yet — allow
	}

	switch hb.state {
	case BreakerClosed:
		return nil
	case BreakerOpen:
		if time.Since(hb.lastFailure) >= cb.cooldown {
			hb.state = BreakerHalfOpen
			hb.successes = 0
			return nil // Allow probe request
		}
		return fmt.Errorf("circuit open for host %s, retry after cooldown", host)
	case BreakerHalfOpen:
		return nil // Allow limited traffic to test recovery
	}
	return nil
}

// RecordSuccess signals a successful request to host.
func (cb *CircuitBreaker) RecordSuccess(host string) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	hb := cb.getOrCreate(host)
	hb.failures = 0

	if hb.state == BreakerHalfOpen {
		hb.successes++
		// After 2 consecutive successes in half-open, close the breaker.
		if hb.successes >= 2 {
			hb.state = BreakerClosed
		}
	}
}

// RecordFailure signals a failed request to host.
func (cb *CircuitBreaker) RecordFailure(host string) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	hb := cb.getOrCreate(host)
	hb.failures++
	hb.lastFailure = time.Now()

	if hb.state == BreakerHalfOpen {
		// Any failure in half-open immediately re-opens.
		hb.state = BreakerOpen
		return
	}

	if hb.failures >= cb.failThreshold {
		hb.state = BreakerOpen
	}
}

// State returns the current breaker state for a host (for monitoring).
func (cb *CircuitBreaker) State(host string) BreakerState {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	hb, ok := cb.hosts[host]
	if !ok {
		return BreakerClosed
	}
	return hb.state
}

func (cb *CircuitBreaker) getOrCreate(host string) *hostBreaker {
	hb, ok := cb.hosts[host]
	if !ok {
		hb = &hostBreaker{state: BreakerClosed}
		cb.hosts[host] = hb
	}
	return hb
}
