package mcpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/suyue/mocktrue/internal/core/config"
	"github.com/suyue/mocktrue/internal/core/eventbus"
	"github.com/suyue/mocktrue/internal/core/logging"
	"github.com/suyue/mocktrue/internal/core/module"
	"github.com/suyue/mocktrue/internal/core/platform"
	"github.com/suyue/mocktrue/internal/modules/serial"
	"github.com/suyue/mocktrue/internal/modules/serial/buffer"
	fb "github.com/suyue/mocktrue/internal/modules/serial/fecbus"
	"github.com/suyue/mocktrue/internal/modules/serial/manager"
	mb "github.com/suyue/mocktrue/internal/modules/serial/modbus"
	"github.com/suyue/mocktrue/internal/modules/serial/monitor"
	"github.com/suyue/mocktrue/internal/modules/serial/port"
)

type fakeSerialService struct {
	ports           []port.PortInfo
	handles         []manager.HandleStatus
	lastSend        serial.SendRequest
	resetPortID     string
	queryPortID     string
	queryOffset     int64
	queryLength     int
	virtualPorts    []serial.VirtualPortInfo
	bridges         []serial.BridgeInfo
	monitors        []monitor.SessionInfo
	startMonitor    serial.AutoVirtualMonitorRequest
	stopMonitor     string
	deleteMonitor   string
	clearMonitor    string
	queryMonitor    monitor.QueryRequest
	modbusSessions  []mb.SessionInfo
	openModbus      mb.OpenSessionRequest
	closeModbus     string
	masterRequest   mb.MasterRequest
	startSlave      mb.StartSlaveRequest
	stopSlave       string
	updateSlaveID   string
	updateSlaveData mb.DataModelSnapshot
	addSlaveUnit    mb.SlaveUnitSnapshot
	addSlaveID      string
	removeSlaveID   string
	removeUnitID    byte
	listSlaveID     string
	updateUnitID    byte
	updateUnitData  mb.DataModelSnapshot
	unitScan        mb.UnitScanRequest
	registerRead    mb.RegisterReadRequest
	registerScan    mb.RegisterScanRequest
	fecbusSessions  []fb.SessionInfo
	openFecbus      fb.OpenSessionRequest
	closeFecbus     string
	fecbusRequest   fb.SendRequest
	startFecbus     fb.StartSlaveRequest
	stopFecbus      string
	updateFecbusID  string
	updateFecbus    fb.SlaveState
	addFecbusID     string
	addFecbusUnit   fb.SlaveUnitState
	removeFecbusID  string
	removeFecbusAdr byte
	listFecbusID    string
	queryFecbus     fb.QueryRequest
	clearFecbus     string
	graphs          []serial.SerialGraphRuntimeInfo
	startGraph      serial.SerialGraphStartRequest
	stopGraph       string
	statusGraph     string
	graphSend       serial.SerialGraphSendRequest
	graphBuffer     serial.SerialGraphBufferQuery
	graphFrames     serial.SerialGraphFrameQuery
	clearGraphID    string
	clearGraphNode  string
	resetGraphID    string
	resetGraphNode  string
}

func (f *fakeSerialService) EnumeratePorts(context.Context) ([]port.PortInfo, error) {
	return f.ports, nil
}

func (f *fakeSerialService) OpenPort(_ context.Context, req manager.OpenRequest) (*manager.HandleStatus, error) {
	status := &manager.HandleStatus{ID: "port-1", Config: req.Config, IsOpen: true}
	f.handles = append(f.handles, *status)
	return status, nil
}

func (f *fakeSerialService) ClosePort(string) error { return nil }
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

func (f *fakeSerialService) CreateVirtualPort(context.Context, string, string) (*serial.VirtualPortInfo, error) {
	return &serial.VirtualPortInfo{ID: "v1", Port: "/tmp/ttyV1"}, nil
}

func (f *fakeSerialService) DeleteVirtualPort(string) error { return nil }
func (f *fakeSerialService) ListVirtualPorts() []serial.VirtualPortInfo {
	return f.virtualPorts
}

func (f *fakeSerialService) CreateVirtualPair(context.Context, string, string, string) (*serial.VirtualPairInfo, error) {
	return &serial.VirtualPairInfo{ID: "pair1", Port1: "/tmp/a", Port2: "/tmp/b"}, nil
}

func (f *fakeSerialService) DeleteVirtualPair(string) error { return nil }
func (f *fakeSerialService) ListVirtualPairs() []serial.VirtualPairInfo {
	return nil
}

func (f *fakeSerialService) CreateBridge(string, string, string, int) (*serial.BridgeInfo, error) {
	return &serial.BridgeInfo{ID: "bridge1", Port1: "/tmp/a", Port2: "/tmp/b", BaudRate: 115200}, nil
}

func (f *fakeSerialService) DeleteBridge(string) error { return nil }
func (f *fakeSerialService) ListBridges() []serial.BridgeInfo {
	return f.bridges
}

func (f *fakeSerialService) CleanupVirtual() {}

func (f *fakeSerialService) StartAutoVirtualMonitor(_ context.Context, req serial.AutoVirtualMonitorRequest) (*monitor.SessionInfo, error) {
	f.startMonitor = req
	session := &monitor.SessionInfo{
		ID:                req.ID,
		Name:              req.Name,
		Provider:          monitor.ProviderBridge,
		PortA:             req.Port,
		PortB:             "/tmp/mocktrue-monitor",
		ExternalPort:      "/tmp/mocktrue-monitor",
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
		Nodes:  []serial.SerialGraphNodeStatus{{ID: "receiver", Type: "serial.receiver", Status: serial.SerialGraphStatusRunning}},
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

func TestListenAddressRejectsNonLocalHosts(t *testing.T) {
	t.Parallel()
	for _, host := range []string{"0.0.0.0", "::", "192.168.1.10", "example.com"} {
		if _, err := listenAddress(config.MCPConfig{Host: host, Port: 39391}); err == nil {
			t.Fatalf("listenAddress(%q) error = nil, want non-local host rejected", host)
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

func TestSerialGraphToolsExposeCatalogAndValidateTopology(t *testing.T) {
	t.Parallel()
	handler := mcpHTTPHandler(newMCPServer(&fakeSerialService{}), true)

	catalog := callMCP(t, handler, `{
		"jsonrpc":"2.0",
		"id":40,
		"method":"tools/call",
		"params":{"name":"serial_graph_provider_catalog","arguments":{}}
	}`)
	providers := catalog["structuredContent"].(map[string]any)["providers"].([]any)
	if len(providers) < 10 {
		t.Fatalf("provider count = %d, want at least 10", len(providers))
	}
	providerTypes := make(map[string]bool, len(providers))
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
		providerDefaults[providerType] = provider["default_config"].(map[string]any)
	}
	for _, want := range []string{"serial.physical", "serial.virtual", "serial.tap", "serial.modbus.master", "serial.fecbus.slave"} {
		if !providerTypes[want] {
			t.Fatalf("catalog missing %s in %#v", want, providerTypes)
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
	if got := providerDefaults["serial.sender"]; !reflect.DeepEqual(got, map[string]any{
		"mode":       "ascii",
		"encoding":   "utf-8",
		"payload":    "",
		"autoSend":   false,
		"intervalMs": float64(1000),
	}) {
		t.Fatalf("serial.sender default_config = %#v, want frontend-compatible defaults", got)
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
				{"id":"sender","type":"serial.sender","position":{"x":20,"y":20},"config":{}},
				{"id":"receiver","type":"serial.receiver","position":{"x":260,"y":20},"config":{}}
			],
			"edges":[{"id":"edge-1","source":"sender","source_handle":"out","target":"receiver","target_handle":"in"}]
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
				{"id":"sender","type":"serial.sender","position":{"x":20,"y":20},"config":{}},
				{"id":"receiver","type":"serial.receiver","position":{"x":260,"y":20},"config":{}}
			],
			"edges":[{"id":"edge-1","source":"sender","sourceHandle":"out","target":"receiver","targetHandle":"in"}]
		}}
	}`)
	if frontEndShape["structuredContent"].(map[string]any)["valid"] != true {
		t.Fatalf("frontend-shaped graph result = %#v", frontEndShape["structuredContent"])
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
				{"id":"sender","type":"serial.sender","position":{"x":0,"y":0},"config":{"mode":"ascii"}},
				{"id":"receiver","type":"serial.receiver","position":{"x":260,"y":0},"config":{}}
			],
			"edges":[{"id":"edge-1","source":"sender","sourceHandle":"out","target":"receiver","targetHandle":"in"}]
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
		"params":{"name":"serial_graph_send","arguments":{"graph_id":"graph-mcp","node_id":"sender","content":"hello","mode":"ascii"}}
	}`)
	if send["structuredContent"].(map[string]any)["bytes_written"] != float64(5) {
		t.Fatalf("send content = %#v", send["structuredContent"])
	}
	if svc.graphSend.GraphID != "graph-mcp" || svc.graphSend.NodeID != "sender" {
		t.Fatalf("graph send args = %#v", svc.graphSend)
	}

	bufferPage := callMCP(t, handler, `{
		"jsonrpc":"2.0",
		"id":46,
		"method":"tools/call",
		"params":{"name":"serial_graph_query_node_buffer","arguments":{"graph_id":"graph-mcp","node_id":"receiver","offset":0,"length":16}}
	}`)
	bufferContent := bufferPage["structuredContent"].(map[string]any)
	if bufferContent["data"] != "hello" || bufferContent["hex"] != "68656c6c6f" {
		t.Fatalf("buffer content = %#v", bufferContent)
	}
	if svc.graphBuffer.NodeID != "receiver" {
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
		"params":{"name":"serial_graph_clear_node_buffer","arguments":{"graph_id":"graph-mcp","node_id":"receiver"}}
	}`)
	if svc.clearGraphID != "graph-mcp" || svc.clearGraphNode != "receiver" {
		t.Fatalf("clear args = %s/%s", svc.clearGraphID, svc.clearGraphNode)
	}

	callMCP(t, handler, `{
		"jsonrpc":"2.0",
		"id":50,
		"method":"tools/call",
		"params":{"name":"serial_graph_reset_node_counters","arguments":{"graph_id":"graph-mcp","node_id":"receiver"}}
	}`)
	if svc.resetGraphID != "graph-mcp" || svc.resetGraphNode != "receiver" {
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
					{"id":"receiver","type":"serial.receiver","position":{"x":260,"y":0},"config":{}}
				],
				"edges":[{"id":"edge-1","source":"monitor","source_handle":"frames","target":"receiver","target_handle":"in"}]
			}`,
			wantError: "output port not found",
		},
		{
			name: "non tap fan out",
			arguments: `{
				"nodes":[
					{"id":"sender","type":"serial.sender","position":{"x":0,"y":0},"config":{}},
					{"id":"rx-a","type":"serial.receiver","position":{"x":260,"y":0},"config":{}},
					{"id":"rx-b","type":"serial.receiver","position":{"x":260,"y":120},"config":{}}
				],
				"edges":[
					{"id":"edge-1","source":"sender","source_handle":"out","target":"rx-a","target_handle":"in"},
					{"id":"edge-2","source":"sender","source_handle":"out","target":"rx-b","target_handle":"in"}
				]
			}`,
			wantError: "fan-out requires a tap node",
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
	if monitorResult["ID"] != "mon-1" || monitorResult["ExternalPort"] != "/tmp/mocktrue-monitor" {
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

func freeTCPPort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen free port: %v", err)
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port
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
