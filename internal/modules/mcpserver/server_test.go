package mcpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/suyue/mocktrue/internal/core/config"
	"github.com/suyue/mocktrue/internal/core/eventbus"
	"github.com/suyue/mocktrue/internal/core/logging"
	"github.com/suyue/mocktrue/internal/core/module"
	"github.com/suyue/mocktrue/internal/core/platform"
	"github.com/suyue/mocktrue/internal/modules/serial"
	"github.com/suyue/mocktrue/internal/modules/serial/buffer"
	"github.com/suyue/mocktrue/internal/modules/serial/manager"
	"github.com/suyue/mocktrue/internal/modules/serial/monitor"
	"github.com/suyue/mocktrue/internal/modules/serial/port"
)

type fakeSerialService struct {
	ports         []port.PortInfo
	handles       []manager.HandleStatus
	lastSend      serial.SendRequest
	resetPortID   string
	queryPortID   string
	queryOffset   int64
	queryLength   int
	virtualPorts  []serial.VirtualPortInfo
	bridges       []serial.BridgeInfo
	monitors      []monitor.SessionInfo
	startMonitor  serial.AutoVirtualMonitorRequest
	stopMonitor   string
	deleteMonitor string
	clearMonitor  string
	queryMonitor  monitor.QueryRequest
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
	} {
		if !names[want] {
			t.Fatalf("tools/list missing %s in %#v", want, names)
		}
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
