package main

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"project-tachyon/internal/core"
)

// App struct
type App struct {
	ctx    context.Context
	logger *slog.Logger
	engine *core.TachyonEngine
}

// NewApp creates a new App application struct
func NewApp(logger *slog.Logger, engine *core.TachyonEngine) *App {
	return &App{
		logger: logger,
		engine: engine,
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.engine.SetContext(ctx)
	a.logger.Info("App started")
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
