package app

import (
	"project-tachyon/internal/core"
	"project-tachyon/internal/storage"
)

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
}

// SetMaxConcurrentDownloads sets the maximum number of concurrent downloads
func (a *App) SetMaxConcurrentDownloads(n int) {
	a.logger.Info("frontend_request", "method", "SetMaxConcurrentDownloads", "n", n)
	a.engine.SetMaxConcurrent(n)
}

// SetHostLimit sets the per-host connection limit
func (a *App) SetHostLimit(domain string, limit int) {
	a.logger.Info("frontend_request", "method", "SetHostLimit", "domain", domain, "limit", limit)
	a.engine.SetHostLimit(domain, limit)
}

// GetHostLimit returns the per-host connection limit
func (a *App) GetHostLimit(domain string) int {
	return a.engine.GetHostLimit(domain)
}

// ProbeURL checks the URL metadata before downloading
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

// CollisionResult provides file collision check results
type CollisionResult struct {
	Exists bool   `json:"exists"`
	Path   string `json:"path"`
}

// CheckCollision checks file system for existing files
func (a *App) CheckCollision(filename string) CollisionResult {
	exists, path, err := a.engine.CheckCollision(filename)
	if err != nil {
		return CollisionResult{Exists: false, Path: ""}
	}
	return CollisionResult{Exists: exists, Path: path}
}
