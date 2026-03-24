package app

import (
	"encoding/json"
	"fmt"
	"os"

	"project-tachyon/internal/filesystem"
	"project-tachyon/internal/storage"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	updateOwner    = "tachyon-org"
	updateRepo     = "project-tachyon"
	currentVersion = "v1.0.0"
)

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
	task, err := a.engine.GetTask(id)
	if err != nil {
		a.logger.Error("Task not found for OpenFolder", "id", id, "error", err)
		return
	}

	if task.SavePath == "" {
		return
	}

	// Use OS Utils
	if err := filesystem.OpenFolder(task.SavePath); err != nil {
		a.logger.Error("Failed to open folder", "path", task.SavePath, "error", err)
	}
}

// OpenFile opens a downloaded file with the default application
func (a *App) OpenFile(id string) {
	task, err := a.engine.GetTask(id)
	if err != nil {
		a.logger.Error("Task not found for OpenFile", "id", id, "error", err)
		return
	}

	if task.SavePath == "" {
		return
	}

	if err := filesystem.OpenFile(task.SavePath); err != nil {
		a.logger.Error("Failed to open file", "path", task.SavePath, "error", err)
	}
}

// UpdateSettings saves user settings from a JSON payload to the database.
func (a *App) UpdateSettings(jsonSettings string) {
	a.logger.Info("UpdateSettings called", "settings", jsonSettings)

	var settings map[string]interface{}
	if err := json.Unmarshal([]byte(jsonSettings), &settings); err != nil {
		a.logger.Error("Failed to parse settings JSON", "error", err)
		return
	}
	for key, val := range settings {
		var strVal string
		switch v := val.(type) {
		case string:
			strVal = v
		case float64:
			strVal = fmt.Sprintf("%v", v)
		case bool:
			if v {
				strVal = "true"
			} else {
				strVal = "false"
			}
		default:
			b, _ := json.Marshal(v)
			strVal = string(b)
		}
		if err := a.engine.GetStorage().SetString(key, strVal); err != nil {
			a.logger.Error("Failed to save setting", "key", key, "error", err)
		}
	}
}

// CheckForUpdates checks for new releases on GitHub
func (a *App) CheckForUpdates() {
	a.logger.Info("Checking for updates...")

	rel, err := checkForUpdates(currentVersion, updateOwner, updateRepo)
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

// UpdateRelease represents a GitHub release for update checking
type UpdateRelease struct {
	TagName string
	Body    string
	HTMLURL string
}

// checkForUpdates is a helper to call the updater package
func checkForUpdates(currentVersion, owner, repo string) (*UpdateRelease, error) {
	// Import updater and call its function
	// This is a thin wrapper to keep import clean
	rel, err := checkUpdaterPackage(currentVersion, owner, repo)
	if err != nil {
		return nil, err
	}
	if rel == nil {
		return nil, nil
	}
	return &UpdateRelease{
		TagName: rel.TagName,
		Body:    rel.Body,
		HTMLURL: rel.HTMLURL,
	}, nil
}

// FactoryReset wipes all data and resets settings
func (a *App) FactoryReset() error {
	a.logger.Info("PERFORMING FACTORY RESET")

	// 1. Reset Database (Tasks, History, Stats)
	if err := a.engine.GetStorage().FactoryReset(); err != nil {
		a.logger.Error("Factory reset failed (DB)", "error", err)
		return err
	}

	// 2. Reset Configuration
	if err := a.cfg.FactoryReset(); err != nil {
		a.logger.Error("Factory reset failed (Config)", "error", err)
		return err
	}

	// 3. Clear engine state (optional, but good practice)
	// For now, relies on frontend reloading or app restart

	a.logger.Info("Factory reset completed successfully")
	return nil
}
