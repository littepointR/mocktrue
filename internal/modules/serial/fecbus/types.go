package fecbus

import (
	"time"

	"github.com/littepointR/portweave/internal/modules/serial/port"
)

const (
	SessionStatusOpen    = "open"
	SessionStatusClosed  = "closed"
	SessionStatusError   = "error"
	SessionStatusRunning = "running"
)

// SessionRole fixes whether a FECbus session acts as a controller or device.
type SessionRole string

const (
	SessionRoleMaster SessionRole = "master"
	SessionRoleSlave  SessionRole = "slave"
)

// FrameType is the FECbus FT field.
type FrameType byte

const (
	FrameTypeRequest FrameType = 0x00
	FrameTypeAnswer  FrameType = 0x01
)

// FunctionCode is the first byte of the FECbus message data.
type FunctionCode byte

const (
	FunctionSyncHeartbeat        FunctionCode = 0x00
	FunctionReset                FunctionCode = 0x01
	FunctionSilence              FunctionCode = 0x02
	FunctionSelfTest             FunctionCode = 0x03
	FunctionBroadcastClock       FunctionCode = 0x04
	FunctionNotifyUrgentEvent    FunctionCode = 0x05
	FunctionNotifyGeneralEvent   FunctionCode = 0x06
	FunctionNotifyDebugEvent     FunctionCode = 0x07
	FunctionStatusAnswer         FunctionCode = 0x0f
	FunctionDeviceUrgentEvent    FunctionCode = 0x11
	FunctionDeviceGeneralEvent   FunctionCode = 0x12
	FunctionDeviceDebugEvent     FunctionCode = 0x13
	FunctionHeartbeatPollInfo    FunctionCode = 0x14
	FunctionPollConnectionStatus FunctionCode = 0x21
	FunctionQueryDeviceStatus    FunctionCode = 0x22
	FunctionQueryConfig          FunctionCode = 0x23
	FunctionQueryIdentifier      FunctionCode = 0x24
	FunctionQueryParameter       FunctionCode = 0x25
	FunctionQueryComment         FunctionCode = 0x26
	FunctionQueryProgramming     FunctionCode = 0x27
	FunctionQueryRegistration    FunctionCode = 0x28
	FunctionQueryCurrentEvent    FunctionCode = 0x29
	FunctionQueryHistoryEvent    FunctionCode = 0x2a
	FunctionStopEventQuery       FunctionCode = 0x2b
	FunctionQueryProtocolVersion FunctionCode = 0x2c
	FunctionQueryDeviceList      FunctionCode = 0x2d
)

// StatusCode is the second byte of a status-answer frame.
type StatusCode byte

const (
	StatusGroupEnd            StatusCode = 0x00
	StatusCRCError            StatusCode = 0x01
	StatusInvalidService      StatusCode = 0x02
	StatusUnitFault           StatusCode = 0x03
	StatusBusy                StatusCode = 0x04
	StatusUnrecognizedCommand StatusCode = 0x05
	StatusAddressNotFound     StatusCode = 0x06
	StatusParameterError      StatusCode = 0x07
	StatusProcessing          StatusCode = 0x08
	StatusEventEnd            StatusCode = 0x09
	StatusReceivedOK          StatusCode = 0x0a
)

// FrameBoundary is the fixed serial start/end byte.
const FrameBoundary byte = 0x7e

// Frame is one decoded FECbus serial frame.
type Frame struct {
	Type          FrameType
	TargetAddress byte
	Priority      byte
	SourceAddress byte
	MessageNumber byte
	GroupNumber   byte
	Data          []byte
	Raw           []byte
	CRCOK         bool
	Timestamp     time.Time
}

// Function returns the first data byte if present.
func (f Frame) Function() FunctionCode {
	if len(f.Data) == 0 {
		return 0
	}
	return FunctionCode(f.Data[0])
}

// FunctionInfo describes one function code from GB 4717-2024 Appendix C table C.2.
type FunctionInfo struct {
	Code        FunctionCode
	Hex         string
	Name        string
	Description string
	Direction   string
	Answer      bool
	Custom      bool
	Reserved    bool
}

// FieldDefinition describes one structured data field after the function code.
type FieldDefinition struct {
	Key     string
	Label   string
	Offset  int
	Length  int
	Type    string
	Endian  string
	Enum    map[byte]string
	Enum16  map[uint16]string
	Meaning string
}

// CustomDataFieldDefinition is a user-defined data segment field.
type CustomDataFieldDefinition struct {
	Key     string
	Label   string
	Offset  int
	Length  int
	Type    string
	Endian  string
	Enum    map[byte]string
	Meaning string
}

// CustomFunctionDefinition customizes Appendix C function names and payload fields.
type CustomFunctionDefinition struct {
	Code        FunctionCode
	Name        string
	Description string
	Direction   string
	Answer      bool
	Fields      []CustomDataFieldDefinition
}

// FrameSegment annotates one raw byte range in an encoded serial frame.
type FrameSegment struct {
	Key       string
	Label     string
	Start     int
	End       int
	Hex       string
	Value     any
	ValueText string
	Meaning   string
}

// FieldAnnotation annotates one application data field.
type FieldAnnotation struct {
	Key       string
	Label     string
	Start     int
	End       int
	Hex       string
	Value     any
	ValueText string
	Meaning   string
}

// AnnotatedFrame is a frontend-safe semantic view of one captured FECbus frame.
type AnnotatedFrame struct {
	Segments        []FrameSegment
	DataFields      []FieldAnnotation
	Function        FunctionInfo
	GroupKey        string
	GroupColorIndex int
	Summary         string
	Warnings        []string
}

// AnnotationContext carries multi-frame package context for continuation frames.
type AnnotationContext struct {
	Function FunctionCode
}

// OpenSessionRequest opens a dedicated FECbus serial session.
type OpenSessionRequest struct {
	ID        string
	Name      string
	Role      SessionRole
	Config    port.SerialConfig
	Endpoint  string
	TimeoutMs int
	Retries   int
}

// SessionInfo is a frontend-safe snapshot of a FECbus session.
type SessionInfo struct {
	ID            string
	Name          string
	Role          SessionRole
	Config        port.SerialConfig
	Status        string
	RxBytes       int64
	TxBytes       int64
	SlaveRunning  bool
	SourceAddress byte
	TargetAddress byte
	SlaveUnits    []SlaveUnitInfo
	StartedAt     time.Time
	StoppedAt     time.Time
	LastError     string
}

// SendRequest describes one FECbus master/device transmission.
type SendRequest struct {
	SessionID     string
	FrameType     FrameType
	TargetAddress byte
	Priority      byte
	SourceAddress byte
	MessageNumber byte
	GroupNumber   byte
	Function      FunctionCode
	DataHex       string
	Data          []byte
	ExpectAnswer  bool
	TimeoutMs     int
	Retries       int
}

// Transaction captures one FECbus request/answer exchange.
type Transaction struct {
	ID               string
	SessionID        string
	StartedAt        time.Time
	CompletedAt      time.Time
	Request          Frame
	Response         *Frame
	RequestFrameHex  string
	ResponseFrameHex string
	BytesWritten     int
	Error            string
}

// SlaveUnitState configures one simulated FECbus slave address.
type SlaveUnitState struct {
	Address                 byte
	DefaultStatus           StatusCode
	AutoStatusAnswer        bool
	AcceptBroadcast         bool
	AnswerPayloadByFunction map[FunctionCode][]byte
}

// SlaveUnitInfo is a frontend-safe summary for one simulated slave address.
type SlaveUnitInfo struct {
	Address          byte
	DefaultStatus    StatusCode
	AutoStatusAnswer bool
	AcceptBroadcast  bool
}

// SlaveState configures automatic status responses for device simulation.
type SlaveState struct {
	Address                 byte
	DefaultStatus           StatusCode
	AutoStatusAnswer        bool
	AcceptBroadcast         bool
	AnswerPayloadByFunction map[FunctionCode][]byte
}

// StartSlaveRequest starts FECbus slave simulation for one session.
type StartSlaveRequest struct {
	SessionID string
	State     SlaveState
	Units     []SlaveUnitState
}

// FrameRecord stores one captured FECbus TX/RX frame.
type FrameRecord struct {
	Seq       int64
	SessionID string
	Direction string
	Frame     Frame
	Hex       string
	Error     string
	Timestamp time.Time
	Annotated AnnotatedFrame
}

// QueryRequest filters frame history.
type QueryRequest struct {
	SessionID string
	Offset    int64
	Limit     int
	Direction string
	Search    string
	Custom    []CustomFunctionDefinition
}

// FramePage is a page of captured FECbus frames.
type FramePage struct {
	Frames []FrameRecord
	Offset int64
	Limit  int
	Total  int64
	EOF    bool
}
