package app

import (
	"context"
	"io"
	"log/slog"
	"sync"
	"testing"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/littepointR/mocktrue/internal/core/logging"
	"github.com/littepointR/mocktrue/internal/core/module"
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
