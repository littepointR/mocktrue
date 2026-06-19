package virtualserial

import (
	"context"
	"os/exec"
	"sync"
	"time"

	"github.com/suyue/mocktrue/internal/core/errors"
)

// VirtualPair holds the two ends of a socat-created virtual serial pair.
type VirtualPair struct {
	ID          string
	Port1       string
	Port2       string
	port2Hidden bool
	cmd         *exec.Cmd
	mu          sync.Mutex
}

// NewVirtualPair creates a socat virtual serial pair. Returns the two port
// paths. Caller must call Stop() when done.
func NewVirtualPair(ctx context.Context) (*VirtualPair, error) {
	// Remove old symlinks if they exist
	exec.Command("rm", "-f", "/tmp/ttyV0", "/tmp/ttyV1").Run()

	cmd := exec.CommandContext(ctx, "socat", "-d", "-d",
		"pty,raw,echo=0,link=/tmp/ttyV0",
		"pty,raw,echo=0,link=/tmp/ttyV1")
	if err := cmd.Start(); err != nil {
		return nil, errors.Wrap(errors.CodeIO, "start socat", err)
	}

	// Wait for symlinks to appear
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := exec.Command("test", "-e", "/tmp/ttyV0").CombinedOutput(); err == nil {
			if _, err := exec.Command("test", "-e", "/tmp/ttyV1").CombinedOutput(); err == nil {
				return &VirtualPair{
					ID:    "default",
					Port1: "/tmp/ttyV0",
					Port2: "/tmp/ttyV1",
					cmd:   cmd,
				}, nil
			}
		}
		time.Sleep(50 * time.Millisecond)
	}

	cmd.Process.Kill()
	return nil, errors.New(errors.CodeIO, "timeout waiting for socat symlinks to appear")
}

// Stop kills the socat process and cleans up.
func (vp *VirtualPair) Stop() {
	vp.mu.Lock()
	defer vp.mu.Unlock()
	if vp.cmd != nil && vp.cmd.Process != nil {
		vp.cmd.Process.Kill()
		vp.cmd.Wait()
	}
	_ = exec.Command("rm", "-f", vp.Port1, vp.Port2).Run()
}

// PortNames returns the two port paths.
func (vp *VirtualPair) PortNames() (string, string) {
	return vp.Port1, vp.Port2
}
