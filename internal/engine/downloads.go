package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"project-tachyon/internal/filesystem"
	"project-tachyon/internal/storage"

	"github.com/google/uuid"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// GetHistory returns all download tasks
func (e *TachyonEngine) GetHistory() ([]storage.Task, error) {
	return e.storage.GetAllTasks()
}

// GetTask returns a specific download task
func (e *TachyonEngine) GetTask(id string) (storage.Task, error) {
	return e.storage.GetTask(id)
}

// GetQueuedDownloads returns all downloads in the queue for UI display
func (e *TachyonEngine) GetQueuedDownloads() []*storage.DownloadTask {
	return e.queue.GetAll()
}

// StartDownload initiates a new download
func (e *TachyonEngine) StartDownload(urlStr string, destPath string, customFilename string, options map[string]string) (string, error) {
	if urlStr == "" {
		return "", fmt.Errorf("empty URL")
	}

	downloadID := uuid.New().String()

	cookies := options["cookies"]
	cookiesJSON := options["cookies_json"]

	if cookies != "" && cookiesJSON == "" {
		// Just store as is for now
	}

	// Guess filename for categorization (preliminary)
	var guessedFilename string
	if customFilename != "" {
		guessedFilename = customFilename
	} else {
		guessedFilename = filepath.Base(urlStr)
		if guessedFilename == "" || guessedFilename == "." {
			guessedFilename = "unknown"
		}
	}

	organizedPath, _ := filesystem.GetOrganizedPath(destPath, guessedFilename)
	// Find available path with _2, _3 suffix if collision
	finalPath := filesystem.FindAvailablePath(organizedPath)
	category := filesystem.GetCategory(guessedFilename)

	// Handle Scheduled Start
	var startTime string
	initialStatus := "pending"

	if st, ok := options["start_time"]; ok && st != "" {
		if _, err := time.Parse(time.RFC3339, st); err == nil {
			startTime = st
			initialStatus = "scheduled"
		} else {
			e.logger.Warn("Invalid start_time format", "time", st)
		}
	}

	task := storage.DownloadTask{
		ID:         downloadID,
		URL:        urlStr,
		Filename:   filepath.Base(finalPath),
		SavePath:   finalPath,
		Status:     initialStatus,
		Category:   category,
		Priority:   2, // Default Normal
		QueueOrder: e.queue.GetNextOrder(),
		CreatedAt:  time.Now().Format(time.RFC3339),
		UpdatedAt:  time.Now().Format(time.RFC3339),
		Headers:    options["headers_json"],
		Cookies:    options["cookies_json"],
		StartTime:  startTime,
	}

	if err := e.storage.SaveTask(task); err != nil {
		e.logger.Error("Failed to save initial task", "error", err)
	}

	e.queue.Push(&task)

	if e.ctx != nil {
		runtime.EventsEmit(e.ctx, "download:progress", map[string]interface{}{
			"id":          downloadID,
			"progress":    0,
			"status":      "pending",
			"filename":    task.Filename,
			"total":       task.TotalSize,
			"path":        task.SavePath,
			"queue_order": task.QueueOrder,
		})
	}

	return downloadID, nil
}

// PauseDownload cancels an active download
func (e *TachyonEngine) PauseDownload(id string) error {
	val, ok := e.activeDownloads.Load(id)
	if !ok {
		// Not active, update DB if pending
		task, err := e.storage.GetTask(id)
		if err == nil && (task.Status == "pending" || task.Status == "downloading") {
			task.Status = "paused"
			e.storage.SaveTask(task)
			// Emit paused event to frontend
			if e.ctx != nil {
				runtime.EventsEmit(e.ctx, "download:paused", map[string]interface{}{
					"id": id,
				})
			}
		}
		return nil
	}

	info, ok := val.(*activeDownloadInfo)
	if !ok {
		return fmt.Errorf("invalid download info")
	}

	if info.Cancel != nil {
		info.Cancel()
	}
	return nil
}

// ResumeDownload re-queues a paused or stopped download to continue
func (e *TachyonEngine) ResumeDownload(id string) error {
	task, err := e.storage.GetTask(id)
	if err != nil {
		return fmt.Errorf("task not found: %w", err)
	}

	// Only resume if it's in a resumable state (including stuck "downloading")
	resumableStates := map[string]bool{"paused": true, "stopped": true, "error": true, "downloading": true, "pending": true}
	if !resumableStates[task.Status] {
		return fmt.Errorf("cannot resume download in status: %s", task.Status)
	}

	// Check if file still exists on disk - if not, reset progress
	if task.SavePath != "" {
		if _, err := os.Stat(task.SavePath); os.IsNotExist(err) {
			e.logger.Info("File missing on disk, restarting download from scratch", "id", task.ID, "path", task.SavePath)
			task.Downloaded = 0
			task.Progress = 0
			task.MetaJSON = "" // Clear chunk metadata
		}
	}

	// Update status to pending and re-queue
	task.Status = "pending"
	if err := e.storage.SaveTask(task); err != nil {
		return err
	}

	// Re-queue for processing
	e.queue.Push(&storage.DownloadTask{
		ID:         task.ID,
		URL:        task.URL,
		Filename:   task.Filename,
		SavePath:   task.SavePath,
		Status:     "pending",
		Priority:   task.Priority,
		TotalSize:  task.TotalSize,
		Downloaded: task.Downloaded,
		MetaJSON:   task.MetaJSON,
		Category:   task.Category,
		QueueOrder: task.QueueOrder,
		CreatedAt:  task.CreatedAt,
		UpdatedAt:  time.Now().Format(time.RFC3339),
	})

	// Emit event to update UI
	if e.ctx != nil {
		runtime.EventsEmit(e.ctx, "download:progress", map[string]interface{}{
			"id":          id,
			"status":      "pending",
			"filename":    task.Filename,
			"queue_order": task.QueueOrder,
		})
	}

	return nil
}

// StopDownload cancels a download and marks it as stopped (cannot auto-resume)
func (e *TachyonEngine) StopDownload(id string) error {
	// First pause/cancel any active download
	e.PauseDownload(id)

	// Then update status to stopped
	task, err := e.storage.GetTask(id)
	if err != nil {
		return err
	}

	task.Status = "stopped"
	e.storage.SaveTask(task)

	// Emit event
	if e.ctx != nil {
		runtime.EventsEmit(e.ctx, "download:stopped", map[string]interface{}{
			"id": id,
		})
	}

	return nil
}

// PauseAllDownloads pauses all active or pending downloads
func (e *TachyonEngine) PauseAllDownloads() {
	active := make([]string, 0)
	e.activeDownloads.Range(func(key, value interface{}) bool {
		active = append(active, key.(string))
		return true
	})

	for _, id := range active {
		e.PauseDownload(id)
	}

	// Also mark any pending tasks as paused in DB
	tasks, _ := e.storage.GetAllTasks()
	for _, task := range tasks {
		if task.Status == "pending" {
			task.Status = "paused"
			e.storage.SaveTask(task)
		}
	}

	if e.ctx != nil {
		runtime.EventsEmit(e.ctx, "download:paused_all", nil)
	}
}

// ResumeAllDownloads resumes all paused downloads
func (e *TachyonEngine) ResumeAllDownloads() {
	tasks, err := e.storage.GetAllTasks()
	if err != nil {
		e.logger.Error("Failed to get tasks for ResumeAll", "error", err)
		return
	}

	for _, task := range tasks {
		if task.Status == "paused" || task.Status == "stopped" || task.Status == "error" {
			e.ResumeDownload(task.ID)
		}
	}

	if e.ctx != nil {
		runtime.EventsEmit(e.ctx, "download:resumed_all", nil)
	}
}

// UpdateDownloadURL updates the URL for a task that requires authentication refresh
// This is used when a download link has expired (HTTP 403) and needs a new URL
func (e *TachyonEngine) UpdateDownloadURL(taskID, newURL string) error {
	task, err := e.storage.GetTask(taskID)
	if err != nil {
		return fmt.Errorf("task not found: %w", err)
	}

	// Only allow URL update for tasks in needs_auth status
	if task.Status != StatusNeedsAuth && task.Status != "paused" && task.Status != "error" {
		return fmt.Errorf("task is not in a state that allows URL refresh (status: %s)", task.Status)
	}

	oldURL := task.URL
	task.URL = newURL
	task.Status = "paused" // Reset to paused so it can be resumed

	if err := e.storage.SaveTask(task); err != nil {
		return fmt.Errorf("failed to save task: %w", err)
	}

	e.logger.Info("Download URL updated", "id", taskID, "oldURL", oldURL, "newURL", newURL)

	if e.ctx != nil {
		runtime.EventsEmit(e.ctx, "download:url_updated", map[string]interface{}{
			"id":      taskID,
			"new_url": newURL,
		})
	}

	return nil
}

// DeleteDownload removes the task and optionally the file
func (e *TachyonEngine) DeleteDownload(id string, deleteFile bool) error {
	e.PauseDownload(id)

	task, err := e.storage.GetTask(id)
	if err != nil {
		return err
	}

	var fileDeleteErr error
	if deleteFile && task.SavePath != "" {
		// Remove file, ignore not found error
		if err := os.Remove(task.SavePath); err != nil && !os.IsNotExist(err) {
			e.logger.Warn("Failed to delete file", "path", task.SavePath, "error", err)
			fileDeleteErr = err
		}
	}

	// Always delete from storage even if file delete failed
	if err := e.storage.DeleteTask(id); err != nil {
		return err
	}

	// Also remove from queue if present
	e.queue.Remove(id)

	// Emit deleted event for instant UI feedback
	if e.ctx != nil {
		runtime.EventsEmit(e.ctx, "download:deleted", map[string]interface{}{
			"id": id,
		})
	}

	if fileDeleteErr != nil {
		return fmt.Errorf("Record deleted but file could not be removed (locked or in use)")
	}
	return nil
}

// CheckHistory checks if the URL has been downloaded before
func (e *TachyonEngine) CheckHistory(urlStr string) (bool, error) {
	task, err := e.storage.GetTaskByURL(urlStr)
	if err != nil {
		return false, nil
	}
	if task.Status == "completed" {
		return true, nil
	}
	return false, nil
}

// CheckCollision checks if the file already exists
func (e *TachyonEngine) CheckCollision(filename string) (bool, string, error) {
	if filename == "" {
		return false, "", fmt.Errorf("filename is empty")
	}
	defaultPath, err := filesystem.GetDefaultDownloadPath()
	if err != nil {
		return false, "", err
	}
	finalPath, _ := filesystem.GetOrganizedPath(defaultPath, filename)
	if _, err := os.Stat(finalPath); err == nil {
		return true, finalPath, nil
	}
	return false, finalPath, nil
}

// ReorderDownload moves a download in the queue
// direction: "first", "prev", "next", "last"
func (e *TachyonEngine) ReorderDownload(id string, direction string) error {
	var success bool
	switch direction {
	case "first":
		success = e.queue.MoveToFirst(id)
	case "prev":
		success = e.queue.MoveToPrev(id)
	case "next":
		success = e.queue.MoveToNext(id)
	case "last":
		success = e.queue.MoveToLast(id)
	default:
		return fmt.Errorf("invalid direction: %s", direction)
	}

	if !success {
		return fmt.Errorf("could not reorder download %s", id)
	}

	// Update QueueOrder in database for all queued items
	items := e.queue.GetAll()
	for _, item := range items {
		e.storage.SaveTask(*item)
	}

	// Emit event notification
	if e.ctx != nil {
		runtime.EventsEmit(e.ctx, "queue:reordered", nil)
	}

	return nil
}

// SetPriority updates the priority of a download (1=Low, 2=Normal, 3=High)
func (e *TachyonEngine) SetPriority(id string, priority int) error {
	e.logger.Info("Setting priority", "id", id, "priority", priority)
	task, err := e.storage.GetTask(id)
	if err != nil {
		return err
	}

	task.Priority = priority
	if err := e.storage.SaveTask(task); err != nil {
		return err
	}

	// Update Bandwidth Manager
	e.bandwidthManager.SetTaskPriority(id, priority)

	// Emit event
	if e.ctx != nil {
		runtime.EventsEmit(e.ctx, "download:updated", map[string]interface{}{
			"id":       id,
			"priority": priority,
		})
	}
	return nil
}
