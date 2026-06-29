//go:build !linux && !darwin && !freebsd && !netbsd && !openbsd && !windows

package virtualserial

// DefaultBackend returns the platform virtual serial backend.
func DefaultBackend() Backend {
	return newUnsupportedBackend(
		"unsupported",
		"virtual serial ports are not supported on this platform",
		"no backend is implemented for this operating system",
	)
}
