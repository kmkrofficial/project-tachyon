package main

import (
	"io"
	"sync"
	"time"
)

// globalThrottle is a token-bucket rate limiter shared across all connections.
// When bytesPerSec <= 0, no throttling is applied.
type globalThrottle struct {
	mu          sync.Mutex
	bytesPerSec int64
	tokens      int64
	maxBurst    int64
	lastRefill  time.Time
}

func newGlobalThrottle(bytesPerSec int64) *globalThrottle {
	burst := bytesPerSec
	if burst < 32*1024 {
		burst = 32 * 1024
	}
	return &globalThrottle{
		bytesPerSec: bytesPerSec,
		tokens:      burst,
		maxBurst:    burst,
		lastRefill:  time.Now(),
	}
}

// take blocks until n tokens are available, returning the granted amount.
func (g *globalThrottle) take(n int64) int64 {
	if g == nil || g.bytesPerSec <= 0 {
		return n
	}

	for {
		g.mu.Lock()
		now := time.Now()
		elapsed := now.Sub(g.lastRefill).Seconds()
		if elapsed > 0 {
			refill := int64(elapsed * float64(g.bytesPerSec))
			g.tokens += refill
			if g.tokens > g.maxBurst {
				g.tokens = g.maxBurst
			}
			g.lastRefill = now
		}

		if g.tokens > 0 {
			grant := n
			if grant > g.tokens {
				grant = g.tokens
			}
			g.tokens -= grant
			g.mu.Unlock()
			return grant
		}
		g.mu.Unlock()
		time.Sleep(10 * time.Millisecond)
	}
}

// throttledWriter wraps an io.Writer with global rate limiting.
type throttledWriter struct {
	w        io.Writer
	throttle *globalThrottle
}

func (tw *throttledWriter) Write(p []byte) (int, error) {
	if tw.throttle == nil || tw.throttle.bytesPerSec <= 0 {
		return tw.w.Write(p)
	}

	var written int
	for written < len(p) {
		chunk := tw.throttle.take(int64(len(p) - written))
		n, err := tw.w.Write(p[written : written+int(chunk)])
		written += n
		if err != nil {
			return written, err
		}
	}
	return written, nil
}
