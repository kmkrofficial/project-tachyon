package core

import (
	"sync"
	"time"
)

// CongestionController implements an AIMD (Additive Increase, Multiplicative Decrease) algorithm
// to dynamically scale worker concurrency based on network conditions.
type CongestionController struct {
	mu         sync.RWMutex
	hosts      map[string]*HostStats
	baseRTT    time.Duration
	minWorkers int
	maxWorkers int
}

type HostStats struct {
	LastRTT      time.Duration
	SmoothedRTT  time.Duration // SRTT
	ErrorRate    float64       // Errors per minute (decaying)
	Concurrency  int
	LastUpdate   time.Time
	SuccessCount int
	ErrorCount   int
}

func NewCongestionController(min, max int) *CongestionController {
	return &CongestionController{
		hosts:      make(map[string]*HostStats),
		baseRTT:    100 * time.Millisecond, // Reasonable default
		minWorkers: min,
		maxWorkers: max,
	}
}

// RecordOutcome updates stats for a host based on a completed chunk download
func (cc *CongestionController) RecordOutcome(host string, latency time.Duration, err error) {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	stats, ok := cc.hosts[host]
	if !ok {
		stats = &HostStats{
			Concurrency: cc.minWorkers,
			SmoothedRTT: latency,
		}
		cc.hosts[host] = stats
	}

	// Exponential Moving Average for RTT
	alpha := 0.125
	stats.SmoothedRTT = time.Duration((1-alpha)*float64(stats.SmoothedRTT) + alpha*float64(latency))
	stats.LastRTT = latency
	stats.LastUpdate = time.Now()

	if err != nil {
		stats.ErrorCount++
	} else {
		stats.SuccessCount++
	}
}

// GetIdealConcurrency calculates the target worker count using AIMD logic
func (cc *CongestionController) GetIdealConcurrency(host string) int {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	stats, ok := cc.hosts[host]
	if !ok {
		return cc.minWorkers // Slow start
	}

	// Decrease on congestion (Packet Loss/Error or High Latency)
	// Thresholds: RTT > 2x Base (Variable) or recent errors

	// Check for errors (Naive "packet loss" equivalent)
	if stats.ErrorCount > 0 {
		// Multiplicative Decrease
		stats.Concurrency = max(1, stats.Concurrency/2)
		stats.ErrorCount = 0 // Reset after reacting
		return stats.Concurrency
	}

	// Check for High Latency (Bufferbloat detection)
	// If RTT is increasing significantly, hold or decrease
	// For simplicity, let's stick to Additive Increase if stable.

	// Additive Increase
	// Increase if stable and we have successful samples
	if stats.SuccessCount > stats.Concurrency {
		if stats.Concurrency < cc.maxWorkers {
			stats.Concurrency++
		}
		stats.SuccessCount = 0 // Reset for next window
	}

	return stats.Concurrency
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
