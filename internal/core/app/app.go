package app

import (
	"context"
	"embed"
	"io/fs"
	"log/slog"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"

	mterrors "github.com/suyue/mocktrue/internal/core/errors"
	"github.com/suyue/mocktrue/internal/core/config"
	"github.com/suyue/mocktrue/internal/core/eventbus"
	"github.com/suyue/mocktrue/internal/core/logging"
	"github.com/suyue/mocktrue/internal/core/module"
	"github.com/suyue/mocktrue/internal/core/platform"
)

// assetsFS returns the sub-filesystem rooted at dir within the embedded FS.
// The embed directive in main captures "all:frontend/dist"; we sub-root it
// so the asset server serves files at the web root.
func assetsFS(assets embed.FS, dir string) fs.FS {
	sub, err := fs.Sub(assets, dir)
	if err != nil {
		// dir is controlled by the caller and must exist in the embed; a
		// failure here is a programmer error surfaced at runtime.
		panic("app: invalid assets embed dir: " + err.Error())
	}
	return sub
}

// Options bundles the inputs needed to assemble the MockTrue application.
type Options struct {
	Name        string
	Description string
	Assets      embed.FS // embedded frontend/dist, provided by main
	AssetsDir   string   // embed root, e.g. "frontend/dist"
	Logger      *logging.Logger
	Paths       *platform.Paths
	Config      *config.Schema
	Registry    *module.ModuleRegistry
}

// Build constructs the Wails *application.App from a module registry. It
// wires the EventBus bridge, aggregates wrapped services into Options.Services,
// and drives module InitAll/StartAll from the ApplicationStarted event (v3
// has no OnStartup). It does not call app.Run — the caller does.
func Build(o Options) (*application.App, *eventbus.EventBus, error) {
	if o.Registry == nil {
		return nil, nil, mterrors.New(mterrors.CodeInvalid, "registry must not be nil")
	}
	if o.Logger == nil {
		return nil, nil, mterrors.New(mterrors.CodeInvalid, "logger must not be nil")
	}
	if o.Paths == nil {
		return nil, nil, mterrors.New(mterrors.CodeInvalid, "paths must not be nil")
	}
	if o.Config == nil {
		return nil, nil, mterrors.New(mterrors.CodeInvalid, "config must not be nil")
	}
	if o.AssetsDir == "" {
		return nil, nil, mterrors.New(mterrors.CodeInvalid, "assetsDir must not be empty")
	}

	bus := eventbus.New()

	app := application.New(application.Options{
		Name:        o.Name,
		Description: o.Description,
		Services:    o.Registry.AllServicesWrapped(),
		Assets: application.AssetOptions{
			// BundledAssetFileServer injects /wails/runtime.js so the
			// frontend @wailsio/runtime works.
			Handler: application.BundledAssetFileServer(assetsFS(o.Assets, o.AssetsDir)),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	bus.BridgeToFrontend(&appBridge{app: app})

	// Drive module lifecycle from ApplicationStarted (v3 has no OnStartup).
	app.Event.OnApplicationEvent(events.Common.ApplicationStarted, func(*application.ApplicationEvent) {
		deps := module.Deps{
			Bus:    bus,
			Config: o.Config,
			Logger: o.Logger,
			Paths:  o.Paths,
		}
		ctx := context.Background()
		if err := o.Registry.InitAll(ctx, deps); err != nil {
			o.Logger.Error("module init failed", slog.Any("err", err))
			app.Event.Emit("app:error", err.Error())
			return
		}
		if err := o.Registry.StartAll(ctx); err != nil {
			o.Logger.Error("module start failed", slog.Any("err", err))
			app.Event.Emit("app:error", err.Error())
			return
		}
		app.Event.Emit("app:modules", o.Registry.FrontendContributions())
	})

	return app, bus, nil
}

// appBridge adapts *application.App to eventbus.Bridge.
type appBridge struct {
	app *application.App
}

func (b *appBridge) Emit(name string, data ...any) bool {
	return b.app.Event.Emit(name, data...)
}
