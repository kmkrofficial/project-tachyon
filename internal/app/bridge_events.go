package app

import (
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// ScanResultEvent represents a security scan result for frontend display
type ScanResultEvent struct {
	File       string `json:"file"`
	Status     string `json:"status"` // "clean", "threat", "error"
	ThreatName string `json:"threat_name,omitempty"`
	Timestamp  string `json:"timestamp"`
}

// NetworkHealthEvent represents network congestion status
type NetworkHealthEvent struct {
	Level   string `json:"level"` // "normal", "stressed", "critical"
	Details string `json:"details,omitempty"`
}

// EmitScanResult emits a security scan result event to the frontend
func (a *App) EmitScanResult(file, status, threatName string) {
	if a.ctx == nil {
		return
	}
	runtime.EventsEmit(a.ctx, "security:scan_result", ScanResultEvent{
		File:       file,
		Status:     status,
		ThreatName: threatName,
		Timestamp:  "", // Will be set by frontend
	})
}

// EmitNetworkHealth emits network congestion level to the frontend
func (a *App) EmitNetworkHealth(level, details string) {
	if a.ctx == nil {
		return
	}
	runtime.EventsEmit(a.ctx, "network:congestion_level", NetworkHealthEvent{
		Level:   level,
		Details: details,
	})
}

// GetNetworkHealth returns the current network health status
// Level is one of: "normal", "stressed", "critical"
func (a *App) GetNetworkHealth() NetworkHealthEvent {
	// Get congestion info from engine stats
	stats := a.engine.GetStats()
	if stats == nil {
		return NetworkHealthEvent{Level: "normal", Details: ""}
	}

	currentSpeed := stats.GetCurrentSpeed()

	level := "normal"
	details := ""

	// Get active download count from storage
	tasks, _ := a.engine.GetHistory()
	activeCount := 0
	for _, task := range tasks {
		if task.Status == "downloading" {
			activeCount++
		}
	}

	// Simple heuristic: if speed is very low but downloads are active, consider stressed
	if activeCount > 0 && currentSpeed < 100*1024 { // Less than 100 KB/s with active downloads
		level = "stressed"
		details = "Low transfer speeds detected"
	}

	return NetworkHealthEvent{
		Level:   level,
		Details: details,
	}
}
