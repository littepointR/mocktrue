package serial

import (
	"context"
	"fmt"

	"github.com/suyue/mocktrue/internal/core/errors"
	"github.com/suyue/mocktrue/internal/core/eventbus"
	"github.com/suyue/mocktrue/internal/modules/serial/buffer"
	"github.com/suyue/mocktrue/internal/modules/serial/manager"
	"github.com/suyue/mocktrue/internal/modules/serial/port"
	"github.com/suyue/mocktrue/internal/modules/serial/virtualserial"
)

// Service is the serial module's facade exposed to the frontend.
type Service struct {
	bus     *eventbus.EventBus
	manager *manager.PortManager
	vmgr    *virtualserial.Manager
	buffers map[string]*buffer.RingBuffer // keyed by handle ID
}

// NewService constructs a Service with the given event bus.
func NewService(bus *eventbus.EventBus) *Service {
	return &Service{
		bus:     bus,
		manager: manager.NewManager(bus),
		vmgr:    virtualserial.NewManager(),
		buffers: make(map[string]*buffer.RingBuffer),
	}
}

// ServiceName provides a friendly service name for logging.
func (s *Service) ServiceName() string { return "serial" }

// Ping echoes msg back as "pong:<msg>", validating the binding channel.
func (s *Service) Ping(ctx context.Context, msg string) (string, error) {
	if msg == "" {
		return "", errors.New(errors.CodeInvalid, "msg must not be empty")
	}
	return "pong:" + msg, nil
}

// EnumeratePorts returns available serial ports.
func (s *Service) EnumeratePorts(ctx context.Context) ([]port.PortInfo, error) {
	return port.Enumerate(ctx)
}

// OpenPort opens a serial port and starts a read loop.
func (s *Service) OpenPort(ctx context.Context, req manager.OpenRequest) (*manager.HandleStatus, error) {
	status, err := s.manager.Open(ctx, req)
	if err != nil {
		return nil, err
	}
	// Create a ring buffer for this handle
	s.buffers[status.ID] = buffer.NewRing(256 * 1024 * 1024) // 256MB
	return status, nil
}

// ClosePort closes a serial port and removes its buffer.
func (s *Service) ClosePort(id string) error {
	err := s.manager.Close(id)
	if err != nil {
		return err
	}
	delete(s.buffers, id)
	return nil
}

// ListPorts returns a snapshot of all open handles.
func (s *Service) ListPorts() []manager.HandleStatus {
	return s.manager.List()
}

// QueryPage returns a snapshot of the buffer for a given port handle.
func (s *Service) QueryPage(portID string, offset int64, length int) (*buffer.Snapshot, error) {
	buf, ok := s.buffers[portID]
	if !ok {
		return nil, errors.New(errors.CodeNotFound, fmt.Sprintf("port not found: %s", portID))
	}
	return buf.Query(offset, length)
}

// SendRequest bundles parameters for sending data.
type SendRequest struct {
	PortID  string
	Content string
	Mode    string // "ascii" or "hex"
}

// Send sends data to the specified port.
func (s *Service) Send(req SendRequest) (int, error) {
	if req.PortID == "" {
		return 0, errors.New(errors.CodeInvalid, "port ID must not be empty")
	}
	if req.Content == "" {
		return 0, errors.New(errors.CodeInvalid, "content must not be empty")
	}
	_ = req.Mode // TODO: handle hex decode in later stage
	// TODO: actually write to the port via manager
	return len(req.Content), nil
}

// ===== Virtual Serial Pair API =====

// VirtualPairInfo represents a virtual serial pair for the frontend.
type VirtualPairInfo struct {
	ID    string
	Port1 string
	Port2 string
}

// CreateVirtualPair creates a new virtual serial pair.
func (s *Service) CreateVirtualPair(ctx context.Context, id, port1Name, port2Name string) (*VirtualPairInfo, error) {
	if id == "" {
		return nil, errors.New(errors.CodeInvalid, "id must not be empty")
	}
	if port1Name == "" || port2Name == "" {
		return nil, errors.New(errors.CodeInvalid, "port names must not be empty")
	}

	pair, err := s.vmgr.CreatePair(ctx, id, port1Name, port2Name)
	if err != nil {
		return nil, err
	}

	return &VirtualPairInfo{
		ID:    pair.ID,
		Port1: pair.Port1,
		Port2: pair.Port2,
	}, nil
}

// DeleteVirtualPair removes a virtual serial pair.
func (s *Service) DeleteVirtualPair(id string) error {
	if id == "" {
		return errors.New(errors.CodeInvalid, "id must not be empty")
	}
	return s.vmgr.DeletePair(id)
}

// ListVirtualPairs returns all virtual serial pairs.
func (s *Service) ListVirtualPairs() []VirtualPairInfo {
	pairs := s.vmgr.ListPairs()
	result := make([]VirtualPairInfo, 0, len(pairs))
	for _, p := range pairs {
		result = append(result, VirtualPairInfo{
			ID:    p.ID,
			Port1: p.Port1,
			Port2: p.Port2,
		})
	}
	return result
}

// ===== Bridge API =====

// BridgeInfo represents a serial bridge for the frontend.
type BridgeInfo struct {
	ID       string
	Port1    string
	Port2    string
	BaudRate int
}

// CreateBridge creates a bridge between two serial ports.
func (s *Service) CreateBridge(id, port1, port2 string, baudRate int) (*BridgeInfo, error) {
	if id == "" {
		return nil, errors.New(errors.CodeInvalid, "id must not be empty")
	}
	if port1 == "" || port2 == "" {
		return nil, errors.New(errors.CodeInvalid, "port names must not be empty")
	}
	if port1 == port2 {
		return nil, errors.New(errors.CodeInvalid, "cannot bridge a port to itself")
	}
	if baudRate <= 0 {
		baudRate = 115200
	}

	bridge, err := s.vmgr.CreateBridge(id, port1, port2, baudRate)
	if err != nil {
		return nil, err
	}

	return &BridgeInfo{
		ID:       bridge.ID,
		Port1:    bridge.Port1,
		Port2:    bridge.Port2,
		BaudRate: bridge.BaudRate,
	}, nil
}

// DeleteBridge removes a bridge.
func (s *Service) DeleteBridge(id string) error {
	if id == "" {
		return errors.New(errors.CodeInvalid, "id must not be empty")
	}
	return s.vmgr.DeleteBridge(id)
}

// ListBridges returns all active bridges.
func (s *Service) ListBridges() []BridgeInfo {
	bridges := s.vmgr.ListBridges()
	result := make([]BridgeInfo, 0, len(bridges))
	for _, b := range bridges {
		result = append(result, BridgeInfo{
			ID:       b.ID,
			Port1:    b.Port1,
			Port2:    b.Port2,
			BaudRate: b.BaudRate,
		})
	}
	return result
}

// CleanupVirtual stops all virtual pairs and bridges.
func (s *Service) CleanupVirtual() {
	s.vmgr.Cleanup()
}
