package virtualserial

import (
	"context"
	"testing"
	"time"

	goserial "go.bug.st/serial"
)

func TestVirtualPairCreation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping socat test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pair, err := NewVirtualPair(ctx)
	if err != nil {
		t.Skipf("socat not available: %v", err)
	}
	defer pair.Stop()

	if pair.Port1 == "" || pair.Port2 == "" {
		t.Fatalf("port names should not be empty")
	}

	p1, p2 := pair.PortNames()
	if p1 == "" || p2 == "" {
		t.Fatalf("PortNames should return non-empty values")
	}
}

func TestVirtualPairStopIdempotent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping socat test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pair, err := NewVirtualPair(ctx)
	if err != nil {
		t.Skipf("socat not available: %v", err)
	}

	pair.Stop()
	pair.Stop() // should not panic
}

func TestManagerPairSurvivesRequestContextCancellation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping socat test in short mode")
	}

	mgr := NewManager()
	ctx, cancel := context.WithCancel(context.Background())
	pair, err := mgr.CreatePair(ctx, "ctx-cancel", "ctxCancelA", "ctxCancelB")
	if err != nil {
		t.Skipf("socat not available: %v", err)
	}
	defer mgr.Cleanup()

	cancel()
	time.Sleep(100 * time.Millisecond)

	port, err := goserial.Open(pair.Port1, &goserial.Mode{BaudRate: 115200})
	if err != nil {
		t.Fatalf("created port should remain openable after request context cancellation: %v", err)
	}
	_ = port.Close()
}
