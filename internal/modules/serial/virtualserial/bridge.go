package virtualserial

import (
	"sync"

	"github.com/suyue/mocktrue/internal/core/errors"
	"go.bug.st/serial"
)

// Bridge connects two serial ports bidirectionally.
type Bridge struct {
	ID       string
	Port1    string
	Port2    string
	BaudRate int

	port1Handle serial.Port
	port2Handle serial.Port
	stopChan    chan struct{}
	wg          sync.WaitGroup
	mu          sync.Mutex
}

// NewBridge creates a new serial bridge.
func NewBridge(id, port1, port2 string, baudRate int) *Bridge {
	return &Bridge{
		ID:       id,
		Port1:    port1,
		Port2:    port2,
		BaudRate: baudRate,
		stopChan: make(chan struct{}),
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

	// Open port1
	port1, err := serial.Open(b.Port1, mode)
	if err != nil {
		return errors.Wrap(errors.CodeIO, "open port1", err)
	}
	b.port1Handle = port1

	// Open port2
	port2, err := serial.Open(b.Port2, mode)
	if err != nil {
		port1.Close()
		return errors.Wrap(errors.CodeIO, "open port2", err)
	}
	b.port2Handle = port2

	// Start forwarding goroutines
	b.wg.Add(2)
	go b.forward(port1, port2, "1->2")
	go b.forward(port2, port1, "2->1")

	return nil
}

// forward reads from src and writes to dst.
func (b *Bridge) forward(src, dst serial.Port, label string) {
	defer b.wg.Done()

	buf := make([]byte, 4096)
	for {
		select {
		case <-b.stopChan:
			return
		default:
		}

		n, err := src.Read(buf)
		if err != nil {
			return
		}
		if n > 0 {
			if _, err := dst.Write(buf[:n]); err != nil {
				return
			}
		}
	}
}

// Stop closes both ports and stops forwarding.
func (b *Bridge) Stop() {
	b.mu.Lock()
	defer b.mu.Unlock()

	close(b.stopChan)
	b.wg.Wait()

	if b.port1Handle != nil {
		b.port1Handle.Close()
		b.port1Handle = nil
	}
	if b.port2Handle != nil {
		b.port2Handle.Close()
		b.port2Handle = nil
	}
}
