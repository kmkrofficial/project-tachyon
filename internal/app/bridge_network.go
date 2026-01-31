package app

import (
	"project-tachyon/internal/core"
	"project-tachyon/internal/storage"
	"project-tachyon/internal/updater"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// RunNetworkSpeedTest performs a network speed test with live updates
func (a *App) RunNetworkSpeedTest() *core.SpeedTestResult {
	// Emit phase updates to frontend during speed test
	onPhase := func(phase core.SpeedTestPhase) {
		runtime.EventsEmit(a.ctx, "speedtest:phase", map[string]interface{}{
			"phase":         phase.Phase,
			"ping_ms":       phase.PingMs,
			"download_mbps": phase.DownloadMbps,
			"upload_mbps":   phase.UploadMbps,
			"server_name":   phase.ServerName,
			"isp":           phase.ISP,
		})
	}

	res, err := core.RunSpeedTestWithEvents(onPhase)
	if err != nil {
		a.logger.Error("Speed test failed", "error", err)
		runtime.EventsEmit(a.ctx, "speedtest:phase", map[string]interface{}{
			"phase": "error",
			"error": err.Error(),
		})
		return nil
	}

	// Persist to history
	history := storage.SpeedTestHistory{
		DownloadSpeed:  res.DownloadSpeed,
		UploadSpeed:    res.UploadSpeed,
		Ping:           res.Ping,
		Jitter:         res.Jitter,
		ISP:            res.ISP,
		ServerName:     res.ServerName,
		ServerLocation: res.ServerLocation,
		Timestamp:      res.Timestamp,
	}
	if err := a.engine.GetStorage().SaveSpeedTest(history); err != nil {
		a.logger.Error("Failed to save speed test history", "error", err)
	}

	return res
}

// GetSpeedTestHistory returns the last 10 speed tests
func (a *App) GetSpeedTestHistory() []storage.SpeedTestHistory {
	history, err := a.engine.GetStorage().GetSpeedTestHistory(10)
	if err != nil {
		a.logger.Error("Failed to get speed test history", "error", err)
		return []storage.SpeedTestHistory{}
	}
	return history
}

// GetLifetimeStats returns total bytes downloaded over lifetime
func (a *App) GetLifetimeStats() int64 {
	stats := a.engine.GetStats()
	if stats == nil {
		return 0
	}
	lifetime, _ := stats.GetLifetimeStats()
	return lifetime
}

// GetAnalytics returns comprehensive analytics data including disk usage
func (a *App) GetAnalytics() core.AnalyticsData {
	stats := a.engine.GetStats()
	if stats == nil {
		return core.AnalyticsData{}
	}
	return stats.GetAnalytics()
}

// GetDiskUsage returns disk space info for the download drive
func (a *App) GetDiskUsage() core.DiskUsageInfo {
	stats := a.engine.GetStats()
	if stats == nil {
		return core.DiskUsageInfo{}
	}
	return stats.GetDiskUsage()
}

// checkUpdaterPackage wraps the updater package call
func checkUpdaterPackage(currentVersion, owner, repo string) (*updater.Release, error) {
	return updater.CheckForUpdates(currentVersion, owner, repo)
}
