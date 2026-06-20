package modbus

import (
	"context"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/suyue/mocktrue/internal/modules/serial/port"
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
