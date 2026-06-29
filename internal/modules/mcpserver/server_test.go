package mcpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/littepointR/portweave/internal/core/config"
	"github.com/littepointR/portweave/internal/core/eventbus"
	"github.com/littepointR/portweave/internal/core/logging"
	"github.com/littepointR/portweave/internal/core/module"
	"github.com/littepointR/portweave/internal/core/platform"
	"github.com/littepointR/portweave/internal/modules/serial"
	"github.com/littepointR/portweave/internal/modules/serial/buffer"
	fb "github.com/littepointR/portweave/internal/modules/serial/fecbus"
	"github.com/littepointR/portweave/internal/modules/serial/manager"
	mb "github.com/littepointR/portweave/internal/modules/serial/modbus"
	"github.com/littepointR/portweave/internal/modules/serial/monitor"
	"github.com/littepointR/portweave/internal/modules/serial/port"
)

type fakeSerialService struct {
	ports                 []port.PortInfo
	handles               []manager.HandleStatus
	closePortID           string
	lastSend              serial.SendRequest
	resetPortID           string
	queryPortID           string
	queryOffset           int64
	queryLength           int
	createVirtualID       string
	createVirtualPortName string
	deleteVirtualPortID   string
	virtualPorts          []serial.VirtualPortInfo
	createPairID          string
	createPairPort1       string
	createPairPort2       string
	deleteVirtualPairID   string
	virtualPairs          []serial.VirtualPairInfo
	createBridgeID        string
	createBridgePort1     string
	createBridgePort2     string
	createBridgeBaud      int
	deleteBridgeID        string
	bridges               []serial.BridgeInfo
	cleanupVirtualCalled  bool
	monitors              []monitor.SessionInfo
	startMonitor          serial.AutoVirtualMonitorRequest
	stopMonitor           string
	deleteMonitor         string
	clearMonitor          string
	queryMonitor          monitor.QueryRequest
	modbusSessions        []mb.SessionInfo
	openModbus            mb.OpenSessionRequest
	closeModbus           string
	masterRequest         mb.MasterRequest
	startSlave            mb.StartSlaveRequest
	stopSlave             string
	updateSlaveID         string
	updateSlaveData       mb.DataModelSnapshot
	addSlaveUnit          mb.SlaveUnitSnapshot
	addSlaveID            string
	removeSlaveID         string
	removeUnitID          byte
	listSlaveID           string
	updateUnitID          byte
	updateUnitData        mb.DataModelSnapshot
	unitScan              mb.UnitScanRequest
	registerRead          mb.RegisterReadRequest
	registerScan          mb.RegisterScanRequest
	fecbusSessions        []fb.SessionInfo
	openFecbus            fb.OpenSessionRequest
	closeFecbus           string
	fecbusRequest         fb.SendRequest
	startFecbus           fb.StartSlaveRequest
	stopFecbus            string
	updateFecbusID        string
	updateFecbus          fb.SlaveState
	addFecbusID           string
	addFecbusUnit         fb.SlaveUnitState
	removeFecbusID        string
	removeFecbusAdr       byte
	listFecbusID          string
	queryFecbus           fb.QueryRequest
	clearFecbus           string
	graphs                []serial.SerialGraphRuntimeInfo
	startGraph            serial.SerialGraphStartRequest
	stopGraph             string
	statusGraph           string
	graphSend             serial.SerialGraphSendRequest
	graphBuffer           serial.SerialGraphBufferQuery
	graphFrames           serial.SerialGraphFrameQuery
	clearGraphID          string
	clearGraphNode        string
	resetGraphID          string
	resetGraphNode        string
}

func (f *fakeSerialService) EnumeratePorts(context.Context) ([]port.PortInfo, error) {
	return f.ports, nil
}

func (f *fakeSerialService) OpenPort(_ context.Context, req manager.OpenRequest) (*manager.HandleStatus, error) {
	status := &manager.HandleStatus{ID: "port-1", Config: req.Config, IsOpen: true}
	f.handles = append(f.handles, *status)
	return status, nil
}

func (f *fakeSerialService) ClosePort(id string) error {
	f.closePortID = id
	return nil
}
func (f *fakeSerialService) ListPorts() []manager.HandleStatus {
	return f.handles
}

func (f *fakeSerialService) QueryPage(portID string, offset int64, length int) (*buffer.Snapshot, error) {
	f.queryPortID = portID
	f.queryOffset = offset
	f.queryLength = length
	return &buffer.Snapshot{Offset: offset, Data: []byte("ok"), Total: 2}, nil
}

func (f *fakeSerialService) Send(req serial.SendRequest) (int, error) {
	f.lastSend = req
	return 2, nil
}

func (f *fakeSerialService) ResetCounters(portID string) error {
	f.resetPortID = portID
	return nil
}

func (f *fakeSerialService) CreateVirtualPort(_ context.Context, id string, portName string) (*serial.VirtualPortInfo, error) {
	f.createVirtualID = id
	f.createVirtualPortName = portName
	return &serial.VirtualPortInfo{ID: "v1", Port: "/tmp/ttyV1"}, nil
}

func (f *fakeSerialService) DeleteVirtualPort(id string) error {
	f.deleteVirtualPortID = id
	return nil
}
func (f *fakeSerialService) ListVirtualPorts() []serial.VirtualPortInfo {
	return f.virtualPorts
}

func (f *fakeSerialService) CreateVirtualPair(_ context.Context, id string, port1Name string, port2Name string) (*serial.VirtualPairInfo, error) {
	f.createPairID = id
	f.createPairPort1 = port1Name
	f.createPairPort2 = port2Name
	return &serial.VirtualPairInfo{ID: "pair1", Port1: "/tmp/a", Port2: "/tmp/b"}, nil
}

func (f *fakeSerialService) DeleteVirtualPair(id string) error {
	f.deleteVirtualPairID = id
	return nil
}
func (f *fakeSerialService) ListVirtualPairs() []serial.VirtualPairInfo {
	return f.virtualPairs
}

func (f *fakeSerialService) CreateBridge(id string, port1 string, port2 string, baudRate int) (*serial.BridgeInfo, error) {
	f.createBridgeID = id
	f.createBridgePort1 = port1
	f.createBridgePort2 = port2
	f.createBridgeBaud = baudRate
	return &serial.BridgeInfo{ID: "bridge1", Port1: "/tmp/a", Port2: "/tmp/b", BaudRate: 115200}, nil
}

func (f *fakeSerialService) DeleteBridge(id string) error {
	f.deleteBridgeID = id
	return nil
}
func (f *fakeSerialService) ListBridges() []serial.BridgeInfo {
	return f.bridges
}

func (f *fakeSerialService) CleanupVirtual() { f.cleanupVirtualCalled = true }

func (f *fakeSerialService) StartAutoVirtualMonitor(_ context.Context, req serial.AutoVirtualMonitorRequest) (*monitor.SessionInfo, error) {
	f.startMonitor = req
	session := &monitor.SessionInfo{
		ID:                req.ID,
		Name:              req.Name,
		Provider:          monitor.ProviderBridge,
		PortA:             req.Port,
		PortB:             "/tmp/portweave-monitor",
		ExternalPort:      "/tmp/portweave-monitor",
		AutoVirtualPortID: "auto-monitor",
		Config:            req.Config,
		Encoding:          req.Encoding,
		Status:            monitor.StatusRunning,
	}
	f.monitors = append(f.monitors, *session)
	return session, nil
}

func (f *fakeSerialService) StopMonitor(id string) error {
	f.stopMonitor = id
	return nil
}

func (f *fakeSerialService) DeleteMonitor(id string) error {
	f.deleteMonitor = id
	return nil
}

func (f *fakeSerialService) ListMonitors() []monitor.SessionInfo {
	return f.monitors
}

func (f *fakeSerialService) QueryMonitorFrames(req monitor.QueryRequest) (*monitor.FramePage, error) {
	f.queryMonitor = req
	return &monitor.FramePage{
		Frames: []monitor.Frame{{
			Seq:         1,
			Direction:   monitor.DirectionAToB,
			Port:        "/tmp/source",
			Length:      2,
			DisplayText: "OK",
			DisplayHex:  "4f 4b",
			Encoding:    "utf-8",
		}},
		Total:      1,
		NextOffset: 1,
	}, nil
}

func (f *fakeSerialService) ClearMonitorFrames(id string) error {
	f.clearMonitor = id
	return nil
}

func (f *fakeSerialService) OpenModbusSession(_ context.Context, req mb.OpenSessionRequest) (*mb.SessionInfo, error) {
	f.openModbus = req
	info := mb.SessionInfo{ID: req.ID, Name: req.Name, Mode: req.Mode, Role: req.Role, Config: req.Config, Status: mb.SessionStatusOpen, UnitID: 1}
	f.modbusSessions = append(f.modbusSessions, info)
	return &info, nil
}

func (f *fakeSerialService) CloseModbusSession(id string) error {
	f.closeModbus = id
	return nil
}

func (f *fakeSerialService) ListModbusSessions() []mb.SessionInfo {
	return f.modbusSessions
}

func (f *fakeSerialService) ModbusMasterRequest(req mb.MasterRequest) (*mb.Transaction, error) {
	f.masterRequest = req
	return &mb.Transaction{ID: "tx-1", SessionID: req.SessionID, UnitID: req.UnitID}, nil
}

func (f *fakeSerialService) StartModbusSlave(req mb.StartSlaveRequest) (*mb.SessionInfo, error) {
	f.startSlave = req
	info := mb.SessionInfo{ID: req.SessionID, Role: mb.SessionRoleSlave, Status: mb.SessionStatusRunning, SlaveRunning: true, UnitID: req.UnitID}
	return &info, nil
}

func (f *fakeSerialService) StopModbusSlave(id string) error {
	f.stopSlave = id
	return nil
}

func (f *fakeSerialService) UpdateModbusSlaveData(sessionID string, data mb.DataModelSnapshot) error {
	f.updateSlaveID = sessionID
	f.updateSlaveData = data
	return nil
}

func (f *fakeSerialService) AddModbusSlaveUnit(sessionID string, unit mb.SlaveUnitSnapshot) error {
	f.addSlaveID = sessionID
	f.addSlaveUnit = unit
	return nil
}

func (f *fakeSerialService) RemoveModbusSlaveUnit(sessionID string, unitID byte) error {
	f.removeSlaveID = sessionID
	f.removeUnitID = unitID
	return nil
}

func (f *fakeSerialService) ListModbusSlaveUnits(sessionID string) ([]mb.SlaveUnitInfo, error) {
	f.listSlaveID = sessionID
	return []mb.SlaveUnitInfo{{UnitID: 2}}, nil
}

func (f *fakeSerialService) UpdateModbusSlaveUnitData(sessionID string, unitID byte, data mb.DataModelSnapshot) error {
	f.updateUnitID = unitID
	f.updateUnitData = data
	return nil
}

func (f *fakeSerialService) ModbusScanUnitIDs(req mb.UnitScanRequest) (*mb.UnitScanResult, error) {
	f.unitScan = req
	return &mb.UnitScanResult{SessionID: req.SessionID, ActiveUnitIDs: []int{int(req.UnitIDs[0])}}, nil
}

func (f *fakeSerialService) ModbusReadRegisters(req mb.RegisterReadRequest) (*mb.RegisterReadResult, error) {
	f.registerRead = req
	return &mb.RegisterReadResult{RawRegisters: []uint16{42}}, nil
}

func (f *fakeSerialService) ModbusScanRegisters(req mb.RegisterScanRequest) (*mb.RegisterScanResult, error) {
	f.registerScan = req
	return &mb.RegisterScanResult{SessionID: req.SessionID, UnitID: req.UnitID, Values: []mb.RegisterScanValue{{Address: req.StartAddress, Value: 42}}}, nil
}

func (f *fakeSerialService) OpenFecbusSession(_ context.Context, req fb.OpenSessionRequest) (*fb.SessionInfo, error) {
	f.openFecbus = req
	info := fb.SessionInfo{ID: req.ID, Name: req.Name, Role: req.Role, Config: req.Config, Status: fb.SessionStatusOpen}
	f.fecbusSessions = append(f.fecbusSessions, info)
	return &info, nil
}

func (f *fakeSerialService) CloseFecbusSession(id string) error {
	f.closeFecbus = id
	return nil
}

func (f *fakeSerialService) ListFecbusSessions() []fb.SessionInfo {
	return f.fecbusSessions
}

func (f *fakeSerialService) FecbusSendRequest(req fb.SendRequest) (*fb.Transaction, error) {
	f.fecbusRequest = req
	response := fb.Frame{
		Type:          fb.FrameTypeAnswer,
		TargetAddress: req.SourceAddress,
		Priority:      3,
		SourceAddress: req.TargetAddress,
		MessageNumber: req.MessageNumber,
		Data:          []byte{byte(req.Function), byte(fb.StatusReceivedOK)},
	}
	return &fb.Transaction{ID: "fec-tx-1", SessionID: req.SessionID, Response: &response}, nil
}

func (f *fakeSerialService) StartFecbusSlave(req fb.StartSlaveRequest) (*fb.SessionInfo, error) {
	f.startFecbus = req
	targetAddress := req.State.Address
	if len(req.Units) > 0 {
		targetAddress = req.Units[0].Address
	}
	info := fb.SessionInfo{ID: req.SessionID, Role: fb.SessionRoleSlave, Status: fb.SessionStatusRunning, SlaveRunning: true, TargetAddress: targetAddress}
	return &info, nil
}

func (f *fakeSerialService) StopFecbusSlave(id string) error {
	f.stopFecbus = id
	return nil
}

func (f *fakeSerialService) UpdateFecbusSlaveState(sessionID string, state fb.SlaveState) error {
	f.updateFecbusID = sessionID
	f.updateFecbus = state
	return nil
}

func (f *fakeSerialService) AddFecbusSlaveUnit(sessionID string, unit fb.SlaveUnitState) error {
	f.addFecbusID = sessionID
	f.addFecbusUnit = unit
	return nil
}

func (f *fakeSerialService) RemoveFecbusSlaveUnit(sessionID string, address byte) error {
	f.removeFecbusID = sessionID
	f.removeFecbusAdr = address
	return nil
}

func (f *fakeSerialService) ListFecbusSlaveUnits(sessionID string) ([]fb.SlaveUnitInfo, error) {
	f.listFecbusID = sessionID
	return []fb.SlaveUnitInfo{{Address: 2, DefaultStatus: fb.StatusReceivedOK}}, nil
}

func (f *fakeSerialService) QueryFecbusFrames(req fb.QueryRequest) (*fb.FramePage, error) {
	f.queryFecbus = req
	return &fb.FramePage{
		Frames: []fb.FrameRecord{{
			Seq:       1,
			SessionID: req.SessionID,
			Direction: "tx",
			Hex:       "7e 00",
		}},
		Total: 1,
		Limit: req.Limit,
	}, nil
}

func (f *fakeSerialService) ClearFecbusFrames(id string) error {
	f.clearFecbus = id
	return nil
}

func (f *fakeSerialService) StartSerialGraph(_ context.Context, req serial.SerialGraphStartRequest) (*serial.SerialGraphRuntimeInfo, error) {
	f.startGraph = req
	info := serial.SerialGraphRuntimeInfo{
		ID:     req.ID,
		Status: serial.SerialGraphStatusRunning,
		Nodes:  []serial.SerialGraphNodeStatus{{ID: "virtual", Type: "serial.virtual", Status: serial.SerialGraphStatusRunning}},
	}
	f.graphs = []serial.SerialGraphRuntimeInfo{info}
	return &info, nil
}

func (f *fakeSerialService) StopSerialGraph(id string) error {
	f.stopGraph = id
	return nil
}

func (f *fakeSerialService) ListSerialGraphs() []serial.SerialGraphRuntimeInfo {
	return f.graphs
}

func (f *fakeSerialService) GetSerialGraphStatus(id string) (*serial.SerialGraphRuntimeInfo, error) {
	f.statusGraph = id
	return &serial.SerialGraphRuntimeInfo{ID: id, Status: serial.SerialGraphStatusRunning}, nil
}

func (f *fakeSerialService) SendSerialGraphNode(req serial.SerialGraphSendRequest) (int, error) {
	f.graphSend = req
	return len(req.Content), nil
}

func (f *fakeSerialService) QuerySerialGraphNodeBuffer(req serial.SerialGraphBufferQuery) (*buffer.Snapshot, error) {
	f.graphBuffer = req
	return &buffer.Snapshot{Offset: req.Offset, Data: []byte("hello"), Total: 5}, nil
}

func (f *fakeSerialService) QuerySerialGraphNodeFrames(req serial.SerialGraphFrameQuery) (*serial.SerialGraphFramePage, error) {
	f.graphFrames = req
	return &serial.SerialGraphFramePage{
		Frames: []monitor.Frame{{Seq: 1, Direction: "接收", Length: 5, DisplayText: "hello"}},
		Total:  1,
	}, nil
}

func (f *fakeSerialService) ClearSerialGraphNodeBuffer(graphID string, nodeID string) error {
	f.clearGraphID = graphID
	f.clearGraphNode = nodeID
	return nil
}

func (f *fakeSerialService) ResetSerialGraphNodeCounters(graphID string, nodeID string) error {
	f.resetGraphID = graphID
	f.resetGraphNode = nodeID
	return nil
}

type failingSerialService struct {
	fakeSerialService
	err error
}

func (f *failingSerialService) EnumeratePorts(context.Context) ([]port.PortInfo, error) {
	return nil, f.err
}

func (f *failingSerialService) OpenPort(context.Context, manager.OpenRequest) (*manager.HandleStatus, error) {
	return nil, f.err
}

func (f *failingSerialService) ClosePort(string) error { return f.err }

func (f *failingSerialService) QueryPage(string, int64, int) (*buffer.Snapshot, error) {
	return nil, f.err
}

func (f *failingSerialService) Send(serial.SendRequest) (int, error) { return 0, f.err }

func (f *failingSerialService) ResetCounters(string) error { return f.err }

func (f *failingSerialService) CreateVirtualPort(context.Context, string, string) (*serial.VirtualPortInfo, error) {
	return nil, f.err
}

func (f *failingSerialService) DeleteVirtualPort(string) error { return f.err }

func (f *failingSerialService) CreateVirtualPair(context.Context, string, string, string) (*serial.VirtualPairInfo, error) {
	return nil, f.err
}

func (f *failingSerialService) DeleteVirtualPair(string) error { return f.err }

func (f *failingSerialService) CreateBridge(string, string, string, int) (*serial.BridgeInfo, error) {
	return nil, f.err
}

func (f *failingSerialService) DeleteBridge(string) error { return f.err }

func (f *failingSerialService) StartAutoVirtualMonitor(context.Context, serial.AutoVirtualMonitorRequest) (*monitor.SessionInfo, error) {
	return nil, f.err
}

func (f *failingSerialService) StopMonitor(string) error { return f.err }

func (f *failingSerialService) DeleteMonitor(string) error { return f.err }

func (f *failingSerialService) QueryMonitorFrames(monitor.QueryRequest) (*monitor.FramePage, error) {
	return nil, f.err
}

func (f *failingSerialService) ClearMonitorFrames(string) error { return f.err }

func (f *failingSerialService) OpenModbusSession(context.Context, mb.OpenSessionRequest) (*mb.SessionInfo, error) {
	return nil, f.err
}

func (f *failingSerialService) CloseModbusSession(string) error { return f.err }

func (f *failingSerialService) ModbusMasterRequest(mb.MasterRequest) (*mb.Transaction, error) {
	return nil, f.err
}

func (f *failingSerialService) StartModbusSlave(mb.StartSlaveRequest) (*mb.SessionInfo, error) {
	return nil, f.err
}

func (f *failingSerialService) StopModbusSlave(string) error { return f.err }

func (f *failingSerialService) UpdateModbusSlaveData(string, mb.DataModelSnapshot) error {
	return f.err
}

func (f *failingSerialService) AddModbusSlaveUnit(string, mb.SlaveUnitSnapshot) error {
	return f.err
}

func (f *failingSerialService) RemoveModbusSlaveUnit(string, byte) error { return f.err }

func (f *failingSerialService) ListModbusSlaveUnits(string) ([]mb.SlaveUnitInfo, error) {
	return nil, f.err
}

func (f *failingSerialService) UpdateModbusSlaveUnitData(string, byte, mb.DataModelSnapshot) error {
	return f.err
}

func (f *failingSerialService) ModbusScanUnitIDs(mb.UnitScanRequest) (*mb.UnitScanResult, error) {
	return nil, f.err
}

func (f *failingSerialService) ModbusReadRegisters(mb.RegisterReadRequest) (*mb.RegisterReadResult, error) {
	return nil, f.err
}

func (f *failingSerialService) ModbusScanRegisters(mb.RegisterScanRequest) (*mb.RegisterScanResult, error) {
	return nil, f.err
}

func (f *failingSerialService) OpenFecbusSession(context.Context, fb.OpenSessionRequest) (*fb.SessionInfo, error) {
	return nil, f.err
}

func (f *failingSerialService) CloseFecbusSession(string) error { return f.err }

func (f *failingSerialService) FecbusSendRequest(fb.SendRequest) (*fb.Transaction, error) {
	return nil, f.err
}

func (f *failingSerialService) StartFecbusSlave(fb.StartSlaveRequest) (*fb.SessionInfo, error) {
	return nil, f.err
}

func (f *failingSerialService) StopFecbusSlave(string) error { return f.err }

func (f *failingSerialService) UpdateFecbusSlaveState(string, fb.SlaveState) error {
	return f.err
}

func (f *failingSerialService) AddFecbusSlaveUnit(string, fb.SlaveUnitState) error {
	return f.err
}

func (f *failingSerialService) RemoveFecbusSlaveUnit(string, byte) error { return f.err }

func (f *failingSerialService) ListFecbusSlaveUnits(string) ([]fb.SlaveUnitInfo, error) {
	return nil, f.err
}

func (f *failingSerialService) QueryFecbusFrames(fb.QueryRequest) (*fb.FramePage, error) {
	return nil, f.err
}

func (f *failingSerialService) ClearFecbusFrames(string) error { return f.err }

func (f *failingSerialService) StartSerialGraph(context.Context, serial.SerialGraphStartRequest) (*serial.SerialGraphRuntimeInfo, error) {
	return nil, f.err
}

func (f *failingSerialService) StopSerialGraph(string) error { return f.err }

func (f *failingSerialService) GetSerialGraphStatus(string) (*serial.SerialGraphRuntimeInfo, error) {
	return nil, f.err
}

func (f *failingSerialService) SendSerialGraphNode(serial.SerialGraphSendRequest) (int, error) {
	return 0, f.err
}

func (f *failingSerialService) QuerySerialGraphNodeBuffer(serial.SerialGraphBufferQuery) (*buffer.Snapshot, error) {
	return nil, f.err
}

func (f *failingSerialService) QuerySerialGraphNodeFrames(serial.SerialGraphFrameQuery) (*serial.SerialGraphFramePage, error) {
	return nil, f.err
}

func (f *failingSerialService) ClearSerialGraphNodeBuffer(string, string) error { return f.err }

func (f *failingSerialService) ResetSerialGraphNodeCounters(string, string) error { return f.err }

func TestOriginGuardAllowsMissingAndLocalOrigins(t *testing.T) {
	t.Parallel()
	handler := originGuard(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}), config.MCPConfig{Host: "127.0.0.1", Port: 39391, AllowLocalOrigins: true})

	for _, origin := range []string{"", "http://127.0.0.1:39391", "http://localhost:39391"} {
		req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:39391/mcp", nil)
		if origin != "" {
			req.Header.Set("Origin", origin)
		}
		resp := httptest.NewRecorder()
		handler.ServeHTTP(resp, req)
		if resp.Code != http.StatusNoContent {
			t.Fatalf("origin %q status = %d, want %d", origin, resp.Code, http.StatusNoContent)
		}
	}
}

func TestOriginGuardRejectsRemoteOrigins(t *testing.T) {
	t.Parallel()
	handler := originGuard(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}), config.MCPConfig{Host: "127.0.0.1", Port: 39391, AllowLocalOrigins: true})

	for _, origin := range []string{
		"https://example.com",
		"http://localhost:39392",
		"file://localhost:39391",
		"http://localhost",
	} {
		req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:39391/mcp", nil)
		req.Header.Set("Origin", origin)
		resp := httptest.NewRecorder()
		handler.ServeHTTP(resp, req)
		if resp.Code != http.StatusForbidden {
			t.Fatalf("origin %q status = %d, want %d", origin, resp.Code, http.StatusForbidden)
		}
	}
}

func TestOriginGuardHonorsLocalOriginAllowance(t *testing.T) {
	t.Parallel()
	handler := originGuard(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}), config.MCPConfig{Host: "127.0.0.1", Port: 39391, AllowLocalOrigins: false})

	allowed := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:39391/mcp", nil)
	allowed.Header.Set("Origin", "http://127.0.0.1:39391")
	allowedResp := httptest.NewRecorder()
	handler.ServeHTTP(allowedResp, allowed)
	if allowedResp.Code != http.StatusNoContent {
		t.Fatalf("configured host origin status = %d, want %d", allowedResp.Code, http.StatusNoContent)
	}

	rejected := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:39391/mcp", nil)
	rejected.Header.Set("Origin", "http://localhost:39391")
	rejectedResp := httptest.NewRecorder()
	handler.ServeHTTP(rejectedResp, rejected)
	if rejectedResp.Code != http.StatusForbidden {
		t.Fatalf("localhost origin with AllowLocalOrigins=false status = %d, want %d", rejectedResp.Code, http.StatusForbidden)
	}
}

func TestListenAddressRejectsNonLocalHosts(t *testing.T) {
	t.Parallel()
	for _, host := range []string{"0.0.0.0", "::", "192.168.1.10", "example.com"} {
		if _, err := listenAddress(config.MCPConfig{Host: host, Port: 39391}); err == nil {
			t.Fatalf("listenAddress(%q) error = nil, want non-local host rejected", host)
		}
	}
}

func TestListenAddressAcceptsLoopbackAndRejectsInvalidPorts(t *testing.T) {
	t.Parallel()
	address, err := listenAddress(config.MCPConfig{Host: "localhost", Port: 39391})
	if err != nil {
		t.Fatalf("listenAddress localhost: %v", err)
	}
	if address != "localhost:39391" {
		t.Fatalf("listenAddress localhost = %q, want localhost:39391", address)
	}
	for _, port := range []int{0, -1, 65536} {
		if _, err := listenAddress(config.MCPConfig{Host: "127.0.0.1", Port: port}); err == nil {
			t.Fatalf("listenAddress port %d error = nil, want invalid port", port)
		}
	}
}

func TestNormalizedPathAddsDefaultAndLeadingSlash(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		path string
		want string
	}{
		{path: "", want: "/mcp"},
		{path: "mcp", want: "/mcp"},
		{path: "/custom", want: "/custom"},
	} {
		if got := normalizedPath(tc.path); got != tc.want {
			t.Fatalf("normalizedPath(%q) = %q, want %q", tc.path, got, tc.want)
		}
	}
}

func TestParseFlexibleIntAcceptsDecimalHexAndInvalidValues(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		value string
		want  int
	}{
		{value: "42", want: 42},
		{value: " 0x2a ", want: 42},
		{value: "0X10", want: 16},
		{value: "not-a-number", want: 0},
		{value: "0xnothex", want: 0},
	} {
		if got := parseFlexibleInt(tc.value); got != tc.want {
			t.Fatalf("parseFlexibleInt(%q) = %d, want %d", tc.value, got, tc.want)
		}
	}
}

func TestRegisterToolsExposesSerialRuntimeTools(t *testing.T) {
	t.Parallel()
	svc := &fakeSerialService{
		ports: []port.PortInfo{{Name: "/tmp/ttyV0", FriendlyName: "ttyV0"}},
		handles: []manager.HandleStatus{{
			ID:     "port-1",
			Config: port.SerialConfig{PortName: "/tmp/ttyV0", BaudRate: 115200},
			IsOpen: true,
		}},
	}
	server := newMCPServer(svc)
	handler := mcpHTTPHandler(server, true)

	tools := callMCP(t, handler, `{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`)
	names := extractToolNames(t, tools)
	for _, want := range []string{
		"serial_enumerate_ports",
		"serial_open_port",
		"serial_send",
		"serial_query_buffer",
		"serial_create_virtual_port",
		"serial_create_bridge",
		"serial_cleanup_virtual",
		"serial_start_monitor",
		"serial_list_monitors",
		"serial_query_monitor_frames",
		"serial_stop_monitor",
		"serial_delete_monitor",
		"serial_clear_monitor_frames",
		"serial_graph_provider_catalog",
		"serial_graph_demo_catalog",
		"serial_graph_demo_template",
		"serial_graph_validate",
		"serial_graph_start",
		"serial_graph_stop",
		"serial_graph_status",
		"serial_graph_send",
		"serial_graph_query_node_buffer",
		"serial_graph_query_node_frames",
		"serial_graph_clear_node_buffer",
		"serial_graph_reset_node_counters",
		"modbus_open_session",
		"modbus_close_session",
		"modbus_list_sessions",
		"modbus_master_request",
		"modbus_read_registers",
		"modbus_scan_unit_ids",
		"modbus_scan_registers",
		"modbus_start_slave",
		"modbus_stop_slave",
		"modbus_update_slave_data",
		"modbus_add_slave_unit",
		"modbus_remove_slave_unit",
		"modbus_update_slave_unit_data",
		"modbus_list_slave_units",
		"fecbus_function_catalog",
		"fecbus_open_session",
		"fecbus_close_session",
		"fecbus_list_sessions",
		"fecbus_send_request",
		"fecbus_start_slave",
		"fecbus_stop_slave",
		"fecbus_update_slave_state",
		"fecbus_query_frames",
		"fecbus_clear_frames",
	} {
		if !names[want] {
			t.Fatalf("tools/list missing %s in %#v", want, names)
		}
	}
}

func TestRegisterToolsExposesDocsCriticalMCPTools(t *testing.T) {
	t.Parallel()
	handler := mcpHTTPHandler(newMCPServer(&fakeSerialService{}), true)

	tools := callMCP(t, handler, `{"jsonrpc":"2.0","id":11,"method":"tools/list","params":{}}`)
	names := extractToolNames(t, tools)
	docsCriticalToolGroups := map[string][]string{
		"protocol templates": {
			"protocol_template_catalog",
			"protocol_template_describe",
		},
		"serial graph": {
			"serial_graph_provider_catalog",
			"serial_graph_demo_catalog",
			"serial_graph_demo_template",
			"serial_graph_validate",
			"serial_graph_start",
			"serial_graph_stop",
			"serial_graph_status",
			"serial_graph_send",
			"serial_graph_query_node_buffer",
			"serial_graph_query_node_frames",
			"serial_graph_clear_node_buffer",
			"serial_graph_reset_node_counters",
		},
		"virtual and monitor": {
			"serial_create_virtual_port",
			"serial_delete_virtual_port",
			"serial_list_virtual_ports",
			"serial_create_virtual_pair",
			"serial_delete_virtual_pair",
			"serial_list_virtual_pairs",
			"serial_create_bridge",
			"serial_delete_bridge",
			"serial_list_bridges",
			"serial_cleanup_virtual",
			"serial_start_monitor",
			"serial_stop_monitor",
			"serial_delete_monitor",
			"serial_list_monitors",
			"serial_query_monitor_frames",
			"serial_clear_monitor_frames",
		},
		"modbus": {
			"modbus_open_session",
			"modbus_close_session",
			"modbus_list_sessions",
			"modbus_master_request",
			"modbus_start_slave",
			"modbus_stop_slave",
			"modbus_update_slave_data",
			"modbus_add_slave_unit",
			"modbus_remove_slave_unit",
			"modbus_list_slave_units",
			"modbus_update_slave_unit_data",
			"modbus_read_registers",
			"modbus_scan_unit_ids",
			"modbus_scan_registers",
		},
		"fecbus": {
			"fecbus_function_catalog",
			"fecbus_open_session",
			"fecbus_close_session",
			"fecbus_list_sessions",
			"fecbus_send_request",
			"fecbus_start_slave",
			"fecbus_stop_slave",
			"fecbus_update_slave_state",
			"fecbus_add_slave_unit",
			"fecbus_remove_slave_unit",
			"fecbus_list_slave_units",
			"fecbus_query_frames",
			"fecbus_clear_frames",
		},
	}

	for group, wants := range docsCriticalToolGroups {
		group, wants := group, wants
		t.Run(group, func(t *testing.T) {
			for _, want := range wants {
				if !names[want] {
					t.Fatalf("docs-critical MCP tool %s missing from tools/list; available tools: %#v", want, names)
				}
			}
		})
	}
}

func TestMCPToolAnnotations(t *testing.T) {
	t.Parallel()
	handler := mcpHTTPHandler(newMCPServer(&fakeSerialService{}), true)

	tools := extractToolsByName(t, callMCP(t, handler, `{"jsonrpc":"2.0","id":12,"method":"tools/list","params":{}}`))
	for _, name := range []string{"protocol_template_catalog", "protocol_template_describe"} {
		annotations := requireToolAnnotations(t, tools, name)
		if !annotationBool(t, annotations, "readOnlyHint", true) {
			t.Fatalf("%s readOnlyHint = %v, want true", name, annotations["readOnlyHint"])
		}
	}

	writeToolDestructiveHints := map[string]bool{
		"serial_open_port":                 false,
		"serial_close_port":                true,
		"serial_send":                      false,
		"serial_reset_counters":            true,
		"serial_create_virtual_port":       false,
		"serial_delete_virtual_port":       true,
		"serial_create_virtual_pair":       false,
		"serial_delete_virtual_pair":       true,
		"serial_create_bridge":             false,
		"serial_delete_bridge":             true,
		"serial_cleanup_virtual":           true,
		"serial_graph_start":               false,
		"serial_graph_stop":                true,
		"serial_graph_send":                false,
		"serial_graph_clear_node_buffer":   true,
		"serial_graph_reset_node_counters": true,
		"serial_start_monitor":             false,
		"serial_stop_monitor":              true,
		"serial_delete_monitor":            true,
		"serial_clear_monitor_frames":      true,
		"modbus_open_session":              false,
		"modbus_close_session":             true,
		"modbus_master_request":            false,
		"modbus_start_slave":               false,
		"modbus_stop_slave":                true,
		"modbus_update_slave_data":         false,
		"modbus_add_slave_unit":            false,
		"modbus_remove_slave_unit":         true,
		"modbus_update_slave_unit_data":    false,
		"fecbus_open_session":              false,
		"fecbus_close_session":             true,
		"fecbus_send_request":              false,
		"fecbus_start_slave":               false,
		"fecbus_stop_slave":                true,
		"fecbus_update_slave_state":        false,
		"fecbus_add_slave_unit":            false,
		"fecbus_remove_slave_unit":         true,
		"fecbus_clear_frames":              true,
	}
	for name, destructive := range writeToolDestructiveHints {
		annotations := requireToolAnnotations(t, tools, name)
		if annotationBool(t, annotations, "readOnlyHint", false) {
			t.Fatalf("%s readOnlyHint = true, want false or omitted", name)
		}
		if got := annotationBool(t, annotations, "destructiveHint", true); got != destructive {
			t.Fatalf("%s destructiveHint = %v, want %v", name, got, destructive)
		}
	}
}

func TestProtocolTemplateMCPToolsExposeReadOnlyCatalog(t *testing.T) {
	t.Parallel()
	handler := mcpHTTPHandler(newMCPServer(&fakeSerialService{}), true)

	catalog := callMCPTool(t, handler, 70, "protocol_template_catalog", map[string]any{})
	templates := catalog["structuredContent"].(map[string]any)["templates"].([]any)
	seen := map[string]bool{}
	for _, raw := range templates {
		item := raw.(map[string]any)
		seen[item["name"].(string)] = true
		if item["description"] == "" || item["kind"] == "" {
			t.Fatalf("catalog item missing metadata: %#v", item)
		}
	}
	for _, want := range []string{"Modbus RTU", "AA55 自定义帧", "NMEA"} {
		if !seen[want] {
			t.Fatalf("protocol_template_catalog missing %q in %#v", want, templates)
		}
	}

	described := callMCPTool(t, handler, 71, "protocol_template_describe", map[string]any{"name": "AA55 自定义帧"})
	template := described["structuredContent"].(map[string]any)["template"].(map[string]any)
	if template["name"] != "AA55 自定义帧" || template["kind"] != "visual" || template["config"] == nil {
		t.Fatalf("protocol_template_describe template = %#v, want AA55 visual template with config", template)
	}
	visual := template["config"].(map[string]any)["visual"].(map[string]any)
	if visual["header_hex"] != "aa55" || visual["checksum_type"] != "sum8" {
		t.Fatalf("AA55 visual summary = %#v, want header_hex aa55 and sum8 checksum", visual)
	}

	missing := callMCPToolAllowError(t, handler, 72, "protocol_template_describe", map[string]any{"name": "not-a-template"})
	if missing["isError"] != true {
		t.Fatalf("missing template result = %#v, want MCP tool error", missing)
	}
}

func TestSerialVirtualLifecycleToolsCallSerialService(t *testing.T) {
	t.Parallel()
	svc := &fakeSerialService{
		handles:      []manager.HandleStatus{{ID: "port-1", Config: port.SerialConfig{PortName: "/tmp/ttyV0"}, IsOpen: true}},
		virtualPorts: []serial.VirtualPortInfo{{ID: "v-listed", Port: "/tmp/listed"}},
		virtualPairs: []serial.VirtualPairInfo{{ID: "pair-listed", Port1: "/tmp/a", Port2: "/tmp/b"}},
		bridges:      []serial.BridgeInfo{{ID: "bridge-listed", Port1: "/tmp/a", Port2: "/tmp/b", BaudRate: 57600}},
	}
	handler := mcpHTTPHandler(newMCPServer(svc), true)

	openPorts := callMCPTool(t, handler, 101, "serial_list_open_ports", map[string]any{})
	if handles := openPorts["structuredContent"].(map[string]any)["handles"].([]any); len(handles) != 1 {
		t.Fatalf("serial_list_open_ports handles = %#v, want one handle", handles)
	}
	callMCPTool(t, handler, 102, "serial_close_port", map[string]any{"port_id": "port-1"})
	callMCPTool(t, handler, 103, "serial_reset_counters", map[string]any{"port_id": "port-1"})
	if svc.closePortID != "port-1" || svc.resetPortID != "port-1" {
		t.Fatalf("serial close/reset ids = %q/%q, want port-1", svc.closePortID, svc.resetPortID)
	}

	createdPort := callMCPTool(t, handler, 104, "serial_create_virtual_port", map[string]any{"id": "v-new", "port_name": "tty-v-new"})
	if createdPort["structuredContent"].(map[string]any)["virtual_port"].(map[string]any)["ID"] != "v1" {
		t.Fatalf("serial_create_virtual_port result = %#v", createdPort["structuredContent"])
	}
	callMCPTool(t, handler, 105, "serial_delete_virtual_port", map[string]any{"id": "v-new"})
	listedPorts := callMCPTool(t, handler, 106, "serial_list_virtual_ports", map[string]any{})
	if ports := listedPorts["structuredContent"].(map[string]any)["virtual_ports"].([]any); len(ports) != 1 {
		t.Fatalf("serial_list_virtual_ports = %#v, want one port", ports)
	}
	if svc.createVirtualID != "v-new" || svc.createVirtualPortName != "tty-v-new" || svc.deleteVirtualPortID != "v-new" {
		t.Fatalf("virtual port args = create:%q/%q delete:%q", svc.createVirtualID, svc.createVirtualPortName, svc.deleteVirtualPortID)
	}

	createdPair := callMCPTool(t, handler, 107, "serial_create_virtual_pair", map[string]any{"id": "pair-new", "port1_name": "tty-a", "port2_name": "tty-b"})
	if createdPair["structuredContent"].(map[string]any)["virtual_pair"].(map[string]any)["ID"] != "pair1" {
		t.Fatalf("serial_create_virtual_pair result = %#v", createdPair["structuredContent"])
	}
	callMCPTool(t, handler, 108, "serial_delete_virtual_pair", map[string]any{"id": "pair-new"})
	listedPairs := callMCPTool(t, handler, 109, "serial_list_virtual_pairs", map[string]any{})
	if pairs := listedPairs["structuredContent"].(map[string]any)["virtual_pairs"].([]any); len(pairs) != 1 {
		t.Fatalf("serial_list_virtual_pairs = %#v, want one pair", pairs)
	}
	if svc.createPairID != "pair-new" || svc.createPairPort1 != "tty-a" || svc.createPairPort2 != "tty-b" || svc.deleteVirtualPairID != "pair-new" {
		t.Fatalf("virtual pair args = create:%q/%q/%q delete:%q", svc.createPairID, svc.createPairPort1, svc.createPairPort2, svc.deleteVirtualPairID)
	}

	createdBridge := callMCPTool(t, handler, 110, "serial_create_bridge", map[string]any{"id": "bridge-new", "port1": "tty-a", "port2": "tty-b", "baud_rate": 57600})
	if createdBridge["structuredContent"].(map[string]any)["bridge"].(map[string]any)["ID"] != "bridge1" {
		t.Fatalf("serial_create_bridge result = %#v", createdBridge["structuredContent"])
	}
	callMCPTool(t, handler, 111, "serial_delete_bridge", map[string]any{"id": "bridge-new"})
	listedBridges := callMCPTool(t, handler, 112, "serial_list_bridges", map[string]any{})
	if bridges := listedBridges["structuredContent"].(map[string]any)["bridges"].([]any); len(bridges) != 1 {
		t.Fatalf("serial_list_bridges = %#v, want one bridge", bridges)
	}
	callMCPTool(t, handler, 113, "serial_cleanup_virtual", map[string]any{})
	if svc.createBridgeID != "bridge-new" || svc.createBridgePort1 != "tty-a" || svc.createBridgePort2 != "tty-b" || svc.createBridgeBaud != 57600 || svc.deleteBridgeID != "bridge-new" || !svc.cleanupVirtualCalled {
		t.Fatalf("bridge/cleanup args = create:%q/%q/%q/%d delete:%q cleanup:%v", svc.createBridgeID, svc.createBridgePort1, svc.createBridgePort2, svc.createBridgeBaud, svc.deleteBridgeID, svc.cleanupVirtualCalled)
	}
}

func TestSerialGraphToolsExposeCatalogAndValidateTopology(t *testing.T) {
	t.Parallel()
	handler := mcpHTTPHandler(newMCPServer(&fakeSerialService{}), true)

	catalog := callMCP(t, handler, `{
		"jsonrpc":"2.0",
		"id":40,
		"method":"tools/call",
		"params":{"name":"serial_graph_provider_catalog","arguments":{}}
	}`)
	catalogContent := catalog["structuredContent"].(map[string]any)
	providers := catalogContent["providers"].([]any)
	rules := catalogContent["rules"].(map[string]any)
	if !strings.Contains(fmt.Sprint(rules["resource_owners"]), "serial.remote") || !strings.Contains(fmt.Sprint(rules["resource_owners"]), "normalized raw TCP endpoint + role") {
		t.Fatalf("resource owner rules = %#v, want remote endpoint + role guidance", rules["resource_owners"])
	}
	if !strings.Contains(fmt.Sprint(rules["remote_security"]), "raw TCP") || !strings.Contains(fmt.Sprint(rules["remote_security"]), "trusted") {
		t.Fatalf("remote security rules = %#v, want raw TCP trusted-network guidance", rules["remote_security"])
	}
	if len(providers) < 12 {
		t.Fatalf("provider count = %d, want at least 12", len(providers))
	}
	providerTypes := make(map[string]bool, len(providers))
	providerSpecs := make(map[string]map[string]any, len(providers))
	providerDefaults := make(map[string]map[string]any, len(providers))
	for _, raw := range providers {
		provider := raw.(map[string]any)
		if _, ok := provider["inputs"].([]any); !ok {
			t.Fatalf("provider %s inputs = %#v, want array", provider["type"], provider["inputs"])
		}
		if _, ok := provider["outputs"].([]any); !ok {
			t.Fatalf("provider %s outputs = %#v, want array", provider["type"], provider["outputs"])
		}
		providerType := provider["type"].(string)
		providerTypes[providerType] = true
		providerSpecs[providerType] = provider
		providerDefaults[providerType] = provider["default_config"].(map[string]any)
	}
	for _, want := range []string{
		"serial.physical",
		"serial.virtual",
		"serial.filter",
		"serial.remote",
		"serial.modbus.master",
		"serial.fecbus.slave",
		"serial.script.transform",
		"serial.script.generator",
		"serial.script.analyzer",
	} {
		if !providerTypes[want] {
			t.Fatalf("catalog missing %s in %#v", want, providerTypes)
		}
	}
	for _, removed := range []string{"serial.sender", "serial.receiver", "serial.tap", "serial.tee"} {
		if providerTypes[removed] {
			t.Fatalf("catalog unexpectedly exposes removed provider %s", removed)
		}
	}
	if len(providerDefaults["serial.bridge"]) != 0 {
		t.Fatalf("serial.bridge default_config = %#v, want empty", providerDefaults["serial.bridge"])
	}
	if _, ok := providerDefaults["serial.modbus.slave"]["addressMode"]; ok {
		t.Fatalf("serial.modbus.slave default_config = %#v, want no addressMode", providerDefaults["serial.modbus.slave"])
	}
	if got := providerDefaults["serial.modbus.slave"]["unitIds"]; got != "1" {
		t.Fatalf("serial.modbus.slave unitIds default = %#v, want 1", got)
	}
	if got := providerDefaults["serial.virtual"]["portName"]; got != "" {
		t.Fatalf("serial.virtual portName default = %#v, want graph-only empty default", got)
	}
	if got := providerDefaults["serial.filter"]; !reflect.DeepEqual(got, map[string]any{
		"mode":          "plain",
		"expression":    "",
		"caseSensitive": false,
		"wholeWord":     false,
	}) {
		t.Fatalf("serial.filter default_config = %#v, want frontend-compatible defaults", got)
	}
	filter := providerSpecs["serial.filter"]
	filterInputs := filter["inputs"].([]any)
	filterOutputs := filter["outputs"].([]any)
	if len(filterInputs) != 1 || filterInputs[0].(map[string]any)["id"] != "in" || filterInputs[0].(map[string]any)["kind"] != "bytes" {
		t.Fatalf("serial.filter inputs = %#v, want one bytes input named in", filterInputs)
	}
	if len(filterOutputs) != 1 || filterOutputs[0].(map[string]any)["id"] != "out" || filterOutputs[0].(map[string]any)["kind"] != "bytes" {
		t.Fatalf("serial.filter outputs = %#v, want one bytes output named out", filterOutputs)
	}
	if got := providerDefaults["serial.script.transform"]; !reflect.DeepEqual(got, map[string]any{
		"script":         "output.bytes(input.bytes())",
		"timeoutMs":      float64(50),
		"maxOutputBytes": float64(65536),
		"maxStateBytes":  float64(262144),
		"onError":        "mark-error-and-drop",
		"encoding":       "utf-8",
		"autoRun":        false,
		"intervalMs":     float64(1000),
		"displayMode":    "hex",
	}) {
		t.Fatalf("serial.script.transform default_config = %#v, want script transform defaults", got)
	}
	if got := providerDefaults["serial.script.generator"]; !reflect.DeepEqual(got, map[string]any{
		"script":         "output.text(\"tick\", \"utf-8\")",
		"timeoutMs":      float64(50),
		"maxOutputBytes": float64(65536),
		"maxStateBytes":  float64(262144),
		"onError":        "mark-error-and-drop",
		"encoding":       "utf-8",
		"autoRun":        true,
		"intervalMs":     float64(1000),
		"displayMode":    "hex",
	}) {
		t.Fatalf("serial.script.generator default_config = %#v, want script generator defaults", got)
	}
	if got := providerDefaults["serial.script.analyzer"]; !reflect.DeepEqual(got, map[string]any{
		"script":         "field(\"length\", input.bytes().length)",
		"timeoutMs":      float64(50),
		"maxOutputBytes": float64(65536),
		"maxStateBytes":  float64(262144),
		"onError":        "mark-error-and-drop",
		"encoding":       "utf-8",
		"autoRun":        false,
		"intervalMs":     float64(1000),
		"displayMode":    "hex",
	}) {
		t.Fatalf("serial.script.analyzer default_config = %#v, want script analyzer defaults", got)
	}
	for _, raw := range providers {
		provider := raw.(map[string]any)
		if provider["type"] != "serial.monitor" {
			continue
		}
		if outputs := provider["outputs"].([]any); len(outputs) != 0 {
			t.Fatalf("serial.monitor outputs = %#v, want no non-emitting graph outputs", outputs)
		}
	}

	valid := callMCP(t, handler, `{
		"jsonrpc":"2.0",
		"id":41,
		"method":"tools/call",
		"params":{"name":"serial_graph_validate","arguments":{
			"nodes":[
				{"id":"generator","type":"serial.script.generator","position":{"x":20,"y":20},"config":{}},
				{"id":"virtual","type":"serial.virtual","position":{"x":260,"y":20},"config":{}},
				{"id":"monitor","type":"serial.monitor","position":{"x":500,"y":20},"config":{}}
			],
			"edges":[
				{"id":"edge-1","source":"generator","source_handle":"out","target":"virtual","target_handle":"tx"},
				{"id":"edge-2","source":"virtual","source_handle":"rx","target":"monitor","target_handle":"in"}
			]
		}}
	}`)
	if valid["structuredContent"].(map[string]any)["valid"] != true {
		t.Fatalf("valid graph result = %#v", valid["structuredContent"])
	}

	frontEndShape := callMCP(t, handler, `{
		"jsonrpc":"2.0",
		"id":411,
		"method":"tools/call",
		"params":{"name":"serial_graph_validate","arguments":{
			"nodes":[
				{"id":"generator","type":"serial.script.generator","position":{"x":20,"y":20},"config":{}},
				{"id":"virtual","type":"serial.virtual","position":{"x":260,"y":20},"config":{}},
				{"id":"monitor","type":"serial.monitor","position":{"x":500,"y":20},"config":{"displayMode":"hex"}}
			],
			"edges":[
				{"id":"edge-1","source":"generator","sourceHandle":"out","target":"virtual","targetHandle":"tx"},
				{"id":"edge-2","source":"virtual","sourceHandle":"rx","target":"monitor","targetHandle":"in"}
			]
		}}
	}`)
	if frontEndShape["structuredContent"].(map[string]any)["valid"] != true {
		t.Fatalf("frontend-shaped graph result = %#v", frontEndShape["structuredContent"])
	}

	scriptGraph := callMCP(t, handler, `{
		"jsonrpc":"2.0",
		"id":412,
		"method":"tools/call",
		"params":{"name":"serial_graph_validate","arguments":{
			"nodes":[
				{"id":"generator","type":"serial.script.generator","position":{"x":20,"y":20},"config":{}},
				{"id":"transform","type":"serial.script.transform","position":{"x":260,"y":20},"config":{}},
				{"id":"analyzer","type":"serial.script.analyzer","position":{"x":500,"y":20},"config":{}}
			],
			"edges":[
				{"id":"edge-1","source":"generator","sourceHandle":"out","target":"transform","targetHandle":"in"},
				{"id":"edge-2","source":"transform","sourceHandle":"out","target":"analyzer","targetHandle":"in"}
			]
		}}
	}`)
	if scriptGraph["structuredContent"].(map[string]any)["valid"] != true {
		t.Fatalf("script graph result = %#v", scriptGraph["structuredContent"])
	}

	invalid := callMCP(t, handler, `{
		"jsonrpc":"2.0",
		"id":42,
		"method":"tools/call",
		"params":{"name":"serial_graph_validate","arguments":{
			"nodes":[
				{"id":"a","type":"serial.bridge","position":{"x":20,"y":20},"config":{}},
				{"id":"b","type":"serial.bridge","position":{"x":260,"y":20},"config":{}}
			],
			"edges":[
				{"id":"edge-1","source":"a","source_handle":"a-out","target":"b","target_handle":"a-in"},
				{"id":"edge-2","source":"b","source_handle":"a-out","target":"a","target_handle":"a-in"}
			]
		}}
	}`)
	invalidContent := invalid["structuredContent"].(map[string]any)
	if invalidContent["valid"] != false {
		t.Fatalf("cycle result = %#v, want invalid", invalidContent)
	}
	errors := invalidContent["errors"].([]any)
	if len(errors) == 0 || !strings.Contains(errors[0].(string), "cycle") {
		t.Fatalf("cycle errors = %#v", errors)
	}
}

func TestSerialGraphToolsExposeAndValidateRemoteProvider(t *testing.T) {
	t.Parallel()
	handler := mcpHTTPHandler(newMCPServer(&fakeSerialService{}), true)

	catalog := callMCPTool(t, handler, 460, "serial_graph_provider_catalog", map[string]any{})
	providers := catalog["structuredContent"].(map[string]any)["providers"].([]any)
	var remote map[string]any
	for _, raw := range providers {
		provider := raw.(map[string]any)
		if provider["type"] == "serial.remote" {
			remote = provider
			break
		}
	}
	if remote == nil {
		t.Fatalf("catalog missing serial.remote in %#v", providers)
	}
	if remote["title"] != "远端串口" || remote["category"] != "串口" || remote["resource_owner"] != true {
		t.Fatalf("remote provider metadata = %#v", remote)
	}
	inputs := remote["inputs"].([]any)
	outputs := remote["outputs"].([]any)
	if len(inputs) != 1 || inputs[0].(map[string]any)["id"] != "tx" || len(outputs) != 1 || outputs[0].(map[string]any)["id"] != "rx" {
		t.Fatalf("remote ports inputs=%#v outputs=%#v", inputs, outputs)
	}
	defaults := remote["default_config"].(map[string]any)
	if defaults["protocol"] != "raw-tcp" || defaults["role"] != "client" || defaults["mode"] != nil || defaults["port"] != float64(3001) {
		t.Fatalf("remote default_config = %#v", defaults)
	}

	validNodes := []map[string]any{
		{"id": "remote", "type": "serial.remote", "position": map[string]any{"x": 0, "y": 0}, "config": map[string]any{"protocol": "raw-tcp", "role": "client", "host": "127.0.0.1", "port": 3001}},
		{"id": "virtual", "type": "serial.virtual", "position": map[string]any{"x": 240, "y": 0}, "config": map[string]any{}},
	}
	validEdges := []map[string]any{
		{"id": "edge-remote-virtual", "source": "remote", "sourceHandle": "rx", "target": "virtual", "targetHandle": "tx"},
	}
	valid := callMCPTool(t, handler, 461, "serial_graph_validate", map[string]any{"nodes": validNodes, "edges": validEdges})
	if valid["structuredContent"].(map[string]any)["valid"] != true {
		t.Fatalf("remote validate result = %#v", valid["structuredContent"])
	}

	invalid := callMCPTool(t, handler, 462, "serial_graph_validate", map[string]any{
		"nodes": []map[string]any{
			{"id": "remote-a", "type": "serial.remote", "position": map[string]any{"x": 0, "y": 0}, "config": map[string]any{"host": "127.0.0.1", "port": 3001}},
			{"id": "remote-b", "type": "serial.remote", "position": map[string]any{"x": 240, "y": 0}, "config": map[string]any{"host": "127.0.0.1", "port": 3001}},
			{"id": "remote-c", "type": "serial.remote", "position": map[string]any{"x": 480, "y": 0}, "config": map[string]any{"host": "tcp://127.0.0.1", "port": 70000, "protocol": "rfc2217", "role": "peer"}},
			{"id": "remote-d", "type": "serial.remote", "position": map[string]any{"x": 720, "y": 0}, "config": map[string]any{"host": "127.0.0.1", "port": "3001x"}},
		},
		"edges": []map[string]any{},
	})
	content := invalid["structuredContent"].(map[string]any)
	if content["valid"] != false {
		t.Fatalf("invalid remote validate result = %#v", content)
	}
	var joined string
	for _, raw := range content["errors"].([]any) {
		joined += raw.(string) + "\n"
	}
	for _, want := range []string{"resource remote endpoint duplicated", "host must not include URL scheme", "port out of range", "port must be an integer", "unsupported protocol", "unsupported role"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("remote errors = %q, want %q", joined, want)
		}
	}

	serverClient := callMCPTool(t, handler, 4621, "serial_graph_validate", map[string]any{
		"nodes": []map[string]any{
			{"id": "remote-server", "type": "serial.remote", "position": map[string]any{"x": 0, "y": 0}, "config": map[string]any{"host": "127.0.0.1", "port": 3001, "role": "server"}},
			{"id": "remote-client", "type": "serial.remote", "position": map[string]any{"x": 240, "y": 0}, "config": map[string]any{"host": "127.0.0.1", "port": 3001, "role": "client"}},
		},
		"edges": []map[string]any{},
	})
	if serverClient["structuredContent"].(map[string]any)["valid"] != true {
		t.Fatalf("server/client remote endpoint validate result = %#v", serverClient["structuredContent"])
	}
}

func TestSerialGraphRemoteConfigValidationCoversNumericRanges(t *testing.T) {
	t.Parallel()

	valid := map[string]any{
		"protocol":            "raw-tcp",
		"role":                "client",
		"host":                "LOCALHOST",
		"port":                int64(3001),
		"connectTimeoutMs":    float32(100),
		"writeTimeoutMs":      float64(60000),
		"reconnectIntervalMs": "250",
		"readBufKB":           "1024",
	}
	if errs := validateSerialGraphRemoteConfig("remote", valid); len(errs) != 0 {
		t.Fatalf("valid remote config errors = %#v", errs)
	}
	if endpoint, ok := serialGraphRemoteEndpoint(valid); !ok || endpoint != "localhost:3001" {
		t.Fatalf("serialGraphRemoteEndpoint = %q, %v; want localhost:3001, true", endpoint, ok)
	}

	for _, tc := range []struct {
		name   string
		config map[string]any
		want   string
	}{
		{name: "missing host", config: map[string]any{"port": 3001}, want: "remote host required"},
		{name: "schemed host", config: map[string]any{"host": "tcp://127.0.0.1", "port": 3001}, want: "remote host must not include URL scheme"},
		{name: "nil port", config: map[string]any{"host": "127.0.0.1", "port": nil}, want: "remote port must be an integer"},
		{name: "fractional port", config: map[string]any{"host": "127.0.0.1", "port": 3.14}, want: "remote port must be an integer"},
		{name: "low port", config: map[string]any{"host": "127.0.0.1", "port": 0}, want: "remote port out of range"},
		{name: "high port", config: map[string]any{"host": "127.0.0.1", "port": 70000}, want: "remote port out of range"},
		{name: "nil connect timeout", config: map[string]any{"host": "127.0.0.1", "connectTimeoutMs": nil}, want: "connectTimeoutMs must be an integer"},
		{name: "zero connect timeout", config: map[string]any{"host": "127.0.0.1", "connectTimeoutMs": 0}, want: "connectTimeoutMs must be positive"},
		{name: "short connect timeout", config: map[string]any{"host": "127.0.0.1", "connectTimeoutMs": 99}, want: "connectTimeoutMs out of range"},
		{name: "long connect timeout", config: map[string]any{"host": "127.0.0.1", "connectTimeoutMs": 60001}, want: "connectTimeoutMs out of range"},
		{name: "nil write timeout", config: map[string]any{"host": "127.0.0.1", "writeTimeoutMs": nil}, want: "writeTimeoutMs must be an integer"},
		{name: "zero write timeout", config: map[string]any{"host": "127.0.0.1", "writeTimeoutMs": 0}, want: "writeTimeoutMs must be positive"},
		{name: "short write timeout", config: map[string]any{"host": "127.0.0.1", "writeTimeoutMs": 99}, want: "writeTimeoutMs out of range"},
		{name: "long write timeout", config: map[string]any{"host": "127.0.0.1", "writeTimeoutMs": 60001}, want: "writeTimeoutMs out of range"},
		{name: "nil reconnect interval", config: map[string]any{"host": "127.0.0.1", "reconnectIntervalMs": nil}, want: "reconnectIntervalMs must be an integer"},
		{name: "zero reconnect interval", config: map[string]any{"host": "127.0.0.1", "reconnectIntervalMs": 0}, want: "reconnectIntervalMs must be positive"},
		{name: "short reconnect interval", config: map[string]any{"host": "127.0.0.1", "reconnectIntervalMs": 99}, want: "reconnectIntervalMs out of range"},
		{name: "long reconnect interval", config: map[string]any{"host": "127.0.0.1", "reconnectIntervalMs": 60001}, want: "reconnectIntervalMs out of range"},
		{name: "nil read buffer", config: map[string]any{"host": "127.0.0.1", "readBufKB": nil}, want: "readBufKB must be an integer"},
		{name: "zero read buffer", config: map[string]any{"host": "127.0.0.1", "readBufKB": 0}, want: "readBufKB must be positive"},
		{name: "large read buffer", config: map[string]any{"host": "127.0.0.1", "readBufKB": 2048}, want: "readBufKB out of range"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			errs := validateSerialGraphRemoteConfig("remote", tc.config)
			if !strings.Contains(strings.Join(errs, "\n"), tc.want) {
				t.Fatalf("errors = %#v, want %q", errs, tc.want)
			}
		})
	}
}

func TestSerialGraphValidateAllowsRemoteProtocolResponseCycle(t *testing.T) {
	t.Parallel()
	handler := mcpHTTPHandler(newMCPServer(&fakeSerialService{}), true)

	result := callMCPTool(t, handler, 463, "serial_graph_validate", map[string]any{
		"nodes": []map[string]any{
			{"id": "master", "type": "serial.modbus.master", "position": map[string]any{"x": 0, "y": 0}, "config": map[string]any{"mode": "rtu", "unitIds": "1"}},
			{"id": "remote", "type": "serial.remote", "position": map[string]any{"x": 240, "y": 0}, "config": map[string]any{"protocol": "raw-tcp", "role": "client", "host": "127.0.0.1", "port": 3001}},
		},
		"edges": []map[string]any{
			{"id": "edge-master-remote", "source": "master", "sourceHandle": "tx", "target": "remote", "targetHandle": "tx"},
			{"id": "edge-remote-master", "source": "remote", "sourceHandle": "rx", "target": "master", "targetHandle": "rx"},
		},
	})
	content := result["structuredContent"].(map[string]any)
	if content["valid"] != true {
		t.Fatalf("remote protocol response cycle validate = %#v", content)
	}
}

func TestSerialGraphRuntimeToolsCallSerialService(t *testing.T) {
	t.Parallel()
	svc := &fakeSerialService{}
	handler := mcpHTTPHandler(newMCPServer(svc), true)

	start := callMCP(t, handler, `{
		"jsonrpc":"2.0",
		"id":44,
		"method":"tools/call",
		"params":{"name":"serial_graph_start","arguments":{
			"id":"graph-mcp",
			"nodes":[
				{"id":"generator","type":"serial.script.generator","position":{"x":0,"y":0},"config":{}},
				{"id":"virtual","type":"serial.virtual","position":{"x":260,"y":0},"config":{}}
			],
			"edges":[{"id":"edge-1","source":"generator","sourceHandle":"out","target":"virtual","targetHandle":"tx"}]
		}}
	}`)
	startContent := start["structuredContent"].(map[string]any)
	if startContent["graph"].(map[string]any)["ID"] != "graph-mcp" {
		t.Fatalf("start content = %#v", startContent)
	}
	if svc.startGraph.ID != "graph-mcp" || len(svc.startGraph.Edges) != 1 || svc.startGraph.Edges[0].SourceHandle != "out" {
		t.Fatalf("start graph args = %#v", svc.startGraph)
	}

	send := callMCP(t, handler, `{
		"jsonrpc":"2.0",
		"id":45,
		"method":"tools/call",
		"params":{"name":"serial_graph_send","arguments":{"graph_id":"graph-mcp","node_id":"virtual","content":"hello","mode":"ascii"}}
	}`)
	if send["structuredContent"].(map[string]any)["bytes_written"] != float64(5) {
		t.Fatalf("send content = %#v", send["structuredContent"])
	}
	if svc.graphSend.GraphID != "graph-mcp" || svc.graphSend.NodeID != "virtual" {
		t.Fatalf("graph send args = %#v", svc.graphSend)
	}

	bufferPage := callMCP(t, handler, `{
		"jsonrpc":"2.0",
		"id":46,
		"method":"tools/call",
		"params":{"name":"serial_graph_query_node_buffer","arguments":{"graph_id":"graph-mcp","node_id":"virtual","offset":0,"length":16}}
	}`)
	bufferContent := bufferPage["structuredContent"].(map[string]any)
	if bufferContent["data"] != "hello" || bufferContent["hex"] != "68656c6c6f" {
		t.Fatalf("buffer content = %#v", bufferContent)
	}
	if svc.graphBuffer.NodeID != "virtual" {
		t.Fatalf("graph buffer args = %#v", svc.graphBuffer)
	}

	frames := callMCP(t, handler, `{
		"jsonrpc":"2.0",
		"id":47,
		"method":"tools/call",
		"params":{"name":"serial_graph_query_node_frames","arguments":{"graph_id":"graph-mcp","node_id":"monitor","limit":20}}
	}`)
	if frames["structuredContent"].(map[string]any)["total"] != float64(1) {
		t.Fatalf("frames content = %#v", frames["structuredContent"])
	}

	status := callMCP(t, handler, `{
		"jsonrpc":"2.0",
		"id":48,
		"method":"tools/call",
		"params":{"name":"serial_graph_status","arguments":{"graph_id":"graph-mcp"}}
	}`)
	if status["structuredContent"].(map[string]any)["graph"].(map[string]any)["ID"] != "graph-mcp" {
		t.Fatalf("status content = %#v", status["structuredContent"])
	}

	callMCP(t, handler, `{
		"jsonrpc":"2.0",
		"id":49,
		"method":"tools/call",
		"params":{"name":"serial_graph_clear_node_buffer","arguments":{"graph_id":"graph-mcp","node_id":"virtual"}}
	}`)
	if svc.clearGraphID != "graph-mcp" || svc.clearGraphNode != "virtual" {
		t.Fatalf("clear args = %s/%s", svc.clearGraphID, svc.clearGraphNode)
	}

	callMCP(t, handler, `{
		"jsonrpc":"2.0",
		"id":50,
		"method":"tools/call",
		"params":{"name":"serial_graph_reset_node_counters","arguments":{"graph_id":"graph-mcp","node_id":"virtual"}}
	}`)
	if svc.resetGraphID != "graph-mcp" || svc.resetGraphNode != "virtual" {
		t.Fatalf("reset args = %s/%s", svc.resetGraphID, svc.resetGraphNode)
	}

	callMCP(t, handler, `{
		"jsonrpc":"2.0",
		"id":51,
		"method":"tools/call",
		"params":{"name":"serial_graph_stop","arguments":{"graph_id":"graph-mcp"}}
	}`)
	if svc.stopGraph != "graph-mcp" {
		t.Fatalf("stop graph = %q", svc.stopGraph)
	}
}

func TestSerialRuntimeToolHandlersSurfaceServiceErrors(t *testing.T) {
	t.Parallel()
	handler := mcpHTTPHandler(newMCPServer(&failingSerialService{err: errors.New("service boom")}), true)
	cases := []struct {
		name string
		args map[string]any
	}{
		{name: "serial_enumerate_ports", args: map[string]any{}},
		{name: "serial_open_port", args: map[string]any{"port_name": "/tmp/ttyA", "baud_rate": 9600}},
		{name: "serial_close_port", args: map[string]any{"port_id": "port-1"}},
		{name: "serial_send", args: map[string]any{"port_id": "port-1", "content": "hello"}},
		{name: "serial_query_buffer", args: map[string]any{"port_id": "port-1", "offset": 0}},
		{name: "serial_reset_counters", args: map[string]any{"port_id": "port-1"}},
		{name: "serial_create_virtual_port", args: map[string]any{"id": "v1", "port_name": "tty-v1"}},
		{name: "serial_delete_virtual_port", args: map[string]any{"id": "v1"}},
		{name: "serial_create_virtual_pair", args: map[string]any{"id": "pair1", "port1_name": "tty-a", "port2_name": "tty-b"}},
		{name: "serial_delete_virtual_pair", args: map[string]any{"id": "pair1"}},
		{name: "serial_create_bridge", args: map[string]any{"id": "bridge1", "port1": "tty-a", "port2": "tty-b"}},
		{name: "serial_delete_bridge", args: map[string]any{"id": "bridge1"}},
		{name: "serial_start_monitor", args: map[string]any{"id": "mon-1", "port": "/tmp/ttyM"}},
		{name: "serial_stop_monitor", args: map[string]any{"monitor_id": "mon-1"}},
		{name: "serial_delete_monitor", args: map[string]any{"monitor_id": "mon-1"}},
		{name: "serial_query_monitor_frames", args: map[string]any{"monitor_id": "mon-1"}},
		{name: "serial_clear_monitor_frames", args: map[string]any{"monitor_id": "mon-1"}},
		{name: "modbus_open_session", args: map[string]any{"id": "mb-1", "port": "/tmp/ttyM"}},
		{name: "modbus_close_session", args: map[string]any{"session_id": "mb-1"}},
		{name: "modbus_master_request", args: map[string]any{"session_id": "mb-1", "unit_id": 1, "function": 3, "address": 0}},
		{name: "modbus_start_slave", args: map[string]any{"session_id": "mb-1", "unit_id": 1}},
		{name: "modbus_stop_slave", args: map[string]any{"session_id": "mb-1"}},
		{name: "modbus_update_slave_data", args: map[string]any{"session_id": "mb-1", "data_model": map[string]any{"coils": []map[string]any{{"address": 1, "value": true}}}}},
		{name: "modbus_add_slave_unit", args: map[string]any{"session_id": "mb-1", "unit_id": 2, "data_model": map[string]any{}}},
		{name: "modbus_remove_slave_unit", args: map[string]any{"session_id": "mb-1", "unit_id": 2}},
		{name: "modbus_list_slave_units", args: map[string]any{"session_id": "mb-1"}},
		{name: "modbus_update_slave_unit_data", args: map[string]any{"session_id": "mb-1", "unit_id": 2, "data_model": map[string]any{}}},
		{name: "modbus_scan_unit_ids", args: map[string]any{"session_id": "mb-1", "unit_ids": []int{1}, "function": 3, "address": 0, "quantity": 1}},
		{name: "modbus_read_registers", args: map[string]any{"session_id": "mb-1", "unit_id": 1, "function": 3, "address": 0, "quantity": 1}},
		{name: "modbus_scan_registers", args: map[string]any{"session_id": "mb-1", "unit_id": 1, "function": 3, "start_address": 0, "end_address": 2}},
		{name: "fecbus_open_session", args: map[string]any{"id": "fec-1", "port": "/tmp/ttyF"}},
		{name: "fecbus_close_session", args: map[string]any{"session_id": "fec-1"}},
		{name: "fecbus_send_request", args: map[string]any{"session_id": "fec-1", "target_address": 2, "function": 44}},
		{name: "fecbus_start_slave", args: map[string]any{"session_id": "fec-1", "address": 2}},
		{name: "fecbus_stop_slave", args: map[string]any{"session_id": "fec-1"}},
		{name: "fecbus_update_slave_state", args: map[string]any{"session_id": "fec-1", "address": 2}},
		{name: "fecbus_add_slave_unit", args: map[string]any{"session_id": "fec-1", "address": 2}},
		{name: "fecbus_remove_slave_unit", args: map[string]any{"session_id": "fec-1", "address": 2}},
		{name: "fecbus_list_slave_units", args: map[string]any{"session_id": "fec-1"}},
		{name: "fecbus_query_frames", args: map[string]any{"session_id": "fec-1"}},
		{name: "fecbus_clear_frames", args: map[string]any{"session_id": "fec-1"}},
		{name: "serial_graph_demo_template", args: map[string]any{"id": "missing"}},
		{name: "serial_graph_start", args: map[string]any{"id": "graph-1", "nodes": []map[string]any{}, "edges": []map[string]any{}}},
		{name: "serial_graph_stop", args: map[string]any{"graph_id": "graph-1"}},
		{name: "serial_graph_status", args: map[string]any{"graph_id": "graph-1"}},
		{name: "serial_graph_send", args: map[string]any{"graph_id": "graph-1", "node_id": "sender", "content": "hello"}},
		{name: "serial_graph_query_node_buffer", args: map[string]any{"graph_id": "graph-1", "node_id": "receiver"}},
		{name: "serial_graph_query_node_frames", args: map[string]any{"graph_id": "graph-1", "node_id": "monitor"}},
		{name: "serial_graph_clear_node_buffer", args: map[string]any{"graph_id": "graph-1", "node_id": "receiver"}},
		{name: "serial_graph_reset_node_counters", args: map[string]any{"graph_id": "graph-1", "node_id": "receiver"}},
	}

	for i, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := callMCPToolAllowError(t, handler, 600+i, tc.name, tc.args)
			if result["isError"] != true {
				t.Fatalf("%s result = %#v, want tool error", tc.name, result)
			}
		})
	}
}

func TestSerialGraphRuntimeMCPFiltersPassAndDrop(t *testing.T) {
	svc := serial.NewService(nil)
	handler := mcpHTTPHandler(newMCPServer(svc), true)

	graphArgs := map[string]any{
		"id": "graph-mcp-filter",
		"nodes": []map[string]any{
			{"id": "source", "type": "serial.virtual", "position": map[string]any{"x": 0, "y": 0}, "config": map[string]any{}},
			{"id": "filter", "type": "serial.filter", "position": map[string]any{"x": 240, "y": 0}, "config": map[string]any{"mode": "plain", "expression": "OK", "caseSensitive": false, "wholeWord": false}},
			{"id": "sink", "type": "serial.virtual", "position": map[string]any{"x": 480, "y": 0}, "config": map[string]any{}},
		},
		"edges": []map[string]any{
			{"id": "edge-source-filter", "source": "source", "sourceHandle": "rx", "target": "filter", "targetHandle": "in"},
			{"id": "edge-filter-sink", "source": "filter", "sourceHandle": "out", "target": "sink", "targetHandle": "tx"},
		},
	}

	valid := callMCPTool(t, handler, 440, "serial_graph_validate", map[string]any{"nodes": graphArgs["nodes"], "edges": graphArgs["edges"]})
	if valid["structuredContent"].(map[string]any)["valid"] != true {
		t.Fatalf("filter graph validate result = %#v", valid["structuredContent"])
	}

	start := callMCPTool(t, handler, 441, "serial_graph_start", graphArgs)
	if start["structuredContent"].(map[string]any)["graph"].(map[string]any)["ID"] != "graph-mcp-filter" {
		t.Fatalf("start content = %#v", start["structuredContent"])
	}
	defer func() { _ = svc.StopSerialGraph("graph-mcp-filter") }()

	callMCPTool(t, handler, 442, "serial_graph_send", map[string]any{"graph_id": "graph-mcp-filter", "node_id": "source", "content": "OK: keep", "mode": "ascii"})
	callMCPTool(t, handler, 443, "serial_graph_send", map[string]any{"graph_id": "graph-mcp-filter", "node_id": "source", "content": "DROP: ignore", "mode": "ascii"})

	bufferPage := callMCPTool(t, handler, 444, "serial_graph_query_node_buffer", map[string]any{"graph_id": "graph-mcp-filter", "node_id": "sink", "offset": 0, "length": 64})
	bufferContent := bufferPage["structuredContent"].(map[string]any)
	if bufferContent["data"] != "OK: keep" || bufferContent["hex"] != "4f4b3a206b656570" {
		t.Fatalf("filtered sink buffer = %#v, want only matching payload", bufferContent)
	}
}

func TestSerialGraphRemoteTemplateMCPRuntimeRoundTrip(t *testing.T) {
	port := freeTCPPort(t)
	svc := serial.NewService(nil)
	handler := mcpHTTPHandler(newMCPServer(svc), true)

	templateResult := callMCPTool(t, handler, 445, "serial_graph_demo_template", map[string]any{
		"id":          "serial-remote-raw-tcp",
		"graph_id":    "graph-mcp-remote-template",
		"remote_host": "127.0.0.1",
		"remote_port": port,
	})
	template := templateResult["structuredContent"].(map[string]any)
	nodes := template["nodes"].([]any)
	edges := template["edges"].([]any)

	valid := callMCPTool(t, handler, 446, "serial_graph_validate", map[string]any{"nodes": nodes, "edges": edges})
	if valid["structuredContent"].(map[string]any)["valid"] != true {
		t.Fatalf("remote graph validate result = %#v", valid["structuredContent"])
	}

	start := callMCPTool(t, handler, 447, "serial_graph_start", map[string]any{"id": template["graph_id"], "nodes": nodes, "edges": edges})
	if start["structuredContent"].(map[string]any)["graph"].(map[string]any)["ID"] != "graph-mcp-remote-template" {
		t.Fatalf("remote graph start result = %#v", start["structuredContent"])
	}
	defer func() { _ = svc.StopSerialGraph("graph-mcp-remote-template") }()

	serverStartup := waitMCPGraphBufferData(t, handler, "graph-mcp-remote-template", "a-remote-server", "client -> server")
	if !strings.Contains(fmt.Sprint(serverStartup["data"]), "client -> server") {
		t.Fatalf("server endpoint startup buffer = %#v", serverStartup)
	}

	callMCPTool(t, handler, 448, "serial_graph_send", map[string]any{"graph_id": "graph-mcp-remote-template", "node_id": "b-remote-client", "content": "client->server", "mode": "ascii"})
	serverBuffer := waitMCPGraphBufferData(t, handler, "graph-mcp-remote-template", "a-remote-server", "client->server")
	if !strings.Contains(fmt.Sprint(serverBuffer["data"]), "client->server") {
		t.Fatalf("server endpoint buffer = %#v", serverBuffer)
	}

	callMCPTool(t, handler, 449, "serial_graph_send", map[string]any{"graph_id": "graph-mcp-remote-template", "node_id": "a-remote-server", "content": "server->client", "mode": "ascii"})
	clientBuffer := waitMCPGraphBufferData(t, handler, "graph-mcp-remote-template", "b-remote-client", "server->client")
	if !strings.Contains(fmt.Sprint(clientBuffer["data"]), "server->client") {
		t.Fatalf("client endpoint buffer = %#v", clientBuffer)
	}

	status := callMCPTool(t, handler, 450, "serial_graph_status", map[string]any{"graph_id": "graph-mcp-remote-template"})
	graph := status["structuredContent"].(map[string]any)["graph"].(map[string]any)
	if graph["ID"] != "graph-mcp-remote-template" {
		t.Fatalf("remote graph status = %#v", graph)
	}
}

func TestSerialGraphDemoTemplateToolsReturnValidFilterLoggingGraph(t *testing.T) {
	handler := mcpHTTPHandler(newMCPServer(&fakeSerialService{}), true)

	catalog := callMCPTool(t, handler, 450, "serial_graph_demo_catalog", map[string]any{})
	templates := catalog["structuredContent"].(map[string]any)["templates"].([]any)
	found := false
	for _, raw := range templates {
		template := raw.(map[string]any)
		if template["id"] == "serial-observability-filter-logging" {
			found = true
			if template["operation_log_state"] != "not_in_backend" {
				t.Fatalf("template metadata = %#v, want operation log limitation documented", template)
			}
		}
	}
	if !found {
		t.Fatalf("serial_graph_demo_catalog templates = %#v, want serial-observability-filter-logging", templates)
	}

	templateResult := callMCPTool(t, handler, 451, "serial_graph_demo_template", map[string]any{
		"id":        "serial-observability-filter-logging",
		"graph_id":  "graph-from-template",
		"port_name": "mcp-demo-vport",
	})
	template := templateResult["structuredContent"].(map[string]any)
	if template["id"] != "serial-observability-filter-logging" || template["usage"] == nil {
		t.Fatalf("template content = %#v, want id and usage", template)
	}
	nodes := template["nodes"].([]any)
	edges := template["edges"].([]any)
	if len(nodes) != 8 || len(edges) != 7 {
		t.Fatalf("template nodes/edges = %d/%d, want generator -> virtual -> filters -> monitors graph", len(nodes), len(edges))
	}

	configsByID := make(map[string]map[string]any, len(nodes))
	var sawGenerator, sawVirtual, sawFilter, sawMonitor bool
	for _, raw := range nodes {
		node := raw.(map[string]any)
		config, _ := node["config"].(map[string]any)
		configsByID[node["id"].(string)] = config
		switch node["type"] {
		case "serial.script.generator":
			sawGenerator = sawGenerator || config["autoRun"] == true && strings.Contains(fmt.Sprint(config["script"]), "OK TEMP=42")
		case "serial.virtual":
			sawVirtual = sawVirtual || config["portName"] == "mcp-demo-vport"
		case "serial.filter":
			sawFilter = sawFilter || config["mode"] == "plain" && config["expression"] == "OK"
		case "serial.monitor":
			sawMonitor = true
		}
	}
	if !sawGenerator || !sawVirtual || !sawFilter || !sawMonitor {
		t.Fatalf("template nodes = %#v, want generator, virtual port, filters, and monitors", nodes)
	}
	if got := configsByID["monitor-plain"]["displayMode"]; got != "ascii" {
		t.Fatalf("monitor-plain displayMode = %#v, want ascii", got)
	}
	if got := configsByID["monitor-regex"]["displayMode"]; got != "hex" {
		t.Fatalf("monitor-regex displayMode = %#v, want hex", got)
	}

	valid := callMCPTool(t, handler, 452, "serial_graph_validate", map[string]any{"nodes": nodes, "edges": edges})
	if valid["structuredContent"].(map[string]any)["valid"] != true {
		t.Fatalf("template validate result = %#v", valid["structuredContent"])
	}
}

func TestSerialGraphDemoTemplateToolsReturnValidRemoteRawTCPGraph(t *testing.T) {
	handler := mcpHTTPHandler(newMCPServer(&fakeSerialService{}), true)

	catalog := callMCPTool(t, handler, 452, "serial_graph_demo_catalog", map[string]any{})
	templates := catalog["structuredContent"].(map[string]any)["templates"].([]any)
	var remoteTemplate map[string]any
	for _, raw := range templates {
		template := raw.(map[string]any)
		if template["id"] == "serial-remote-raw-tcp" {
			remoteTemplate = template
			break
		}
	}
	if remoteTemplate == nil {
		t.Fatalf("serial_graph_demo_catalog templates = %#v, want serial-remote-raw-tcp", templates)
	}
	if remoteTemplate["security"] == nil || remoteTemplate["operation_log_state"] != "not_in_backend" {
		t.Fatalf("remote template metadata = %#v, want security and backend limitation documented", remoteTemplate)
	}

	templateResult := callMCPTool(t, handler, 453, "serial_graph_demo_template", map[string]any{
		"id":                       "serial-remote-raw-tcp",
		"graph_id":                 "graph-remote-template",
		"remote_host":              "serial-gateway.local",
		"remote_port":              3002,
		"allow_start_disconnected": true,
	})
	template := templateResult["structuredContent"].(map[string]any)
	if template["id"] != "serial-remote-raw-tcp" || template["usage"] == nil {
		t.Fatalf("remote template content = %#v, want id and usage", template)
	}

	nodes := template["nodes"].([]any)
	edges := template["edges"].([]any)
	if len(nodes) != 5 || len(edges) != 3 {
		t.Fatalf("remote template nodes/edges = %d/%d, want in-graph server/client loopback with endpoint buffers", len(nodes), len(edges))
	}

	var sawServer, sawClient, sawClientGenerator, sawServerMonitor, sawClientMonitor bool
	for _, raw := range nodes {
		node := raw.(map[string]any)
		config, _ := node["config"].(map[string]any)
		switch node["id"] {
		case "client-generator":
			sawClientGenerator = node["type"] == "serial.script.generator" && config["autoRun"] == true && strings.Contains(fmt.Sprint(config["script"]), "client")
		case "a-remote-server":
			sawServer = node["type"] == "serial.remote" &&
				config["protocol"] == "raw-tcp" &&
				config["role"] == "server" &&
				config["host"] == "serial-gateway.local" &&
				config["port"] == float64(3002) &&
				config["allowStartDisconnected"] == false &&
				config["mode"] == nil
		case "b-remote-client":
			sawClient = node["type"] == "serial.remote" &&
				config["protocol"] == "raw-tcp" &&
				config["role"] == "client" &&
				config["host"] == "serial-gateway.local" &&
				config["port"] == float64(3002) &&
				config["allowStartDisconnected"] == true &&
				config["mode"] == nil
		case "server-monitor":
			sawServerMonitor = node["type"] == "serial.monitor" && config["displayMode"] == "hex"
		case "client-monitor":
			sawClientMonitor = node["type"] == "serial.monitor" && config["displayMode"] == "hex"
		}
	}
	if !sawServer || !sawClient || !sawClientGenerator || !sawServerMonitor || !sawClientMonitor {
		t.Fatalf("remote template nodes = %#v", nodes)
	}

	var sawClientToServer, sawServerMonitorEdge, sawClientMonitorEdge bool
	for _, raw := range edges {
		edge := raw.(map[string]any)
		switch edge["id"] {
		case "edge-client-generator-remote":
			sawClientToServer = edge["source"] == "client-generator" && edge["target"] == "b-remote-client"
		case "edge-server-remote-monitor":
			sawServerMonitorEdge = edge["source"] == "a-remote-server" && edge["target"] == "server-monitor"
		case "edge-client-remote-monitor":
			sawClientMonitorEdge = edge["source"] == "b-remote-client" && edge["target"] == "client-monitor"
		}
	}
	if !sawClientToServer || !sawServerMonitorEdge || !sawClientMonitorEdge {
		t.Fatalf("remote template edges = %#v", edges)
	}

	validate := callMCPTool(t, handler, 454, "serial_graph_validate", map[string]any{"nodes": nodes, "edges": edges})
	if validate["structuredContent"].(map[string]any)["valid"] != true {
		t.Fatalf("remote template validate = %#v", validate["structuredContent"])
	}

	usage := template["usage"].(map[string]any)
	for _, key := range []string{"validate", "start", "status_example", "send_example", "query_buffer_example", "security"} {
		if usage[key] == nil {
			t.Fatalf("remote template usage missing %s: %#v", key, usage)
		}
	}

	stringPortResult := callMCPTool(t, handler, 455, "serial_graph_demo_template", map[string]any{
		"id":          "serial-remote-raw-tcp",
		"graph_id":    "graph-remote-template-string-port",
		"remote_host": "127.0.0.1",
		"remote_port": "3003",
	})
	stringPortNodes := stringPortResult["structuredContent"].(map[string]any)["nodes"].([]any)
	var sawStringPort bool
	for _, raw := range stringPortNodes {
		node := raw.(map[string]any)
		if node["type"] != "serial.remote" {
			continue
		}
		config := node["config"].(map[string]any)
		sawStringPort = config["port"] == float64(3003)
	}
	if !sawStringPort {
		t.Fatalf("string remote_port template nodes = %#v, want parsed port 3003", stringPortNodes)
	}
}

func TestSerialGraphDemoTemplateRejectsInvalidRemoteArgs(t *testing.T) {
	handler := mcpHTTPHandler(newMCPServer(&fakeSerialService{}), true)

	for i, tc := range []struct {
		name string
		args map[string]any
		want string
	}{
		{name: "schemed host", args: map[string]any{"id": "serial-remote-raw-tcp", "remote_host": "tcp://127.0.0.1"}, want: "host must not include URL scheme"},
		{name: "bad port", args: map[string]any{"id": "serial-remote-raw-tcp", "remote_host": "127.0.0.1", "remote_port": 70000}, want: "port out of range"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := callMCPToolAllowError(t, handler, 470+i, "serial_graph_demo_template", tc.args)
			if result["isError"] != true || !strings.Contains(fmt.Sprint(result), tc.want) {
				t.Fatalf("result = %#v, want error containing %q", result, tc.want)
			}
		})
	}
}

func TestSerialGraphValidateRejectsInvalidTopologyRules(t *testing.T) {
	t.Parallel()
	handler := mcpHTTPHandler(newMCPServer(&fakeSerialService{}), true)

	for _, tc := range []struct {
		name      string
		arguments string
		wantError string
	}{
		{
			name: "unknown provider",
			arguments: `{
				"nodes":[{"id":"missing","type":"serial.missing","position":{"x":0,"y":0},"config":{}}],
				"edges":[]
			}`,
			wantError: "provider not found",
		},
		{
			name: "monitor has no emitting graph output",
			arguments: `{
				"nodes":[
					{"id":"monitor","type":"serial.monitor","position":{"x":0,"y":0},"config":{}},
					{"id":"sink","type":"serial.virtual","position":{"x":260,"y":0},"config":{}}
				],
				"edges":[{"id":"edge-1","source":"monitor","source_handle":"frames","target":"sink","target_handle":"tx"}]
			}`,
			wantError: "output port not found",
		},
		{
			name: "duplicate resource owner port",
			arguments: `{
				"nodes":[
					{"id":"port-a","type":"serial.physical","position":{"x":0,"y":0},"config":{"portName":"/tmp/ttyA"}},
					{"id":"port-b","type":"serial.virtual","position":{"x":260,"y":0},"config":{"portName":"/tmp/ttyA"}}
				],
				"edges":[]
			}`,
			wantError: "resource port duplicated",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := callMCP(t, handler, `{
				"jsonrpc":"2.0",
				"id":43,
				"method":"tools/call",
				"params":{"name":"serial_graph_validate","arguments":`+tc.arguments+`}
			}`)
			content := result["structuredContent"].(map[string]any)
			if content["valid"] != false {
				t.Fatalf("result=%#v, want invalid", content)
			}
			var joined string
			for _, raw := range content["errors"].([]any) {
				joined += raw.(string) + "\n"
			}
			if !strings.Contains(joined, tc.wantError) {
				t.Fatalf("errors = %q, want %q", joined, tc.wantError)
			}
		})
	}
}

func TestSerialGraphValidateAllowsDirectFanOut(t *testing.T) {
	t.Parallel()
	handler := mcpHTTPHandler(newMCPServer(&fakeSerialService{}), true)

	result := callMCP(t, handler, `{
		"jsonrpc":"2.0",
		"id":44,
		"method":"tools/call",
		"params":{"name":"serial_graph_validate","arguments":{
			"nodes":[
				{"id":"source","type":"serial.virtual","position":{"x":0,"y":0},"config":{}},
				{"id":"rx-a","type":"serial.virtual","position":{"x":260,"y":0},"config":{}},
				{"id":"rx-b","type":"serial.virtual","position":{"x":260,"y":120},"config":{}}
			],
			"edges":[
				{"id":"edge-1","source":"source","source_handle":"rx","target":"rx-a","target_handle":"tx"},
				{"id":"edge-2","source":"source","source_handle":"rx","target":"rx-b","target_handle":"tx"}
			]
		}}
	}`)
	content := result["structuredContent"].(map[string]any)
	if content["valid"] != true {
		t.Fatalf("direct fan-out validation = %#v, want valid", content)
	}
}

func TestFecbusToolsCallSerialService(t *testing.T) {
	t.Parallel()
	svc := &fakeSerialService{}
	handler := mcpHTTPHandler(newMCPServer(svc), true)

	catalog := callMCP(t, handler, `{
		"jsonrpc":"2.0",
		"id":20,
		"method":"tools/call",
		"params":{"name":"fecbus_function_catalog","arguments":{}}
	}`)
	functions := catalog["structuredContent"].(map[string]any)["functions"].([]any)
	if len(functions) != 256 {
		t.Fatalf("catalog length = %d, want 256", len(functions))
	}

	open := callMCP(t, handler, `{
		"jsonrpc":"2.0",
		"id":21,
		"method":"tools/call",
		"params":{"name":"fecbus_open_session","arguments":{"id":"fec-1","name":"FECbus 主控","port":"/tmp/ttyF0","role":"master","baud_rate":9600,"timeout_ms":1000,"retries":3}}
	}`)
	session := open["structuredContent"].(map[string]any)["session"].(map[string]any)
	if session["ID"] != "fec-1" || svc.openFecbus.Config.PortName != "/tmp/ttyF0" || svc.openFecbus.Config.BaudRate != 9600 {
		t.Fatalf("open fecbus result=%#v request=%+v", session, svc.openFecbus)
	}

	send := callMCP(t, handler, `{
		"jsonrpc":"2.0",
		"id":22,
		"method":"tools/call",
		"params":{"name":"fecbus_send_request","arguments":{"session_id":"fec-1","frame_type":0,"target_address":2,"source_address":1,"message_number":9,"function":44,"expect_answer":true}}
	}`)
	tx := send["structuredContent"].(map[string]any)["transaction"].(map[string]any)
	if tx["ID"] != "fec-tx-1" || svc.fecbusRequest.SessionID != "fec-1" || svc.fecbusRequest.Function != fb.FunctionQueryProtocolVersion {
		t.Fatalf("fecbus tx=%#v request=%+v", tx, svc.fecbusRequest)
	}

	callMCP(t, handler, `{
		"jsonrpc":"2.0",
		"id":23,
		"method":"tools/call",
		"params":{"name":"fecbus_start_slave","arguments":{"session_id":"fec-1","units":[{"address":2,"status_code":10,"auto_status_answer":true},{"address":3,"status_code":4,"auto_status_answer":true}]}}
	}`)
	if svc.startFecbus.SessionID != "fec-1" || len(svc.startFecbus.Units) != 2 || svc.startFecbus.Units[1].Address != 3 || svc.startFecbus.Units[1].DefaultStatus != fb.StatusBusy {
		t.Fatalf("start fecbus request = %+v", svc.startFecbus)
	}

	units := callMCP(t, handler, `{
		"jsonrpc":"2.0",
		"id":231,
		"method":"tools/call",
		"params":{"name":"fecbus_list_slave_units","arguments":{"session_id":"fec-1"}}
	}`)
	if svc.listFecbusID != "fec-1" || len(units["structuredContent"].(map[string]any)["units"].([]any)) != 1 {
		t.Fatalf("list fecbus units=%#v id=%q", units, svc.listFecbusID)
	}

	callMCP(t, handler, `{
		"jsonrpc":"2.0",
		"id":232,
		"method":"tools/call",
		"params":{"name":"fecbus_add_slave_unit","arguments":{"session_id":"fec-1","address":4,"status_code":8,"auto_status_answer":true}}
	}`)
	if svc.addFecbusID != "fec-1" || svc.addFecbusUnit.Address != 4 || svc.addFecbusUnit.DefaultStatus != fb.StatusProcessing {
		t.Fatalf("add fecbus unit=%+v id=%q", svc.addFecbusUnit, svc.addFecbusID)
	}

	callMCP(t, handler, `{
		"jsonrpc":"2.0",
		"id":233,
		"method":"tools/call",
		"params":{"name":"fecbus_remove_slave_unit","arguments":{"session_id":"fec-1","address":4}}
	}`)
	if svc.removeFecbusID != "fec-1" || svc.removeFecbusAdr != 4 {
		t.Fatalf("remove fecbus id/address = %q/%d", svc.removeFecbusID, svc.removeFecbusAdr)
	}

	query := callMCP(t, handler, `{
		"jsonrpc":"2.0",
		"id":24,
		"method":"tools/call",
		"params":{"name":"fecbus_query_frames","arguments":{"session_id":"fec-1","direction":"tx","search":"7e","limit":20,"custom":[{"code":8,"name":"厂商测试","fields":[{"key":"value","label":"测试值","offset":1,"length":2,"type":"uint16","endian":"little"}]}]}}
	}`)
	page := query["structuredContent"].(map[string]any)["page"].(map[string]any)
	if page["Total"].(float64) != 1 || svc.queryFecbus.Search != "7e" || svc.queryFecbus.Limit != 20 || len(svc.queryFecbus.Custom) != 1 {
		t.Fatalf("query fecbus page=%#v request=%+v", page, svc.queryFecbus)
	}

	callMCP(t, handler, `{
		"jsonrpc":"2.0",
		"id":25,
		"method":"tools/call",
		"params":{"name":"fecbus_update_slave_state","arguments":{"session_id":"fec-1","address":3,"status_code":4,"auto_status_answer":false}}
	}`)
	callMCP(t, handler, `{
		"jsonrpc":"2.0",
		"id":26,
		"method":"tools/call",
		"params":{"name":"fecbus_clear_frames","arguments":{"session_id":"fec-1"}}
	}`)
	callMCP(t, handler, `{
		"jsonrpc":"2.0",
		"id":27,
		"method":"tools/call",
		"params":{"name":"fecbus_stop_slave","arguments":{"session_id":"fec-1"}}
	}`)
	callMCP(t, handler, `{
		"jsonrpc":"2.0",
		"id":28,
		"method":"tools/call",
		"params":{"name":"fecbus_close_session","arguments":{"session_id":"fec-1"}}
	}`)
	if svc.updateFecbusID != "fec-1" || svc.updateFecbus.Address != 3 || svc.clearFecbus != "fec-1" || svc.stopFecbus != "fec-1" || svc.closeFecbus != "fec-1" {
		t.Fatalf("fecbus lifecycle update=%q/%+v clear=%q stop=%q close=%q", svc.updateFecbusID, svc.updateFecbus, svc.clearFecbus, svc.stopFecbus, svc.closeFecbus)
	}
}

func TestSerialSendToolCallsSerialService(t *testing.T) {
	t.Parallel()
	svc := &fakeSerialService{}
	handler := mcpHTTPHandler(newMCPServer(svc), true)

	result := callMCP(t, handler, `{
		"jsonrpc":"2.0",
		"id":2,
		"method":"tools/call",
		"params":{"name":"serial_send","arguments":{"port_id":"port-1","content":"4869","mode":"hex"}}
	}`)
	structured := result["structuredContent"].(map[string]any)
	if structured["bytes_written"].(float64) != 2 {
		t.Fatalf("bytes_written = %v, want 2", structured["bytes_written"])
	}
	if svc.lastSend.PortID != "port-1" || svc.lastSend.Content != "4869" || svc.lastSend.Mode != "hex" {
		t.Fatalf("last send = %+v", svc.lastSend)
	}
}

func TestSerialMonitorToolsCallSerialService(t *testing.T) {
	t.Parallel()
	svc := &fakeSerialService{}
	handler := mcpHTTPHandler(newMCPServer(svc), true)

	start := callMCP(t, handler, `{
		"jsonrpc":"2.0",
		"id":3,
		"method":"tools/call",
		"params":{"name":"serial_start_monitor","arguments":{"id":"mon-1","name":"监控","port":"/tmp/source","baud_rate":9600,"encoding":"utf-8"}}
	}`)
	monitorResult := start["structuredContent"].(map[string]any)["monitor"].(map[string]any)
	if monitorResult["ID"] != "mon-1" || monitorResult["ExternalPort"] != "/tmp/portweave-monitor" {
		t.Fatalf("monitor result = %#v", monitorResult)
	}
	if svc.startMonitor.ID != "mon-1" || svc.startMonitor.Port != "/tmp/source" || svc.startMonitor.Config.BaudRate != 9600 {
		t.Fatalf("start monitor request = %+v", svc.startMonitor)
	}

	query := callMCP(t, handler, `{
		"jsonrpc":"2.0",
		"id":4,
		"method":"tools/call",
		"params":{"name":"serial_query_monitor_frames","arguments":{"monitor_id":"mon-1","direction":"a_to_b","search":"OK","limit":20}}
	}`)
	page := query["structuredContent"].(map[string]any)["page"].(map[string]any)
	if page["Total"].(float64) != 1 {
		t.Fatalf("page = %#v", page)
	}
	if svc.queryMonitor.MonitorID != "mon-1" || svc.queryMonitor.Direction != monitor.DirectionAToB || svc.queryMonitor.Search != "OK" || svc.queryMonitor.Limit != 20 {
		t.Fatalf("query monitor request = %+v", svc.queryMonitor)
	}

	callMCP(t, handler, `{
		"jsonrpc":"2.0",
		"id":5,
		"method":"tools/call",
		"params":{"name":"serial_clear_monitor_frames","arguments":{"monitor_id":"mon-1"}}
	}`)
	callMCP(t, handler, `{
		"jsonrpc":"2.0",
		"id":6,
		"method":"tools/call",
		"params":{"name":"serial_stop_monitor","arguments":{"monitor_id":"mon-1"}}
	}`)
	callMCP(t, handler, `{
		"jsonrpc":"2.0",
		"id":7,
		"method":"tools/call",
		"params":{"name":"serial_delete_monitor","arguments":{"monitor_id":"mon-1"}}
	}`)
	if svc.clearMonitor != "mon-1" || svc.stopMonitor != "mon-1" || svc.deleteMonitor != "mon-1" {
		t.Fatalf("monitor operations = clear:%q stop:%q delete:%q", svc.clearMonitor, svc.stopMonitor, svc.deleteMonitor)
	}
}

func TestModbusToolsCallSerialService(t *testing.T) {
	t.Parallel()
	svc := &fakeSerialService{}
	handler := mcpHTTPHandler(newMCPServer(svc), true)

	open := callMCP(t, handler, `{
		"jsonrpc":"2.0",
		"id":8,
		"method":"tools/call",
		"params":{"name":"modbus_open_session","arguments":{"id":"mb-1","name":"主站","port":"/tmp/ttyM0","mode":"rtu","role":"master","baud_rate":19200,"timeout_ms":500,"retries":1}}
	}`)
	session := open["structuredContent"].(map[string]any)["session"].(map[string]any)
	if session["ID"] != "mb-1" || svc.openModbus.Config.PortName != "/tmp/ttyM0" || svc.openModbus.Config.BaudRate != 19200 {
		t.Fatalf("open session result=%#v request=%+v", session, svc.openModbus)
	}

	read := callMCP(t, handler, `{
		"jsonrpc":"2.0",
		"id":9,
		"method":"tools/call",
		"params":{"name":"modbus_read_registers","arguments":{"session_id":"mb-1","unit_id":9,"function":3,"address_mode":"zero-based","address":20,"quantity":2,"mappings":[{"address":20,"data_type":"uint16","word_order":"big","comment":"value"}]}}
	}`)
	if read["structuredContent"].(map[string]any)["result"].(map[string]any)["RawRegisters"].([]any)[0].(float64) != 42 {
		t.Fatalf("read result = %#v", read)
	}
	if svc.registerRead.SessionID != "mb-1" || svc.registerRead.UnitID != 9 || svc.registerRead.Function != mb.FunctionReadHoldingRegisters || svc.registerRead.Mappings[0].Comment != "value" {
		t.Fatalf("register read request = %+v", svc.registerRead)
	}

	callMCP(t, handler, `{
		"jsonrpc":"2.0",
		"id":10,
		"method":"tools/call",
		"params":{"name":"modbus_update_slave_unit_data","arguments":{"session_id":"mb-1","unit_id":2,"data_model":{"coils":[{"address":1,"value":true}],"holding_registers":[{"address":0,"value":77}]}}}
	}`)
	if svc.updateUnitID != 2 || len(svc.updateUnitData.Coils) != 1 || !svc.updateUnitData.Coils[0].Value || len(svc.updateUnitData.HoldingRegisters) != 1 || svc.updateUnitData.HoldingRegisters[0].Value != 77 {
		t.Fatalf("update unit request = unit:%d data:%+v", svc.updateUnitID, svc.updateUnitData)
	}

	callMCP(t, handler, `{
		"jsonrpc":"2.0",
		"id":11,
		"method":"tools/call",
		"params":{"name":"modbus_start_slave","arguments":{"session_id":"mb-1","unit_id":7,"units":[{"unit_id":7,"data_model":{"holding_registers":[{"address":3,"value":12}]}},{"unit_id":8,"data_model":{"coils":[{"address":5,"value":true}]}}]}}
	}`)
	if svc.startSlave.SessionID != "mb-1" || svc.startSlave.UnitID != 7 || len(svc.startSlave.Units) != 2 || svc.startSlave.Units[1].UnitID != 8 || !svc.startSlave.Units[1].DataModel.Coils[0].Value {
		t.Fatalf("start slave request = %+v", svc.startSlave)
	}

	callMCP(t, handler, `{
		"jsonrpc":"2.0",
		"id":12,
		"method":"tools/call",
		"params":{"name":"modbus_update_slave_data","arguments":{"session_id":"mb-1","data_model":{"input_registers":[{"address":10,"value":88}]}}}
	}`)
	if svc.updateSlaveID != "mb-1" || len(svc.updateSlaveData.InputRegisters) != 1 || svc.updateSlaveData.InputRegisters[0].Value != 88 {
		t.Fatalf("update slave request = id:%q data:%+v", svc.updateSlaveID, svc.updateSlaveData)
	}

	callMCP(t, handler, `{
		"jsonrpc":"2.0",
		"id":13,
		"method":"tools/call",
		"params":{"name":"modbus_add_slave_unit","arguments":{"session_id":"mb-1","unit_id":3,"data_model":{"discrete_inputs":[{"address":4,"value":true}]}}}
	}`)
	if svc.addSlaveID != "mb-1" || svc.addSlaveUnit.UnitID != 3 || len(svc.addSlaveUnit.DataModel.DiscreteInputs) != 1 || !svc.addSlaveUnit.DataModel.DiscreteInputs[0].Value {
		t.Fatalf("add slave unit request = id:%q unit:%+v", svc.addSlaveID, svc.addSlaveUnit)
	}

	list := callMCP(t, handler, `{
		"jsonrpc":"2.0",
		"id":14,
		"method":"tools/call",
		"params":{"name":"modbus_list_slave_units","arguments":{"session_id":"mb-1"}}
	}`)
	units := list["structuredContent"].(map[string]any)["units"].([]any)
	if svc.listSlaveID != "mb-1" || units[0].(map[string]any)["UnitID"].(float64) != 2 {
		t.Fatalf("list slave units result=%#v request id=%q", units, svc.listSlaveID)
	}

	callMCP(t, handler, `{
		"jsonrpc":"2.0",
		"id":15,
		"method":"tools/call",
		"params":{"name":"modbus_scan_unit_ids","arguments":{"session_id":"mb-1","unit_ids":[1,2,3],"function":3,"address":0,"quantity":1}}
	}`)
	if svc.unitScan.SessionID != "mb-1" || len(svc.unitScan.UnitIDs) != 3 || svc.unitScan.Function != mb.FunctionReadHoldingRegisters {
		t.Fatalf("unit scan request = %+v", svc.unitScan)
	}

	callMCP(t, handler, `{
		"jsonrpc":"2.0",
		"id":16,
		"method":"tools/call",
		"params":{"name":"modbus_scan_registers","arguments":{"session_id":"mb-1","unit_id":4,"function":4,"start_address":0,"end_address":32,"chunk_size":8}}
	}`)
	if svc.registerScan.SessionID != "mb-1" || svc.registerScan.UnitID != 4 || svc.registerScan.Function != mb.FunctionReadInputRegisters || svc.registerScan.ChunkSize != 8 {
		t.Fatalf("register scan request = %+v", svc.registerScan)
	}

	callMCP(t, handler, `{
		"jsonrpc":"2.0",
		"id":17,
		"method":"tools/call",
		"params":{"name":"modbus_remove_slave_unit","arguments":{"session_id":"mb-1","unit_id":3}}
	}`)
	callMCP(t, handler, `{
		"jsonrpc":"2.0",
		"id":18,
		"method":"tools/call",
		"params":{"name":"modbus_stop_slave","arguments":{"session_id":"mb-1"}}
	}`)
	callMCP(t, handler, `{
		"jsonrpc":"2.0",
		"id":19,
		"method":"tools/call",
		"params":{"name":"modbus_close_session","arguments":{"session_id":"mb-1"}}
	}`)
	if svc.removeSlaveID != "mb-1" || svc.removeUnitID != 3 || svc.stopSlave != "mb-1" || svc.closeModbus != "mb-1" {
		t.Fatalf("modbus lifecycle operations = remove:%q/%d stop:%q close:%q", svc.removeSlaveID, svc.removeUnitID, svc.stopSlave, svc.closeModbus)
	}
}

func TestModuleStartsAndStopsLocalHTTPServer(t *testing.T) {
	t.Parallel()
	cfg := config.Default()
	cfg.MCP.Port = freeTCPPort(t)
	mod := New(&fakeSerialService{})
	bus := eventbus.New()
	if err := mod.Init(context.Background(), module.Deps{
		Bus:    bus,
		Config: cfg,
		Logger: logging.Default(),
		Paths:  &platform.Paths{},
	}); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if err := mod.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}

	url := "http://" + mod.status.Address + cfg.MCP.Path
	waitForHTTPReachable(t, url)

	if err := mod.Stop(context.Background()); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	waitForServerClosed(t, url)
}

func callMCP(t *testing.T, handler http.Handler, payload string) map[string]any {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:39391/mcp", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("MCP status = %d body = %s", resp.Code, resp.Body.String())
	}
	var envelope map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode MCP response: %v body=%s", err, resp.Body.String())
	}
	if envelope["error"] != nil {
		t.Fatalf("MCP error response: %#v", envelope["error"])
	}
	result, ok := envelope["result"].(map[string]any)
	if !ok {
		t.Fatalf("MCP result = %#v, want object", envelope["result"])
	}
	if result["isError"] == true {
		t.Fatalf("MCP tool error response: %#v", result["content"])
	}
	return result
}

func callMCPTool(t *testing.T, handler http.Handler, id int, name string, arguments map[string]any) map[string]any {
	t.Helper()
	payload, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      name,
			"arguments": arguments,
		},
	})
	if err != nil {
		t.Fatalf("marshal MCP tool call: %v", err)
	}
	return callMCP(t, handler, string(payload))
}

func callMCPToolAllowError(t *testing.T, handler http.Handler, id int, name string, arguments map[string]any) map[string]any {
	t.Helper()
	payload, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      name,
			"arguments": arguments,
		},
	})
	if err != nil {
		t.Fatalf("marshal MCP tool call: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:39391/mcp", bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("MCP status = %d body = %s", resp.Code, resp.Body.String())
	}
	var envelope map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode MCP response: %v body=%s", err, resp.Body.String())
	}
	if rawErr := envelope["error"]; rawErr != nil {
		return map[string]any{"isError": true, "error": rawErr}
	}
	result, ok := envelope["result"].(map[string]any)
	if !ok {
		t.Fatalf("MCP result = %#v, want object", envelope["result"])
	}
	return result
}

func extractToolNames(t *testing.T, result map[string]any) map[string]bool {
	t.Helper()
	tools, ok := result["tools"].([]any)
	if !ok {
		t.Fatalf("tools result = %#v", result["tools"])
	}
	names := make(map[string]bool, len(tools))
	for _, item := range tools {
		tool, ok := item.(map[string]any)
		if !ok {
			t.Fatalf("tool = %#v, want object", item)
		}
		name, ok := tool["name"].(string)
		if !ok {
			t.Fatalf("tool name = %#v, want string", tool["name"])
		}
		names[name] = true
	}
	return names
}

func extractToolsByName(t *testing.T, result map[string]any) map[string]map[string]any {
	t.Helper()
	tools, ok := result["tools"].([]any)
	if !ok {
		t.Fatalf("tools result = %#v", result["tools"])
	}
	byName := make(map[string]map[string]any, len(tools))
	for _, item := range tools {
		tool, ok := item.(map[string]any)
		if !ok {
			t.Fatalf("tool = %#v, want object", item)
		}
		name, ok := tool["name"].(string)
		if !ok {
			t.Fatalf("tool name = %#v, want string", tool["name"])
		}
		byName[name] = tool
	}
	return byName
}

func requireToolAnnotations(t *testing.T, tools map[string]map[string]any, name string) map[string]any {
	t.Helper()
	tool, ok := tools[name]
	if !ok {
		t.Fatalf("tools/list missing %s in %#v", name, tools)
	}
	annotations, ok := tool["annotations"].(map[string]any)
	if !ok {
		t.Fatalf("%s annotations = %#v, want object", name, tool["annotations"])
	}
	return annotations
}

func annotationBool(t *testing.T, annotations map[string]any, key string, defaultValue bool) bool {
	t.Helper()
	raw, ok := annotations[key]
	if !ok {
		return defaultValue
	}
	value, ok := raw.(bool)
	if !ok {
		t.Fatalf("annotation %s = %#v, want bool", key, raw)
	}
	return value
}

func freeTCPPort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen free port: %v", err)
	}
	defer func() { _ = listener.Close() }()
	return listener.Addr().(*net.TCPAddr).Port
}

func waitMCPGraphBufferData(t *testing.T, handler http.Handler, graphID, nodeID, want string) map[string]any {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		bufferPage := callMCPTool(t, handler, 455, "serial_graph_query_node_buffer", map[string]any{"graph_id": graphID, "node_id": nodeID, "offset": 0, "length": 256})
		content := bufferPage["structuredContent"].(map[string]any)
		if content["data"] == want {
			return content
		}
		time.Sleep(20 * time.Millisecond)
	}
	bufferPage := callMCPTool(t, handler, 456, "serial_graph_query_node_buffer", map[string]any{"graph_id": graphID, "node_id": nodeID, "offset": 0, "length": 256})
	return bufferPage["structuredContent"].(map[string]any)
}

func waitForHTTPReachable(t *testing.T, url string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		req, err := http.NewRequest(http.MethodPost, url, bytes.NewBufferString(`{"jsonrpc":"2.0","id":1,"method":"ping","params":{}}`))
		if err != nil {
			t.Fatalf("new request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json, text/event-stream")
		resp, err := http.DefaultClient.Do(req)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode != http.StatusNotFound {
				return
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("%s was not reachable", url)
}

func waitForServerClosed(t *testing.T, url string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err != nil {
			return
		}
		_ = resp.Body.Close()
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("%s still accepts connections after Stop", url)
}
