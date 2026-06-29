package port

import (
	"context"
	"fmt"
	"time"

	"github.com/littepointR/portweave/internal/modules/serial/virtualserial"
	serial "go.bug.st/serial"
)

// SerialConfig bundles the parameters for opening a serial port.
// Immutable once created.
type SerialConfig struct {
	PortName  string
	BaudRate  int
	DataBits  int    // 5/6/7/8
	StopBits  string // "1"|"1.5"|"2"
	Parity    string // "none"|"even"|"odd"|"mark"|"space"
	FlowMode  string // "none"|"hw_rtscts"|"sw_xonxoff"
	ReadBufKB int    // read buffer size in KB (default 32)
}

// Port is the interface for an open serial port (thin wrapper around
// go.bug.st/serial.Port for testability).
type Port interface {
	Read(p []byte) (int, error)
	Write(p []byte) (int, error)
	SetReadTimeout(time.Duration) error
	Close() error
}

// Open opens a serial port with the given config.
func Open(cfg SerialConfig) (Port, error) {
	if cfg.BaudRate <= 0 {
		cfg.BaudRate = 115200
	}
	if cfg.DataBits == 0 {
		cfg.DataBits = 8
	}

	mode := &serial.Mode{
		BaudRate: cfg.BaudRate,
		DataBits: cfg.DataBits,
	}

	switch cfg.Parity {
	case "even":
		mode.Parity = serial.EvenParity
	case "odd":
		mode.Parity = serial.OddParity
	case "mark":
		mode.Parity = serial.MarkParity
	case "space":
		mode.Parity = serial.SpaceParity
	default:
		mode.Parity = serial.NoParity
	}

	switch cfg.StopBits {
	case "1.5":
		mode.StopBits = serial.OnePointFiveStopBits
	case "2":
		mode.StopBits = serial.TwoStopBits
	default:
		mode.StopBits = serial.OneStopBit
	}

	p, err := serial.Open(cfg.PortName, mode)
	if err != nil {
		return nil, fmt.Errorf("open serial port %s: %w", cfg.PortName, err)
	}
	return p, nil
}

// OpenForTest opens a port directly (used in tests to write to one end of a
// virtual pair).
func OpenForTest(portName string, baudRate int) (Port, error) {
	return Open(SerialConfig{PortName: portName, BaudRate: baudRate})
}

// VirtualPair holds the two ends of an OS-backed virtual serial pair.
type VirtualPair struct {
	Port1  string
	Port2  string
	handle *virtualserial.VirtualPair
}

// StartVirtualPair creates an OS-backed virtual serial pair. Returns the two
// port names. Caller must call Stop() when done.
func StartVirtualPair(ctx context.Context) (*VirtualPair, error) {
	pair, err := virtualserial.NewVirtualPair(ctx)
	if err != nil {
		return nil, err
	}
	return &VirtualPair{Port1: pair.Port1, Port2: pair.Port2, handle: pair}, nil
}

// Stop releases the virtual serial pair resources.
func (vp *VirtualPair) Stop() {
	if vp.handle != nil {
		_ = vp.handle.Stop()
	}
}

// ListPorts is a convenience wrapper that returns port names only.
func ListPorts(ctx context.Context) ([]string, error) {
	ports, err := Enumerate(ctx)
	if err != nil {
		return nil, err
	}
	names := make([]string, len(ports))
	for i, p := range ports {
		names[i] = p.Name
	}
	return names, nil
}
