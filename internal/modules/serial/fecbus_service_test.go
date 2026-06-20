package serial

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/suyue/mocktrue/internal/core/errors"
	"github.com/suyue/mocktrue/internal/core/eventbus"
	fb "github.com/suyue/mocktrue/internal/modules/serial/fecbus"
	"github.com/suyue/mocktrue/internal/modules/serial/manager"
	mb "github.com/suyue/mocktrue/internal/modules/serial/modbus"
	"github.com/suyue/mocktrue/internal/modules/serial/port"
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
