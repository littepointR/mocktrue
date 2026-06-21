package monitor

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/littepointR/mocktrue/internal/core/errors"
	"github.com/littepointR/mocktrue/internal/modules/serial/port"
)

const defaultFrameLimit = 100000

// Manager owns serial monitor sessions.
type Manager struct {
	mu       sync.RWMutex
	sessions map[string]*Session
}

// NewManager constructs an empty monitor manager.
func NewManager() *Manager {
	return &Manager{sessions: make(map[string]*Session)}
}

// Start starts a bridge-based monitor session.
func (m *Manager) Start(ctx context.Context, req StartRequest) (*SessionInfo, error) {
	if err := validateStart(req); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	m.mu.Lock()
	if _, exists := m.sessions[req.ID]; exists {
		m.mu.Unlock()
		return nil, errors.New(errors.CodeConflict, "monitor ID already exists")
	}
	if m.portInUseLocked(req.PortA) || m.portInUseLocked(req.PortB) {
		m.mu.Unlock()
		return nil, errors.New(errors.CodeConflict, "monitor port already in use")
	}
	session := newSession(req)
	m.sessions[req.ID] = session
	m.mu.Unlock()

	if err := session.start(ctx); err != nil {
		m.mu.Lock()
		delete(m.sessions, req.ID)
		m.mu.Unlock()
		return nil, err
	}
	info := session.info()
	return &info, nil
}

func validateStart(req StartRequest) error {
	if req.ID == "" {
		return errors.New(errors.CodeInvalid, "monitor ID must not be empty")
	}
	if req.PortA == "" || req.PortB == "" {
		return errors.New(errors.CodeInvalid, "monitor ports must not be empty")
	}
	if req.PortA == req.PortB {
		return errors.New(errors.CodeInvalid, "cannot monitor a port against itself")
	}
	provider := req.Provider
	if provider == "" {
		provider = ProviderBridge
	}
	if provider != ProviderBridge {
		return errors.New(errors.CodeInvalid, "only bridge monitor provider is available")
	}
	return nil
}

func (m *Manager) portInUseLocked(portName string) bool {
	for _, session := range m.sessions {
		info := session.info()
		if info.Status == StatusRunning && (info.PortA == portName || info.PortB == portName) {
			return true
		}
	}
	return false
}

// Stop stops a monitor session but keeps captured frames available.
func (m *Manager) Stop(id string) error {
	session, err := m.get(id)
	if err != nil {
		return err
	}
	return session.stop("")
}

// Delete stops and removes a session.
func (m *Manager) Delete(id string) error {
	m.mu.Lock()
	session, ok := m.sessions[id]
	if !ok {
		m.mu.Unlock()
		return errors.New(errors.CodeNotFound, "monitor not found")
	}
	delete(m.sessions, id)
	m.mu.Unlock()
	return session.stop("")
}

// StopAll stops every monitor session and leaves their history in memory.
func (m *Manager) StopAll() {
	for _, info := range m.List() {
		_ = m.Stop(info.ID)
	}
}

// Cleanup releases all monitor resources and clears history.
func (m *Manager) Cleanup() {
	m.mu.Lock()
	sessions := m.sessions
	m.sessions = make(map[string]*Session)
	m.mu.Unlock()
	for _, session := range sessions {
		_ = session.stop("")
	}
}

// List returns snapshots of all sessions.
func (m *Manager) List() []SessionInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]SessionInfo, 0, len(m.sessions))
	for _, session := range m.sessions {
		result = append(result, session.info())
	}
	return result
}

// Query returns a filtered page of monitor frames.
func (m *Manager) Query(req QueryRequest) (*FramePage, error) {
	session, err := m.get(req.MonitorID)
	if err != nil {
		return nil, err
	}
	return session.query(req), nil
}

// ClearFrames removes captured frames for a session.
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
		return nil, errors.New(errors.CodeInvalid, "monitor ID must not be empty")
	}
	m.mu.RLock()
	session, ok := m.sessions[id]
	m.mu.RUnlock()
	if !ok {
		return nil, errors.New(errors.CodeNotFound, "monitor not found")
	}
	return session, nil
}

// Session is one running or stopped monitor capture.
type Session struct {
	mu        sync.RWMutex
	req       StartRequest
	portA     port.Port
	portB     port.Port
	stopCh    chan struct{}
	wg        sync.WaitGroup
	status    string
	errText   string
	startedAt time.Time
	stoppedAt time.Time
	rxBytes   int64
	txBytes   int64
	nextSeq   int64
	frames    []Frame
}

func newSession(req StartRequest) *Session {
	if req.Provider == "" {
		req.Provider = ProviderBridge
	}
	if req.EndpointA == "" {
		req.EndpointA = req.PortA
	}
	if req.EndpointB == "" {
		req.EndpointB = req.PortB
	}
	if req.Name == "" {
		req.Name = fmt.Sprintf("监控 %s", req.PortA)
	}
	if req.Encoding == "" {
		req.Encoding = "utf-8"
	}
	return &Session{
		req:    req,
		stopCh: make(chan struct{}),
		status: StatusStopped,
	}
}

func (s *Session) start(ctx context.Context) error {
	cfgA := s.req.Config
	cfgA.PortName = s.req.EndpointA
	portA, err := port.Open(cfgA)
	if err != nil {
		s.setError(fmt.Sprintf("open port A: %v", err))
		return errors.Wrap(errors.CodeIO, "open monitor port A", err)
	}

	cfgB := s.req.Config
	cfgB.PortName = s.req.EndpointB
	portB, err := port.Open(cfgB)
	if err != nil {
		_ = portA.Close()
		s.setError(fmt.Sprintf("open port B: %v", err))
		return errors.Wrap(errors.CodeIO, "open monitor port B", err)
	}

	s.mu.Lock()
	s.portA = portA
	s.portB = portB
	s.status = StatusRunning
	s.errText = ""
	s.startedAt = time.Now()
	s.stoppedAt = time.Time{}
	s.mu.Unlock()

	s.wg.Add(2)
	go s.forward(ctx, portA, portB, DirectionAToB, s.req.PortA)
	go s.forward(ctx, portB, portA, DirectionBToA, s.req.PortB)
	return nil
}

func (s *Session) forward(ctx context.Context, src port.Port, dst port.Port, direction string, publicPort string) {
	defer s.wg.Done()
	buf := make([]byte, 4096)
	for {
		n, err := src.Read(buf)
		if err != nil {
			select {
			case <-s.stopCh:
				return
			case <-ctx.Done():
				return
			default:
				s.fail(fmt.Sprintf("%s read failed: %v", displayDirection(direction), err))
				return
			}
		}
		if n <= 0 {
			continue
		}
		readAt := time.Now()
		data := append([]byte(nil), buf[:n]...)
		if _, err := dst.Write(data); err != nil {
			s.fail(fmt.Sprintf("%s write failed: %v", displayDirection(direction), err))
			return
		}
		s.appendFrame(direction, publicPort, data, readAt)
	}
}

func (s *Session) appendFrame(direction string, publicPort string, data []byte, capturedAt time.Time) {
	s.mu.Lock()
	s.nextSeq++
	frame := enrichFrame(Frame{
		Seq:       s.nextSeq,
		Timestamp: capturedAt,
		Direction: direction,
		Port:      publicPort,
		Data:      append([]byte(nil), data...),
	}, s.req.Encoding)
	if direction == DirectionAToB {
		s.txBytes += int64(len(data))
	} else if direction == DirectionBToA {
		s.rxBytes += int64(len(data))
	}
	s.frames = append(s.frames, frame)
	if len(s.frames) > defaultFrameLimit {
		copy(s.frames, s.frames[len(s.frames)-defaultFrameLimit:])
		s.frames = s.frames[:defaultFrameLimit]
	}
	s.mu.Unlock()
}

func (s *Session) stop(reason string) error {
	s.mu.Lock()
	if s.status == StatusStopped || (s.status == StatusError && s.portA == nil && s.portB == nil) {
		s.mu.Unlock()
		return nil
	}
	if reason != "" {
		s.status = StatusError
		s.errText = reason
	} else {
		s.status = StatusStopped
	}
	s.stoppedAt = time.Now()
	select {
	case <-s.stopCh:
	default:
		close(s.stopCh)
	}
	portA := s.portA
	portB := s.portB
	s.portA = nil
	s.portB = nil
	s.mu.Unlock()

	if portA != nil {
		_ = portA.Close()
	}
	if portB != nil {
		_ = portB.Close()
	}
	s.wg.Wait()
	return nil
}

func (s *Session) setError(message string) {
	s.mu.Lock()
	s.status = StatusError
	s.errText = message
	s.mu.Unlock()
}

func (s *Session) fail(reason string) {
	s.mu.Lock()
	if s.status == StatusStopped {
		s.mu.Unlock()
		return
	}
	s.status = StatusError
	s.errText = reason
	s.stoppedAt = time.Now()
	select {
	case <-s.stopCh:
	default:
		close(s.stopCh)
	}
	portA := s.portA
	portB := s.portB
	s.portA = nil
	s.portB = nil
	s.mu.Unlock()

	if portA != nil {
		_ = portA.Close()
	}
	if portB != nil {
		_ = portB.Close()
	}
}

func (s *Session) info() SessionInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return SessionInfo{
		ID:                s.req.ID,
		Name:              s.req.Name,
		Provider:          s.req.Provider,
		PortA:             s.req.PortA,
		PortB:             s.req.PortB,
		ExternalPort:      s.req.ExternalPort,
		AutoVirtualPortID: s.req.AutoVirtualPortID,
		Config:            s.req.Config,
		Encoding:          s.req.Encoding,
		Status:            s.status,
		RxBytes:           s.rxBytes,
		TxBytes:           s.txBytes,
		FrameCount:        int64(len(s.frames)),
		StartedAt:         s.startedAt,
		StoppedAt:         s.stoppedAt,
		Error:             s.errText,
	}
}

func (s *Session) query(req QueryRequest) *FramePage {
	s.mu.RLock()
	frames := append([]Frame(nil), s.frames...)
	s.mu.RUnlock()

	if req.Limit <= 0 || req.Limit > 5000 {
		req.Limit = 1000
	}
	if req.Offset < 0 {
		req.Offset = 0
	}

	filtered := make([]Frame, 0, len(frames))
	for _, frame := range frames {
		if req.Direction != "" && req.Direction != "all" && frame.Direction != req.Direction {
			continue
		}
		if req.Search != "" && !frameMatchesSearch(frame, req.Search) {
			continue
		}
		filtered = append(filtered, frame)
	}

	total := int64(len(filtered))
	if req.Offset >= total {
		return &FramePage{Frames: []Frame{}, Total: total, NextOffset: total}
	}
	end := req.Offset + int64(req.Limit)
	if end > total {
		end = total
	}
	return &FramePage{
		Frames:     filtered[req.Offset:end],
		Total:      total,
		NextOffset: end,
	}
}

func frameMatchesSearch(frame Frame, query string) bool {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return true
	}
	return strings.Contains(strings.ToLower(frame.DisplayText), query) ||
		strings.Contains(strings.ToLower(frame.DisplayHex), query) ||
		strings.Contains(strings.ToLower(frame.DisplayDec), query) ||
		strings.Contains(strings.ToLower(frame.Port), query)
}

func (s *Session) clearFrames() {
	s.mu.Lock()
	s.frames = nil
	s.nextSeq = 0
	s.rxBytes = 0
	s.txBytes = 0
	s.mu.Unlock()
}
