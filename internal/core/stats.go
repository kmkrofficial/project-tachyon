package core

import (
	"project-tachyon/internal/storage"
	"sync"
	"time"
)

type StatsManager struct {
	storage *storage.Storage
	mu      sync.Mutex
	cache   map[string]interface{}
}

func NewStatsManager(s *storage.Storage) *StatsManager {
	return &StatsManager{
		storage: s,
		cache:   make(map[string]interface{}),
	}
}

// TrackDownloadBytes increments the daily and total counters
func (sm *StatsManager) TrackDownloadBytes(bytes int64) {
	go func() {
		// Optimization: Batching could be done here, but for now simple atomic-like writes via storage methods
		// We'll trust BadgerDB's speed or implement batching later if needed.

		today := time.Now().Format("2006-01-02")

		sm.storage.IncrementStat("stat_total_lifetime", bytes)
		sm.storage.IncrementStat("stat_daily_"+today, bytes)
	}()
}

func (sm *StatsManager) GetLifetimeStats() (int64, error) {
	return sm.storage.GetStatInt("stat_total_lifetime")
}

func (sm *StatsManager) GetDailyStats(days int) (map[string]int64, error) {
	// Return last N days
	res := make(map[string]int64)
	now := time.Now()
	for i := 0; i < days; i++ {
		date := now.AddDate(0, 0, -i).Format("2006-01-02")
		val, _ := sm.storage.GetStatInt("stat_daily_" + date)
		res[date] = val
	}
	return res, nil
}
