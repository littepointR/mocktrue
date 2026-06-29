package virtualserial

import (
	"context"

	"github.com/littepointR/portweave/internal/core/errors"
)

// Backend creates and removes OS-backed virtual serial pairs.
type Backend interface {
	Name() string
	Status(ctx context.Context) BackendStatus
	CreatePair(ctx context.Context, pairID, port1Name, port2Name string) (*VirtualPair, error)
	CreatePort(ctx context.Context, portID, publicName string) (*VirtualPair, error)
}

// BackendStatus is a frontend-safe snapshot of the active virtual serial
// backend.
type BackendStatus struct {
	Name          string
	Available     bool
	Message       string
	Reason        string
	RequiresAdmin bool
}

type unsupportedBackend struct {
	name    string
	message string
	reason  string
}

func newUnsupportedBackend(name, message, reason string) Backend {
	if name == "" {
		name = "unsupported"
	}
	if message == "" {
		message = "virtual serial ports are not available on this platform"
	}
	return unsupportedBackend{name: name, message: message, reason: reason}
}

func (b unsupportedBackend) Name() string { return b.name }

func (b unsupportedBackend) Status(context.Context) BackendStatus {
	return BackendStatus{
		Name:      b.name,
		Available: false,
		Message:   b.message,
		Reason:    b.reason,
	}
}

func (b unsupportedBackend) CreatePair(context.Context, string, string, string) (*VirtualPair, error) {
	return nil, errors.New(errors.CodePlatform, b.message)
}

func (b unsupportedBackend) CreatePort(context.Context, string, string) (*VirtualPair, error) {
	return nil, errors.New(errors.CodePlatform, b.message)
}

type pairHandle interface {
	Stop() error
}

type noopPairHandle struct{}

func (noopPairHandle) Stop() error { return nil }

func newVirtualPair(id, port1, port2 string, handle pairHandle) *VirtualPair {
	if handle == nil {
		handle = noopPairHandle{}
	}
	return &VirtualPair{
		ID:     id,
		Port1:  port1,
		Port2:  port2,
		handle: handle,
	}
}
