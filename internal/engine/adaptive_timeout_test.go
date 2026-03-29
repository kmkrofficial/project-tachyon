package engine

import (
	"testing"
	"time"
)

func TestAdaptiveStallTimeout_ZeroSpeed(t *testing.T) {
	timeout := adaptiveStallTimeout(0, BufferSize)
	if timeout != maxStallTimeout {
		t.Errorf("expected maxStallTimeout (%v), got %v", maxStallTimeout, timeout)
	}
}

func TestAdaptiveStallTimeout_NegativeSpeed(t *testing.T) {
	timeout := adaptiveStallTimeout(-100, BufferSize)
	if timeout != maxStallTimeout {
		t.Errorf("expected maxStallTimeout for negative speed, got %v", timeout)
	}
}

func TestAdaptiveStallTimeout_HighSpeed(t *testing.T) {
	// 100 MB/s — expect buffer fills in ~2.5ms, 3x = ~7.5ms, clamped to minStallTimeout
	speed := float64(100 * 1024 * 1024)
	timeout := adaptiveStallTimeout(speed, BufferSize)
	if timeout != minStallTimeout {
		t.Errorf("expected minStallTimeout (%v) for high speed, got %v", minStallTimeout, timeout)
	}
}

func TestAdaptiveStallTimeout_LowSpeed(t *testing.T) {
	// 10 KB/s — expect buffer fills in ~25s, 3x = ~77s, clamped to maxStallTimeout
	speed := float64(10 * 1024)
	timeout := adaptiveStallTimeout(speed, BufferSize)
	if timeout != maxStallTimeout {
		t.Errorf("expected maxStallTimeout (%v) for very low speed, got %v", maxStallTimeout, timeout)
	}
}

func TestAdaptiveStallTimeout_ModerateSpeed(t *testing.T) {
	// 1 MB/s — expect buffer fills in ~0.25s, 3x = ~0.75s, clamped to minStallTimeout
	speed := float64(1 * 1024 * 1024)
	timeout := adaptiveStallTimeout(speed, BufferSize)
	if timeout < minStallTimeout || timeout > maxStallTimeout {
		t.Errorf("timeout %v out of bounds [%v, %v]", timeout, minStallTimeout, maxStallTimeout)
	}
}

func TestAdaptiveStallTimeout_Bounds(t *testing.T) {
	speeds := []float64{1, 100, 1024, 10240, 102400, 1048576, 10485760, 104857600}
	for _, speed := range speeds {
		timeout := adaptiveStallTimeout(speed, BufferSize)
		if timeout < minStallTimeout {
			t.Errorf("speed=%.0f: timeout %v < min %v", speed, timeout, minStallTimeout)
		}
		if timeout > maxStallTimeout {
			t.Errorf("speed=%.0f: timeout %v > max %v", speed, timeout, maxStallTimeout)
		}
	}
}

func TestAdaptiveStallTimeout_TypeIsDuration(t *testing.T) {
	timeout := adaptiveStallTimeout(1000, BufferSize)
	var _ time.Duration = timeout // compile-time type check
}
