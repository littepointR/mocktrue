package app

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"sync"
	"testing"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/littepointR/mocktrue/internal/core/config"
	"github.com/littepointR/mocktrue/internal/core/logging"
	"github.com/littepointR/mocktrue/internal/core/module"
	"github.com/littepointR/mocktrue/internal/core/platform"
)

type shutdownTestModule struct {
	id           string
	stopCount    int
	disposeCount int
}

func (m *shutdownTestModule) ID() string { return m.id }
func (m *shutdownTestModule) Manifest() module.Manifest {
	return module.Manifest{ID: m.id}
}
func (m *shutdownTestModule) Init(context.Context, module.Deps) error { return nil }
func (m *shutdownTestModule) Services() []any                         { return nil }
func (m *shutdownTestModule) ServicesWrapped() []application.Service  { return nil }
func (m *shutdownTestModule) Start(context.Context) error             { return nil }
func (m *shutdownTestModule) Stop(context.Context) error {
	m.stopCount++
	return nil
}
func (m *shutdownTestModule) Dispose() {
	m.disposeCount++
}

func TestShutdownRegistryStopsAndDisposesOnce(t *testing.T) {
	registry := module.NewRegistry()
	mod := &shutdownTestModule{id: "cleanup"}
	if err := registry.Register(mod); err != nil {
		t.Fatalf("Register: %v", err)
	}

	logger := logging.New(slog.NewTextHandler(io.Discard, nil))
	var once sync.Once
	shutdownRegistryOnce(&once, registry, logger)
	shutdownRegistryOnce(&once, registry, logger)

	if mod.stopCount != 1 {
		t.Fatalf("Stop count = %d, want 1", mod.stopCount)
	}
	if mod.disposeCount != 1 {
		t.Fatalf("Dispose count = %d, want 1", mod.disposeCount)
	}
}

func TestBuildRejectsMissingRequiredOptions(t *testing.T) {
	logger := logging.New(slog.NewTextHandler(io.Discard, nil))
	valid := Options{
		Registry:  module.NewRegistry(),
		Logger:    logger,
		Paths:     &platform.Paths{},
		Config:    config.Default(),
		AssetsDir: "frontend/dist",
	}

	for _, tc := range []struct {
		name    string
		options Options
		want    string
	}{
		{name: "registry", options: func() Options { o := valid; o.Registry = nil; return o }(), want: "registry"},
		{name: "logger", options: func() Options { o := valid; o.Logger = nil; return o }(), want: "logger"},
		{name: "paths", options: func() Options { o := valid; o.Paths = nil; return o }(), want: "paths"},
		{name: "config", options: func() Options { o := valid; o.Config = nil; return o }(), want: "config"},
		{name: "assets dir", options: func() Options { o := valid; o.AssetsDir = ""; return o }(), want: "assetsDir"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := Build(tc.options)
			if err == nil {
				t.Fatalf("Build must reject missing %s", tc.name)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Build error = %q, want it to mention %q", err.Error(), tc.want)
			}
		})
	}
}
