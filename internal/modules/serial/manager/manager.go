package manager

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/suyue/mocktrue/internal/core/eventbus"
	"github.com/suyue/mocktrue/internal/core/errors"
	"github.com/suyue/mocktrue/internal/modules/serial/port"
)

// PortManager manages multiple concurrent serial port handles.
// All methods are safe for concurrent use.
type PortManager struct {
	mu       sync.RWMutex
	handles  map[string]*Handle
	bus      *eventbus.EventBus
	nextID   atomic.Int64
}

// NewManager constructs a PortManager that emits events on the given bus.
func NewManager(bus *eventbus.EventBus) *PortManager {
	return &PortManager{
		handles: make(map[string]*Handle),
		bus:     bus,
	}
}

// Open opens a serial port and starts a read loop. Returns the handle ID.
// Duplicate port names are rejected (CodeConflict). Empty port name or
// baud <= 0 is rejected (CodeInvalid).
func (m *PortManager) Open(ctx context.Context, req OpenRequest) (*HandleStatus, error) {
	if req.Config.PortName == "" {
		return nil, errors.New(errors.CodeInvalid, "port name must not be empty")
	}
	if req.Config.BaudRate <= 0 {
		return nil, errors.New(errors.CodeInvalid, "baud rate must be positive")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Check for duplicate
	for _, h := range m.handles {
		if h.config.PortName == req.Config.PortName {
			return nil, errors.New(errors.CodeConflict, fmt.Sprintf("port already open: %s", req.Config.PortName))
		}
	}

	// Open the port
	p, err := port.Open(req.Config)
	if err != nil {
		return nil, errors.Wrap(errors.CodeIO, fmt.Sprintf("open port %s", req.Config.PortName), err)
	}

	id := fmt.Sprintf("port-%d", m.nextID.Add(1))
	h := newHandle(id, req.Config, p, m.bus)
	m.handles[id] = h

	h.start(ctx)
	return h.Status(), nil
}

// Close stops the read loop and closes the port. Idempotent.
func (m *PortManager) Close(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	h, ok := m.handles[id]
	if !ok {
		return errors.New(errors.CodeNotFound, fmt.Sprintf("handle not found: %s", id))
	}
	h.stop()
	delete(m.handles, id)
	return nil
}

// List returns a snapshot of all open handles. Returns empty slice (not nil)
// if no handles are open.
func (m *PortManager) List() []HandleStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]HandleStatus, 0, len(m.handles))
	for _, h := range m.handles {
		out = append(out, *h.Status())
	}
	return out
}
