// Package app provides the Wails bridge between the frontend and backend.
// It is split into multiple files by domain for maintainability.
package app

import (
	"context"
	"log/slog"

	"project-tachyon/internal/config"
	"project-tachyon/internal/engine"
	"project-tachyon/internal/logger"
	"project-tachyon/internal/security"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct is the main Wails application binding.
// It bridges frontend calls to the TachyonEngine and other services.
type App struct {
	ctx          context.Context
	logger       *slog.Logger
	wailsHandler *logger.WailsHandler
	engine       *engine.TachyonEngine
	cfg          *config.ConfigManager
	audit        *security.AuditLogger
	isQuitting   bool
}

// NewApp creates a new App application struct with all dependencies injected.
func NewApp(
	logger *slog.Logger,
	engine *engine.TachyonEngine,
	wailsHandler *logger.WailsHandler,
	cfg *config.ConfigManager,
	audit *security.AuditLogger,
) *App {
	return &App{
		logger:       logger,
		engine:       engine,
		wailsHandler: wailsHandler,
		cfg:          cfg,
		audit:        audit,
		isQuitting:   false,
	}
}

// Startup is called when the app starts. The context is saved
// so we can call the runtime methods.
func (a *App) Startup(ctx context.Context) {
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

// BeforeClose is called when the application is about to close.
// We return true to prevent closing (and hide instead), unless isQuitting is true.
func (a *App) BeforeClose(ctx context.Context) (prevent bool) {
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

// GetContext returns the Wails context for emitting events from other bridge files
func (a *App) GetContext() context.Context {
	return a.ctx
}

