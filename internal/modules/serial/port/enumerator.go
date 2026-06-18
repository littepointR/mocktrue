package port

import (
	"context"

	"go.bug.st/serial/enumerator"
)

// Enumerate discovers all available serial ports on the system using the
// go.bug.st/serial enumerator. It returns a snapshot (immutable slice) of
// PortInfo. An empty list is not an error (e.g. on CI with no hardware).
//
// On macOS the enumerator requires CGO (IOKit); on Linux/Windows it is
// pure Go.
func Enumerate(ctx context.Context) ([]PortInfo, error) {
	detailed, err := enumerator.GetDetailedPortsList()
	if err != nil {
		return nil, err
	}
	ports := make([]PortInfo, 0, len(detailed))
	for _, d := range detailed {
		ports = append(ports, PortInfo{
			Name:         d.Name,
			VID:          d.VID,
			PID:          d.PID,
			SerialNumber: d.SerialNumber,
			IsUSB:        d.IsUSB,
			FriendlyName: d.Name,
		})
	}
	return ports, nil
}
