package monitor

import (
	"context"
	"testing"
	"time"

	coreerrors "github.com/littepointR/mocktrue/internal/core/errors"
	"github.com/littepointR/mocktrue/internal/modules/serial/port"
	"github.com/littepointR/mocktrue/internal/modules/serial/virtualserial"
)

func TestManagerRejectsInvalidStartRequests(t *testing.T) {
	t.Parallel()
	mgr := NewManager()
	_, err := mgr.Start(context.Background(), StartRequest{ID: "", PortA: "a", PortB: "b"})
	if err == nil {
		t.Fatalf("Start must reject empty ID")
	}
	_, err = mgr.Start(context.Background(), StartRequest{ID: "m1", PortA: "a", PortB: "a"})
	if err == nil {
		t.Fatalf("Start must reject identical ports")
	}
	_, err = mgr.Start(context.Background(), StartRequest{ID: "m1", PortA: "a", PortB: "b", Provider: ProviderWindowsDriver})
	if err == nil {
		t.Fatalf("Start must reject unavailable provider")
	}
}

func TestManagerStartRejectsDuplicateMonitorID(t *testing.T) {
	t.Parallel()
	mgr := NewManager()
	mgr.sessions["m1"] = newSession(StartRequest{ID: "m1", PortA: "a", PortB: "b", Provider: ProviderBridge})

	_, err := mgr.Start(context.Background(), StartRequest{ID: "m1", PortA: "c", PortB: "d"})
	requireErrorCode(t, err, coreerrors.CodeConflict)
}

func TestManagerStartRejectsRunningPortReuse(t *testing.T) {
	t.Parallel()
	mgr := NewManager()
	session := newSession(StartRequest{ID: "m1", PortA: "occupied", PortB: "peer", Provider: ProviderBridge})
	session.status = StatusRunning
	mgr.sessions["m1"] = session

	_, err := mgr.Start(context.Background(), StartRequest{ID: "m2", PortA: "occupied", PortB: "other"})
	requireErrorCode(t, err, coreerrors.CodeConflict)
}

func TestManagerQueriesSeededFrames(t *testing.T) {
	t.Parallel()
	firstAt := time.Date(2026, 6, 20, 9, 30, 0, 123000000, time.UTC)
	secondAt := firstAt.Add(time.Second)
	session := newSession(StartRequest{ID: "m1", PortA: "a", PortB: "b", Provider: ProviderBridge})
	session.appendFrame(DirectionAToB, "a", []byte("alpha"), firstAt)
	session.appendFrame(DirectionBToA, "b", []byte("beta"), secondAt)

	mgr := NewManager()
	mgr.sessions["m1"] = session

	page, err := mgr.Query(QueryRequest{MonitorID: "m1", Direction: DirectionBToA, Search: "beta", Limit: 10})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if page.Total != 1 || page.Frames[0].Direction != DirectionBToA {
		t.Fatalf("page = %+v, want one b_to_a frame", page)
	}
	if !page.Frames[0].Timestamp.Equal(secondAt) {
		t.Fatalf("timestamp = %s, want captured read time %s", page.Frames[0].Timestamp, secondAt)
	}
}

func TestManagerQueryReportsLookupErrors(t *testing.T) {
	t.Parallel()
	mgr := NewManager()

	_, err := mgr.Query(QueryRequest{})
	requireErrorCode(t, err, coreerrors.CodeInvalid)

	_, err = mgr.Query(QueryRequest{MonitorID: "missing"})
	requireErrorCode(t, err, coreerrors.CodeNotFound)
}

func TestManagerDeleteRemovesSession(t *testing.T) {
	t.Parallel()
	mgr := NewManager()
	session := newSession(StartRequest{ID: "m1", PortA: "a", PortB: "b", Provider: ProviderBridge})
	session.status = StatusRunning
	mgr.sessions["m1"] = session

	if err := mgr.Delete("m1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if got := len(mgr.List()); got != 0 {
		t.Fatalf("len(List()) after Delete = %d, want 0", got)
	}
	_, err := mgr.Query(QueryRequest{MonitorID: "m1"})
	requireErrorCode(t, err, coreerrors.CodeNotFound)
}

func TestManagerDeleteReportsMissingSession(t *testing.T) {
	t.Parallel()
	mgr := NewManager()

	err := mgr.Delete("missing")
	requireErrorCode(t, err, coreerrors.CodeNotFound)
}

func TestManagerStopAllLeavesFrameHistory(t *testing.T) {
	t.Parallel()
	capturedAt := time.Date(2026, 6, 22, 8, 0, 0, 0, time.UTC)
	mgr := NewManager()
	for _, id := range []string{"m1", "m2"} {
		session := newSession(StartRequest{ID: id, PortA: id + "a", PortB: id + "b", Provider: ProviderBridge})
		session.status = StatusRunning
		session.appendFrame(DirectionAToB, id+"a", []byte(id), capturedAt)
		mgr.sessions[id] = session
	}

	mgr.StopAll()

	statuses := map[string]string{}
	for _, info := range mgr.List() {
		statuses[info.ID] = info.Status
	}
	for _, id := range []string{"m1", "m2"} {
		if statuses[id] != StatusStopped {
			t.Fatalf("status[%s] = %q, want %q", id, statuses[id], StatusStopped)
		}
		page, err := mgr.Query(QueryRequest{MonitorID: id, Limit: 10})
		if err != nil {
			t.Fatalf("Query(%s): %v", id, err)
		}
		if page.Total != 1 || string(page.Frames[0].Data) != id {
			t.Fatalf("Query(%s) = %+v, want retained frame", id, page)
		}
	}
}

func TestManagerClearFramesResetsCapturedState(t *testing.T) {
	t.Parallel()
	mgr := NewManager()
	session := newSession(StartRequest{ID: "m1", PortA: "a", PortB: "b", Provider: ProviderBridge})
	session.appendFrame(DirectionAToB, "a", []byte("tx"), time.Date(2026, 6, 22, 8, 1, 0, 0, time.UTC))
	session.appendFrame(DirectionBToA, "b", []byte("rx"), time.Date(2026, 6, 22, 8, 2, 0, 0, time.UTC))
	mgr.sessions["m1"] = session

	if err := mgr.ClearFrames("m1"); err != nil {
		t.Fatalf("ClearFrames: %v", err)
	}
	infos := mgr.List()
	if got := infos[0].FrameCount; got != 0 {
		t.Fatalf("FrameCount after ClearFrames = %d, want 0", got)
	}
	if infos[0].RxBytes != 0 || infos[0].TxBytes != 0 {
		t.Fatalf("byte counters after ClearFrames = rx %d tx %d, want zeros", infos[0].RxBytes, infos[0].TxBytes)
	}
	page, err := mgr.Query(QueryRequest{MonitorID: "m1", Limit: 10})
	if err != nil {
		t.Fatalf("Query after ClearFrames: %v", err)
	}
	if page.Total != 0 || len(page.Frames) != 0 {
		t.Fatalf("page after ClearFrames = %+v, want empty page", page)
	}

	session.appendFrame(DirectionAToB, "a", []byte("again"), time.Date(2026, 6, 22, 8, 3, 0, 0, time.UTC))
	page, err = mgr.Query(QueryRequest{MonitorID: "m1", Limit: 10})
	if err != nil {
		t.Fatalf("Query after reappend: %v", err)
	}
	if page.Total != 1 || page.Frames[0].Seq != 1 {
		t.Fatalf("page after reappend = %+v, want sequence reset to 1", page)
	}
}

func TestManagerClearFramesReportsLookupErrors(t *testing.T) {
	t.Parallel()
	mgr := NewManager()

	err := mgr.ClearFrames("")
	requireErrorCode(t, err, coreerrors.CodeInvalid)

	err = mgr.ClearFrames("missing")
	requireErrorCode(t, err, coreerrors.CodeNotFound)
}

func TestSessionSetErrorUpdatesInfo(t *testing.T) {
	t.Parallel()
	session := newSession(StartRequest{ID: "m1", PortA: "a", PortB: "b", Provider: ProviderBridge})

	session.setError("open failed")

	info := session.info()
	if info.Status != StatusError || info.Error != "open failed" {
		t.Fatalf("info after setError = %+v, want error status with message", info)
	}
}

func TestSessionFailRecordsError(t *testing.T) {
	t.Parallel()
	session := newSession(StartRequest{ID: "m1", PortA: "a", PortB: "b", Provider: ProviderBridge})
	session.status = StatusRunning

	session.fail("read failed")

	info := session.info()
	if info.Status != StatusError || info.Error != "read failed" {
		t.Fatalf("info after fail = %+v, want error status with message", info)
	}
	if info.StoppedAt.IsZero() {
		t.Fatalf("StoppedAt after fail is zero")
	}
	select {
	case <-session.stopCh:
	default:
		t.Fatalf("stopCh was not closed by fail")
	}
}

func TestManagerBridgeMonitorCapturesBothDirections(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping socat integration test in short mode")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	vmgr := virtualserial.NewManager()
	pairA, err := vmgr.CreatePair(ctx, "mon-a", uniquePortName("monA1"), uniquePortName("monA2"))
	if err != nil {
		t.Skipf("socat not available: %v", err)
	}
	defer vmgr.Cleanup()
	pairB, err := vmgr.CreatePair(ctx, "mon-b", uniquePortName("monB1"), uniquePortName("monB2"))
	if err != nil {
		t.Fatalf("CreatePair B: %v", err)
	}

	mgr := NewManager()
	if _, err := mgr.Start(ctx, StartRequest{
		ID:       "m1",
		Provider: ProviderBridge,
		PortA:    pairA.Port1,
		PortB:    pairB.Port1,
		Config:   port.SerialConfig{BaudRate: 115200, DataBits: 8, StopBits: "1", Parity: "none"},
	}); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer mgr.Cleanup()

	peerA, err := port.OpenForTest(pairA.Port2, 115200)
	if err != nil {
		t.Fatalf("OpenForTest peerA: %v", err)
	}
	defer peerA.Close()
	peerB, err := port.OpenForTest(pairB.Port2, 115200)
	if err != nil {
		t.Fatalf("OpenForTest peerB: %v", err)
	}
	defer peerB.Close()

	if _, err := peerA.Write([]byte("hello-b")); err != nil {
		t.Fatalf("Write A: %v", err)
	}
	readBuf := make([]byte, 32)
	n, err := peerB.Read(readBuf)
	if err != nil {
		t.Fatalf("Read B: %v", err)
	}
	if string(readBuf[:n]) != "hello-b" {
		t.Fatalf("B read %q", string(readBuf[:n]))
	}

	if _, err := peerB.Write([]byte("hello-a")); err != nil {
		t.Fatalf("Write B: %v", err)
	}
	n, err = peerA.Read(readBuf)
	if err != nil {
		t.Fatalf("Read A: %v", err)
	}
	if string(readBuf[:n]) != "hello-a" {
		t.Fatalf("A read %q", string(readBuf[:n]))
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		page, err := mgr.Query(QueryRequest{MonitorID: "m1", Limit: 10})
		if err != nil {
			t.Fatalf("Query: %v", err)
		}
		if page.Total >= 2 {
			directions := map[string]bool{}
			for _, frame := range page.Frames {
				directions[frame.Direction] = true
			}
			if directions[DirectionAToB] && directions[DirectionBToA] {
				return
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("monitor did not capture both directions")
}

func uniquePortName(prefix string) string {
	return prefix + time.Now().Format("150405000000")
}

func requireErrorCode(t *testing.T, err error, want coreerrors.Code) {
	t.Helper()
	if err == nil {
		t.Fatalf("error = nil, want code %q", want)
	}
	if got := coreerrors.AsCode(err); got != want {
		t.Fatalf("error code = %q, want %q (err: %v)", got, want, err)
	}
}
