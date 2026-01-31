package main

import (
	"embed"
	"io"
	"os"
	"project-tachyon/internal/api"
	"project-tachyon/internal/config"
	"project-tachyon/internal/core"
	"project-tachyon/internal/logger"
	"project-tachyon/internal/security"
	"project-tachyon/internal/storage"

	"github.com/getlantern/systray"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/menu/keys"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Parse Flags
	mcpMode := false
	for _, arg := range os.Args {
		if arg == "--mcp" {
			mcpMode = true
			break
		}
	}

	// Initialize Logger
	var logOutput io.Writer = os.Stdout
	if mcpMode {
		logOutput = os.Stderr // Redirect logs to stderr in MCP mode to keep stdout clean for RPC
	}

	log, wailsHandler, err := logger.New(logOutput)
	if err != nil {
		println("Error initializing logger:", err.Error())
		return
	}

	// Initialize Storage
	store, err := storage.NewStorage()
	if err != nil {
		log.Error("Error initializing storage", "error", err)
		println("Error initializing storage:", err.Error())
		return
	}
	defer store.Close()

	// Initialize Core Components
	engine := core.NewEngine(log, store)
	cfg := config.NewConfigManager(store)
	audit := security.NewAuditLogger(log)
	defer audit.Close()

	// Initialize Control Server (background)
	controlServer := api.NewControlServer(engine, cfg, audit)
	controlServer.Start(cfg.GetAIPort())

	// Deprecated API Server (Legacy) - Keeping commented out or removed to favor new ControlServer
	// apiServer := core.NewAPIServer(log, engine, store)
	// apiServer.Start(45000)

	// MCP Mode Execution
	if mcpMode {
		mcpServer := api.NewMCPServer(engine)
		mcpServer.Start() // Blocking
		return
	}

	// GUI Mode (Wails)

	// Create an instance of the app structure, injecting dependencies
	app := NewApp(log, engine, wailsHandler, cfg, audit)

	// Handle standard OS signals (Ctrl+C) for graceful shutdown
	core.WaitForSignals(func() {
		log.Info("OS Signal received, initiating shutdown...")
		app.QuitApp()
	})

	// Parse StartHidden flag
	startHidden := false
	for _, arg := range os.Args {
		if arg == "--minimized" {
			startHidden = true
		}
	}

	// Start System Tray (Run in goroutine for Windows)
	go func() {
		systray.Run(func() {
			systray.SetIcon(appIcon) // AppIcon embedded below
			systray.SetTitle("Tachyon")
			systray.SetTooltip("Project Tachyon")

			mOpen := systray.AddMenuItem("Open Tachyon", "Restore the window")
			systray.AddSeparator()
			// mPause := systray.AddMenuItem("Pause All", "Pause all active downloads")
			// mResume := systray.AddMenuItem("Resume All", "Resume all downloads")
			// systray.AddSeparator()
			mQuit := systray.AddMenuItem("Quit", "Quit the application")

			go func() {
				for {
					select {
					case <-mOpen.ClickedCh:
						app.ShowApp()
					case <-mQuit.ClickedCh:
						app.QuitApp()
					}
				}
			}()
		}, func() {
			// Tray exit cleanup
		})
	}()

	// Create System Tray Menu (Wails App Menu)
	appMenu := menu.NewMenu()
	fileMenu := appMenu.AddSubmenu("File")
	fileMenu.AddText("Open Tachyon", keys.CmdOrCtrl("o"), func(_ *menu.CallbackData) {
		app.ShowApp()
	})
	fileMenu.AddSeparator()
	fileMenu.AddText("Quit", keys.CmdOrCtrl("q"), func(_ *menu.CallbackData) {
		app.QuitApp()
	})

	// Create application with options
	err = wails.Run(&options.App{
		Title:  "project-tachyon",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		OnBeforeClose:    app.beforeClose, // Hook the close event
		StartHidden:      startHidden,     // Handle --minimized
		Menu:             appMenu,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}

//go:embed build/appicon.png
var appIcon []byte
