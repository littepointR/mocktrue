package serial

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestSerialGraphScriptTransformRewritesOutput(t *testing.T) {
	svc := NewService(nil)
	defer svc.cleanup()

	graph := startTestGraph(t, svc, SerialGraphStartRequest{
		ID: "graph-script-transform",
		Nodes: []SerialGraphNodeSpec{
			{ID: "sender", Type: "serial.virtual"},
			{ID: "script", Type: "serial.script.transform", Config: map[string]any{
				"script":         `output.hex("de ad " + input.hex());`,
				"timeoutMs":      100,
				"maxOutputBytes": 16,
				"maxStateBytes":  1024,
				"onError":        "mark",
				"encoding":       "utf-8",
				"autoRun":        false,
				"intervalMs":     0,
				"displayMode":    "hex",
			}},
			{ID: "receiver", Type: "serial.virtual"},
		},
		Edges: []SerialGraphEdgeSpec{
			{ID: "edge-1", Source: "sender", SourceHandle: "rx", Target: "script", TargetHandle: "in"},
			{ID: "edge-2", Source: "script", SourceHandle: "out", Target: "receiver", TargetHandle: "tx"},
		},
	})

	if _, err := svc.SendSerialGraphNode(SerialGraphSendRequest{GraphID: graph.ID, NodeID: "sender", Content: "01 02", Mode: "hex"}); err != nil {
		t.Fatalf("SendSerialGraphNode returned error: %v", err)
	}
	page, err := svc.QuerySerialGraphNodeBuffer(SerialGraphBufferQuery{GraphID: graph.ID, NodeID: "receiver", Offset: 0, Length: 8})
	if err != nil {
		t.Fatalf("QuerySerialGraphNodeBuffer returned error: %v", err)
	}
	if got := formatHexBytes(page.Data); got != "de ad 01 02" {
		t.Fatalf("receiver data = %q, want de ad 01 02", got)
	}
}

func TestSerialGraphScriptTransformDropDoesNotEmit(t *testing.T) {
	svc := NewService(nil)
	defer svc.cleanup()

	graph := startTestGraph(t, svc, SerialGraphStartRequest{
		ID: "graph-script-drop",
		Nodes: []SerialGraphNodeSpec{
			{ID: "sender", Type: "serial.virtual"},
			{ID: "script", Type: "serial.script.transform", Config: map[string]any{
				"script":        `drop(); output.text("should-not-emit");`,
				"timeoutMs":     100,
				"encoding":      "utf-8",
				"maxStateBytes": 1024,
			}},
			{ID: "receiver", Type: "serial.virtual"},
		},
		Edges: []SerialGraphEdgeSpec{
			{ID: "edge-1", Source: "sender", SourceHandle: "rx", Target: "script", TargetHandle: "in"},
			{ID: "edge-2", Source: "script", SourceHandle: "out", Target: "receiver", TargetHandle: "tx"},
		},
	})

	if _, err := svc.SendSerialGraphNode(SerialGraphSendRequest{GraphID: graph.ID, NodeID: "sender", Content: "hello", Mode: "ascii"}); err != nil {
		t.Fatalf("SendSerialGraphNode returned error: %v", err)
	}
	page, err := svc.QuerySerialGraphNodeBuffer(SerialGraphBufferQuery{GraphID: graph.ID, NodeID: "receiver", Offset: 0, Length: 32})
	if err != nil {
		t.Fatalf("QuerySerialGraphNodeBuffer returned error: %v", err)
	}
	if len(page.Data) != 0 || page.Total != 0 {
		t.Fatalf("endpoint buffer len=%d total=%d, want no output", len(page.Data), page.Total)
	}
}

func TestSerialGraphScriptAnalyzerWritesFrame(t *testing.T) {
	svc := NewService(nil)
	defer svc.cleanup()

	graph := startTestGraph(t, svc, SerialGraphStartRequest{
		ID: "graph-script-analyzer",
		Nodes: []SerialGraphNodeSpec{
			{ID: "sender", Type: "serial.virtual"},
			{ID: "analyzer", Type: "serial.script.analyzer", Config: map[string]any{
				"script":      `field("sum", sum8(input.bytes()));`,
				"timeoutMs":   100,
				"encoding":    "utf-8",
				"displayMode": "hex",
			}},
		},
		Edges: []SerialGraphEdgeSpec{
			{ID: "edge-1", Source: "sender", SourceHandle: "rx", Target: "analyzer", TargetHandle: "in"},
		},
	})

	if _, err := svc.SendSerialGraphNode(SerialGraphSendRequest{GraphID: graph.ID, NodeID: "sender", Content: "01 02 03", Mode: "hex"}); err != nil {
		t.Fatalf("SendSerialGraphNode returned error: %v", err)
	}
	page, err := svc.QuerySerialGraphNodeFrames(SerialGraphFrameQuery{GraphID: graph.ID, NodeID: "analyzer", Offset: 0, Limit: 10})
	if err != nil {
		t.Fatalf("QuerySerialGraphNodeFrames returned error: %v", err)
	}
	if page.Total != 1 || len(page.Frames) != 1 {
		t.Fatalf("frame page total=%d len=%d, want one analyzer frame", page.Total, len(page.Frames))
	}
	frame := page.Frames[0]
	if frame.DisplayHex != "01 02 03" || !strings.Contains(frame.DisplayText, "sum=6") {
		t.Fatalf("analyzer frame text=%q hex=%q, want sum field and input bytes", frame.DisplayText, frame.DisplayHex)
	}
}

func TestSerialGraphScriptGeneratorAutoRunsAndStops(t *testing.T) {
	svc := NewService(nil)
	defer svc.cleanup()

	graph := startTestGraph(t, svc, SerialGraphStartRequest{
		ID: "graph-script-generator",
		Nodes: []SerialGraphNodeSpec{
			{ID: "gen", Type: "serial.script.generator", Config: map[string]any{
				"script":         `output.text("g");`,
				"timeoutMs":      100,
				"maxOutputBytes": 8,
				"autoRun":        true,
				"intervalMs":     10,
				"encoding":       "utf-8",
			}},
			{ID: "receiver", Type: "serial.virtual"},
		},
		Edges: []SerialGraphEdgeSpec{
			{ID: "edge-1", Source: "gen", SourceHandle: "out", Target: "receiver", TargetHandle: "tx"},
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
		t.Fatalf("generator kept writing after stop: length %d -> %d", stoppedLength, len(page.Data))
	}
}

func TestSerialGraphScriptGeneratorCannotReadInputAPI(t *testing.T) {
	svc := NewService(nil)
	defer svc.cleanup()

	graph := startTestGraph(t, svc, SerialGraphStartRequest{
		ID: "graph-script-generator-no-input-api",
		Nodes: []SerialGraphNodeSpec{
			{ID: "gen", Type: "serial.script.generator", Config: map[string]any{
				"script":    `output.text(input.hex());`,
				"timeoutMs": 100,
				"encoding":  "utf-8",
			}},
			{ID: "receiver", Type: "serial.virtual"},
		},
		Edges: []SerialGraphEdgeSpec{
			{ID: "edge-1", Source: "gen", SourceHandle: "out", Target: "receiver", TargetHandle: "tx"},
		},
	})

	svc.mu.RLock()
	runtime := svc.graphs[graph.ID]
	svc.mu.RUnlock()
	if runtime == nil {
		t.Fatalf("runtime not found")
	}
	err := runtime.runScriptGeneratorOnce("gen")
	if err == nil || !strings.Contains(err.Error(), "input is not defined") {
		t.Fatalf("runScriptGeneratorOnce error = %v, want unavailable input API", err)
	}
}

func TestSerialGraphScriptGeneratorStopDoesNotBecomeErrorAfterInFlightTimeout(t *testing.T) {
	svc := NewService(nil)
	defer svc.cleanup()

	graph := startTestGraph(t, svc, SerialGraphStartRequest{
		ID: "graph-script-generator-stop-timeout",
		Nodes: []SerialGraphNodeSpec{
			{ID: "gen", Type: "serial.script.generator", Config: map[string]any{
				"script":     `state.set("started", true); while (true) {}`,
				"timeoutMs":  40,
				"autoRun":    true,
				"intervalMs": 100,
			}},
		},
	})

	svc.mu.RLock()
	runtime := svc.graphs[graph.ID]
	svc.mu.RUnlock()
	if runtime == nil {
		t.Fatalf("runtime not found")
	}
	node := runtime.node("gen")
	if node == nil || node.script == nil {
		t.Fatalf("generator runtime not found")
	}
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) && node.script.state.get("started") == nil {
		time.Sleep(5 * time.Millisecond)
	}
	if node.script.state.get("started") == nil {
		t.Fatalf("generator script did not start")
	}
	if err := svc.StopSerialGraph(graph.ID); err != nil {
		t.Fatalf("StopSerialGraph returned error: %v", err)
	}
	time.Sleep(80 * time.Millisecond)

	info := runtime.info()
	status := graphNodeStatus(&info, "gen")
	if status.Status != SerialGraphStatusStopped || status.Error != "" {
		t.Fatalf("generator status=%q error=%q, want stopped without error", status.Status, status.Error)
	}
}

func TestSerialGraphScriptTransformErrorFunctionRemainsFatal(t *testing.T) {
	svc := NewService(nil)
	defer svc.cleanup()

	graph := startTestGraph(t, svc, SerialGraphStartRequest{
		ID: "graph-script-transform-error-fatal",
		Nodes: []SerialGraphNodeSpec{
			{ID: "sender", Type: "serial.virtual"},
			{ID: "script", Type: "serial.script.transform", Config: map[string]any{
				"script":    `error("marked"); output.text("nope");`,
				"timeoutMs": 100,
				"encoding":  "utf-8",
			}},
			{ID: "receiver", Type: "serial.virtual"},
		},
		Edges: []SerialGraphEdgeSpec{
			{ID: "edge-1", Source: "sender", SourceHandle: "rx", Target: "script", TargetHandle: "in"},
			{ID: "edge-2", Source: "script", SourceHandle: "out", Target: "receiver", TargetHandle: "tx"},
		},
	})

	if _, err := svc.SendSerialGraphNode(SerialGraphSendRequest{GraphID: graph.ID, NodeID: "sender", Content: "01", Mode: "hex"}); err != nil {
		t.Fatalf("SendSerialGraphNode returned error: %v", err)
	}
	info, err := svc.GetSerialGraphStatus(graph.ID)
	if err != nil {
		t.Fatalf("GetSerialGraphStatus returned error: %v", err)
	}
	status := graphNodeStatus(info, "script")
	if status.Status != SerialGraphStatusError || !strings.Contains(status.Error, "marked") {
		t.Fatalf("script status=%q error=%q, want fatal script error", status.Status, status.Error)
	}
	page, err := svc.QuerySerialGraphNodeBuffer(SerialGraphBufferQuery{GraphID: graph.ID, NodeID: "receiver", Offset: 0, Length: 32})
	if err != nil {
		t.Fatalf("QuerySerialGraphNodeBuffer returned error: %v", err)
	}
	if len(page.Data) != 0 || page.Total != 0 {
		t.Fatalf("endpoint buffer len=%d total=%d, want no output", len(page.Data), page.Total)
	}
}

func TestSerialGraphScriptGeneratorErrorFunctionRemainsFatal(t *testing.T) {
	svc := NewService(nil)
	defer svc.cleanup()

	graph := startTestGraph(t, svc, SerialGraphStartRequest{
		ID: "graph-script-generator-error-fatal",
		Nodes: []SerialGraphNodeSpec{
			{ID: "gen", Type: "serial.script.generator", Config: map[string]any{
				"script":    `error("marked"); output.text("nope");`,
				"timeoutMs": 100,
				"encoding":  "utf-8",
			}},
			{ID: "receiver", Type: "serial.virtual"},
		},
		Edges: []SerialGraphEdgeSpec{
			{ID: "edge-1", Source: "gen", SourceHandle: "out", Target: "receiver", TargetHandle: "tx"},
		},
	})

	svc.mu.RLock()
	runtime := svc.graphs[graph.ID]
	svc.mu.RUnlock()
	if runtime == nil {
		t.Fatalf("runtime not found")
	}
	err := runtime.runScriptGeneratorOnce("gen")
	if err == nil || !strings.Contains(err.Error(), "marked") {
		t.Fatalf("runScriptGeneratorOnce error = %v, want fatal script error", err)
	}
	page, err := svc.QuerySerialGraphNodeBuffer(SerialGraphBufferQuery{GraphID: graph.ID, NodeID: "receiver", Offset: 0, Length: 32})
	if err != nil {
		t.Fatalf("QuerySerialGraphNodeBuffer returned error: %v", err)
	}
	if len(page.Data) != 0 || page.Total != 0 {
		t.Fatalf("endpoint buffer len=%d total=%d, want no output", len(page.Data), page.Total)
	}
}

func TestSerialGraphScriptAnalyzerCannotWriteOutputAPI(t *testing.T) {
	svc := NewService(nil)
	defer svc.cleanup()

	graph := startTestGraph(t, svc, SerialGraphStartRequest{
		ID: "graph-script-analyzer-no-output-api",
		Nodes: []SerialGraphNodeSpec{
			{ID: "sender", Type: "serial.virtual"},
			{ID: "analyzer", Type: "serial.script.analyzer", Config: map[string]any{
				"script":    `output.text("nope");`,
				"timeoutMs": 100,
				"encoding":  "utf-8",
			}},
		},
		Edges: []SerialGraphEdgeSpec{
			{ID: "edge-1", Source: "sender", SourceHandle: "rx", Target: "analyzer", TargetHandle: "in"},
		},
	})

	if _, err := svc.SendSerialGraphNode(SerialGraphSendRequest{GraphID: graph.ID, NodeID: "sender", Content: "01", Mode: "hex"}); err != nil {
		t.Fatalf("SendSerialGraphNode returned error: %v", err)
	}
	info, err := svc.GetSerialGraphStatus(graph.ID)
	if err != nil {
		t.Fatalf("GetSerialGraphStatus returned error: %v", err)
	}
	status := graphNodeStatus(info, "analyzer")
	if status.Status != SerialGraphStatusError || !strings.Contains(status.Error, "output is not defined") {
		t.Fatalf("analyzer status=%q error=%q, want unavailable output API", status.Status, status.Error)
	}
}

func TestSerialGraphScriptTimeoutMarksErrorAndDoesNotBlockStop(t *testing.T) {
	svc := NewService(nil)
	defer svc.cleanup()

	graph := startTestGraph(t, svc, SerialGraphStartRequest{
		ID: "graph-script-timeout",
		Nodes: []SerialGraphNodeSpec{
			{ID: "sender", Type: "serial.virtual"},
			{ID: "script", Type: "serial.script.transform", Config: map[string]any{
				"script":    `while (true) {}`,
				"timeoutMs": 20,
				"onError":   "mark",
			}},
			{ID: "receiver", Type: "serial.virtual"},
		},
		Edges: []SerialGraphEdgeSpec{
			{ID: "edge-1", Source: "sender", SourceHandle: "rx", Target: "script", TargetHandle: "in"},
			{ID: "edge-2", Source: "script", SourceHandle: "out", Target: "receiver", TargetHandle: "tx"},
		},
	})

	if _, err := svc.SendSerialGraphNode(SerialGraphSendRequest{GraphID: graph.ID, NodeID: "sender", Content: "x", Mode: "ascii"}); err != nil {
		t.Fatalf("SendSerialGraphNode returned error: %v", err)
	}

	info, err := svc.GetSerialGraphStatus(graph.ID)
	if err != nil {
		t.Fatalf("GetSerialGraphStatus returned error: %v", err)
	}
	status := graphNodeStatus(info, "script")
	if status.Status != SerialGraphStatusError || !strings.Contains(strings.ToLower(status.Error), "timeout") {
		t.Fatalf("script status=%q error=%q, want timeout error", status.Status, status.Error)
	}

	stopped := make(chan error, 1)
	go func() {
		stopped <- svc.StopSerialGraph(graph.ID)
	}()
	select {
	case err := <-stopped:
		if err != nil {
			t.Fatalf("StopSerialGraph returned error: %v", err)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatalf("StopSerialGraph blocked after script timeout")
	}

	if _, err := svc.GetSerialGraphStatus(graph.ID); err == nil {
		t.Fatalf("graph should not remain listed after stop")
	}
}

func TestSerialGraphRuntimeRejectsScriptGeneratorInput(t *testing.T) {
	svc := NewService(nil)
	defer svc.cleanup()

	_, err := svc.StartSerialGraph(context.Background(), SerialGraphStartRequest{
		ID: "graph-script-generator-input",
		Nodes: []SerialGraphNodeSpec{
			{ID: "sender", Type: "serial.virtual"},
			{ID: "gen", Type: "serial.script.generator"},
		},
		Edges: []SerialGraphEdgeSpec{
			{ID: "edge-1", Source: "sender", SourceHandle: "rx", Target: "gen", TargetHandle: "in"},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "input port not found: serial.script.generator.in") {
		t.Fatalf("StartSerialGraph error = %v, want generator input port validation", err)
	}
}
