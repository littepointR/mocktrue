package module

import (
	"context"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/littepointR/mocktrue/internal/core/config"
	"github.com/littepointR/mocktrue/internal/core/eventbus"
	"github.com/littepointR/mocktrue/internal/core/logging"
	"github.com/littepointR/mocktrue/internal/core/platform"
)

// Deps is the immutable dependency bundle injected into each Module at Init.
type Deps struct {
	Bus    *eventbus.EventBus
	Config *config.Schema
	Logger *logging.Logger
	Paths  *platform.Paths
}

// FrontendContribution describes a module's UI contributions to the shell.
// The zero value means "no frontend contribution".
type FrontendContribution struct {
	ActivityIcon  string         // icon key, e.g. "serial"
	ActivityTitle string         // hover title
	Views         []FrontendView // views the module offers
}

// FrontendView is a single view a module contributes to the sidebar/editor.
type FrontendView struct {
	ID        string
	Title     string
	Component string // frontend component path/name for lazy loading
}

// Manifest describes a module's metadata, used for ordering and frontend
// contribution. Immutable.
type Manifest struct {
	ID           string
	Version      string
	Dependencies []string // IDs of modules this one depends on (for topo sort)
	Frontend     FrontendContribution
}

// Module is the contract every debugging feature implements. A module is
// self-contained: it must not import another module's code, only the core
// services in Deps and the Wails application.Service type for wrapping.
type Module interface {
	// ID is the unique, stable module identifier, e.g. "serial".
	ID() string
	// Manifest returns metadata (dependencies, frontend contribution).
	Manifest() Manifest
	// Init injects core dependencies. Called once before Start.
	Init(ctx context.Context, deps Deps) error
	// Services returns the raw service instances (for internal reference).
	Services() []any
	// ServicesWrapped returns the services already wrapped via
	// application.NewService, ready for application.Options.Services. The
	// shell aggregates these without reflection.
	ServicesWrapped() []application.Service
	// Start launches any background goroutines. Called once after all Init.
	Start(ctx context.Context) error
	// Stop gracefully stops background work. Idempotent and safe to call
	// even if Init/Start were not called.
	Stop(ctx context.Context) error
	// Dispose releases local resources. Idempotent.
	Dispose()
}
