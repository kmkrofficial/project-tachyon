package app

import (
	"context"
	"time"

	"project-tachyon/internal/network"
	"project-tachyon/internal/storage"
	"project-tachyon/internal/updater"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// RunNetworkSpeedTest performs a network speed test with live updates
func (a *App) RunNetworkSpeedTest() *network.SpeedTestResult {
	// Create a cancellable context for this test
	ctx, cancel := context.WithTimeout(a.ctx, 60*time.Second)

	a.speedTestMu.Lock()
	// Cancel any previous running test
	if a.speedTestCancel != nil {
		a.speedTestCancel()
	}
	a.speedTestCancel = cancel
	a.speedTestMu.Unlock()

	defer func() {
		cancel()
		a.speedTestMu.Lock()
		a.speedTestCancel = nil
		a.speedTestMu.Unlock()
	}()

	// Emit phase updates to frontend during speed test
	onPhase := func(phase network.SpeedTestPhase) {
		runtime.EventsEmit(a.ctx, "speedtest:phase", map[string]interface{}{
			"phase":         phase.Phase,
			"ping_ms":       phase.PingMs,
			"download_mbps": phase.DownloadMbps,
			"upload_mbps":   phase.UploadMbps,
			"server_name":   phase.ServerName,
			"isp":           phase.ISP,
		})
	}

	res, err := network.RunSpeedTestWithContext(ctx, onPhase)
	if err != nil {
		a.logger.Error("Speed test failed", "error", err)
		// Only emit error if not cancelled
		if ctx.Err() == nil {
			runtime.EventsEmit(a.ctx, "speedtest:phase", map[string]interface{}{
				"phase": "error",
				"error": err.Error(),
			})
		} else {
			runtime.EventsEmit(a.ctx, "speedtest:phase", map[string]interface{}{
				"phase": "cancelled",
			})
		}
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

// CancelSpeedTest cancels a running speed test
func (a *App) CancelSpeedTest() {
	a.speedTestMu.Lock()
	defer a.speedTestMu.Unlock()
	if a.speedTestCancel != nil {
		a.speedTestCancel()
		a.speedTestCancel = nil
	}
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

// ClearSpeedTestHistory deletes all speed test records
func (a *App) ClearSpeedTestHistory() error {
	return a.engine.GetStorage().ClearSpeedTestHistory()
}

// checkUpdaterPackage wraps the updater package call
func checkUpdaterPackage(currentVersion, owner, repo string) (*updater.Release, error) {
	return updater.CheckForUpdates(currentVersion, owner, repo)
}
