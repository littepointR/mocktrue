package serial

import (
	"context"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/littepointR/mocktrue/internal/core/errors"
	"github.com/littepointR/mocktrue/internal/core/eventbus"
	"github.com/littepointR/mocktrue/internal/modules/serial/buffer"
	fb "github.com/littepointR/mocktrue/internal/modules/serial/fecbus"
	"github.com/littepointR/mocktrue/internal/modules/serial/manager"
	mb "github.com/littepointR/mocktrue/internal/modules/serial/modbus"
	"github.com/littepointR/mocktrue/internal/modules/serial/monitor"
	"github.com/littepointR/mocktrue/internal/modules/serial/port"
	"github.com/littepointR/mocktrue/internal/modules/serial/virtualserial"
)

// Service is the serial module's facade exposed to the frontend.
type Service struct {
	mu                  sync.RWMutex
	bus                 *eventbus.EventBus
	manager             *manager.PortManager
	monitors            *monitor.Manager
	modbus              *mb.Manager
	fecbus              *fb.Manager
	vmgr                *virtualserial.Manager
	serialBackend       port.Backend
	buffers             map[string]*buffer.RingBuffer // keyed by handle ID
	autoMonitorVirtuals map[string]string
	graphs              map[string]*serialGraphRuntime
	subscribed          bool
}

// ServiceOption customizes a Service during construction.
type ServiceOption func(*Service)

// WithPortBackend injects the serial backend used for enumeration and opening
// ports. A nil backend is ignored so production defaults remain safe.
func WithPortBackend(backend port.Backend) ServiceOption {
	return func(s *Service) {
		if backend != nil {
			s.serialBackend = backend
		}
	}
}

// NewService constructs a Service with the given event bus.
func NewService(bus *eventbus.EventBus, options ...ServiceOption) *Service {
	svc := &Service{}
	for _, option := range options {
		option(svc)
	}
	svc.init(bus)
	return svc
}

func (s *Service) init(bus *eventbus.EventBus) {
	if bus == nil {
		bus = eventbus.New()
	}
	s.mu.Lock()
	if s.bus == nil {
		s.bus = bus
	}
	activeBus := s.bus
	if s.serialBackend == nil {
		s.serialBackend = port.RealBackend{}
	}
	if s.manager == nil {
		s.manager = manager.NewManager(activeBus, manager.WithBackend(s.serialBackend))
	}
	if s.monitors == nil {
		s.monitors = monitor.NewManager()
	}
	if s.modbus == nil {
		s.modbus = mb.NewManager(nil)
	}
	if s.fecbus == nil {
		s.fecbus = fb.NewManager(nil)
	}
	if s.vmgr == nil {
		s.vmgr = virtualserial.NewManager()
	}
	if s.buffers == nil {
		s.buffers = make(map[string]*buffer.RingBuffer)
	}
	if s.autoMonitorVirtuals == nil {
		s.autoMonitorVirtuals = make(map[string]string)
	}
	if s.graphs == nil {
		s.graphs = make(map[string]*serialGraphRuntime)
	}
	if s.subscribed {
		s.mu.Unlock()
		return
	}
	s.subscribed = true
	s.mu.Unlock()

	// Subscribe to serial:data to populate buffers from readLoop
	activeBus.Subscribe("serial:data", func(payload any) {
		if evt, ok := payload.(manager.DataEvent); ok {
			s.mu.Lock()
			buf, exists := s.buffers[evt.PortID]
			s.mu.Unlock()
			if exists {
				buf.Append(buffer.Chunk{
					Seq:        0,
					BaseOffset: buf.Total(),
					Data:       evt.Data,
				})
			}
		}
	})
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
	s.init(nil)
	ports, err := s.serialBackend.Enumerate(ctx)
	if err != nil {
		return nil, err
	}
	for _, pair := range s.ListVirtualPorts() {
		ports = append(ports,
			port.PortInfo{Name: pair.Port, FriendlyName: pair.Port},
		)
	}
	return ports, nil
}

// OpenPort opens a serial port and starts a read loop.
func (s *Service) OpenPort(ctx context.Context, req manager.OpenRequest) (*manager.HandleStatus, error) {
	if existing := s.findOpenHandle(req.Config.PortName); existing != nil {
		return existing, nil
	}
	if s.modbus != nil && s.modbus.PortInUse(req.Config.PortName) {
		return nil, errors.New(errors.CodeConflict, fmt.Sprintf("port already open in modbus session: %s", req.Config.PortName))
	}
	if s.fecbus != nil && s.fecbus.PortInUse(req.Config.PortName) {
		return nil, errors.New(errors.CodeConflict, fmt.Sprintf("port already open in fecbus session: %s", req.Config.PortName))
	}
	status, err := s.manager.Open(ctx, req)
	if err != nil {
		if errors.AsCode(err) == errors.CodeConflict {
			if existing := s.findOpenHandle(req.Config.PortName); existing != nil {
				return existing, nil
			}
		}
		return nil, err
	}
	// Create a ring buffer for this handle
	s.mu.Lock()
	s.buffers[status.ID] = buffer.NewRing(256 * 1024 * 1024) // 256MB
	s.mu.Unlock()
	return status, nil
}

func (s *Service) findOpenHandle(portName string) *manager.HandleStatus {
	if portName == "" {
		return nil
	}
	for _, handle := range s.manager.List() {
		if handle.IsOpen && handle.Config.PortName == portName {
			h := handle
			return &h
		}
	}
	return nil
}

// ClosePort closes a serial port and removes its buffer.
func (s *Service) ClosePort(id string) error {
	err := s.manager.Close(id)
	if err != nil {
		return err
	}
	s.mu.Lock()
	delete(s.buffers, id)
	s.mu.Unlock()
	return nil
}

// cleanup releases every resource owned by the serial service: open ports,
// virtual serial pairs, bridges, and in-memory buffers. It is idempotent.
func (s *Service) cleanup() {
	s.stopAllSerialGraphs()
	if s.manager != nil {
		s.manager.CloseAll()
	}
	if s.monitors != nil {
		s.monitors.Cleanup()
	}
	if s.modbus != nil {
		s.modbus.CloseAll()
	}
	if s.fecbus != nil {
		s.fecbus.CloseAll()
	}
	if s.vmgr != nil {
		s.vmgr.Cleanup()
	}
	s.mu.Lock()
	s.buffers = make(map[string]*buffer.RingBuffer)
	s.autoMonitorVirtuals = make(map[string]string)
	s.graphs = make(map[string]*serialGraphRuntime)
	s.mu.Unlock()
}

// ListPorts returns a snapshot of all open handles.
func (s *Service) ListPorts() []manager.HandleStatus {
	return s.manager.List()
}

// QueryPage returns a snapshot of the buffer for a given port handle.
func (s *Service) QueryPage(portID string, offset int64, length int) (*buffer.Snapshot, error) {
	s.mu.RLock()
	buf, ok := s.buffers[portID]
	s.mu.RUnlock()
	if !ok {
		return nil, errors.New(errors.CodeNotFound, fmt.Sprintf("port not found: %s", portID))
	}
	return buf.Query(offset, length)
}

// SendRequest bundles parameters for sending data.
type SendRequest struct {
	PortID   string
	Content  string
	Mode     string // "ascii" or "hex"
	Encoding string
}

// EncodeTextRequest bundles parameters for converting text to HEX.
type EncodeTextRequest struct {
	Content  string
	Encoding string
}

// DecodeHexRequest bundles parameters for converting HEX to text.
type DecodeHexRequest struct {
	Content  string
	Encoding string
}

// EncodeTextToHex converts text to formatted HEX using the requested encoding.
func (s *Service) EncodeTextToHex(req EncodeTextRequest) (string, error) {
	data, err := encodeSerialText(req.Content, req.Encoding)
	if err != nil {
		return "", errors.Wrap(errors.CodeInvalid, "encode text content", err)
	}
	return formatHexBytes(data), nil
}

// DecodeHexToText converts formatted HEX to text using the requested encoding.
func (s *Service) DecodeHexToText(req DecodeHexRequest) (string, error) {
	data, err := decodeHexContent(req.Content)
	if err != nil {
		return "", errors.Wrap(errors.CodeInvalid, "decode hex content", err)
	}
	text, err := decodeSerialText(data, req.Encoding)
	if err != nil {
		return "", errors.Wrap(errors.CodeInvalid, "decode text content", err)
	}
	return text, nil
}

// Send sends data to the specified port.
func (s *Service) Send(req SendRequest) (int, error) {
	if req.PortID == "" {
		return 0, errors.New(errors.CodeInvalid, "port ID must not be empty")
	}
	if req.Content == "" {
		return 0, errors.New(errors.CodeInvalid, "content must not be empty")
	}
	var data []byte
	if req.Mode == "hex" {
		decoded, err := decodeHexContent(req.Content)
		if err != nil {
			return 0, errors.Wrap(errors.CodeInvalid, "decode hex content", err)
		}
		data = decoded
	} else {
		var err error
		data, err = encodeSerialText(req.Content, req.Encoding)
		if err != nil {
			return 0, errors.Wrap(errors.CodeInvalid, "encode text content", err)
		}
	}
	return s.manager.Write(req.PortID, data)
}

func decodeHexContent(content string) ([]byte, error) {
	compact := strings.NewReplacer(" ", "", "\n", "", "\t", "", "\r", "").Replace(content)
	return hex.DecodeString(compact)
}

func formatHexBytes(data []byte) string {
	if len(data) == 0 {
		return ""
	}
	const hex = "0123456789abcdef"
	out := make([]byte, len(data)*3-1)
	for i, b := range data {
		j := i * 3
		out[j] = hex[b>>4]
		out[j+1] = hex[b&0x0f]
		if i < len(data)-1 {
			out[j+2] = ' '
		}
	}
	return string(out)
}

// ResetCounters clears RX and TX byte counters for an open port handle.
func (s *Service) ResetCounters(portID string) error {
	return s.manager.ResetCounters(portID)
}

// RestoreCounters sets RX and TX byte counters for an open port handle.
func (s *Service) RestoreCounters(portID string, rxBytes int64, txBytes int64) error {
	return s.manager.RestoreCounters(portID, rxBytes, txBytes)
}

// AutoVirtualMonitorRequest starts monitoring one real serial port while
// exposing an automatically-created virtual port for external tools.
type AutoVirtualMonitorRequest struct {
	ID       string
	Name     string
	Port     string
	Config   port.SerialConfig
	Encoding string
}

// StartMonitor starts a bridge-based serial monitor session.
func (s *Service) StartMonitor(ctx context.Context, req monitor.StartRequest) (*monitor.SessionInfo, error) {
	if s.findOpenHandle(req.PortA) != nil || s.findOpenHandle(req.PortB) != nil {
		return nil, errors.New(errors.CodeConflict, "monitor port already open in serial terminal")
	}
	if s.modbus != nil && (s.modbus.PortInUse(req.PortA) || s.modbus.PortInUse(req.PortB)) {
		return nil, errors.New(errors.CodeConflict, "monitor port already open in modbus session")
	}
	if s.fecbus != nil && (s.fecbus.PortInUse(req.PortA) || s.fecbus.PortInUse(req.PortB)) {
		return nil, errors.New(errors.CodeConflict, "monitor port already open in fecbus session")
	}
	req.EndpointA = s.monitorEndpoint(req.PortA)
	req.EndpointB = s.monitorEndpoint(req.PortB)
	return s.monitors.Start(ctx, req)
}

// StartAutoVirtualMonitor starts monitoring a real port and creates one
// user-facing virtual port that external serial tools can connect to.
func (s *Service) StartAutoVirtualMonitor(ctx context.Context, req AutoVirtualMonitorRequest) (*monitor.SessionInfo, error) {
	if req.ID == "" {
		return nil, errors.New(errors.CodeInvalid, "monitor ID must not be empty")
	}
	if req.Port == "" {
		return nil, errors.New(errors.CodeInvalid, "monitor port must not be empty")
	}
	if s.findOpenHandle(req.Port) != nil {
		return nil, errors.New(errors.CodeConflict, "monitor port already open in serial terminal")
	}
	if s.modbus != nil && s.modbus.PortInUse(req.Port) {
		return nil, errors.New(errors.CodeConflict, "monitor port already open in modbus session")
	}
	if s.fecbus != nil && s.fecbus.PortInUse(req.Port) {
		return nil, errors.New(errors.CodeConflict, "monitor port already open in fecbus session")
	}

	virtualID := autoVirtualPortID(req.ID)
	virtualPortName := autoVirtualPortName(req.Port, req.ID)
	virtualPort, err := s.CreateVirtualPort(ctx, virtualID, virtualPortName)
	if err != nil {
		return nil, err
	}

	name := req.Name
	if name == "" {
		name = fmt.Sprintf("%s monitor", req.Port)
	}
	session, err := s.StartMonitor(ctx, monitor.StartRequest{
		ID:                req.ID,
		Name:              name,
		Provider:          monitor.ProviderBridge,
		PortA:             req.Port,
		PortB:             virtualPort.Port,
		ExternalPort:      virtualPort.Port,
		AutoVirtualPortID: virtualPort.ID,
		Config:            req.Config,
		Encoding:          req.Encoding,
	})
	if err != nil {
		_ = s.DeleteVirtualPort(virtualPort.ID)
		return nil, err
	}

	s.mu.Lock()
	s.autoMonitorVirtuals[req.ID] = virtualPort.ID
	s.mu.Unlock()
	return session, nil
}

// StopMonitor stops a monitor session while keeping captured frames available.
func (s *Service) StopMonitor(id string) error {
	if err := s.monitors.Stop(id); err != nil {
		return err
	}
	return s.cleanupAutoMonitorVirtual(id)
}

// DeleteMonitor stops and removes a monitor session.
func (s *Service) DeleteMonitor(id string) error {
	if err := s.monitors.Delete(id); err != nil {
		return err
	}
	return s.cleanupAutoMonitorVirtual(id)
}

// ListMonitors returns all serial monitor sessions.
func (s *Service) ListMonitors() []monitor.SessionInfo {
	return s.monitors.List()
}

// QueryMonitorFrames returns a filtered page of monitor frames.
func (s *Service) QueryMonitorFrames(req monitor.QueryRequest) (*monitor.FramePage, error) {
	return s.monitors.Query(req)
}

// ClearMonitorFrames clears captured monitor frames and counters.
func (s *Service) ClearMonitorFrames(id string) error {
	return s.monitors.ClearFrames(id)
}

// OpenModbusSession opens a dedicated Modbus serial session.
func (s *Service) OpenModbusSession(ctx context.Context, req mb.OpenSessionRequest) (*mb.SessionInfo, error) {
	if req.Config.PortName == "" {
		return nil, errors.New(errors.CodeInvalid, "modbus port must not be empty")
	}
	if s.findOpenHandle(req.Config.PortName) != nil {
		return nil, errors.New(errors.CodeConflict, "modbus port already open in serial terminal")
	}
	if s.monitorPortInUse(req.Config.PortName) {
		return nil, errors.New(errors.CodeConflict, "modbus port already used by serial monitor")
	}
	if s.fecbus != nil && s.fecbus.PortInUse(req.Config.PortName) {
		return nil, errors.New(errors.CodeConflict, "modbus port already open in fecbus session")
	}
	if s.bridgePortInUse(req.Config.PortName) {
		return nil, errors.New(errors.CodeConflict, "modbus port already used by serial bridge")
	}
	req.Endpoint = s.monitorEndpoint(req.Config.PortName)
	return s.modbus.OpenSession(ctx, req)
}

// CloseModbusSession closes and removes a Modbus serial session.
func (s *Service) CloseModbusSession(id string) error {
	return s.modbus.CloseSession(id)
}

// ListModbusSessions returns all open Modbus serial sessions.
func (s *Service) ListModbusSessions() []mb.SessionInfo {
	return s.modbus.List()
}

// ModbusMasterRequest runs one Modbus master transaction.
func (s *Service) ModbusMasterRequest(req mb.MasterRequest) (*mb.Transaction, error) {
	return s.modbus.MasterRequest(req)
}

// StartModbusSlave starts slave simulation for an open Modbus session.
func (s *Service) StartModbusSlave(req mb.StartSlaveRequest) (*mb.SessionInfo, error) {
	return s.modbus.StartSlave(req)
}

// StopModbusSlave stops slave simulation for an open Modbus session.
func (s *Service) StopModbusSlave(id string) error {
	return s.modbus.StopSlave(id)
}

// UpdateModbusSlaveData replaces the slave simulation data model.
func (s *Service) UpdateModbusSlaveData(sessionID string, data mb.DataModelSnapshot) error {
	return s.modbus.UpdateSlaveData(sessionID, data)
}

// AddModbusSlaveUnit adds or replaces one Unit ID in a slave simulation.
func (s *Service) AddModbusSlaveUnit(sessionID string, unit mb.SlaveUnitSnapshot) error {
	return s.modbus.AddSlaveUnit(sessionID, unit)
}

// RemoveModbusSlaveUnit removes one Unit ID from a slave simulation.
func (s *Service) RemoveModbusSlaveUnit(sessionID string, unitID byte) error {
	return s.modbus.RemoveSlaveUnit(sessionID, unitID)
}

// ListModbusSlaveUnits returns configured Unit IDs for a Modbus session.
func (s *Service) ListModbusSlaveUnits(sessionID string) ([]mb.SlaveUnitInfo, error) {
	return s.modbus.ListSlaveUnits(sessionID)
}

// UpdateModbusSlaveUnitData replaces one Unit ID data model.
func (s *Service) UpdateModbusSlaveUnitData(sessionID string, unitID byte, data mb.DataModelSnapshot) error {
	return s.modbus.UpdateSlaveUnitData(sessionID, unitID, data)
}

// ModbusScanUnitIDs probes a set of Modbus Unit IDs.
func (s *Service) ModbusScanUnitIDs(req mb.UnitScanRequest) (*mb.UnitScanResult, error) {
	return s.modbus.ScanUnitIDs(req)
}

// ModbusReadRegisters reads and decodes one Modbus register block.
func (s *Service) ModbusReadRegisters(req mb.RegisterReadRequest) (*mb.RegisterReadResult, error) {
	return s.modbus.ReadRegisters(req)
}

// ModbusScanRegisters scans a Modbus register range.
func (s *Service) ModbusScanRegisters(req mb.RegisterScanRequest) (*mb.RegisterScanResult, error) {
	return s.modbus.ScanRegisters(req)
}

// OpenFecbusSession opens a dedicated FECbus serial session.
func (s *Service) OpenFecbusSession(ctx context.Context, req fb.OpenSessionRequest) (*fb.SessionInfo, error) {
	if req.Config.PortName == "" {
		return nil, errors.New(errors.CodeInvalid, "fecbus port must not be empty")
	}
	if s.findOpenHandle(req.Config.PortName) != nil {
		return nil, errors.New(errors.CodeConflict, "fecbus port already open in serial terminal")
	}
	if s.monitorPortInUse(req.Config.PortName) {
		return nil, errors.New(errors.CodeConflict, "fecbus port already used by serial monitor")
	}
	if s.modbus != nil && s.modbus.PortInUse(req.Config.PortName) {
		return nil, errors.New(errors.CodeConflict, "fecbus port already open in modbus session")
	}
	if s.bridgePortInUse(req.Config.PortName) {
		return nil, errors.New(errors.CodeConflict, "fecbus port already used by serial bridge")
	}
	req.Endpoint = s.monitorEndpoint(req.Config.PortName)
	return s.fecbus.OpenSession(ctx, req)
}

// CloseFecbusSession closes and removes a FECbus serial session.
func (s *Service) CloseFecbusSession(id string) error {
	return s.fecbus.CloseSession(id)
}

// ListFecbusSessions returns all open FECbus serial sessions.
func (s *Service) ListFecbusSessions() []fb.SessionInfo {
	return s.fecbus.List()
}

// FecbusSendRequest sends one FECbus frame and optionally waits for an answer.
func (s *Service) FecbusSendRequest(req fb.SendRequest) (*fb.Transaction, error) {
	return s.fecbus.SendRequest(req)
}

// StartFecbusSlave starts FECbus device simulation for an open session.
func (s *Service) StartFecbusSlave(req fb.StartSlaveRequest) (*fb.SessionInfo, error) {
	return s.fecbus.StartSlave(req)
}

// StopFecbusSlave stops FECbus device simulation for an open session.
func (s *Service) StopFecbusSlave(id string) error {
	return s.fecbus.StopSlave(id)
}

// UpdateFecbusSlaveState replaces FECbus device simulation state.
func (s *Service) UpdateFecbusSlaveState(sessionID string, state fb.SlaveState) error {
	return s.fecbus.UpdateSlaveState(sessionID, state)
}

// AddFecbusSlaveUnit adds or replaces one FECbus simulated slave address.
func (s *Service) AddFecbusSlaveUnit(sessionID string, unit fb.SlaveUnitState) error {
	return s.fecbus.AddSlaveUnit(sessionID, unit)
}

// RemoveFecbusSlaveUnit removes one FECbus simulated slave address.
func (s *Service) RemoveFecbusSlaveUnit(sessionID string, address byte) error {
	return s.fecbus.RemoveSlaveUnit(sessionID, address)
}

// ListFecbusSlaveUnits returns configured FECbus slave addresses.
func (s *Service) ListFecbusSlaveUnits(sessionID string) ([]fb.SlaveUnitInfo, error) {
	return s.fecbus.ListSlaveUnits(sessionID)
}

// QueryFecbusFrames returns a filtered FECbus frame history page.
func (s *Service) QueryFecbusFrames(req fb.QueryRequest) (*fb.FramePage, error) {
	return s.fecbus.QueryFrames(req)
}

// ClearFecbusFrames clears FECbus frame history and counters.
func (s *Service) ClearFecbusFrames(id string) error {
	return s.fecbus.ClearFrames(id)
}

func (s *Service) monitorEndpoint(portName string) string {
	if s.vmgr == nil {
		return portName
	}
	return s.vmgr.EndpointFor(portName)
}

func (s *Service) monitorPortInUse(portName string) bool {
	if portName == "" || s.monitors == nil {
		return false
	}
	for _, session := range s.monitors.List() {
		if session.Status == monitor.StatusRunning && (session.PortA == portName || session.PortB == portName || session.ExternalPort == portName) {
			return true
		}
	}
	return false
}

func (s *Service) bridgePortInUse(portName string) bool {
	if portName == "" || s.vmgr == nil {
		return false
	}
	for _, bridge := range s.vmgr.ListBridges() {
		if bridge.Port1 == portName || bridge.Port2 == portName {
			return true
		}
	}
	return false
}

func (s *Service) cleanupAutoMonitorVirtual(monitorID string) error {
	s.mu.RLock()
	virtualID := s.autoMonitorVirtuals[monitorID]
	s.mu.RUnlock()
	if virtualID == "" {
		return nil
	}
	err := s.DeleteVirtualPort(virtualID)
	if err != nil && errors.AsCode(err) != errors.CodeNotFound {
		return err
	}
	s.mu.Lock()
	if s.autoMonitorVirtuals[monitorID] == virtualID {
		delete(s.autoMonitorVirtuals, monitorID)
	}
	s.mu.Unlock()
	return nil
}

func autoVirtualPortID(monitorID string) string {
	token := sanitizeSerialToken(monitorID)
	if token == "" {
		token = "monitor"
	}
	return fmt.Sprintf("auto-monitor-%s-%x", token, time.Now().UnixNano())
}

func autoVirtualPortName(portName string, monitorID string) string {
	portToken := sanitizeSerialToken(filepath.Base(portName))
	if portToken == "" || portToken == "." || portToken == string(filepath.Separator) {
		portToken = "port"
	}
	idToken := sanitizeSerialToken(monitorID)
	if idToken == "" {
		idToken = "monitor"
	}
	return fmt.Sprintf("mocktrue-%s-%s-%x", portToken, idToken, time.Now().UnixNano())
}

func sanitizeSerialToken(input string) string {
	var builder strings.Builder
	lastDash := false
	for _, r := range strings.TrimSpace(input) {
		allowed := (r >= 'a' && r <= 'z') ||
			(r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') ||
			r == '_' ||
			r == '.'
		if allowed {
			builder.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash && builder.Len() > 0 {
			builder.WriteByte('-')
			lastDash = true
		}
	}
	result := strings.Trim(builder.String(), "-.")
	if len(result) > 48 {
		result = result[:48]
	}
	return result
}

// ===== Virtual Serial Pair API =====

// VirtualPortInfo represents a user-facing virtual serial port. The backing
// peer is intentionally hidden from the frontend.
type VirtualPortInfo struct {
	ID   string
	Port string
}

// VirtualPairInfo represents a virtual serial pair for tests and bridge setup.
type VirtualPairInfo struct {
	ID    string
	Port1 string
	Port2 string
}

// CreateVirtualPort creates a user-facing virtual serial port.
func (s *Service) CreateVirtualPort(ctx context.Context, id, portName string) (*VirtualPortInfo, error) {
	if id == "" {
		return nil, errors.New(errors.CodeInvalid, "id must not be empty")
	}
	if portName == "" {
		return nil, errors.New(errors.CodeInvalid, "port name must not be empty")
	}

	pair, err := s.vmgr.CreatePort(ctx, id, portName)
	if err != nil {
		return nil, err
	}
	return &VirtualPortInfo{
		ID:   pair.ID,
		Port: pair.Port1,
	}, nil
}

// DeleteVirtualPort removes a user-facing virtual serial port.
func (s *Service) DeleteVirtualPort(id string) error {
	return s.DeleteVirtualPair(id)
}

// ListVirtualPorts returns all user-facing virtual serial ports.
func (s *Service) ListVirtualPorts() []VirtualPortInfo {
	pairs := s.vmgr.ListPairs()
	result := make([]VirtualPortInfo, 0, len(pairs))
	for _, p := range pairs {
		result = append(result, VirtualPortInfo{
			ID:   p.ID,
			Port: p.Port1,
		})
	}
	return result
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
	if s.modbus != nil && (s.modbus.PortInUse(port1) || s.modbus.PortInUse(port2)) {
		return nil, errors.New(errors.CodeConflict, "bridge port already open in modbus session")
	}
	if s.fecbus != nil && (s.fecbus.PortInUse(port1) || s.fecbus.PortInUse(port2)) {
		return nil, errors.New(errors.CodeConflict, "bridge port already open in fecbus session")
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
	s.cleanup()
}
