package modbus

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

// SessionRole fixes whether a Modbus session is used as a master or slave.
type SessionRole string

const (
	SessionRoleMaster SessionRole = "master"
	SessionRoleSlave  SessionRole = "slave"
)

// OpenSessionRequest opens a dedicated Modbus serial session.
type OpenSessionRequest struct {
	ID        string
	Name      string
	Mode      FrameMode
	Role      SessionRole
	Config    port.SerialConfig
	Endpoint  string
	TimeoutMs int
	Retries   int
}

// SessionInfo is a frontend-safe snapshot of a Modbus serial session.
type SessionInfo struct {
	ID           string
	Name         string
	Mode         FrameMode
	Role         SessionRole
	Config       port.SerialConfig
	Status       string
	RxBytes      int64
	TxBytes      int64
	SlaveRunning bool
	UnitID       byte
	UnitIDs      []int
	StartedAt    time.Time
	StoppedAt    time.Time
	LastError    string
}

// MasterRequest describes one Modbus master transaction.
type MasterRequest struct {
	SessionID      string
	UnitID         byte
	Function       FunctionCode
	AddressMode    AddressMode
	Address        uint16
	Quantity       uint16
	Value          uint16
	CoilValues     []bool
	RegisterValues []uint16
	TimeoutMs      int
	Retries        int
}

// Transaction captures one master request/response.
type Transaction struct {
	ID               string
	SessionID        string
	StartedAt        time.Time
	CompletedAt      time.Time
	UnitID           byte
	Mode             FrameMode
	RequestPDU       PDU
	ResponsePDU      PDU
	RequestFrameHex  string
	ResponseFrameHex string
	BytesWritten     int
	Response         ParsedResponse
	Error            string
}

// StartSlaveRequest starts slave simulation for one open session.
type StartSlaveRequest struct {
	SessionID string
	UnitID    byte
	DataModel DataModelSnapshot
	Units     []SlaveUnitSnapshot
}

// SlaveUnitSnapshot is one independent simulated Modbus slave address.
type SlaveUnitSnapshot struct {
	UnitID    byte
	DataModel DataModelSnapshot
}

// SlaveUnitInfo is a frontend-safe snapshot of a simulated unit.
type SlaveUnitInfo struct {
	UnitID    byte
	DataModel DataModelSnapshot
}

// BoolPoint is one address/value entry in a boolean Modbus table.
type BoolPoint struct {
	Address uint16
	Value   bool
}

// RegisterPoint is one address/value entry in a register Modbus table.
type RegisterPoint struct {
	Address uint16
	Value   uint16
}

// DataModelSnapshot is the serializable representation of a slave data model.
type DataModelSnapshot struct {
	Coils            []BoolPoint
	DiscreteInputs   []BoolPoint
	InputRegisters   []RegisterPoint
	HoldingRegisters []RegisterPoint
}

// DataType selects how register words are interpreted.
type DataType string

const (
	DataTypeNone     DataType = "none"
	DataTypeInt16    DataType = "int16"
	DataTypeUint16   DataType = "uint16"
	DataTypeInt32    DataType = "int32"
	DataTypeUint32   DataType = "uint32"
	DataTypeInt64    DataType = "int64"
	DataTypeUint64   DataType = "uint64"
	DataTypeFloat    DataType = "float"
	DataTypeDouble   DataType = "double"
	DataTypeUnix     DataType = "unix"
	DataTypeDateTime DataType = "datetime"
	DataTypeUTF8     DataType = "utf8"
)

// WordOrder controls the order of 16-bit registers for 32/64-bit values.
type WordOrder string

const (
	WordOrderBig    WordOrder = "big"
	WordOrderLittle WordOrder = "little"
)

// LinearInterpolation maps a decoded numeric value from x1-x2 into y1-y2.
type LinearInterpolation struct {
	X1 float64
	X2 float64
	Y1 float64
	Y2 float64
}

// RegisterValueSpec describes one value conversion.
type RegisterValueSpec struct {
	DataType      DataType
	WordOrder     WordOrder
	Length        uint16
	ScalingFactor float64
	Interpolate   *LinearInterpolation
	LocalTime     bool
}

// DecodedRegisterValue is one interpreted register value.
type DecodedRegisterValue struct {
	DataType DataType
	Raw      []uint16
	Display  string
	Numeric  float64
}

// RegisterMapping describes a configured register value in a read map.
type RegisterMapping struct {
	Address       uint16
	DataType      DataType
	WordOrder     WordOrder
	Length        uint16
	ScalingFactor float64
	Comment       string
	Interpolate   *LinearInterpolation
	GroupEnd      bool
}

// AddressGroup is a continuous Modbus read block.
type AddressGroup struct {
	Address  uint16
	Quantity uint16
}

// UnitScanRequest probes a set of Unit IDs with a small read request.
type UnitScanRequest struct {
	SessionID   string
	UnitIDs     []int
	Function    FunctionCode
	AddressMode AddressMode
	Address     uint16
	Quantity    uint16
	TimeoutMs   int
	Retries     int
}

// UnitScanResult reports which Unit IDs responded.
type UnitScanResult struct {
	SessionID     string
	ActiveUnitIDs []int
	Results       []UnitScanEntry
}

// UnitScanEntry is one Unit ID probe result.
type UnitScanEntry struct {
	UnitID byte
	Active bool
	Error  string
}

// RegisterReadRequest reads one register/coil block and optionally decodes mappings.
type RegisterReadRequest struct {
	SessionID   string
	UnitID      byte
	Function    FunctionCode
	AddressMode AddressMode
	Address     uint16
	Quantity    uint16
	Mappings    []RegisterMapping
	TimeoutMs   int
	Retries     int
}

// RegisterReadResult contains raw and mapped read data.
type RegisterReadResult struct {
	Transaction  *Transaction
	RawRegisters []uint16
	Bits         []bool
	Values       []MappedRegisterValue
}

// MappedRegisterValue is one decoded configured register.
type MappedRegisterValue struct {
	Address uint16
	Mapping RegisterMapping
	Value   DecodedRegisterValue
	Error   string
}

// RegisterScanRequest scans a register range in chunks.
type RegisterScanRequest struct {
	SessionID    string
	UnitID       byte
	Function     FunctionCode
	AddressMode  AddressMode
	StartAddress uint16
	EndAddress   uint16
	ChunkSize    uint16
	TimeoutMs    int
	Retries      int
}

// RegisterScanResult reports non-zero register values and range errors.
type RegisterScanResult struct {
	SessionID string
	UnitID    byte
	Values    []RegisterScanValue
	Ranges    []RegisterScanRange
}

// RegisterScanValue is a non-zero register discovered during a scan.
type RegisterScanValue struct {
	Address uint16
	Value   uint16
}

// RegisterScanRange is one scanned read chunk.
type RegisterScanRange struct {
	Address  uint16
	Quantity uint16
	Error    string
}
