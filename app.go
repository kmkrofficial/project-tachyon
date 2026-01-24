package main

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"project-tachyon/internal/core"
	"project-tachyon/internal/storage"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx        context.Context
	logger     *slog.Logger
	engine     *core.TachyonEngine
	isQuitting bool
}

// NewApp creates a new App application struct
func NewApp(logger *slog.Logger, engine *core.TachyonEngine) *App {
	return &App{
		logger:     logger,
		engine:     engine,
		isQuitting: false,
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.engine.SetContext(ctx)
	a.logger.Info("App started")
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
	runtime.Quit(a.ctx)
}

// ShowApp is called from the Tray menu to restore the window
func (a *App) ShowApp() {
	runtime.WindowShow(a.ctx)
	runtime.WindowSetAlwaysOnTop(a.ctx, true) // Bring to front
	runtime.WindowSetAlwaysOnTop(a.ctx, false)
}

// AddDownload is exposed to the Frontend
func (a *App) AddDownload(url string) string {
	a.logger.Info("frontend_request", "method", "AddDownload", "url", url)

	// Use User Home / Downloads as default
	homeDir, err := os.UserHomeDir()
	defaultPath := filepath.Join(homeDir, "Downloads")
	if err != nil {
		a.logger.Error("Failed to get home dir", "error", err)
		return "ERROR: " + err.Error()
	}

	id, err := a.engine.StartDownload(url, defaultPath)
	if err != nil {
		a.logger.Error("Failed to start download", "error", err)
		return "ERROR: " + err.Error()
	}

	return id
}

// GetTasks returns all saved tasks from the database
func (a *App) GetTasks() []storage.Task {
	tasks, err := a.engine.GetHistory()
	if err != nil {
		a.logger.Error("Failed to get tasks", "error", err)
		return []storage.Task{}
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

	if task.Path == "" {
		return
	}

	// Use OS Utils
	if err := core.OpenFolder(task.Path); err != nil {
		a.logger.Error("Failed to open folder", "path", task.Path, "error", err)
	}
}

func (a *App) OpenFile(id string) {
	task, err := a.engine.GetTask(id)
	if err != nil {
		a.logger.Error("Task not found for OpenFile", "id", id, "error", err)
		return
	}

	if task.Path == "" {
		return
	}

	if err := core.OpenFile(task.Path); err != nil {
		a.logger.Error("Failed to open file", "path", task.Path, "error", err)
	}
}

func (a *App) UpdateSettings(jsonSettings string) {
	a.logger.Info("UpdateSettings called", "settings", jsonSettings)
	// TODO: Parse and save to DB
}

func (a *App) RunNetworkSpeedTest() *core.SpeedTestResult {
	res, err := core.RunSpeedTest()
	if err != nil {
		a.logger.Error("Speed test failed", "error", err)
		return nil
	}
	return res
}

func (a *App) GetLifetimeStats() int64 {
	// Assuming Engine has StatsManager or App holds it
	// I need to add StatsManager to Engine or App.
	// Let's add it to Engine for encapsulation.
	return 0 // Placeholder until Engine update
}
