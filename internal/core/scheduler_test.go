package core

import (
	"log/slog"
	"os"
	"testing"
)

func TestSchedulerConfig(t *testing.T) {
	// Mock Engine (nil for this test as we don't execute engine methods)
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	sched := NewScheduler(logger, nil)

	cfg := ScheduleConfig{
		Enabled:   true,
		StartHour: 2,
		StopHour:  8,
	}

	sched.UpdateSchedule(cfg)

	if len(sched.cron.Entries()) != 2 {
		t.Errorf("Expected 2 cron entries, got %d", len(sched.cron.Entries()))
	}

	// Verify Entry Specs?
	// robfig/cron methods are limited for inspection, but count is good proxy.

	sched.Stop()
}
