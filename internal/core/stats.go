package core

import (
	"path/filepath"
	"project-tachyon/internal/storage"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/disk"
)

// DiskUsageInfo holds disk space information
type DiskUsageInfo struct {
	UsedGB  float64 `json:"used_gb"`
	FreeGB  float64 `json:"free_gb"`
	TotalGB float64 `json:"total_gb"`
	Percent float64 `json:"percent"`
}

// AnalyticsData holds all analytics information for the frontend
type AnalyticsData struct {
	TotalDownloaded int64            `json:"total_downloaded"`
	TotalFiles      int64            `json:"total_files"`
	DailyHistory    map[string]int64 `json:"daily_history"`
	DiskUsage       DiskUsageInfo    `json:"disk_usage"`
}

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
		today := time.Now().Format("2006-01-02")
		sm.storage.IncrementStat("stat_total_lifetime", bytes)
		sm.storage.IncrementStat("stat_daily_"+today, bytes)
	}()
}

// TrackFileCompleted increments the total file count
func (sm *StatsManager) TrackFileCompleted() {
	go func() {
		sm.storage.IncrementStat("stat_total_files", 1)
	}()
}

func (sm *StatsManager) GetLifetimeStats() (int64, error) {
	return sm.storage.GetStatInt("stat_total_lifetime")
}

func (sm *StatsManager) GetTotalFiles() (int64, error) {
	return sm.storage.GetStatInt("stat_total_files")
}

func (sm *StatsManager) GetDailyStats(days int) (map[string]int64, error) {
	res := make(map[string]int64)
	now := time.Now()
	for i := 0; i < days; i++ {
		date := now.AddDate(0, 0, -i).Format("2006-01-02")
		val, _ := sm.storage.GetStatInt("stat_daily_" + date)
		res[date] = val
	}
	return res, nil
}

// GetDiskUsage returns disk space info for the download drive
func (sm *StatsManager) GetDiskUsage() DiskUsageInfo {
	// Get the default download path to determine the drive
	downloadPath, err := GetDefaultDownloadPath()
	if err != nil {
		return DiskUsageInfo{} // Return zeros on error
	}

	// Get the volume root (e.g., C:\ on Windows, / on Unix)
	volumePath := filepath.VolumeName(downloadPath)
	if volumePath == "" {
		volumePath = "/"
	} else {
		volumePath += "\\"
	}

	usage, err := disk.Usage(volumePath)
	if err != nil {
		return DiskUsageInfo{} // Return zeros on error
	}

	const bytesPerGB = 1024 * 1024 * 1024
	return DiskUsageInfo{
		UsedGB:  float64(usage.Used) / bytesPerGB,
		FreeGB:  float64(usage.Free) / bytesPerGB,
		TotalGB: float64(usage.Total) / bytesPerGB,
		Percent: usage.UsedPercent,
	}
}

// GetAnalytics returns comprehensive analytics data
func (sm *StatsManager) GetAnalytics() AnalyticsData {
	lifetime, _ := sm.GetLifetimeStats()
	totalFiles, _ := sm.GetTotalFiles()
	daily, _ := sm.GetDailyStats(7)
	diskUsage := sm.GetDiskUsage()

	return AnalyticsData{
		TotalDownloaded: lifetime,
		TotalFiles:      totalFiles,
		DailyHistory:    daily,
		DiskUsage:       diskUsage,
	}
}
