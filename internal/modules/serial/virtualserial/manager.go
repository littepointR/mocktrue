package virtualserial

import (
	"context"
	"sync"

	"github.com/littepointR/portweave/internal/core/errors"
)

// Manager manages multiple virtual serial pairs and bridges.
type Manager struct {
	pairs   map[string]*VirtualPair // key: pairID
	bridges map[string]*Bridge      // key: bridgeID
	backend Backend
	mu      sync.RWMutex
}

// ManagerOption customizes a virtual serial manager.
type ManagerOption func(*Manager)

// WithBackend injects the OS virtual serial backend. A nil backend is ignored.
func WithBackend(backend Backend) ManagerOption {
	return func(m *Manager) {
		if backend != nil {
			m.backend = backend
		}
	}
}

// NewManager creates a virtual serial manager.
func NewManager(options ...ManagerOption) *Manager {
	m := &Manager{
		pairs:   make(map[string]*VirtualPair),
		bridges: make(map[string]*Bridge),
	}
	for _, option := range options {
		option(m)
	}
	if m.backend == nil {
		m.backend = DefaultBackend()
	}
	return m
}

// BackendStatus returns the active virtual serial backend status.
func (m *Manager) BackendStatus(ctx context.Context) BackendStatus {
	m.mu.RLock()
	backend := m.backend
	m.mu.RUnlock()
	if backend == nil {
		backend = newUnsupportedBackend(
			"unsupported",
			"virtual serial backend is not configured",
			"missing backend",
		)
	}
	return backend.Status(ctx)
}

// CreatePair creates a new virtual serial pair with custom names.
func (m *Manager) CreatePair(ctx context.Context, pairID, port1Name, port2Name string) (*VirtualPair, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.pairs[pairID]; exists {
		return nil, errors.New(errors.CodeConflict, "pair ID already exists")
	}
	if m.backend == nil {
		m.backend = DefaultBackend()
	}

	pair, err := m.backend.CreatePair(ctx, pairID, port1Name, port2Name)
	if err != nil {
		return nil, err
	}
	if pair == nil {
		return nil, errors.New(errors.CodeInternal, "virtual serial backend returned nil pair")
	}
	m.pairs[pairID] = pair
	return pair, nil
}

// DeletePair removes a virtual serial pair.
func (m *Manager) DeletePair(pairID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	pair, exists := m.pairs[pairID]
	if !exists {
		return errors.New(errors.CodeNotFound, "pair not found")
	}

	if err := pair.Stop(); err != nil {
		return err
	}
	delete(m.pairs, pairID)
	return nil
}

// CreatePort creates a user-facing virtual serial port. A hidden peer is
// created internally because PTYs are backed by pairs on Unix-like systems.
func (m *Manager) CreatePort(ctx context.Context, portID, portName string) (*VirtualPair, error) {
	if portName == "" {
		return nil, errors.New(errors.CodeInvalid, "port name must not be empty")
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.pairs[portID]; exists {
		return nil, errors.New(errors.CodeConflict, "pair ID already exists")
	}
	if m.backend == nil {
		m.backend = DefaultBackend()
	}
	pair, err := m.backend.CreatePort(ctx, portID, portName)
	if err != nil {
		return nil, err
	}
	if pair == nil {
		return nil, errors.New(errors.CodeInternal, "virtual serial backend returned nil pair")
	}
	pair.port2Hidden = true
	m.pairs[portID] = pair
	return pair, nil
}

// ListPairs returns all virtual serial pairs.
func (m *Manager) ListPairs() []*VirtualPair {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*VirtualPair, 0, len(m.pairs))
	for _, pair := range m.pairs {
		result = append(result, pair)
	}
	return result
}

// CreateBridge creates a bridge between two serial ports.
func (m *Manager) CreateBridge(bridgeID, port1, port2 string, baudRate int) (*Bridge, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.bridges[bridgeID]; exists {
		return nil, errors.New(errors.CodeConflict, "bridge ID already exists")
	}

	bridgePort1 := m.bridgeEndpointLocked(port1)
	bridgePort2 := m.bridgeEndpointLocked(port2)
	bridge := NewBridgeWithEndpoints(bridgeID, port1, port2, bridgePort1, bridgePort2, baudRate)
	if err := bridge.Start(); err != nil {
		return nil, err
	}

	m.bridges[bridgeID] = bridge
	return bridge, nil
}

func (m *Manager) bridgeEndpointLocked(portName string) string {
	for _, pair := range m.pairs {
		if pair.port2Hidden && pair.Port1 == portName {
			return pair.Port2
		}
	}
	return portName
}

// EndpointFor returns the actual device endpoint to open for a public virtual
// port name. User-facing single virtual ports expose Port1 while their hidden
// peer is opened internally for bridges and monitors.
func (m *Manager) EndpointFor(portName string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.bridgeEndpointLocked(portName)
}

// DeleteBridge removes a bridge.
func (m *Manager) DeleteBridge(bridgeID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	bridge, exists := m.bridges[bridgeID]
	if !exists {
		return errors.New(errors.CodeNotFound, "bridge not found")
	}

	bridge.Stop()
	delete(m.bridges, bridgeID)
	return nil
}

// ListBridges returns all active bridges.
func (m *Manager) ListBridges() []*Bridge {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*Bridge, 0, len(m.bridges))
	for _, bridge := range m.bridges {
		result = append(result, bridge)
	}
	return result
}

// Cleanup stops all pairs and bridges.
func (m *Manager) Cleanup() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, bridge := range m.bridges {
		bridge.Stop()
	}
	m.bridges = make(map[string]*Bridge)

	for _, pair := range m.pairs {
		_ = pair.Stop()
	}
	m.pairs = make(map[string]*VirtualPair)
}
