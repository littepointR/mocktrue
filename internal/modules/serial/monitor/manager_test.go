package monitor

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/suyue/mocktrue/internal/modules/serial/port"
	"github.com/suyue/mocktrue/internal/modules/serial/virtualserial"
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

func TestManagerQueriesAndExportsSeededFrames(t *testing.T) {
	t.Parallel()
	session := newSession(StartRequest{ID: "m1", PortA: "a", PortB: "b", Provider: ProviderBridge})
	session.appendFrame(DirectionAToB, "a", []byte("alpha"))
	session.appendFrame(DirectionBToA, "b", []byte("beta"))

	mgr := NewManager()
	mgr.sessions["m1"] = session

	page, err := mgr.Query(QueryRequest{MonitorID: "m1", Direction: DirectionBToA, Search: "beta", Limit: 10})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if page.Total != 1 || page.Frames[0].Direction != DirectionBToA {
		t.Fatalf("page = %+v, want one b_to_a frame", page)
	}

	path := filepath.Join(t.TempDir(), "capture.csv")
	if _, err := mgr.Export(ExportRequest{MonitorID: "m1", Format: ExportCSV, Path: path}); err != nil {
		t.Fatalf("Export: %v", err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !strings.Contains(string(content), "alpha") || !strings.Contains(string(content), "beta") {
		t.Fatalf("export = %s, want both frames", string(content))
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
