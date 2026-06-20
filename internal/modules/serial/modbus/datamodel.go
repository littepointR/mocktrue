package modbus

import "fmt"

// DataModel is the four-table Modbus slave data model.
type DataModel struct {
	Coils            map[uint16]bool
	DiscreteInputs   map[uint16]bool
	InputRegisters   map[uint16]uint16
	HoldingRegisters map[uint16]uint16
}

// NewDataModel constructs an empty slave data model.
func NewDataModel() *DataModel {
	return &DataModel{
		Coils:            make(map[uint16]bool),
		DiscreteInputs:   make(map[uint16]bool),
		InputRegisters:   make(map[uint16]uint16),
		HoldingRegisters: make(map[uint16]uint16),
	}
}

// Clone returns a deep copy of the data model.
func (m *DataModel) Clone() *DataModel {
	if m == nil {
		return NewDataModel()
	}
	out := NewDataModel()
	copyBoolMap(out.Coils, m.Coils)
	copyBoolMap(out.DiscreteInputs, m.DiscreteInputs)
	copyU16Map(out.InputRegisters, m.InputRegisters)
	copyU16Map(out.HoldingRegisters, m.HoldingRegisters)
	return out
}

func copyBoolMap(dst map[uint16]bool, src map[uint16]bool) {
	for k, v := range src {
		dst[k] = v
	}
}

func copyU16Map(dst map[uint16]uint16, src map[uint16]uint16) {
	for k, v := range src {
		dst[k] = v
	}
}

// ReadCoils reads coil values.
func (m *DataModel) ReadCoils(address uint16, quantity uint16) ([]bool, error) {
	return readBoolRange(m.Coils, address, quantity)
}

// ReadDiscreteInputs reads discrete input values.
func (m *DataModel) ReadDiscreteInputs(address uint16, quantity uint16) ([]bool, error) {
	return readBoolRange(m.DiscreteInputs, address, quantity)
}

// ReadInputRegisters reads input registers.
func (m *DataModel) ReadInputRegisters(address uint16, quantity uint16) ([]uint16, error) {
	return readU16Range(m.InputRegisters, address, quantity)
}

// ReadHoldingRegisters reads holding registers.
func (m *DataModel) ReadHoldingRegisters(address uint16, quantity uint16) ([]uint16, error) {
	return readU16Range(m.HoldingRegisters, address, quantity)
}

// WriteCoils writes coil values.
func (m *DataModel) WriteCoils(address uint16, values []bool) error {
	if len(values) == 0 {
		return fmt.Errorf("coil values must not be empty")
	}
	for i, value := range values {
		m.Coils[address+uint16(i)] = value
	}
	return nil
}

// WriteHoldingRegisters writes holding register values.
func (m *DataModel) WriteHoldingRegisters(address uint16, values []uint16) error {
	if len(values) == 0 {
		return fmt.Errorf("register values must not be empty")
	}
	for i, value := range values {
		m.HoldingRegisters[address+uint16(i)] = value
	}
	return nil
}

// WriteDiscreteInputs writes discrete input values.
func (m *DataModel) WriteDiscreteInputs(address uint16, values []bool) error {
	if len(values) == 0 {
		return fmt.Errorf("discrete input values must not be empty")
	}
	for i, value := range values {
		m.DiscreteInputs[address+uint16(i)] = value
	}
	return nil
}

// WriteInputRegisters writes input register values.
func (m *DataModel) WriteInputRegisters(address uint16, values []uint16) error {
	if len(values) == 0 {
		return fmt.Errorf("input register values must not be empty")
	}
	for i, value := range values {
		m.InputRegisters[address+uint16(i)] = value
	}
	return nil
}

func readBoolRange(values map[uint16]bool, address uint16, quantity uint16) ([]bool, error) {
	if quantity == 0 {
		return nil, fmt.Errorf("quantity must be positive")
	}
	out := make([]bool, quantity)
	for i := range out {
		out[i] = values[address+uint16(i)]
	}
	return out, nil
}

func readU16Range(values map[uint16]uint16, address uint16, quantity uint16) ([]uint16, error) {
	if quantity == 0 {
		return nil, fmt.Errorf("quantity must be positive")
	}
	out := make([]uint16, quantity)
	for i := range out {
		out[i] = values[address+uint16(i)]
	}
	return out, nil
}

// BuildSlaveResponse applies a request PDU to a slave data model.
func BuildSlaveResponse(unitID byte, request PDU, model *DataModel) SlaveResponse {
	_ = unitID
	if model == nil {
		model = NewDataModel()
	}
	if request.Function&exceptionMask != 0 {
		return exceptionResponse(request.Function&^exceptionMask, ExceptionIllegalFunction)
	}
	switch request.Function {
	case FunctionReadCoils:
		address, quantity, ok := parseAddressQuantity(request.Data)
		if !ok {
			return exceptionResponse(request.Function, ExceptionIllegalDataValue)
		}
		values, err := model.ReadCoils(address, quantity)
		if err != nil {
			return exceptionResponse(request.Function, ExceptionIllegalDataAddress)
		}
		packed := packBits(values)
		return normalResponse(PDU{Function: request.Function, Data: append([]byte{byte(len(packed))}, packed...)})
	case FunctionReadDiscreteInputs:
		address, quantity, ok := parseAddressQuantity(request.Data)
		if !ok {
			return exceptionResponse(request.Function, ExceptionIllegalDataValue)
		}
		values, err := model.ReadDiscreteInputs(address, quantity)
		if err != nil {
			return exceptionResponse(request.Function, ExceptionIllegalDataAddress)
		}
		packed := packBits(values)
		return normalResponse(PDU{Function: request.Function, Data: append([]byte{byte(len(packed))}, packed...)})
	case FunctionReadHoldingRegisters:
		address, quantity, ok := parseAddressQuantity(request.Data)
		if !ok {
			return exceptionResponse(request.Function, ExceptionIllegalDataValue)
		}
		values, err := model.ReadHoldingRegisters(address, quantity)
		if err != nil {
			return exceptionResponse(request.Function, ExceptionIllegalDataAddress)
		}
		return normalResponse(registerReadResponse(request.Function, values))
	case FunctionReadInputRegisters:
		address, quantity, ok := parseAddressQuantity(request.Data)
		if !ok {
			return exceptionResponse(request.Function, ExceptionIllegalDataValue)
		}
		values, err := model.ReadInputRegisters(address, quantity)
		if err != nil {
			return exceptionResponse(request.Function, ExceptionIllegalDataAddress)
		}
		return normalResponse(registerReadResponse(request.Function, values))
	case FunctionWriteSingleCoil:
		if len(request.Data) != 4 {
			return exceptionResponse(request.Function, ExceptionIllegalDataValue)
		}
		address := beU16(request.Data[:2])
		value := beU16(request.Data[2:])
		if value != 0x0000 && value != 0xff00 {
			return exceptionResponse(request.Function, ExceptionIllegalDataValue)
		}
		_ = model.WriteCoils(address, []bool{value == 0xff00})
		return normalResponse(request)
	case FunctionWriteSingleRegister:
		if len(request.Data) != 4 {
			return exceptionResponse(request.Function, ExceptionIllegalDataValue)
		}
		address := beU16(request.Data[:2])
		value := beU16(request.Data[2:])
		_ = model.WriteHoldingRegisters(address, []uint16{value})
		return normalResponse(request)
	case FunctionWriteMultipleCoils:
		address, quantity, byteCount, values, ok := parseMultipleCoils(request.Data)
		_ = byteCount
		if !ok {
			return exceptionResponse(request.Function, ExceptionIllegalDataValue)
		}
		_ = model.WriteCoils(address, values[:quantity])
		return normalResponse(PDU{Function: request.Function, Data: u16Pair(address, uint16(quantity))})
	case FunctionWriteMultipleRegisters:
		address, values, ok := parseMultipleRegisters(request.Data)
		if !ok {
			return exceptionResponse(request.Function, ExceptionIllegalDataValue)
		}
		_ = model.WriteHoldingRegisters(address, values)
		return normalResponse(PDU{Function: request.Function, Data: u16Pair(address, uint16(len(values)))})
	default:
		return exceptionResponse(request.Function, ExceptionIllegalFunction)
	}
}

func parseAddressQuantity(data []byte) (uint16, uint16, bool) {
	if len(data) != 4 {
		return 0, 0, false
	}
	quantity := beU16(data[2:])
	if quantity == 0 {
		return 0, 0, false
	}
	return beU16(data[:2]), quantity, true
}

func parseMultipleCoils(data []byte) (uint16, int, int, []bool, bool) {
	if len(data) < 6 {
		return 0, 0, 0, nil, false
	}
	address := beU16(data[:2])
	quantity := int(beU16(data[2:4]))
	byteCount := int(data[4])
	if quantity <= 0 || byteCount <= 0 || len(data[5:]) != byteCount {
		return 0, 0, 0, nil, false
	}
	values := unpackBits(data[5:], byteCount*8)
	if quantity > len(values) {
		return 0, 0, 0, nil, false
	}
	return address, quantity, byteCount, values, true
}

func parseMultipleRegisters(data []byte) (uint16, []uint16, bool) {
	if len(data) < 6 {
		return 0, nil, false
	}
	address := beU16(data[:2])
	quantity := int(beU16(data[2:4]))
	byteCount := int(data[4])
	if quantity <= 0 || byteCount != quantity*2 || len(data[5:]) != byteCount {
		return 0, nil, false
	}
	return address, bytesToU16s(data[5:]), true
}

func registerReadResponse(function FunctionCode, values []uint16) PDU {
	data := make([]byte, 0, 1+len(values)*2)
	data = append(data, byte(len(values)*2))
	for _, value := range values {
		data = append(data, u16(value)...)
	}
	return PDU{Function: function, Data: data}
}

func normalResponse(pdu PDU) SlaveResponse {
	return SlaveResponse{PDU: pdu}
}

func exceptionResponse(function FunctionCode, code byte) SlaveResponse {
	return SlaveResponse{
		PDU:           PDU{Function: function | exceptionMask, Data: []byte{code}},
		Exception:     true,
		ExceptionCode: code,
	}
}
