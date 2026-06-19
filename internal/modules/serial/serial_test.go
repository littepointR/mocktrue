package serial

import (
	"context"
	"testing"

	mterrors "github.com/suyue/mocktrue/internal/core/errors"
	"github.com/suyue/mocktrue/internal/core/eventbus"
	"github.com/suyue/mocktrue/internal/modules/serial/buffer"
	"github.com/suyue/mocktrue/internal/modules/serial/manager"
	"github.com/suyue/mocktrue/internal/modules/serial/port"
)

func TestIDAndManifest(t *testing.T) {
	t.Parallel()
	m := New()
	if m.ID() != "serial" {
		t.Fatalf("ID = %q, want serial", m.ID())
	}
	mf := m.Manifest()
	if mf.ID != "serial" || mf.Version == "" {
		t.Fatalf("Manifest = %+v, want ID serial and non-empty version", mf)
	}
	if mf.Frontend.ActivityIcon != "serial" || mf.Frontend.ActivityTitle == "" {
		t.Fatalf("Frontend contribution missing: %+v", mf.Frontend)
	}
	if len(mf.Frontend.Views) == 0 {
		t.Fatalf("Manifest must contribute at least one view")
	}
}

func TestServicesWrappedReturnsOneService(t *testing.T) {
	t.Parallel()
	m := New()
	if err := m.Init(context.Background(), testDeps()); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	wrapped := m.ServicesWrapped()
	if len(wrapped) != 1 {
		t.Fatalf("ServicesWrapped len = %d, want 1", len(wrapped))
	}
	if wrapped[0].Instance() == nil {
		t.Fatalf("wrapped service instance must not be nil")
	}
	if _, ok := wrapped[0].Instance().(*Service); !ok {
		t.Fatalf("wrapped instance must be *serial.Service, got %T", wrapped[0].Instance())
	}
}

func TestModuleInitWiresEmptyService(t *testing.T) {
	t.Parallel()
	m := New()
	deps := testDeps()
	if err := m.Init(context.Background(), deps); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	if m.svc.bus != deps.Bus {
		t.Fatalf("service bus was not wired from deps")
	}
	if m.svc.manager == nil {
		t.Fatalf("service manager must be initialized")
	}
	if m.svc.vmgr == nil {
		t.Fatalf("virtual serial manager must be initialized")
	}
	if m.svc.buffers == nil {
		t.Fatalf("buffers map must be initialized")
	}
}

func TestNewServiceNilBusIsUsable(t *testing.T) {
	t.Parallel()
	svc := NewService(nil)
	if svc.bus == nil {
		t.Fatalf("nil bus must be replaced with an event bus")
	}
	got, err := svc.Ping(context.Background(), "hi")
	if err != nil {
		t.Fatalf("Ping failed: %v", err)
	}
	if got != "pong:hi" {
		t.Fatalf("Ping = %q, want pong:hi", got)
	}
}

func TestServiceInitDoesNotReplaceExistingBus(t *testing.T) {
	t.Parallel()
	first := eventbus.New()
	second := eventbus.New()
	svc := NewService(first)
	svc.init(second)
	if svc.bus != first {
		t.Fatalf("service bus was replaced by a later init")
	}
}

func TestServiceInitSubscribesExistingBus(t *testing.T) {
	t.Parallel()
	first := eventbus.New()
	second := eventbus.New()
	svc := &Service{bus: first}
	svc.init(second)

	svc.mu.Lock()
	svc.buffers["port-1"] = buffer.NewRing(1024)
	svc.mu.Unlock()
	first.Publish("serial:data", manager.DataEvent{
		PortID: "port-1",
		Data:   []byte("from existing bus"),
	})

	snap, err := svc.QueryPage("port-1", 0, 1024)
	if err != nil {
		t.Fatalf("QueryPage failed: %v", err)
	}
	if string(snap.Data) != "from existing bus" {
		t.Fatalf("buffer data = %q, want existing bus event", snap.Data)
	}
}

func TestPingEmptyReturnsInvalid(t *testing.T) {
	t.Parallel()
	svc := &Service{}
	_, err := svc.Ping(context.Background(), "")
	if err == nil {
		t.Fatalf("Ping(\"\") must error")
	}
	if mterrors.AsCode(err) != mterrors.CodeInvalid {
		t.Fatalf("Ping(\"\") code = %q, want invalid", mterrors.AsCode(err))
	}
}

func TestPingReturnsPong(t *testing.T) {
	t.Parallel()
	svc := &Service{}
	got, err := svc.Ping(context.Background(), "hi")
	if err != nil {
		t.Fatalf("Ping failed: %v", err)
	}
	if got != "pong:hi" {
		t.Fatalf("Ping = %q, want pong:hi", got)
	}
}

func TestLifecycleDoesNotPanicBeforeInit(t *testing.T) {
	t.Parallel()
	m := New()
	// Stop/Dispose must be safe even if Init was never called (rollback path).
	if err := m.Stop(context.Background()); err != nil {
		t.Fatalf("Stop before Init must not error: %v", err)
	}
	m.Dispose()
	m.Dispose() // idempotent
}

func TestStartStopAfterInit(t *testing.T) {
	t.Parallel()
	m := New()
	deps := testDeps()
	if err := m.Init(context.Background(), deps); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if err := m.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := m.Stop(context.Background()); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	m.Dispose()
}

func TestModuleStopCleansSerialResources(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping socat integration test in short mode")
	}

	m := New()
	if err := m.Init(context.Background(), testDeps()); err != nil {
		t.Fatalf("Init: %v", err)
	}

	pair, err := port.StartVirtualPair(context.Background())
	if err != nil {
		t.Skipf("socat not available: %v", err)
	}
	defer pair.Stop()

	handle, err := m.svc.OpenPort(context.Background(), manager.OpenRequest{
		Config: port.SerialConfig{PortName: pair.Port1, BaudRate: 115200},
	})
	if err != nil {
		t.Fatalf("OpenPort: %v", err)
	}
	m.svc.mu.Lock()
	m.svc.buffers[handle.ID] = buffer.NewRing(1024)
	m.svc.mu.Unlock()

	if _, err := m.svc.CreateVirtualPort(context.Background(), "shutdown-vport", "shutdownVPort"); err != nil {
		t.Fatalf("CreateVirtualPort: %v", err)
	}

	if len(m.svc.ListPorts()) == 0 {
		t.Fatalf("expected an open serial handle before Stop")
	}
	if len(m.svc.ListVirtualPorts()) == 0 {
		t.Fatalf("expected a virtual port before Stop")
	}

	if err := m.Stop(context.Background()); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	if got := len(m.svc.ListPorts()); got != 0 {
		t.Fatalf("open handles after Stop = %d, want 0", got)
	}
	if got := len(m.svc.ListVirtualPorts()); got != 0 {
		t.Fatalf("virtual ports after Stop = %d, want 0", got)
	}
	m.svc.mu.RLock()
	bufferCount := len(m.svc.buffers)
	m.svc.mu.RUnlock()
	if bufferCount != 0 {
		t.Fatalf("buffers after Stop = %d, want 0", bufferCount)
	}

	m.Dispose()
}
