package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"

	coreapp "github.com/suyue/mocktrue/internal/core/app"
	"github.com/suyue/mocktrue/internal/core/config"
	"github.com/suyue/mocktrue/internal/core/logging"
	"github.com/suyue/mocktrue/internal/core/module"
	"github.com/suyue/mocktrue/internal/core/platform"
	"github.com/suyue/mocktrue/internal/modules/serial"
)

// Wails uses Go's `embed` package to embed the frontend files into the
// binary. Any files under frontend/dist are embedded and served to the
// webview. See https://pkg.go.dev/embed for more information.

//go:embed all:frontend/dist
var assets embed.FS

const assetsDir = "frontend/dist"

func main() {
	logger := logging.Default()

	paths, err := platform.ResolvePaths("mocktrue")
	if err != nil {
		log.Fatalf("resolve paths: %v", err)
	}

	cfg, err := config.Load(paths.ConfigDir + "/config.toml")
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	reg := module.NewRegistry()
	if err := reg.Register(serial.New()); err != nil {
		log.Fatalf("register serial module: %v", err)
	}

	app, _, err := coreapp.Build(coreapp.Options{
		Name:        "MockTrue",
		Description: "跨平台高性能嵌入式调试工具",
		Assets:      assets,
		AssetsDir:   assetsDir,
		Logger:      logger,
		Paths:       paths,
		Config:      cfg,
		Registry:    reg,
	})
	if err != nil {
		log.Fatalf("build app: %v", err)
	}

	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title: "MockTrue",
		URL:   "/",
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 50,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInset,
		},
		BackgroundColour: application.NewRGB(30, 30, 30),
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
