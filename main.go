package main

import (
	"Aether/pkg/storage"
	"Aether/pkg/updater"
	"embed"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	if len(os.Args) == 2 && os.Args[1] == "--initialize" {
		store := storage.NewStore()
		for _, dir := range []string{"instances", "libraries", "assets", "versions", "runtimes"} {
			if info, err := os.Stat(filepath.Join(store.LauncherDir, dir)); err != nil || !info.IsDir() {
				os.Exit(1)
			}
		}
		if _, err := os.Stat(filepath.Join(store.LauncherDir, "camlauncher_data.json")); err != nil {
			os.Exit(1)
		}
		return
	}
	// Clean up any .old binary from previous auto-update
	updater.CleanupOldVersions()

	// Create an instance of the app structure
	app := NewApp()

	// Create application with options
	err := wails.Run(&options.App{
		Title:     "CamLauncher",
		Width:     1200,
		Height:    780,
		MinWidth:  1160,
		MinHeight: 740,
		Frameless: true,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 11, G: 15, B: 23, A: 255},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		Bind: []interface{}{
			app,
		},
		Windows: &windows.Options{
			WebviewIsTransparent:              true,
			WindowIsTranslucent:               true,
			BackdropType:                      windows.Mica,
			DisableWindowIcon:                 false,
			DisableFramelessWindowDecorations: true,
			Theme:                             windows.Dark,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
