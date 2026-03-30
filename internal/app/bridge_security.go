package app

import (
	"project-tachyon/internal/integrity"
	"project-tachyon/internal/security"
)

// Security / AI Interface Methods

// GetEnableAI returns whether the AI interface is enabled
func (a *App) GetEnableAI() bool {
	return a.cfg.GetEnableAI()
}

// SetEnableAI toggles the AI interface
func (a *App) SetEnableAI(enabled bool) {
	a.cfg.SetEnableAI(enabled)
	a.logger.Info("AI Interface setting changed", "enabled", enabled)
}

// GetAIToken returns the AI interface token
func (a *App) GetAIToken() string {
	return a.cfg.GetAIToken()
}

// GetAIPort returns the AI interface port
func (a *App) GetAIPort() int {
	return a.cfg.GetAIPort()
}

// SetAIPort sets the AI interface port (requires restart)
func (a *App) SetAIPort(port int) {
	a.cfg.SetAIPort(port)
	a.logger.Info("AI Port setting changed (requires restart)", "port", port)
}

// GetAIMaxConcurrent returns the max concurrent AI requests
func (a *App) GetAIMaxConcurrent() int {
	return a.cfg.GetAIMaxConcurrent()
}

// SetAIMaxConcurrent sets the max concurrent AI requests
func (a *App) SetAIMaxConcurrent(max int) {
	a.cfg.SetAIMaxConcurrent(max)
	a.logger.Info("AI Max Concurrent setting changed", "max", max)
}

// GetRecentAuditLogs returns recent security audit logs
func (a *App) GetRecentAuditLogs() []security.AccessLogEntry {
	if a.audit == nil {
		return []security.AccessLogEntry{}
	}
	return a.audit.GetRecentLogs(50)
}

// GetAVScannerInfo returns the scanner name and whether it is available on this system
func (a *App) GetAVScannerInfo() map[string]interface{} {
	scanner := a.engine.GetScanner()
	return map[string]interface{}{
		"name":      scanner.Name(),
		"available": scanner.IsAvailable(),
	}
}

// GetEnableAVScan returns whether AV scanning of completed downloads is enabled
func (a *App) GetEnableAVScan() bool {
	return a.cfg.GetEnableAVScan()
}

// SetEnableAVScan toggles AV scanning of completed downloads
func (a *App) SetEnableAVScan(enabled bool) {
	a.cfg.SetEnableAVScan(enabled)
	a.logger.Info("AV scan setting changed", "enabled", enabled)
}

// CalculateHash computes the hash of a file for checksum verification
// algorithm should be "sha256" or "md5"
func (a *App) CalculateHash(filePath string, algorithm string) (string, error) {
	a.logger.Info("frontend_request", "method", "CalculateHash", "path", filePath, "algorithm", algorithm)
	return integrity.CalculateHash(filePath, algorithm)
}

// GetUserAgent returns the current custom User-Agent
func (a *App) GetUserAgent() string {
	return a.engine.GetUserAgent()
}

// SetUserAgent sets a custom User-Agent for all downloads
func (a *App) SetUserAgent(userAgent string) {
	a.logger.Info("frontend_request", "method", "SetUserAgent", "user_agent", userAgent)
	a.engine.SetUserAgent(userAgent)
	// Persist to config
	if a.cfg != nil {
		a.cfg.SetUserAgent(userAgent)
	}
}
