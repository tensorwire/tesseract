package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := NewApp()

	err := wails.Run(&options.App{
		Title:            "Tesseract",
		Width:            900,
		Height:           700,
		MinWidth:         600,
		MinHeight:        400,
		AssetServer:      &assetserver.Options{Assets: assets},
		BackgroundColour: &options.RGBA{R: 10, G: 10, B: 15, A: 255},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		Frameless: true,
		Mac: &mac.Options{
			About: &mac.AboutInfo{
				Title:   "Tesseract",
				Message: "GPU-accelerated LLM desktop app\nPowered by mongoose",
			},
		},
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
