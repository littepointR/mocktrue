package monitor

import (
	"time"

	"github.com/suyue/mocktrue/internal/modules/serial/port"
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

	ExportCSV  = "csv"
	ExportText = "txt"
	ExportHTML = "html"
	ExportPCAP = "pcapng"
	SplitNone  = "none"
	SplitSize  = "size"
	SplitTime  = "time"
)

// StartRequest starts a serial monitor session.
type StartRequest struct {
	ID        string
	Name      string
	Provider  string
	PortA     string
	PortB     string
	EndpointA string
	EndpointB string
	Config    port.SerialConfig
	Encoding  string
	AutoSave  *AutoSaveOptions
}

// SessionInfo is a frontend-safe snapshot of a monitor session.
type SessionInfo struct {
	ID         string
	Name       string
	Provider   string
	PortA      string
	PortB      string
	Config     port.SerialConfig
	Encoding   string
	Status     string
	RxBytes    int64
	TxBytes    int64
	FrameCount int64
	StartedAt  time.Time
	StoppedAt  time.Time
	Error      string
	AutoSave   AutoSaveOptions
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
	Modbus      *ModbusFrame
}

// ModbusFrame contains a best-effort Modbus RTU/ASCII parse.
type ModbusFrame struct {
	Protocol    string
	Slave       byte
	Function    byte
	FunctionHex string
	PayloadHex  string
	CRCOK       bool
	LRCOK       bool
	Summary     string
	Error       string
}

// QueryRequest asks for a filtered page of frames.
type QueryRequest struct {
	MonitorID      string
	Offset         int64
	Limit          int
	Direction      string
	Search         string
	ModbusFunction int
}

// FramePage is a filtered frame page.
type FramePage struct {
	Frames     []Frame
	Total      int64
	NextOffset int64
}

// ExportRequest exports monitor frames.
type ExportRequest struct {
	MonitorID string
	Format    string
	Path      string
	Encoding  string
	Direction string
	Search    string
}

// AutoSaveRequest updates a session's automatic save settings.
type AutoSaveRequest struct {
	MonitorID string
	Options   AutoSaveOptions
}

// AutoSaveOptions controls automatic frame persistence.
type AutoSaveOptions struct {
	Enabled              bool
	Path                 string
	Directory            string
	BaseName             string
	Format               string
	SplitMode            string
	SplitSizeKB          int
	SplitIntervalSeconds int
	Encoding             string
}
