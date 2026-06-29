package virtualserial

import (
	"context"
	"sync"
)

// VirtualPair holds the two ends of an OS-backed virtual serial pair.
type VirtualPair struct {
	ID          string
	Port1       string
	Port2       string
	port2Hidden bool
	handle      pairHandle
	mu          sync.Mutex
	stopped     bool
}

// NewVirtualPair creates a default virtual serial pair. Caller must call Stop()
// when done.
func NewVirtualPair(ctx context.Context) (*VirtualPair, error) {
	return DefaultBackend().CreatePair(ctx, "default", "ttyV0", "ttyV1")
}

// Stop releases the OS resources backing the pair.
func (vp *VirtualPair) Stop() error {
	vp.mu.Lock()
	defer vp.mu.Unlock()
	if vp.stopped {
		return nil
	}
	if vp.handle != nil {
		if err := vp.handle.Stop(); err != nil {
			return err
		}
	}
	vp.stopped = true
	return nil
}

// PortNames returns the two port paths.
func (vp *VirtualPair) PortNames() (string, string) {
	return vp.Port1, vp.Port2
}

// IsUserFacingPort reports whether this pair represents a single public
// virtual port backed by a hidden peer.
func (vp *VirtualPair) IsUserFacingPort() bool {
	return vp.port2Hidden
}
