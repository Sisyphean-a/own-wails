package main

import (
	"crypto/sha256"
	"embed"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/windows/icon.ico
var trayIcon []byte

func main() {
	configDir, err := os.UserConfigDir()
	if err != nil {
		log.Fatal(err)
	}
	data := flag.String("data-dir", filepath.Join(configDir, "LanSync"), "应用数据目录")
	listen := flag.String("listen", "", "监听地址，默认自动选择局域网端口")
	noDiscovery := flag.Bool("no-discovery", false, "禁用局域网广播与扫描")
	flag.Parse()
	*data, err = filepath.Abs(*data)
	if err != nil {
		log.Fatal(err)
	}
	instanceKey := fmt.Sprintf("LanSync-%x", sha256.Sum256([]byte(strings.ToLower(filepath.Clean(*data)))))
	app := NewApp(*data, *listen, !*noDiscovery)
	if err = wails.Run(&options.App{
		Title: "局域同步", Width: 1120, Height: 780, MinWidth: 720, MinHeight: 540,
		AssetServer:      &assetserver.Options{Assets: assets},
		Windows:          &windows.Options{WebviewUserDataPath: filepath.Join(*data, "webview")},
		BackgroundColour: &options.RGBA{R: 245, G: 246, B: 248, A: 255},
		OnStartup:        app.startup, OnBeforeClose: app.beforeClose, OnShutdown: app.shutdown,
		Bind:               []interface{}{app},
		SingleInstanceLock: &options.SingleInstanceLock{UniqueId: instanceKey, OnSecondInstanceLaunch: app.secondInstance},
	}); err != nil {
		if app.engine != nil {
			app.engine.Close()
		}
		log.Fatal(err)
	}
}
