package port

import "context"

// Backend abstracts serial port enumeration and opening so production code can
// use the OS serial implementation while tests can inject deterministic ports.
type Backend interface {
	Enumerate(ctx context.Context) ([]PortInfo, error)
	Open(cfg SerialConfig) (Port, error)
}

// RealBackend delegates to the platform serial implementation in this package.
type RealBackend struct{}

// Enumerate discovers serial ports using the real OS enumerator.
func (RealBackend) Enumerate(ctx context.Context) ([]PortInfo, error) {
	return Enumerate(ctx)
}

// Open opens a real OS serial port using the existing SerialConfig mapping.
func (RealBackend) Open(cfg SerialConfig) (Port, error) {
	return Open(cfg)
}
