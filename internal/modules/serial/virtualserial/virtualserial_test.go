package virtualserial

import (
	"context"
	"testing"
	"time"
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
