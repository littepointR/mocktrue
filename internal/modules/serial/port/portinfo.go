package port

// PortInfo describes a discovered serial port with its hardware metadata.
// Immutable once created.
type PortInfo struct {
	Name         string // e.g. "/dev/tty.usbmodem...", "COM3"
	Vendor       string // USB vendor name
	Product      string // USB product name
	VID          string // USB Vendor ID (hex, e.g. "2341")
	PID          string // USB Product ID (hex, e.g. "0043")
	SerialNumber string // USB serial number
	IsUSB        bool   // true if port is USB-connected
	FriendlyName string // human-readable name (may equal Name)
}
