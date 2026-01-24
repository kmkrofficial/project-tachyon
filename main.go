package main

import (
	"embed"
	"project-tachyon/internal/core"
	"project-tachyon/internal/logger"
	"project-tachyon/internal/storage"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/menu/keys"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Initialize Logger
	log, err := logger.New()
	if err != nil {
		println("Error initializing logger:", err.Error())
		return
	}

	// Initialize Storage
	store, err := storage.NewStorage()
	if err != nil {
		log.Error("Error initializing storage", "error", err) // Log it but maybe proceed or fatal?
		// For now, let's fatal if storage fails as app depends on it
		println("Error initializing storage:", err.Error())
		return
	}
	defer store.Close()

	// Initialize Engine with Storage
	engine := core.NewEngine(log, store)

	// Create an instance of the app structure, injecting dependencies
	app := NewApp(log, engine)

	// Create System Tray Menu
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
		Menu:             appMenu,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
