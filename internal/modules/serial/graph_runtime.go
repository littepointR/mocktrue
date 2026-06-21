package serial

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/suyue/mocktrue/internal/core/errors"
	"github.com/suyue/mocktrue/internal/modules/serial/buffer"
	fb "github.com/suyue/mocktrue/internal/modules/serial/fecbus"
	"github.com/suyue/mocktrue/internal/modules/serial/manager"
	mb "github.com/suyue/mocktrue/internal/modules/serial/modbus"
	"github.com/suyue/mocktrue/internal/modules/serial/monitor"
	"github.com/suyue/mocktrue/internal/modules/serial/port"
)

const (
	SerialGraphStatusIdle    = "idle"
	SerialGraphStatusRunning = "running"
	SerialGraphStatusStopped = "stopped"
	SerialGraphStatusError   = "error"

	serialGraphFrameLimit      = 100000
	serialGraphFrameBytesLimit = 16 * 1024 * 1024
)

// SerialGraphStartRequest starts one executable serial topology graph.
type SerialGraphStartRequest struct {
	ID    string
	Nodes []SerialGraphNodeSpec
	Edges []SerialGraphEdgeSpec
}

// SerialGraphNodeSpec is the backend representation of one graph node.
type SerialGraphNodeSpec struct {
	ID       string
	Type     string
	Config   map[string]any
	Position SerialGraphPosition
}

// SerialGraphPosition is kept for frontend snapshot round-tripping.
type SerialGraphPosition struct {
	X int
	Y int
}

// SerialGraphEdgeSpec connects one output handle to one input handle.
type SerialGraphEdgeSpec struct {
	ID           string
	Source       string
	SourceHandle string
	Target       string
	TargetHandle string
}

// SerialGraphRuntimeInfo is a frontend-safe snapshot of one running graph.
type SerialGraphRuntimeInfo struct {
	ID        string
	Status    string
	StartedAt time.Time
	StoppedAt time.Time
	Error     string
	Nodes     []SerialGraphNodeStatus
}

// SerialGraphNodeStatus is a frontend-safe snapshot of one runtime node.
type SerialGraphNodeStatus struct {
	ID         string
	Type       string
	Status     string
	RxBytes    int64
	TxBytes    int64
	FrameCount int64
	ResourceID string
	Error      string
}

// SerialGraphSendRequest injects bytes into a graph node.
type SerialGraphSendRequest struct {
	GraphID  string
	NodeID   string
	Content  string
	Mode     string
	Encoding string
}

// SerialGraphBufferQuery reads one node's byte buffer.
type SerialGraphBufferQuery struct {
	GraphID string
	NodeID  string
	Offset  int64
	Length  int
}

// SerialGraphFrameQuery reads one node's captured frame history.
type SerialGraphFrameQuery struct {
	GraphID   string
	NodeID    string
	Offset    int64
	Limit     int
	Direction string
	Search    string
}

// SerialGraphFramePage is a filtered frame page for graph monitor nodes.
type SerialGraphFramePage struct {
	Frames     []monitor.Frame
	Total      int64
	NextOffset int64
}

type serialGraphProviderSpec struct {
	Type          string
	Inputs        []serialGraphPortSpec
	Outputs       []serialGraphPortSpec
	ResourceOwner bool
	ResourceKeys  []string
}

type serialGraphPortSpec struct {
	ID        string
	Kind      string
	Multiple  bool
	Direction string
}

type serialGraphRuntime struct {
	mu          sync.RWMutex
	svc         *Service
	id          string
	status      string
	startedAt   time.Time
	stoppedAt   time.Time
	err         string
	nodes       map[string]*serialGraphRuntimeNode
	edges       []SerialGraphEdgeSpec
	routes      map[serialGraphPortRef][]serialGraphPortRef
	cancelData  func()
	cancelAuto  func()
	ownedPorts  []string
	ownedVPorts []string
}

type serialGraphRuntimeNode struct {
	mu                    sync.RWMutex
	spec                  SerialGraphNodeSpec
	status                string
	err                   string
	rxBytes               int64
	txBytes               int64
	frameSeq              int64
	buffer                *buffer.RingBuffer
	frames                []monitor.Frame
	frameBytes            int64
	resourceID            string
	handleID              string
	externalWriteInFlight atomic.Bool
}

type serialGraphPortRef struct {
	nodeID string
	handle string
}

func (s *Service) ensureGraphRuntimeManagerLocked() {
	if s.graphs == nil {
		s.graphs = make(map[string]*serialGraphRuntime)
	}
}

// StartSerialGraph validates and starts a graph runtime.
func (s *Service) StartSerialGraph(ctx context.Context, req SerialGraphStartRequest) (*SerialGraphRuntimeInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if req.ID == "" {
		req.ID = fmt.Sprintf("serial-graph-%d", time.Now().UnixNano())
	}
	if errs := validateSerialGraphRuntimeRequest(req); len(errs) > 0 {
		return nil, errors.New(errors.CodeInvalid, strings.Join(errs, "; "))
	}

	s.mu.Lock()
	s.ensureGraphRuntimeManagerLocked()
	if _, exists := s.graphs[req.ID]; exists {
		s.mu.Unlock()
		return nil, errors.New(errors.CodeConflict, "serial graph already running: "+req.ID)
	}
	runtime := newSerialGraphRuntime(s, req)
	s.graphs[req.ID] = runtime
	s.mu.Unlock()

	if err := runtime.start(ctx); err != nil {
		s.mu.Lock()
		if s.graphs[req.ID] == runtime {
			delete(s.graphs, req.ID)
		}
		s.mu.Unlock()
		runtime.stop()
		return nil, err
	}
	info := runtime.info()
	return &info, nil
}

// StopSerialGraph stops one graph and releases every graph-owned resource.
func (s *Service) StopSerialGraph(id string) error {
	if id == "" {
		return errors.New(errors.CodeInvalid, "graph ID must not be empty")
	}
	s.mu.Lock()
	runtime, ok := s.graphs[id]
	if ok {
		delete(s.graphs, id)
	}
	s.mu.Unlock()
	if !ok {
		return errors.New(errors.CodeNotFound, "serial graph not found: "+id)
	}
	runtime.stop()
	return nil
}

// ListSerialGraphs returns all active graph runtimes.
func (s *Service) ListSerialGraphs() []SerialGraphRuntimeInfo {
	s.mu.RLock()
	graphs := make([]*serialGraphRuntime, 0, len(s.graphs))
	for _, runtime := range s.graphs {
		graphs = append(graphs, runtime)
	}
	s.mu.RUnlock()

	result := make([]SerialGraphRuntimeInfo, 0, len(graphs))
	for _, runtime := range graphs {
		result = append(result, runtime.info())
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

// GetSerialGraphStatus returns one graph runtime snapshot.
func (s *Service) GetSerialGraphStatus(id string) (*SerialGraphRuntimeInfo, error) {
	runtime, err := s.serialGraphRuntime(id)
	if err != nil {
		return nil, err
	}
	info := runtime.info()
	return &info, nil
}

// SendSerialGraphNode injects a payload into a sender/protocol-capable graph node.
func (s *Service) SendSerialGraphNode(req SerialGraphSendRequest) (int, error) {
	if req.NodeID == "" {
		return 0, errors.New(errors.CodeInvalid, "node ID must not be empty")
	}
	runtime, err := s.serialGraphRuntime(req.GraphID)
	if err != nil {
		return 0, err
	}
	written, err := runtime.sendRequest(req)
	if err != nil {
		return 0, err
	}
	return written, nil
}

// QuerySerialGraphNodeBuffer reads buffered bytes for a receiver-like node.
func (s *Service) QuerySerialGraphNodeBuffer(req SerialGraphBufferQuery) (*buffer.Snapshot, error) {
	if req.Length <= 0 {
		req.Length = 4096
	}
	runtime, err := s.serialGraphRuntime(req.GraphID)
	if err != nil {
		return nil, err
	}
	return runtime.queryBuffer(req.NodeID, req.Offset, req.Length)
}

// ClearSerialGraphNodeBuffer clears buffered data for one node.
func (s *Service) ClearSerialGraphNodeBuffer(graphID string, nodeID string) error {
	runtime, err := s.serialGraphRuntime(graphID)
	if err != nil {
		return err
	}
	return runtime.clearBuffer(nodeID)
}

// ResetSerialGraphNodeCounters resets RX/TX counters for one graph node.
func (s *Service) ResetSerialGraphNodeCounters(graphID string, nodeID string) error {
	runtime, err := s.serialGraphRuntime(graphID)
	if err != nil {
		return err
	}
	return runtime.resetCounters(nodeID)
}

// QuerySerialGraphNodeFrames reads captured frames for a graph monitor node.
func (s *Service) QuerySerialGraphNodeFrames(req SerialGraphFrameQuery) (*SerialGraphFramePage, error) {
	if req.Limit <= 0 {
		req.Limit = 100
	}
	runtime, err := s.serialGraphRuntime(req.GraphID)
	if err != nil {
		return nil, err
	}
	return runtime.queryFrames(req)
}

func (s *Service) serialGraphRuntime(id string) (*serialGraphRuntime, error) {
	if id == "" {
		return nil, errors.New(errors.CodeInvalid, "graph ID must not be empty")
	}
	s.mu.RLock()
	runtime, ok := s.graphs[id]
	s.mu.RUnlock()
	if !ok {
		return nil, errors.New(errors.CodeNotFound, "serial graph not found: "+id)
	}
	return runtime, nil
}

func (s *Service) stopAllSerialGraphs() {
	s.mu.Lock()
	graphs := s.graphs
	s.graphs = make(map[string]*serialGraphRuntime)
	s.mu.Unlock()
	for _, runtime := range graphs {
		runtime.stop()
	}
}

func (s *Service) ensureGraphPortAvailable(portName string) error {
	if portName == "" {
		return errors.New(errors.CodeInvalid, "port name must not be empty")
	}
	if s.findOpenHandle(portName) != nil {
		return errors.New(errors.CodeConflict, "graph port already open in serial terminal: "+portName)
	}
	if s.monitorPortInUse(portName) {
		return errors.New(errors.CodeConflict, "graph port already used by serial monitor: "+portName)
	}
	if s.bridgePortInUse(portName) {
		return errors.New(errors.CodeConflict, "graph port already used by serial bridge: "+portName)
	}
	if s.modbus != nil && s.modbus.PortInUse(portName) {
		return errors.New(errors.CodeConflict, "graph port already open in modbus session: "+portName)
	}
	if s.fecbus != nil && s.fecbus.PortInUse(portName) {
		return errors.New(errors.CodeConflict, "graph port already open in fecbus session: "+portName)
	}
	return nil
}

func newSerialGraphRuntime(svc *Service, req SerialGraphStartRequest) *serialGraphRuntime {
	nodes := make(map[string]*serialGraphRuntimeNode, len(req.Nodes))
	for _, spec := range req.Nodes {
		nodes[spec.ID] = &serialGraphRuntimeNode{
			spec:   spec,
			status: SerialGraphStatusIdle,
			buffer: buffer.NewRing(64 * 1024 * 1024),
		}
	}
	runtime := &serialGraphRuntime{
		svc:       svc,
		id:        req.ID,
		status:    SerialGraphStatusIdle,
		nodes:     nodes,
		edges:     append([]SerialGraphEdgeSpec(nil), req.Edges...),
		routes:    make(map[serialGraphPortRef][]serialGraphPortRef),
		startedAt: time.Now(),
	}
	for _, edge := range req.Edges {
		source := serialGraphPortRef{nodeID: edge.Source, handle: edge.SourceHandle}
		target := serialGraphPortRef{nodeID: edge.Target, handle: edge.TargetHandle}
		runtime.routes[source] = append(runtime.routes[source], target)
	}
	return runtime
}

func (r *serialGraphRuntime) start(ctx context.Context) error {
	r.mu.Lock()
	if r.status == SerialGraphStatusRunning {
		r.mu.Unlock()
		return nil
	}
	r.status = SerialGraphStatusRunning
	r.startedAt = time.Now()
	for _, node := range r.nodes {
		node.setStatus(SerialGraphStatusRunning, "")
	}
	r.mu.Unlock()

	if err := r.startResourceNodes(ctx); err != nil {
		r.setError(err.Error())
		return err
	}
	r.subscribeSerialData()
	if err := r.startAutoSenders(); err != nil {
		r.setError(err.Error())
		return err
	}
	return nil
}

func (r *serialGraphRuntime) startResourceNodes(ctx context.Context) error {
	for _, node := range r.sortedNodes() {
		switch node.spec.Type {
		case "serial.physical":
			if err := r.startPhysicalNode(ctx, node); err != nil {
				return err
			}
		case "serial.virtual":
			if err := r.startVirtualNode(ctx, node); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *serialGraphRuntime) startPhysicalNode(ctx context.Context, node *serialGraphRuntimeNode) error {
	portName := graphStringConfig(node.spec.Config, "portName")
	if portName == "" {
		return nil
	}
	if err := r.svc.ensureGraphPortAvailable(portName); err != nil {
		node.setStatus(SerialGraphStatusError, err.Error())
		return err
	}
	status, err := r.svc.manager.Open(ctx, manager.OpenRequest{Config: graphSerialConfig(node.spec.Config)})
	if err != nil {
		node.setStatus(SerialGraphStatusError, err.Error())
		return errors.Wrap(errors.AsCode(err), "start graph physical node "+node.spec.ID, err)
	}
	node.setResourceID(portName)
	node.setHandleID(status.ID)
	r.addOwnedPort(status.ID)
	return nil
}

func (r *serialGraphRuntime) startVirtualNode(ctx context.Context, node *serialGraphRuntimeNode) error {
	portName := graphStringConfig(node.spec.Config, "portName")
	if portName == "" {
		return nil
	}
	if err := r.svc.ensureGraphPortAvailable(portName); err != nil {
		node.setStatus(SerialGraphStatusError, err.Error())
		return err
	}
	id := graphStringConfig(node.spec.Config, "id")
	if id == "" {
		id = r.id + "-" + node.spec.ID
	}
	info, err := r.svc.CreateVirtualPort(ctx, id, portName)
	if err != nil {
		node.setStatus(SerialGraphStatusError, err.Error())
		return errors.Wrap(errors.AsCode(err), "start graph virtual node "+node.spec.ID, err)
	}
	config := graphSerialConfig(node.spec.Config)
	config.PortName = r.svc.monitorEndpoint(info.Port)
	status, err := r.svc.manager.Open(ctx, manager.OpenRequest{Config: config})
	if err != nil {
		_ = r.svc.DeleteVirtualPort(info.ID)
		node.setStatus(SerialGraphStatusError, err.Error())
		return errors.Wrap(errors.AsCode(err), "start graph virtual node peer "+node.spec.ID, err)
	}
	node.setResourceID(info.Port)
	node.setHandleID(status.ID)
	r.addOwnedPort(status.ID)
	r.addOwnedVirtualPort(info.ID)
	return nil
}

func (r *serialGraphRuntime) subscribeSerialData() {
	if r.svc.bus == nil {
		return
	}
	r.mu.Lock()
	if r.cancelData != nil {
		r.mu.Unlock()
		return
	}
	r.cancelData = r.svc.bus.Subscribe("serial:data", func(payload any) {
		evt, ok := payload.(manager.DataEvent)
		if !ok {
			return
		}
		node := r.nodeByHandle(evt.PortID)
		if node == nil {
			return
		}
		node.addRx(len(evt.Data))
		node.appendBuffer(evt.Data)
		r.emit(serialGraphPortRef{nodeID: node.spec.ID, handle: "rx"}, evt.Data)
	})
	r.mu.Unlock()
}

func (r *serialGraphRuntime) startAutoSenders() error {
	type autoSender struct {
		nodeID   string
		data     []byte
		interval time.Duration
	}

	senders := []autoSender{}
	for _, node := range r.sortedNodes() {
		if !graphBoolConfig(node.spec.Config, "autoSend", false) {
			continue
		}
		data, err := graphAutoNodeData(node)
		if err != nil {
			node.setStatus(SerialGraphStatusError, err.Error())
			return errors.Wrap(errors.CodeInvalid, "auto sender "+node.spec.ID, err)
		}
		interval := time.Duration(graphIntConfig(node.spec.Config, "intervalMs", 1000)) * time.Millisecond
		if interval < 10*time.Millisecond {
			interval = 10 * time.Millisecond
		}
		senders = append(senders, autoSender{
			nodeID:   node.spec.ID,
			data:     data,
			interval: interval,
		})
	}
	if len(senders) == 0 {
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	r.mu.Lock()
	r.cancelAuto = cancel
	r.mu.Unlock()
	for _, sender := range senders {
		go r.runAutoSender(ctx, sender.nodeID, sender.data, sender.interval)
	}
	return nil
}

func (r *serialGraphRuntime) runAutoSender(ctx context.Context, nodeID string, data []byte, interval time.Duration) {
	send := func() bool {
		select {
		case <-ctx.Done():
			return false
		default:
		}
		if err := r.send(nodeID, append([]byte(nil), data...)); err != nil {
			if node := r.node(nodeID); node != nil {
				node.setStatus(SerialGraphStatusError, err.Error())
			}
			return false
		}
		return true
	}
	if !send() {
		return
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !send() {
				return
			}
		}
	}
}

func (r *serialGraphRuntime) stop() {
	r.mu.Lock()
	if r.status == SerialGraphStatusStopped {
		r.mu.Unlock()
		return
	}
	r.status = SerialGraphStatusStopped
	r.stoppedAt = time.Now()
	cancelData := r.cancelData
	r.cancelData = nil
	cancelAuto := r.cancelAuto
	r.cancelAuto = nil
	ownedPorts := append([]string(nil), r.ownedPorts...)
	ownedVPorts := append([]string(nil), r.ownedVPorts...)
	r.ownedPorts = nil
	r.ownedVPorts = nil
	for _, node := range r.nodes {
		node.setStatus(SerialGraphStatusStopped, "")
	}
	r.mu.Unlock()

	if cancelData != nil {
		cancelData()
	}
	if cancelAuto != nil {
		cancelAuto()
	}
	for _, portID := range ownedVPorts {
		_ = r.svc.DeleteVirtualPort(portID)
	}
	for _, portID := range ownedPorts {
		_ = r.svc.ClosePort(portID)
	}
}

func (r *serialGraphRuntime) sendRequest(req SerialGraphSendRequest) (int, error) {
	node := r.node(req.NodeID)
	if node == nil {
		return 0, errors.New(errors.CodeNotFound, "graph node not found: "+req.NodeID)
	}
	data, err := graphNodeSendData(node, req)
	if err != nil {
		return 0, err
	}
	if len(data) == 0 {
		return 0, errors.New(errors.CodeInvalid, "content must not be empty")
	}
	if err := r.send(req.NodeID, data); err != nil {
		return 0, err
	}
	return len(data), nil
}

func (r *serialGraphRuntime) send(nodeID string, data []byte) error {
	node := r.node(nodeID)
	if node == nil {
		return errors.New(errors.CodeNotFound, "graph node not found: "+nodeID)
	}
	switch node.spec.Type {
	case "serial.sender", "serial.modbus.master", "serial.modbus.slave", "serial.fecbus.master", "serial.fecbus.slave":
		node.addTx(len(data))
		r.emit(serialGraphPortRef{nodeID: nodeID, handle: defaultGraphOutputHandle(node.spec.Type)}, data)
		return nil
	case "serial.physical":
		handleID := node.handle()
		if handleID == "" {
			return errors.New(errors.CodeInvalid, "physical node is not bound to an open port: "+nodeID)
		}
		n, err := r.svc.manager.Write(handleID, data)
		if err != nil {
			node.setStatus(SerialGraphStatusError, err.Error())
			return err
		}
		node.addTx(n)
		return nil
	default:
		return errors.New(errors.CodeInvalid, "node does not support direct send: "+nodeID)
	}
}

func (r *serialGraphRuntime) receive(ref serialGraphPortRef, data []byte) {
	node := r.node(ref.nodeID)
	if node == nil || len(data) == 0 {
		return
	}
	node.addRx(len(data))
	switch node.spec.Type {
	case "serial.receiver":
		node.appendBuffer(data)
	case "serial.monitor":
		node.appendFrame(data)
	case "serial.tap", "serial.tee":
		node.addTx(len(data))
		r.emit(serialGraphPortRef{nodeID: ref.nodeID, handle: "out"}, data)
	case "serial.bridge":
		switch ref.handle {
		case "a-in":
			node.addTx(len(data))
			r.emit(serialGraphPortRef{nodeID: ref.nodeID, handle: "b-out"}, data)
		case "b-in":
			node.addTx(len(data))
			r.emit(serialGraphPortRef{nodeID: ref.nodeID, handle: "a-out"}, data)
		}
	case "serial.physical":
		handleID := node.handle()
		if handleID != "" {
			if n, err := r.svc.manager.Write(handleID, data); err == nil {
				node.addTx(n)
			} else {
				node.setStatus(SerialGraphStatusError, err.Error())
			}
		}
	case "serial.virtual":
		loopback := append([]byte(nil), data...)
		node.addTx(len(loopback))
		node.appendBuffer(loopback)
		r.emit(serialGraphPortRef{nodeID: ref.nodeID, handle: "rx"}, loopback)
		r.writeVirtualExternal(node, loopback)
	case "serial.modbus.slave":
		node.appendBuffer(data)
		r.handleModbusSlave(node, data)
	case "serial.fecbus.slave":
		node.appendBuffer(data)
		r.handleFecbusSlave(node, data)
	case "serial.modbus.master", "serial.fecbus.master":
		node.appendBuffer(data)
	}
}

func (r *serialGraphRuntime) writeVirtualExternal(node *serialGraphRuntimeNode, data []byte) {
	handleID := node.handle()
	if handleID == "" || len(data) == 0 {
		return
	}
	if !node.tryBeginExternalWrite() {
		return
	}
	payload := append([]byte(nil), data...)
	go func() {
		defer node.endExternalWrite()
		if _, err := r.svc.manager.Write(handleID, payload); err != nil && !r.isStopped() {
			node.setStatus(SerialGraphStatusError, err.Error())
		}
	}()
}

func (r *serialGraphRuntime) isStopped() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.status == SerialGraphStatusStopped
}

func (r *serialGraphRuntime) handleModbusSlave(node *serialGraphRuntimeNode, data []byte) {
	mode := mb.FrameMode(graphStringConfigWithDefault(node.spec.Config, "mode", string(mb.FrameModeRTU)))
	frame, err := mb.DecodeFrame(mode, data)
	if err != nil {
		node.setStatus(SerialGraphStatusError, err.Error())
		return
	}
	if !graphUnitIDAllowed(node.spec.Config, frame.UnitID) {
		return
	}
	response, err := graphModbusSlaveResponse(frame)
	if err != nil {
		node.setStatus(SerialGraphStatusError, err.Error())
		return
	}
	encoded, err := mb.EncodeFrame(mode, frame.UnitID, response)
	if err != nil {
		node.setStatus(SerialGraphStatusError, err.Error())
		return
	}
	node.addTx(len(encoded))
	r.emit(serialGraphPortRef{nodeID: node.spec.ID, handle: "tx"}, encoded)
}

func (r *serialGraphRuntime) handleFecbusSlave(node *serialGraphRuntimeNode, data []byte) {
	frame, err := fb.DecodeFrame(data)
	if err != nil {
		node.setStatus(SerialGraphStatusError, err.Error())
		return
	}
	address := byte(graphIntConfig(node.spec.Config, "address", 2))
	if frame.TargetAddress != address && frame.TargetAddress != 0 {
		return
	}
	if !graphBoolConfig(node.spec.Config, "autoStatusAnswer", true) {
		return
	}
	answer, err := fb.BuildStatusAnswer(frame, fb.StatusCode(byte(graphIntConfig(node.spec.Config, "defaultStatus", int(fb.StatusReceivedOK)))))
	if err != nil {
		node.setStatus(SerialGraphStatusError, err.Error())
		return
	}
	encoded, err := fb.EncodeFrame(answer)
	if err != nil {
		node.setStatus(SerialGraphStatusError, err.Error())
		return
	}
	node.addTx(len(encoded))
	r.emit(serialGraphPortRef{nodeID: node.spec.ID, handle: "tx"}, encoded)
}

func (r *serialGraphRuntime) emit(source serialGraphPortRef, data []byte) {
	targets := r.targets(source)
	for _, target := range targets {
		next := append([]byte(nil), data...)
		r.receive(target, next)
	}
}

func (r *serialGraphRuntime) targets(source serialGraphPortRef) []serialGraphPortRef {
	r.mu.RLock()
	targets := append([]serialGraphPortRef(nil), r.routes[source]...)
	r.mu.RUnlock()
	return targets
}

func (r *serialGraphRuntime) queryBuffer(nodeID string, offset int64, length int) (*buffer.Snapshot, error) {
	node := r.node(nodeID)
	if node == nil {
		return nil, errors.New(errors.CodeNotFound, "graph node not found: "+nodeID)
	}
	return node.queryBuffer(offset, length)
}

func (r *serialGraphRuntime) clearBuffer(nodeID string) error {
	node := r.node(nodeID)
	if node == nil {
		return errors.New(errors.CodeNotFound, "graph node not found: "+nodeID)
	}
	node.clearBuffer()
	return nil
}

func (r *serialGraphRuntime) resetCounters(nodeID string) error {
	node := r.node(nodeID)
	if node == nil {
		return errors.New(errors.CodeNotFound, "graph node not found: "+nodeID)
	}
	node.resetCounters()
	return nil
}

func (r *serialGraphRuntime) queryFrames(req SerialGraphFrameQuery) (*SerialGraphFramePage, error) {
	node := r.node(req.NodeID)
	if node == nil {
		return nil, errors.New(errors.CodeNotFound, "graph node not found: "+req.NodeID)
	}
	return node.queryFrames(req), nil
}

func (r *serialGraphRuntime) info() SerialGraphRuntimeInfo {
	r.mu.RLock()
	info := SerialGraphRuntimeInfo{
		ID:        r.id,
		Status:    r.status,
		StartedAt: r.startedAt,
		StoppedAt: r.stoppedAt,
		Error:     r.err,
		Nodes:     make([]SerialGraphNodeStatus, 0, len(r.nodes)),
	}
	nodes := r.sortedNodesLocked()
	r.mu.RUnlock()

	for _, node := range nodes {
		info.Nodes = append(info.Nodes, node.statusSnapshot())
	}
	return info
}

func (r *serialGraphRuntime) node(id string) *serialGraphRuntimeNode {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.nodes[id]
}

func (r *serialGraphRuntime) nodeByHandle(handleID string) *serialGraphRuntimeNode {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, node := range r.nodes {
		if node.handle() == handleID {
			return node
		}
	}
	return nil
}

func (r *serialGraphRuntime) setError(message string) {
	r.mu.Lock()
	r.status = SerialGraphStatusError
	r.err = message
	r.mu.Unlock()
}

func (r *serialGraphRuntime) addOwnedPort(portID string) {
	if portID == "" {
		return
	}
	r.mu.Lock()
	r.ownedPorts = append(r.ownedPorts, portID)
	r.mu.Unlock()
}

func (r *serialGraphRuntime) addOwnedVirtualPort(portID string) {
	if portID == "" {
		return
	}
	r.mu.Lock()
	r.ownedVPorts = append(r.ownedVPorts, portID)
	r.mu.Unlock()
}

func (r *serialGraphRuntime) sortedNodes() []*serialGraphRuntimeNode {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.sortedNodesLocked()
}

func (r *serialGraphRuntime) sortedNodesLocked() []*serialGraphRuntimeNode {
	nodes := make([]*serialGraphRuntimeNode, 0, len(r.nodes))
	for _, node := range r.nodes {
		nodes = append(nodes, node)
	}
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].spec.ID < nodes[j].spec.ID })
	return nodes
}

func (n *serialGraphRuntimeNode) setStatus(status string, errMessage string) {
	n.mu.Lock()
	n.status = status
	n.err = errMessage
	n.mu.Unlock()
}

func (n *serialGraphRuntimeNode) setResourceID(resourceID string) {
	n.mu.Lock()
	n.resourceID = resourceID
	n.mu.Unlock()
}

func (n *serialGraphRuntimeNode) setHandleID(handleID string) {
	n.mu.Lock()
	n.handleID = handleID
	n.mu.Unlock()
}

func (n *serialGraphRuntimeNode) resource() string {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.resourceID
}

func (n *serialGraphRuntimeNode) handle() string {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.handleID
}

func (n *serialGraphRuntimeNode) tryBeginExternalWrite() bool {
	return n.externalWriteInFlight.CompareAndSwap(false, true)
}

func (n *serialGraphRuntimeNode) endExternalWrite() {
	n.externalWriteInFlight.Store(false)
}

func (n *serialGraphRuntimeNode) addRx(length int) {
	n.mu.Lock()
	n.rxBytes += int64(length)
	n.mu.Unlock()
}

func (n *serialGraphRuntimeNode) addTx(length int) {
	n.mu.Lock()
	n.txBytes += int64(length)
	n.mu.Unlock()
}

func (n *serialGraphRuntimeNode) appendBuffer(data []byte) {
	n.mu.RLock()
	buf := n.buffer
	n.mu.RUnlock()
	if buf == nil {
		return
	}
	base := buf.Total()
	buf.Append(buffer.Chunk{BaseOffset: base, Data: append([]byte(nil), data...)})
}

func (n *serialGraphRuntimeNode) queryBuffer(offset int64, length int) (*buffer.Snapshot, error) {
	n.mu.RLock()
	buf := n.buffer
	n.mu.RUnlock()
	if buf == nil {
		return &buffer.Snapshot{Offset: offset, EOF: true}, nil
	}
	return buf.Query(offset, length)
}

func (n *serialGraphRuntimeNode) clearBuffer() {
	n.mu.RLock()
	buf := n.buffer
	n.mu.RUnlock()
	if buf != nil {
		buf.Reset()
	}
	n.mu.Lock()
	n.frames = nil
	n.frameBytes = 0
	n.frameSeq = 0
	n.mu.Unlock()
}

func (n *serialGraphRuntimeNode) resetCounters() {
	n.mu.Lock()
	n.rxBytes = 0
	n.txBytes = 0
	n.frameSeq = 0
	n.mu.Unlock()
}

func (n *serialGraphRuntimeNode) appendFrame(data []byte) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.frameSeq += 1
	frame := monitor.Frame{
		Seq:         n.frameSeq,
		Timestamp:   time.Now(),
		Direction:   "接收",
		Port:        n.spec.ID,
		Length:      len(data),
		Data:        append([]byte(nil), data...),
		DisplayText: string(data),
		DisplayHex:  formatGraphFrameBytes(data, "%02x"),
		DisplayDec:  formatGraphFrameBytes(data, "%d"),
		DisplayOct:  formatGraphFrameBytes(data, "%03o"),
		DisplayBin:  formatGraphFrameBytes(data, "%08b"),
		Encoding:    graphStringConfig(n.spec.Config, "encoding"),
	}
	n.frames = append(n.frames, frame)
	n.frameBytes += int64(len(data))
	for len(n.frames) > 0 && (len(n.frames) > serialGraphFrameLimit || (n.frameBytes > serialGraphFrameBytesLimit && len(n.frames) > 1)) {
		n.frameBytes -= int64(len(n.frames[0].Data))
		n.frames = n.frames[1:]
	}
}

func formatGraphFrameBytes(data []byte, format string) string {
	parts := make([]string, 0, len(data))
	for _, b := range data {
		parts = append(parts, fmt.Sprintf(format, b))
	}
	return strings.Join(parts, " ")
}

func (n *serialGraphRuntimeNode) queryFrames(req SerialGraphFrameQuery) *SerialGraphFramePage {
	n.mu.RLock()
	frames := append([]monitor.Frame(nil), n.frames...)
	n.mu.RUnlock()

	filtered := frames[:0]
	search := strings.TrimSpace(strings.ToLower(req.Search))
	for _, frame := range frames {
		if req.Direction != "" && frame.Direction != req.Direction {
			continue
		}
		if search != "" {
			haystack := strings.ToLower(frame.DisplayText + " " + frame.DisplayHex + " " + frame.DisplayDec + " " + frame.DisplayOct + " " + frame.DisplayBin)
			if !strings.Contains(haystack, search) {
				continue
			}
		}
		filtered = append(filtered, frame)
	}

	offset := req.Offset
	if offset < 0 {
		offset = 0
	}
	if offset > int64(len(filtered)) {
		offset = int64(len(filtered))
	}
	limit := req.Limit
	if limit <= 0 {
		limit = 100
	}
	end := int(offset) + limit
	if end > len(filtered) {
		end = len(filtered)
	}
	pageFrames := append([]monitor.Frame(nil), filtered[offset:int64(end)]...)
	return &SerialGraphFramePage{
		Frames:     pageFrames,
		Total:      int64(len(filtered)),
		NextOffset: int64(end),
	}
}

func (n *serialGraphRuntimeNode) statusSnapshot() SerialGraphNodeStatus {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return SerialGraphNodeStatus{
		ID:         n.spec.ID,
		Type:       n.spec.Type,
		Status:     n.status,
		RxBytes:    n.rxBytes,
		TxBytes:    n.txBytes,
		FrameCount: int64(len(n.frames)),
		ResourceID: n.resourceID,
		Error:      n.err,
	}
}

func encodeGraphPayload(content string, mode string, encoding string) ([]byte, error) {
	if mode == "hex" {
		decoded, err := decodeHexContent(content)
		if err != nil {
			return nil, errors.Wrap(errors.CodeInvalid, "decode hex content", err)
		}
		return decoded, nil
	}
	data, err := encodeSerialText(content, encoding)
	if err != nil {
		return nil, errors.Wrap(errors.CodeInvalid, "encode text content", err)
	}
	return data, nil
}

func graphNodeSendData(node *serialGraphRuntimeNode, req SerialGraphSendRequest) ([]byte, error) {
	if req.Content != "" {
		return encodeGraphPayload(req.Content, req.Mode, req.Encoding)
	}
	switch node.spec.Type {
	case "serial.modbus.master":
		return graphModbusMasterFrame(node.spec.Config)
	case "serial.fecbus.master", "serial.fecbus.slave":
		return graphFecbusFrame(node.spec.Config)
	default:
		return nil, errors.New(errors.CodeInvalid, "content must not be empty")
	}
}

func graphAutoSenderData(config map[string]any) ([]byte, error) {
	content := graphStringConfigRaw(config, "payload")
	data, err := encodeGraphPayload(
		content,
		graphStringConfigWithDefault(config, "mode", "ascii"),
		graphStringConfigWithDefault(config, "encoding", "utf-8"),
	)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, errors.New(errors.CodeInvalid, "payload must not be empty")
	}
	return data, nil
}

func graphAutoNodeData(node *serialGraphRuntimeNode) ([]byte, error) {
	switch node.spec.Type {
	case "serial.sender":
		return graphAutoSenderData(node.spec.Config)
	case "serial.modbus.master", "serial.fecbus.master":
		data, err := graphNodeSendData(node, SerialGraphSendRequest{})
		if err != nil {
			return nil, err
		}
		if len(data) == 0 {
			return nil, errors.New(errors.CodeInvalid, "auto sender data must not be empty")
		}
		return data, nil
	default:
		return nil, errors.New(errors.CodeInvalid, "node does not support auto send: "+node.spec.Type)
	}
}

func graphModbusMasterFrame(config map[string]any) ([]byte, error) {
	mode := mb.FrameMode(graphStringConfigWithDefault(config, "mode", string(mb.FrameModeRTU)))
	unitID := graphFirstUnitID(config)
	function := mb.FunctionCode(byte(graphIntConfig(config, "functionCode", int(mb.FunctionReadHoldingRegisters))))
	addressMode := mb.AddressMode(graphStringConfigWithDefault(config, "addressMode", string(mb.AddressModeZeroBased)))
	addressInput := uint16(graphIntConfig(config, "address", 0))
	address, err := mb.ProtocolAddress(addressMode, function, addressInput)
	if err != nil {
		return nil, errors.Wrap(errors.CodeInvalid, "modbus address", err)
	}
	quantity := uint16(graphIntConfig(config, "quantity", 1))
	value := uint16(graphIntConfig(config, "value", 0))

	var pdu mb.PDU
	switch function {
	case mb.FunctionReadCoils, mb.FunctionReadDiscreteInputs, mb.FunctionReadHoldingRegisters, mb.FunctionReadInputRegisters:
		pdu, err = mb.BuildReadRequest(function, address, quantity)
	case mb.FunctionWriteSingleCoil, mb.FunctionWriteSingleRegister:
		pdu, err = mb.BuildWriteSingleRequest(function, address, value)
	default:
		err = fmt.Errorf("unsupported graph modbus master function: %02x", byte(function))
	}
	if err != nil {
		return nil, errors.Wrap(errors.CodeInvalid, "build modbus request", err)
	}
	frame, err := mb.EncodeFrame(mode, unitID, pdu)
	if err != nil {
		return nil, errors.Wrap(errors.CodeInvalid, "encode modbus frame", err)
	}
	return frame, nil
}

func graphModbusSlaveResponse(frame mb.DecodedFrame) (mb.PDU, error) {
	data := frame.PDU.Data
	exception := func(code byte) mb.PDU {
		return mb.PDU{Function: frame.PDU.Function | 0x80, Data: []byte{code}}
	}
	switch frame.PDU.Function {
	case mb.FunctionReadCoils, mb.FunctionReadDiscreteInputs:
		if len(data) < 4 {
			return exception(mb.ExceptionIllegalDataValue), nil
		}
		quantity := int(uint16(data[2])<<8 | uint16(data[3]))
		if quantity <= 0 {
			return exception(mb.ExceptionIllegalDataValue), nil
		}
		byteCount := (quantity + 7) / 8
		return mb.PDU{Function: frame.PDU.Function, Data: append([]byte{byte(byteCount)}, make([]byte, byteCount)...)}, nil
	case mb.FunctionReadHoldingRegisters, mb.FunctionReadInputRegisters:
		if len(data) < 4 {
			return exception(mb.ExceptionIllegalDataValue), nil
		}
		quantity := int(uint16(data[2])<<8 | uint16(data[3]))
		if quantity <= 0 || quantity > 125 {
			return exception(mb.ExceptionIllegalDataValue), nil
		}
		return mb.PDU{Function: frame.PDU.Function, Data: append([]byte{byte(quantity * 2)}, make([]byte, quantity*2)...)}, nil
	case mb.FunctionWriteSingleCoil, mb.FunctionWriteSingleRegister:
		if len(data) < 4 {
			return exception(mb.ExceptionIllegalDataValue), nil
		}
		return mb.PDU{Function: frame.PDU.Function, Data: append([]byte(nil), data[:4]...)}, nil
	case mb.FunctionWriteMultipleCoils, mb.FunctionWriteMultipleRegisters:
		if len(data) < 4 {
			return exception(mb.ExceptionIllegalDataValue), nil
		}
		return mb.PDU{Function: frame.PDU.Function, Data: append([]byte(nil), data[:4]...)}, nil
	default:
		return exception(mb.ExceptionIllegalFunction), nil
	}
}

func graphFecbusFrame(config map[string]any) ([]byte, error) {
	payload, err := fb.ParseHex(graphStringConfig(config, "dataHex"))
	if err != nil {
		return nil, errors.Wrap(errors.CodeInvalid, "parse fecbus data", err)
	}
	function := fb.FunctionCode(byte(graphIntConfig(config, "functionCode", int(fb.FunctionQueryProtocolVersion))))
	data, err := fb.BuildData(function, payload)
	if err != nil {
		return nil, errors.Wrap(errors.CodeInvalid, "build fecbus data", err)
	}
	frame := fb.Frame{
		Type:          fb.FrameType(byte(graphIntConfig(config, "frameType", int(fb.FrameTypeRequest)))),
		TargetAddress: byte(graphIntConfig(config, "targetAddress", 2)),
		Priority:      byte(graphIntConfig(config, "priority", 3)),
		SourceAddress: byte(graphIntConfig(config, "sourceAddress", 1)),
		MessageNumber: byte(graphIntConfig(config, "messageNumber", 1)),
		GroupNumber:   byte(graphIntConfig(config, "groupNumber", 0)),
		Data:          data,
	}
	encoded, err := fb.EncodeFrame(frame)
	if err != nil {
		return nil, errors.Wrap(errors.CodeInvalid, "encode fecbus frame", err)
	}
	return encoded, nil
}

func graphFirstUnitID(config map[string]any) byte {
	unitID := graphIntConfig(config, "unitID", 0)
	if unitID > 0 {
		return byte(unitID)
	}
	for _, token := range strings.FieldsFunc(graphStringConfigWithDefault(config, "unitIds", "1"), func(r rune) bool {
		return r == ',' || r == ';' || r == ' ' || r == '\t' || r == '\n'
	}) {
		parsed, err := strconv.Atoi(token)
		if err == nil && parsed > 0 && parsed <= 247 {
			return byte(parsed)
		}
	}
	return 1
}

func graphUnitIDAllowed(config map[string]any, unitID byte) bool {
	for _, token := range strings.FieldsFunc(graphStringConfigWithDefault(config, "unitIds", "1"), func(r rune) bool {
		return r == ',' || r == ';' || r == ' ' || r == '\t' || r == '\n'
	}) {
		parsed, err := strconv.Atoi(token)
		if err == nil && parsed == int(unitID) {
			return true
		}
	}
	return false
}

func defaultGraphOutputHandle(nodeType string) string {
	switch nodeType {
	case "serial.modbus.master", "serial.modbus.slave", "serial.fecbus.master", "serial.fecbus.slave":
		return "tx"
	default:
		return "out"
	}
}

func validateSerialGraphRuntimeRequest(req SerialGraphStartRequest) []string {
	var errs []string
	providers := serialGraphRuntimeProviderMap()
	nodes := make(map[string]SerialGraphNodeSpec, len(req.Nodes))
	for _, node := range req.Nodes {
		if node.ID == "" {
			errs = append(errs, "node id must not be empty")
			continue
		}
		if _, exists := nodes[node.ID]; exists {
			errs = append(errs, "duplicate node id: "+node.ID)
			continue
		}
		if _, ok := providers[node.Type]; !ok {
			errs = append(errs, "provider not found: "+node.Type)
		}
		nodes[node.ID] = node
	}
	errs = append(errs, validateGraphRuntimeResources(req.Nodes, providers)...)
	seenEdges := make(map[string]bool, len(req.Edges))
	for _, edge := range req.Edges {
		if edge.ID == "" {
			errs = append(errs, "edge id must not be empty")
			continue
		}
		if seenEdges[edge.ID] {
			errs = append(errs, "duplicate edge id: "+edge.ID)
			continue
		}
		seenEdges[edge.ID] = true
		errs = append(errs, validateGraphRuntimeEdge(req, edge, nodes, providers)...)
	}
	return errs
}

func validateGraphRuntimeResources(nodes []SerialGraphNodeSpec, providers map[string]serialGraphProviderSpec) []string {
	var errs []string
	used := map[string]string{}
	for _, node := range nodes {
		provider, ok := providers[node.Type]
		if !ok || !provider.ResourceOwner {
			continue
		}
		for _, key := range provider.ResourceKeys {
			value := strings.TrimSpace(fmt.Sprint(node.Config[key]))
			if value == "" || value == "<nil>" {
				continue
			}
			resourceKey := key + ":" + value
			if previous, exists := used[resourceKey]; exists {
				errs = append(errs, fmt.Sprintf("resource port duplicated: %s (%s, %s)", value, previous, node.ID))
			} else {
				used[resourceKey] = node.ID
			}
		}
	}
	return errs
}

func validateGraphRuntimeEdge(
	req SerialGraphStartRequest,
	edge SerialGraphEdgeSpec,
	nodes map[string]SerialGraphNodeSpec,
	providers map[string]serialGraphProviderSpec,
) []string {
	var errs []string
	sourceNode, sourceOK := nodes[edge.Source]
	targetNode, targetOK := nodes[edge.Target]
	if !sourceOK {
		errs = append(errs, edge.ID+": source node not found: "+edge.Source)
	}
	if !targetOK {
		errs = append(errs, edge.ID+": target node not found: "+edge.Target)
	}
	if !sourceOK || !targetOK {
		return errs
	}
	if sourceNode.ID == targetNode.ID {
		errs = append(errs, edge.ID+": node cannot connect to itself")
	}
	sourceProvider, sourceProviderOK := providers[sourceNode.Type]
	targetProvider, targetProviderOK := providers[targetNode.Type]
	if !sourceProviderOK || !targetProviderOK {
		return errs
	}
	sourcePort, sourcePortOK := graphRuntimeOutputPort(sourceProvider, edge.SourceHandle)
	targetPort, targetPortOK := graphRuntimeInputPort(targetProvider, edge.TargetHandle)
	if !sourcePortOK {
		errs = append(errs, edge.ID+": output port not found: "+sourceNode.Type+"."+edge.SourceHandle)
	}
	if !targetPortOK {
		errs = append(errs, edge.ID+": input port not found: "+targetNode.Type+"."+edge.TargetHandle)
	}
	if !sourcePortOK || !targetPortOK {
		return errs
	}
	if sourcePort.Kind != targetPort.Kind {
		errs = append(errs, edge.ID+": incompatible port kinds: "+sourcePort.Kind+" -> "+targetPort.Kind)
	}
	otherEdges := graphRuntimeOtherEdges(req.Edges, edge.ID)
	if !targetPort.Multiple {
		for _, other := range otherEdges {
			if other.Target == edge.Target && other.TargetHandle == edge.TargetHandle {
				errs = append(errs, edge.ID+": input already connected: "+targetNode.ID+"."+targetPort.ID)
				break
			}
		}
	}
	fanOutAllowed := sourcePort.Multiple || sourceProvider.Type == "serial.tap" || sourceProvider.Type == "serial.tee"
	if !fanOutAllowed {
		for _, other := range otherEdges {
			if other.Source == edge.Source && other.SourceHandle == edge.SourceHandle {
				errs = append(errs, edge.ID+": fan-out requires a tap node: "+sourceNode.ID+"."+sourcePort.ID)
				break
			}
		}
	}
	if graphRuntimeCreatesCycle(edge, otherEdges) {
		errs = append(errs, edge.ID+": directed cycle not allowed")
	}
	return errs
}

func serialGraphRuntimeProviders() []serialGraphProviderSpec {
	bytesIn := serialGraphPortSpec{ID: "in", Kind: "bytes", Direction: "input"}
	bytesOut := serialGraphPortSpec{ID: "out", Kind: "bytes", Direction: "output"}
	return []serialGraphProviderSpec{
		{Type: "serial.physical", Inputs: []serialGraphPortSpec{{ID: "tx", Kind: "bytes", Direction: "input"}}, Outputs: []serialGraphPortSpec{{ID: "rx", Kind: "bytes", Direction: "output"}}, ResourceOwner: true, ResourceKeys: []string{"portName"}},
		{Type: "serial.virtual", Inputs: []serialGraphPortSpec{{ID: "tx", Kind: "bytes", Direction: "input"}}, Outputs: []serialGraphPortSpec{{ID: "rx", Kind: "bytes", Direction: "output"}}, ResourceOwner: true, ResourceKeys: []string{"portName"}},
		{Type: "serial.bridge", Inputs: []serialGraphPortSpec{{ID: "a-in", Kind: "bytes", Direction: "input"}, {ID: "b-in", Kind: "bytes", Direction: "input"}}, Outputs: []serialGraphPortSpec{{ID: "a-out", Kind: "bytes", Direction: "output"}, {ID: "b-out", Kind: "bytes", Direction: "output"}}},
		{Type: "serial.monitor", Inputs: []serialGraphPortSpec{bytesIn}, Outputs: []serialGraphPortSpec{}},
		{Type: "serial.tap", Inputs: []serialGraphPortSpec{bytesIn}, Outputs: []serialGraphPortSpec{{ID: "out", Kind: "bytes", Direction: "output", Multiple: true}}},
		{Type: "serial.tee", Inputs: []serialGraphPortSpec{bytesIn}, Outputs: []serialGraphPortSpec{{ID: "out", Kind: "bytes", Direction: "output", Multiple: true}}},
		{Type: "serial.sender", Outputs: []serialGraphPortSpec{bytesOut}},
		{Type: "serial.receiver", Inputs: []serialGraphPortSpec{bytesIn}},
		{Type: "serial.modbus.master", Inputs: []serialGraphPortSpec{{ID: "rx", Kind: "bytes", Direction: "input"}}, Outputs: []serialGraphPortSpec{{ID: "tx", Kind: "bytes", Direction: "output"}}},
		{Type: "serial.modbus.slave", Inputs: []serialGraphPortSpec{{ID: "rx", Kind: "bytes", Direction: "input"}}, Outputs: []serialGraphPortSpec{{ID: "tx", Kind: "bytes", Direction: "output"}}},
		{Type: "serial.fecbus.master", Inputs: []serialGraphPortSpec{{ID: "rx", Kind: "bytes", Direction: "input"}}, Outputs: []serialGraphPortSpec{{ID: "tx", Kind: "bytes", Direction: "output"}}},
		{Type: "serial.fecbus.slave", Inputs: []serialGraphPortSpec{{ID: "rx", Kind: "bytes", Direction: "input"}}, Outputs: []serialGraphPortSpec{{ID: "tx", Kind: "bytes", Direction: "output"}}},
	}
}

func serialGraphRuntimeProviderMap() map[string]serialGraphProviderSpec {
	providers := serialGraphRuntimeProviders()
	out := make(map[string]serialGraphProviderSpec, len(providers))
	for _, provider := range providers {
		out[provider.Type] = provider
	}
	return out
}

func graphRuntimeInputPort(provider serialGraphProviderSpec, id string) (serialGraphPortSpec, bool) {
	for _, port := range provider.Inputs {
		if port.ID == id {
			return port, true
		}
	}
	return serialGraphPortSpec{}, false
}

func graphRuntimeOutputPort(provider serialGraphProviderSpec, id string) (serialGraphPortSpec, bool) {
	for _, port := range provider.Outputs {
		if port.ID == id {
			return port, true
		}
	}
	return serialGraphPortSpec{}, false
}

func graphRuntimeOtherEdges(edges []SerialGraphEdgeSpec, id string) []SerialGraphEdgeSpec {
	out := make([]SerialGraphEdgeSpec, 0, len(edges))
	for _, edge := range edges {
		if edge.ID != id {
			out = append(out, edge)
		}
	}
	return out
}

func graphRuntimeCreatesCycle(draft SerialGraphEdgeSpec, edges []SerialGraphEdgeSpec) bool {
	visited := map[string]bool{}
	stack := []string{draft.Target}
	for len(stack) > 0 {
		nodeID := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if visited[nodeID] {
			continue
		}
		if nodeID == draft.Source {
			return true
		}
		visited[nodeID] = true
		for _, edge := range edges {
			if edge.Source == nodeID {
				stack = append(stack, edge.Target)
			}
		}
	}
	return false
}

func graphStringConfig(config map[string]any, key string) string {
	if config == nil {
		return ""
	}
	value, ok := config[key]
	if !ok || value == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

func graphStringConfigRaw(config map[string]any, key string) string {
	if config == nil {
		return ""
	}
	value, ok := config[key]
	if !ok || value == nil {
		return ""
	}
	return fmt.Sprint(value)
}

func graphIntConfig(config map[string]any, key string, fallback int) int {
	if config == nil {
		return fallback
	}
	switch value := config[key].(type) {
	case int:
		return value
	case int64:
		return int(value)
	case float64:
		return int(value)
	case float32:
		return int(value)
	case string:
		var parsed int
		if _, err := fmt.Sscanf(value, "%d", &parsed); err == nil {
			return parsed
		}
	}
	return fallback
}

func graphBoolConfig(config map[string]any, key string, fallback bool) bool {
	if config == nil {
		return fallback
	}
	switch value := config[key].(type) {
	case bool:
		return value
	case string:
		parsed, err := strconv.ParseBool(value)
		if err == nil {
			return parsed
		}
	}
	return fallback
}

func graphSerialConfig(config map[string]any) port.SerialConfig {
	return port.SerialConfig{
		PortName:  graphStringConfig(config, "portName"),
		BaudRate:  graphIntConfig(config, "baudRate", 115200),
		DataBits:  graphIntConfig(config, "dataBits", 8),
		StopBits:  graphStringConfigWithDefault(config, "stopBits", "1"),
		Parity:    graphStringConfigWithDefault(config, "parity", "none"),
		FlowMode:  graphStringConfigWithDefault(config, "flowMode", "none"),
		ReadBufKB: graphIntConfig(config, "readBufKB", 32),
	}
}

func graphStringConfigWithDefault(config map[string]any, key string, fallback string) string {
	value := graphStringConfig(config, key)
	if value == "" || value == "<nil>" {
		return fallback
	}
	return value
}
