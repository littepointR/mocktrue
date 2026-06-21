package monitor

import (
	"time"

	"github.com/littepointR/mocktrue/internal/modules/serial/port"
)

const (
	ProviderBridge        = "bridge"
	ProviderWindowsDriver = "windows-driver"

	StatusStopped = "stopped"
	StatusRunning = "running"
	StatusError   = "error"

	DirectionAToB   = "a_to_b"
	DirectionBToA   = "b_to_a"
	DirectionStatus = "status"
)

// StartRequest starts a serial monitor session.
type StartRequest struct {
	ID                string
	Name              string
	Provider          string
	PortA             string
	PortB             string
	EndpointA         string
	EndpointB         string
	ExternalPort      string
	AutoVirtualPortID string
	Config            port.SerialConfig
	Encoding          string
}

// SessionInfo is a frontend-safe snapshot of a monitor session.
type SessionInfo struct {
	ID                string
	Name              string
	Provider          string
	PortA             string
	PortB             string
	ExternalPort      string
	AutoVirtualPortID string
	Config            port.SerialConfig
	Encoding          string
	Status            string
	RxBytes           int64
	TxBytes           int64
	FrameCount        int64
	StartedAt         time.Time
	StoppedAt         time.Time
	Error             string
}

// Frame is one captured data or status record.
type Frame struct {
	Seq         int64
	Timestamp   time.Time
	Direction   string
	Port        string
	Length      int
	Data        []byte
	DisplayText string
	DisplayHex  string
	DisplayDec  string
	DisplayOct  string
	DisplayBin  string
	Encoding    string
	Error       string
}

// QueryRequest asks for a filtered page of frames.
type QueryRequest struct {
	MonitorID string
	Offset    int64
	Limit     int
	Direction string
	Search    string
}

// FramePage is a filtered frame page.
type FramePage struct {
	Frames     []Frame
	Total      int64
	NextOffset int64
}
