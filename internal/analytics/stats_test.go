package analytics

import (
	"project-tachyon/internal/storage"
	"strings"
	"testing"
)

// mockDownloadPathFn is a test helper that returns a predictable path
func mockDownloadPathFn() (string, error) {
	return "C:\\Users\\test\\Downloads", nil
}

func TestStatsManager(t *testing.T) {
	s, err := storage.NewStorage()
	if err != nil {
		if strings.Contains(err.Error(), "lock") || strings.Contains(err.Error(), "LOCK") {
			t.Skip("Skipping test - database locked (app running)")
		}
		t.Fatalf("Failed to init storage: %v", err)
	}
	defer s.Close()

	sm := NewStatsManager(s, mockDownloadPathFn)
	if sm == nil {
		t.Fatal("NewStatsManager returned nil")
	}

	// Test TrackDownloadBytes (fire and forget, no error)
	sm.TrackDownloadBytes(1024)

	// Test TrackFileCompleted
	sm.TrackFileCompleted()

	// Test GetLifetimeStats
	_, err = sm.GetLifetimeStats()
	if err != nil {
		t.Errorf("GetLifetimeStats returned error: %v", err)
	}

	// Test GetTotalFiles
	_, err = sm.GetTotalFiles()
	if err != nil {
		t.Errorf("GetTotalFiles returned error: %v", err)
	}

	// Test GetDailyStats (returns up to N days, may be less if no data)
	daily, err := sm.GetDailyStats(7)
	if err != nil {
		t.Errorf("GetDailyStats returned error: %v", err)
	}
	if len(daily) > 7 {
		t.Errorf("Expected at most 7 days of stats, got %d", len(daily))
	}

	// Test GetDiskUsage
	usage := sm.GetDiskUsage()
	if usage.Percent < 0 || usage.Percent > 100 {
		t.Errorf("Disk usage percent out of range: %f", usage.Percent)
	}
	t.Logf("Disk Usage: %.2f GB used of %.2f GB total (%.1f%%)", usage.UsedGB, usage.TotalGB, usage.Percent)

	// Test GetAnalytics
	analyticsData := sm.GetAnalytics()
	if len(analyticsData.DailyHistory) > 7 {
		t.Errorf("Expected at most 7 days of history, got %d", len(analyticsData.DailyHistory))
	}
}
