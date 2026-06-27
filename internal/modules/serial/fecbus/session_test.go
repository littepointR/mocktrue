package fecbus

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/littepointR/portweave/internal/modules/serial/port"
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

func TestManagerOpenSessionRejectsInvalidAndConflictingRequests(t *testing.T) {
	mgr := NewManager(func(cfg port.SerialConfig) (port.Port, error) {
		if cfg.PortName == "boom" {
			return nil, io.ErrClosedPipe
		}
		return newFakePort(), nil
	})

	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := mgr.OpenSession(canceled, OpenSessionRequest{Config: port.SerialConfig{PortName: "loop"}}); err == nil {
		t.Fatalf("OpenSession() with canceled context error = nil, want error")
	}
	if _, err := mgr.OpenSession(context.Background(), OpenSessionRequest{}); err == nil || !strings.Contains(err.Error(), "port name") {
		t.Fatalf("OpenSession() empty port error = %v, want port name error", err)
	}
	if _, err := mgr.OpenSession(context.Background(), OpenSessionRequest{Config: port.SerialConfig{PortName: "boom"}}); err == nil || !strings.Contains(err.Error(), "open fecbus port") {
		t.Fatalf("OpenSession() opener error = %v, want wrapped open error", err)
	}

	info, err := mgr.OpenSession(context.Background(), OpenSessionRequest{
		ID:     "one",
		Config: port.SerialConfig{PortName: "loop"},
	})
	if err != nil {
		t.Fatalf("OpenSession() valid error = %v", err)
	}
	if info.Name != "FECbus loop" || info.Role != SessionRoleMaster {
		t.Fatalf("OpenSession() defaults name/role = %q/%q, want FECbus loop/master", info.Name, info.Role)
	}
	if info.Config.BaudRate != 9600 || info.Config.DataBits != 8 || info.Config.StopBits != "1" || info.Config.Parity != "none" || info.Config.FlowMode != "none" || info.Config.ReadBufKB != 32 {
		t.Fatalf("OpenSession() default serial config = %+v, want populated defaults", info.Config)
	}
	if _, err := mgr.OpenSession(context.Background(), OpenSessionRequest{ID: "one", Config: port.SerialConfig{PortName: "other"}}); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("OpenSession() duplicate ID error = %v, want already exists", err)
	}
	if _, err := mgr.OpenSession(context.Background(), OpenSessionRequest{ID: "two", Config: port.SerialConfig{PortName: "loop"}}); err == nil || !strings.Contains(err.Error(), "port already in use") {
		t.Fatalf("OpenSession() duplicate port error = %v, want port already in use", err)
	}
}

func TestManagerSlaveUnitFrameHistoryAndClearAPIs(t *testing.T) {
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

	if err := mgr.UpdateSlaveState("session", SlaveState{Address: 5, DefaultStatus: StatusBusy, AcceptBroadcast: true}); err != nil {
		t.Fatalf("UpdateSlaveState() error = %v", err)
	}
	if err := mgr.AddSlaveUnit("session", SlaveUnitState{
		Address:       3,
		DefaultStatus: StatusUnitFault,
		AnswerPayloadByFunction: map[FunctionCode][]byte{
			FunctionQueryProtocolVersion: {0x01, 0x02},
		},
	}); err != nil {
		t.Fatalf("AddSlaveUnit() error = %v", err)
	}
	units, err := mgr.ListSlaveUnits("session")
	if err != nil {
		t.Fatalf("ListSlaveUnits() error = %v", err)
	}
	if got := unitAddresses(units); got != "2,3,5" {
		t.Fatalf("ListSlaveUnits() addresses = %s, want 2,3,5", got)
	}
	if err := mgr.RemoveSlaveUnit("session", 0); err == nil || !strings.Contains(err.Error(), "1-63") {
		t.Fatalf("RemoveSlaveUnit(0) error = %v, want range error", err)
	}
	if err := mgr.RemoveSlaveUnit("session", 9); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("RemoveSlaveUnit(missing) error = %v, want not found", err)
	}
	if err := mgr.RemoveSlaveUnit("session", 2); err != nil {
		t.Fatalf("RemoveSlaveUnit(existing) error = %v", err)
	}

	tx, err := mgr.SendRequest(SendRequest{
		SessionID:    "session",
		FrameType:    FrameTypeAnswer,
		DataHex:      "2c 01 02",
		ExpectAnswer: false,
	})
	if err != nil {
		t.Fatalf("SendRequest(no answer) error = %v", err)
	}
	if tx.Response != nil || tx.Error != "" {
		t.Fatalf("SendRequest(no answer) response/error = %#v/%q, want nil/empty", tx.Response, tx.Error)
	}
	if tx.Request.Type != FrameTypeAnswer || tx.Request.TargetAddress != 1 || tx.Request.SourceAddress != 1 || tx.Request.MessageNumber != 1 {
		t.Fatalf("request defaults = %#v, want answer frame with 1/1 addresses and message 1", tx.Request)
	}
	assertHex(t, tx.Request.Data, "2c0102")

	page, err := mgr.QueryFrames(QueryRequest{SessionID: "session", Direction: "rx", Limit: 10})
	if err != nil {
		t.Fatalf("QueryFrames(rx) error = %v", err)
	}
	if page.Total != 0 || !page.EOF {
		t.Fatalf("QueryFrames(rx) = total %d eof %v, want empty EOF", page.Total, page.EOF)
	}
	page, err = mgr.QueryFrames(QueryRequest{SessionID: "session", Direction: "tx", Search: "2c", Limit: -1})
	if err != nil {
		t.Fatalf("QueryFrames(search tx) error = %v", err)
	}
	if page.Total != 1 || len(page.Frames) != 1 || page.Limit != 200 || !page.EOF {
		t.Fatalf("QueryFrames(search tx) = total %d len %d limit %d eof %v, want one default-limited EOF page", page.Total, len(page.Frames), page.Limit, page.EOF)
	}
	page, err = mgr.QueryFrames(QueryRequest{SessionID: "session", Offset: 10, Limit: 1})
	if err != nil {
		t.Fatalf("QueryFrames(offset) error = %v", err)
	}
	if page.Total != 1 || len(page.Frames) != 0 || !page.EOF {
		t.Fatalf("QueryFrames(offset) = total %d len %d eof %v, want empty EOF beyond end", page.Total, len(page.Frames), page.EOF)
	}
	if err := mgr.ClearFrames("session"); err != nil {
		t.Fatalf("ClearFrames() error = %v", err)
	}
	page, err = mgr.QueryFrames(QueryRequest{SessionID: "session", Limit: 10})
	if err != nil {
		t.Fatalf("QueryFrames(after clear) error = %v", err)
	}
	if page.Total != 0 {
		t.Fatalf("QueryFrames(after clear) total = %d, want 0", page.Total)
	}
	info := mgr.List()[0]
	if info.RxBytes != 0 || info.TxBytes != 0 {
		t.Fatalf("counters after ClearFrames = rx %d tx %d, want 0/0", info.RxBytes, info.TxBytes)
	}
}

func TestManagerSendRequestRejectsInvalidFramesAndRunningSlave(t *testing.T) {
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

	tx, err := mgr.SendRequest(SendRequest{SessionID: "session", DataHex: "zz"})
	if err == nil || tx == nil || !strings.Contains(tx.Error, "parse data hex") {
		t.Fatalf("SendRequest(invalid hex) tx/error = %#v/%v, want parse data hex", tx, err)
	}
	tx, err = mgr.SendRequest(SendRequest{SessionID: "session", Function: FunctionCode(0x10)})
	if err == nil || tx == nil || !strings.Contains(tx.Error, "reserved") {
		t.Fatalf("SendRequest(reserved function) tx/error = %#v/%v, want reserved function", tx, err)
	}

	if _, err := mgr.StartSlave(StartSlaveRequest{SessionID: "session"}); err != nil {
		t.Fatalf("StartSlave() error = %v", err)
	}
	defer func() { _ = mgr.StopSlave("session") }()
	if _, err := mgr.SendRequest(SendRequest{SessionID: "session", Function: FunctionQueryDeviceStatus}); err == nil || !strings.Contains(err.Error(), "slave simulation") {
		t.Fatalf("SendRequest(while slave running) error = %v, want slave simulation error", err)
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
	defer func() { _ = mgr.StopSlave("slave") }()

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
	defer func() { _ = mgr.StopSlave("slave") }()

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

func TestManagerSlaveBroadcastsToMatchingUnitsAndCustomPayload(t *testing.T) {
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
			{
				Address:         2,
				AcceptBroadcast: true,
				AnswerPayloadByFunction: map[FunctionCode][]byte{
					FunctionQueryProtocolVersion: {0x01, 0x02},
				},
			},
			{Address: 3, DefaultStatus: StatusBusy, AutoStatusAnswer: true, AcceptBroadcast: true},
			{Address: 4, DefaultStatus: StatusUnitFault, AutoStatusAnswer: true},
		},
	}); err != nil {
		t.Fatalf("StartSlave() error = %v", err)
	}
	defer func() { _ = mgr.StopSlave("slave") }()

	request, err := EncodeFrame(Frame{
		Type:          FrameTypeRequest,
		TargetAddress: 0,
		Priority:      2,
		SourceAddress: 1,
		MessageNumber: 8,
		Data:          []byte{byte(FunctionQueryProtocolVersion)},
	})
	if err != nil {
		t.Fatalf("EncodeFrame() error = %v", err)
	}
	fake.pushRead(request)

	first, err := DecodeFrame(fake.waitWrite(t))
	if err != nil {
		t.Fatalf("DecodeFrame(first answer) error = %v", err)
	}
	second, err := DecodeFrame(fake.waitWrite(t))
	if err != nil {
		t.Fatalf("DecodeFrame(second answer) error = %v", err)
	}
	if first.SourceAddress != 2 || first.Function() != FunctionQueryProtocolVersion {
		t.Fatalf("first broadcast answer = source %d function %d, want unit 2 custom protocol version", first.SourceAddress, first.Function())
	}
	assertHex(t, first.Data, "2c0102")
	if second.SourceAddress != 3 || second.Function() != FunctionStatusAnswer || StatusCode(second.Data[1]) != StatusBusy {
		t.Fatalf("second broadcast answer = source %d data %x, want unit 3 busy status", second.SourceAddress, second.Data)
	}
	fake.expectNoWrite(t)
}

func TestManagerSlaveRecordsReadError(t *testing.T) {
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
	if _, err := mgr.StartSlave(StartSlaveRequest{SessionID: "slave"}); err != nil {
		t.Fatalf("StartSlave() error = %v", err)
	}
	if err := fake.Close(); err != nil {
		t.Fatalf("fake.Close() error = %v", err)
	}

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		info := mgr.List()[0]
		if info.Status == SessionStatusError {
			if info.LastError == "" {
				t.Fatalf("LastError is empty after slave read failure")
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("slave status did not become error after fake port close; last info = %#v", mgr.List()[0])
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

func TestSessionReadFrameSkipsMalformedInputAndHonorsStop(t *testing.T) {
	fake := newFakePort()
	session := newSession(OpenSessionRequest{ID: "session", Config: port.SerialConfig{PortName: "loop"}}, fake)
	valid, err := EncodeFrame(Frame{
		Type:          FrameTypeRequest,
		TargetAddress: 2,
		Priority:      1,
		SourceAddress: 1,
		MessageNumber: 4,
		Data:          []byte{byte(FunctionQueryDeviceStatus)},
	})
	if err != nil {
		t.Fatalf("EncodeFrame(valid) error = %v", err)
	}
	fake.pushRead([]byte{0x01, 0x02})
	fake.pushRead(corrupt(valid, 3))
	fake.pushRead(valid)

	raw, frame, err := session.readFrame(200*time.Millisecond, nil)
	if err != nil {
		t.Fatalf("readFrame() error = %v", err)
	}
	if string(raw) != string(valid) || frame.Function() != FunctionQueryDeviceStatus {
		t.Fatalf("readFrame() raw/function = %x/%d, want valid query status", raw, frame.Function())
	}
	if session.info().RxBytes == 0 {
		t.Fatalf("readFrame() did not increment rx bytes")
	}

	stop := make(chan struct{})
	close(stop)
	if _, _, err := session.readFrame(time.Second, stop); err == nil || !strings.Contains(err.Error(), "stopped") {
		t.Fatalf("readFrame(stopped) error = %v, want stopped", err)
	}
	if err := session.close("closed for test"); err != nil {
		t.Fatalf("session.close(reason) error = %v", err)
	}
	info := session.info()
	if info.Status != SessionStatusError || info.LastError != "closed for test" {
		t.Fatalf("session info after close(reason) = status %q error %q, want error/closed for test", info.Status, info.LastError)
	}
	if _, _, err := session.readFrame(50*time.Millisecond, nil); err == nil || !strings.Contains(err.Error(), "closed") {
		t.Fatalf("readFrame(closed) error = %v, want closed", err)
	}
}

func TestSessionInternalEdgeHelpers(t *testing.T) {
	var raceIDMgr *Manager
	raceIDPort := newFakePort()
	raceIDMgr = NewManager(func(cfg port.SerialConfig) (port.Port, error) {
		raceIDMgr.mu.Lock()
		raceIDMgr.sessions["race"] = newSession(OpenSessionRequest{ID: "race", Config: port.SerialConfig{PortName: "other"}}, newFakePort())
		raceIDMgr.mu.Unlock()
		return raceIDPort, nil
	})
	if _, err := raceIDMgr.OpenSession(context.Background(), OpenSessionRequest{ID: "race", Config: port.SerialConfig{PortName: "race-id"}}); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("OpenSession(racing ID) error = %v, want already exists", err)
	}
	if !raceIDPort.closed {
		t.Fatalf("OpenSession(racing ID) did not close opened port")
	}

	var racePortMgr *Manager
	racePort := newFakePort()
	racePortMgr = NewManager(func(cfg port.SerialConfig) (port.Port, error) {
		racePortMgr.mu.Lock()
		racePortMgr.sessions["other"] = newSession(OpenSessionRequest{ID: "other", Config: port.SerialConfig{PortName: "race-port"}}, newFakePort())
		racePortMgr.mu.Unlock()
		return racePort, nil
	})
	if _, err := racePortMgr.OpenSession(context.Background(), OpenSessionRequest{ID: "race", Config: port.SerialConfig{PortName: "race-port"}}); err == nil || !strings.Contains(err.Error(), "port already in use") {
		t.Fatalf("OpenSession(racing port) error = %v, want port already in use", err)
	}
	if !racePort.closed {
		t.Fatalf("OpenSession(racing port) did not close opened port")
	}

	session := newSession(OpenSessionRequest{ID: "direct", Config: port.SerialConfig{PortName: "loop"}}, newFakePort())
	if err := session.startSlave(StartSlaveRequest{}); err != nil {
		t.Fatalf("startSlave() error = %v", err)
	}
	if err := session.startSlave(StartSlaveRequest{}); err == nil || !strings.Contains(err.Error(), "already running") {
		t.Fatalf("startSlave(already running) error = %v, want already running", err)
	}
	if err := session.stopSlave(); err != nil {
		t.Fatalf("stopSlave() error = %v", err)
	}
	if err := session.close(""); err != nil {
		t.Fatalf("close() error = %v", err)
	}
	if err := session.startSlave(StartSlaveRequest{}); err == nil || !strings.Contains(err.Error(), "closed") {
		t.Fatalf("startSlave(closed) error = %v, want closed", err)
	}
	if _, err := session.write([]byte{0x01}); err == nil || !strings.Contains(err.Error(), "closed") {
		t.Fatalf("write(closed) error = %v, want closed", err)
	}

	badPayload := SlaveUnitState{Address: 2, AnswerPayloadByFunction: map[FunctionCode][]byte{FunctionCode(0x10): {0x01}}}
	if _, ok, err := buildSlaveAnswer(Frame{SourceAddress: 1, TargetAddress: 2, MessageNumber: 1, Data: []byte{0x10}}, badPayload); err == nil || ok {
		t.Fatalf("buildSlaveAnswer(reserved custom payload) ok/error = %v/%v, want error", ok, err)
	}
	if _, ok, err := buildSlaveAnswer(Frame{SourceAddress: 1, TargetAddress: 2, MessageNumber: 1, Data: []byte{byte(FunctionQueryDeviceStatus)}}, SlaveUnitState{Address: 2}); err != nil || ok {
		t.Fatalf("buildSlaveAnswer(no auto answer) ok/error = %v/%v, want false/nil", ok, err)
	}

	state := cloneSlaveState(SlaveState{AnswerPayloadByFunction: map[FunctionCode][]byte{FunctionQueryProtocolVersion: {0x01}}})
	state.AnswerPayloadByFunction[FunctionQueryProtocolVersion][0] = 0x02
	if state.AnswerPayloadByFunction[FunctionQueryProtocolVersion][0] != 0x02 {
		t.Fatalf("cloneSlaveState() did not preserve cloned payload mutation")
	}
	if got := defaultTargetAddress(nil); got != 2 {
		t.Fatalf("defaultTargetAddress(nil) = %d, want 2", got)
	}
	if got := requestTimeout(0, 12); got != 12*time.Millisecond {
		t.Fatalf("requestTimeout(session default) = %v, want 12ms", got)
	}
	if got := requestRetries(2, 9); got != 2 {
		t.Fatalf("requestRetries(local) = %d, want 2", got)
	}
	if got := (Frame{}).Function(); got != 0 {
		t.Fatalf("empty Frame.Function() = %d, want 0", got)
	}

	historySession := newSession(OpenSessionRequest{ID: "history", Config: port.SerialConfig{PortName: "loop"}}, newFakePort())
	historySession.history = make([]FrameRecord, maxHistory)
	historySession.recordFrame("tx", Frame{Type: FrameTypeRequest, SourceAddress: 1, TargetAddress: 2, MessageNumber: 1, Data: []byte{byte(FunctionSyncHeartbeat)}}, []byte{0x7e}, "")
	if len(historySession.history) != maxHistory {
		t.Fatalf("recordFrame trim length = %d, want %d", len(historySession.history), maxHistory)
	}
	historySession.history = nil
	_, first := mustEncodedFrame(t, Frame{Type: FrameTypeAnswer, TargetAddress: 1, Priority: 3, SourceAddress: 2, MessageNumber: 2, GroupNumber: 1, Data: []byte{byte(FunctionQueryCurrentEvent), 0x01}})
	_, second := mustEncodedFrame(t, Frame{Type: FrameTypeAnswer, TargetAddress: 1, Priority: 3, SourceAddress: 2, MessageNumber: 2, GroupNumber: 2, Data: []byte{0x80}})
	historySession.recordFrame("rx", first, first.Raw, "")
	historySession.recordFrame("rx", second, second.Raw, "")
	page := historySession.queryFrames(QueryRequest{Offset: -5, Limit: 1})
	if page.Offset != 0 || len(page.Frames) != 1 || page.EOF {
		t.Fatalf("queryFrames(negative offset) = offset %d len %d eof %v, want first page", page.Offset, len(page.Frames), page.EOF)
	}
	page = historySession.queryFrames(QueryRequest{Search: "not-present"})
	if page.Total != 0 || !page.EOF {
		t.Fatalf("queryFrames(no search match) = total %d eof %v, want empty EOF", page.Total, page.EOF)
	}
}

func unitAddresses(units []SlaveUnitInfo) string {
	parts := make([]string, 0, len(units))
	for _, unit := range units {
		parts = append(parts, fmt.Sprintf("%d", unit.Address))
	}
	return strings.Join(parts, ",")
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
