package serial

import (
	"context"
	"testing"
	"time"

	"github.com/littepointR/mocktrue/internal/core/eventbus"
	"github.com/littepointR/mocktrue/internal/modules/serial/manager"
	"github.com/littepointR/mocktrue/internal/modules/serial/monitor"
	"github.com/littepointR/mocktrue/internal/modules/serial/port"
	goserial "go.bug.st/serial"
)

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
	if testing.Short() {
		t.Skip("skipping socat integration test in short mode")
	}
	pair, err := port.StartVirtualPair(context.Background())
	if err != nil {
		t.Skipf("socat not available: %v", err)
	}
	defer pair.Stop()

	svc := NewService(eventbus.New())
	handle, err := svc.OpenPort(context.Background(), manager.OpenRequest{
		Config: port.SerialConfig{PortName: pair.Port1, BaudRate: 115200},
	})
	if err != nil {
		t.Fatalf("OpenPort: %v", err)
	}
	defer svc.ClosePort(handle.ID)

	if _, err := svc.Send(SendRequest{PortID: handle.ID, Content: "tx", Mode: "ascii"}); err != nil {
		t.Fatalf("Send: %v", err)
	}
	conn, err := port.OpenForTest(pair.Port2, 115200)
	if err != nil {
		t.Fatalf("OpenForTest: %v", err)
	}
	defer conn.Close()
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
	if testing.Short() {
		t.Skip("skipping socat integration test in short mode")
	}
	pair, err := port.StartVirtualPair(context.Background())
	if err != nil {
		t.Skipf("socat not available: %v", err)
	}
	defer pair.Stop()

	svc := NewService(eventbus.New())
	handle, err := svc.OpenPort(context.Background(), manager.OpenRequest{
		Config: port.SerialConfig{PortName: pair.Port1, BaudRate: 115200},
	})
	if err != nil {
		t.Fatalf("OpenPort: %v", err)
	}
	defer svc.ClosePort(handle.ID)

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

func TestServiceOpenPortReturnsExistingHandleForDuplicatePort(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping socat integration test in short mode")
	}
	pair, err := port.StartVirtualPair(context.Background())
	if err != nil {
		t.Skipf("socat not available: %v", err)
	}
	defer pair.Stop()

	svc := NewService(eventbus.New())
	req := manager.OpenRequest{
		Config: port.SerialConfig{PortName: pair.Port1, BaudRate: 115200},
	}

	first, err := svc.OpenPort(context.Background(), req)
	if err != nil {
		t.Fatalf("first OpenPort failed: %v", err)
	}
	defer svc.ClosePort(first.ID)

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
	defer sender.Close()

	receiver, err := goserial.Open(portB.Port, &goserial.Mode{BaudRate: 115200})
	if err != nil {
		t.Fatalf("Open receiver: %v", err)
	}
	defer receiver.Close()
	receiver.SetReadTimeout(2 * time.Second)

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
	defer hardware.Close()
	hardware.SetReadTimeout(2 * time.Second)

	external, err := goserial.Open(session.ExternalPort, &goserial.Mode{BaudRate: 115200})
	if err != nil {
		t.Fatalf("Open external virtual port: %v", err)
	}
	defer external.Close()
	external.SetReadTimeout(2 * time.Second)

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
