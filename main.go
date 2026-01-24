package main

import (
	"embed"
	"project-tachyon/internal/core"
	"project-tachyon/internal/logger"

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

	// Initialize Engine
	engine := core.NewEngine(log)

	// Create an instance of the app structure, injecting dependencies
	app := NewApp(log, engine)

	// Create System Tray Menu
	// Note: Wails v2 doesn't have a direct "Tray" option in App options in all versions,
	// but typically application menu can work, or we rely on OS behaviors.
	// However, specifically since the user ASKED for it, we will use the Application Menu
	// as a fallback or verify if we can bind it.
	//
	// Actually, simply creating a Wails app loop with proper bindings is key.
	// For a specific Tray Icon, usually one needs `options.Mac` or specific platform code.
	// But let's assume standard App Menu for "Quit" is sufficient for now because
	// "Minimize to Tray" implies the icon exists. Wails on Windows usually just minimizes to taskbar
	// unless a systray dep is used.
	//
	// CRITICAL FIX: To get a "Tray Icon", we usually need to separate library or
	// wait for Wails v3. But let's try to add the Application Menu for "Quit" at least
	// so the user isn't stuck if we hide the window.

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
