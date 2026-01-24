package main

import (
	"embed"
	"project-tachyon/internal/core"
	"project-tachyon/internal/logger"

	"github.com/wailsapp/wails/v2"
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
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
