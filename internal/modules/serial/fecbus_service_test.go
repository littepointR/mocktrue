package serial

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/littepointR/portweave/internal/core/errors"
	"github.com/littepointR/portweave/internal/core/eventbus"
	fb "github.com/littepointR/portweave/internal/modules/serial/fecbus"
	"github.com/littepointR/portweave/internal/modules/serial/manager"
	mb "github.com/littepointR/portweave/internal/modules/serial/modbus"
	"github.com/littepointR/portweave/internal/modules/serial/port"
	"github.com/littepointR/portweave/internal/modules/serial/porttest"
)

func TestServiceFecbusConflictsAndCleanup(t *testing.T) {
	fake := &serviceFakePort{timeout: time.Millisecond}
	svc := NewService(eventbus.New())
	svc.fecbus = fb.NewManager(func(cfg port.SerialConfig) (port.Port, error) {
		return fake, nil
	})

	if _, err := svc.OpenFecbusSession(context.Background(), fb.OpenSessionRequest{
		ID:     "fec",
		Config: port.SerialConfig{PortName: "loop", BaudRate: 9600},
	}); err != nil {
		t.Fatalf("OpenFecbusSession() error = %v", err)
	}

	if _, err := svc.OpenPort(context.Background(), manager.OpenRequest{Config: port.SerialConfig{PortName: "loop", BaudRate: 9600}}); errors.AsCode(err) != errors.CodeConflict {
		t.Fatalf("OpenPort conflict error = %v, want conflict", err)
	}
	if _, err := svc.OpenModbusSession(context.Background(), mb.OpenSessionRequest{Config: port.SerialConfig{PortName: "loop", BaudRate: 9600}}); errors.AsCode(err) != errors.CodeConflict {
		t.Fatalf("OpenModbusSession conflict error = %v, want conflict", err)
	}
	if _, err := svc.CreateBridge("bridge", "loop", "other", 9600); errors.AsCode(err) != errors.CodeConflict {
		t.Fatalf("CreateBridge conflict error = %v, want conflict", err)
	}

	svc.cleanup()
	if !fake.closed {
		t.Fatalf("cleanup did not close FECbus port")
	}
	if got := svc.ListFecbusSessions(); len(got) != 0 {
		t.Fatalf("ListFecbusSessions after cleanup = %#v, want empty", got)
	}
}

func TestServiceFecbusWrapperLifecycle(t *testing.T) {
	fake := &serviceFakePort{timeout: time.Millisecond}
	svc := NewService(eventbus.New())
	svc.fecbus = fb.NewManager(func(cfg port.SerialConfig) (port.Port, error) {
		return fake, nil
	})

	info, err := svc.OpenFecbusSession(context.Background(), fb.OpenSessionRequest{
		ID:     "fec",
		Config: port.SerialConfig{PortName: "loop", BaudRate: 9600},
	})
	if err != nil {
		t.Fatalf("OpenFecbusSession() error = %v", err)
	}
	if info.ID != "fec" || len(svc.ListFecbusSessions()) != 1 {
		t.Fatalf("Open/ListFecbusSessions = info %#v list %#v, want one fec session", info, svc.ListFecbusSessions())
	}

	tx, err := svc.FecbusSendRequest(fb.SendRequest{
		SessionID:     "fec",
		TargetAddress: 2,
		SourceAddress: 1,
		MessageNumber: 3,
		Function:      fb.FunctionQueryDeviceStatus,
		ExpectAnswer:  false,
	})
	if err != nil {
		t.Fatalf("FecbusSendRequest(no answer) error = %v", err)
	}
	if tx.BytesWritten == 0 {
		t.Fatalf("FecbusSendRequest(no answer) bytes written = 0, want positive")
	}

	if _, err := svc.StartFecbusSlave(fb.StartSlaveRequest{SessionID: "fec"}); err != nil {
		t.Fatalf("StartFecbusSlave() error = %v", err)
	}
	if err := svc.UpdateFecbusSlaveState("fec", fb.SlaveState{Address: 5, DefaultStatus: fb.StatusBusy}); err != nil {
		t.Fatalf("UpdateFecbusSlaveState() error = %v", err)
	}
	if err := svc.AddFecbusSlaveUnit("fec", fb.SlaveUnitState{Address: 4, DefaultStatus: fb.StatusUnitFault}); err != nil {
		t.Fatalf("AddFecbusSlaveUnit() error = %v", err)
	}
	units, err := svc.ListFecbusSlaveUnits("fec")
	if err != nil {
		t.Fatalf("ListFecbusSlaveUnits() error = %v", err)
	}
	if len(units) < 2 {
		t.Fatalf("ListFecbusSlaveUnits() = %#v, want updated and added units", units)
	}
	if err := svc.RemoveFecbusSlaveUnit("fec", 4); err != nil {
		t.Fatalf("RemoveFecbusSlaveUnit() error = %v", err)
	}
	page, err := svc.QueryFecbusFrames(fb.QueryRequest{SessionID: "fec", Limit: 10})
	if err != nil {
		t.Fatalf("QueryFecbusFrames() error = %v", err)
	}
	if page.Total == 0 {
		t.Fatalf("QueryFecbusFrames() total = 0, want request frame history")
	}
	if err := svc.ClearFecbusFrames("fec"); err != nil {
		t.Fatalf("ClearFecbusFrames() error = %v", err)
	}
	if err := svc.StopFecbusSlave("fec"); err != nil {
		t.Fatalf("StopFecbusSlave() error = %v", err)
	}
	if err := svc.CloseFecbusSession("fec"); err != nil {
		t.Fatalf("CloseFecbusSession() error = %v", err)
	}
	if got := svc.ListFecbusSessions(); len(got) != 0 {
		t.Fatalf("ListFecbusSessions after close = %#v, want empty", got)
	}
}

func TestServiceOpenFecbusSessionRejectsTerminalAndModbusConflicts(t *testing.T) {
	svc, _ := newMemoryServiceWithPair(t, "fecA", "fecB")
	handle, err := svc.OpenPort(context.Background(), manager.OpenRequest{Config: porttest.DefaultSerialConfig("fecA")})
	if err != nil {
		t.Fatalf("OpenPort() error = %v", err)
	}
	if _, err := svc.OpenFecbusSession(context.Background(), fb.OpenSessionRequest{Config: port.SerialConfig{PortName: "fecA", BaudRate: 9600}}); errors.AsCode(err) != errors.CodeConflict {
		t.Fatalf("OpenFecbusSession(serial conflict) error = %v, want conflict", err)
	}
	if err := svc.ClosePort(handle.ID); err != nil {
		t.Fatalf("ClosePort() error = %v", err)
	}

	fake := &serviceFakePort{timeout: time.Millisecond}
	svc.modbus = mb.NewManager(func(cfg port.SerialConfig) (port.Port, error) {
		return fake, nil
	})
	if _, err := svc.OpenModbusSession(context.Background(), mb.OpenSessionRequest{ID: "mod", Config: port.SerialConfig{PortName: "shared", BaudRate: 9600}}); err != nil {
		t.Fatalf("OpenModbusSession() error = %v", err)
	}
	defer svc.modbus.CloseAll()
	if _, err := svc.OpenFecbusSession(context.Background(), fb.OpenSessionRequest{Config: port.SerialConfig{PortName: "shared", BaudRate: 9600}}); errors.AsCode(err) != errors.CodeConflict {
		t.Fatalf("OpenFecbusSession(modbus conflict) error = %v, want conflict", err)
	}
}

type serviceFakePort struct {
	timeout time.Duration
	closed  bool
}

func (p *serviceFakePort) Read([]byte) (int, error) {
	if p.closed {
		return 0, io.ErrClosedPipe
	}
	time.Sleep(p.timeout)
	return 0, nil
}

func (p *serviceFakePort) Write(data []byte) (int, error) {
	if p.closed {
		return 0, io.ErrClosedPipe
	}
	return len(data), nil
}

func (p *serviceFakePort) SetReadTimeout(timeout time.Duration) error {
	p.timeout = timeout
	return nil
}

func (p *serviceFakePort) Close() error {
	p.closed = true
	return nil
}
