package mcpserver

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/suyue/mocktrue/internal/core/config"
	"github.com/suyue/mocktrue/internal/core/eventbus"
	"github.com/suyue/mocktrue/internal/core/logging"
	"github.com/suyue/mocktrue/internal/core/module"
)

// Status is emitted to the frontend when the MCP server state changes.
type Status struct {
	Enabled bool
	Running bool
	Address string
	Path    string
	Error   string
}

// Module hosts the local MCP HTTP server.
type Module struct {
	serial SerialRuntime
	cfg    config.MCPConfig
	bus    *eventbus.EventBus
	log    *logging.Logger
	server *http.Server
	status Status
}

// New constructs an MCP server module that exposes the shared serial runtime.
func New(serialService SerialRuntime) *Module {
	return &Module{serial: serialService}
}

func (m *Module) ID() string { return "mcpserver" }

func (m *Module) Manifest() module.Manifest {
	return module.Manifest{
		ID:           "mcpserver",
		Version:      "0.1.0",
		Dependencies: []string{"serial"},
	}
}

func (m *Module) Init(ctx context.Context, deps module.Deps) error {
	_ = ctx
	m.bus = deps.Bus
	m.log = deps.Logger.Named("mcpserver")
	m.cfg = deps.Config.MCP
	m.cfg.Path = normalizedPath(m.cfg.Path)
	m.status = Status{
		Enabled: m.cfg.Enabled,
		Path:    m.cfg.Path,
	}
	if !m.cfg.Enabled {
		m.publishStatus()
	}
	return nil
}

func (m *Module) Services() []any { return nil }

func (m *Module) ServicesWrapped() []application.Service { return nil }

func (m *Module) Start(ctx context.Context) error {
	_ = ctx
	if !m.cfg.Enabled {
		return nil
	}
	if m.serial == nil {
		m.setError("serial service is not available")
		return nil
	}

	address, err := listenAddress(m.cfg)
	if err != nil {
		m.setError(err.Error())
		return nil
	}

	mcpServer := newMCPServer(m.serial)
	mux := http.NewServeMux()
	mux.Handle(m.cfg.Path, originGuard(mcpSessionHTTPHandler(mcpServer), m.cfg))
	httpServer := &http.Server{Addr: address, Handler: mux}
	listener, err := net.Listen("tcp", address)
	if err != nil {
		m.setError(err.Error())
		return nil
	}

	m.server = httpServer
	m.status = Status{
		Enabled: true,
		Running: true,
		Address: address,
		Path:    m.cfg.Path,
	}
	m.publishStatus()

	go func() {
		if err := httpServer.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			if m.log != nil {
				m.log.Error("mcp server failed", slog.Any("err", err))
			}
			m.setError(err.Error())
		}
	}()
	return nil
}

func (m *Module) Stop(ctx context.Context) error {
	if m.server == nil {
		return nil
	}
	err := shutdownHTTPServer(ctx, m.server)
	m.server = nil
	m.status.Running = false
	m.publishStatus()
	return err
}

func (m *Module) Dispose() {
	if m.server != nil {
		_ = m.server.Close()
		m.server = nil
	}
}

func (m *Module) setError(message string) {
	m.status = Status{
		Enabled: m.cfg.Enabled,
		Running: false,
		Path:    m.cfg.Path,
		Error:   message,
	}
	m.publishStatus()
	if m.log != nil {
		m.log.Error("mcp server unavailable", slog.String("error", message))
	}
}

func (m *Module) publishStatus() {
	if m.bus != nil {
		m.bus.Publish("mcp:status", m.status)
	}
}
