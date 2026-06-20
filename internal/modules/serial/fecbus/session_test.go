package fecbus

import (
	"context"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/suyue/mocktrue/internal/modules/serial/port"
)

func TestManagerSendRequestRecordsResponse(t *testing.T) {
	fake := newFakePort()
	mgr := NewManager(func(cfg port.SerialConfig) (port.Port, error) {
		return fake, nil
	})
	if _, err := mgr.OpenSession(context.Background(), OpenSessionRequest{
		ID:        "master",
		Role:      SessionRoleMaster,
		Config:    port.SerialConfig{PortName: "loop", BaudRate: 9600},
		TimeoutMs: 50,
	}); err != nil {
		t.Fatalf("OpenSession() error = %v", err)
	}

	fake.onWrite = func(data []byte) {
		request, err := DecodeFrame(data)
		if err != nil {
			t.Errorf("DecodeFrame(request) error = %v", err)
			return
		}
		answer := Frame{
			Type:          FrameTypeAnswer,
			TargetAddress: request.SourceAddress,
			Priority:      3,
			SourceAddress: request.TargetAddress,
			MessageNumber: request.MessageNumber,
			GroupNumber:   request.GroupNumber,
			Data:          []byte{byte(request.Function()), 0x42},
		}
		frame, err := EncodeFrame(answer)
		if err != nil {
			t.Errorf("EncodeFrame(answer) error = %v", err)
			return
		}
		fake.pushRead(frame)
	}

	tx, err := mgr.SendRequest(SendRequest{
		SessionID:     "master",
		TargetAddress: 2,
		SourceAddress: 1,
		MessageNumber: 7,
		Function:      FunctionQueryProtocolVersion,
		ExpectAnswer:  true,
		TimeoutMs:     40,
	})
	if err != nil {
		t.Fatalf("SendRequest() error = %v", err)
	}
	if tx.Response == nil {
		t.Fatalf("response is nil")
	}
	if tx.Response.SourceAddress != 2 || tx.Response.TargetAddress != 1 {
		t.Fatalf("response addresses = %d/%d, want 2/1", tx.Response.SourceAddress, tx.Response.TargetAddress)
	}
	if tx.Response.Function() != FunctionQueryProtocolVersion {
		t.Fatalf("response function = %d, want %d", tx.Response.Function(), FunctionQueryProtocolVersion)
	}

	page, err := mgr.QueryFrames(QueryRequest{SessionID: "master", Limit: 10})
	if err != nil {
		t.Fatalf("QueryFrames() error = %v", err)
	}
	if page.Total != 2 || len(page.Frames) != 2 {
		t.Fatalf("frame page = total %d len %d, want 2/2", page.Total, len(page.Frames))
	}
	if page.Frames[0].Direction != "tx" || page.Frames[1].Direction != "rx" {
		t.Fatalf("directions = %s/%s, want tx/rx", page.Frames[0].Direction, page.Frames[1].Direction)
	}
}

func TestManagerSendRequestRetriesOnTimeout(t *testing.T) {
	fake := newFakePort()
	mgr := NewManager(func(cfg port.SerialConfig) (port.Port, error) {
		return fake, nil
	})
	if _, err := mgr.OpenSession(context.Background(), OpenSessionRequest{
		ID:        "master",
		Config:    port.SerialConfig{PortName: "loop", BaudRate: 9600},
		TimeoutMs: 20,
		Retries:   1,
	}); err != nil {
		t.Fatalf("OpenSession() error = %v", err)
	}
	fake.timeout = 5 * time.Millisecond

	_, err := mgr.SendRequest(SendRequest{
		SessionID:     "master",
		TargetAddress: 2,
		SourceAddress: 1,
		MessageNumber: 1,
		Function:      FunctionQueryDeviceStatus,
		ExpectAnswer:  true,
		TimeoutMs:     10,
	})
	if err == nil || !strings.Contains(err.Error(), "timeout") {
		t.Fatalf("SendRequest() error = %v, want timeout", err)
	}
	if len(fake.writes()) != 2 {
		t.Fatalf("writes = %d, want retry + original = 2", len(fake.writes()))
	}
}

func TestManagerSlaveRespondsWithStatusAnswer(t *testing.T) {
	fake := newFakePort()
	mgr := NewManager(func(cfg port.SerialConfig) (port.Port, error) {
		return fake, nil
	})
	if _, err := mgr.OpenSession(context.Background(), OpenSessionRequest{
		ID:     "slave",
		Role:   SessionRoleSlave,
		Config: port.SerialConfig{PortName: "loop", BaudRate: 9600},
	}); err != nil {
		t.Fatalf("OpenSession() error = %v", err)
	}
	if _, err := mgr.StartSlave(StartSlaveRequest{
		SessionID: "slave",
		State: SlaveState{
			Address:          2,
			DefaultStatus:    StatusReceivedOK,
			AutoStatusAnswer: true,
		},
	}); err != nil {
		t.Fatalf("StartSlave() error = %v", err)
	}
	defer mgr.StopSlave("slave")

	request, err := EncodeFrame(Frame{
		Type:          FrameTypeRequest,
		TargetAddress: 2,
		Priority:      2,
		SourceAddress: 1,
		MessageNumber: 5,
		GroupNumber:   0,
		Data:          []byte{byte(FunctionQueryDeviceStatus)},
	})
	if err != nil {
		t.Fatalf("EncodeFrame() error = %v", err)
	}
	fake.pushRead(request)
	written := fake.waitWrite(t)
	answer, err := DecodeFrame(written)
	if err != nil {
		t.Fatalf("DecodeFrame(answer) error = %v", err)
	}
	if answer.Type != FrameTypeAnswer || answer.Function() != FunctionStatusAnswer {
		t.Fatalf("answer = %#v, want status answer", answer)
	}
	if got := answer.Data; len(got) != 2 || StatusCode(got[1]) != StatusReceivedOK {
		t.Fatalf("answer data = %x, want status received ok", got)
	}
}

func TestManagerSlaveRoutesRequestsToMultipleUnits(t *testing.T) {
	fake := newFakePort()
	mgr := NewManager(func(cfg port.SerialConfig) (port.Port, error) {
		return fake, nil
	})
	if _, err := mgr.OpenSession(context.Background(), OpenSessionRequest{
		ID:     "slave",
		Role:   SessionRoleSlave,
		Config: port.SerialConfig{PortName: "loop", BaudRate: 9600},
	}); err != nil {
		t.Fatalf("OpenSession() error = %v", err)
	}
	if _, err := mgr.StartSlave(StartSlaveRequest{
		SessionID: "slave",
		Units: []SlaveUnitState{
			{Address: 2, DefaultStatus: StatusReceivedOK, AutoStatusAnswer: true},
			{Address: 3, DefaultStatus: StatusBusy, AutoStatusAnswer: true},
		},
	}); err != nil {
		t.Fatalf("StartSlave() error = %v", err)
	}
	defer mgr.StopSlave("slave")

	pushRequest := func(target byte, message byte) Frame {
		t.Helper()
		request, err := EncodeFrame(Frame{
			Type:          FrameTypeRequest,
			TargetAddress: target,
			Priority:      2,
			SourceAddress: 1,
			MessageNumber: message,
			GroupNumber:   0,
			Data:          []byte{byte(FunctionQueryDeviceStatus)},
		})
		if err != nil {
			t.Fatalf("EncodeFrame() error = %v", err)
		}
		fake.pushRead(request)
		written := fake.waitWrite(t)
		answer, err := DecodeFrame(written)
		if err != nil {
			t.Fatalf("DecodeFrame(answer) error = %v", err)
		}
		return answer
	}

	answer := pushRequest(3, 5)
	if answer.SourceAddress != 3 || StatusCode(answer.Data[1]) != StatusBusy {
		t.Fatalf("answer for address 3 = source %d data %x, want source 3 busy", answer.SourceAddress, answer.Data)
	}

	answer = pushRequest(2, 6)
	if answer.SourceAddress != 2 || StatusCode(answer.Data[1]) != StatusReceivedOK {
		t.Fatalf("answer for address 2 = source %d data %x, want source 2 received ok", answer.SourceAddress, answer.Data)
	}

	request, err := EncodeFrame(Frame{
		Type:          FrameTypeRequest,
		TargetAddress: 4,
		Priority:      2,
		SourceAddress: 1,
		MessageNumber: 7,
		GroupNumber:   0,
		Data:          []byte{byte(FunctionQueryDeviceStatus)},
	})
	if err != nil {
		t.Fatalf("EncodeFrame(unmatched) error = %v", err)
	}
	fake.pushRead(request)
	fake.expectNoWrite(t)
}

func TestManagerCloseSessionClosesPort(t *testing.T) {
	fake := newFakePort()
	mgr := NewManager(func(cfg port.SerialConfig) (port.Port, error) {
		return fake, nil
	})
	if _, err := mgr.OpenSession(context.Background(), OpenSessionRequest{
		ID:     "session",
		Config: port.SerialConfig{PortName: "loop", BaudRate: 9600},
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

func (p *fakePort) expectNoWrite(t *testing.T) {
	t.Helper()
	select {
	case data := <-p.writeCh:
		t.Fatalf("unexpected fake port write: %x", data)
	case <-time.After(80 * time.Millisecond):
	}
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

func (p *fakePort) writes() [][]byte {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([][]byte, 0, len(p.writeCh))
	for {
		select {
		case data := <-p.writeCh:
			out = append(out, data)
		default:
			return out
		}
	}
}
