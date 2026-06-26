//go:build (darwin || linux) && integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/littepointR/portweave/internal/core/eventbus"
	"github.com/littepointR/portweave/internal/modules/serial/buffer"
	"github.com/littepointR/portweave/internal/modules/serial/manager"
	"github.com/littepointR/portweave/internal/modules/serial/port"
)

func TestSerialEndToEnd(t *testing.T) {
	// Start socat virtual pair
	pair, err := port.StartVirtualPair(context.Background())
	if err != nil {
		t.Skipf("socat not available: %v", err)
	}
	defer pair.Stop()

	bus := eventbus.New()
	mgr := manager.NewManager(bus)

	// Collect DataEvents
	received := make(chan manager.DataEvent, 64)
	bus.Subscribe("serial:data", func(data any) {
		if evt, ok := data.(manager.DataEvent); ok {
			received <- evt
		}
	})

	// Open one end of the pair
	handle, err := mgr.Open(context.Background(), manager.OpenRequest{
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

	testData := []byte("hello world")
	_, err = conn.Write(testData)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Wait for DataEvent
	select {
	case evt := <-received:
		if string(evt.Data) != "hello world" {
			t.Fatalf("DataEvent.Data = %q, want %q", evt.Data, "hello world")
		}
		t.Logf("Received %d bytes: %q", len(evt.Data), evt.Data)
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for DataEvent")
	}

	// Wait a bit for rxBytes to be updated by readLoop
	time.Sleep(100 * time.Millisecond)

	// Verify handle status
	status := mgr.List()
	if len(status) == 0 {
		t.Fatalf("no handles in list")
	}
	if status[0].RxBytes < int64(len(testData)) {
		t.Fatalf("RxBytes = %d, want >= %d", status[0].RxBytes, len(testData))
	}

	// Test buffer query
	buf := buffer.NewRing(1024 * 1024)
	buf.Append(buffer.Chunk{Seq: 0, BaseOffset: 0, Data: []byte("test data")})
	snap, err := buf.Query(0, 100)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if string(snap.Data) != "test data" {
		t.Fatalf("Query result = %q, want %q", snap.Data, "test data")
	}

	// Close
	if err := mgr.Close(handle.ID); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	t.Log("End-to-end serial test passed")
}
