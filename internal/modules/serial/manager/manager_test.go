package manager

import (
	"context"
	"testing"
	"time"

	"github.com/suyue/mocktrue/internal/core/eventbus"
	"github.com/suyue/mocktrue/internal/modules/serial/port"
)

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

// TestReadLoopWithSocat verifies the full read loop using a socat virtual
// serial pair. This is the core integration test for stage 1.
func TestReadLoopWithSocat(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping socat integration test in short mode")
	}
	// Start socat virtual pair
	pair, err := port.StartVirtualPair(context.Background())
	if err != nil {
		t.Skipf("socat not available: %v", err)
	}
	defer pair.Stop()

	bus := eventbus.New()
	mgr := NewManager(bus)

	// Collect DataEvents
	received := make(chan DataEvent, 64)
	bus.Subscribe("serial:data", func(data any) {
		if evt, ok := data.(DataEvent); ok {
			received <- evt
		}
	})

	// Open one end of the pair
	handle, err := mgr.Open(context.Background(), OpenRequest{
		Config: port.SerialConfig{PortName: pair.Port1, BaudRate: 115200},
	})
	if err != nil {
		t.Fatalf("Open %s failed: %v", pair.Port1, err)
	}

	// Write to the other end
	conn, err := port.OpenForTest(pair.Port2, 115200)
	if err != nil {
		t.Fatalf("OpenForTest %s failed: %v", pair.Port2, err)
	}
	defer conn.Close()

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
	if testing.Short() {
		t.Skip("skipping socat integration test in short mode")
	}
	pair, err := port.StartVirtualPair(context.Background())
	if err != nil {
		t.Skipf("socat not available: %v", err)
	}
	defer pair.Stop()

	bus := eventbus.New()
	mgr := NewManager(bus)

	received := make(chan DataEvent, 1)
	bus.Subscribe("serial:data", func(data any) {
		if evt, ok := data.(DataEvent); ok {
			received <- evt
		}
	})

	reqCtx, cancel := context.WithCancel(context.Background())
	handle, err := mgr.Open(reqCtx, OpenRequest{
		Config: port.SerialConfig{PortName: pair.Port1, BaudRate: 115200},
	})
	if err != nil {
		t.Fatalf("Open %s failed: %v", pair.Port1, err)
	}
	defer mgr.Close(handle.ID)

	cancel()
	time.Sleep(100 * time.Millisecond)

	conn, err := port.OpenForTest(pair.Port2, 115200)
	if err != nil {
		t.Fatalf("OpenForTest %s failed: %v", pair.Port2, err)
	}
	defer conn.Close()

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
