package port

import (
	"context"
	"fmt"
	"os/exec"
	"time"

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

// VirtualPair holds the two ends of a socat-created virtual serial pair.
type VirtualPair struct {
	Port1 string
	Port2 string
	cmd   *exec.Cmd
}

// StartVirtualPair creates a socat virtual serial pair. Returns the two port
// paths. Caller must call Stop() when done.
func StartVirtualPair(ctx context.Context) (*VirtualPair, error) {
	// Remove old symlinks if they exist
	exec.Command("rm", "-f", "/tmp/ttyV0", "/tmp/ttyV1").Run()

	cmd := exec.CommandContext(ctx, "socat", "-d", "-d",
		"pty,raw,echo=0,link=/tmp/ttyV0",
		"pty,raw,echo=0,link=/tmp/ttyV1")
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("socat start failed: %w", err)
	}

	// Wait for symlinks to appear
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := exec.Command("test", "-e", "/tmp/ttyV0").CombinedOutput(); err == nil {
			if _, err := exec.Command("test", "-e", "/tmp/ttyV1").CombinedOutput(); err == nil {
				return &VirtualPair{
					Port1: "/tmp/ttyV0",
					Port2: "/tmp/ttyV1",
					cmd:   cmd,
				}, nil
			}
		}
		time.Sleep(50 * time.Millisecond)
	}

	cmd.Process.Kill()
	return nil, fmt.Errorf("timeout waiting for socat symlinks to appear")
}

// Stop kills the socat process and cleans up.
func (vp *VirtualPair) Stop() {
	if vp.cmd != nil && vp.cmd.Process != nil {
		vp.cmd.Process.Kill()
		vp.cmd.Wait()
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
