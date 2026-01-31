package main

import (
	"context"
	"log/slog"
	"os"
	"project-tachyon/internal/config"
	"project-tachyon/internal/core"
	"project-tachyon/internal/logger"
	"project-tachyon/internal/security"
	"project-tachyon/internal/storage"
	"project-tachyon/internal/updater"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx          context.Context
	logger       *slog.Logger
	wailsHandler *logger.WailsHandler
	engine       *core.TachyonEngine
	cfg          *config.ConfigManager
	audit        *security.AuditLogger
	isQuitting   bool
}

// NewApp creates a new App application struct
func NewApp(logger *slog.Logger, engine *core.TachyonEngine, wailsHandler *logger.WailsHandler, cfg *config.ConfigManager, audit *security.AuditLogger) *App {
	return &App{
		logger:       logger,
		engine:       engine,
		wailsHandler: wailsHandler,
		cfg:          cfg,
		audit:        audit,
		isQuitting:   false,
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.engine.SetContext(ctx)
	if a.wailsHandler != nil {
		a.wailsHandler.SetContext(ctx)
	}
	a.logger.Info("App started")
	// Set context for audit logger
	if a.audit != nil {
		a.audit.SetContext(ctx)
	}
}

// Security / AI Interface Methods

func (a *App) GetEnableAI() bool {
	return a.cfg.GetEnableAI()
}

func (a *App) SetEnableAI(enabled bool) {
	a.cfg.SetEnableAI(enabled)
	a.logger.Info("AI Interface setting changed", "enabled", enabled)
}

func (a *App) GetAIToken() string {
	return a.cfg.GetAIToken()
}

func (a *App) GetAIPort() int {
	return a.cfg.GetAIPort()
}

func (a *App) SetAIPort(port int) {
	a.cfg.SetAIPort(port)
	a.logger.Info("AI Port setting changed (requires restart)", "port", port)
}

func (a *App) GetAIMaxConcurrent() int {
	return a.cfg.GetAIMaxConcurrent()
}

func (a *App) SetAIMaxConcurrent(max int) {
	a.cfg.SetAIMaxConcurrent(max)
	a.logger.Info("AI Max Concurrent setting changed", "max", max)
}

func (a *App) GetRecentAuditLogs() []security.AccessLogEntry {
	if a.audit == nil {
		return []security.AccessLogEntry{}
	}
	return a.audit.GetRecentLogs(50)
}

// beforeClose is called when the application is about to close.
// We return true to prevent closing (and hide instead), unless isQuitting is true.
func (a *App) beforeClose(ctx context.Context) (prevent bool) {
	if a.isQuitting {
		return false // Allow close
	}

	// Hide window instead of closing
	a.logger.Info("Window close requested, minimizing to tray")
	runtime.WindowHide(ctx)
	return true // Prevent close
}

// QuitApp is called from the Tray menu to truly exit
func (a *App) QuitApp() {
	a.isQuitting = true
	// Ensure engine shuts down gracefully
	if err := a.engine.Shutdown(); err != nil {
		a.logger.Error("Error during shutdown", "error", err)
	}
	runtime.Quit(a.ctx)
}

// ShowApp is called from the Tray menu to restore the window
func (a *App) ShowApp() {
	runtime.WindowShow(a.ctx)
	if runtime.WindowIsMinimised(a.ctx) {
		runtime.WindowUnminimise(a.ctx)
	}
	runtime.WindowSetAlwaysOnTop(a.ctx, true) // Bring to front
	runtime.WindowSetAlwaysOnTop(a.ctx, false)
}

// AddDownload is exposed to the Frontend
func (a *App) AddDownload(url string) string {
	a.logger.Info("frontend_request", "method", "AddDownload", "url", url)

	// Use Tachyon Downloads folder (auto-created with subfolders)
	defaultPath, err := core.GetDefaultDownloadPath()
	if err != nil {
		a.logger.Error("Failed to get default download path", "error", err)
		return "ERROR: " + err.Error()
	}

	id, err := a.engine.StartDownload(url, defaultPath, "", nil)
	if err != nil {
		a.logger.Error("Failed to start download", "error", err)
		return "ERROR: " + err.Error()
	}

	return id
}

// AddDownloadWithFilename allows specifying the filename (e.g. for duplicates)
func (a *App) AddDownloadWithFilename(url, filename string) string {
	a.logger.Info("frontend_request", "method", "AddDownloadWithFilename", "url", url, "filename", filename)

	defaultPath, err := core.GetDefaultDownloadPath()
	if err != nil {
		a.logger.Error("Failed to get default download path", "error", err)
		return "ERROR: " + err.Error()
	}

	id, err := a.engine.StartDownload(url, defaultPath, filename, nil)
	if err != nil {
		a.logger.Error("Failed to start download", "error", err)
		return "ERROR: " + err.Error()
	}

	return id
}

// AddDownloadWithOptions allows specifying path and filename
func (a *App) AddDownloadWithOptions(url, path, filename string) string {
	a.logger.Info("frontend_request", "method", "AddDownloadWithOptions", "url", url, "path", path, "filename", filename)

	if path == "" {
		var err error
		path, err = core.GetDefaultDownloadPath()
		if err != nil {
			a.logger.Error("Failed to get default download path", "error", err)
			return "ERROR: " + err.Error()
		}
	}

	id, err := a.engine.StartDownload(url, path, filename, nil)
	if err != nil {
		a.logger.Error("Failed to start download", "error", err)
		return "ERROR: " + err.Error()
	}

	return id
}

// AddDownloadWithParams allows specifying options like StartTime, Headers, Cookies, etc.
func (a *App) AddDownloadWithParams(url, path, filename string, options map[string]string) string {
	a.logger.Info("frontend_request", "method", "AddDownloadWithParams", "url", url, "options", options)

	if path == "" {
		var err error
		path, err = core.GetDefaultDownloadPath()
		if err != nil {
			a.logger.Error("Failed to get default download path", "error", err)
			return "ERROR: " + err.Error()
		}
	}

	id, err := a.engine.StartDownload(url, path, filename, options)
	if err != nil {
		a.logger.Error("Failed to start download", "error", err)
		return "ERROR: " + err.Error()
	}

	return id
}

// GetDownloadLocations returns saved download paths
func (a *App) GetDownloadLocations() []storage.DownloadLocation {
	locs, err := a.engine.GetStorage().GetLocations()
	if err != nil {
		a.logger.Error("Failed to get download locations", "error", err)
		return []storage.DownloadLocation{}
	}
	return locs
}

// AddDownloadLocation saves a new download path
func (a *App) AddDownloadLocation(path, nickname string) {
	if err := a.engine.GetStorage().AddLocation(path, nickname); err != nil {
		a.logger.Error("Failed to add download location", "error", err)
	}
}

// PauseDownload pauses/cancels an active download
func (a *App) PauseDownload(id string) {
	a.logger.Info("frontend_request", "method", "PauseDownload", "id", id)
	if err := a.engine.PauseDownload(id); err != nil {
		a.logger.Error("Failed to pause download", "id", id, "error", err)
	}
}

// ResumeDownload resumes a paused or stopped download
func (a *App) ResumeDownload(id string) error {
	a.logger.Info("frontend_request", "method", "ResumeDownload", "id", id)
	if err := a.engine.ResumeDownload(id); err != nil {
		a.logger.Error("Failed to resume download", "id", id, "error", err)
		return err
	}
	return nil
}

// PauseAllDownloads pauses all downloads
func (a *App) PauseAllDownloads() {
	a.logger.Info("frontend_request", "method", "PauseAllDownloads")
	a.engine.PauseAllDownloads()
}

// ResumeAllDownloads resumes all downloads
func (a *App) ResumeAllDownloads() {
	a.logger.Info("frontend_request", "method", "ResumeAllDownloads")
	a.engine.ResumeAllDownloads()
}

// UpdateDownloadURL updates the URL for a task that needs authentication refresh
// This is used when a download link has expired (HTTP 403) and needs a new URL
func (a *App) UpdateDownloadURL(taskID, newURL string) error {
	a.logger.Info("frontend_request", "method", "UpdateDownloadURL", "taskID", taskID)
	return a.engine.UpdateDownloadURL(taskID, newURL)
}

// StopDownload stops a download permanently (can still be resumed manually)
func (a *App) StopDownload(id string) {
	a.logger.Info("frontend_request", "method", "StopDownload", "id", id)
	if err := a.engine.StopDownload(id); err != nil {
		a.logger.Error("Failed to stop download", "id", id, "error", err)
	}
}

// DeleteDownload deletes a download task and optionally the file
func (a *App) DeleteDownload(id string, deleteFile bool) {
	a.logger.Info("frontend_request", "method", "DeleteDownload", "id", id, "deleteFile", deleteFile)
	if err := a.engine.DeleteDownload(id, deleteFile); err != nil {
		a.logger.Error("Failed to delete download", "id", id, "error", err)
	}
}

// ReorderDownload moves a download in the queue
// direction: "first", "prev", "next", "last"
func (a *App) ReorderDownload(id string, direction string) error {
	a.logger.Info("frontend_request", "method", "ReorderDownload", "id", id, "direction", direction)
	return a.engine.ReorderDownload(id, direction)
}

// SetPriority sets the priority of a download
func (a *App) SetPriority(id string, priority int) error {
	a.logger.Info("frontend_request", "method", "SetPriority", "id", id, "priority", priority)
	return a.engine.SetPriority(id, priority)
}

// SetGlobalSpeedLimit sets the global download speed limit
func (a *App) SetGlobalSpeedLimit(bytesPerSec int) {
	a.logger.Info("frontend_request", "method", "SetGlobalSpeedLimit", "bytesPerSec", bytesPerSec)
	a.engine.SetGlobalLimit(bytesPerSec)
	// TODO: Save to Settings Manager
	if a.cfg != nil {
		// Implement saving if settings key exists
		// a.cfg.SetInt("global_speed_limit", bytesPerSec)
	}
}

// SetMaxConcurrentDownloads sets the maximum number of concurrent downloads
func (a *App) SetMaxConcurrentDownloads(n int) {
	a.logger.Info("frontend_request", "method", "SetMaxConcurrentDownloads", "n", n)
	a.engine.SetMaxConcurrent(n)
}

func (a *App) SetHostLimit(domain string, limit int) {
	a.logger.Info("frontend_request", "method", "SetHostLimit", "domain", domain, "limit", limit)
	a.engine.SetHostLimit(domain, limit)
}

func (a *App) GetHostLimit(domain string) int {
	return a.engine.GetHostLimit(domain)
}

// GetQueuedDownloads returns all downloads currently in the queue
func (a *App) GetQueuedDownloads() []map[string]interface{} {
	items := a.engine.GetQueuedDownloads()
	result := make([]map[string]interface{}, len(items))
	for i, item := range items {
		// Check file existence
		fileExists := false
		if item.SavePath != "" {
			if _, err := os.Stat(item.SavePath); err == nil {
				fileExists = true
			}
		}

		result[i] = map[string]interface{}{
			"id":          item.ID,
			"filename":    item.Filename,
			"queue_order": item.QueueOrder,
			"status":      item.Status,
			"file_exists": fileExists,
		}
	}
	return result
}

// ProbeURL checks the URL
func (a *App) ProbeURL(url string) (*core.ProbeResult, error) {
	res, err := a.engine.ProbeURL(url, "", "")
	if err != nil {
		a.logger.Error("Probe failed", "url", url, "error", err)
		return nil, err
	}
	return res, nil
}

// CheckHistory checks DB for duplicates
func (a *App) CheckHistory(url string) bool {
	exists, err := a.engine.CheckHistory(url)
	if err != nil {
		return false
	}
	return exists
}

type CollisionResult struct {
	Exists bool   `json:"exists"`
	Path   string `json:"path"`
}

// CheckCollision checks file system
func (a *App) CheckCollision(filename string) CollisionResult {
	exists, path, err := a.engine.CheckCollision(filename)
	if err != nil {
		return CollisionResult{Exists: false, Path: ""}
	}
	return CollisionResult{Exists: exists, Path: path}
}

// VerifyFileExists checks if a file exists at the given path
func (a *App) VerifyFileExists(path string) bool {
	if path == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}

// GetTasks returns all saved tasks from the database
func (a *App) GetTasks() []storage.Task {
	tasks, err := a.engine.GetHistory()
	if err != nil {
		a.logger.Error("Failed to get tasks", "error", err)
		return []storage.Task{}
	}

	// Populate FileExists for each task
	for i := range tasks {
		if tasks[i].SavePath != "" {
			if _, err := os.Stat(tasks[i].SavePath); err == nil {
				tasks[i].FileExists = true
			} else {
				tasks[i].FileExists = false
			}
		}
	}
	return tasks
}

// OpenFolder opens the file explorer with the file selected
func (a *App) OpenFolder(id string) {
	// Lookup path from DB
	// Using simple loop for now or Engine.GetTask if exposed.
	// Let's assume we fetch all or ask Engine.

	// Better: Helper on Engine/Storage to get one
	task, err := a.engine.GetTask(id)
	if err != nil {
		a.logger.Error("Task not found for OpenFolder", "id", id, "error", err)
		return
	}

	if task.SavePath == "" {
		return
	}

	// Use OS Utils
	if err := core.OpenFolder(task.SavePath); err != nil {
		a.logger.Error("Failed to open folder", "path", task.SavePath, "error", err)
	}
}

func (a *App) OpenFile(id string) {
	task, err := a.engine.GetTask(id)
	if err != nil {
		a.logger.Error("Task not found for OpenFile", "id", id, "error", err)
		return
	}

	if task.SavePath == "" {
		return
	}

	if err := core.OpenFile(task.SavePath); err != nil {
		a.logger.Error("Failed to open file", "path", task.SavePath, "error", err)
	}
}

func (a *App) UpdateSettings(jsonSettings string) {
	a.logger.Info("UpdateSettings called", "settings", jsonSettings)
	// TODO: Parse and save to DB
}

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

func (a *App) GetLifetimeStats() int64 {
	stats := a.engine.GetStats()
	if stats == nil {
		return 0
	}
	lifetime, _ := stats.GetLifetimeStats()
	return lifetime
}

// CalculateHash computes the hash of a file for checksum verification
// algorithm should be "sha256" or "md5"
func (a *App) CalculateHash(filePath string, algorithm string) (string, error) {
	a.logger.Info("frontend_request", "method", "CalculateHash", "path", filePath, "algorithm", algorithm)
	return core.CalculateHash(filePath, algorithm)
}

// GetAnalytics returns comprehensive analytics data including disk usage
func (a *App) GetAnalytics() core.AnalyticsData {
	stats := a.engine.GetStats()
	if stats == nil {
		return core.AnalyticsData{}
	}
	return stats.GetAnalytics()
}

// CheckForUpdates checks for new releases on GitHub
func (a *App) CheckForUpdates() {
	a.logger.Info("Checking for updates...")
	// TODO: Get owner/repo from config or constants
	owner := "tachyon-org"
	repo := "project-tachyon"
	currentVersion := "v0.1.0"

	rel, err := updater.CheckForUpdates(currentVersion, owner, repo)
	if err != nil {
		a.logger.Error("Update check failed", "error", err)
		return
	}

	if rel != nil {
		a.logger.Info("Update available", "version", rel.TagName)
		runtime.EventsEmit(a.ctx, "update:available", map[string]string{
			"version":  rel.TagName,
			"body":     rel.Body,
			"html_url": rel.HTMLURL,
		})
	} else {
		a.logger.Info("No updates available")
	}
}

// GetDiskUsage returns disk space info for the download drive
func (a *App) GetDiskUsage() core.DiskUsageInfo {
	stats := a.engine.GetStats()
	if stats == nil {
		return core.DiskUsageInfo{}
	}
	return stats.GetDiskUsage()
}
