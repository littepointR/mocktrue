package serial

import (
	"context"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/suyue/mocktrue/internal/core/eventbus"
	"github.com/suyue/mocktrue/internal/core/logging"
	"github.com/suyue/mocktrue/internal/core/module"
	"github.com/suyue/mocktrue/internal/modules/serial/buffer"
	"github.com/suyue/mocktrue/internal/modules/serial/manager"
)

// Module implements module.Module for the serial debugging feature.
type Module struct {
	svc *Service
	bus *eventbus.EventBus
	log *logging.Logger
}

// New constructs a serial module with a nil-wired service. The service is
// populated during Init when the event bus becomes available.
func New() *Module {
	return &Module{svc: &Service{}}
}

// ID returns the stable module identifier.
func (m *Module) ID() string { return "serial" }

// Manifest returns metadata including the frontend contribution.
func (m *Module) Manifest() module.Manifest {
	return module.Manifest{
		ID:      "serial",
		Version: "0.1.0",
		Frontend: module.FrontendContribution{
			ActivityIcon:  "serial",
			ActivityTitle: "串口调试",
			Views: []module.FrontendView{
				{ID: "serial.connect", Title: "连接", Component: "serial/ConnectView"},
			},
		},
	}
}

// Init injects dependencies and wires the Service with the event bus.
func (m *Module) Init(ctx context.Context, deps module.Deps) error {
	_ = ctx
	m.bus = deps.Bus
	m.log = deps.Logger.Named("serial")
	// Wire the service with the event bus if it was created without one
	if m.svc.bus == nil {
		m.svc.bus = deps.Bus
		m.svc.manager = manager.NewManager(deps.Bus)
		m.svc.buffers = make(map[string]*buffer.RingBuffer)
	}
	m.log.Info("serial module init")
	return nil
}

// Services returns the raw service instance.
func (m *Module) Services() []any {
	return []any{m.svc}
}

// ServicesWrapped returns the service wrapped for Wails binding.
func (m *Module) ServicesWrapped() []application.Service {
	return []application.Service{application.NewService(m.svc)}
}

// Start is a no-op for the serial module (port goroutines start on OpenPort).
func (m *Module) Start(ctx context.Context) error {
	_ = ctx
	if m.log != nil {
		m.log.Info("serial module start")
	}
	return nil
}

// Stop is a no-op (ports are closed individually via ClosePort).
func (m *Module) Stop(ctx context.Context) error {
	_ = ctx
	if m.log != nil {
		m.log.Info("serial module stop")
	}
	return nil
}

// Dispose releases resources. Idempotent and safe before Init.
func (m *Module) Dispose() {}
