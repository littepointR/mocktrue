package modbus

import (
	"context"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/suyue/mocktrue/internal/modules/serial/port"
)

const (
	defaultTimeout = 800 * time.Millisecond
	readPoll       = 20 * time.Millisecond
	maxFrameBytes  = 513
)

// PortOpener opens a serial port. It is injectable for tests.
type PortOpener func(port.SerialConfig) (port.Port, error)

// Manager owns dedicated Modbus serial sessions.
type Manager struct {
	mu       sync.RWMutex
	sessions map[string]*Session
	open     PortOpener
	nextID   atomic.Int64
}

// NewManager constructs a Modbus session manager.
func NewManager(open PortOpener) *Manager {
	if open == nil {
		open = port.Open
	}
	return &Manager{
		sessions: make(map[string]*Session),
		open:     open,
	}
}

// OpenSession opens a serial port for Modbus-only use.
func (m *Manager) OpenSession(ctx context.Context, req OpenSessionRequest) (*SessionInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if req.Config.PortName == "" {
		return nil, fmt.Errorf("port name must not be empty")
	}
	if req.ID == "" {
		req.ID = fmt.Sprintf("modbus-%d", m.nextID.Add(1))
	}
	if req.Name == "" {
		req.Name = fmt.Sprintf("Modbus %s", req.Config.PortName)
	}
	if req.Mode == "" {
		req.Mode = FrameModeRTU
	}
	req.Role = normalizeSessionRole(req.Role)
	if req.Config.BaudRate <= 0 {
		req.Config.BaudRate = 115200
	}
	if req.Config.DataBits == 0 {
		req.Config.DataBits = 8
	}
	if req.Config.StopBits == "" {
		req.Config.StopBits = "1"
	}
	if req.Config.Parity == "" {
		req.Config.Parity = "none"
	}
	if req.Config.FlowMode == "" {
		req.Config.FlowMode = "none"
	}
	if req.Config.ReadBufKB == 0 {
		req.Config.ReadBufKB = 32
	}

	m.mu.Lock()
	if _, exists := m.sessions[req.ID]; exists {
		m.mu.Unlock()
		return nil, fmt.Errorf("modbus session already exists")
	}
	if m.portInUseLocked(req.Config.PortName) {
		m.mu.Unlock()
		return nil, fmt.Errorf("modbus port already in use")
	}
	m.mu.Unlock()

	openConfig := req.Config
	if req.Endpoint != "" {
		openConfig.PortName = req.Endpoint
	}
	serialPort, err := m.open(openConfig)
	if err != nil {
		return nil, fmt.Errorf("open modbus port %s: %w", req.Config.PortName, err)
	}
	_ = serialPort.SetReadTimeout(readPoll)

	session := newSession(req, serialPort)

	m.mu.Lock()
	if _, exists := m.sessions[req.ID]; exists {
		m.mu.Unlock()
		_ = serialPort.Close()
		return nil, fmt.Errorf("modbus session already exists")
	}
	if m.portInUseLocked(req.Config.PortName) {
		m.mu.Unlock()
		_ = serialPort.Close()
		return nil, fmt.Errorf("modbus port already in use")
	}
	m.sessions[req.ID] = session
	m.mu.Unlock()

	info := session.info()
	return &info, nil
}

// CloseSession closes and removes one Modbus session.
func (m *Manager) CloseSession(id string) error {
	m.mu.Lock()
	session, ok := m.sessions[id]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("modbus session not found")
	}
	delete(m.sessions, id)
	m.mu.Unlock()
	return session.close("")
}

// CloseAll closes every Modbus session.
func (m *Manager) CloseAll() {
	m.mu.Lock()
	sessions := m.sessions
	m.sessions = make(map[string]*Session)
	m.mu.Unlock()
	for _, session := range sessions {
		_ = session.close("")
	}
}

// List returns all Modbus sessions.
func (m *Manager) List() []SessionInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]SessionInfo, 0, len(m.sessions))
	for _, session := range m.sessions {
		result = append(result, session.info())
	}
	return result
}

// PortInUse reports whether a public port name is owned by Modbus.
func (m *Manager) PortInUse(portName string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.portInUseLocked(portName)
}

func (m *Manager) portInUseLocked(portName string) bool {
	for _, session := range m.sessions {
		info := session.info()
		if info.Status != SessionStatusClosed && info.Config.PortName == portName {
			return true
		}
	}
	return false
}

// MasterRequest runs one master request/response transaction.
func (m *Manager) MasterRequest(req MasterRequest) (*Transaction, error) {
	session, err := m.get(req.SessionID)
	if err != nil {
		return nil, err
	}
	return session.masterRequest(req)
}

// StartSlave starts slave simulation on an open session.
func (m *Manager) StartSlave(req StartSlaveRequest) (*SessionInfo, error) {
	session, err := m.get(req.SessionID)
	if err != nil {
		return nil, err
	}
	if err := session.startSlave(req); err != nil {
		return nil, err
	}
	info := session.info()
	return &info, nil
}

// StopSlave stops slave simulation while keeping the serial session open.
func (m *Manager) StopSlave(id string) error {
	session, err := m.get(id)
	if err != nil {
		return err
	}
	return session.stopSlave()
}

// UpdateSlaveData replaces the slave data model for a session.
func (m *Manager) UpdateSlaveData(sessionID string, snapshot DataModelSnapshot) error {
	session, err := m.get(sessionID)
	if err != nil {
		return err
	}
	session.setDataModel(DataModelFromSnapshot(snapshot))
	return nil
}

// AddSlaveUnit adds or replaces one simulated Unit ID for a session.
func (m *Manager) AddSlaveUnit(sessionID string, unit SlaveUnitSnapshot) error {
	session, err := m.get(sessionID)
	if err != nil {
		return err
	}
	return session.addSlaveUnit(unit)
}

// RemoveSlaveUnit removes one simulated Unit ID for a session.
func (m *Manager) RemoveSlaveUnit(sessionID string, unitID byte) error {
	session, err := m.get(sessionID)
	if err != nil {
		return err
	}
	return session.removeSlaveUnit(unitID)
}

// ListSlaveUnits returns configured simulated units for a session.
func (m *Manager) ListSlaveUnits(sessionID string) ([]SlaveUnitInfo, error) {
	session, err := m.get(sessionID)
	if err != nil {
		return nil, err
	}
	return session.listSlaveUnits(), nil
}

// UpdateSlaveUnitData replaces the data model for one simulated Unit ID.
func (m *Manager) UpdateSlaveUnitData(sessionID string, unitID byte, snapshot DataModelSnapshot) error {
	session, err := m.get(sessionID)
	if err != nil {
		return err
	}
	return session.updateSlaveUnitData(unitID, snapshot)
}

// ScanUnitIDs probes a set of Unit IDs with a read request.
func (m *Manager) ScanUnitIDs(req UnitScanRequest) (*UnitScanResult, error) {
	session, err := m.get(req.SessionID)
	if err != nil {
		return nil, err
	}
	return session.scanUnitIDs(req)
}

// ReadRegisters reads one block and decodes configured register mappings.
func (m *Manager) ReadRegisters(req RegisterReadRequest) (*RegisterReadResult, error) {
	session, err := m.get(req.SessionID)
	if err != nil {
		return nil, err
	}
	return session.readRegisters(req)
}

// ScanRegisters scans a register range and returns non-zero values.
func (m *Manager) ScanRegisters(req RegisterScanRequest) (*RegisterScanResult, error) {
	session, err := m.get(req.SessionID)
	if err != nil {
		return nil, err
	}
	return session.scanRegisters(req)
}

func (m *Manager) get(id string) (*Session, error) {
	if id == "" {
		return nil, fmt.Errorf("modbus session id must not be empty")
	}
	m.mu.RLock()
	session, ok := m.sessions[id]
	m.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("modbus session not found")
	}
	return session, nil
}

// Session is one open Modbus serial port.
type Session struct {
	mu            sync.RWMutex
	transactionMu sync.Mutex
	req           OpenSessionRequest
	port          port.Port
	status        string
	startedAt     time.Time
	stoppedAt     time.Time
	lastError     string
	rxBytes       int64
	txBytes       int64
	slaveRunning  bool
	slaveUnitID   byte
	slaveUnits    map[byte]*DataModel
	slaveStop     chan struct{}
	slaveDone     chan struct{}
	model         *DataModel
	nextTx        atomic.Int64
}

func newSession(req OpenSessionRequest, serialPort port.Port) *Session {
	return &Session{
		req:        req,
		port:       serialPort,
		status:     SessionStatusOpen,
		startedAt:  time.Now(),
		model:      NewDataModel(),
		slaveUnits: make(map[byte]*DataModel),
	}
}

func (s *Session) info() SessionInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	unitIDs := sortedUnitIDsAsInts(s.slaveUnits)
	return SessionInfo{
		ID:           s.req.ID,
		Name:         s.req.Name,
		Mode:         normalizeFrameMode(s.req.Mode),
		Role:         normalizeSessionRole(s.req.Role),
		Config:       s.req.Config,
		Status:       s.status,
		RxBytes:      s.rxBytes,
		TxBytes:      s.txBytes,
		SlaveRunning: s.slaveRunning,
		UnitID:       s.slaveUnitID,
		UnitIDs:      unitIDs,
		StartedAt:    s.startedAt,
		StoppedAt:    s.stoppedAt,
		LastError:    s.lastError,
	}
}

func (s *Session) close(reason string) error {
	_ = s.stopSlave()
	s.mu.Lock()
	if reason != "" {
		s.status = SessionStatusError
		s.lastError = reason
	} else {
		s.status = SessionStatusClosed
	}
	s.stoppedAt = time.Now()
	serialPort := s.port
	s.port = nil
	s.mu.Unlock()
	if serialPort != nil {
		return serialPort.Close()
	}
	return nil
}

func (s *Session) masterRequest(req MasterRequest) (*Transaction, error) {
	if req.UnitID == 0 {
		return nil, fmt.Errorf("unit id must be 1-247")
	}
	s.transactionMu.Lock()
	defer s.transactionMu.Unlock()

	s.mu.RLock()
	if s.port == nil || s.status == SessionStatusClosed {
		s.mu.RUnlock()
		return nil, fmt.Errorf("modbus session is closed")
	}
	if s.slaveRunning {
		s.mu.RUnlock()
		return nil, fmt.Errorf("cannot run master transaction while slave simulation is running")
	}
	mode := normalizeFrameMode(s.req.Mode)
	timeout := requestTimeout(req.TimeoutMs, s.req.TimeoutMs)
	retries := requestRetries(req.Retries, s.req.Retries)
	s.mu.RUnlock()

	startedAt := time.Now()
	tx := &Transaction{
		ID:        fmt.Sprintf("%s-tx-%d", req.SessionID, s.nextTx.Add(1)),
		SessionID: req.SessionID,
		StartedAt: startedAt,
		UnitID:    req.UnitID,
		Mode:      mode,
	}

	pdu, err := BuildMasterPDU(req)
	if err != nil {
		tx.Error = err.Error()
		tx.CompletedAt = time.Now()
		return tx, err
	}
	tx.RequestPDU = pdu
	frame, err := EncodeFrame(mode, req.UnitID, pdu)
	if err != nil {
		tx.Error = err.Error()
		tx.CompletedAt = time.Now()
		return tx, err
	}
	tx.RequestFrameHex = FormatHex(frame)

	var lastErr error
	for attempt := 0; attempt <= retries; attempt++ {
		written, err := s.write(frame)
		tx.BytesWritten += written
		if err != nil {
			lastErr = err
			continue
		}
		responseFrame, decoded, err := s.readFrame(mode, timeout, nil)
		if err != nil {
			lastErr = err
			continue
		}
		if decoded.UnitID != req.UnitID {
			lastErr = fmt.Errorf("unexpected unit id %d", decoded.UnitID)
			continue
		}
		parsed, err := ParseResponse(decoded.PDU)
		if err != nil {
			lastErr = err
			continue
		}
		tx.ResponsePDU = decoded.PDU
		tx.Response = parsed
		tx.ResponseFrameHex = FormatHex(responseFrame)
		tx.CompletedAt = time.Now()
		return tx, nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("modbus transaction failed")
	}
	tx.Error = lastErr.Error()
	tx.CompletedAt = time.Now()
	return tx, lastErr
}

func (s *Session) write(data []byte) (int, error) {
	s.mu.RLock()
	serialPort := s.port
	s.mu.RUnlock()
	if serialPort == nil {
		return 0, fmt.Errorf("modbus session is closed")
	}
	n, err := serialPort.Write(data)
	if n > 0 {
		s.mu.Lock()
		s.txBytes += int64(n)
		s.mu.Unlock()
	}
	return n, err
}

func (s *Session) readFrame(mode FrameMode, timeout time.Duration, stop <-chan struct{}) ([]byte, DecodedFrame, error) {
	deadline := time.Now().Add(timeout)
	var buf []byte
	tmp := make([]byte, 256)
	for {
		if stop != nil {
			select {
			case <-stop:
				return nil, DecodedFrame{}, fmt.Errorf("modbus read stopped")
			default:
			}
		}
		if time.Now().After(deadline) {
			return nil, DecodedFrame{}, fmt.Errorf("modbus response timeout")
		}
		s.mu.RLock()
		serialPort := s.port
		s.mu.RUnlock()
		if serialPort == nil {
			return nil, DecodedFrame{}, fmt.Errorf("modbus session is closed")
		}
		n, err := serialPort.Read(tmp)
		if err != nil {
			return nil, DecodedFrame{}, err
		}
		if n == 0 {
			continue
		}
		s.mu.Lock()
		s.rxBytes += int64(n)
		s.mu.Unlock()
		buf = append(buf, tmp[:n]...)
		if len(buf) > maxFrameBytes {
			return nil, DecodedFrame{}, fmt.Errorf("modbus frame too long")
		}
		if mode == FrameModeASCII {
			if idx := strings.Index(string(buf), "\r\n"); idx >= 0 {
				frame := append([]byte(nil), buf[:idx+2]...)
				decoded, err := DecodeFrame(mode, frame)
				return frame, decoded, err
			}
			continue
		}
		decoded, err := DecodeFrame(mode, buf)
		if err == nil {
			return append([]byte(nil), buf...), decoded, nil
		}
	}
}

func (s *Session) startSlave(req StartSlaveRequest) error {
	units, err := normalizeSlaveUnits(req)
	if err != nil {
		return err
	}
	s.transactionMu.Lock()
	defer s.transactionMu.Unlock()

	s.mu.Lock()
	if s.port == nil || s.status == SessionStatusClosed {
		s.mu.Unlock()
		return fmt.Errorf("modbus session is closed")
	}
	if s.slaveRunning {
		s.mu.Unlock()
		return fmt.Errorf("modbus slave already running")
	}
	s.slaveUnits = units
	s.slaveUnitID = firstUnitID(units)
	if model := units[s.slaveUnitID]; model != nil {
		s.model = model.Clone()
	}
	s.slaveRunning = true
	s.status = SessionStatusRunning
	stop := make(chan struct{})
	done := make(chan struct{})
	s.slaveStop = stop
	s.slaveDone = done
	mode := normalizeFrameMode(s.req.Mode)
	s.mu.Unlock()

	go s.slaveLoop(mode, stop, done)
	return nil
}

func (s *Session) stopSlave() error {
	s.mu.Lock()
	if !s.slaveRunning {
		s.mu.Unlock()
		return nil
	}
	stop := s.slaveStop
	done := s.slaveDone
	s.slaveRunning = false
	s.status = SessionStatusOpen
	s.slaveStop = nil
	s.slaveDone = nil
	s.mu.Unlock()

	close(stop)
	select {
	case <-done:
	case <-time.After(time.Second):
		return fmt.Errorf("timeout stopping modbus slave")
	}
	return nil
}

func (s *Session) slaveLoop(mode FrameMode, stop <-chan struct{}, done chan<- struct{}) {
	defer close(done)
	for {
		frame, decoded, err := s.readFrame(mode, 200*time.Millisecond, stop)
		_ = frame
		if err != nil {
			select {
			case <-stop:
				return
			default:
				if strings.Contains(err.Error(), "timeout") {
					continue
				}
				s.setError(err.Error())
				return
			}
		}
		s.mu.RLock()
		model, ok := s.slaveUnits[decoded.UnitID]
		if ok {
			model = model.Clone()
		}
		s.mu.RUnlock()
		var response SlaveResponse
		if ok {
			response = BuildSlaveResponse(decoded.UnitID, decoded.PDU, model)
			s.setUnitDataModel(decoded.UnitID, model)
		} else {
			response = exceptionResponse(decoded.PDU.Function, ExceptionServerFailure)
		}
		responseFrame, err := EncodeFrame(mode, decoded.UnitID, response.PDU)
		if err != nil {
			s.setError(err.Error())
			return
		}
		if _, err := s.write(responseFrame); err != nil {
			s.setError(err.Error())
			return
		}
	}
}

func (s *Session) setError(message string) {
	s.mu.Lock()
	s.status = SessionStatusError
	s.lastError = message
	s.slaveRunning = false
	s.mu.Unlock()
}

func (s *Session) setDataModel(model *DataModel) {
	s.mu.Lock()
	if s.slaveUnitID == 0 {
		s.slaveUnitID = 1
	}
	if s.slaveUnits == nil {
		s.slaveUnits = make(map[byte]*DataModel)
	}
	cloned := model.Clone()
	s.slaveUnits[s.slaveUnitID] = cloned
	s.model = cloned.Clone()
	s.mu.Unlock()
}

func (s *Session) setUnitDataModel(unitID byte, model *DataModel) {
	s.mu.Lock()
	if s.slaveUnits == nil {
		s.slaveUnits = make(map[byte]*DataModel)
	}
	cloned := model.Clone()
	s.slaveUnits[unitID] = cloned
	if s.slaveUnitID == unitID {
		s.model = cloned.Clone()
	}
	s.mu.Unlock()
}

func (s *Session) addSlaveUnit(unit SlaveUnitSnapshot) error {
	if err := validateUnitID(unit.UnitID); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.slaveUnits == nil {
		s.slaveUnits = make(map[byte]*DataModel)
	}
	model := DataModelFromSnapshot(unit.DataModel)
	s.slaveUnits[unit.UnitID] = model
	if s.slaveUnitID == 0 {
		s.slaveUnitID = unit.UnitID
		s.model = model.Clone()
	}
	return nil
}

func (s *Session) removeSlaveUnit(unitID byte) error {
	if err := validateUnitID(unitID); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.slaveUnits, unitID)
	if s.slaveUnitID == unitID {
		s.slaveUnitID = firstUnitID(s.slaveUnits)
		if model := s.slaveUnits[s.slaveUnitID]; model != nil {
			s.model = model.Clone()
		} else {
			s.model = NewDataModel()
		}
	}
	return nil
}

func (s *Session) listSlaveUnits() []SlaveUnitInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ids := sortedUnitIDs(s.slaveUnits)
	result := make([]SlaveUnitInfo, 0, len(ids))
	for _, id := range ids {
		result = append(result, SlaveUnitInfo{
			UnitID:    id,
			DataModel: SnapshotFromDataModel(s.slaveUnits[id]),
		})
	}
	return result
}

func (s *Session) updateSlaveUnitData(unitID byte, snapshot DataModelSnapshot) error {
	if err := validateUnitID(unitID); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.slaveUnits == nil {
		s.slaveUnits = make(map[byte]*DataModel)
	}
	model := DataModelFromSnapshot(snapshot)
	s.slaveUnits[unitID] = model
	if s.slaveUnitID == unitID {
		s.model = model.Clone()
	}
	return nil
}

func (s *Session) scanUnitIDs(req UnitScanRequest) (*UnitScanResult, error) {
	if len(req.UnitIDs) == 0 {
		req.UnitIDs = make([]int, 247)
		for i := range req.UnitIDs {
			req.UnitIDs[i] = i + 1
		}
	}
	if req.Function == 0 {
		req.Function = FunctionReadHoldingRegisters
	}
	if req.Quantity == 0 {
		req.Quantity = 1
	}
	result := &UnitScanResult{
		SessionID: req.SessionID,
		Results:   make([]UnitScanEntry, 0, len(req.UnitIDs)),
	}
	for _, rawUnitID := range req.UnitIDs {
		unitID, err := unitIDByte(rawUnitID)
		if err != nil {
			result.Results = append(result.Results, UnitScanEntry{UnitID: byte(rawUnitID), Error: err.Error()})
			continue
		}
		tx, err := s.masterRequest(MasterRequest{
			SessionID:   req.SessionID,
			UnitID:      unitID,
			Function:    req.Function,
			AddressMode: req.AddressMode,
			Address:     req.Address,
			Quantity:    req.Quantity,
			TimeoutMs:   req.TimeoutMs,
			Retries:     req.Retries,
		})
		if err != nil {
			result.Results = append(result.Results, UnitScanEntry{UnitID: unitID, Error: err.Error()})
			continue
		}
		active := tx != nil && tx.Response.Function != 0
		if active {
			result.ActiveUnitIDs = append(result.ActiveUnitIDs, int(unitID))
		}
		result.Results = append(result.Results, UnitScanEntry{UnitID: unitID, Active: active})
	}
	return result, nil
}

func (s *Session) readRegisters(req RegisterReadRequest) (*RegisterReadResult, error) {
	if req.Function == 0 {
		req.Function = FunctionReadHoldingRegisters
	}
	if req.Quantity == 0 {
		req.Quantity = 1
	}
	tx, err := s.masterRequest(MasterRequest{
		SessionID:   req.SessionID,
		UnitID:      req.UnitID,
		Function:    req.Function,
		AddressMode: req.AddressMode,
		Address:     req.Address,
		Quantity:    req.Quantity,
		TimeoutMs:   req.TimeoutMs,
		Retries:     req.Retries,
	})
	if err != nil {
		return nil, err
	}
	result := &RegisterReadResult{
		Transaction: tx,
	}
	if tx.Response.Exception {
		return result, nil
	}
	result.RawRegisters = append([]uint16(nil), tx.Response.Values...)
	result.Bits = append([]bool(nil), tx.Response.Bits...)
	if len(req.Mappings) == 0 || len(result.RawRegisters) == 0 {
		return result, nil
	}
	for _, mapping := range req.Mappings {
		if mapping.Address < req.Address {
			continue
		}
		offset := int(mapping.Address - req.Address)
		if offset >= len(result.RawRegisters) {
			continue
		}
		spec := RegisterValueSpec{
			DataType:      mapping.DataType,
			WordOrder:     mapping.WordOrder,
			Length:        mapping.Length,
			ScalingFactor: mapping.ScalingFactor,
			Interpolate:   mapping.Interpolate,
		}
		decoded, decodeErr := DecodeRegisterValue(spec, result.RawRegisters[offset:])
		entry := MappedRegisterValue{
			Address: mapping.Address,
			Mapping: mapping,
			Value:   decoded,
		}
		if decodeErr != nil {
			entry.Error = decodeErr.Error()
		}
		result.Values = append(result.Values, entry)
	}
	return result, nil
}

func (s *Session) scanRegisters(req RegisterScanRequest) (*RegisterScanResult, error) {
	if req.Function == 0 {
		req.Function = FunctionReadHoldingRegisters
	}
	if req.Function != FunctionReadHoldingRegisters && req.Function != FunctionReadInputRegisters {
		return nil, fmt.Errorf("register scan requires function 03 or 04")
	}
	if req.EndAddress < req.StartAddress {
		return nil, fmt.Errorf("end address must be greater than or equal to start address")
	}
	if req.ChunkSize == 0 {
		req.ChunkSize = 16
	}
	if req.ChunkSize > 125 {
		req.ChunkSize = 125
	}
	result := &RegisterScanResult{
		SessionID: req.SessionID,
		UnitID:    req.UnitID,
	}
	for address := req.StartAddress; ; {
		remaining := uint32(req.EndAddress) - uint32(address) + 1
		quantity := req.ChunkSize
		if uint32(quantity) > remaining {
			quantity = uint16(remaining)
		}
		read, err := s.readRegisters(RegisterReadRequest{
			SessionID:   req.SessionID,
			UnitID:      req.UnitID,
			Function:    req.Function,
			AddressMode: req.AddressMode,
			Address:     address,
			Quantity:    quantity,
			TimeoutMs:   req.TimeoutMs,
			Retries:     req.Retries,
		})
		rangeInfo := RegisterScanRange{Address: address, Quantity: quantity}
		if err != nil {
			rangeInfo.Error = err.Error()
			result.Ranges = append(result.Ranges, rangeInfo)
		} else {
			result.Ranges = append(result.Ranges, rangeInfo)
			for i, value := range read.RawRegisters {
				if value != 0 {
					result.Values = append(result.Values, RegisterScanValue{
						Address: address + uint16(i),
						Value:   value,
					})
				}
			}
		}
		if address+quantity == 0 || address+quantity > req.EndAddress {
			break
		}
		address += quantity
	}
	return result, nil
}

// BuildMasterPDU builds a master request PDU from frontend-friendly fields.
func BuildMasterPDU(req MasterRequest) (PDU, error) {
	address, err := ProtocolAddress(req.AddressMode, req.Function, req.Address)
	if err != nil {
		return PDU{}, err
	}
	switch req.Function {
	case FunctionReadCoils, FunctionReadDiscreteInputs, FunctionReadHoldingRegisters, FunctionReadInputRegisters:
		return BuildReadRequest(req.Function, address, req.Quantity)
	case FunctionWriteSingleCoil, FunctionWriteSingleRegister:
		return BuildWriteSingleRequest(req.Function, address, req.Value)
	case FunctionWriteMultipleCoils:
		return BuildWriteMultipleCoilsRequest(address, req.CoilValues)
	case FunctionWriteMultipleRegisters:
		return BuildWriteMultipleRegistersRequest(address, req.RegisterValues)
	default:
		return PDU{}, fmt.Errorf("unsupported function: %02x", byte(req.Function))
	}
}

// FormatHex returns lower-case space separated bytes.
func FormatHex(data []byte) string {
	encoded := hex.EncodeToString(data)
	if encoded == "" {
		return ""
	}
	parts := make([]string, 0, len(encoded)/2)
	for i := 0; i < len(encoded); i += 2 {
		parts = append(parts, encoded[i:i+2])
	}
	return strings.Join(parts, " ")
}

// DataModelFromSnapshot converts serializable data into a runtime model.
func DataModelFromSnapshot(snapshot DataModelSnapshot) *DataModel {
	model := NewDataModel()
	for _, point := range snapshot.Coils {
		model.Coils[point.Address] = point.Value
	}
	for _, point := range snapshot.DiscreteInputs {
		model.DiscreteInputs[point.Address] = point.Value
	}
	for _, point := range snapshot.InputRegisters {
		model.InputRegisters[point.Address] = point.Value
	}
	for _, point := range snapshot.HoldingRegisters {
		model.HoldingRegisters[point.Address] = point.Value
	}
	return model
}

// SnapshotFromDataModel converts a runtime model into serializable data.
func SnapshotFromDataModel(model *DataModel) DataModelSnapshot {
	if model == nil {
		model = NewDataModel()
	}
	snapshot := DataModelSnapshot{}
	for address, value := range model.Coils {
		snapshot.Coils = append(snapshot.Coils, BoolPoint{Address: address, Value: value})
	}
	for address, value := range model.DiscreteInputs {
		snapshot.DiscreteInputs = append(snapshot.DiscreteInputs, BoolPoint{Address: address, Value: value})
	}
	for address, value := range model.InputRegisters {
		snapshot.InputRegisters = append(snapshot.InputRegisters, RegisterPoint{Address: address, Value: value})
	}
	for address, value := range model.HoldingRegisters {
		snapshot.HoldingRegisters = append(snapshot.HoldingRegisters, RegisterPoint{Address: address, Value: value})
	}
	return snapshot
}

func normalizeSlaveUnits(req StartSlaveRequest) (map[byte]*DataModel, error) {
	source := req.Units
	if len(source) == 0 {
		source = []SlaveUnitSnapshot{{
			UnitID:    req.UnitID,
			DataModel: req.DataModel,
		}}
	}
	units := make(map[byte]*DataModel, len(source))
	for _, unit := range source {
		if err := validateUnitID(unit.UnitID); err != nil {
			return nil, err
		}
		units[unit.UnitID] = DataModelFromSnapshot(unit.DataModel)
	}
	if len(units) == 0 {
		return nil, fmt.Errorf("at least one unit id is required")
	}
	return units, nil
}

func normalizeSessionRole(role SessionRole) SessionRole {
	if role == SessionRoleSlave {
		return SessionRoleSlave
	}
	return SessionRoleMaster
}

func validateUnitID(unitID byte) error {
	if unitID == 0 || unitID > 247 {
		return fmt.Errorf("unit id must be 1-247")
	}
	return nil
}

func unitIDByte(unitID int) (byte, error) {
	if unitID < 1 || unitID > 247 {
		return 0, fmt.Errorf("unit id must be 1-247")
	}
	return byte(unitID), nil
}

func sortedUnitIDs(units map[byte]*DataModel) []byte {
	ids := make([]byte, 0, len(units))
	for unitID := range units {
		ids = append(ids, unitID)
	}
	sort.Slice(ids, func(i, j int) bool {
		return ids[i] < ids[j]
	})
	return ids
}

func sortedUnitIDsAsInts(units map[byte]*DataModel) []int {
	byteIDs := sortedUnitIDs(units)
	ids := make([]int, 0, len(byteIDs))
	for _, id := range byteIDs {
		ids = append(ids, int(id))
	}
	return ids
}

func firstUnitID(units map[byte]*DataModel) byte {
	ids := sortedUnitIDs(units)
	if len(ids) == 0 {
		return 0
	}
	return ids[0]
}

func requestTimeout(requestMs int, sessionMs int) time.Duration {
	if requestMs > 0 {
		return time.Duration(requestMs) * time.Millisecond
	}
	if sessionMs > 0 {
		return time.Duration(sessionMs) * time.Millisecond
	}
	return defaultTimeout
}

func requestRetries(requestRetries int, sessionRetries int) int {
	if requestRetries > 0 {
		return requestRetries
	}
	if sessionRetries > 0 {
		return sessionRetries
	}
	return 0
}
