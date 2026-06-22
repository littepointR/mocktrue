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

func TestMemoryBackendRejectsDuplicateOpenUntilClose(t *testing.T) {
	backend := NewMemoryBackend()
	backend.AddPair("A", "B")

	first, err := backend.Open(DefaultSerialConfig("A"))
	if err != nil {
		t.Fatalf("first open: %v", err)
	}
	if _, err := backend.Open(DefaultSerialConfig("A")); err == nil {
		t.Fatal("duplicate open must fail while endpoint is active")
	}
	if err := first.Close(); err != nil {
		t.Fatalf("close first: %v", err)
	}
	second, err := backend.Open(DefaultSerialConfig("A"))
	if err != nil {
		t.Fatalf("reopen after close: %v", err)
	}
	defer func() { _ = second.Close() }()
}

func TestMemoryBackendReopenPairAfterClose(t *testing.T) {
	backend := NewMemoryBackend()
	backend.AddPair("A", "B")

	a, err := backend.Open(DefaultSerialConfig("A"))
	if err != nil {
		t.Fatalf("open A: %v", err)
	}
	b, err := backend.Open(DefaultSerialConfig("B"))
	if err != nil {
		t.Fatalf("open B: %v", err)
	}
	if err := a.Close(); err != nil {
		t.Fatalf("close A: %v", err)
	}
	if err := b.Close(); err != nil {
		t.Fatalf("close B: %v", err)
	}

	a, err = backend.Open(DefaultSerialConfig("A"))
	if err != nil {
		t.Fatalf("reopen A: %v", err)
	}
	defer func() { _ = a.Close() }()
	b, err = backend.Open(DefaultSerialConfig("B"))
	if err != nil {
		t.Fatalf("reopen B: %v", err)
	}
	defer func() { _ = b.Close() }()
	if err := b.SetReadTimeout(time.Second); err != nil {
		t.Fatalf("set read timeout: %v", err)
	}

	if _, err := a.Write([]byte("again")); err != nil {
		t.Fatalf("write after reopen: %v", err)
	}
	buf := make([]byte, 5)
	if _, err := io.ReadFull(b, buf); err != nil {
		t.Fatalf("read after reopen: %v", err)
	}
	if string(buf) != "again" {
		t.Fatalf("got %q, want again", string(buf))
	}
}
