package modbus

import (
	"context"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/littepointR/mocktrue/internal/modules/serial/port"
)

func TestManagerMasterRequest(t *testing.T) {
	fake := newFakePort()
	mgr := NewManager(func(cfg port.SerialConfig) (port.Port, error) {
		return fake, nil
	})
	info, err := mgr.OpenSession(context.Background(), OpenSessionRequest{
		ID:   "master",
		Mode: FrameModeRTU,
		Config: port.SerialConfig{
			PortName: "loop",
			BaudRate: 115200,
		},
		TimeoutMs: 200,
	})
	if err != nil {
		t.Fatalf("OpenSession() error = %v", err)
	}
	if info.ID != "master" {
		t.Fatalf("session ID = %s, want master", info.ID)
	}

	fake.onWrite = func(data []byte) {
		decoded, err := DecodeFrame(FrameModeRTU, data)
		if err != nil {
			t.Errorf("DecodeFrame(request) error = %v", err)
			return
		}
		response := BuildSlaveResponse(decoded.UnitID, decoded.PDU, modelWithHolding(24, 42))
		frame, err := EncodeFrame(FrameModeRTU, decoded.UnitID, response.PDU)
		if err != nil {
			t.Errorf("EncodeFrame(response) error = %v", err)
			return
		}
		fake.pushRead(frame)
	}

	tx, err := mgr.MasterRequest(MasterRequest{
		SessionID: "master",
		UnitID:    1,
		Function:  FunctionReadHoldingRegisters,
		Address:   0,
		Quantity:  2,
	})
	if err != nil {
		t.Fatalf("MasterRequest() error = %v", err)
	}
	if tx.RequestFrameHex != "01 03 00 00 00 02 c4 0b" {
		t.Fatalf("request frame = %s", tx.RequestFrameHex)
	}
	if got, want := tx.Response.Values, []uint16{24, 42}; !equalU16(got, want) {
		t.Fatalf("response values = %v, want %v", got, want)
	}
}

func TestManagerSlaveResponds(t *testing.T) {
	fake := newFakePort()
	mgr := NewManager(func(cfg port.SerialConfig) (port.Port, error) {
		return fake, nil
	})
	if _, err := mgr.OpenSession(context.Background(), OpenSessionRequest{
		ID:     "slave",
		Mode:   FrameModeRTU,
		Config: port.SerialConfig{PortName: "loop", BaudRate: 115200},
	}); err != nil {
		t.Fatalf("OpenSession() error = %v", err)
	}
	if _, err := mgr.StartSlave(StartSlaveRequest{
		SessionID: "slave",
		UnitID:    2,
		DataModel: DataModelSnapshot{
			HoldingRegisters: []RegisterPoint{
				{Address: 0, Value: 24},
				{Address: 1, Value: 42},
			},
		},
	}); err != nil {
		t.Fatalf("StartSlave() error = %v", err)
	}
	defer mgr.StopSlave("slave")

	requestFrame, err := EncodeFrame(FrameModeRTU, 2, MustBuildReadRequest(FunctionReadHoldingRegisters, 0, 2))
	if err != nil {
		t.Fatalf("EncodeFrame() error = %v", err)
	}
	fake.pushRead(requestFrame)

	written := fake.waitWrite(t)
	decoded, err := DecodeFrame(FrameModeRTU, written)
	if err != nil {
		t.Fatalf("DecodeFrame(response) error = %v", err)
	}
	parsed, err := ParseResponse(decoded.PDU)
	if err != nil {
		t.Fatalf("ParseResponse() error = %v", err)
	}
	if got, want := parsed.Values, []uint16{24, 42}; !equalU16(got, want) {
		t.Fatalf("response values = %v, want %v", got, want)
	}
}

func TestManagerSlaveRoutesMultipleUnitIDs(t *testing.T) {
	fake := newFakePort()
	mgr := NewManager(func(cfg port.SerialConfig) (port.Port, error) {
		return fake, nil
	})
	if _, err := mgr.OpenSession(context.Background(), OpenSessionRequest{
		ID:     "slave",
		Mode:   FrameModeRTU,
		Config: port.SerialConfig{PortName: "loop", BaudRate: 115200},
	}); err != nil {
		t.Fatalf("OpenSession() error = %v", err)
	}
	if _, err := mgr.StartSlave(StartSlaveRequest{
		SessionID: "slave",
		Units: []SlaveUnitSnapshot{
			{
				UnitID: 1,
				DataModel: DataModelSnapshot{HoldingRegisters: []RegisterPoint{
					{Address: 0, Value: 24},
				}},
			},
			{
				UnitID: 2,
				DataModel: DataModelSnapshot{HoldingRegisters: []RegisterPoint{
					{Address: 0, Value: 42},
				}},
			},
		},
	}); err != nil {
		t.Fatalf("StartSlave() error = %v", err)
	}
	defer mgr.StopSlave("slave")

	assertSlaveReadValue(t, fake, 1, 24)
	assertSlaveReadValue(t, fake, 2, 42)

	requestFrame, err := EncodeFrame(FrameModeRTU, 3, MustBuildReadRequest(FunctionReadHoldingRegisters, 0, 1))
	if err != nil {
		t.Fatalf("EncodeFrame() error = %v", err)
	}
	fake.pushRead(requestFrame)
	written := fake.waitWrite(t)
	decoded, err := DecodeFrame(FrameModeRTU, written)
	if err != nil {
		t.Fatalf("DecodeFrame(response) error = %v", err)
	}
	parsed, err := ParseResponse(decoded.PDU)
	if err != nil {
		t.Fatalf("ParseResponse() error = %v", err)
	}
	if !parsed.Exception || parsed.ExceptionCode != ExceptionServerFailure {
		t.Fatalf("unknown unit response = %#v, want exception %d", parsed, ExceptionServerFailure)
	}
}

func TestManagerSlaveUnitCRUD(t *testing.T) {
	fake := newFakePort()
	mgr := NewManager(func(cfg port.SerialConfig) (port.Port, error) {
		return fake, nil
	})
	if _, err := mgr.OpenSession(context.Background(), OpenSessionRequest{
		ID:     "slave",
		Mode:   FrameModeRTU,
		Config: port.SerialConfig{PortName: "loop", BaudRate: 115200},
	}); err != nil {
		t.Fatalf("OpenSession() error = %v", err)
	}
	if err := mgr.AddSlaveUnit("slave", SlaveUnitSnapshot{
		UnitID: 7,
		DataModel: DataModelSnapshot{HoldingRegisters: []RegisterPoint{
			{Address: 0, Value: 11},
		}},
	}); err != nil {
		t.Fatalf("AddSlaveUnit() error = %v", err)
	}
	if err := mgr.UpdateSlaveUnitData("slave", 7, DataModelSnapshot{HoldingRegisters: []RegisterPoint{
		{Address: 0, Value: 99},
	}}); err != nil {
		t.Fatalf("UpdateSlaveUnitData() error = %v", err)
	}
	units, err := mgr.ListSlaveUnits("slave")
	if err != nil {
		t.Fatalf("ListSlaveUnits() error = %v", err)
	}
	if len(units) != 1 || units[0].UnitID != 7 || units[0].DataModel.HoldingRegisters[0].Value != 99 {
		t.Fatalf("units = %#v, want updated unit 7", units)
	}
	if err := mgr.RemoveSlaveUnit("slave", 7); err != nil {
		t.Fatalf("RemoveSlaveUnit() error = %v", err)
	}
	units, err = mgr.ListSlaveUnits("slave")
	if err != nil {
		t.Fatalf("ListSlaveUnits() after remove error = %v", err)
	}
	if len(units) != 0 {
		t.Fatalf("units after remove = %#v, want empty", units)
	}
}

func TestManagerScanUnitIDs(t *testing.T) {
	fake := newFakePort()
	mgr := NewManager(func(cfg port.SerialConfig) (port.Port, error) {
		return fake, nil
	})
	if _, err := mgr.OpenSession(context.Background(), OpenSessionRequest{
		ID:        "master",
		Mode:      FrameModeRTU,
		Config:    port.SerialConfig{PortName: "loop", BaudRate: 115200},
		TimeoutMs: 50,
	}); err != nil {
		t.Fatalf("OpenSession() error = %v", err)
	}
	fake.onWrite = func(data []byte) {
		decoded, err := DecodeFrame(FrameModeRTU, data)
		if err != nil {
			t.Errorf("DecodeFrame(request) error = %v", err)
			return
		}
		if decoded.UnitID != 2 && decoded.UnitID != 4 {
			return
		}
		response := BuildSlaveResponse(decoded.UnitID, decoded.PDU, modelWithHolding(uint16(decoded.UnitID)))
		frame, err := EncodeFrame(FrameModeRTU, decoded.UnitID, response.PDU)
		if err != nil {
			t.Errorf("EncodeFrame(response) error = %v", err)
			return
		}
		fake.pushRead(frame)
	}

	result, err := mgr.ScanUnitIDs(UnitScanRequest{
		SessionID: "master",
		UnitIDs:   []int{1, 2, 3, 4},
		Function:  FunctionReadHoldingRegisters,
		Address:   0,
		Quantity:  1,
		TimeoutMs: 40,
	})
	if err != nil {
		t.Fatalf("ScanUnitIDs() error = %v", err)
	}
	if got, want := result.ActiveUnitIDs, []int{2, 4}; !equalInts(got, want) {
		t.Fatalf("active unit IDs = %v, want %v", got, want)
	}
}

func TestManagerReadRegistersDecodesMapping(t *testing.T) {
	fake := newFakePort()
	mgr := NewManager(func(cfg port.SerialConfig) (port.Port, error) {
		return fake, nil
	})
	if _, err := mgr.OpenSession(context.Background(), OpenSessionRequest{
		ID:        "master",
		Mode:      FrameModeRTU,
		Config:    port.SerialConfig{PortName: "loop", BaudRate: 115200},
		TimeoutMs: 50,
	}); err != nil {
		t.Fatalf("OpenSession() error = %v", err)
	}
	fake.onWrite = func(data []byte) {
		decoded, err := DecodeFrame(FrameModeRTU, data)
		if err != nil {
			t.Errorf("DecodeFrame(request) error = %v", err)
			return
		}
		response := BuildSlaveResponse(decoded.UnitID, decoded.PDU, modelWithHolding(0x1234, 0x5678, 0x3fc0, 0x0000))
		frame, err := EncodeFrame(FrameModeRTU, decoded.UnitID, response.PDU)
		if err != nil {
			t.Errorf("EncodeFrame(response) error = %v", err)
			return
		}
		fake.pushRead(frame)
	}

	result, err := mgr.ReadRegisters(RegisterReadRequest{
		SessionID: "master",
		UnitID:    1,
		Function:  FunctionReadHoldingRegisters,
		Address:   0,
		Quantity:  4,
		Mappings: []RegisterMapping{
			{Address: 0, DataType: DataTypeInt32, WordOrder: WordOrderBig},
			{Address: 2, DataType: DataTypeFloat, WordOrder: WordOrderBig},
		},
		TimeoutMs: 40,
	})
	if err != nil {
		t.Fatalf("ReadRegisters() error = %v", err)
	}
	if got, want := result.RawRegisters, []uint16{0x1234, 0x5678, 0x3fc0, 0x0000}; !equalU16(got, want) {
		t.Fatalf("raw registers = %#v, want %#v", got, want)
	}
	if len(result.Values) != 2 {
		t.Fatalf("decoded values = %#v, want 2 values", result.Values)
	}
	if result.Values[0].Value.Display != "305419896" || result.Values[1].Value.Display != "1.5" {
		t.Fatalf("decoded values = %#v", result.Values)
	}
}

func TestManagerScanRegistersReturnsNonZeroValues(t *testing.T) {
	fake := newFakePort()
	mgr := NewManager(func(cfg port.SerialConfig) (port.Port, error) {
		return fake, nil
	})
	if _, err := mgr.OpenSession(context.Background(), OpenSessionRequest{
		ID:        "master",
		Mode:      FrameModeRTU,
		Config:    port.SerialConfig{PortName: "loop", BaudRate: 115200},
		TimeoutMs: 50,
	}); err != nil {
		t.Fatalf("OpenSession() error = %v", err)
	}
	fake.onWrite = func(data []byte) {
		decoded, err := DecodeFrame(FrameModeRTU, data)
		if err != nil {
			t.Errorf("DecodeFrame(request) error = %v", err)
			return
		}
		response := BuildSlaveResponse(decoded.UnitID, decoded.PDU, modelWithHolding(0, 7, 0, 9))
		frame, err := EncodeFrame(FrameModeRTU, decoded.UnitID, response.PDU)
		if err != nil {
			t.Errorf("EncodeFrame(response) error = %v", err)
			return
		}
		fake.pushRead(frame)
	}

	result, err := mgr.ScanRegisters(RegisterScanRequest{
		SessionID:    "master",
		UnitID:       1,
		Function:     FunctionReadHoldingRegisters,
		StartAddress: 0,
		EndAddress:   3,
		ChunkSize:    2,
		TimeoutMs:    40,
	})
	if err != nil {
		t.Fatalf("ScanRegisters() error = %v", err)
	}
	want := []RegisterScanValue{
		{Address: 1, Value: 7},
		{Address: 3, Value: 9},
	}
	if len(result.Values) != len(want) {
		t.Fatalf("scan values = %#v, want %#v", result.Values, want)
	}
	for i := range want {
		if result.Values[i] != want[i] {
			t.Fatalf("scan values[%d] = %#v, want %#v", i, result.Values[i], want[i])
		}
	}
}

func TestManagerCloseSessionClosesPort(t *testing.T) {
	fake := newFakePort()
	mgr := NewManager(func(cfg port.SerialConfig) (port.Port, error) {
		return fake, nil
	})
	if _, err := mgr.OpenSession(context.Background(), OpenSessionRequest{
		ID:     "session",
		Mode:   FrameModeRTU,
		Config: port.SerialConfig{PortName: "loop", BaudRate: 115200},
	}); err != nil {
		t.Fatalf("OpenSession() error = %v", err)
	}
	if err := mgr.CloseSession("session"); err != nil {
		t.Fatalf("CloseSession() error = %v", err)
	}
	if !fake.closed {
		t.Fatalf("fake port was not closed")
	}
}

func TestManagerOpenSessionDefaultsListPortReuseAndCloseAll(t *testing.T) {
	var opened []*fakePort
	var configs []port.SerialConfig
	mgr := NewManager(func(cfg port.SerialConfig) (port.Port, error) {
		configs = append(configs, cfg)
		fake := newFakePort()
		opened = append(opened, fake)
		return fake, nil
	})

	info, err := mgr.OpenSession(context.Background(), OpenSessionRequest{
		Endpoint: "backend-loop",
		Config:   port.SerialConfig{PortName: "public-loop"},
	})
	if err != nil {
		t.Fatalf("OpenSession() error = %v", err)
	}
	if info.ID != "modbus-1" || info.Name != "Modbus public-loop" {
		t.Fatalf("session identity = %#v, want generated id/name", info)
	}
	if info.Mode != FrameModeRTU || info.Role != SessionRoleMaster {
		t.Fatalf("session mode/role = %s/%s, want rtu/master", info.Mode, info.Role)
	}
	if info.Config.PortName != "public-loop" || info.Config.BaudRate != 115200 || info.Config.DataBits != 8 || info.Config.StopBits != "1" || info.Config.Parity != "none" || info.Config.FlowMode != "none" || info.Config.ReadBufKB != 32 {
		t.Fatalf("session config defaults = %#v", info.Config)
	}
	if len(configs) != 1 || configs[0].PortName != "backend-loop" {
		t.Fatalf("open configs = %#v, want endpoint port name", configs)
	}
	if !mgr.PortInUse("public-loop") {
		t.Fatalf("PortInUse(public-loop) = false, want true")
	}
	list := mgr.List()
	if len(list) != 1 || list[0].ID != info.ID {
		t.Fatalf("List() = %#v, want one generated session", list)
	}
	if _, err := mgr.OpenSession(context.Background(), OpenSessionRequest{ID: info.ID, Config: port.SerialConfig{PortName: "other-loop"}}); err == nil {
		t.Fatalf("OpenSession(duplicate id) error = nil, want error")
	}
	if _, err := mgr.OpenSession(context.Background(), OpenSessionRequest{ID: "other", Config: port.SerialConfig{PortName: "public-loop"}}); err == nil {
		t.Fatalf("OpenSession(duplicate port) error = nil, want error")
	}
	if len(opened) != 1 {
		t.Fatalf("opener called %d times, want only initial successful open", len(opened))
	}

	mgr.CloseAll()
	if !opened[0].closed {
		t.Fatalf("CloseAll() did not close the opened port")
	}
	if mgr.PortInUse("public-loop") || len(mgr.List()) != 0 {
		t.Fatalf("manager still reports sessions after CloseAll: inUse=%v list=%#v", mgr.PortInUse("public-loop"), mgr.List())
	}
}

func TestManagerOpenSessionValidationAndOpenFailure(t *testing.T) {
	mgr := NewManager(func(cfg port.SerialConfig) (port.Port, error) {
		return newFakePort(), nil
	})
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := mgr.OpenSession(canceled, OpenSessionRequest{Config: port.SerialConfig{PortName: "loop"}}); err == nil {
		t.Fatalf("OpenSession(canceled context) error = nil, want error")
	}
	if _, err := mgr.OpenSession(context.Background(), OpenSessionRequest{}); err == nil {
		t.Fatalf("OpenSession(empty port) error = nil, want error")
	}

	failing := NewManager(func(cfg port.SerialConfig) (port.Port, error) {
		return nil, io.ErrClosedPipe
	})
	if _, err := failing.OpenSession(context.Background(), OpenSessionRequest{Config: port.SerialConfig{PortName: "loop"}}); err == nil {
		t.Fatalf("OpenSession(open failure) error = nil, want error")
	}
	if err := mgr.CloseSession("missing"); err == nil {
		t.Fatalf("CloseSession(missing) error = nil, want error")
	}
	if _, err := mgr.MasterRequest(MasterRequest{}); err == nil {
		t.Fatalf("MasterRequest(empty session id) error = nil, want error")
	}
}

func TestManagerUpdateSlaveDataAndClosedSessionPaths(t *testing.T) {
	fake := newFakePort()
	mgr := NewManager(func(cfg port.SerialConfig) (port.Port, error) {
		return fake, nil
	})
	if _, err := mgr.OpenSession(context.Background(), OpenSessionRequest{
		ID:     "slave",
		Mode:   FrameModeRTU,
		Config: port.SerialConfig{PortName: "loop", BaudRate: 115200},
	}); err != nil {
		t.Fatalf("OpenSession() error = %v", err)
	}
	if err := mgr.UpdateSlaveData("missing", DataModelSnapshot{}); err == nil {
		t.Fatalf("UpdateSlaveData(missing) error = nil, want error")
	}
	if err := mgr.UpdateSlaveData("slave", DataModelSnapshot{
		Coils:            []BoolPoint{{Address: 0, Value: true}},
		DiscreteInputs:   []BoolPoint{{Address: 1, Value: true}},
		InputRegisters:   []RegisterPoint{{Address: 2, Value: 22}},
		HoldingRegisters: []RegisterPoint{{Address: 3, Value: 33}},
	}); err != nil {
		t.Fatalf("UpdateSlaveData() error = %v", err)
	}
	units, err := mgr.ListSlaveUnits("slave")
	if err != nil {
		t.Fatalf("ListSlaveUnits() error = %v", err)
	}
	if len(units) != 1 || units[0].UnitID != 1 || len(units[0].DataModel.Coils) != 1 || len(units[0].DataModel.InputRegisters) != 1 {
		t.Fatalf("units after UpdateSlaveData = %#v, want default unit with copied data", units)
	}
	if err := mgr.AddSlaveUnit("slave", SlaveUnitSnapshot{}); err == nil {
		t.Fatalf("AddSlaveUnit(invalid unit) error = nil, want error")
	}
	if err := mgr.RemoveSlaveUnit("slave", 0); err == nil {
		t.Fatalf("RemoveSlaveUnit(invalid unit) error = nil, want error")
	}
	if err := mgr.UpdateSlaveUnitData("slave", 0, DataModelSnapshot{}); err == nil {
		t.Fatalf("UpdateSlaveUnitData(invalid unit) error = nil, want error")
	}
	if err := mgr.RemoveSlaveUnit("slave", 1); err != nil {
		t.Fatalf("RemoveSlaveUnit(1) error = %v", err)
	}
	if err := mgr.UpdateSlaveUnitData("slave", 9, DataModelSnapshot{HoldingRegisters: []RegisterPoint{{Address: 0, Value: 9}}}); err != nil {
		t.Fatalf("UpdateSlaveUnitData(9) error = %v", err)
	}

	session, err := mgr.get("slave")
	if err != nil {
		t.Fatalf("get(slave) error = %v", err)
	}
	if err := session.close("forced failure"); err != nil {
		t.Fatalf("session.close() error = %v", err)
	}
	info := session.info()
	if info.Status != SessionStatusError || info.LastError != "forced failure" {
		t.Fatalf("closed session info = %#v, want error status with reason", info)
	}
	if _, err := session.masterRequest(MasterRequest{SessionID: "slave", UnitID: 1, Function: FunctionReadHoldingRegisters, Quantity: 1}); err == nil {
		t.Fatalf("masterRequest(closed) error = nil, want error")
	}
	if _, err := session.write([]byte{1}); err == nil {
		t.Fatalf("write(closed) error = nil, want error")
	}
	if _, _, err := session.readFrame(FrameModeRTU, time.Millisecond, nil); err == nil {
		t.Fatalf("readFrame(closed) error = nil, want error")
	}
}

func TestManagerASCIIRequestRetriesUnexpectedUnit(t *testing.T) {
	fake := newFakePort()
	mgr := NewManager(func(cfg port.SerialConfig) (port.Port, error) {
		return fake, nil
	})
	if _, err := mgr.OpenSession(context.Background(), OpenSessionRequest{
		ID:        "master",
		Mode:      FrameModeASCII,
		Config:    port.SerialConfig{PortName: "loop", BaudRate: 115200},
		TimeoutMs: 50,
	}); err != nil {
		t.Fatalf("OpenSession() error = %v", err)
	}
	var writes int
	fake.onWrite = func(data []byte) {
		writes++
		decoded, err := DecodeFrame(FrameModeASCII, data)
		if err != nil {
			t.Errorf("DecodeFrame(request) error = %v", err)
			return
		}
		unitID := byte(2)
		if writes == 2 {
			unitID = decoded.UnitID
		}
		response := BuildSlaveResponse(unitID, decoded.PDU, modelWithHolding(77))
		frame, err := EncodeFrame(FrameModeASCII, unitID, response.PDU)
		if err != nil {
			t.Errorf("EncodeFrame(response) error = %v", err)
			return
		}
		fake.pushRead(frame)
	}

	tx, err := mgr.MasterRequest(MasterRequest{
		SessionID: "master",
		UnitID:    1,
		Function:  FunctionReadHoldingRegisters,
		Quantity:  1,
		Retries:   1,
	})
	if err != nil {
		t.Fatalf("MasterRequest() error = %v", err)
	}
	if writes != 2 {
		t.Fatalf("writes = %d, want retry after unexpected unit", writes)
	}
	if got, want := tx.Response.Values, []uint16{77}; !equalU16(got, want) {
		t.Fatalf("response values = %v, want %v", got, want)
	}
}

func TestManagerMasterRequestBuildAndWriteErrors(t *testing.T) {
	fake := newFakePort()
	mgr := NewManager(func(cfg port.SerialConfig) (port.Port, error) {
		return fake, nil
	})
	if _, err := mgr.OpenSession(context.Background(), OpenSessionRequest{
		ID:     "master",
		Mode:   FrameModeRTU,
		Config: port.SerialConfig{PortName: "loop", BaudRate: 115200},
	}); err != nil {
		t.Fatalf("OpenSession() error = %v", err)
	}
	if _, err := mgr.MasterRequest(MasterRequest{SessionID: "master", Function: FunctionReadHoldingRegisters}); err == nil {
		t.Fatalf("MasterRequest(unit=0) error = nil, want error")
	}
	tx, err := mgr.MasterRequest(MasterRequest{SessionID: "master", UnitID: 1, Function: FunctionCode(0x2b), Quantity: 1})
	if err == nil || tx == nil || tx.Error == "" {
		t.Fatalf("MasterRequest(unsupported function) tx=%#v err=%v, want transaction error", tx, err)
	}
	if err := fake.Close(); err != nil {
		t.Fatalf("fake.Close() error = %v", err)
	}
	tx, err = mgr.MasterRequest(MasterRequest{SessionID: "master", UnitID: 1, Function: FunctionReadHoldingRegisters, Quantity: 1})
	if err == nil || tx == nil || tx.Error == "" {
		t.Fatalf("MasterRequest(write error) tx=%#v err=%v, want transaction error", tx, err)
	}
}

func TestManagerSlaveStateAndRequestGuards(t *testing.T) {
	fake := newFakePort()
	mgr := NewManager(func(cfg port.SerialConfig) (port.Port, error) {
		return fake, nil
	})
	if _, err := mgr.OpenSession(context.Background(), OpenSessionRequest{
		ID:     "slave",
		Mode:   FrameModeRTU,
		Config: port.SerialConfig{PortName: "loop", BaudRate: 115200},
	}); err != nil {
		t.Fatalf("OpenSession() error = %v", err)
	}
	if _, err := mgr.StartSlave(StartSlaveRequest{SessionID: "slave"}); err == nil {
		t.Fatalf("StartSlave(invalid unit) error = nil, want error")
	}
	if _, err := mgr.StartSlave(StartSlaveRequest{SessionID: "slave", UnitID: 1}); err != nil {
		t.Fatalf("StartSlave() error = %v", err)
	}
	if _, err := mgr.StartSlave(StartSlaveRequest{SessionID: "slave", UnitID: 2}); err == nil {
		t.Fatalf("StartSlave(already running) error = nil, want error")
	}
	if _, err := mgr.MasterRequest(MasterRequest{SessionID: "slave", UnitID: 1, Function: FunctionReadHoldingRegisters, Quantity: 1}); err == nil {
		t.Fatalf("MasterRequest(while slave running) error = nil, want error")
	}
	if err := mgr.StopSlave("slave"); err != nil {
		t.Fatalf("StopSlave() error = %v", err)
	}
	if err := mgr.StopSlave("missing"); err == nil {
		t.Fatalf("StopSlave(missing) error = nil, want error")
	}
	session, err := mgr.get("slave")
	if err != nil {
		t.Fatalf("get(slave) error = %v", err)
	}
	if err := session.close(""); err != nil {
		t.Fatalf("session.close() error = %v", err)
	}
	if _, err := mgr.StartSlave(StartSlaveRequest{SessionID: "slave", UnitID: 1}); err == nil {
		t.Fatalf("StartSlave(closed) error = nil, want error")
	}
}

func TestManagerScanAndReadValidationBranches(t *testing.T) {
	fake := newFakePort()
	mgr := NewManager(func(cfg port.SerialConfig) (port.Port, error) {
		return fake, nil
	})
	if _, err := mgr.OpenSession(context.Background(), OpenSessionRequest{
		ID:        "master",
		Mode:      FrameModeRTU,
		Config:    port.SerialConfig{PortName: "loop", BaudRate: 115200},
		TimeoutMs: 1,
	}); err != nil {
		t.Fatalf("OpenSession() error = %v", err)
	}
	unitScan, err := mgr.ScanUnitIDs(UnitScanRequest{
		SessionID: "master",
		UnitIDs:   []int{0},
	})
	if err != nil {
		t.Fatalf("ScanUnitIDs(invalid unit) error = %v", err)
	}
	if len(unitScan.Results) != 1 || unitScan.Results[0].Error == "" {
		t.Fatalf("ScanUnitIDs(invalid unit) = %#v, want one errored result", unitScan)
	}
	if _, err := mgr.ScanRegisters(RegisterScanRequest{SessionID: "master", UnitID: 1, Function: FunctionReadCoils, StartAddress: 0, EndAddress: 1}); err == nil {
		t.Fatalf("ScanRegisters(invalid function) error = nil, want error")
	}
	if _, err := mgr.ScanRegisters(RegisterScanRequest{SessionID: "master", UnitID: 1, Function: FunctionReadHoldingRegisters, StartAddress: 2, EndAddress: 1}); err == nil {
		t.Fatalf("ScanRegisters(reversed range) error = nil, want error")
	}
}

func TestManagerReadRegistersExceptionAndMappingSkips(t *testing.T) {
	fake := newFakePort()
	mgr := NewManager(func(cfg port.SerialConfig) (port.Port, error) {
		return fake, nil
	})
	if _, err := mgr.OpenSession(context.Background(), OpenSessionRequest{
		ID:        "master",
		Mode:      FrameModeRTU,
		Config:    port.SerialConfig{PortName: "loop", BaudRate: 115200},
		TimeoutMs: 50,
	}); err != nil {
		t.Fatalf("OpenSession() error = %v", err)
	}

	fake.onWrite = func(data []byte) {
		decoded, err := DecodeFrame(FrameModeRTU, data)
		if err != nil {
			t.Errorf("DecodeFrame(request) error = %v", err)
			return
		}
		frame, err := EncodeFrame(FrameModeRTU, decoded.UnitID, PDU{Function: decoded.PDU.Function | exceptionMask, Data: []byte{ExceptionIllegalFunction}})
		if err != nil {
			t.Errorf("EncodeFrame(exception response) error = %v", err)
			return
		}
		fake.pushRead(frame)
	}
	result, err := mgr.ReadRegisters(RegisterReadRequest{SessionID: "master", UnitID: 1, Quantity: 1})
	if err != nil {
		t.Fatalf("ReadRegisters(exception response) error = %v", err)
	}
	if result.Transaction == nil || !result.Transaction.Response.Exception || len(result.RawRegisters) != 0 {
		t.Fatalf("exception read result = %#v, want exception transaction without raw registers", result)
	}

	fake.onWrite = func(data []byte) {
		decoded, err := DecodeFrame(FrameModeRTU, data)
		if err != nil {
			t.Errorf("DecodeFrame(request) error = %v", err)
			return
		}
		response := BuildSlaveResponse(decoded.UnitID, decoded.PDU, modelWithHolding(0x1234))
		frame, err := EncodeFrame(FrameModeRTU, decoded.UnitID, response.PDU)
		if err != nil {
			t.Errorf("EncodeFrame(response) error = %v", err)
			return
		}
		fake.pushRead(frame)
	}
	result, err = mgr.ReadRegisters(RegisterReadRequest{
		SessionID: "master",
		UnitID:    1,
		Address:   2,
		Mappings: []RegisterMapping{
			{Address: 1, DataType: DataTypeUint16},
			{Address: 2, DataType: DataTypeDouble},
			{Address: 9, DataType: DataTypeUint16},
		},
	})
	if err != nil {
		t.Fatalf("ReadRegisters(mapping skips) error = %v", err)
	}
	if len(result.Values) != 1 || result.Values[0].Error == "" {
		t.Fatalf("mapped values = %#v, want one decode error after skips", result.Values)
	}
}

func TestBuildMasterPDUAndSessionHelpers(t *testing.T) {
	tests := []MasterRequest{
		{Function: FunctionWriteSingleCoil, Address: 1, Value: 1},
		{Function: FunctionWriteMultipleCoils, Address: 2, CoilValues: []bool{true, false, true}},
		{Function: FunctionWriteMultipleRegisters, Address: 3, RegisterValues: []uint16{7, 8}},
	}
	for _, req := range tests {
		if _, err := BuildMasterPDU(req); err != nil {
			t.Fatalf("BuildMasterPDU(%#v) error = %v", req, err)
		}
	}
	if _, err := BuildMasterPDU(MasterRequest{AddressMode: AddressModePLC, Function: FunctionReadHoldingRegisters, Address: 30001, Quantity: 1}); err == nil {
		t.Fatalf("BuildMasterPDU(PLC mismatch) error = nil, want error")
	}
	if got := FormatHex(nil); got != "" {
		t.Fatalf("FormatHex(nil) = %q, want empty", got)
	}
	if got := requestRetries(0, 2); got != 2 {
		t.Fatalf("requestRetries(0, 2) = %d, want 2", got)
	}
	if got := requestRetries(3, 2); got != 3 {
		t.Fatalf("requestRetries(3, 2) = %d, want 3", got)
	}
	if got := normalizeSessionRole(SessionRoleSlave); got != SessionRoleSlave {
		t.Fatalf("normalizeSessionRole(slave) = %s, want slave", got)
	}
	if _, err := unitIDByte(248); err == nil {
		t.Fatalf("unitIDByte(248) error = nil, want error")
	}
}

func modelWithHolding(values ...uint16) *DataModel {
	model := NewDataModel()
	_ = model.WriteHoldingRegisters(0, values)
	return model
}

func assertSlaveReadValue(t *testing.T, fake *fakePort, unitID byte, want uint16) {
	t.Helper()
	requestFrame, err := EncodeFrame(FrameModeRTU, unitID, MustBuildReadRequest(FunctionReadHoldingRegisters, 0, 1))
	if err != nil {
		t.Fatalf("EncodeFrame() error = %v", err)
	}
	fake.pushRead(requestFrame)
	written := fake.waitWrite(t)
	decoded, err := DecodeFrame(FrameModeRTU, written)
	if err != nil {
		t.Fatalf("DecodeFrame(response) error = %v", err)
	}
	if decoded.UnitID != unitID {
		t.Fatalf("response unit id = %d, want %d", decoded.UnitID, unitID)
	}
	parsed, err := ParseResponse(decoded.PDU)
	if err != nil {
		t.Fatalf("ParseResponse() error = %v", err)
	}
	if got := parsed.Values; len(got) != 1 || got[0] != want {
		t.Fatalf("response values = %v, want [%d]", got, want)
	}
}

func equalInts(a, b []int) bool {
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

type fakePort struct {
	mu      sync.Mutex
	readCh  chan []byte
	writeCh chan []byte
	timeout time.Duration
	closed  bool
	onWrite func([]byte)
}

func newFakePort() *fakePort {
	return &fakePort{
		readCh:  make(chan []byte, 16),
		writeCh: make(chan []byte, 16),
		timeout: 20 * time.Millisecond,
	}
}

func (p *fakePort) Read(buf []byte) (int, error) {
	p.mu.Lock()
	timeout := p.timeout
	closed := p.closed
	p.mu.Unlock()
	if closed {
		return 0, io.ErrClosedPipe
	}
	select {
	case data := <-p.readCh:
		return copy(buf, data), nil
	case <-time.After(timeout):
		return 0, nil
	}
}

func (p *fakePort) Write(data []byte) (int, error) {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return 0, io.ErrClosedPipe
	}
	copied := append([]byte(nil), data...)
	onWrite := p.onWrite
	p.mu.Unlock()

	p.writeCh <- copied
	if onWrite != nil {
		onWrite(copied)
	}
	return len(data), nil
}

func (p *fakePort) SetReadTimeout(timeout time.Duration) error {
	p.mu.Lock()
	p.timeout = timeout
	p.mu.Unlock()
	return nil
}

func (p *fakePort) Close() error {
	p.mu.Lock()
	p.closed = true
	p.mu.Unlock()
	return nil
}

func (p *fakePort) pushRead(data []byte) {
	p.readCh <- append([]byte(nil), data...)
}

func (p *fakePort) waitWrite(t *testing.T) []byte {
	t.Helper()
	select {
	case data := <-p.writeCh:
		return data
	case <-time.After(time.Second):
		t.Fatalf("timeout waiting for fake port write")
		return nil
	}
}
