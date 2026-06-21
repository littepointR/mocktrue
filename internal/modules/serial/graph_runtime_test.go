package serial

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/suyue/mocktrue/internal/modules/serial/buffer"
	fb "github.com/suyue/mocktrue/internal/modules/serial/fecbus"
	mb "github.com/suyue/mocktrue/internal/modules/serial/modbus"
	"github.com/suyue/mocktrue/internal/modules/serial/port"
)

func TestSerialGraphRuntimeSenderReceiver(t *testing.T) {
	svc := NewService(nil)
	defer svc.cleanup()

	graph := startTestGraph(t, svc, SerialGraphStartRequest{
		ID: "graph-basic",
		Nodes: []SerialGraphNodeSpec{
			{ID: "sender", Type: "serial.sender", Config: map[string]any{"mode": "ascii"}},
			{ID: "receiver", Type: "serial.receiver", Config: map[string]any{"viewMode": "ascii"}},
		},
		Edges: []SerialGraphEdgeSpec{
			{ID: "edge-1", Source: "sender", SourceHandle: "out", Target: "receiver", TargetHandle: "in"},
		},
	})

	written, err := svc.SendSerialGraphNode(SerialGraphSendRequest{
		GraphID: graph.ID,
		NodeID:  "sender",
		Content: "hello",
		Mode:    "ascii",
	})
	if err != nil {
		t.Fatalf("SendSerialGraphNode returned error: %v", err)
	}
	if written != 5 {
		t.Fatalf("written = %d, want 5", written)
	}

	page, err := svc.QuerySerialGraphNodeBuffer(SerialGraphBufferQuery{
		GraphID: graph.ID,
		NodeID:  "receiver",
		Offset:  0,
		Length:  32,
	})
	if err != nil {
		t.Fatalf("QuerySerialGraphNodeBuffer returned error: %v", err)
	}
	if string(page.Data) != "hello" {
		t.Fatalf("receiver data = %q, want hello", string(page.Data))
	}

	info, err := svc.GetSerialGraphStatus(graph.ID)
	if err != nil {
		t.Fatalf("GetSerialGraphStatus returned error: %v", err)
	}
	receiver := graphNodeStatus(info, "receiver")
	if receiver.RxBytes != 5 {
		t.Fatalf("receiver rx = %d, want 5", receiver.RxBytes)
	}
}

func TestSerialGraphRuntimeAutoSenderDeliversRepeatedPayload(t *testing.T) {
	svc := NewService(nil)
	defer svc.cleanup()

	graph := startTestGraph(t, svc, SerialGraphStartRequest{
		ID: "graph-auto-send",
		Nodes: []SerialGraphNodeSpec{
			{ID: "sender", Type: "serial.sender", Config: map[string]any{"mode": "ascii", "payload": "tick", "autoSend": true, "intervalMs": 10}},
			{ID: "receiver", Type: "serial.receiver", Config: map[string]any{"viewMode": "ascii"}},
		},
		Edges: []SerialGraphEdgeSpec{
			{ID: "edge-1", Source: "sender", SourceHandle: "out", Target: "receiver", TargetHandle: "in"},
		},
	})

	page := waitGraphBuffer(t, svc, graph.ID, "receiver", len("ticktick"))
	if string(page.Data[:len("ticktick")]) != "ticktick" {
		t.Fatalf("receiver data prefix = %q, want ticktick", string(page.Data))
	}
}

func TestSerialGraphRuntimeAutoSenderStopsWithGraph(t *testing.T) {
	svc := NewService(nil)
	defer svc.cleanup()

	graph := startTestGraph(t, svc, SerialGraphStartRequest{
		ID: "graph-auto-send-stop",
		Nodes: []SerialGraphNodeSpec{
			{ID: "sender", Type: "serial.sender", Config: map[string]any{"mode": "ascii", "payload": "x", "autoSend": true, "intervalMs": 10}},
			{ID: "receiver", Type: "serial.receiver", Config: map[string]any{"viewMode": "ascii"}},
		},
		Edges: []SerialGraphEdgeSpec{
			{ID: "edge-1", Source: "sender", SourceHandle: "out", Target: "receiver", TargetHandle: "in"},
		},
	})

	waitGraphBuffer(t, svc, graph.ID, "receiver", 2)

	svc.mu.RLock()
	runtime := svc.graphs[graph.ID]
	svc.mu.RUnlock()
	if runtime == nil {
		t.Fatalf("runtime not found")
	}
	if err := svc.StopSerialGraph(graph.ID); err != nil {
		t.Fatalf("StopSerialGraph returned error: %v", err)
	}

	time.Sleep(40 * time.Millisecond)
	page, err := runtime.queryBuffer("receiver", 0, 4096)
	if err != nil {
		t.Fatalf("queryBuffer after stop returned error: %v", err)
	}
	stoppedLength := len(page.Data)
	time.Sleep(40 * time.Millisecond)
	page, err = runtime.queryBuffer("receiver", 0, 4096)
	if err != nil {
		t.Fatalf("queryBuffer after stop returned error: %v", err)
	}
	if len(page.Data) != stoppedLength {
		t.Fatalf("auto sender kept writing after stop: length %d -> %d", stoppedLength, len(page.Data))
	}
}

func TestSerialGraphRuntimeTapFansOut(t *testing.T) {
	svc := NewService(nil)
	defer svc.cleanup()

	graph := startTestGraph(t, svc, SerialGraphStartRequest{
		ID: "graph-tap",
		Nodes: []SerialGraphNodeSpec{
			{ID: "sender", Type: "serial.sender"},
			{ID: "tap", Type: "serial.tap"},
			{ID: "rx-a", Type: "serial.receiver"},
			{ID: "rx-b", Type: "serial.receiver"},
		},
		Edges: []SerialGraphEdgeSpec{
			{ID: "edge-1", Source: "sender", SourceHandle: "out", Target: "tap", TargetHandle: "in"},
			{ID: "edge-2", Source: "tap", SourceHandle: "out", Target: "rx-a", TargetHandle: "in"},
			{ID: "edge-3", Source: "tap", SourceHandle: "out", Target: "rx-b", TargetHandle: "in"},
		},
	})

	if _, err := svc.SendSerialGraphNode(SerialGraphSendRequest{
		GraphID: graph.ID,
		NodeID:  "sender",
		Content: "01 02 03",
		Mode:    "hex",
	}); err != nil {
		t.Fatalf("SendSerialGraphNode returned error: %v", err)
	}

	for _, nodeID := range []string{"rx-a", "rx-b"} {
		page, err := svc.QuerySerialGraphNodeBuffer(SerialGraphBufferQuery{
			GraphID: graph.ID,
			NodeID:  nodeID,
			Offset:  0,
			Length:  8,
		})
		if err != nil {
			t.Fatalf("QuerySerialGraphNodeBuffer(%s) returned error: %v", nodeID, err)
		}
		if got := formatHexBytes(page.Data); got != "01 02 03" {
			t.Fatalf("%s data = %q, want 01 02 03", nodeID, got)
		}
	}
}

func TestSerialGraphRuntimeBridgeRoutesOppositeSide(t *testing.T) {
	svc := NewService(nil)
	defer svc.cleanup()

	graph := startTestGraph(t, svc, SerialGraphStartRequest{
		ID: "graph-bridge",
		Nodes: []SerialGraphNodeSpec{
			{ID: "left", Type: "serial.sender"},
			{ID: "right", Type: "serial.receiver"},
			{ID: "bridge", Type: "serial.bridge"},
		},
		Edges: []SerialGraphEdgeSpec{
			{ID: "edge-1", Source: "left", SourceHandle: "out", Target: "bridge", TargetHandle: "a-in"},
			{ID: "edge-2", Source: "bridge", SourceHandle: "b-out", Target: "right", TargetHandle: "in"},
		},
	})

	if _, err := svc.SendSerialGraphNode(SerialGraphSendRequest{
		GraphID: graph.ID,
		NodeID:  "left",
		Content: "bridge",
		Mode:    "ascii",
	}); err != nil {
		t.Fatalf("SendSerialGraphNode returned error: %v", err)
	}

	page, err := svc.QuerySerialGraphNodeBuffer(SerialGraphBufferQuery{
		GraphID: graph.ID,
		NodeID:  "right",
		Offset:  0,
		Length:  32,
	})
	if err != nil {
		t.Fatalf("QuerySerialGraphNodeBuffer returned error: %v", err)
	}
	if string(page.Data) != "bridge" {
		t.Fatalf("right data = %q, want bridge", string(page.Data))
	}
}

func TestSerialGraphRuntimeMonitorCapturesBytes(t *testing.T) {
	svc := NewService(nil)
	defer svc.cleanup()

	graph := startTestGraph(t, svc, SerialGraphStartRequest{
		ID: "graph-monitor",
		Nodes: []SerialGraphNodeSpec{
			{ID: "sender", Type: "serial.sender"},
			{ID: "monitor", Type: "serial.monitor"},
		},
		Edges: []SerialGraphEdgeSpec{
			{ID: "edge-1", Source: "sender", SourceHandle: "out", Target: "monitor", TargetHandle: "in"},
		},
	})

	if _, err := svc.SendSerialGraphNode(SerialGraphSendRequest{
		GraphID: graph.ID,
		NodeID:  "sender",
		Content: "monitored",
		Mode:    "ascii",
	}); err != nil {
		t.Fatalf("SendSerialGraphNode returned error: %v", err)
	}

	page, err := svc.QuerySerialGraphNodeFrames(SerialGraphFrameQuery{
		GraphID: graph.ID,
		NodeID:  "monitor",
		Offset:  0,
		Limit:   10,
	})
	if err != nil {
		t.Fatalf("QuerySerialGraphNodeFrames returned error: %v", err)
	}
	if page.Total != 1 || len(page.Frames) != 1 {
		t.Fatalf("frame page total=%d len=%d, want 1", page.Total, len(page.Frames))
	}
	if page.Frames[0].Length != len("monitored") || page.Frames[0].DisplayText != "monitored" {
		t.Fatalf("frame = %#v, want monitored frame", page.Frames[0])
	}
}

func TestSerialGraphRuntimeModbusMasterBuildsRequestFrame(t *testing.T) {
	svc := NewService(nil)
	defer svc.cleanup()

	graph := startTestGraph(t, svc, SerialGraphStartRequest{
		ID: "graph-modbus-master",
		Nodes: []SerialGraphNodeSpec{
			{ID: "master", Type: "serial.modbus.master", Config: map[string]any{"mode": "rtu", "unitIds": "7", "functionCode": 3, "address": 12, "quantity": 2}},
			{ID: "receiver", Type: "serial.receiver"},
		},
		Edges: []SerialGraphEdgeSpec{
			{ID: "edge-1", Source: "master", SourceHandle: "tx", Target: "receiver", TargetHandle: "in"},
		},
	})

	if _, err := svc.SendSerialGraphNode(SerialGraphSendRequest{GraphID: graph.ID, NodeID: "master"}); err != nil {
		t.Fatalf("SendSerialGraphNode returned error: %v", err)
	}
	page, err := svc.QuerySerialGraphNodeBuffer(SerialGraphBufferQuery{GraphID: graph.ID, NodeID: "receiver", Offset: 0, Length: 64})
	if err != nil {
		t.Fatalf("QuerySerialGraphNodeBuffer returned error: %v", err)
	}
	frame, err := mb.DecodeFrame(mb.FrameModeRTU, page.Data)
	if err != nil {
		t.Fatalf("DecodeFrame: %v", err)
	}
	if frame.UnitID != 7 || frame.PDU.Function != mb.FunctionReadHoldingRegisters {
		t.Fatalf("decoded frame = %#v, want unit 7 function 3", frame)
	}
}

func TestSerialGraphRuntimeModbusSlaveAutoResponds(t *testing.T) {
	svc := NewService(nil)
	defer svc.cleanup()

	graph := startTestGraph(t, svc, SerialGraphStartRequest{
		ID: "graph-modbus-slave",
		Nodes: []SerialGraphNodeSpec{
			{ID: "master", Type: "serial.modbus.master", Config: map[string]any{"mode": "rtu", "unitIds": "3", "functionCode": 3, "address": 0, "quantity": 2}},
			{ID: "slave", Type: "serial.modbus.slave", Config: map[string]any{"mode": "rtu", "unitIds": "3"}},
			{ID: "receiver", Type: "serial.receiver"},
		},
		Edges: []SerialGraphEdgeSpec{
			{ID: "edge-1", Source: "master", SourceHandle: "tx", Target: "slave", TargetHandle: "rx"},
			{ID: "edge-2", Source: "slave", SourceHandle: "tx", Target: "receiver", TargetHandle: "in"},
		},
	})

	if _, err := svc.SendSerialGraphNode(SerialGraphSendRequest{GraphID: graph.ID, NodeID: "master"}); err != nil {
		t.Fatalf("SendSerialGraphNode returned error: %v", err)
	}
	page, err := svc.QuerySerialGraphNodeBuffer(SerialGraphBufferQuery{GraphID: graph.ID, NodeID: "receiver", Offset: 0, Length: 64})
	if err != nil {
		t.Fatalf("QuerySerialGraphNodeBuffer returned error: %v", err)
	}
	frame, err := mb.DecodeFrame(mb.FrameModeRTU, page.Data)
	if err != nil {
		t.Fatalf("DecodeFrame: %v", err)
	}
	parsed, err := mb.ParseResponse(frame.PDU)
	if err != nil {
		t.Fatalf("ParseResponse: %v", err)
	}
	if frame.UnitID != 3 || parsed.Function != mb.FunctionReadHoldingRegisters || len(parsed.Values) != 2 {
		t.Fatalf("parsed response = %#v unit=%d, want 2 holding registers for unit 3", parsed, frame.UnitID)
	}
}

func TestSerialGraphRuntimeFecbusSlaveAutoResponds(t *testing.T) {
	svc := NewService(nil)
	defer svc.cleanup()

	graph := startTestGraph(t, svc, SerialGraphStartRequest{
		ID: "graph-fecbus-slave",
		Nodes: []SerialGraphNodeSpec{
			{ID: "master", Type: "serial.fecbus.master", Config: map[string]any{"sourceAddress": 1, "targetAddress": 2, "functionCode": int(fb.FunctionQueryProtocolVersion), "messageNumber": 5}},
			{ID: "slave", Type: "serial.fecbus.slave", Config: map[string]any{"address": 2, "autoStatusAnswer": true}},
			{ID: "receiver", Type: "serial.receiver"},
		},
		Edges: []SerialGraphEdgeSpec{
			{ID: "edge-1", Source: "master", SourceHandle: "tx", Target: "slave", TargetHandle: "rx"},
			{ID: "edge-2", Source: "slave", SourceHandle: "tx", Target: "receiver", TargetHandle: "in"},
		},
	})

	if _, err := svc.SendSerialGraphNode(SerialGraphSendRequest{GraphID: graph.ID, NodeID: "master"}); err != nil {
		t.Fatalf("SendSerialGraphNode returned error: %v", err)
	}
	page, err := svc.QuerySerialGraphNodeBuffer(SerialGraphBufferQuery{GraphID: graph.ID, NodeID: "receiver", Offset: 0, Length: 64})
	if err != nil {
		t.Fatalf("QuerySerialGraphNodeBuffer returned error: %v", err)
	}
	frame, err := fb.DecodeFrame(page.Data)
	if err != nil {
		t.Fatalf("DecodeFrame: %v", err)
	}
	if frame.Type != fb.FrameTypeAnswer || frame.TargetAddress != 1 || frame.SourceAddress != 2 || frame.Function() != fb.FunctionStatusAnswer {
		t.Fatalf("fecbus answer = %#v, want status answer from 2 to 1", frame)
	}
}

func TestSerialGraphRuntimeResetCountersAndClearBuffer(t *testing.T) {
	svc := NewService(nil)
	defer svc.cleanup()

	graph := startTestGraph(t, svc, SerialGraphStartRequest{
		ID: "graph-reset",
		Nodes: []SerialGraphNodeSpec{
			{ID: "sender", Type: "serial.sender"},
			{ID: "receiver", Type: "serial.receiver"},
		},
		Edges: []SerialGraphEdgeSpec{
			{ID: "edge-1", Source: "sender", SourceHandle: "out", Target: "receiver", TargetHandle: "in"},
		},
	})
	if _, err := svc.SendSerialGraphNode(SerialGraphSendRequest{
		GraphID: graph.ID,
		NodeID:  "sender",
		Content: "data",
		Mode:    "ascii",
	}); err != nil {
		t.Fatalf("SendSerialGraphNode returned error: %v", err)
	}

	if err := svc.ResetSerialGraphNodeCounters(graph.ID, "receiver"); err != nil {
		t.Fatalf("ResetSerialGraphNodeCounters returned error: %v", err)
	}
	if err := svc.ClearSerialGraphNodeBuffer(graph.ID, "receiver"); err != nil {
		t.Fatalf("ClearSerialGraphNodeBuffer returned error: %v", err)
	}

	info, err := svc.GetSerialGraphStatus(graph.ID)
	if err != nil {
		t.Fatalf("GetSerialGraphStatus returned error: %v", err)
	}
	if receiver := graphNodeStatus(info, "receiver"); receiver.RxBytes != 0 {
		t.Fatalf("receiver rx after reset = %d, want 0", receiver.RxBytes)
	}
	page, err := svc.QuerySerialGraphNodeBuffer(SerialGraphBufferQuery{
		GraphID: graph.ID,
		NodeID:  "receiver",
		Offset:  0,
		Length:  8,
	})
	if err != nil {
		t.Fatalf("QuerySerialGraphNodeBuffer returned error: %v", err)
	}
	if page.Total != 0 || len(page.Data) != 0 {
		t.Fatalf("buffer after clear total=%d len=%d, want empty", page.Total, len(page.Data))
	}
}

func TestSerialGraphRuntimeRejectsInvalidGraph(t *testing.T) {
	svc := NewService(nil)
	defer svc.cleanup()

	_, err := svc.StartSerialGraph(context.Background(), SerialGraphStartRequest{
		ID: "invalid",
		Nodes: []SerialGraphNodeSpec{
			{ID: "sender", Type: "serial.sender"},
			{ID: "rx-a", Type: "serial.receiver"},
			{ID: "rx-b", Type: "serial.receiver"},
		},
		Edges: []SerialGraphEdgeSpec{
			{ID: "edge-1", Source: "sender", SourceHandle: "out", Target: "rx-a", TargetHandle: "in"},
			{ID: "edge-2", Source: "sender", SourceHandle: "out", Target: "rx-b", TargetHandle: "in"},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "fan-out requires a tap node") {
		t.Fatalf("StartSerialGraph error = %v, want fan-out validation error", err)
	}
}

func TestSerialGraphRuntimePhysicalPortRoutesReceivedBytes(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping socat integration test in short mode")
	}
	pair, err := port.StartVirtualPair(context.Background())
	if err != nil {
		t.Skipf("socat not available: %v", err)
	}
	defer pair.Stop()

	svc := NewService(nil)
	defer svc.cleanup()
	graph := startTestGraph(t, svc, SerialGraphStartRequest{
		ID: "graph-physical",
		Nodes: []SerialGraphNodeSpec{
			{ID: "port", Type: "serial.physical", Config: map[string]any{"portName": pair.Port1, "baudRate": 115200}},
			{ID: "receiver", Type: "serial.receiver"},
		},
		Edges: []SerialGraphEdgeSpec{
			{ID: "edge-1", Source: "port", SourceHandle: "rx", Target: "receiver", TargetHandle: "in"},
		},
	})

	conn, err := port.OpenForTest(pair.Port2, 115200)
	if err != nil {
		t.Fatalf("OpenForTest: %v", err)
	}
	defer conn.Close()
	if _, err := conn.Write([]byte("from-external")); err != nil {
		t.Fatalf("external Write: %v", err)
	}

	page := waitGraphBuffer(t, svc, graph.ID, "receiver", len("from-external"))
	if string(page.Data) != "from-external" {
		t.Fatalf("receiver data = %q, want from-external", string(page.Data))
	}
}

func TestSerialGraphRuntimeVirtualPortExposesExternalEndpoint(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping socat integration test in short mode")
	}
	svc := NewService(nil)
	defer svc.cleanup()
	portName := "mocktrue-graph-test-vp"
	graph, err := svc.StartSerialGraph(context.Background(), SerialGraphStartRequest{
		ID: "graph-virtual",
		Nodes: []SerialGraphNodeSpec{
			{ID: "sender", Type: "serial.sender"},
			{ID: "vport", Type: "serial.virtual", Config: map[string]any{"portName": portName, "baudRate": 115200}},
			{ID: "receiver", Type: "serial.receiver"},
		},
		Edges: []SerialGraphEdgeSpec{
			{ID: "edge-1", Source: "sender", SourceHandle: "out", Target: "vport", TargetHandle: "tx"},
			{ID: "edge-2", Source: "vport", SourceHandle: "rx", Target: "receiver", TargetHandle: "in"},
		},
	})
	if err != nil {
		t.Skipf("virtual serial unavailable: %v", err)
	}

	virtualNode := graphNodeStatus(graph, "vport")
	if virtualNode.ResourceID == "" {
		t.Fatalf("virtual node resource ID must expose public port")
	}
	external, err := port.OpenForTest(virtualNode.ResourceID, 115200)
	if err != nil {
		t.Fatalf("OpenForTest virtual public port: %v", err)
	}
	defer external.Close()
	if err := external.SetReadTimeout(2 * time.Second); err != nil {
		t.Fatalf("SetReadTimeout: %v", err)
	}

	if _, err := svc.SendSerialGraphNode(SerialGraphSendRequest{
		GraphID: graph.ID,
		NodeID:  "sender",
		Content: "to-external",
		Mode:    "ascii",
	}); err != nil {
		t.Fatalf("SendSerialGraphNode: %v", err)
	}
	read := make([]byte, len("to-external"))
	total := 0
	for total < len(read) {
		n, err := external.Read(read[total:])
		if err != nil {
			t.Fatalf("external Read: %v", err)
		}
		total += n
	}
	if string(read) != "to-external" {
		t.Fatalf("external read = %q, want to-external", string(read))
	}

	if err := svc.ClearSerialGraphNodeBuffer(graph.ID, "receiver"); err != nil {
		t.Fatalf("ClearSerialGraphNodeBuffer: %v", err)
	}
	if _, err := external.Write([]byte("to-graph")); err != nil {
		t.Fatalf("external Write: %v", err)
	}
	page := waitGraphBuffer(t, svc, graph.ID, "receiver", len("to-graph"))
	if string(page.Data) != "to-graph" {
		t.Fatalf("receiver data = %q, want to-graph", string(page.Data))
	}
}

func TestSerialGraphRuntimeAutoSenderThroughVirtualPort(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping socat integration test in short mode")
	}
	svc := NewService(nil)
	defer svc.cleanup()
	graph, err := svc.StartSerialGraph(context.Background(), SerialGraphStartRequest{
		ID: "graph-auto-virtual",
		Nodes: []SerialGraphNodeSpec{
			{ID: "sender", Type: "serial.sender", Config: map[string]any{"mode": "ascii", "payload": "vp", "autoSend": true, "intervalMs": 10}},
			{ID: "vport", Type: "serial.virtual", Config: map[string]any{"portName": "mocktrue-graph-auto-vp", "baudRate": 115200}},
			{ID: "receiver", Type: "serial.receiver", Config: map[string]any{"viewMode": "ascii"}},
		},
		Edges: []SerialGraphEdgeSpec{
			{ID: "edge-1", Source: "sender", SourceHandle: "out", Target: "vport", TargetHandle: "tx"},
			{ID: "edge-2", Source: "vport", SourceHandle: "rx", Target: "receiver", TargetHandle: "in"},
		},
	})
	if err != nil {
		t.Skipf("virtual serial unavailable: %v", err)
	}

	page := waitGraphBuffer(t, svc, graph.ID, "receiver", len("vpvp"))
	if string(page.Data[:len("vpvp")]) != "vpvp" {
		t.Fatalf("receiver data prefix = %q, want vpvp", string(page.Data))
	}
}

func TestSerialGraphRuntimeAutoSenderThroughVirtualPortSurvivesExternalBackpressure(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping socat integration test in short mode")
	}
	svc := NewService(nil)
	defer svc.cleanup()
	payload := strings.Repeat("x", 64*1024)
	graph, err := svc.StartSerialGraph(context.Background(), SerialGraphStartRequest{
		ID: "graph-auto-virtual-backpressure",
		Nodes: []SerialGraphNodeSpec{
			{ID: "sender", Type: "serial.sender", Config: map[string]any{"mode": "ascii", "payload": payload, "autoSend": true, "intervalMs": 10}},
			{ID: "vport", Type: "serial.virtual", Config: map[string]any{"portName": "mocktrue-graph-auto-backpressure-vp", "baudRate": 115200}},
			{ID: "receiver", Type: "serial.receiver", Config: map[string]any{"viewMode": "ascii"}},
		},
		Edges: []SerialGraphEdgeSpec{
			{ID: "edge-1", Source: "sender", SourceHandle: "out", Target: "vport", TargetHandle: "tx"},
			{ID: "edge-2", Source: "vport", SourceHandle: "rx", Target: "receiver", TargetHandle: "in"},
		},
	})
	if err != nil {
		t.Skipf("virtual serial unavailable: %v", err)
	}

	waitGraphBuffer(t, svc, graph.ID, "receiver", len(payload)*3)
	stopped := make(chan error, 1)
	go func() {
		stopped <- svc.StopSerialGraph(graph.ID)
	}()

	select {
	case err := <-stopped:
		if err != nil {
			t.Fatalf("StopSerialGraph returned error: %v", err)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("StopSerialGraph did not return while virtual auto sender was under external backpressure")
	}
}

func TestSerialGraphRuntimeMonitorRetainsBoundedFrameHistoryUnderLoad(t *testing.T) {
	svc := NewService(nil)
	defer svc.cleanup()
	graph := startTestGraph(t, svc, SerialGraphStartRequest{
		ID: "graph-monitor-bounded-load",
		Nodes: []SerialGraphNodeSpec{
			{ID: "sender", Type: "serial.sender"},
			{ID: "monitor", Type: "serial.monitor"},
		},
		Edges: []SerialGraphEdgeSpec{
			{ID: "edge-1", Source: "sender", SourceHandle: "out", Target: "monitor", TargetHandle: "in"},
		},
	})

	payload := strings.Repeat("x", 4*1024*1024)
	for i := 0; i < 5; i++ {
		if _, err := svc.SendSerialGraphNode(SerialGraphSendRequest{
			GraphID: graph.ID,
			NodeID:  "sender",
			Content: payload,
			Mode:    "ascii",
		}); err != nil {
			t.Fatalf("SendSerialGraphNode #%d returned error: %v", i, err)
		}
	}

	page, err := svc.QuerySerialGraphNodeFrames(SerialGraphFrameQuery{
		GraphID: graph.ID,
		NodeID:  "monitor",
		Offset:  0,
		Limit:   10,
	})
	if err != nil {
		t.Fatalf("QuerySerialGraphNodeFrames returned error: %v", err)
	}
	if page.Total != 4 {
		t.Fatalf("retained monitor frames = %d, want 4 frames within 16MiB cap", page.Total)
	}
}

func TestSerialGraphRuntimeDemoTopologiesStressAllDemos(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping full-load demo topology test in short mode")
	}
	const totalBytes = 100 * 1024 * 1024
	const chunkBytes = 1024 * 1024
	demos := []string{
		"serial-open-demo",
		"virtual-port-demo",
		"bridge-demo",
		"monitor-demo",
		"modbus-demo",
		"fecbus-demo",
		"serial-graph-demo",
		"full-workspace-demo",
	}

	for _, demoID := range demos {
		t.Run(demoID, func(t *testing.T) {
			svc := NewService(nil)
			defer svc.cleanup()
			spec := demoLoadGraphStressCase(demoID, totalBytes)
			graph, err := svc.StartSerialGraph(context.Background(), spec.Request)
			if err != nil {
				t.Skipf("virtual serial unavailable: %v", err)
			}
			stopped := false
			defer func() {
				if !stopped {
					_ = svc.StopSerialGraph(graph.ID)
				}
			}()

			for _, workload := range spec.ByteWorkloads {
				sendGraphAsciiBytes(t, svc, graph.ID, workload.NodeID, workload.Bytes, chunkBytes)
			}
			for _, workload := range spec.ProtocolWorkloads {
				for i := 0; i < workload.Iterations; i++ {
					if _, err := svc.SendSerialGraphNode(SerialGraphSendRequest{
						GraphID: graph.ID,
						NodeID:  workload.NodeID,
					}); err != nil {
						t.Fatalf("SendSerialGraphNode %s iteration %d returned error: %v", workload.NodeID, i, err)
					}
				}
			}

			info, err := svc.GetSerialGraphStatus(graph.ID)
			if err != nil {
				t.Fatalf("GetSerialGraphStatus returned error: %v", err)
			}
			for _, id := range spec.StatusNodes {
				status := graphNodeStatus(info, id)
				if status.Status != SerialGraphStatusRunning {
					t.Fatalf("%s status = %q error=%q, want running", id, status.Status, status.Error)
				}
			}

			for _, workload := range spec.ByteWorkloads {
				if got := graphNodeStatus(info, workload.NodeID).TxBytes; got != int64(workload.Bytes) {
					t.Fatalf("%s tx = %d, want %d", workload.NodeID, got, workload.Bytes)
				}
				assertGraphBufferTotal(t, svc, graph.ID, workload.ReceiverID, workload.Bytes)
			}
			for _, workload := range spec.ProtocolWorkloads {
				if got := graphNodeStatus(info, workload.NodeID).TxBytes; got <= 0 {
					t.Fatalf("%s tx = %d, want protocol traffic", workload.NodeID, got)
				}
				assertGraphBufferHasTraffic(t, svc, graph.ID, workload.ReceiverID)
			}
			for _, nodeID := range spec.MonitorNodes {
				page, err := svc.QuerySerialGraphNodeFrames(SerialGraphFrameQuery{
					GraphID: graph.ID,
					NodeID:  nodeID,
					Offset:  0,
					Limit:   32,
				})
				if err != nil {
					t.Fatalf("QuerySerialGraphNodeFrames(%s) returned error: %v", nodeID, err)
				}
				if page.Total == 0 {
					t.Fatalf("monitor %s captured no frames", nodeID)
				}
			}

			stopGraphWithin(t, svc, graph.ID, 2*time.Second)
			stopped = true
			if graphs := svc.ListSerialGraphs(); len(graphs) != 0 {
				t.Fatalf("running graphs after stop = %d, want 0", len(graphs))
			}
			if ports := svc.ListVirtualPorts(); len(ports) != 0 {
				t.Fatalf("virtual ports after graph stop = %d, want 0", len(ports))
			}
		})
	}
}

type demoGraphStressCase struct {
	Request           SerialGraphStartRequest
	StatusNodes       []string
	ByteWorkloads     []demoGraphByteWorkload
	ProtocolWorkloads []demoGraphProtocolWorkload
	MonitorNodes      []string
}

type demoGraphByteWorkload struct {
	NodeID     string
	ReceiverID string
	Bytes      int
}

type demoGraphProtocolWorkload struct {
	NodeID     string
	ReceiverID string
	Iterations int
}

func TestSerialGraphRuntimeRejectsDuplicateVirtualPortNameAcrossGraphs(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping socat integration test in short mode")
	}
	svc := NewService(nil)
	defer svc.cleanup()
	req := func(id string) SerialGraphStartRequest {
		return SerialGraphStartRequest{
			ID: id,
			Nodes: []SerialGraphNodeSpec{
				{ID: "vport", Type: "serial.virtual", Config: map[string]any{"portName": "mocktrue-graph-dup-vp", "baudRate": 115200}},
			},
		}
	}
	if _, err := svc.StartSerialGraph(context.Background(), req("graph-one")); err != nil {
		t.Skipf("virtual serial unavailable: %v", err)
	}
	if _, err := svc.StartSerialGraph(context.Background(), req("graph-two")); err == nil || !strings.Contains(err.Error(), "already") {
		t.Fatalf("second StartSerialGraph error = %v, want already-in-use conflict", err)
	}
}

func startTestGraph(t *testing.T, svc *Service, req SerialGraphStartRequest) *SerialGraphRuntimeInfo {
	t.Helper()
	info, err := svc.StartSerialGraph(context.Background(), req)
	if err != nil {
		t.Fatalf("StartSerialGraph returned error: %v", err)
	}
	if info.Status != SerialGraphStatusRunning {
		t.Fatalf("graph status = %q, want %q", info.Status, SerialGraphStatusRunning)
	}
	return info
}

func demoLoadGraphStressCase(demoID string, totalBytes int) demoGraphStressCase {
	portToken := strings.NewReplacer("-", "_").Replace(demoID)
	portName := func(name string) string {
		return fmt.Sprintf("mocktrue-load-%s-%s-%d", portToken, name, time.Now().UnixNano())
	}
	request := func(nodes []SerialGraphNodeSpec, edges []SerialGraphEdgeSpec) SerialGraphStartRequest {
		return SerialGraphStartRequest{ID: fmt.Sprintf("load-%s", demoID), Nodes: nodes, Edges: edges}
	}
	byteWorkloads := func(paths ...[2]string) []demoGraphByteWorkload {
		parts := distributeBytes(totalBytes, len(paths))
		workloads := make([]demoGraphByteWorkload, 0, len(paths))
		for index, path := range paths {
			workloads = append(workloads, demoGraphByteWorkload{
				NodeID:     path[0],
				ReceiverID: path[1],
				Bytes:      parts[index],
			})
		}
		return workloads
	}
	protocolWorkload := func(nodeID string, receiverID string) []demoGraphProtocolWorkload {
		return []demoGraphProtocolWorkload{{NodeID: nodeID, ReceiverID: receiverID, Iterations: 10_000}}
	}

	switch demoID {
	case "serial-open-demo":
		nodes := []SerialGraphNodeSpec{
			{ID: "sender", Type: "serial.sender", Config: graphSenderConfig()},
			{ID: "vport", Type: "serial.virtual", Config: graphVirtualConfig(portName("open"))},
			{ID: "receiver", Type: "serial.receiver", Config: graphReceiverConfig()},
		}
		edges := []SerialGraphEdgeSpec{
			{ID: "edge-sender-vport", Source: "sender", SourceHandle: "out", Target: "vport", TargetHandle: "tx"},
			{ID: "edge-vport-receiver", Source: "vport", SourceHandle: "rx", Target: "receiver", TargetHandle: "in"},
		}
		return demoGraphStressCase{
			Request:       request(nodes, edges),
			StatusNodes:   []string{"sender", "vport", "receiver"},
			ByteWorkloads: byteWorkloads([2]string{"sender", "receiver"}),
		}
	case "virtual-port-demo":
		nodes := []SerialGraphNodeSpec{
			{ID: "sensor-sender", Type: "serial.sender", Config: graphSenderConfig()},
			{ID: "sensor-vport", Type: "serial.virtual", Config: graphVirtualConfig(portName("sensor"))},
			{ID: "sensor-receiver", Type: "serial.receiver", Config: graphReceiverConfig()},
			{ID: "gateway-sender", Type: "serial.sender", Config: graphSenderConfig()},
			{ID: "gateway-vport", Type: "serial.virtual", Config: graphVirtualConfig(portName("gateway"))},
			{ID: "gateway-receiver", Type: "serial.receiver", Config: graphReceiverConfig()},
			{ID: "logger-sender", Type: "serial.sender", Config: graphSenderConfig()},
			{ID: "logger-vport", Type: "serial.virtual", Config: graphVirtualConfig(portName("logger"))},
			{ID: "logger-receiver", Type: "serial.receiver", Config: graphReceiverConfig()},
		}
		edges := []SerialGraphEdgeSpec{
			{ID: "edge-sensor-sender-vport", Source: "sensor-sender", SourceHandle: "out", Target: "sensor-vport", TargetHandle: "tx"},
			{ID: "edge-sensor-vport-receiver", Source: "sensor-vport", SourceHandle: "rx", Target: "sensor-receiver", TargetHandle: "in"},
			{ID: "edge-gateway-sender-vport", Source: "gateway-sender", SourceHandle: "out", Target: "gateway-vport", TargetHandle: "tx"},
			{ID: "edge-gateway-vport-receiver", Source: "gateway-vport", SourceHandle: "rx", Target: "gateway-receiver", TargetHandle: "in"},
			{ID: "edge-logger-sender-vport", Source: "logger-sender", SourceHandle: "out", Target: "logger-vport", TargetHandle: "tx"},
			{ID: "edge-logger-vport-receiver", Source: "logger-vport", SourceHandle: "rx", Target: "logger-receiver", TargetHandle: "in"},
		}
		return demoGraphStressCase{
			Request:     request(nodes, edges),
			StatusNodes: []string{"sensor-sender", "sensor-vport", "sensor-receiver", "gateway-sender", "gateway-vport", "gateway-receiver", "logger-sender", "logger-vport", "logger-receiver"},
			ByteWorkloads: byteWorkloads(
				[2]string{"sensor-sender", "sensor-receiver"},
				[2]string{"gateway-sender", "gateway-receiver"},
				[2]string{"logger-sender", "logger-receiver"},
			),
		}
	case "bridge-demo":
		nodes := []SerialGraphNodeSpec{
			{ID: "sender-a", Type: "serial.sender", Config: graphSenderConfig()},
			{ID: "vport-a", Type: "serial.virtual", Config: graphVirtualConfig(portName("bridge-a"))},
			{ID: "sender-b", Type: "serial.sender", Config: graphSenderConfig()},
			{ID: "vport-b", Type: "serial.virtual", Config: graphVirtualConfig(portName("bridge-b"))},
			{ID: "bridge", Type: "serial.bridge", Config: map[string]any{"baudRate": 115200}},
			{ID: "receiver-a", Type: "serial.receiver", Config: graphReceiverConfig()},
			{ID: "receiver-b", Type: "serial.receiver", Config: graphReceiverConfig()},
		}
		edges := []SerialGraphEdgeSpec{
			{ID: "edge-sender-a-vport-a", Source: "sender-a", SourceHandle: "out", Target: "vport-a", TargetHandle: "tx"},
			{ID: "edge-vport-a-bridge", Source: "vport-a", SourceHandle: "rx", Target: "bridge", TargetHandle: "a-in"},
			{ID: "edge-bridge-b-receiver", Source: "bridge", SourceHandle: "b-out", Target: "receiver-b", TargetHandle: "in"},
			{ID: "edge-sender-b-vport-b", Source: "sender-b", SourceHandle: "out", Target: "vport-b", TargetHandle: "tx"},
			{ID: "edge-vport-b-bridge", Source: "vport-b", SourceHandle: "rx", Target: "bridge", TargetHandle: "b-in"},
			{ID: "edge-bridge-a-receiver", Source: "bridge", SourceHandle: "a-out", Target: "receiver-a", TargetHandle: "in"},
		}
		return demoGraphStressCase{
			Request:     request(nodes, edges),
			StatusNodes: []string{"sender-a", "vport-a", "sender-b", "vport-b", "bridge", "receiver-a", "receiver-b"},
			ByteWorkloads: byteWorkloads(
				[2]string{"sender-a", "receiver-b"},
				[2]string{"sender-b", "receiver-a"},
			),
		}
	case "monitor-demo":
		nodes := []SerialGraphNodeSpec{
			{ID: "sender", Type: "serial.sender", Config: graphSenderConfig()},
			{ID: "vport", Type: "serial.virtual", Config: graphVirtualConfig(portName("monitor"))},
			{ID: "tap", Type: "serial.tap"},
			{ID: "receiver", Type: "serial.receiver", Config: graphReceiverConfig()},
			{ID: "monitor", Type: "serial.monitor", Config: graphMonitorConfig()},
		}
		edges := []SerialGraphEdgeSpec{
			{ID: "edge-sender-vport", Source: "sender", SourceHandle: "out", Target: "vport", TargetHandle: "tx"},
			{ID: "edge-vport-tap", Source: "vport", SourceHandle: "rx", Target: "tap", TargetHandle: "in"},
			{ID: "edge-tap-receiver", Source: "tap", SourceHandle: "out", Target: "receiver", TargetHandle: "in"},
			{ID: "edge-tap-monitor", Source: "tap", SourceHandle: "out", Target: "monitor", TargetHandle: "in"},
		}
		return demoGraphStressCase{
			Request:       request(nodes, edges),
			StatusNodes:   []string{"sender", "vport", "tap", "receiver", "monitor"},
			ByteWorkloads: byteWorkloads([2]string{"sender", "receiver"}),
			MonitorNodes:  []string{"monitor"},
		}
	case "modbus-demo":
		nodes := []SerialGraphNodeSpec{
			{ID: "master", Type: "serial.modbus.master", Config: graphModbusMasterConfig()},
			{ID: "slave", Type: "serial.modbus.slave", Config: graphModbusSlaveConfig()},
			{ID: "tap", Type: "serial.tap"},
			{ID: "receiver", Type: "serial.receiver", Config: graphReceiverConfig()},
			{ID: "monitor", Type: "serial.monitor", Config: graphMonitorConfig()},
		}
		edges := []SerialGraphEdgeSpec{
			{ID: "edge-master-slave", Source: "master", SourceHandle: "tx", Target: "slave", TargetHandle: "rx"},
			{ID: "edge-slave-tap", Source: "slave", SourceHandle: "tx", Target: "tap", TargetHandle: "in"},
			{ID: "edge-tap-receiver", Source: "tap", SourceHandle: "out", Target: "receiver", TargetHandle: "in"},
			{ID: "edge-tap-monitor", Source: "tap", SourceHandle: "out", Target: "monitor", TargetHandle: "in"},
		}
		return demoGraphStressCase{
			Request:           request(nodes, edges),
			StatusNodes:       []string{"master", "slave", "tap", "receiver", "monitor"},
			ProtocolWorkloads: protocolWorkload("master", "receiver"),
			MonitorNodes:      []string{"monitor"},
		}
	case "fecbus-demo":
		nodes := []SerialGraphNodeSpec{
			{ID: "master", Type: "serial.fecbus.master", Config: graphFecbusMasterConfig()},
			{ID: "slave", Type: "serial.fecbus.slave", Config: graphFecbusSlaveConfig()},
			{ID: "tap", Type: "serial.tap"},
			{ID: "receiver", Type: "serial.receiver", Config: graphReceiverConfig()},
			{ID: "monitor", Type: "serial.monitor", Config: graphMonitorConfig()},
		}
		edges := []SerialGraphEdgeSpec{
			{ID: "edge-master-slave", Source: "master", SourceHandle: "tx", Target: "slave", TargetHandle: "rx"},
			{ID: "edge-slave-tap", Source: "slave", SourceHandle: "tx", Target: "tap", TargetHandle: "in"},
			{ID: "edge-tap-receiver", Source: "tap", SourceHandle: "out", Target: "receiver", TargetHandle: "in"},
			{ID: "edge-tap-monitor", Source: "tap", SourceHandle: "out", Target: "monitor", TargetHandle: "in"},
		}
		return demoGraphStressCase{
			Request:           request(nodes, edges),
			StatusNodes:       []string{"master", "slave", "tap", "receiver", "monitor"},
			ProtocolWorkloads: protocolWorkload("master", "receiver"),
			MonitorNodes:      []string{"monitor"},
		}
	case "serial-graph-demo":
		nodes := []SerialGraphNodeSpec{
			{ID: "sender", Type: "serial.sender", Config: graphSenderConfig()},
			{ID: "vport", Type: "serial.virtual", Config: graphVirtualConfig(portName("graph"))},
			{ID: "tap", Type: "serial.tap"},
			{ID: "receiver", Type: "serial.receiver", Config: graphReceiverConfig()},
			{ID: "modbus", Type: "serial.modbus.master", Config: graphModbusMasterConfig()},
			{ID: "monitor", Type: "serial.monitor", Config: graphMonitorConfig()},
		}
		edges := []SerialGraphEdgeSpec{
			{ID: "edge-sender-vport", Source: "sender", SourceHandle: "out", Target: "vport", TargetHandle: "tx"},
			{ID: "edge-vport-tap", Source: "vport", SourceHandle: "rx", Target: "tap", TargetHandle: "in"},
			{ID: "edge-tap-receiver", Source: "tap", SourceHandle: "out", Target: "receiver", TargetHandle: "in"},
			{ID: "edge-tap-modbus", Source: "tap", SourceHandle: "out", Target: "modbus", TargetHandle: "rx"},
			{ID: "edge-tap-monitor", Source: "tap", SourceHandle: "out", Target: "monitor", TargetHandle: "in"},
		}
		return demoGraphStressCase{
			Request:       request(nodes, edges),
			StatusNodes:   []string{"sender", "vport", "tap", "receiver", "modbus", "monitor"},
			ByteWorkloads: byteWorkloads([2]string{"sender", "receiver"}),
			MonitorNodes:  []string{"monitor"},
		}
	case "full-workspace-demo":
		nodes := []SerialGraphNodeSpec{
			{ID: "main-sender", Type: "serial.sender", Config: graphSenderConfig()},
			{ID: "main-vport", Type: "serial.virtual", Config: graphVirtualConfig(portName("main"))},
			{ID: "main-tap", Type: "serial.tap"},
			{ID: "main-receiver", Type: "serial.receiver", Config: graphReceiverConfig()},
			{ID: "main-monitor", Type: "serial.monitor", Config: graphMonitorConfig()},
			{ID: "bridge-sender-a", Type: "serial.sender", Config: graphSenderConfig()},
			{ID: "bridge-vport-a", Type: "serial.virtual", Config: graphVirtualConfig(portName("full-a"))},
			{ID: "bridge-sender-b", Type: "serial.sender", Config: graphSenderConfig()},
			{ID: "bridge-vport-b", Type: "serial.virtual", Config: graphVirtualConfig(portName("full-b"))},
			{ID: "bridge", Type: "serial.bridge", Config: map[string]any{"baudRate": 115200}},
			{ID: "bridge-receiver-a", Type: "serial.receiver", Config: graphReceiverConfig()},
			{ID: "bridge-receiver-b", Type: "serial.receiver", Config: graphReceiverConfig()},
			{ID: "modbus-master", Type: "serial.modbus.master", Config: graphModbusMasterConfig()},
			{ID: "modbus-slave", Type: "serial.modbus.slave", Config: graphModbusSlaveConfig()},
			{ID: "modbus-tap", Type: "serial.tap"},
			{ID: "modbus-receiver", Type: "serial.receiver", Config: graphReceiverConfig()},
			{ID: "modbus-monitor", Type: "serial.monitor", Config: graphMonitorConfig()},
			{ID: "fecbus-master", Type: "serial.fecbus.master", Config: graphFecbusMasterConfig()},
			{ID: "fecbus-slave", Type: "serial.fecbus.slave", Config: graphFecbusSlaveConfig()},
			{ID: "fecbus-tap", Type: "serial.tap"},
			{ID: "fecbus-receiver", Type: "serial.receiver", Config: graphReceiverConfig()},
		}
		edges := []SerialGraphEdgeSpec{
			{ID: "edge-main-sender-vport", Source: "main-sender", SourceHandle: "out", Target: "main-vport", TargetHandle: "tx"},
			{ID: "edge-main-vport-tap", Source: "main-vport", SourceHandle: "rx", Target: "main-tap", TargetHandle: "in"},
			{ID: "edge-main-tap-receiver", Source: "main-tap", SourceHandle: "out", Target: "main-receiver", TargetHandle: "in"},
			{ID: "edge-main-tap-monitor", Source: "main-tap", SourceHandle: "out", Target: "main-monitor", TargetHandle: "in"},
			{ID: "edge-bridge-sender-a-vport", Source: "bridge-sender-a", SourceHandle: "out", Target: "bridge-vport-a", TargetHandle: "tx"},
			{ID: "edge-bridge-vport-a-in", Source: "bridge-vport-a", SourceHandle: "rx", Target: "bridge", TargetHandle: "a-in"},
			{ID: "edge-bridge-b-receiver", Source: "bridge", SourceHandle: "b-out", Target: "bridge-receiver-b", TargetHandle: "in"},
			{ID: "edge-bridge-sender-b-vport", Source: "bridge-sender-b", SourceHandle: "out", Target: "bridge-vport-b", TargetHandle: "tx"},
			{ID: "edge-bridge-vport-b-in", Source: "bridge-vport-b", SourceHandle: "rx", Target: "bridge", TargetHandle: "b-in"},
			{ID: "edge-bridge-a-receiver", Source: "bridge", SourceHandle: "a-out", Target: "bridge-receiver-a", TargetHandle: "in"},
			{ID: "edge-modbus-master-slave", Source: "modbus-master", SourceHandle: "tx", Target: "modbus-slave", TargetHandle: "rx"},
			{ID: "edge-modbus-slave-tap", Source: "modbus-slave", SourceHandle: "tx", Target: "modbus-tap", TargetHandle: "in"},
			{ID: "edge-modbus-tap-receiver", Source: "modbus-tap", SourceHandle: "out", Target: "modbus-receiver", TargetHandle: "in"},
			{ID: "edge-modbus-tap-monitor", Source: "modbus-tap", SourceHandle: "out", Target: "modbus-monitor", TargetHandle: "in"},
			{ID: "edge-fecbus-master-slave", Source: "fecbus-master", SourceHandle: "tx", Target: "fecbus-slave", TargetHandle: "rx"},
			{ID: "edge-fecbus-slave-tap", Source: "fecbus-slave", SourceHandle: "tx", Target: "fecbus-tap", TargetHandle: "in"},
			{ID: "edge-fecbus-tap-receiver", Source: "fecbus-tap", SourceHandle: "out", Target: "fecbus-receiver", TargetHandle: "in"},
		}
		return demoGraphStressCase{
			Request: request(nodes, edges),
			StatusNodes: []string{
				"main-sender", "main-vport", "main-tap", "main-receiver", "main-monitor",
				"bridge-sender-a", "bridge-vport-a", "bridge-sender-b", "bridge-vport-b", "bridge", "bridge-receiver-a", "bridge-receiver-b",
				"modbus-master", "modbus-slave", "modbus-tap", "modbus-receiver", "modbus-monitor",
				"fecbus-master", "fecbus-slave", "fecbus-tap", "fecbus-receiver",
			},
			ByteWorkloads: byteWorkloads(
				[2]string{"main-sender", "main-receiver"},
				[2]string{"bridge-sender-a", "bridge-receiver-b"},
				[2]string{"bridge-sender-b", "bridge-receiver-a"},
			),
			ProtocolWorkloads: append(protocolWorkload("modbus-master", "modbus-receiver"), protocolWorkload("fecbus-master", "fecbus-receiver")...),
			MonitorNodes:      []string{"main-monitor", "modbus-monitor"},
		}
	default:
		panic("unknown demo id: " + demoID)
	}
}

func sendGraphAsciiBytes(t *testing.T, svc *Service, graphID string, nodeID string, totalBytes int, chunkBytes int) {
	t.Helper()
	chunk := strings.Repeat("x", chunkBytes)
	for written := 0; written < totalBytes; {
		next := chunk
		remaining := totalBytes - written
		if remaining < chunkBytes {
			next = strings.Repeat("x", remaining)
		}
		if _, err := svc.SendSerialGraphNode(SerialGraphSendRequest{
			GraphID: graphID,
			NodeID:  nodeID,
			Content: next,
			Mode:    "ascii",
		}); err != nil {
			t.Fatalf("SendSerialGraphNode %s at %d bytes returned error: %v", nodeID, written, err)
		}
		written += len(next)
	}
}

func assertGraphBufferTotal(t *testing.T, svc *Service, graphID string, nodeID string, want int) {
	t.Helper()
	page, err := svc.QuerySerialGraphNodeBuffer(SerialGraphBufferQuery{
		GraphID: graphID,
		NodeID:  nodeID,
		Offset:  0,
		Length:  16,
	})
	if err != nil {
		t.Fatalf("QuerySerialGraphNodeBuffer(%s) returned error after retained-window eviction: %v", nodeID, err)
	}
	if page.Total != int64(want) {
		t.Fatalf("%s buffer total = %d, want lifetime total %d", nodeID, page.Total, want)
	}
}

func assertGraphBufferHasTraffic(t *testing.T, svc *Service, graphID string, nodeID string) {
	t.Helper()
	page, err := svc.QuerySerialGraphNodeBuffer(SerialGraphBufferQuery{
		GraphID: graphID,
		NodeID:  nodeID,
		Offset:  0,
		Length:  64,
	})
	if err != nil {
		t.Fatalf("QuerySerialGraphNodeBuffer(%s) returned error: %v", nodeID, err)
	}
	if page.Total <= 0 || len(page.Data) == 0 {
		t.Fatalf("%s buffer total=%d len=%d, want protocol traffic", nodeID, page.Total, len(page.Data))
	}
}

func distributeBytes(total int, parts int) []int {
	if parts <= 0 {
		return nil
	}
	out := make([]int, parts)
	base := total / parts
	remainder := total % parts
	for index := range out {
		out[index] = base
		if index < remainder {
			out[index]++
		}
	}
	return out
}

func graphSenderConfig() map[string]any {
	return map[string]any{"mode": "ascii", "encoding": "utf-8", "payload": "load", "autoSend": false, "intervalMs": 0}
}

func graphVirtualConfig(portName string) map[string]any {
	return map[string]any{
		"portName":  portName,
		"baudRate":  115200,
		"dataBits":  8,
		"stopBits":  "1",
		"parity":    "none",
		"flowMode":  "none",
		"readBufKB": 32,
	}
}

func graphReceiverConfig() map[string]any {
	return map[string]any{"viewMode": "hexClassic", "autoScroll": true}
}

func graphMonitorConfig() map[string]any {
	return map[string]any{"mode": "auto-virtual", "displayMode": "hex"}
}

func graphModbusMasterConfig() map[string]any {
	return map[string]any{
		"mode":         "rtu",
		"unitIds":      "1,2",
		"functionCode": int(mb.FunctionReadHoldingRegisters),
		"addressMode":  "zero-based",
		"address":      0,
		"quantity":     125,
	}
}

func graphModbusSlaveConfig() map[string]any {
	return map[string]any{"mode": "rtu", "unitIds": "1,2", "addressMode": "zero-based"}
}

func graphFecbusMasterConfig() map[string]any {
	return map[string]any{
		"frameType":     int(fb.FrameTypeRequest),
		"sourceAddress": 1,
		"targetAddress": 2,
		"priority":      2,
		"messageNumber": 7,
		"groupNumber":   0,
		"functionCode":  int(fb.FunctionQueryProtocolVersion),
		"dataHex":       "",
	}
}

func graphFecbusSlaveConfig() map[string]any {
	return map[string]any{"address": 2, "defaultStatus": int(fb.StatusReceivedOK), "autoStatusAnswer": true}
}

func stopGraphWithin(t *testing.T, svc *Service, graphID string, timeout time.Duration) {
	t.Helper()
	stopped := make(chan error, 1)
	go func() {
		stopped <- svc.StopSerialGraph(graphID)
	}()
	select {
	case err := <-stopped:
		if err != nil {
			t.Fatalf("StopSerialGraph returned error: %v", err)
		}
	case <-time.After(timeout):
		t.Fatalf("StopSerialGraph did not return within %s", timeout)
	}
}

func graphNodeStatus(info *SerialGraphRuntimeInfo, id string) SerialGraphNodeStatus {
	for _, node := range info.Nodes {
		if node.ID == id {
			return node
		}
	}
	return SerialGraphNodeStatus{}
}

func waitGraphBuffer(t *testing.T, svc *Service, graphID string, nodeID string, wantBytes int) *buffer.Snapshot {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		page, err := svc.QuerySerialGraphNodeBuffer(SerialGraphBufferQuery{
			GraphID: graphID,
			NodeID:  nodeID,
			Offset:  0,
			Length:  wantBytes,
		})
		if err == nil && len(page.Data) >= wantBytes {
			return page
		}
		time.Sleep(20 * time.Millisecond)
	}
	page, err := svc.QuerySerialGraphNodeBuffer(SerialGraphBufferQuery{
		GraphID: graphID,
		NodeID:  nodeID,
		Offset:  0,
		Length:  wantBytes,
	})
	if err != nil {
		t.Fatalf("QuerySerialGraphNodeBuffer: %v", err)
	}
	t.Fatalf("buffer length = %d, want at least %d", len(page.Data), wantBytes)
	return page
}
