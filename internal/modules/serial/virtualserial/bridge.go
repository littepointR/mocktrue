package virtualserial

import (
	"sync"

	"github.com/littepointR/portweave/internal/core/errors"
	"go.bug.st/serial"
)

// Bridge connects two serial ports bidirectionally.
type Bridge struct {
	ID       string
	Port1    string
	Port2    string
	BaudRate int

	endpoint1   string
	endpoint2   string
	port1Handle serial.Port
	port2Handle serial.Port
	stopChan    chan struct{}
	wg          sync.WaitGroup
	mu          sync.Mutex
	stopped     bool
}

// NewBridge creates a new serial bridge.
func NewBridge(id, port1, port2 string, baudRate int) *Bridge {
	return NewBridgeWithEndpoints(id, port1, port2, port1, port2, baudRate)
}

// NewBridgeWithEndpoints creates a bridge whose public port names may differ
// from the device endpoints it opens.
func NewBridgeWithEndpoints(id, port1, port2, endpoint1, endpoint2 string, baudRate int) *Bridge {
	return &Bridge{
		ID:        id,
		Port1:     port1,
		Port2:     port2,
		BaudRate:  baudRate,
		endpoint1: endpoint1,
		endpoint2: endpoint2,
		stopChan:  make(chan struct{}),
	}
}

// Start opens both ports and starts bidirectional forwarding.
func (b *Bridge) Start() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	mode := &serial.Mode{
		BaudRate: b.BaudRate,
		DataBits: 8,
		Parity:   serial.NoParity,
		StopBits: serial.OneStopBit,
	}

	port1, err := serial.Open(b.endpoint1, mode)
	if err != nil {
		return errors.Wrap(errors.CodeIO, "open port1", err)
	}
	b.port1Handle = port1

	port2, err := serial.Open(b.endpoint2, mode)
	if err != nil {
		_ = port1.Close()
		return errors.Wrap(errors.CodeIO, "open port2", err)
	}
	b.port2Handle = port2

	b.wg.Add(2)
	go b.forward(port1, port2)
	go b.forward(port2, port1)

	return nil
}

// forward reads from src and writes to dst until stop is requested or an
// error occurs (e.g. port closed).
func (b *Bridge) forward(src, dst serial.Port) {
	defer b.wg.Done()

	buf := make([]byte, 4096)
	for {
		// Read returns an error when the port is closed (during Stop).
		n, err := src.Read(buf)
		if err != nil {
			return
		}
		if n > 0 {
			// Check stop signal between reads to drain buffer
			select {
			case <-b.stopChan:
				return
			default:
			}
			if _, err := dst.Write(buf[:n]); err != nil {
				return
			}
		}
	}
}

// Stop closes both ports (which causes Read to return an error and goroutines
// to exit) and waits for cleanup. Idempotent.
func (b *Bridge) Stop() {
	b.mu.Lock()
	if b.stopped {
		b.mu.Unlock()
		return
	}
	b.stopped = true
	close(b.stopChan)

	// Close ports first to unblock Read() in forward goroutines
	port1 := b.port1Handle
	port2 := b.port2Handle
	b.port1Handle = nil
	b.port2Handle = nil
	b.mu.Unlock()

	if port1 != nil {
		_ = port1.Close()
	}
	if port2 != nil {
		_ = port2.Close()
	}

	// Now wait for goroutines to exit (they will after Read returns error)
	b.wg.Wait()
}
