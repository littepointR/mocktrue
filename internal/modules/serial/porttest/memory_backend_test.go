package porttest

import (
	"io"
	"testing"
	"time"
)

func TestMemoryBackendPairRoundTrip(t *testing.T) {
	backend := NewMemoryBackend()
	backend.AddPair("COM_A", "COM_B")

	a, err := backend.Open(DefaultSerialConfig("COM_A"))
	if err != nil {
		t.Fatalf("open COM_A: %v", err)
	}
	defer func() { _ = a.Close() }()

	b, err := backend.Open(DefaultSerialConfig("COM_B"))
	if err != nil {
		t.Fatalf("open COM_B: %v", err)
	}
	defer func() { _ = b.Close() }()
	if err := b.SetReadTimeout(time.Second); err != nil {
		t.Fatalf("set read timeout: %v", err)
	}

	if _, err := a.Write([]byte("ping")); err != nil {
		t.Fatalf("write: %v", err)
	}

	buf := make([]byte, 4)
	if _, err := io.ReadFull(b, buf); err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(buf) != "ping" {
		t.Fatalf("got %q, want ping", string(buf))
	}
}

func TestMemoryBackendEnumerate(t *testing.T) {
	backend := NewMemoryBackend()
	backend.AddPair("A", "B")

	ports, err := backend.Enumerate(t.Context())
	if err != nil {
		t.Fatalf("enumerate: %v", err)
	}
	if len(ports) != 2 {
		t.Fatalf("len(ports) = %d, want 2", len(ports))
	}
	if ports[0].Name != "A" || ports[1].Name != "B" {
		t.Fatalf("ports = %#v, want A/B sorted by name", ports)
	}
}
