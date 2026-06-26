//go:build windows && integration

package integration

import (
	"io"
	"os"
	"testing"
	"time"

	"github.com/littepointR/portweave/internal/modules/serial/port"
)

func TestWindowsVirtualCOMPairRoundTrip(t *testing.T) {
	portA := os.Getenv("PORTWEAVE_TEST_COM_A")
	portB := os.Getenv("PORTWEAVE_TEST_COM_B")
	if portA == "" || portB == "" {
		t.Skip("set PORTWEAVE_TEST_COM_A and PORTWEAVE_TEST_COM_B to a linked com0com pair")
	}

	backend := port.RealBackend{}
	options := port.SerialConfig{
		BaudRate: 115200,
		DataBits: 8,
		StopBits: "1",
		Parity:   "none",
		FlowMode: "none",
	}

	options.PortName = portA
	a, err := backend.Open(options)
	if err != nil {
		t.Fatalf("open %s: %v", portA, err)
	}
	defer func() { _ = a.Close() }()

	options.PortName = portB
	b, err := backend.Open(options)
	if err != nil {
		t.Fatalf("open %s: %v", portB, err)
	}
	defer func() { _ = b.Close() }()
	if err := b.SetReadTimeout(3 * time.Second); err != nil {
		t.Fatalf("set %s read timeout: %v", portB, err)
	}

	payload := []byte("portweave-windows-com")
	if _, err := a.Write(payload); err != nil {
		t.Fatalf("write %s: %v", portA, err)
	}

	buf := make([]byte, len(payload))
	if _, err := io.ReadFull(b, buf); err != nil {
		t.Fatalf("read %s: %v", portB, err)
	}
	if string(buf) != string(payload) {
		t.Fatalf("got %q, want %q", string(buf), string(payload))
	}
}
