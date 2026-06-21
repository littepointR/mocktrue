package manager

import (
	"github.com/littepointR/mocktrue/internal/modules/serial/port"
)

// OpenRequest bundles the parameters for opening a serial port.
// Immutable once created.
type OpenRequest struct {
	Config port.SerialConfig
}

// DataEvent is emitted on the event bus when data is received from a port.
// Immutable once created.
type DataEvent struct {
	PortID string
	Data   []byte // received bytes (copy, safe to hold)
}

// HandleStatus is a snapshot of a port handle's current state.
// Immutable once created.
type HandleStatus struct {
	ID       string
	Config   port.SerialConfig
	IsOpen   bool
	RxBytes  int64
	TxBytes  int64
}
