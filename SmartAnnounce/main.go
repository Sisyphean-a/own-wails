package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app, err := NewApp()
	if err != nil {
		println("Error:", err.Error())
		return
	}

	layout := defaultWindowLayout()

	err = wails.Run(&options.App{
		Title:     "SmartAnnounce",
		Width:     layout.Width,
		Height:    layout.Height,
		MinWidth:  layout.MinWidth,
		MinHeight: layout.MinHeight,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 7, G: 11, B: 20, A: 1},
		OnStartup:        app.startup,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
