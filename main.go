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
	runtimemetrics "github.com/suyue/mocktrue/internal/core/runtime"
	"github.com/suyue/mocktrue/internal/core/workspace"
	"github.com/suyue/mocktrue/internal/modules/mcpserver"
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
	serialModule := serial.New()
	if err := reg.Register(serialModule); err != nil {
		log.Fatalf("register serial module: %v", err)
	}
	if err := reg.Register(mcpserver.New(serialModule.Service())); err != nil {
		log.Fatalf("register mcp server module: %v", err)
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
		ExtraServices: []application.Service{
			application.NewService(runtimemetrics.NewService()),
			application.NewService(workspace.NewService(paths)),
		},
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
