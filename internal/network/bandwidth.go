// Package network provides bandwidth management and congestion control
// for download operations.
package network

import (
	"context"
	"sync/atomic"

	"golang.org/x/time/rate"
)

// BandwidthManager handles global speed limiting for concurrent downloads.
type BandwidthManager struct {
	globalLimiter *rate.Limiter
	limitEnabled  atomic.Bool
}

// NewBandwidthManager creates a new bandwidth manager with no limits
func NewBandwidthManager() *BandwidthManager {
	return &BandwidthManager{
		globalLimiter: rate.NewLimiter(rate.Inf, 0),
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

// Wait blocks until the requested bytes can be consumed under the global
// rate limit.  Returns immediately when no limit is configured.
func (bm *BandwidthManager) Wait(ctx context.Context, taskID string, bytes int) error {
	if bm.limitEnabled.Load() {
		return bm.globalLimiter.WaitN(ctx, bytes)
	}
	return nil
}
