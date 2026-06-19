package serial

import (
	"context"
	"testing"
	"time"

	"github.com/suyue/mocktrue/internal/core/eventbus"
	"github.com/suyue/mocktrue/internal/modules/serial/manager"
	"github.com/suyue/mocktrue/internal/modules/serial/port"
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
