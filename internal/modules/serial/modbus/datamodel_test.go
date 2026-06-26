package modbus

import "testing"

func TestDataModelTablesCloneAndSnapshots(t *testing.T) {
	var nilModel *DataModel
	if clone := nilModel.Clone(); clone == nil {
		t.Fatalf("nil Clone() returned nil")
	}

	model := NewDataModel()
	if err := model.WriteCoils(0, nil); err == nil {
		t.Fatalf("WriteCoils(nil) error = nil, want error")
	}
	if err := model.WriteHoldingRegisters(0, nil); err == nil {
		t.Fatalf("WriteHoldingRegisters(nil) error = nil, want error")
	}
	if err := model.WriteDiscreteInputs(0, nil); err == nil {
		t.Fatalf("WriteDiscreteInputs(nil) error = nil, want error")
	}
	if err := model.WriteInputRegisters(0, nil); err == nil {
		t.Fatalf("WriteInputRegisters(nil) error = nil, want error")
	}
	if _, err := model.ReadDiscreteInputs(0, 0); err == nil {
		t.Fatalf("ReadDiscreteInputs(quantity=0) error = nil, want error")
	}
	if _, err := model.ReadInputRegisters(0, 0); err == nil {
		t.Fatalf("ReadInputRegisters(quantity=0) error = nil, want error")
	}

	if err := model.WriteCoils(5, []bool{true, false}); err != nil {
		t.Fatalf("WriteCoils() error = %v", err)
	}
	if err := model.WriteDiscreteInputs(7, []bool{false, true}); err != nil {
		t.Fatalf("WriteDiscreteInputs() error = %v", err)
	}
	if err := model.WriteInputRegisters(9, []uint16{11, 12}); err != nil {
		t.Fatalf("WriteInputRegisters() error = %v", err)
	}
	if err := model.WriteHoldingRegisters(11, []uint16{13, 14}); err != nil {
		t.Fatalf("WriteHoldingRegisters() error = %v", err)
	}

	clone := model.Clone()
	if err := model.WriteCoils(5, []bool{false}); err != nil {
		t.Fatalf("WriteCoils(mutate) error = %v", err)
	}
	cloneCoils, err := clone.ReadCoils(5, 2)
	if err != nil {
		t.Fatalf("clone.ReadCoils() error = %v", err)
	}
	if got, want := cloneCoils, []bool{true, false}; !equalBool(got, want) {
		t.Fatalf("clone coils = %v, want %v", got, want)
	}

	snapshot := SnapshotFromDataModel(model)
	roundTrip := DataModelFromSnapshot(snapshot)
	coils, err := roundTrip.ReadCoils(5, 2)
	if err != nil {
		t.Fatalf("roundTrip.ReadCoils() error = %v", err)
	}
	if got, want := coils, []bool{false, false}; !equalBool(got, want) {
		t.Fatalf("roundTrip coils = %v, want %v", got, want)
	}
	discrete, err := roundTrip.ReadDiscreteInputs(7, 2)
	if err != nil {
		t.Fatalf("roundTrip.ReadDiscreteInputs() error = %v", err)
	}
	if got, want := discrete, []bool{false, true}; !equalBool(got, want) {
		t.Fatalf("roundTrip discrete inputs = %v, want %v", got, want)
	}
	inputRegs, err := roundTrip.ReadInputRegisters(9, 2)
	if err != nil {
		t.Fatalf("roundTrip.ReadInputRegisters() error = %v", err)
	}
	if got, want := inputRegs, []uint16{11, 12}; !equalU16(got, want) {
		t.Fatalf("roundTrip input registers = %v, want %v", got, want)
	}
	holding, err := roundTrip.ReadHoldingRegisters(11, 2)
	if err != nil {
		t.Fatalf("roundTrip.ReadHoldingRegisters() error = %v", err)
	}
	if got, want := holding, []uint16{13, 14}; !equalU16(got, want) {
		t.Fatalf("roundTrip holding registers = %v, want %v", got, want)
	}

	if empty := SnapshotFromDataModel(nil); len(empty.Coils)+len(empty.DiscreteInputs)+len(empty.InputRegisters)+len(empty.HoldingRegisters) != 0 {
		t.Fatalf("SnapshotFromDataModel(nil) = %#v, want empty", empty)
	}
}

func TestBuildSlaveResponseHandlesReadsWritesAndExceptions(t *testing.T) {
	model := NewDataModel()
	if err := model.WriteCoils(0, []bool{true, false, true}); err != nil {
		t.Fatalf("WriteCoils() error = %v", err)
	}
	if err := model.WriteDiscreteInputs(1, []bool{false, true, true}); err != nil {
		t.Fatalf("WriteDiscreteInputs() error = %v", err)
	}
	if err := model.WriteInputRegisters(2, []uint16{21, 22}); err != nil {
		t.Fatalf("WriteInputRegisters() error = %v", err)
	}
	if err := model.WriteHoldingRegisters(3, []uint16{31, 32}); err != nil {
		t.Fatalf("WriteHoldingRegisters() error = %v", err)
	}

	tests := []struct {
		name      string
		request   PDU
		wantHex   string
		exception bool
		code      byte
	}{
		{"read coils", MustBuildReadRequest(FunctionReadCoils, 0, 3), "010105", false, 0},
		{"read discrete inputs", MustBuildReadRequest(FunctionReadDiscreteInputs, 1, 3), "020106", false, 0},
		{"read input registers", MustBuildReadRequest(FunctionReadInputRegisters, 2, 2), "040400150016", false, 0},
		{"write single coil", MustBuildWriteSingleRequest(FunctionWriteSingleCoil, 10, 1), "05000aff00", false, 0},
		{"write single register", MustBuildWriteSingleRequest(FunctionWriteSingleRegister, 11, 123), "06000b007b", false, 0},
		{"write multiple coils", MustBuildWriteMultipleCoilsRequest(12, []bool{true, false, true, true}), "0f000c0004", false, 0},
		{"write multiple registers", MustBuildWriteMultipleRegistersRequest(13, []uint16{201, 202}), "10000d0002", false, 0},
		{"exception request", PDU{Function: FunctionReadCoils | exceptionMask, Data: []byte{0}}, "8101", true, ExceptionIllegalFunction},
		{"invalid read data", PDU{Function: FunctionReadCoils, Data: []byte{0, 1}}, "8103", true, ExceptionIllegalDataValue},
		{"invalid single coil value", PDU{Function: FunctionWriteSingleCoil, Data: []byte{0, 1, 0x12, 0x34}}, "8503", true, ExceptionIllegalDataValue},
		{"unsupported function", PDU{Function: FunctionCode(0x2b)}, "ab01", true, ExceptionIllegalFunction},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := BuildSlaveResponse(1, tt.request, model)
			if response.Exception != tt.exception {
				t.Fatalf("Exception = %v, want %v", response.Exception, tt.exception)
			}
			if response.ExceptionCode != tt.code {
				t.Fatalf("ExceptionCode = %d, want %d", response.ExceptionCode, tt.code)
			}
			assertHex(t, response.PDU.Bytes(), tt.wantHex)
		})
	}

	coils, err := model.ReadCoils(10, 3)
	if err != nil {
		t.Fatalf("ReadCoils() after writes error = %v", err)
	}
	if got, want := coils, []bool{true, false, true}; !equalBool(got, want) {
		t.Fatalf("coils after writes = %v, want %v", got, want)
	}
	holding, err := model.ReadHoldingRegisters(11, 4)
	if err != nil {
		t.Fatalf("ReadHoldingRegisters() after writes error = %v", err)
	}
	if got, want := holding, []uint16{123, 0, 201, 202}; !equalU16(got, want) {
		t.Fatalf("holding after writes = %v, want %v", got, want)
	}
}

func TestBuildSlaveResponseRejectsMalformedMultipleWrites(t *testing.T) {
	model := NewDataModel()
	tests := []PDU{
		{Function: FunctionWriteMultipleCoils, Data: []byte{0, 1, 0, 1, 2, 0xff}},
		{Function: FunctionWriteMultipleCoils, Data: []byte{0, 1, 0, 0, 1, 0xff}},
		{Function: FunctionWriteMultipleRegisters, Data: []byte{0, 1, 0, 2, 3, 0, 1, 0}},
		{Function: FunctionWriteMultipleRegisters, Data: []byte{0, 1, 0, 0, 0}},
	}
	for _, request := range tests {
		response := BuildSlaveResponse(1, request, model)
		if !response.Exception || response.ExceptionCode != ExceptionIllegalDataValue {
			t.Fatalf("BuildSlaveResponse(%#v) = %#v, want illegal data value", request, response)
		}
	}
}
