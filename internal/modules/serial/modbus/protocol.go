package modbus

import (
	"encoding/hex"
	"fmt"
	"strings"
)

const exceptionMask = 0x80

// FrameMode selects the Modbus serial framing format.
type FrameMode string

const (
	FrameModeRTU   FrameMode = "rtu"
	FrameModeASCII FrameMode = "ascii"
)

// AddressMode selects how UI addresses are converted to protocol offsets.
type AddressMode string

const (
	AddressModeZeroBased AddressMode = "zero-based"
	AddressModePLC       AddressMode = "plc"
)

// FunctionCode is a Modbus function byte.
type FunctionCode byte

const (
	FunctionReadCoils              FunctionCode = 0x01
	FunctionReadDiscreteInputs     FunctionCode = 0x02
	FunctionReadHoldingRegisters   FunctionCode = 0x03
	FunctionReadInputRegisters     FunctionCode = 0x04
	FunctionWriteSingleCoil        FunctionCode = 0x05
	FunctionWriteSingleRegister    FunctionCode = 0x06
	FunctionWriteMultipleCoils     FunctionCode = 0x0f
	FunctionWriteMultipleRegisters FunctionCode = 0x10
)

const (
	ExceptionIllegalFunction    byte = 0x01
	ExceptionIllegalDataAddress byte = 0x02
	ExceptionIllegalDataValue   byte = 0x03
	ExceptionServerFailure      byte = 0x04
)

// PDU is the function byte plus function-specific data.
type PDU struct {
	Function FunctionCode
	Data     []byte
}

// Bytes returns the wire representation of the PDU.
func (p PDU) Bytes() []byte {
	out := make([]byte, 1, 1+len(p.Data))
	out[0] = byte(p.Function)
	out = append(out, p.Data...)
	return out
}

// DecodedFrame is a verified serial ADU.
type DecodedFrame struct {
	Mode   FrameMode
	UnitID byte
	PDU    PDU
}

// ParsedResponse is a decoded master-side response.
type ParsedResponse struct {
	Function      FunctionCode
	Exception     bool
	ExceptionCode byte
	Address       uint16
	Quantity      uint16
	Value         uint16
	Values        []uint16
	Bits          []bool
	Raw           []byte
}

// SlaveResponse contains a generated slave reply.
type SlaveResponse struct {
	PDU           PDU
	Exception     bool
	ExceptionCode byte
}

// EncodeFrame encodes a unit id and PDU as RTU or ASCII.
func EncodeFrame(mode FrameMode, unitID byte, pdu PDU) ([]byte, error) {
	if unitID == 0 {
		return nil, fmt.Errorf("unit id must be 1-247")
	}
	payload := append([]byte{unitID}, pdu.Bytes()...)
	switch normalizeFrameMode(mode) {
	case FrameModeRTU:
		crc := CRC16(payload)
		return append(payload, byte(crc), byte(crc>>8)), nil
	case FrameModeASCII:
		lrc := LRC(payload)
		body := append(payload, lrc)
		out := make([]byte, 0, 1+len(body)*2+2)
		out = append(out, ':')
		hexBody := strings.ToUpper(hex.EncodeToString(body))
		out = append(out, hexBody...)
		out = append(out, '\r', '\n')
		return out, nil
	default:
		return nil, fmt.Errorf("unsupported frame mode: %s", mode)
	}
}

// DecodeFrame verifies and decodes an RTU or ASCII serial frame.
func DecodeFrame(mode FrameMode, frame []byte) (DecodedFrame, error) {
	switch normalizeFrameMode(mode) {
	case FrameModeRTU:
		return decodeRTU(frame)
	case FrameModeASCII:
		return decodeASCII(frame)
	default:
		return DecodedFrame{}, fmt.Errorf("unsupported frame mode: %s", mode)
	}
}

func decodeRTU(frame []byte) (DecodedFrame, error) {
	if len(frame) < 4 {
		return DecodedFrame{}, fmt.Errorf("rtu frame too short")
	}
	payload := frame[:len(frame)-2]
	got := uint16(frame[len(frame)-2]) | uint16(frame[len(frame)-1])<<8
	if want := CRC16(payload); got != want {
		return DecodedFrame{}, fmt.Errorf("rtu crc mismatch")
	}
	return decodedFromPayload(FrameModeRTU, payload)
}

func decodeASCII(frame []byte) (DecodedFrame, error) {
	text := string(frame)
	if !strings.HasPrefix(text, ":") || !strings.HasSuffix(text, "\r\n") {
		return DecodedFrame{}, fmt.Errorf("ascii frame must start with ':' and end with CRLF")
	}
	bodyText := strings.TrimSuffix(strings.TrimPrefix(text, ":"), "\r\n")
	if len(bodyText) < 6 || len(bodyText)%2 != 0 {
		return DecodedFrame{}, fmt.Errorf("ascii frame has invalid hex length")
	}
	body, err := hex.DecodeString(bodyText)
	if err != nil {
		return DecodedFrame{}, fmt.Errorf("ascii hex decode: %w", err)
	}
	payload := body[:len(body)-1]
	got := body[len(body)-1]
	if want := LRC(payload); got != want {
		return DecodedFrame{}, fmt.Errorf("ascii lrc mismatch")
	}
	return decodedFromPayload(FrameModeASCII, payload)
}

func decodedFromPayload(mode FrameMode, payload []byte) (DecodedFrame, error) {
	if len(payload) < 2 {
		return DecodedFrame{}, fmt.Errorf("frame payload too short")
	}
	return DecodedFrame{
		Mode:   mode,
		UnitID: payload[0],
		PDU: PDU{
			Function: FunctionCode(payload[1]),
			Data:     append([]byte(nil), payload[2:]...),
		},
	}, nil
}

func normalizeFrameMode(mode FrameMode) FrameMode {
	if mode == "" {
		return FrameModeRTU
	}
	return FrameMode(strings.ToLower(string(mode)))
}

// CRC16 returns the Modbus RTU CRC.
func CRC16(data []byte) uint16 {
	var crc uint16 = 0xffff
	for _, b := range data {
		crc ^= uint16(b)
		for i := 0; i < 8; i++ {
			if crc&1 != 0 {
				crc = (crc >> 1) ^ 0xa001
			} else {
				crc >>= 1
			}
		}
	}
	return crc
}

// LRC returns the Modbus ASCII longitudinal redundancy check.
func LRC(data []byte) byte {
	var sum byte
	for _, b := range data {
		sum += b
	}
	return byte(-int8(sum))
}

// BuildReadRequest creates a function 01-04 read request.
func BuildReadRequest(function FunctionCode, address uint16, quantity uint16) (PDU, error) {
	if function != FunctionReadCoils &&
		function != FunctionReadDiscreteInputs &&
		function != FunctionReadHoldingRegisters &&
		function != FunctionReadInputRegisters {
		return PDU{}, fmt.Errorf("unsupported read function: %02x", byte(function))
	}
	if quantity == 0 {
		return PDU{}, fmt.Errorf("quantity must be positive")
	}
	return PDU{Function: function, Data: u16Pair(address, quantity)}, nil
}

// MustBuildReadRequest panics if BuildReadRequest fails.
func MustBuildReadRequest(function FunctionCode, address uint16, quantity uint16) PDU {
	pdu, err := BuildReadRequest(function, address, quantity)
	if err != nil {
		panic(err)
	}
	return pdu
}

// BuildWriteSingleRequest creates function 05 or 06 requests.
func BuildWriteSingleRequest(function FunctionCode, address uint16, value uint16) (PDU, error) {
	switch function {
	case FunctionWriteSingleCoil:
		if value != 0 {
			value = 0xff00
		}
	case FunctionWriteSingleRegister:
	default:
		return PDU{}, fmt.Errorf("unsupported single write function: %02x", byte(function))
	}
	return PDU{Function: function, Data: u16Pair(address, value)}, nil
}

// MustBuildWriteSingleRequest panics if BuildWriteSingleRequest fails.
func MustBuildWriteSingleRequest(function FunctionCode, address uint16, value uint16) PDU {
	pdu, err := BuildWriteSingleRequest(function, address, value)
	if err != nil {
		panic(err)
	}
	return pdu
}

// BuildWriteMultipleCoilsRequest creates function 15 requests.
func BuildWriteMultipleCoilsRequest(address uint16, values []bool) (PDU, error) {
	if len(values) == 0 {
		return PDU{}, fmt.Errorf("coil values must not be empty")
	}
	if len(values) > 1968 {
		return PDU{}, fmt.Errorf("too many coil values")
	}
	packed := packBits(values)
	data := make([]byte, 0, 5+len(packed))
	data = append(data, u16(address)...)
	data = append(data, u16(uint16(len(values)))...)
	data = append(data, byte(len(packed)))
	data = append(data, packed...)
	return PDU{Function: FunctionWriteMultipleCoils, Data: data}, nil
}

// MustBuildWriteMultipleCoilsRequest panics if BuildWriteMultipleCoilsRequest fails.
func MustBuildWriteMultipleCoilsRequest(address uint16, values []bool) PDU {
	pdu, err := BuildWriteMultipleCoilsRequest(address, values)
	if err != nil {
		panic(err)
	}
	return pdu
}

// BuildWriteMultipleRegistersRequest creates function 16 requests.
func BuildWriteMultipleRegistersRequest(address uint16, values []uint16) (PDU, error) {
	if len(values) == 0 {
		return PDU{}, fmt.Errorf("register values must not be empty")
	}
	if len(values) > 123 {
		return PDU{}, fmt.Errorf("too many register values")
	}
	data := make([]byte, 0, 5+len(values)*2)
	data = append(data, u16(address)...)
	data = append(data, u16(uint16(len(values)))...)
	data = append(data, byte(len(values)*2))
	for _, value := range values {
		data = append(data, u16(value)...)
	}
	return PDU{Function: FunctionWriteMultipleRegisters, Data: data}, nil
}

// MustBuildWriteMultipleRegistersRequest panics if BuildWriteMultipleRegistersRequest fails.
func MustBuildWriteMultipleRegistersRequest(address uint16, values []uint16) PDU {
	pdu, err := BuildWriteMultipleRegistersRequest(address, values)
	if err != nil {
		panic(err)
	}
	return pdu
}

// ParseResponse parses a Modbus PDU response.
func ParseResponse(pdu PDU) (ParsedResponse, error) {
	raw := pdu.Bytes()
	if pdu.Function&exceptionMask != 0 {
		if len(pdu.Data) != 1 {
			return ParsedResponse{}, fmt.Errorf("invalid exception response")
		}
		return ParsedResponse{
			Function:      pdu.Function &^ exceptionMask,
			Exception:     true,
			ExceptionCode: pdu.Data[0],
			Raw:           raw,
		}, nil
	}

	response := ParsedResponse{Function: pdu.Function, Raw: raw}
	switch pdu.Function {
	case FunctionReadCoils, FunctionReadDiscreteInputs:
		if len(pdu.Data) < 1 {
			return ParsedResponse{}, fmt.Errorf("invalid bit read response")
		}
		byteCount := int(pdu.Data[0])
		if len(pdu.Data[1:]) != byteCount {
			return ParsedResponse{}, fmt.Errorf("bit read byte count mismatch")
		}
		response.Bits = unpackBits(pdu.Data[1:], byteCount*8)
	case FunctionReadHoldingRegisters, FunctionReadInputRegisters:
		if len(pdu.Data) < 1 {
			return ParsedResponse{}, fmt.Errorf("invalid register read response")
		}
		byteCount := int(pdu.Data[0])
		if byteCount%2 != 0 || len(pdu.Data[1:]) != byteCount {
			return ParsedResponse{}, fmt.Errorf("register read byte count mismatch")
		}
		response.Values = bytesToU16s(pdu.Data[1:])
	case FunctionWriteSingleCoil, FunctionWriteSingleRegister:
		if len(pdu.Data) != 4 {
			return ParsedResponse{}, fmt.Errorf("invalid single write response")
		}
		response.Address = beU16(pdu.Data[0:2])
		response.Value = beU16(pdu.Data[2:4])
	case FunctionWriteMultipleCoils, FunctionWriteMultipleRegisters:
		if len(pdu.Data) != 4 {
			return ParsedResponse{}, fmt.Errorf("invalid multiple write response")
		}
		response.Address = beU16(pdu.Data[0:2])
		response.Quantity = beU16(pdu.Data[2:4])
	default:
		return ParsedResponse{}, fmt.Errorf("unsupported response function: %02x", byte(pdu.Function))
	}
	return response, nil
}

// ProtocolAddress converts a UI address to a zero-based Modbus protocol address.
func ProtocolAddress(mode AddressMode, function FunctionCode, input uint16) (uint16, error) {
	if mode == "" || mode == AddressModeZeroBased {
		return input, nil
	}
	if mode != AddressModePLC {
		return 0, fmt.Errorf("unsupported address mode: %s", mode)
	}
	base, err := plcBase(function)
	if err != nil {
		return 0, err
	}
	if input < base+1 {
		return 0, fmt.Errorf("plc address %d is below base %d", input, base+1)
	}
	offset := input - base - 1
	if !plcRangeMatches(function, input) {
		return 0, fmt.Errorf("plc address %d does not match function %02x", input, byte(function))
	}
	return offset, nil
}

func plcBase(function FunctionCode) (uint16, error) {
	switch function {
	case FunctionReadCoils, FunctionWriteSingleCoil, FunctionWriteMultipleCoils:
		return 0, nil
	case FunctionReadDiscreteInputs:
		return 10000, nil
	case FunctionReadInputRegisters:
		return 30000, nil
	case FunctionReadHoldingRegisters, FunctionWriteSingleRegister, FunctionWriteMultipleRegisters:
		return 40000, nil
	default:
		return 0, fmt.Errorf("unsupported function for plc address: %02x", byte(function))
	}
}

func plcRangeMatches(function FunctionCode, input uint16) bool {
	switch function {
	case FunctionReadCoils, FunctionWriteSingleCoil, FunctionWriteMultipleCoils:
		return input >= 1 && input < 10000
	case FunctionReadDiscreteInputs:
		return input >= 10001 && input < 20000
	case FunctionReadInputRegisters:
		return input >= 30001 && input < 40000
	case FunctionReadHoldingRegisters, FunctionWriteSingleRegister, FunctionWriteMultipleRegisters:
		return input >= 40001 && input < 50000
	default:
		return false
	}
}

func u16(value uint16) []byte {
	return []byte{byte(value >> 8), byte(value)}
}

func u16Pair(first, second uint16) []byte {
	return []byte{byte(first >> 8), byte(first), byte(second >> 8), byte(second)}
}

func beU16(data []byte) uint16 {
	return uint16(data[0])<<8 | uint16(data[1])
}

func bytesToU16s(data []byte) []uint16 {
	values := make([]uint16, 0, len(data)/2)
	for i := 0; i+1 < len(data); i += 2 {
		values = append(values, beU16(data[i:i+2]))
	}
	return values
}

func packBits(values []bool) []byte {
	out := make([]byte, (len(values)+7)/8)
	for i, value := range values {
		if value {
			out[i/8] |= 1 << uint(i%8)
		}
	}
	return out
}

func unpackBits(data []byte, count int) []bool {
	out := make([]bool, count)
	for i := 0; i < count; i++ {
		out[i] = data[i/8]&(1<<uint(i%8)) != 0
	}
	return out
}
