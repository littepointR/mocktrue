package serial

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/littepointR/portweave/internal/core/eventbus"
	"github.com/littepointR/portweave/internal/modules/serial/manager"
	"github.com/littepointR/portweave/internal/modules/serial/monitor"
	"github.com/littepointR/portweave/internal/modules/serial/port"
	"github.com/littepointR/portweave/internal/modules/serial/porttest"
	"github.com/littepointR/portweave/internal/modules/serial/virtualserial"
	goserial "go.bug.st/serial"
)

func newMemoryServiceWithPair(t *testing.T, a, b string) (*Service, *porttest.MemoryBackend) {
	t.Helper()
	backend := porttest.NewMemoryBackend()
	backend.AddPair(a, b)
	return NewService(eventbus.New(), WithPortBackend(backend)), backend
}

func TestServicePing(t *testing.T) {
	t.Parallel()
	bus := eventbus.New()
	svc := NewService(bus)

	got, err := svc.Ping(context.Background(), "hi")
	if err != nil {
		t.Fatalf("Ping failed: %v", err)
	}
	if got != "pong:hi" {
		t.Fatalf("Ping = %q, want pong:hi", got)
	}
}

func TestServicePingEmptyRejects(t *testing.T) {
	t.Parallel()
	bus := eventbus.New()
	svc := NewService(bus)
	_, err := svc.Ping(context.Background(), "")
	if err == nil {
		t.Fatalf("Ping(\"\") must error")
	}
}

func TestServiceEnumeratePorts(t *testing.T) {
	t.Parallel()
	bus := eventbus.New()
	svc := NewService(bus)
	ports, err := svc.EnumeratePorts(context.Background())
	if err != nil {
		t.Fatalf("EnumeratePorts failed: %v", err)
	}
	if ports == nil {
		t.Fatalf("EnumeratePorts must not return nil")
	}
}

func TestServiceListPortsEmpty(t *testing.T) {
	t.Parallel()
	bus := eventbus.New()
	svc := NewService(bus)
	list := svc.ListPorts()
	if list == nil {
		t.Fatalf("ListPorts must not return nil")
	}
	if len(list) != 0 {
		t.Fatalf("empty ListPorts = %d, want 0", len(list))
	}
}

func TestServiceResetCountersRejectsMissingHandle(t *testing.T) {
	t.Parallel()
	svc := NewService(eventbus.New())
	if err := svc.ResetCounters("ghost"); err == nil {
		t.Fatalf("ResetCounters must reject missing handle")
	}
}

func TestServiceRestoreCountersRejectsInvalidInputs(t *testing.T) {
	t.Parallel()
	svc := NewService(eventbus.New())
	if err := svc.RestoreCounters("", 1, 1); err == nil {
		t.Fatalf("RestoreCounters must reject empty handle")
	}
	if err := svc.RestoreCounters("ghost", -1, 1); err == nil {
		t.Fatalf("RestoreCounters must reject negative rx count")
	}
	if err := svc.RestoreCounters("ghost", 1, -1); err == nil {
		t.Fatalf("RestoreCounters must reject negative tx count")
	}
	if err := svc.RestoreCounters("ghost", 1, 1); err == nil {
		t.Fatalf("RestoreCounters must reject missing handle")
	}
}

func TestServiceResetCountersClearsOpenHandleStats(t *testing.T) {
	svc, backend := newMemoryServiceWithPair(t, "memA", "memB")
	handle, err := svc.OpenPort(context.Background(), manager.OpenRequest{
		Config: porttest.DefaultSerialConfig("memA"),
	})
	if err != nil {
		t.Fatalf("OpenPort: %v", err)
	}
	defer func() { _ = svc.ClosePort(handle.ID) }()

	if _, err := svc.Send(SendRequest{PortID: handle.ID, Content: "tx", Mode: "ascii"}); err != nil {
		t.Fatalf("Send: %v", err)
	}
	conn, err := backend.Open(porttest.DefaultSerialConfig("memB"))
	if err != nil {
		t.Fatalf("Open memory peer: %v", err)
	}
	defer func() { _ = conn.Close() }()
	if _, err := conn.Write([]byte("rx")); err != nil {
		t.Fatalf("Write rx: %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	before := svc.ListPorts()
	if len(before) != 1 {
		t.Fatalf("ListPorts before reset len = %d, want 1", len(before))
	}
	if before[0].RxBytes == 0 || before[0].TxBytes == 0 {
		t.Fatalf("counters before reset = rx %d tx %d, want both non-zero", before[0].RxBytes, before[0].TxBytes)
	}

	if err := svc.ResetCounters(handle.ID); err != nil {
		t.Fatalf("ResetCounters: %v", err)
	}

	after := svc.ListPorts()
	if len(after) != 1 {
		t.Fatalf("ListPorts after reset len = %d, want 1", len(after))
	}
	if after[0].RxBytes != 0 || after[0].TxBytes != 0 {
		t.Fatalf("counters after reset = rx %d tx %d, want 0/0", after[0].RxBytes, after[0].TxBytes)
	}
}

func TestServiceRestoreCountersSetsOpenHandleStats(t *testing.T) {
	svc, _ := newMemoryServiceWithPair(t, "memA", "memB")
	handle, err := svc.OpenPort(context.Background(), manager.OpenRequest{
		Config: porttest.DefaultSerialConfig("memA"),
	})
	if err != nil {
		t.Fatalf("OpenPort: %v", err)
	}
	defer func() { _ = svc.ClosePort(handle.ID) }()

	if err := svc.RestoreCounters(handle.ID, 42, 17); err != nil {
		t.Fatalf("RestoreCounters: %v", err)
	}

	status := svc.ListPorts()
	if len(status) != 1 {
		t.Fatalf("ListPorts len = %d, want 1", len(status))
	}
	if status[0].RxBytes != 42 || status[0].TxBytes != 17 {
		t.Fatalf("restored counters = rx %d tx %d, want 42/17", status[0].RxBytes, status[0].TxBytes)
	}
}

func TestServiceQueryPageNonExistent(t *testing.T) {
	t.Parallel()
	bus := eventbus.New()
	svc := NewService(bus)
	_, err := svc.QueryPage("ghost", 0, 100)
	if err == nil {
		t.Fatalf("QueryPage non-existent must error")
	}
}

func TestServiceCreateVirtualPortRejectsEmptyInputs(t *testing.T) {
	t.Parallel()
	bus := eventbus.New()
	svc := NewService(bus)

	if _, err := svc.CreateVirtualPort(context.Background(), "", "ttyVTest"); err == nil {
		t.Fatalf("CreateVirtualPort must reject empty ID")
	}
	if _, err := svc.CreateVirtualPort(context.Background(), "id", ""); err == nil {
		t.Fatalf("CreateVirtualPort must reject empty port name")
	}
}

func TestServiceVirtualBackendInjectionAndStatus(t *testing.T) {
	t.Parallel()
	backend := &serviceVirtualBackend{
		status: virtualserial.BackendStatus{
			Name:          "fake",
			Available:     true,
			Message:       "ready",
			RequiresAdmin: true,
		},
	}
	svc := NewService(eventbus.New(), WithVirtualBackend(backend))

	status := svc.GetVirtualSerialBackendStatus(context.Background())
	if status.Name != "fake" || !status.Available || !status.RequiresAdmin {
		t.Fatalf("status = %+v", status)
	}

	pair, err := svc.CreateVirtualPair(context.Background(), "pair-1", "A", "B")
	if err != nil {
		t.Fatalf("CreateVirtualPair: %v", err)
	}
	if pair.Port1 != "A" || pair.Port2 != "B" {
		t.Fatalf("pair = %+v", pair)
	}
	if got := len(svc.ListVirtualPorts()); got != 0 {
		t.Fatalf("ListVirtualPorts after pair = %d, want 0", got)
	}

	vport, err := svc.CreateVirtualPort(context.Background(), "port-1", "COM10")
	if err != nil {
		t.Fatalf("CreateVirtualPort: %v", err)
	}
	if vport.Port != "COM10" {
		t.Fatalf("virtual port = %+v", vport)
	}
	ports := svc.ListVirtualPorts()
	if len(ports) != 1 || ports[0].ID != "port-1" || ports[0].Port != "COM10" {
		t.Fatalf("ListVirtualPorts = %+v", ports)
	}
}

func TestServiceSendRejectsInvalidRequestsAndHexMode(t *testing.T) {
	svc := NewService(eventbus.New())
	if _, err := svc.Send(SendRequest{Content: "x"}); err == nil {
		t.Fatalf("Send must reject empty port IDs")
	}
	if _, err := svc.Send(SendRequest{PortID: "missing"}); err == nil {
		t.Fatalf("Send must reject empty content")
	}
	if _, err := svc.Send(SendRequest{PortID: "missing", Content: "zz", Mode: "hex"}); err == nil {
		t.Fatalf("Send must reject malformed hex content")
	}
	if _, err := svc.Send(SendRequest{PortID: "missing", Content: "é", Encoding: "ascii"}); err == nil {
		t.Fatalf("Send must reject content that cannot be encoded")
	}

	memorySvc, _ := newMemoryServiceWithPair(t, "sendA", "sendB")
	handle, err := memorySvc.OpenPort(context.Background(), manager.OpenRequest{Config: porttest.DefaultSerialConfig("sendA")})
	if err != nil {
		t.Fatalf("OpenPort: %v", err)
	}
	defer func() { _ = memorySvc.ClosePort(handle.ID) }()
	written, err := memorySvc.Send(SendRequest{PortID: handle.ID, Content: "41 42", Mode: "hex"})
	if err != nil {
		t.Fatalf("Send hex returned error: %v", err)
	}
	if written != 2 {
		t.Fatalf("Send hex wrote %d bytes, want 2", written)
	}
}

type serviceVirtualBackend struct {
	status virtualserial.BackendStatus
}

func (b *serviceVirtualBackend) Name() string { return "fake" }

func (b *serviceVirtualBackend) Status(context.Context) virtualserial.BackendStatus {
	return b.status
}

func (b *serviceVirtualBackend) CreatePair(_ context.Context, pairID, port1Name, port2Name string) (*virtualserial.VirtualPair, error) {
	return &virtualserial.VirtualPair{ID: pairID, Port1: port1Name, Port2: port2Name}, nil
}

func (b *serviceVirtualBackend) CreatePort(_ context.Context, portID, publicName string) (*virtualserial.VirtualPair, error) {
	return &virtualserial.VirtualPair{ID: portID, Port1: publicName, Port2: publicName + "-peer"}, nil
}

func TestServiceHelpersCoverNilVirtualManagerAndTokenFallbacks(t *testing.T) {
	svc := NewService(eventbus.New())
	svc.vmgr = nil
	if got := svc.monitorEndpoint("COM1"); got != "COM1" {
		t.Fatalf("monitorEndpoint without virtual manager = %q, want COM1", got)
	}
	if svc.monitorPortInUse("") {
		t.Fatalf("monitorPortInUse must ignore empty port names")
	}
	if svc.bridgePortInUse("COM1") {
		t.Fatalf("bridgePortInUse without virtual manager must be false")
	}
	if id := autoVirtualPortID("!!!"); id == "" {
		t.Fatalf("autoVirtualPortID must fall back to a non-empty monitor token")
	}
	if name := autoVirtualPortName(string(filepath.Separator), "???"); name == "" {
		t.Fatalf("autoVirtualPortName must fall back to non-empty port and monitor tokens")
	}
}

func TestServiceOpenPortReturnsExistingHandleForDuplicatePort(t *testing.T) {
	svc, _ := newMemoryServiceWithPair(t, "memA", "memB")
	req := manager.OpenRequest{
		Config: porttest.DefaultSerialConfig("memA"),
	}

	first, err := svc.OpenPort(context.Background(), req)
	if err != nil {
		t.Fatalf("first OpenPort failed: %v", err)
	}
	defer func() { _ = svc.ClosePort(first.ID) }()

	second, err := svc.OpenPort(context.Background(), req)
	if err != nil {
		t.Fatalf("duplicate OpenPort should return existing handle, got error: %v", err)
	}
	if second.ID != first.ID {
		t.Fatalf("duplicate OpenPort ID = %q, want existing %q", second.ID, first.ID)
	}
}

func TestServiceCleanupClearsVirtualPortsAndBridges(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping socat integration test in short mode")
	}

	svc := NewService(eventbus.New())
	pairA, err := svc.CreateVirtualPair(context.Background(), "cleanup-a", "cleanupA1", "cleanupA2")
	if err != nil {
		t.Skipf("socat not available: %v", err)
	}
	if _, err := svc.CreateVirtualPair(context.Background(), "cleanup-b", "cleanupB1", "cleanupB2"); err != nil {
		t.Fatalf("CreateVirtualPair cleanup-b: %v", err)
	}
	if _, err := svc.CreateBridge("cleanup-bridge", pairA.Port1, pairA.Port2, 115200); err != nil {
		t.Fatalf("CreateBridge: %v", err)
	}

	if got := len(svc.ListVirtualPairs()); got != 2 {
		t.Fatalf("virtual pairs before Cleanup = %d, want 2", got)
	}
	if got := len(svc.ListBridges()); got != 1 {
		t.Fatalf("bridges before Cleanup = %d, want 1", got)
	}

	svc.cleanup()
	svc.cleanup()

	if got := len(svc.ListVirtualPairs()); got != 0 {
		t.Fatalf("virtual pairs after Cleanup = %d, want 0", got)
	}
	if got := len(svc.ListBridges()); got != 0 {
		t.Fatalf("bridges after Cleanup = %d, want 0", got)
	}
}

func TestServiceBridgeConnectsUserFacingVirtualPorts(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping socat integration test in short mode")
	}

	svc := NewService(eventbus.New())
	defer svc.CleanupVirtual()

	portA, err := svc.CreateVirtualPort(context.Background(), "user-facing-a", "userFacingA")
	if err != nil {
		t.Skipf("socat not available: %v", err)
	}
	portB, err := svc.CreateVirtualPort(context.Background(), "user-facing-b", "userFacingB")
	if err != nil {
		t.Fatalf("CreateVirtualPort B: %v", err)
	}

	if _, err := svc.CreateBridge("user-facing-bridge", portA.Port, portB.Port, 115200); err != nil {
		t.Fatalf("CreateBridge: %v", err)
	}
	bridges := svc.ListBridges()
	if len(bridges) != 1 {
		t.Fatalf("ListBridges length = %d, want 1", len(bridges))
	}
	if bridges[0].Port1 != portA.Port || bridges[0].Port2 != portB.Port {
		t.Fatalf("bridge ports = %q/%q, want %q/%q",
			bridges[0].Port1, bridges[0].Port2, portA.Port, portB.Port)
	}
	time.Sleep(150 * time.Millisecond)

	sender, err := goserial.Open(portA.Port, &goserial.Mode{BaudRate: 115200})
	if err != nil {
		t.Fatalf("Open sender: %v", err)
	}
	defer func() { _ = sender.Close() }()

	receiver, err := goserial.Open(portB.Port, &goserial.Mode{BaudRate: 115200})
	if err != nil {
		t.Fatalf("Open receiver: %v", err)
	}
	defer func() { _ = receiver.Close() }()
	if err := receiver.SetReadTimeout(2 * time.Second); err != nil {
		t.Fatalf("SetReadTimeout receiver: %v", err)
	}

	payload := []byte("visible bridge")
	if _, err := sender.Write(payload); err != nil {
		t.Fatalf("Write sender: %v", err)
	}

	buf := make([]byte, len(payload))
	total := 0
	deadline := time.Now().Add(3 * time.Second)
	for total < len(payload) && time.Now().Before(deadline) {
		n, err := receiver.Read(buf[total:])
		if err != nil {
			break
		}
		total += n
	}
	if string(buf[:total]) != string(payload) {
		t.Fatalf("received %q, want %q", buf[:total], payload)
	}
}

func TestServiceAutoVirtualMonitorExposesSingleVirtualPort(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping socat integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	devicePair, err := port.StartVirtualPair(ctx)
	if err != nil {
		t.Skipf("socat not available: %v", err)
	}
	defer devicePair.Stop()

	svc := NewService(eventbus.New())
	defer svc.cleanup()

	session, err := svc.StartAutoVirtualMonitor(ctx, AutoVirtualMonitorRequest{
		ID:   "auto-monitor",
		Name: "自动监听",
		Port: devicePair.Port1,
		Config: port.SerialConfig{
			BaudRate: 115200,
			DataBits: 8,
			StopBits: "1",
			Parity:   "none",
			FlowMode: "none",
		},
		Encoding: "utf-8",
	})
	if err != nil {
		t.Fatalf("StartAutoVirtualMonitor: %v", err)
	}
	if session.PortA != devicePair.Port1 {
		t.Fatalf("PortA = %q, want monitored port %q", session.PortA, devicePair.Port1)
	}
	if session.ExternalPort == "" || session.AutoVirtualPortID == "" {
		t.Fatalf("auto monitor session missing external virtual port info: %+v", session)
	}
	if session.PortB != session.ExternalPort {
		t.Fatalf("PortB = %q, want external port %q", session.PortB, session.ExternalPort)
	}
	if got := len(svc.ListVirtualPorts()); got != 1 {
		t.Fatalf("virtual ports after start = %d, want 1", got)
	}

	hardware, err := goserial.Open(devicePair.Port2, &goserial.Mode{BaudRate: 115200})
	if err != nil {
		t.Fatalf("Open hardware peer: %v", err)
	}
	defer func() { _ = hardware.Close() }()
	if err := hardware.SetReadTimeout(2 * time.Second); err != nil {
		t.Fatalf("SetReadTimeout hardware: %v", err)
	}

	external, err := goserial.Open(session.ExternalPort, &goserial.Mode{BaudRate: 115200})
	if err != nil {
		t.Fatalf("Open external virtual port: %v", err)
	}
	defer func() { _ = external.Close() }()
	if err := external.SetReadTimeout(2 * time.Second); err != nil {
		t.Fatalf("SetReadTimeout external: %v", err)
	}

	if _, err := external.Write([]byte("to-device")); err != nil {
		t.Fatalf("Write external: %v", err)
	}
	if got := readSerialPayload(t, hardware, len("to-device")); got != "to-device" {
		t.Fatalf("hardware read %q", got)
	}

	if _, err := hardware.Write([]byte("to-client")); err != nil {
		t.Fatalf("Write hardware: %v", err)
	}
	if got := readSerialPayload(t, external, len("to-client")); got != "to-client" {
		t.Fatalf("external read %q", got)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		page, err := svc.QueryMonitorFrames(monitor.QueryRequest{MonitorID: "auto-monitor", Limit: 10})
		if err != nil {
			t.Fatalf("QueryMonitorFrames: %v", err)
		}
		if page.Total >= 2 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	if err := svc.StopMonitor("auto-monitor"); err != nil {
		t.Fatalf("StopMonitor: %v", err)
	}
	if got := len(svc.ListVirtualPorts()); got != 0 {
		t.Fatalf("virtual ports after stop = %d, want 0", got)
	}
}

func readSerialPayload(t *testing.T, reader interface{ Read([]byte) (int, error) }, length int) string {
	t.Helper()
	buf := make([]byte, length)
	total := 0
	deadline := time.Now().Add(3 * time.Second)
	for total < length && time.Now().Before(deadline) {
		n, err := reader.Read(buf[total:])
		if err != nil {
			break
		}
		total += n
	}
	return string(buf[:total])
}
