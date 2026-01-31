package app

import (
	"project-tachyon/internal/core"
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

// CalculateHash computes the hash of a file for checksum verification
// algorithm should be "sha256" or "md5"
func (a *App) CalculateHash(filePath string, algorithm string) (string, error) {
	a.logger.Info("frontend_request", "method", "CalculateHash", "path", filePath, "algorithm", algorithm)
	return core.CalculateHash(filePath, algorithm)
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
