package virtualserial

import (
	"context"
	"fmt"
	"os/exec"
	"sync"
	"time"

	"github.com/suyue/mocktrue/internal/core/errors"
)

// Manager manages multiple virtual serial pairs and bridges.
type Manager struct {
	pairs   map[string]*VirtualPair // key: pairID
	bridges map[string]*Bridge      // key: bridgeID
	mu      sync.RWMutex
}

// NewManager creates a virtual serial manager.
func NewManager() *Manager {
	return &Manager{
		pairs:   make(map[string]*VirtualPair),
		bridges: make(map[string]*Bridge),
	}
}

// CreatePair creates a new virtual serial pair with custom names.
func (m *Manager) CreatePair(ctx context.Context, pairID, port1Name, port2Name string) (*VirtualPair, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	_ = ctx

	if _, exists := m.pairs[pairID]; exists {
		return nil, errors.New(errors.CodeConflict, "pair ID already exists")
	}

	// Create socat command with custom port names
	port1Path := fmt.Sprintf("/tmp/%s", port1Name)
	port2Path := fmt.Sprintf("/tmp/%s", port2Name)

	// Remove old symlinks if they exist
	exec.Command("rm", "-f", port1Path, port2Path).Run()

	cmd := exec.Command("socat", "-d", "-d",
		fmt.Sprintf("pty,raw,echo=0,link=%s", port1Path),
		fmt.Sprintf("pty,raw,echo=0,link=%s", port2Path))

	if err := cmd.Start(); err != nil {
		return nil, errors.Wrap(errors.CodeIO, "start socat", err)
	}

	// Wait for symlinks to appear
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if fileExists(port1Path) && fileExists(port2Path) {
			pair := &VirtualPair{
				ID:    pairID,
				Port1: port1Path,
				Port2: port2Path,
				cmd:   cmd,
			}
			m.pairs[pairID] = pair
			return pair, nil
		}
		time.Sleep(50 * time.Millisecond)
	}

	cmd.Process.Kill()
	return nil, errors.New(errors.CodeIO, "timeout waiting for socat symlinks")
}

// DeletePair removes a virtual serial pair.
func (m *Manager) DeletePair(pairID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	pair, exists := m.pairs[pairID]
	if !exists {
		return errors.New(errors.CodeNotFound, "pair not found")
	}

	pair.Stop()
	delete(m.pairs, pairID)
	return nil
}

// CreatePort creates a user-facing virtual serial port. A hidden peer is
// created internally because PTYs are backed by pairs on Unix-like systems.
func (m *Manager) CreatePort(ctx context.Context, portID, portName string) (*VirtualPair, error) {
	if portName == "" {
		return nil, errors.New(errors.CodeInvalid, "port name must not be empty")
	}
	pair, err := m.CreatePair(ctx, portID, portName, portName+"-peer")
	if err != nil {
		return nil, err
	}
	pair.port2Hidden = true
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
		pair.Stop()
	}
	m.pairs = make(map[string]*VirtualPair)
}

func fileExists(path string) bool {
	_, err := exec.Command("test", "-e", path).CombinedOutput()
	return err == nil
}
