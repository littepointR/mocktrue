package serial

import (
	"context"
	"testing"

	"github.com/suyue/mocktrue/internal/core/eventbus"
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
