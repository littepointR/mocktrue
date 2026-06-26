package manager

import (
	"context"
	"testing"
	"time"

	"github.com/littepointR/portweave/internal/core/eventbus"
	"github.com/littepointR/portweave/internal/modules/serial/port"
	"github.com/littepointR/portweave/internal/modules/serial/porttest"
)

func TestNewManagerUsesRealBackendByDefault(t *testing.T) {
	t.Parallel()
	mgr := NewManager(eventbus.New())
	if mgr.backend == nil {
		t.Fatal("NewManager must initialize a default backend")
	}
	if _, ok := mgr.backend.(port.RealBackend); !ok {
		t.Fatalf("NewManager backend = %T, want port.RealBackend", mgr.backend)
	}
}

func TestNewManagerAcceptsInjectedBackend(t *testing.T) {
	t.Parallel()
	backend := fakeBackend{}
	mgr := NewManager(eventbus.New(), WithBackend(backend))
	if mgr.backend == nil {
		t.Fatal("NewManager must keep injected backend")
	}
	if _, ok := mgr.backend.(fakeBackend); !ok {
		t.Fatalf("NewManager backend = %T, want fakeBackend", mgr.backend)
	}
}

func TestManagerOpenRejectsEmptyPortName(t *testing.T) {
	t.Parallel()
	bus := eventbus.New()
	mgr := NewManager(bus)
	_, err := mgr.Open(context.Background(), OpenRequest{
		Config: port.SerialConfig{PortName: "", BaudRate: 115200},
	})
	if err == nil {
		t.Fatalf("Open with empty PortName must error")
	}
}

func TestManagerOpenRejectsInvalidBaud(t *testing.T) {
	t.Parallel()
	bus := eventbus.New()
	mgr := NewManager(bus)
	_, err := mgr.Open(context.Background(), OpenRequest{
		Config: port.SerialConfig{PortName: "/dev/null", BaudRate: 0},
	})
	if err == nil {
		t.Fatalf("Open with baud=0 must error")
	}
}

func TestManagerCloseNonExistentReturnsError(t *testing.T) {
	t.Parallel()
	bus := eventbus.New()
	mgr := NewManager(bus)
	err := mgr.Close("ghost")
	if err == nil {
		t.Fatalf("Close non-existent must error")
	}
}

func TestManagerListReturnsSnapshot(t *testing.T) {
	t.Parallel()
	bus := eventbus.New()
	mgr := NewManager(bus)
	list := mgr.List()
	if list == nil {
		t.Fatalf("List must not return nil")
	}
	if len(list) != 0 {
		t.Fatalf("empty manager List = %d, want 0", len(list))
	}
}

// TestReadLoopWithMemoryBackend verifies the full read loop using a deterministic
// in-memory serial pair. Real OS serial-pair checks live in opt-in integration
// tests so default tests stay cross-platform.
func TestReadLoopWithMemoryBackend(t *testing.T) {
	t.Parallel()
	backend := porttest.NewMemoryBackend()
	backend.AddPair("memA", "memB")

	bus := eventbus.New()
	mgr := NewManager(bus, WithBackend(backend))

	// Collect DataEvents
	received := make(chan DataEvent, 64)
	bus.Subscribe("serial:data", func(data any) {
		if evt, ok := data.(DataEvent); ok {
			received <- evt
		}
	})

	// Open one end of the pair
	handle, err := mgr.Open(context.Background(), OpenRequest{
		Config: porttest.DefaultSerialConfig("memA"),
	})
	if err != nil {
		t.Fatalf("Open memA failed: %v", err)
	}

	// Write to the other end
	conn, err := backend.Open(porttest.DefaultSerialConfig("memB"))
	if err != nil {
		t.Fatalf("Open memB failed: %v", err)
	}
	defer func() { _ = conn.Close() }()

	testData := []byte("hello")
	_, err = conn.Write(testData)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Wait for DataEvent
	select {
	case evt := <-received:
		if string(evt.Data) != "hello" {
			t.Fatalf("DataEvent.Data = %q, want %q", evt.Data, "hello")
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for DataEvent")
	}

	// Verify handle status (already returned by Open)
	// Wait a bit for readLoop to update rxBytes
	time.Sleep(100 * time.Millisecond)
	status := mgr.List()
	if len(status) == 0 {
		t.Fatalf("no handles in list")
	}
	if status[0].RxBytes < int64(len(testData)) {
		t.Fatalf("RxBytes = %d, want >= %d", status[0].RxBytes, len(testData))
	}

	// Close
	if err := mgr.Close(handle.ID); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
}

func TestReadLoopSurvivesRequestContextCancellation(t *testing.T) {
	t.Parallel()
	backend := porttest.NewMemoryBackend()
	backend.AddPair("memA", "memB")

	bus := eventbus.New()
	mgr := NewManager(bus, WithBackend(backend))

	received := make(chan DataEvent, 1)
	bus.Subscribe("serial:data", func(data any) {
		if evt, ok := data.(DataEvent); ok {
			received <- evt
		}
	})

	reqCtx, cancel := context.WithCancel(context.Background())
	handle, err := mgr.Open(reqCtx, OpenRequest{
		Config: porttest.DefaultSerialConfig("memA"),
	})
	if err != nil {
		t.Fatalf("Open memA failed: %v", err)
	}
	defer func() { _ = mgr.Close(handle.ID) }()

	cancel()
	time.Sleep(100 * time.Millisecond)

	conn, err := backend.Open(porttest.DefaultSerialConfig("memB"))
	if err != nil {
		t.Fatalf("Open memB failed: %v", err)
	}
	defer func() { _ = conn.Close() }()

	if _, err := conn.Write([]byte("alive")); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	select {
	case evt := <-received:
		if string(evt.Data) != "alive" {
			t.Fatalf("DataEvent.Data = %q, want %q", evt.Data, "alive")
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for DataEvent after request context cancellation")
	}
}

type fakeBackend struct{}

func (fakeBackend) Enumerate(ctx context.Context) ([]port.PortInfo, error) {
	return nil, nil
}

func (fakeBackend) Open(cfg port.SerialConfig) (port.Port, error) {
	return nil, nil
}
