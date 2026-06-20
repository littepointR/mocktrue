package modbus

import (
	"encoding/hex"
	"testing"
)

func TestRTUGoldenFrame(t *testing.T) {
	pdu := MustBuildReadRequest(FunctionReadHoldingRegisters, 0, 2)
	frame, err := EncodeFrame(FrameModeRTU, 1, pdu)
	if err != nil {
		t.Fatalf("EncodeFrame() error = %v", err)
	}
	assertHex(t, frame, "010300000002c40b")

	decoded, err := DecodeFrame(FrameModeRTU, frame)
	if err != nil {
		t.Fatalf("DecodeFrame() error = %v", err)
	}
	if decoded.UnitID != 1 {
		t.Fatalf("UnitID = %d, want 1", decoded.UnitID)
	}
	assertHex(t, decoded.PDU.Bytes(), "0300000002")
}

func TestASCIIGoldenFrame(t *testing.T) {
	pdu := MustBuildReadRequest(FunctionReadHoldingRegisters, 0, 2)
	frame, err := EncodeFrame(FrameModeASCII, 1, pdu)
	if err != nil {
		t.Fatalf("EncodeFrame() error = %v", err)
	}
	if got, want := string(frame), ":010300000002FA\r\n"; got != want {
		t.Fatalf("ASCII frame = %q, want %q", got, want)
	}

	decoded, err := DecodeFrame(FrameModeASCII, frame)
	if err != nil {
		t.Fatalf("DecodeFrame() error = %v", err)
	}
	if decoded.UnitID != 1 {
		t.Fatalf("UnitID = %d, want 1", decoded.UnitID)
	}
	assertHex(t, decoded.PDU.Bytes(), "0300000002")
}

func TestParseHoldingRegisterResponse(t *testing.T) {
	frame := mustHex(t, "0103040018002afbeb")
	decoded, err := DecodeFrame(FrameModeRTU, frame)
	if err != nil {
		t.Fatalf("DecodeFrame() error = %v", err)
	}
	response, err := ParseResponse(decoded.PDU)
	if err != nil {
		t.Fatalf("ParseResponse() error = %v", err)
	}
	if response.Function != FunctionReadHoldingRegisters {
		t.Fatalf("Function = %d, want %d", response.Function, FunctionReadHoldingRegisters)
	}
	if got, want := response.Values, []uint16{24, 42}; !equalU16(got, want) {
		t.Fatalf("Values = %v, want %v", got, want)
	}
}

func TestParseExceptionResponse(t *testing.T) {
	pdu := PDU{Function: FunctionReadHoldingRegisters | exceptionMask, Data: []byte{2}}
	response, err := ParseResponse(pdu)
	if err != nil {
		t.Fatalf("ParseResponse() error = %v", err)
	}
	if !response.Exception {
		t.Fatalf("Exception = false, want true")
	}
	if response.ExceptionCode != 2 {
		t.Fatalf("ExceptionCode = %d, want 2", response.ExceptionCode)
	}
	if response.Function != FunctionReadHoldingRegisters {
		t.Fatalf("Function = %d, want %d", response.Function, FunctionReadHoldingRegisters)
	}
}

func TestBuildCommonRequests(t *testing.T) {
	tests := []struct {
		name string
		pdu  PDU
		want string
	}{
		{"read coils", MustBuildReadRequest(FunctionReadCoils, 1, 8), "0100010008"},
		{"read discrete inputs", MustBuildReadRequest(FunctionReadDiscreteInputs, 2, 4), "0200020004"},
		{"read input registers", MustBuildReadRequest(FunctionReadInputRegisters, 3, 2), "0400030002"},
		{"write single coil", MustBuildWriteSingleRequest(FunctionWriteSingleCoil, 5, 1), "050005ff00"},
		{"write single register", MustBuildWriteSingleRequest(FunctionWriteSingleRegister, 6, 42), "060006002a"},
		{"write multiple coils", MustBuildWriteMultipleCoilsRequest(10, []bool{true, false, true, true, false, false, false, true, true}), "0f000a0009028d01"},
		{"write multiple registers", MustBuildWriteMultipleRegistersRequest(11, []uint16{1, 258}), "10000b00020400010102"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertHex(t, tt.pdu.Bytes(), tt.want)
		})
	}
}

func TestAddressConversion(t *testing.T) {
	tests := []struct {
		name         string
		mode         AddressMode
		function     FunctionCode
		input        uint16
		wantProtocol uint16
		wantErr      bool
	}{
		{"zero based", AddressModeZeroBased, FunctionReadHoldingRegisters, 12, 12, false},
		{"holding plc", AddressModePLC, FunctionReadHoldingRegisters, 40012, 11, false},
		{"input plc", AddressModePLC, FunctionReadInputRegisters, 30003, 2, false},
		{"coil plc", AddressModePLC, FunctionReadCoils, 8, 7, false},
		{"discrete plc", AddressModePLC, FunctionReadDiscreteInputs, 10006, 5, false},
		{"wrong plc range", AddressModePLC, FunctionReadHoldingRegisters, 30001, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ProtocolAddress(tt.mode, tt.function, tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ProtocolAddress() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("ProtocolAddress() error = %v", err)
			}
			if got != tt.wantProtocol {
				t.Fatalf("ProtocolAddress() = %d, want %d", got, tt.wantProtocol)
			}
		})
	}
}

func TestSlaveDataModelReadWrite(t *testing.T) {
	model := NewDataModel()
	if err := model.WriteHoldingRegisters(0, []uint16{24, 42}); err != nil {
		t.Fatalf("WriteHoldingRegisters() error = %v", err)
	}
	values, err := model.ReadHoldingRegisters(0, 2)
	if err != nil {
		t.Fatalf("ReadHoldingRegisters() error = %v", err)
	}
	if got, want := values, []uint16{24, 42}; !equalU16(got, want) {
		t.Fatalf("holding values = %v, want %v", got, want)
	}

	if err := model.WriteCoils(3, []bool{true, false, true}); err != nil {
		t.Fatalf("WriteCoils() error = %v", err)
	}
	coils, err := model.ReadCoils(3, 3)
	if err != nil {
		t.Fatalf("ReadCoils() error = %v", err)
	}
	if got, want := coils, []bool{true, false, true}; !equalBool(got, want) {
		t.Fatalf("coils = %v, want %v", got, want)
	}
}

func TestBuildSlaveResponse(t *testing.T) {
	model := NewDataModel()
	if err := model.WriteHoldingRegisters(0, []uint16{24, 42}); err != nil {
		t.Fatalf("WriteHoldingRegisters() error = %v", err)
	}
	response := BuildSlaveResponse(1, MustBuildReadRequest(FunctionReadHoldingRegisters, 0, 2), model)
	if response.Exception {
		t.Fatalf("response exception = %v, want normal", response.ExceptionCode)
	}
	assertHex(t, response.PDU.Bytes(), "03040018002a")
}

func mustHex(t *testing.T, value string) []byte {
	t.Helper()
	data, err := hex.DecodeString(value)
	if err != nil {
		t.Fatalf("hex.DecodeString(%q) error = %v", value, err)
	}
	return data
}

func assertHex(t *testing.T, got []byte, wantHex string) {
	t.Helper()
	if gotHex := hex.EncodeToString(got); gotHex != wantHex {
		t.Fatalf("hex = %s, want %s", gotHex, wantHex)
	}
}

func equalU16(a, b []uint16) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func equalBool(a, b []bool) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
