package fecbus

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/littepointR/portweave/internal/modules/serial/port"
)

const (
	defaultTimeout = time.Second
	readPoll       = 20 * time.Millisecond
	maxFrameBytes  = 256
	maxHistory     = 4096
)

// PortOpener opens a serial port. It is injectable for tests.
type PortOpener func(port.SerialConfig) (port.Port, error)

// Manager owns dedicated FECbus serial sessions.
type Manager struct {
	mu       sync.RWMutex
	sessions map[string]*Session
	open     PortOpener
	nextID   atomic.Int64
}

// NewManager constructs a FECbus session manager.
func NewManager(open PortOpener) *Manager {
	if open == nil {
		open = port.Open
	}
	return &Manager{
		sessions: make(map[string]*Session),
		open:     open,
	}
}

// OpenSession opens a serial port for FECbus-only use.
func (m *Manager) OpenSession(ctx context.Context, req OpenSessionRequest) (*SessionInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if req.Config.PortName == "" {
		return nil, fmt.Errorf("fecbus port name must not be empty")
	}
	if req.ID == "" {
		req.ID = fmt.Sprintf("fecbus-%d", m.nextID.Add(1))
	}
	if req.Name == "" {
		req.Name = fmt.Sprintf("FECbus %s", req.Config.PortName)
	}
	req.Role = normalizeSessionRole(req.Role)
	defaultSerialConfig(&req.Config)

	m.mu.Lock()
	if _, exists := m.sessions[req.ID]; exists {
		m.mu.Unlock()
		return nil, fmt.Errorf("fecbus session already exists")
	}
	if m.portInUseLocked(req.Config.PortName) {
		m.mu.Unlock()
		return nil, fmt.Errorf("fecbus port already in use")
	}
	m.mu.Unlock()

	openConfig := req.Config
	if req.Endpoint != "" {
		openConfig.PortName = req.Endpoint
	}
	serialPort, err := m.open(openConfig)
	if err != nil {
		return nil, fmt.Errorf("open fecbus port %s: %w", req.Config.PortName, err)
	}
	_ = serialPort.SetReadTimeout(readPoll)
	session := newSession(req, serialPort)

	m.mu.Lock()
	if _, exists := m.sessions[req.ID]; exists {
		m.mu.Unlock()
		_ = serialPort.Close()
		return nil, fmt.Errorf("fecbus session already exists")
	}
	if m.portInUseLocked(req.Config.PortName) {
		m.mu.Unlock()
		_ = serialPort.Close()
		return nil, fmt.Errorf("fecbus port already in use")
	}
	m.sessions[req.ID] = session
	m.mu.Unlock()

	info := session.info()
	return &info, nil
}

// CloseSession closes and removes one FECbus session.
func (m *Manager) CloseSession(id string) error {
	m.mu.Lock()
	session, ok := m.sessions[id]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("fecbus session not found")
	}
	delete(m.sessions, id)
	m.mu.Unlock()
	return session.close("")
}

// CloseAll closes every FECbus session.
func (m *Manager) CloseAll() {
	m.mu.Lock()
	sessions := m.sessions
	m.sessions = make(map[string]*Session)
	m.mu.Unlock()
	for _, session := range sessions {
		_ = session.close("")
	}
}

// List returns all FECbus sessions.
func (m *Manager) List() []SessionInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]SessionInfo, 0, len(m.sessions))
	for _, session := range m.sessions {
		result = append(result, session.info())
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

// PortInUse reports whether a public port name is owned by FECbus.
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

// SendRequest sends one FECbus frame and optionally waits for a matching answer.
func (m *Manager) SendRequest(req SendRequest) (*Transaction, error) {
	session, err := m.get(req.SessionID)
	if err != nil {
		return nil, err
	}
	return session.sendRequest(req)
}

// StartSlave starts device simulation on an open session.
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

// StopSlave stops device simulation while keeping the serial session open.
func (m *Manager) StopSlave(id string) error {
	session, err := m.get(id)
	if err != nil {
		return err
	}
	return session.stopSlave()
}

// UpdateSlaveState replaces one session's slave simulation state.
func (m *Manager) UpdateSlaveState(sessionID string, state SlaveState) error {
	session, err := m.get(sessionID)
	if err != nil {
		return err
	}
	session.setSlaveState(state)
	return nil
}

// AddSlaveUnit adds or replaces one simulated FECbus slave address.
func (m *Manager) AddSlaveUnit(sessionID string, unit SlaveUnitState) error {
	session, err := m.get(sessionID)
	if err != nil {
		return err
	}
	session.setSlaveUnit(unit)
	return nil
}

// RemoveSlaveUnit removes one simulated FECbus slave address.
func (m *Manager) RemoveSlaveUnit(sessionID string, address byte) error {
	session, err := m.get(sessionID)
	if err != nil {
		return err
	}
	return session.removeSlaveUnit(address)
}

// ListSlaveUnits returns configured simulated FECbus slave addresses.
func (m *Manager) ListSlaveUnits(sessionID string) ([]SlaveUnitInfo, error) {
	session, err := m.get(sessionID)
	if err != nil {
		return nil, err
	}
	return session.listSlaveUnits(), nil
}

// QueryFrames returns a filtered frame history page.
func (m *Manager) QueryFrames(req QueryRequest) (*FramePage, error) {
	session, err := m.get(req.SessionID)
	if err != nil {
		return nil, err
	}
	page := session.queryFrames(req)
	return &page, nil
}

// ClearFrames clears frame history and counters for a session.
func (m *Manager) ClearFrames(id string) error {
	session, err := m.get(id)
	if err != nil {
		return err
	}
	session.clearFrames()
	return nil
}

func (m *Manager) get(id string) (*Session, error) {
	if id == "" {
		return nil, fmt.Errorf("fecbus session id must not be empty")
	}
	m.mu.RLock()
	session, ok := m.sessions[id]
	m.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("fecbus session not found")
	}
	return session, nil
}

// Session is one open FECbus serial port.
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
	slaveUnits    map[byte]SlaveUnitState
	slaveStop     chan struct{}
	slaveDone     chan struct{}
	history       []FrameRecord
	nextSeq       atomic.Int64
	nextTx        atomic.Int64
}

func newSession(req OpenSessionRequest, serialPort port.Port) *Session {
	return &Session{
		req:       req,
		port:      serialPort,
		status:    SessionStatusOpen,
		startedAt: time.Now(),
		slaveUnits: map[byte]SlaveUnitState{
			2: {
				Address:          2,
				DefaultStatus:    StatusReceivedOK,
				AutoStatusAnswer: true,
				AcceptBroadcast:  true,
			},
		},
	}
}

func (s *Session) info() SessionInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return SessionInfo{
		ID:            s.req.ID,
		Name:          s.req.Name,
		Role:          normalizeSessionRole(s.req.Role),
		Config:        s.req.Config,
		Status:        s.status,
		RxBytes:       s.rxBytes,
		TxBytes:       s.txBytes,
		SlaveRunning:  s.slaveRunning,
		SourceAddress: defaultSourceAddress(s.req.Role),
		TargetAddress: defaultTargetAddress(s.slaveUnits),
		SlaveUnits:    slaveUnitInfos(s.slaveUnits),
		StartedAt:     s.startedAt,
		StoppedAt:     s.stoppedAt,
		LastError:     s.lastError,
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

func (s *Session) sendRequest(req SendRequest) (*Transaction, error) {
	s.transactionMu.Lock()
	defer s.transactionMu.Unlock()

	s.mu.RLock()
	if s.port == nil || s.status == SessionStatusClosed {
		s.mu.RUnlock()
		return nil, fmt.Errorf("fecbus session is closed")
	}
	if s.slaveRunning {
		s.mu.RUnlock()
		return nil, fmt.Errorf("cannot send fecbus request while slave simulation is running")
	}
	timeout := requestTimeout(req.TimeoutMs, s.req.TimeoutMs)
	retries := requestRetries(req.Retries, s.req.Retries)
	s.mu.RUnlock()

	startedAt := time.Now()
	tx := &Transaction{
		ID:        fmt.Sprintf("%s-tx-%d", req.SessionID, s.nextTx.Add(1)),
		SessionID: req.SessionID,
		StartedAt: startedAt,
	}

	frame, err := requestFrame(req)
	if err != nil {
		tx.Error = err.Error()
		tx.CompletedAt = time.Now()
		return tx, err
	}
	tx.Request = frame
	raw, err := EncodeFrame(frame)
	if err != nil {
		tx.Error = err.Error()
		tx.CompletedAt = time.Now()
		return tx, err
	}
	tx.RequestFrameHex = FormatHex(raw)

	var lastErr error
	for attempt := 0; attempt <= retries; attempt++ {
		written, err := s.write(raw)
		tx.BytesWritten += written
		s.recordFrame("tx", frame, raw, "")
		if err != nil {
			lastErr = err
			continue
		}
		if !req.ExpectAnswer {
			tx.CompletedAt = time.Now()
			return tx, nil
		}
		responseRaw, response, err := s.readFrame(timeout, nil)
		if err != nil {
			lastErr = err
			continue
		}
		s.recordFrame("rx", response, responseRaw, "")
		if !matchesAnswer(frame, response) {
			lastErr = fmt.Errorf("unexpected fecbus answer")
			continue
		}
		tx.Response = &response
		tx.ResponseFrameHex = FormatHex(responseRaw)
		tx.CompletedAt = time.Now()
		return tx, nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("fecbus transaction failed")
	}
	tx.Error = lastErr.Error()
	tx.CompletedAt = time.Now()
	return tx, lastErr
}

func requestFrame(req SendRequest) (Frame, error) {
	data := append([]byte(nil), req.Data...)
	if req.DataHex != "" {
		parsed, err := ParseHex(req.DataHex)
		if err != nil {
			return Frame{}, fmt.Errorf("parse data hex: %w", err)
		}
		data = parsed
	}
	if len(data) == 0 {
		var err error
		data, err = BuildData(req.Function, nil)
		if err != nil {
			return Frame{}, err
		}
	}
	if req.FrameType != FrameTypeRequest && req.FrameType != FrameTypeAnswer {
		req.FrameType = FrameTypeRequest
	}
	if req.SourceAddress == 0 {
		req.SourceAddress = 1
	}
	if req.TargetAddress == 0 && req.FrameType == FrameTypeAnswer {
		req.TargetAddress = 1
	}
	if req.MessageNumber == 0 {
		req.MessageNumber = 1
	}
	return Frame{
		Type:          req.FrameType,
		TargetAddress: req.TargetAddress,
		Priority:      req.Priority,
		SourceAddress: req.SourceAddress,
		MessageNumber: req.MessageNumber,
		GroupNumber:   req.GroupNumber,
		Data:          data,
	}, nil
}

func (s *Session) write(data []byte) (int, error) {
	s.mu.RLock()
	serialPort := s.port
	s.mu.RUnlock()
	if serialPort == nil {
		return 0, fmt.Errorf("fecbus session is closed")
	}
	n, err := serialPort.Write(data)
	if n > 0 {
		s.mu.Lock()
		s.txBytes += int64(n)
		s.mu.Unlock()
	}
	return n, err
}

func (s *Session) readFrame(timeout time.Duration, stop <-chan struct{}) ([]byte, Frame, error) {
	deadline := time.Now().Add(timeout)
	var buf []byte
	tmp := make([]byte, 64)
	for {
		if stop != nil {
			select {
			case <-stop:
				return nil, Frame{}, fmt.Errorf("fecbus read stopped")
			default:
			}
		}
		if time.Now().After(deadline) {
			return nil, Frame{}, fmt.Errorf("fecbus response timeout")
		}
		s.mu.RLock()
		serialPort := s.port
		s.mu.RUnlock()
		if serialPort == nil {
			return nil, Frame{}, fmt.Errorf("fecbus session is closed")
		}
		n, err := serialPort.Read(tmp)
		if err != nil {
			return nil, Frame{}, err
		}
		if n == 0 {
			continue
		}
		s.mu.Lock()
		s.rxBytes += int64(n)
		s.mu.Unlock()
		buf = append(buf, tmp[:n]...)
		if len(buf) > maxFrameBytes {
			return nil, Frame{}, fmt.Errorf("fecbus frame too long")
		}
		raw, rest, ok, err := ExtractFrame(buf)
		if err != nil {
			buf = rest
			continue
		}
		if !ok {
			continue
		}
		decoded, err := DecodeFrame(raw)
		if err != nil {
			buf = rest
			continue
		}
		return raw, decoded, nil
	}
}

func (s *Session) startSlave(req StartSlaveRequest) error {
	units := normalizeSlaveUnits(req)

	s.transactionMu.Lock()
	defer s.transactionMu.Unlock()

	s.mu.Lock()
	if s.port == nil || s.status == SessionStatusClosed {
		s.mu.Unlock()
		return fmt.Errorf("fecbus session is closed")
	}
	if s.slaveRunning {
		s.mu.Unlock()
		return fmt.Errorf("fecbus slave already running")
	}
	s.slaveUnits = units
	s.slaveRunning = true
	s.status = SessionStatusRunning
	stop := make(chan struct{})
	done := make(chan struct{})
	s.slaveStop = stop
	s.slaveDone = done
	s.mu.Unlock()

	go s.slaveLoop(stop, done)
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
		return fmt.Errorf("timeout stopping fecbus slave")
	}
	return nil
}

func (s *Session) slaveLoop(stop <-chan struct{}, done chan<- struct{}) {
	defer close(done)
	for {
		raw, frame, err := s.readFrame(200*time.Millisecond, stop)
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
		s.recordFrame("rx", frame, raw, "")
		s.mu.RLock()
		units := cloneSlaveUnits(s.slaveUnits)
		s.mu.RUnlock()
		matched := matchingSlaveUnits(frame, units)
		if len(matched) == 0 {
			continue
		}
		for _, state := range matched {
			answer, ok, err := buildSlaveAnswer(frame, state)
			if err != nil {
				s.recordFrame("tx", Frame{}, nil, err.Error())
				continue
			}
			if !ok {
				continue
			}
			answerRaw, err := EncodeFrame(answer)
			if err != nil {
				s.setError(err.Error())
				return
			}
			if _, err := s.write(answerRaw); err != nil {
				s.setError(err.Error())
				return
			}
			s.recordFrame("tx", answer, answerRaw, "")
		}
	}
}

func buildSlaveAnswer(request Frame, state SlaveUnitState) (Frame, bool, error) {
	if payload, ok := state.AnswerPayloadByFunction[request.Function()]; ok {
		data, err := BuildData(request.Function(), payload)
		if err != nil {
			return Frame{}, false, err
		}
		return Frame{
			Type:          FrameTypeAnswer,
			TargetAddress: request.SourceAddress,
			Priority:      3,
			SourceAddress: state.Address,
			MessageNumber: request.MessageNumber,
			GroupNumber:   request.GroupNumber,
			Data:          data,
		}, true, nil
	}
	if !state.AutoStatusAnswer {
		return Frame{}, false, nil
	}
	answer, err := BuildStatusAnswer(request, state.DefaultStatus)
	answer.SourceAddress = state.Address
	return answer, true, err
}

func (s *Session) setError(message string) {
	s.mu.Lock()
	s.status = SessionStatusError
	s.lastError = message
	s.slaveRunning = false
	s.mu.Unlock()
}

func (s *Session) setSlaveState(state SlaveState) {
	s.setSlaveUnit(SlaveUnitState(state))
}

func (s *Session) setSlaveUnit(unit SlaveUnitState) {
	s.mu.Lock()
	next := normalizeSlaveUnit(unit)
	if s.slaveUnits == nil {
		s.slaveUnits = make(map[byte]SlaveUnitState)
	}
	s.slaveUnits[next.Address] = next
	s.mu.Unlock()
}

func (s *Session) removeSlaveUnit(address byte) error {
	if address == 0 || address > 63 {
		return fmt.Errorf("fecbus slave address must be 1-63")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.slaveUnits[address]; !ok {
		return fmt.Errorf("fecbus slave unit not found")
	}
	delete(s.slaveUnits, address)
	return nil
}

func (s *Session) listSlaveUnits() []SlaveUnitInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return slaveUnitInfos(s.slaveUnits)
}

func (s *Session) recordFrame(direction string, frame Frame, raw []byte, message string) {
	now := time.Now()
	if frame.Timestamp.IsZero() {
		frame.Timestamp = now
	}
	if len(raw) > 0 && len(frame.Raw) == 0 {
		frame.Raw = append([]byte(nil), raw...)
	}
	record := FrameRecord{
		Seq:       s.nextSeq.Add(1),
		SessionID: s.req.ID,
		Direction: direction,
		Frame:     frame,
		Hex:       FormatHex(raw),
		Error:     message,
		Timestamp: now,
	}
	record.Annotated = AnnotateFrame(record, nil, nil)
	s.mu.Lock()
	s.history = append(s.history, record)
	if len(s.history) > maxHistory {
		copy(s.history, s.history[len(s.history)-maxHistory:])
		s.history = s.history[:maxHistory]
	}
	s.mu.Unlock()
}

func (s *Session) queryFrames(req QueryRequest) FramePage {
	s.mu.RLock()
	history := append([]FrameRecord(nil), s.history...)
	s.mu.RUnlock()

	search := strings.ToLower(strings.TrimSpace(req.Search))
	filtered := make([]FrameRecord, 0, len(history))
	groupFunctions := make(map[string]FunctionCode)
	for _, record := range history {
		groupKey := multiFrameGroupKey(record)
		var context *AnnotationContext
		if record.Frame.GroupNumber > 1 {
			if function, ok := groupFunctions[groupKey]; ok {
				context = &AnnotationContext{Function: function}
			}
		}
		record.Annotated = AnnotateFrame(record, req.Custom, context)
		if groupKey != "" && record.Frame.GroupNumber == 1 {
			groupFunctions[groupKey] = record.Annotated.Function.Code
		}
		if req.Direction != "" && record.Direction != req.Direction {
			continue
		}
		if search != "" {
			info, _ := LookupFunction(record.Frame.Function())
			haystack := strings.ToLower(record.Hex + " " + record.Error + " " + fmt.Sprintf("%d %s", record.Frame.Function(), info.Name))
			if !strings.Contains(haystack, search) {
				continue
			}
		}
		filtered = append(filtered, record)
	}
	total := int64(len(filtered))
	offset := req.Offset
	if offset < 0 {
		offset = 0
	}
	limit := req.Limit
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	if offset >= total {
		return FramePage{Frames: []FrameRecord{}, Offset: offset, Limit: limit, Total: total, EOF: true}
	}
	end := offset + int64(limit)
	if end > total {
		end = total
	}
	return FramePage{
		Frames: filtered[offset:end],
		Offset: offset,
		Limit:  limit,
		Total:  total,
		EOF:    end >= total,
	}
}

func (s *Session) clearFrames() {
	s.mu.Lock()
	s.history = nil
	s.rxBytes = 0
	s.txBytes = 0
	s.mu.Unlock()
}

func matchesAnswer(request Frame, response Frame) bool {
	return response.Type == FrameTypeAnswer &&
		response.TargetAddress == request.SourceAddress &&
		response.SourceAddress == request.TargetAddress &&
		response.MessageNumber == request.MessageNumber
}

func cloneSlaveState(state SlaveState) SlaveState {
	out := state
	if out.AnswerPayloadByFunction != nil {
		out.AnswerPayloadByFunction = make(map[FunctionCode][]byte, len(state.AnswerPayloadByFunction))
		for code, payload := range state.AnswerPayloadByFunction {
			out.AnswerPayloadByFunction[code] = append([]byte(nil), payload...)
		}
	}
	return out
}

func normalizeSlaveUnits(req StartSlaveRequest) map[byte]SlaveUnitState {
	if len(req.Units) == 0 {
		unit := normalizeSlaveUnit(SlaveUnitState(req.State))
		return map[byte]SlaveUnitState{unit.Address: unit}
	}
	units := make(map[byte]SlaveUnitState, len(req.Units))
	for _, unit := range req.Units {
		next := normalizeSlaveUnit(unit)
		units[next.Address] = next
	}
	return units
}

func normalizeSlaveUnit(unit SlaveUnitState) SlaveUnitState {
	if unit.Address == 0 {
		unit.Address = 2
	}
	if unit.DefaultStatus == 0 {
		unit.DefaultStatus = StatusReceivedOK
	}
	if unit.AnswerPayloadByFunction == nil {
		unit.AnswerPayloadByFunction = make(map[FunctionCode][]byte)
	}
	if !unit.AutoStatusAnswer && len(unit.AnswerPayloadByFunction) == 0 {
		unit.AutoStatusAnswer = true
	}
	return unit
}

func cloneSlaveUnits(units map[byte]SlaveUnitState) map[byte]SlaveUnitState {
	out := make(map[byte]SlaveUnitState, len(units))
	for address, unit := range units {
		next := unit
		if next.AnswerPayloadByFunction != nil {
			next.AnswerPayloadByFunction = make(map[FunctionCode][]byte, len(unit.AnswerPayloadByFunction))
			for code, payload := range unit.AnswerPayloadByFunction {
				next.AnswerPayloadByFunction[code] = append([]byte(nil), payload...)
			}
		}
		out[address] = next
	}
	return out
}

func matchingSlaveUnits(frame Frame, units map[byte]SlaveUnitState) []SlaveUnitState {
	if frame.TargetAddress == 0 {
		addresses := sortedSlaveAddresses(units)
		matched := make([]SlaveUnitState, 0, len(addresses))
		for _, address := range addresses {
			unit := units[byte(address)]
			if unit.AcceptBroadcast {
				matched = append(matched, unit)
			}
		}
		return matched
	}
	if unit, ok := units[frame.TargetAddress]; ok {
		return []SlaveUnitState{unit}
	}
	return nil
}

func slaveUnitInfos(units map[byte]SlaveUnitState) []SlaveUnitInfo {
	addresses := sortedSlaveAddresses(units)
	out := make([]SlaveUnitInfo, 0, len(addresses))
	for _, address := range addresses {
		unit := units[byte(address)]
		out = append(out, SlaveUnitInfo{
			Address:          unit.Address,
			DefaultStatus:    unit.DefaultStatus,
			AutoStatusAnswer: unit.AutoStatusAnswer,
			AcceptBroadcast:  unit.AcceptBroadcast,
		})
	}
	return out
}

func defaultTargetAddress(units map[byte]SlaveUnitState) byte {
	addresses := sortedSlaveAddresses(units)
	if len(addresses) == 0 {
		return 2
	}
	return byte(addresses[0])
}

func sortedSlaveAddresses(units map[byte]SlaveUnitState) []int {
	addresses := make([]int, 0, len(units))
	for address := range units {
		addresses = append(addresses, int(address))
	}
	sort.Ints(addresses)
	return addresses
}

func normalizeSessionRole(role SessionRole) SessionRole {
	if role == SessionRoleSlave {
		return SessionRoleSlave
	}
	return SessionRoleMaster
}

func defaultSourceAddress(role SessionRole) byte {
	if normalizeSessionRole(role) == SessionRoleSlave {
		return 2
	}
	return 1
}

func defaultSerialConfig(cfg *port.SerialConfig) {
	if cfg.BaudRate <= 0 {
		cfg.BaudRate = 9600
	}
	if cfg.DataBits == 0 {
		cfg.DataBits = 8
	}
	if cfg.StopBits == "" {
		cfg.StopBits = "1"
	}
	if cfg.Parity == "" {
		cfg.Parity = "none"
	}
	if cfg.FlowMode == "" {
		cfg.FlowMode = "none"
	}
	if cfg.ReadBufKB == 0 {
		cfg.ReadBufKB = 32
	}
}

func requestTimeout(local int, session int) time.Duration {
	if local > 0 {
		return time.Duration(local) * time.Millisecond
	}
	if session > 0 {
		return time.Duration(session) * time.Millisecond
	}
	return defaultTimeout
}

func requestRetries(local int, session int) int {
	if local > 0 {
		return local
	}
	if session > 0 {
		return session
	}
	return 0
}
