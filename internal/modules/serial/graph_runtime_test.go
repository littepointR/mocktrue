package serial

import (
	"context"
	"fmt"
	"net"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/littepointR/portweave/internal/modules/serial/buffer"
	fb "github.com/littepointR/portweave/internal/modules/serial/fecbus"
	mb "github.com/littepointR/portweave/internal/modules/serial/modbus"
	"github.com/littepointR/portweave/internal/modules/serial/port"
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

func TestSerialGraphRuntimeRemoteRawTCPValidation(t *testing.T) {
	remote := func(id string, config map[string]any) SerialGraphNodeSpec {
		return SerialGraphNodeSpec{ID: id, Type: "serial.remote", Config: config}
	}

	valid := SerialGraphStartRequest{
		ID: "graph-remote-valid",
		Nodes: []SerialGraphNodeSpec{
			remote("remote", graphRemoteConfig("127.0.0.1", 3001)),
			{ID: "receiver", Type: "serial.receiver"},
		},
		Edges: []SerialGraphEdgeSpec{{ID: "edge-remote-receiver", Source: "remote", SourceHandle: "rx", Target: "receiver", TargetHandle: "in"}},
	}
	if errs := validateSerialGraphRuntimeRequest(valid); len(errs) != 0 {
		t.Fatalf("valid remote graph errors = %#v", errs)
	}

	cases := []struct {
		name      string
		config    map[string]any
		wantError string
	}{
		{name: "missing host", config: graphRemoteConfig("", 3001), wantError: "host required"},
		{name: "url host", config: graphRemoteConfig("tcp://127.0.0.1", 3001), wantError: "host must not include URL scheme"},
		{name: "bad port", config: graphRemoteConfig("127.0.0.1", 70000), wantError: "port out of range"},
		{name: "fractional port", config: graphRemoteConfig("127.0.0.1", 3001, map[string]any{"port": 3.14}), wantError: "port must be an integer"},
		{name: "bad string port", config: graphRemoteConfig("127.0.0.1", 3001, map[string]any{"port": "3001x"}), wantError: "port must be an integer"},
		{name: "null timeout", config: graphRemoteConfig("127.0.0.1", 3001, map[string]any{"connectTimeoutMs": nil}), wantError: "connectTimeoutMs must be an integer"},
		{name: "zero connect timeout", config: graphRemoteConfig("127.0.0.1", 3001, map[string]any{"connectTimeoutMs": 0}), wantError: "connectTimeoutMs must be positive"},
		{name: "short connect timeout", config: graphRemoteConfig("127.0.0.1", 3001, map[string]any{"connectTimeoutMs": 99}), wantError: "connectTimeoutMs out of range"},
		{name: "long connect timeout", config: graphRemoteConfig("127.0.0.1", 3001, map[string]any{"connectTimeoutMs": 60001}), wantError: "connectTimeoutMs out of range"},
		{name: "bad write timeout", config: graphRemoteConfig("127.0.0.1", 3001, map[string]any{"writeTimeoutMs": "slow"}), wantError: "writeTimeoutMs must be an integer"},
		{name: "zero write timeout", config: graphRemoteConfig("127.0.0.1", 3001, map[string]any{"writeTimeoutMs": 0}), wantError: "writeTimeoutMs must be positive"},
		{name: "short write timeout", config: graphRemoteConfig("127.0.0.1", 3001, map[string]any{"writeTimeoutMs": 99}), wantError: "writeTimeoutMs out of range"},
		{name: "long write timeout", config: graphRemoteConfig("127.0.0.1", 3001, map[string]any{"writeTimeoutMs": 60001}), wantError: "writeTimeoutMs out of range"},
		{name: "bad reconnect interval", config: graphRemoteConfig("127.0.0.1", 3001, map[string]any{"reconnectIntervalMs": nil}), wantError: "reconnectIntervalMs must be an integer"},
		{name: "zero reconnect interval", config: graphRemoteConfig("127.0.0.1", 3001, map[string]any{"reconnectIntervalMs": 0}), wantError: "reconnectIntervalMs must be positive"},
		{name: "short reconnect interval", config: graphRemoteConfig("127.0.0.1", 3001, map[string]any{"reconnectIntervalMs": 99}), wantError: "reconnectIntervalMs out of range"},
		{name: "long reconnect interval", config: graphRemoteConfig("127.0.0.1", 3001, map[string]any{"reconnectIntervalMs": 60001}), wantError: "reconnectIntervalMs out of range"},
		{name: "bad read buffer", config: graphRemoteConfig("127.0.0.1", 3001, map[string]any{"readBufKB": "big"}), wantError: "readBufKB must be an integer"},
		{name: "zero read buffer", config: graphRemoteConfig("127.0.0.1", 3001, map[string]any{"readBufKB": 0}), wantError: "readBufKB must be positive"},
		{name: "large read buffer", config: graphRemoteConfig("127.0.0.1", 3001, map[string]any{"readBufKB": 2048}), wantError: "readBufKB out of range"},
		{name: "unsupported protocol", config: graphRemoteConfig("127.0.0.1", 3001, map[string]any{"protocol": "rfc2217"}), wantError: "unsupported protocol"},
		{name: "unsupported role", config: graphRemoteConfig("127.0.0.1", 3001, map[string]any{"role": "server"}), wantError: "unsupported role"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			errs := validateSerialGraphRuntimeRequest(SerialGraphStartRequest{ID: "graph-remote-invalid", Nodes: []SerialGraphNodeSpec{remote("remote", tc.config)}})
			if !strings.Contains(strings.Join(errs, "\n"), tc.wantError) {
				t.Fatalf("errors = %#v, want %q", errs, tc.wantError)
			}
		})
	}

	duplicate := validateSerialGraphRuntimeRequest(SerialGraphStartRequest{
		ID: "graph-remote-duplicate",
		Nodes: []SerialGraphNodeSpec{
			remote("remote-a", graphRemoteConfig("127.0.0.1", 3001)),
			remote("remote-b", graphRemoteConfig("127.0.0.1", 3001)),
		},
	})
	if !strings.Contains(strings.Join(duplicate, "\n"), "resource remote endpoint duplicated") {
		t.Fatalf("duplicate remote endpoint errors = %#v", duplicate)
	}

	disjoint := validateSerialGraphRuntimeRequest(SerialGraphStartRequest{
		ID: "graph-remote-disjoint",
		Nodes: []SerialGraphNodeSpec{
			remote("remote-a", graphRemoteConfig("127.0.0.1", 3001)),
			remote("remote-b", graphRemoteConfig("127.0.0.1", 3002)),
			remote("remote-c", graphRemoteConfig("localhost", 3001)),
		},
	})
	if len(disjoint) != 0 {
		t.Fatalf("disjoint remote endpoints errors = %#v", disjoint)
	}
}

func TestSerialGraphRuntimeRemoteRawTCPReadWrite(t *testing.T) {
	tcpServer := newGraphTCPTestServer(t)
	host, port := tcpServer.hostPort(t)
	svc := NewService(nil)
	defer svc.cleanup()

	graph := startTestGraph(t, svc, SerialGraphStartRequest{
		ID: "graph-remote-raw-tcp",
		Nodes: []SerialGraphNodeSpec{
			{ID: "sender", Type: "serial.sender", Config: graphSenderConfig()},
			{ID: "remote", Type: "serial.remote", Config: graphRemoteConfig(host, port)},
			{ID: "receiver", Type: "serial.receiver", Config: graphReceiverConfig()},
		},
		Edges: []SerialGraphEdgeSpec{
			{ID: "edge-sender-remote", Source: "sender", SourceHandle: "out", Target: "remote", TargetHandle: "tx"},
			{ID: "edge-remote-receiver", Source: "remote", SourceHandle: "rx", Target: "receiver", TargetHandle: "in"},
		},
	})
	defer func() { _ = svc.StopSerialGraph(graph.ID) }()

	conn := tcpServer.waitConn(t)
	inbound := []byte("device->graph")
	_ = conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
	if _, err := conn.Write(inbound); err != nil {
		t.Fatalf("server write returned error: %v", err)
	}
	page := waitGraphBuffer(t, svc, graph.ID, "receiver", len(inbound))
	if string(page.Data) != string(inbound) {
		t.Fatalf("receiver data = %q, want %q", string(page.Data), string(inbound))
	}

	outbound := "graph->device"
	if _, err := svc.SendSerialGraphNode(SerialGraphSendRequest{GraphID: graph.ID, NodeID: "sender", Content: outbound, Mode: "ascii"}); err != nil {
		t.Fatalf("SendSerialGraphNode returned error: %v", err)
	}
	if got := string(tcpServer.waitReceived(t)); got != outbound {
		t.Fatalf("server received = %q, want %q", got, outbound)
	}

	info, err := svc.GetSerialGraphStatus(graph.ID)
	if err != nil {
		t.Fatalf("GetSerialGraphStatus returned error: %v", err)
	}
	remote := graphNodeStatus(info, "remote")
	if remote.Status != SerialGraphStatusRunning || remote.RxBytes != int64(len(inbound)) || remote.TxBytes != int64(len(outbound)) {
		t.Fatalf("remote status = %+v, want running rx/tx counters", remote)
	}
	if remote.ResourceID != "raw-tcp://"+net.JoinHostPort(host, strconv.Itoa(port)) {
		t.Fatalf("remote resource = %q", remote.ResourceID)
	}
}

func TestSerialGraphRuntimeRemoteSenderWriteFailureReturnsError(t *testing.T) {
	svc := NewService(nil)
	defer svc.cleanup()

	graph := startTestGraph(t, svc, SerialGraphStartRequest{
		ID: "graph-remote-disconnected-write",
		Nodes: []SerialGraphNodeSpec{
			{ID: "sender", Type: "serial.sender", Config: graphSenderConfig()},
			{ID: "remote", Type: "serial.remote", Config: graphRemoteConfig("127.0.0.1", 1, map[string]any{"allowStartDisconnected": true, "reconnect": false})},
		},
		Edges: []SerialGraphEdgeSpec{{ID: "edge-sender-remote", Source: "sender", SourceHandle: "out", Target: "remote", TargetHandle: "tx"}},
	})
	defer func() { _ = svc.StopSerialGraph(graph.ID) }()

	if _, err := svc.SendSerialGraphNode(SerialGraphSendRequest{GraphID: graph.ID, NodeID: "sender", Content: "fails", Mode: "ascii"}); err == nil {
		t.Fatalf("SendSerialGraphNode(sender -> disconnected remote) must return a write error")
	}
	remote := waitGraphNodeStatus(t, svc, graph.ID, "remote", SerialGraphStatusError)
	if remote.Error == "" {
		t.Fatalf("remote error message is empty after disconnected write")
	}
}

func TestSerialGraphRuntimeRemoteReadFailureWithoutReconnectMarksError(t *testing.T) {
	tcpServer := newGraphTCPTestServer(t)
	host, port := tcpServer.hostPort(t)
	svc := NewService(nil)
	defer svc.cleanup()

	graph := startTestGraph(t, svc, SerialGraphStartRequest{
		ID: "graph-remote-read-failure-no-reconnect",
		Nodes: []SerialGraphNodeSpec{
			{ID: "remote", Type: "serial.remote", Config: graphRemoteConfig(host, port, map[string]any{"reconnect": false})},
		},
	})
	defer func() { _ = svc.StopSerialGraph(graph.ID) }()

	conn := tcpServer.waitConn(t)
	if err := conn.Close(); err != nil {
		t.Fatalf("server close returned error: %v", err)
	}

	remote := waitGraphNodeStatus(t, svc, graph.ID, "remote", SerialGraphStatusError)
	if remote.Error == "" {
		t.Fatalf("remote error message is empty after read failure")
	}
	select {
	case extra := <-tcpServer.accepted:
		_ = extra.Close()
		t.Fatalf("remote reconnected despite reconnect=false")
	case <-time.After(150 * time.Millisecond):
	}
}

func TestSerialGraphRuntimeRemoteReconnectsAfterInitialDialFailure(t *testing.T) {
	host, port := reserveGraphTCPPort(t)
	svc := NewService(nil)
	defer svc.cleanup()

	graph := startTestGraph(t, svc, SerialGraphStartRequest{
		ID: "graph-remote-reconnect-after-start",
		Nodes: []SerialGraphNodeSpec{
			{ID: "sender", Type: "serial.sender", Config: graphSenderConfig()},
			{ID: "remote", Type: "serial.remote", Config: graphRemoteConfig(host, port, map[string]any{
				"allowStartDisconnected": true,
				"reconnect":              true,
				"connectTimeoutMs":       100,
				"reconnectIntervalMs":    100,
			})},
			{ID: "receiver", Type: "serial.receiver", Config: graphReceiverConfig()},
		},
		Edges: []SerialGraphEdgeSpec{
			{ID: "edge-sender-remote", Source: "sender", SourceHandle: "out", Target: "remote", TargetHandle: "tx"},
			{ID: "edge-remote-receiver", Source: "remote", SourceHandle: "rx", Target: "receiver", TargetHandle: "in"},
		},
	})
	defer func() { _ = svc.StopSerialGraph(graph.ID) }()

	listener, err := net.Listen("tcp", net.JoinHostPort(host, strconv.Itoa(port)))
	if err != nil {
		t.Fatalf("net.Listen after disconnected start returned error: %v", err)
	}
	server := newGraphTCPTestServerOnListener(t, listener)
	conn := server.waitConn(t)

	inbound := []byte("after-reconnect")
	if _, err := conn.Write(inbound); err != nil {
		t.Fatalf("server write after reconnect returned error: %v", err)
	}
	page := waitGraphBuffer(t, svc, graph.ID, "receiver", len(inbound))
	if string(page.Data) != string(inbound) {
		t.Fatalf("receiver data after reconnect = %q, want %q", string(page.Data), string(inbound))
	}

	outbound := "client-write-after-reconnect"
	if _, err := svc.SendSerialGraphNode(SerialGraphSendRequest{GraphID: graph.ID, NodeID: "sender", Content: outbound, Mode: "ascii"}); err != nil {
		t.Fatalf("SendSerialGraphNode after reconnect returned error: %v", err)
	}
	if got := string(server.waitReceived(t)); got != outbound {
		t.Fatalf("server received after reconnect = %q, want %q", got, outbound)
	}

	info, err := svc.GetSerialGraphStatus(graph.ID)
	if err != nil {
		t.Fatalf("GetSerialGraphStatus returned error: %v", err)
	}
	remote := graphNodeStatus(info, "remote")
	if remote.Status != SerialGraphStatusRunning || remote.RxBytes != int64(len(inbound)) || remote.TxBytes != int64(len(outbound)) {
		t.Fatalf("remote status after reconnect = %+v, want running rx/tx counters", remote)
	}
}

func TestSerialGraphRuntimeRemoteStartFailsWhenInitialDialUnavailable(t *testing.T) {
	host, port := reserveGraphTCPPort(t)
	svc := NewService(nil)
	defer svc.cleanup()

	_, err := svc.StartSerialGraph(context.Background(), SerialGraphStartRequest{
		ID: "graph-remote-start-fails",
		Nodes: []SerialGraphNodeSpec{
			{ID: "remote", Type: "serial.remote", Config: graphRemoteConfig(host, port, map[string]any{"connectTimeoutMs": 100})},
		},
	})
	if err == nil {
		t.Fatalf("StartSerialGraph with unavailable remote endpoint returned nil error")
	}
	if !strings.Contains(err.Error(), "start graph remote node remote") {
		t.Fatalf("StartSerialGraph error = %v, want remote start context", err)
	}
	if graphs := svc.ListSerialGraphs(); len(graphs) != 0 {
		t.Fatalf("running graphs after failed remote start = %#v, want none", graphs)
	}
}

func TestSerialGraphRuntimeRemoteDirectSendWritesTCP(t *testing.T) {
	tcpServer := newGraphTCPTestServer(t)
	host, port := tcpServer.hostPort(t)
	svc := NewService(nil)
	defer svc.cleanup()

	graph := startTestGraph(t, svc, SerialGraphStartRequest{
		ID: "graph-remote-direct-send",
		Nodes: []SerialGraphNodeSpec{
			{ID: "remote", Type: "serial.remote", Config: graphRemoteConfig(host, port)},
		},
	})
	defer func() { _ = svc.StopSerialGraph(graph.ID) }()
	tcpServer.waitConn(t)

	payload := "direct->device"
	if written, err := svc.SendSerialGraphNode(SerialGraphSendRequest{GraphID: graph.ID, NodeID: "remote", Content: payload, Mode: "ascii"}); err != nil || written != len(payload) {
		t.Fatalf("SendSerialGraphNode(remote direct) = %d, %v; want %d, nil", written, err, len(payload))
	}
	if got := string(tcpServer.waitReceived(t)); got != payload {
		t.Fatalf("server received direct send = %q, want %q", got, payload)
	}

	info, err := svc.GetSerialGraphStatus(graph.ID)
	if err != nil {
		t.Fatalf("GetSerialGraphStatus returned error: %v", err)
	}
	remote := graphNodeStatus(info, "remote")
	if remote.Status != SerialGraphStatusRunning || remote.TxBytes != int64(len(payload)) {
		t.Fatalf("remote status after direct send = %+v, want running tx counter", remote)
	}
}

func TestSerialGraphRuntimeRemoteReconnectsAfterConnectedDrop(t *testing.T) {
	tcpServer := newGraphTCPTestServer(t)
	host, port := tcpServer.hostPort(t)
	svc := NewService(nil)
	defer svc.cleanup()

	graph := startTestGraph(t, svc, SerialGraphStartRequest{
		ID: "graph-remote-reconnect-after-drop",
		Nodes: []SerialGraphNodeSpec{
			{ID: "sender", Type: "serial.sender", Config: graphSenderConfig()},
			{ID: "remote", Type: "serial.remote", Config: graphRemoteConfig(host, port, map[string]any{
				"reconnect":           true,
				"connectTimeoutMs":    100,
				"reconnectIntervalMs": 100,
			})},
			{ID: "receiver", Type: "serial.receiver", Config: graphReceiverConfig()},
		},
		Edges: []SerialGraphEdgeSpec{
			{ID: "edge-sender-remote", Source: "sender", SourceHandle: "out", Target: "remote", TargetHandle: "tx"},
			{ID: "edge-remote-receiver", Source: "remote", SourceHandle: "rx", Target: "receiver", TargetHandle: "in"},
		},
	})
	defer func() { _ = svc.StopSerialGraph(graph.ID) }()

	first := tcpServer.waitConn(t)
	if err := first.Close(); err != nil {
		t.Fatalf("first server conn close returned error: %v", err)
	}
	second := tcpServer.waitConn(t)

	inbound := []byte("after-connected-drop")
	if _, err := second.Write(inbound); err != nil {
		t.Fatalf("server write after connected reconnect returned error: %v", err)
	}
	page := waitGraphBuffer(t, svc, graph.ID, "receiver", len(inbound))
	if string(page.Data) != string(inbound) {
		t.Fatalf("receiver data after connected reconnect = %q, want %q", string(page.Data), string(inbound))
	}

	outbound := "client-write-after-drop"
	if _, err := svc.SendSerialGraphNode(SerialGraphSendRequest{GraphID: graph.ID, NodeID: "sender", Content: outbound, Mode: "ascii"}); err != nil {
		t.Fatalf("SendSerialGraphNode after connected reconnect returned error: %v", err)
	}
	if got := string(tcpServer.waitReceived(t)); got != outbound {
		t.Fatalf("server received after connected reconnect = %q, want %q", got, outbound)
	}
}

func TestSerialGraphRuntimeRemoteConnectedDropClearsConnBeforeReconnect(t *testing.T) {
	tcpServer := newGraphTCPTestServer(t)
	host, port := tcpServer.hostPort(t)
	svc := NewService(nil)
	defer svc.cleanup()

	graph := startTestGraph(t, svc, SerialGraphStartRequest{
		ID: "graph-remote-clears-conn-before-reconnect",
		Nodes: []SerialGraphNodeSpec{
			{ID: "sender", Type: "serial.sender", Config: graphSenderConfig()},
			{ID: "remote", Type: "serial.remote", Config: graphRemoteConfig(host, port, map[string]any{
				"reconnect":           true,
				"connectTimeoutMs":    100,
				"reconnectIntervalMs": 100,
			})},
		},
		Edges: []SerialGraphEdgeSpec{{ID: "edge-sender-remote", Source: "sender", SourceHandle: "out", Target: "remote", TargetHandle: "tx"}},
	})
	defer func() { _ = svc.StopSerialGraph(graph.ID) }()

	first := tcpServer.waitConn(t)
	if err := tcpServer.listener.Close(); err != nil {
		t.Fatalf("close remote listener before reconnect returned error: %v", err)
	}
	if err := first.Close(); err != nil {
		t.Fatalf("first server conn close returned error: %v", err)
	}
	waitGraphNodeStatus(t, svc, graph.ID, "remote", SerialGraphStatusReconnecting)

	runtime, err := svc.serialGraphRuntime(graph.ID)
	if err != nil {
		t.Fatalf("serialGraphRuntime returned error: %v", err)
	}
	node := runtime.node("remote")
	deadline := time.Now().Add(500 * time.Millisecond)
	for node.remoteConnection() != nil && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if conn := node.remoteConnection(); conn != nil {
		t.Fatalf("remoteConnection() remained set while reconnecting after connected drop: %T", conn)
	}

	if written, err := svc.SendSerialGraphNode(SerialGraphSendRequest{GraphID: graph.ID, NodeID: "sender", Content: "during-reconnect", Mode: "ascii"}); err == nil || written != 0 || !strings.Contains(err.Error(), "not connected") {
		t.Fatalf("SendSerialGraphNode sender during reconnect = %d, %v; want disconnected error", written, err)
	}
	if written, err := svc.SendSerialGraphNode(SerialGraphSendRequest{GraphID: graph.ID, NodeID: "remote", Content: "direct-during-reconnect", Mode: "ascii"}); err == nil || written != 0 || !strings.Contains(err.Error(), "not connected") {
		t.Fatalf("SendSerialGraphNode direct remote during reconnect = %d, %v; want disconnected error", written, err)
	}
	info, err := svc.GetSerialGraphStatus(graph.ID)
	if err != nil {
		t.Fatalf("GetSerialGraphStatus returned error: %v", err)
	}
	if remote := graphNodeStatus(info, "remote"); remote.TxBytes != 0 {
		t.Fatalf("remote TxBytes after disconnected write = %d, want 0", remote.TxBytes)
	}
}

func TestSerialGraphRuntimeRemoteWriteFailureDoesNotCloseReplacementConn(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	runtime := &serialGraphRuntime{remoteCtx: ctx}
	node := &serialGraphRuntimeNode{
		spec: SerialGraphNodeSpec{ID: "remote", Type: "serial.remote", Config: graphRemoteConfig("127.0.0.1", 1, map[string]any{
			"reconnect":           true,
			"connectTimeoutMs":    100,
			"reconnectIntervalMs": 100,
		})},
		status: SerialGraphStatusRunning,
	}
	stale := &graphFailingConn{}
	replacement := &graphFailingConn{}
	stale.onWrite = func() {
		node.setRemoteConn(replacement)
		node.setStatus(SerialGraphStatusRunning, "")
	}
	node.setRemoteConn(stale)

	n, err := runtime.writeRemoteNode(node, []byte("stale-write"))
	if err == nil {
		t.Fatalf("writeRemoteNode with stale failing conn returned nil error")
	}
	if n != 0 {
		t.Fatalf("writeRemoteNode wrote %d bytes, want 0", n)
	}
	if got := node.remoteConnection(); got != replacement {
		t.Fatalf("remoteConnection() = %p, want replacement %p", got, replacement)
	}
	if replacement.isClosed() {
		t.Fatalf("replacement conn was closed by stale write failure cleanup")
	}
}

func TestSerialGraphRuntimeRemoteReconnectAllowsImmediateReplacementFailure(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	runtime := &serialGraphRuntime{remoteCtx: ctx}
	node := &serialGraphRuntimeNode{
		spec: SerialGraphNodeSpec{ID: "remote", Type: "serial.remote", Config: graphRemoteConfig("127.0.0.1", 1, map[string]any{
			"reconnect":           true,
			"connectTimeoutMs":    100,
			"reconnectIntervalMs": 10,
		})},
		status: SerialGraphStatusReconnecting,
	}
	if !node.tryBeginRemoteReconnect() {
		t.Fatalf("tryBeginRemoteReconnect returned false for idle node")
	}
	replacement := newGraphBlockingFailingConn()
	defer func() { _ = replacement.Close() }()
	runtime.completeRemoteReconnect(ctx, node, replacement)
	if got := node.remoteConnection(); got != replacement {
		t.Fatalf("remoteConnection() = %p, want replacement %p", got, replacement)
	}

	if written, err := runtime.writeRemoteNode(node, []byte("immediate-failure")); err == nil || written != 0 || !strings.Contains(err.Error(), "graph blocking failing conn write") {
		t.Fatalf("writeRemoteNode immediate replacement failure = %d, %v; want replacement write error", written, err)
	}

	deadline := time.Now().Add(500 * time.Millisecond)
	var lastErr string
	for time.Now().Before(deadline) {
		node.mu.RLock()
		lastErr = node.err
		node.mu.RUnlock()
		if lastErr != "" && !strings.Contains(lastErr, "graph blocking failing conn write") {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("remote reconnect did not retry after immediate replacement failure; last error = %q", lastErr)
}

func TestSerialGraphRuntimeRemoteStopCancelsDisconnectedReconnect(t *testing.T) {
	host, port := reserveGraphTCPPort(t)
	svc := NewService(nil)
	defer svc.cleanup()

	graph := startTestGraph(t, svc, SerialGraphStartRequest{
		ID: "graph-remote-stop-disconnected-reconnect",
		Nodes: []SerialGraphNodeSpec{
			{ID: "remote", Type: "serial.remote", Config: graphRemoteConfig(host, port, map[string]any{
				"allowStartDisconnected": true,
				"reconnect":              true,
				"connectTimeoutMs":       100,
				"reconnectIntervalMs":    100,
			})},
		},
	})
	waitGraphNodeStatus(t, svc, graph.ID, "remote", SerialGraphStatusReconnecting)
	if err := svc.StopSerialGraph(graph.ID); err != nil {
		t.Fatalf("StopSerialGraph returned error: %v", err)
	}

	listener, err := net.Listen("tcp", net.JoinHostPort(host, strconv.Itoa(port)))
	if err != nil {
		t.Fatalf("net.Listen after stop returned error: %v", err)
	}
	server := newGraphTCPTestServerOnListener(t, listener)
	select {
	case conn := <-server.accepted:
		_ = conn.Close()
		t.Fatalf("remote reconnected after StopSerialGraph")
	case <-time.After(250 * time.Millisecond):
	}
}

func TestSerialGraphRuntimeRemoteConfigNumericCoercion(t *testing.T) {
	validConfig := graphRemoteConfig("LOCALHOST", 3001, map[string]any{
		"port":                int64(3001),
		"connectTimeoutMs":    float32(250),
		"writeTimeoutMs":      float64(250),
		"reconnectIntervalMs": "250",
		"readBufKB":           "16",
	})
	if errs := validateSerialGraphRuntimeRequest(SerialGraphStartRequest{ID: "graph-remote-numeric", Nodes: []SerialGraphNodeSpec{{ID: "remote", Type: "serial.remote", Config: validConfig}}}); len(errs) != 0 {
		t.Fatalf("numeric coercion config errors = %#v", errs)
	}
	_, _, endpoint, resourceID, err := graphRemoteEndpoint(validConfig)
	if err != nil {
		t.Fatalf("graphRemoteEndpoint returned error: %v", err)
	}
	if endpoint != "localhost:3001" || resourceID != "raw-tcp://localhost:3001" {
		t.Fatalf("endpoint=%q resourceID=%q", endpoint, resourceID)
	}

	for _, tc := range []struct {
		name   string
		config map[string]any
	}{
		{name: "nil port", config: graphRemoteConfig("127.0.0.1", 3001, map[string]any{"port": nil})},
		{name: "boolean port", config: graphRemoteConfig("127.0.0.1", 3001, map[string]any{"port": true})},
		{name: "fractional connect timeout", config: graphRemoteConfig("127.0.0.1", 3001, map[string]any{"connectTimeoutMs": 1.5})},
		{name: "missing host", config: graphRemoteConfig("", 3001)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, _, _, _, err := graphRemoteEndpoint(tc.config); err == nil {
				t.Fatalf("graphRemoteEndpoint(%s) returned nil error", tc.name)
			}
		})
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

func TestSerialGraphRuntimeFilterForwardsPlainMatchesAndDropsMismatches(t *testing.T) {
	svc := NewService(nil)
	defer svc.cleanup()

	graph := startTestGraph(t, svc, SerialGraphStartRequest{
		ID: "graph-filter-plain",
		Nodes: []SerialGraphNodeSpec{
			{ID: "sender", Type: "serial.sender"},
			{ID: "filter", Type: "serial.filter", Config: map[string]any{"mode": "plain", "expression": "OK"}},
			{ID: "receiver", Type: "serial.receiver"},
		},
		Edges: []SerialGraphEdgeSpec{
			{ID: "edge-sender-filter", Source: "sender", SourceHandle: "out", Target: "filter", TargetHandle: "in"},
			{ID: "edge-filter-receiver", Source: "filter", SourceHandle: "out", Target: "receiver", TargetHandle: "in"},
		},
	})

	if _, err := svc.SendSerialGraphNode(SerialGraphSendRequest{GraphID: graph.ID, NodeID: "sender", Content: "ok telemetry", Mode: "ascii"}); err != nil {
		t.Fatalf("SendSerialGraphNode(match) returned error: %v", err)
	}
	if _, err := svc.SendSerialGraphNode(SerialGraphSendRequest{GraphID: graph.ID, NodeID: "sender", Content: "drop telemetry", Mode: "ascii"}); err != nil {
		t.Fatalf("SendSerialGraphNode(drop) returned error: %v", err)
	}

	page, err := svc.QuerySerialGraphNodeBuffer(SerialGraphBufferQuery{GraphID: graph.ID, NodeID: "receiver", Offset: 0, Length: 64})
	if err != nil {
		t.Fatalf("QuerySerialGraphNodeBuffer returned error: %v", err)
	}
	if string(page.Data) != "ok telemetry" {
		t.Fatalf("receiver data = %q, want only matching payload", string(page.Data))
	}
	info, err := svc.GetSerialGraphStatus(graph.ID)
	if err != nil {
		t.Fatalf("GetSerialGraphStatus returned error: %v", err)
	}
	filter := graphNodeStatus(info, "filter")
	if filter.RxBytes != int64(len("ok telemetry")+len("drop telemetry")) || filter.TxBytes != int64(len("ok telemetry")) {
		t.Fatalf("filter rx/tx = %d/%d, want all input counted and only matching bytes forwarded", filter.RxBytes, filter.TxBytes)
	}
}

func TestSerialGraphRuntimeFilterHonorsCaseSensitiveWholeWordRegexAndExpression(t *testing.T) {
	tests := []struct {
		name       string
		config     map[string]any
		sends      []SerialGraphSendRequest
		wantBuffer string
	}{
		{
			name:   "case sensitive whole word plain filter",
			config: map[string]any{"mode": "plain", "expression": "ok", "caseSensitive": true, "wholeWord": true},
			sends: []SerialGraphSendRequest{
				{Content: "token", Mode: "ascii"},
				{Content: "OK", Mode: "ascii"},
				{Content: "ok", Mode: "ascii"},
			},
			wantBuffer: "ok",
		},
		{
			name:       "case insensitive regex filter",
			config:     map[string]any{"mode": "regex", "expression": `temp=\d+`},
			sends:      []SerialGraphSendRequest{{Content: "TEMP=42", Mode: "ascii"}, {Content: "humidity=10", Mode: "ascii"}},
			wantBuffer: "TEMP=42",
		},
		{
			name:       "wireshark-like expression filter",
			config:     map[string]any{"mode": "expression", "expression": `len >= 4 and hex contains "0d0a"`},
			sends:      []SerialGraphSendRequest{{Content: "00 ff", Mode: "hex"}, {Content: "0d 0a ff aa", Mode: "hex"}},
			wantBuffer: string([]byte{0x0d, 0x0a, 0xff, 0xaa}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(nil)
			defer svc.cleanup()
			graph := startTestGraph(t, svc, SerialGraphStartRequest{
				ID: "graph-filter-" + strings.ReplaceAll(tt.name, " ", "-"),
				Nodes: []SerialGraphNodeSpec{
					{ID: "sender", Type: "serial.sender"},
					{ID: "filter", Type: "serial.filter", Config: tt.config},
					{ID: "receiver", Type: "serial.receiver"},
				},
				Edges: []SerialGraphEdgeSpec{
					{ID: "edge-sender-filter", Source: "sender", SourceHandle: "out", Target: "filter", TargetHandle: "in"},
					{ID: "edge-filter-receiver", Source: "filter", SourceHandle: "out", Target: "receiver", TargetHandle: "in"},
				},
			})
			for _, send := range tt.sends {
				send.GraphID = graph.ID
				send.NodeID = "sender"
				if _, err := svc.SendSerialGraphNode(send); err != nil {
					t.Fatalf("SendSerialGraphNode(%q) returned error: %v", send.Content, err)
				}
			}
			page, err := svc.QuerySerialGraphNodeBuffer(SerialGraphBufferQuery{GraphID: graph.ID, NodeID: "receiver", Offset: 0, Length: 64})
			if err != nil {
				t.Fatalf("QuerySerialGraphNodeBuffer returned error: %v", err)
			}
			if string(page.Data) != tt.wantBuffer {
				t.Fatalf("receiver data = %q (%s), want %q (%s)", string(page.Data), formatHexBytes(page.Data), tt.wantBuffer, formatHexBytes([]byte(tt.wantBuffer)))
			}
		})
	}
}

func TestSerialGraphRuntimeFilterInvalidExpressionMarksErrorAndDrops(t *testing.T) {
	svc := NewService(nil)
	defer svc.cleanup()

	graph := startTestGraph(t, svc, SerialGraphStartRequest{
		ID: "graph-filter-invalid",
		Nodes: []SerialGraphNodeSpec{
			{ID: "sender", Type: "serial.sender"},
			{ID: "filter", Type: "serial.filter", Config: map[string]any{"mode": "regex", "expression": "[unterminated"}},
			{ID: "receiver", Type: "serial.receiver"},
		},
		Edges: []SerialGraphEdgeSpec{
			{ID: "edge-sender-filter", Source: "sender", SourceHandle: "out", Target: "filter", TargetHandle: "in"},
			{ID: "edge-filter-receiver", Source: "filter", SourceHandle: "out", Target: "receiver", TargetHandle: "in"},
		},
	})

	if _, err := svc.SendSerialGraphNode(SerialGraphSendRequest{GraphID: graph.ID, NodeID: "sender", Content: "anything", Mode: "ascii"}); err != nil {
		t.Fatalf("SendSerialGraphNode returned error: %v", err)
	}
	page, err := svc.QuerySerialGraphNodeBuffer(SerialGraphBufferQuery{GraphID: graph.ID, NodeID: "receiver", Offset: 0, Length: 64})
	if err != nil {
		t.Fatalf("QuerySerialGraphNodeBuffer returned error: %v", err)
	}
	if page.Total != 0 || len(page.Data) != 0 {
		t.Fatalf("receiver buffer total=%d len=%d, want invalid filter to drop", page.Total, len(page.Data))
	}
	info, err := svc.GetSerialGraphStatus(graph.ID)
	if err != nil {
		t.Fatalf("GetSerialGraphStatus returned error: %v", err)
	}
	filter := graphNodeStatus(info, "filter")
	if filter.Status != SerialGraphStatusError || !strings.Contains(filter.Error, "invalid regex") {
		t.Fatalf("filter status=%q error=%q, want invalid regex error", filter.Status, filter.Error)
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

func TestSerialGraphRuntimeMonitorPopulatesDisplayModes(t *testing.T) {
	svc := NewService(nil)
	defer svc.cleanup()

	graph := startTestGraph(t, svc, SerialGraphStartRequest{
		ID: "graph-monitor-display",
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
		Content: "41 00 ff",
		Mode:    "hex",
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
	frame := page.Frames[0]
	if frame.DisplayHex != "41 00 ff" || frame.DisplayDec != "65 0 255" || frame.DisplayOct != "101 000 377" || frame.DisplayBin != "01000001 00000000 11111111" {
		t.Fatalf("frame displays = hex=%q dec=%q oct=%q bin=%q", frame.DisplayHex, frame.DisplayDec, frame.DisplayOct, frame.DisplayBin)
	}
}

func TestSerialGraphRuntimeRejectsMonitorGraphOutputs(t *testing.T) {
	svc := NewService(nil)
	defer svc.cleanup()

	_, err := svc.StartSerialGraph(context.Background(), SerialGraphStartRequest{
		ID: "graph-monitor-output",
		Nodes: []SerialGraphNodeSpec{
			{ID: "monitor", Type: "serial.monitor"},
			{ID: "receiver", Type: "serial.receiver"},
		},
		Edges: []SerialGraphEdgeSpec{
			{ID: "edge-monitor-frame", Source: "monitor", SourceHandle: "frames", Target: "receiver", TargetHandle: "in"},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "output port not found: serial.monitor.frames") {
		t.Fatalf("StartSerialGraph error = %v, want monitor output port validation", err)
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

func TestSerialGraphModbusMasterFrameBuildsMultiWriteRequests(t *testing.T) {
	tests := []struct {
		name     string
		config   map[string]any
		function mb.FunctionCode
		data     []byte
	}{
		{
			name: "multiple coils from string values",
			config: map[string]any{
				"mode":         "rtu",
				"unitIds":      "7",
				"functionCode": int(mb.FunctionWriteMultipleCoils),
				"address":      10,
				"coilValues":   "1, 0 true off 1",
			},
			function: mb.FunctionWriteMultipleCoils,
			data:     []byte{0x00, 0x0a, 0x00, 0x05, 0x01, 0x15},
		},
		{
			name: "multiple registers from array values",
			config: map[string]any{
				"mode":           "rtu",
				"unitIds":        "9",
				"functionCode":   int(mb.FunctionWriteMultipleRegisters),
				"address":        11,
				"registerValues": []any{uint16(1), float64(258), "0x0103"},
			},
			function: mb.FunctionWriteMultipleRegisters,
			data:     []byte{0x00, 0x0b, 0x00, 0x03, 0x06, 0x00, 0x01, 0x01, 0x02, 0x01, 0x03},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded, err := graphModbusMasterFrame(tt.config)
			if err != nil {
				t.Fatalf("graphModbusMasterFrame returned error: %v", err)
			}
			frame, err := mb.DecodeFrame(mb.FrameModeRTU, encoded)
			if err != nil {
				t.Fatalf("DecodeFrame: %v", err)
			}
			if frame.PDU.Function != tt.function {
				t.Fatalf("function = %02x, want %02x", byte(frame.PDU.Function), byte(tt.function))
			}
			if got := formatGraphFrameBytes(frame.PDU.Data, "%02x"); got != formatGraphFrameBytes(tt.data, "%02x") {
				t.Fatalf("pdu data = %s, want %s", got, formatGraphFrameBytes(tt.data, "%02x"))
			}
		})
	}
}

func TestSerialGraphValueConfigParsesSupportedBoolShapes(t *testing.T) {
	if got := graphBoolValuesConfig(nil, "values"); got != nil {
		t.Fatalf("nil bool config = %#v, want nil", got)
	}

	bools := []bool{true, false}
	got := graphBoolValuesConfig(map[string]any{"values": bools}, "values")
	bools[0] = false
	if !reflect.DeepEqual(got, []bool{true, false}) {
		t.Fatalf("bool slice clone = %#v, want [true false]", got)
	}

	tests := []struct {
		name   string
		config map[string]any
		want   []bool
	}{
		{
			name:   "mixed any values skip invalid entries",
			config: map[string]any{"values": []any{true, 1, int64(0), float64(1), "off", "skip", 2}},
			want:   []bool{true, true, false, true, false},
		},
		{
			name:   "string slice tokens",
			config: map[string]any{"values": []string{"yes", "n", "bad", "on"}},
			want:   []bool{true, false, true},
		},
		{
			name:   "delimited string tokens",
			config: map[string]any{"values": "1; false on\nno"},
			want:   []bool{true, false, true, false},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := graphBoolValuesConfig(tt.config, "values")
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("graphBoolValuesConfig = %#v, want %#v", got, tt.want)
			}
		})
	}

	for _, value := range []any{int(2), int64(2), float64(0.5), struct{}{}} {
		if got, ok := graphBoolValue(value); ok || got {
			t.Fatalf("graphBoolValue(%T(%v)) = %v, %v; want false, false", value, value, got, ok)
		}
	}
}

func TestSerialGraphValueConfigParsesSupportedUint16Shapes(t *testing.T) {
	if got := graphUint16ValuesConfig(nil, "values"); got != nil {
		t.Fatalf("nil uint16 config = %#v, want nil", got)
	}

	uints := []uint16{1, 2}
	got := graphUint16ValuesConfig(map[string]any{"values": uints}, "values")
	uints[0] = 9
	if !reflect.DeepEqual(got, []uint16{1, 2}) {
		t.Fatalf("uint16 slice clone = %#v, want [1 2]", got)
	}

	tests := []struct {
		name   string
		config map[string]any
		want   []uint16
	}{
		{
			name:   "int slice skips out of range",
			config: map[string]any{"values": []int{-1, 0, 65535, 65536}},
			want:   []uint16{0, 65535},
		},
		{
			name:   "mixed any values skip invalid entries",
			config: map[string]any{"values": []any{uint16(1), uint(2), uint64(3), int(4), int64(5), float64(6), "0x7", float64(1.5), -1, "bad"}},
			want:   []uint16{1, 2, 3, 4, 5, 6, 7},
		},
		{
			name:   "string slice values",
			config: map[string]any{"values": []string{"8", "0x9", "bad", "65536"}},
			want:   []uint16{8, 9},
		},
		{
			name:   "delimited string values",
			config: map[string]any{"values": "10, 0x0b; bad\n12"},
			want:   []uint16{10, 11, 12},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := graphUint16ValuesConfig(tt.config, "values")
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("graphUint16ValuesConfig = %#v, want %#v", got, tt.want)
			}
		})
	}

	for _, value := range []any{uint(1 << 16), uint64(1 << 16), int(-1), int64(-1), float64(1.5), struct{}{}} {
		if got, ok := graphUint16Value(value); ok || got != 0 {
			t.Fatalf("graphUint16Value(%T(%v)) = %d, %v; want 0, false", value, value, got, ok)
		}
	}
}

func TestGraphModbusDisplayHelpersCoverFunctionsAndExceptionDetails(t *testing.T) {
	labels := map[mb.FunctionCode]string{
		mb.FunctionReadCoils:              "Read Coils",
		mb.FunctionReadDiscreteInputs:     "Read Discrete Inputs",
		mb.FunctionReadHoldingRegisters:   "Read Holding Registers",
		mb.FunctionReadInputRegisters:     "Read Input Registers",
		mb.FunctionWriteSingleCoil:        "Write Single Coil",
		mb.FunctionWriteSingleRegister:    "Write Single Register",
		mb.FunctionWriteMultipleCoils:     "Write Multiple Coils",
		mb.FunctionWriteMultipleRegisters: "Write Multiple Registers",
		mb.FunctionCode(0x7f):             "Unknown",
	}
	for function, want := range labels {
		if got := graphModbusFunctionLabel(function); got != want {
			t.Fatalf("graphModbusFunctionLabel(%02x) = %q, want %q", byte(function), got, want)
		}
	}

	exceptions := map[byte]string{
		mb.ExceptionIllegalFunction:    "Illegal Function",
		mb.ExceptionIllegalDataAddress: "Illegal Data Address",
		mb.ExceptionIllegalDataValue:   "Illegal Data Value",
		mb.ExceptionServerFailure:      "Server Failure",
		0xff:                           "Unknown",
	}
	for code, want := range exceptions {
		if got := graphModbusExceptionLabel(code); got != want {
			t.Fatalf("graphModbusExceptionLabel(%02x) = %q, want %q", code, got, want)
		}
	}

	tests := []struct {
		name string
		pdu  mb.PDU
		want string
	}{
		{name: "exception without data", pdu: mb.PDU{Function: mb.FunctionReadCoils | 0x80}, want: "Exception"},
		{name: "exception with data", pdu: mb.PDU{Function: mb.FunctionReadCoils | 0x80, Data: []byte{mb.ExceptionIllegalDataAddress}}, want: "Exception 02 Illegal Data Address"},
		{name: "read coils request", pdu: mb.PDU{Function: mb.FunctionReadCoils, Data: []byte{0, 10, 0, 3}}, want: "Address 10 Quantity 3"},
		{name: "read coils response", pdu: mb.PDU{Function: mb.FunctionReadDiscreteInputs, Data: []byte{1, 0xaa}}, want: "Byte Count 1 Data aa"},
		{name: "read registers response values", pdu: mb.PDU{Function: mb.FunctionReadHoldingRegisters, Data: []byte{4, 0, 1, 0, 2}}, want: "Byte Count 4 Values 1, 2"},
		{name: "read registers response odd data", pdu: mb.PDU{Function: mb.FunctionReadInputRegisters, Data: []byte{1, 0xff}}, want: "Byte Count 1 Data ff"},
		{name: "write single", pdu: mb.PDU{Function: mb.FunctionWriteSingleRegister, Data: []byte{0, 11, 0, 12}}, want: "Address 11 Value 12"},
		{name: "write multiple request", pdu: mb.PDU{Function: mb.FunctionWriteMultipleCoils, Data: []byte{0, 13, 0, 2}}, want: "Address 13 Quantity 2"},
		{name: "write multiple payload", pdu: mb.PDU{Function: mb.FunctionWriteMultipleRegisters, Data: []byte{0, 14, 0, 2, 4, 0, 1, 0, 2}}, want: "Address 14 Quantity 2 Byte Count 4 Data 00 01 00 02"},
		{name: "unknown with data", pdu: mb.PDU{Function: mb.FunctionCode(0x44), Data: []byte{0xaa}}, want: "Data aa"},
		{name: "unknown empty", pdu: mb.PDU{Function: mb.FunctionCode(0x44)}, want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := graphModbusPDUDisplay(tt.pdu); got != tt.want {
				t.Fatalf("graphModbusPDUDisplay = %q, want %q", got, tt.want)
			}
		})
	}

	if got := graphModbusChecksumDisplay(mb.FrameModeASCII, []byte(":01030000FC\r\n")); got != "LRC FC" {
		t.Fatalf("ASCII checksum display = %q, want LRC FC", got)
	}
	if got := graphModbusChecksumDisplay(mb.FrameModeASCII, []byte(":")); got != "" {
		t.Fatalf("short ASCII checksum display = %q, want empty", got)
	}
	if got := graphModbusChecksumDisplay(mb.FrameModeRTU, []byte{0x01}); got != "" {
		t.Fatalf("short RTU checksum display = %q, want empty", got)
	}
	if got := graphModbusU16([]byte{0x01}); got != 0 {
		t.Fatalf("short graphModbusU16 = %d, want 0", got)
	}
}

func TestSerialGraphRuntimeNodeResourceAndNilBufferBranches(t *testing.T) {
	node := &serialGraphRuntimeNode{}
	node.setResourceID("resource-port")
	if got := node.resource(); got != "resource-port" {
		t.Fatalf("resource = %q, want resource-port", got)
	}

	node.appendBuffer([]byte("ignored"))
	snapshot, err := node.queryBuffer(5, 10)
	if err != nil {
		t.Fatalf("queryBuffer returned error: %v", err)
	}
	if !snapshot.EOF || snapshot.Offset != 5 || len(snapshot.Data) != 0 {
		t.Fatalf("nil buffer snapshot = %#v, want EOF at requested offset", snapshot)
	}
}

func TestSerialGraphRuntimeFacadeValidationAndMissingResources(t *testing.T) {
	svc := &Service{}
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := svc.StartSerialGraph(canceled, SerialGraphStartRequest{}); err == nil {
		t.Fatalf("StartSerialGraph must respect canceled contexts")
	}

	graph, err := svc.StartSerialGraph(context.Background(), SerialGraphStartRequest{})
	if err != nil {
		t.Fatalf("StartSerialGraph with generated ID returned error: %v", err)
	}
	if graph.ID == "" {
		t.Fatalf("generated graph ID must not be empty")
	}
	if got := svc.ListSerialGraphs(); len(got) != 1 || got[0].ID != graph.ID {
		t.Fatalf("ListSerialGraphs = %#v, want generated graph", got)
	}
	if _, err := svc.StartSerialGraph(context.Background(), SerialGraphStartRequest{ID: graph.ID}); err == nil {
		t.Fatalf("StartSerialGraph must reject duplicate graph IDs")
	}
	if _, err := svc.GetSerialGraphStatus(""); err == nil {
		t.Fatalf("GetSerialGraphStatus must reject empty graph IDs")
	}
	if _, err := svc.GetSerialGraphStatus("missing"); err == nil {
		t.Fatalf("GetSerialGraphStatus must reject missing graphs")
	}
	if _, err := svc.SendSerialGraphNode(SerialGraphSendRequest{GraphID: graph.ID}); err == nil {
		t.Fatalf("SendSerialGraphNode must reject empty node IDs")
	}
	if _, err := svc.SendSerialGraphNode(SerialGraphSendRequest{GraphID: "missing", NodeID: "sender", Content: "x"}); err == nil {
		t.Fatalf("SendSerialGraphNode must reject missing graphs")
	}
	if _, err := svc.QuerySerialGraphNodeBuffer(SerialGraphBufferQuery{GraphID: "missing", NodeID: "node"}); err == nil {
		t.Fatalf("QuerySerialGraphNodeBuffer must reject missing graphs")
	}
	if err := svc.ClearSerialGraphNodeBuffer("missing", "node"); err == nil {
		t.Fatalf("ClearSerialGraphNodeBuffer must reject missing graphs")
	}
	if err := svc.ResetSerialGraphNodeCounters("missing", "node"); err == nil {
		t.Fatalf("ResetSerialGraphNodeCounters must reject missing graphs")
	}
	if _, err := svc.QuerySerialGraphNodeFrames(SerialGraphFrameQuery{GraphID: "missing", NodeID: "node"}); err == nil {
		t.Fatalf("QuerySerialGraphNodeFrames must reject missing graphs")
	}
	if err := svc.StopSerialGraph(""); err == nil {
		t.Fatalf("StopSerialGraph must reject empty graph IDs")
	}
	if err := svc.StopSerialGraph("missing"); err == nil {
		t.Fatalf("StopSerialGraph must reject missing graphs")
	}
	if err := svc.StopSerialGraph(graph.ID); err != nil {
		t.Fatalf("StopSerialGraph returned error: %v", err)
	}
	if got := svc.ListSerialGraphs(); len(got) != 0 {
		t.Fatalf("ListSerialGraphs after stop = %#v, want empty", got)
	}
}

func TestSerialGraphRuntimeDirectSendReceiveAndFrameBranches(t *testing.T) {
	svc := NewService(nil)
	runtime := newSerialGraphRuntime(svc, SerialGraphStartRequest{
		ID: "graph-direct-branches",
		Nodes: []SerialGraphNodeSpec{
			{ID: "sender", Type: "serial.sender"},
			{ID: "receiver", Type: "serial.receiver"},
			{ID: "left", Type: "serial.receiver"},
			{ID: "bridge", Type: "serial.bridge"},
			{ID: "physical", Type: "serial.physical"},
			{ID: "modbus-slave", Type: "serial.modbus.slave", Config: map[string]any{"unitIds": "3"}},
			{ID: "fecbus-mismatch", Type: "serial.fecbus.slave", Config: map[string]any{"address": 2}},
			{ID: "fecbus-off", Type: "serial.fecbus.slave", Config: map[string]any{"address": 2, "autoStatusAnswer": false}},
		},
		Edges: []SerialGraphEdgeSpec{
			{ID: "edge-bridge-left", Source: "bridge", SourceHandle: "a-out", Target: "left", TargetHandle: "in"},
		},
	})

	if _, err := runtime.sendRequest(SerialGraphSendRequest{NodeID: "missing", Content: "x"}); err == nil {
		t.Fatalf("sendRequest must reject missing nodes")
	}
	if _, err := runtime.sendRequest(SerialGraphSendRequest{NodeID: "receiver"}); err == nil {
		t.Fatalf("sendRequest must reject empty content for non-protocol nodes")
	}
	if err := runtime.send("missing", []byte("x")); err == nil {
		t.Fatalf("send must reject missing nodes")
	}
	if err := runtime.send("receiver", []byte("x")); err == nil {
		t.Fatalf("send must reject nodes that do not support direct send")
	}
	if err := runtime.send("physical", []byte("x")); err == nil {
		t.Fatalf("send must reject unbound physical nodes")
	}

	_ = runtime.receive(serialGraphPortRef{nodeID: "missing", handle: "in"}, []byte("ignored"))
	_ = runtime.receive(serialGraphPortRef{nodeID: "receiver", handle: "in"}, nil)
	_ = runtime.receive(serialGraphPortRef{nodeID: "bridge", handle: "b-in"}, []byte("left"))
	leftPage, err := runtime.queryBuffer("left", 0, 16)
	if err != nil {
		t.Fatalf("query left buffer returned error: %v", err)
	}
	if string(leftPage.Data) != "left" {
		t.Fatalf("bridge b-in routed %q, want left", string(leftPage.Data))
	}
	_ = runtime.receive(serialGraphPortRef{nodeID: "physical", handle: "tx"}, []byte("no-handle"))

	modbusFrame, err := mb.EncodeFrame(mb.FrameModeRTU, 9, mb.PDU{Function: mb.FunctionReadHoldingRegisters, Data: []byte{0, 0, 0, 1}})
	if err != nil {
		t.Fatalf("EncodeFrame(modbus): %v", err)
	}
	_ = runtime.receive(serialGraphPortRef{nodeID: "modbus-slave", handle: "rx"}, modbusFrame)
	info := runtime.info()
	if got := graphNodeStatus(&info, "modbus-slave").TxBytes; got != 0 {
		t.Fatalf("modbus slave tx for disallowed unit = %d, want 0", got)
	}

	fecMismatch, err := graphFecbusFrame(map[string]any{"targetAddress": 9})
	if err != nil {
		t.Fatalf("graphFecbusFrame mismatch: %v", err)
	}
	_ = runtime.receive(serialGraphPortRef{nodeID: "fecbus-mismatch", handle: "rx"}, fecMismatch)
	info = runtime.info()
	if got := graphNodeStatus(&info, "fecbus-mismatch").TxBytes; got != 0 {
		t.Fatalf("fecbus mismatched address tx = %d, want 0", got)
	}
	fecNoAuto, err := graphFecbusFrame(map[string]any{"targetAddress": 2})
	if err != nil {
		t.Fatalf("graphFecbusFrame no-auto: %v", err)
	}
	_ = runtime.receive(serialGraphPortRef{nodeID: "fecbus-off", handle: "rx"}, fecNoAuto)
	info = runtime.info()
	if got := graphNodeStatus(&info, "fecbus-off").TxBytes; got != 0 {
		t.Fatalf("fecbus autoStatusAnswer=false tx = %d, want 0", got)
	}

	if _, err := runtime.queryBuffer("missing", 0, 1); err == nil {
		t.Fatalf("queryBuffer must reject missing nodes")
	}
	if err := runtime.clearBuffer("missing"); err == nil {
		t.Fatalf("clearBuffer must reject missing nodes")
	}
	if err := runtime.resetCounters("missing"); err == nil {
		t.Fatalf("resetCounters must reject missing nodes")
	}
	if _, err := runtime.queryFrames(SerialGraphFrameQuery{NodeID: "missing"}); err == nil {
		t.Fatalf("queryFrames must reject missing nodes")
	}
	if runtime.nodeByHandle("missing") != nil {
		t.Fatalf("nodeByHandle missing = non-nil")
	}
	runtime.addOwnedPort("")
	runtime.addOwnedVirtualPort("")
}

func TestSerialGraphRuntimeFrameFilteringAndHelperBranches(t *testing.T) {
	node := &serialGraphRuntimeNode{spec: SerialGraphNodeSpec{ID: "frames", Type: "serial.monitor"}}
	node.appendFrame([]byte("alpha"))
	node.appendFrame([]byte("beta"))
	page := node.queryFrames(SerialGraphFrameQuery{Direction: "missing", Search: "alpha", Offset: -10, Limit: 0})
	if page.Total != 0 || len(page.Frames) != 0 || page.NextOffset != 0 {
		t.Fatalf("filtered empty page = %#v, want no frames", page)
	}
	page = node.queryFrames(SerialGraphFrameQuery{Search: "beta", Offset: 99, Limit: 5})
	if page.Total != 1 || len(page.Frames) != 0 || page.NextOffset != 1 {
		t.Fatalf("offset past filtered frames = %#v, want empty page at end", page)
	}

	if got := graphScriptFrameText([]byte{0xff}, serialScriptRunResult{}, "unsupported"); got != string([]byte{0xff}) {
		t.Fatalf("graphScriptFrameText unsupported encoding = %q, want raw bytes", got)
	}
	for _, alias := range []string{"pass", "passthrough", "pass-through", "emit-input"} {
		if !graphScriptOnErrorPassesInput(map[string]any{"onError": alias}) {
			t.Fatalf("graphScriptOnErrorPassesInput(%q) = false, want true", alias)
		}
	}
	if got := graphNormalizeModbusFrameMode(mb.FrameModeASCII); got != mb.FrameModeASCII {
		t.Fatalf("graphNormalizeModbusFrameMode ASCII = %q", got)
	}
	if got := graphFirstUnitID(map[string]any{"unitID": 8, "unitIds": "3"}); got != 8 {
		t.Fatalf("graphFirstUnitID unitID = %d, want 8", got)
	}
	if got := graphFirstUnitID(map[string]any{"unitIds": "bad 0 248"}); got != 1 {
		t.Fatalf("graphFirstUnitID invalid list = %d, want fallback 1", got)
	}
	if graphUnitIDAllowed(map[string]any{"unitIds": "bad 4"}, 3) {
		t.Fatalf("graphUnitIDAllowed must reject units not present in config")
	}
	if got := graphStringConfigRaw(nil, "missing"); got != "" {
		t.Fatalf("graphStringConfigRaw(nil) = %q, want empty", got)
	}
	if got := graphStringConfigRaw(map[string]any{}, "missing"); got != "" {
		t.Fatalf("graphStringConfigRaw missing = %q, want empty", got)
	}
}

func TestGraphModbusSlaveResponseCoversReadWriteAndExceptionBranches(t *testing.T) {
	tests := []struct {
		name         string
		pdu          mb.PDU
		wantFunction mb.FunctionCode
		wantData     []byte
	}{
		{name: "read coils rejects short request", pdu: mb.PDU{Function: mb.FunctionReadCoils, Data: []byte{0, 1}}, wantFunction: mb.FunctionReadCoils | 0x80, wantData: []byte{mb.ExceptionIllegalDataValue}},
		{name: "read coils rejects zero quantity", pdu: mb.PDU{Function: mb.FunctionReadCoils, Data: []byte{0, 0, 0, 0}}, wantFunction: mb.FunctionReadCoils | 0x80, wantData: []byte{mb.ExceptionIllegalDataValue}},
		{name: "read discrete inputs returns zero-filled byte count", pdu: mb.PDU{Function: mb.FunctionReadDiscreteInputs, Data: []byte{0, 0, 0, 9}}, wantFunction: mb.FunctionReadDiscreteInputs, wantData: []byte{2, 0, 0}},
		{name: "read registers rejects short request", pdu: mb.PDU{Function: mb.FunctionReadHoldingRegisters, Data: []byte{0, 1}}, wantFunction: mb.FunctionReadHoldingRegisters | 0x80, wantData: []byte{mb.ExceptionIllegalDataValue}},
		{name: "read registers rejects too many values", pdu: mb.PDU{Function: mb.FunctionReadInputRegisters, Data: []byte{0, 0, 0, 126}}, wantFunction: mb.FunctionReadInputRegisters | 0x80, wantData: []byte{mb.ExceptionIllegalDataValue}},
		{name: "read registers returns zero-filled register bytes", pdu: mb.PDU{Function: mb.FunctionReadHoldingRegisters, Data: []byte{0, 0, 0, 2}}, wantFunction: mb.FunctionReadHoldingRegisters, wantData: []byte{4, 0, 0, 0, 0}},
		{name: "write single rejects short request", pdu: mb.PDU{Function: mb.FunctionWriteSingleCoil, Data: []byte{0, 1}}, wantFunction: mb.FunctionWriteSingleCoil | 0x80, wantData: []byte{mb.ExceptionIllegalDataValue}},
		{name: "write single echoes address and value", pdu: mb.PDU{Function: mb.FunctionWriteSingleRegister, Data: []byte{0, 3, 0, 4, 0xff}}, wantFunction: mb.FunctionWriteSingleRegister, wantData: []byte{0, 3, 0, 4}},
		{name: "write multiple rejects short request", pdu: mb.PDU{Function: mb.FunctionWriteMultipleCoils, Data: []byte{0, 1}}, wantFunction: mb.FunctionWriteMultipleCoils | 0x80, wantData: []byte{mb.ExceptionIllegalDataValue}},
		{name: "write multiple echoes address and quantity", pdu: mb.PDU{Function: mb.FunctionWriteMultipleRegisters, Data: []byte{0, 5, 0, 6, 2, 0, 1}}, wantFunction: mb.FunctionWriteMultipleRegisters, wantData: []byte{0, 5, 0, 6}},
		{name: "unsupported function returns illegal function exception", pdu: mb.PDU{Function: mb.FunctionCode(0x7f)}, wantFunction: mb.FunctionCode(0xff), wantData: []byte{mb.ExceptionIllegalFunction}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := graphModbusSlaveResponse(mb.DecodedFrame{PDU: tt.pdu})
			if err != nil {
				t.Fatalf("graphModbusSlaveResponse returned error: %v", err)
			}
			if got.Function != tt.wantFunction || !reflect.DeepEqual(got.Data, tt.wantData) {
				t.Fatalf("graphModbusSlaveResponse = function %02x data % x, want function %02x data % x", byte(got.Function), got.Data, byte(tt.wantFunction), tt.wantData)
			}
		})
	}
}

func TestGraphConfigPayloadAndValidationHelpersCoverBranches(t *testing.T) {
	intConfig := map[string]any{"int": 1, "int64": int64(2), "float64": float64(3.8), "float32": float32(4.2), "string": "5", "bad": "five"}
	for _, tt := range []struct {
		key  string
		want int
	}{
		{key: "int", want: 1},
		{key: "int64", want: 2},
		{key: "float64", want: 3},
		{key: "float32", want: 4},
		{key: "string", want: 5},
		{key: "bad", want: 99},
		{key: "missing", want: 99},
	} {
		if got := graphIntConfig(intConfig, tt.key, 99); got != tt.want {
			t.Fatalf("graphIntConfig(%q) = %d, want %d", tt.key, got, tt.want)
		}
	}
	if got := graphIntConfig(nil, "missing", 42); got != 42 {
		t.Fatalf("nil graphIntConfig = %d, want fallback", got)
	}

	boolConfig := map[string]any{"bool": true, "true": "true", "false": "false", "bad": "maybe"}
	for _, tt := range []struct {
		key      string
		fallback bool
		want     bool
	}{
		{key: "bool", want: true},
		{key: "true", want: true},
		{key: "false", fallback: true, want: false},
		{key: "bad", fallback: true, want: true},
		{key: "missing", fallback: true, want: true},
	} {
		if got := graphBoolConfig(boolConfig, tt.key, tt.fallback); got != tt.want {
			t.Fatalf("graphBoolConfig(%q) = %v, want %v", tt.key, got, tt.want)
		}
	}
	if got := graphBoolConfig(nil, "missing", true); !got {
		t.Fatalf("nil graphBoolConfig = false, want fallback true")
	}

	stringConfig := map[string]any{"trim": "  value  ", "raw": "  raw  ", "nil": nil, "nilString": "<nil>"}
	if got := graphStringConfig(stringConfig, "trim"); got != "value" {
		t.Fatalf("graphStringConfig trim = %q", got)
	}
	if got := graphStringConfigRaw(stringConfig, "raw"); got != "  raw  " {
		t.Fatalf("graphStringConfigRaw = %q", got)
	}
	if got := graphStringConfigWithDefault(stringConfig, "nilString", "fallback"); got != "fallback" {
		t.Fatalf("graphStringConfigWithDefault = %q, want fallback", got)
	}
	serialConfig := graphSerialConfig(map[string]any{"portName": " p ", "baudRate": "9600", "dataBits": float64(7), "stopBits": "2", "parity": "even", "flowMode": "xon", "readBufKB": int64(64)})
	if serialConfig.PortName != "p" || serialConfig.BaudRate != 9600 || serialConfig.DataBits != 7 || serialConfig.StopBits != "2" || serialConfig.Parity != "even" || serialConfig.FlowMode != "xon" || serialConfig.ReadBufKB != 64 {
		t.Fatalf("graphSerialConfig = %#v", serialConfig)
	}

	if data, err := encodeGraphPayload("41 42", "hex", "utf-8"); err != nil || string(data) != "AB" {
		t.Fatalf("encodeGraphPayload hex = %q, %v; want AB", data, err)
	}
	if _, err := encodeGraphPayload("zz", "hex", "utf-8"); err == nil {
		t.Fatalf("encodeGraphPayload must reject invalid hex")
	}
	if _, err := encodeGraphPayload("é", "ascii", "ascii"); err == nil {
		t.Fatalf("encodeGraphPayload must reject invalid ascii content")
	}
	if _, err := graphAutoSenderData(map[string]any{"payload": ""}); err == nil {
		t.Fatalf("graphAutoSenderData must reject empty payload")
	}
	if data, err := graphAutoSenderData(map[string]any{"payload": "41", "mode": "hex"}); err != nil || string(data) != "A" {
		t.Fatalf("graphAutoSenderData hex = %q, %v; want A", data, err)
	}
	if data, err := graphAutoNodeData(&serialGraphRuntimeNode{spec: SerialGraphNodeSpec{Type: "serial.sender", Config: map[string]any{"payload": "ok"}}}); err != nil || string(data) != "ok" {
		t.Fatalf("graphAutoNodeData sender = %q, %v; want ok", data, err)
	}
	if _, err := graphAutoNodeData(&serialGraphRuntimeNode{spec: SerialGraphNodeSpec{Type: "serial.receiver"}}); err == nil {
		t.Fatalf("graphAutoNodeData must reject unsupported node type")
	}
	if _, err := graphNodeSendData(&serialGraphRuntimeNode{spec: SerialGraphNodeSpec{Type: "serial.receiver"}}, SerialGraphSendRequest{}); err == nil {
		t.Fatalf("graphNodeSendData must reject empty content for receiver")
	}
	if _, err := graphNodeSendData(&serialGraphRuntimeNode{spec: SerialGraphNodeSpec{Type: "serial.fecbus.master", Config: map[string]any{"dataHex": "zz"}}}, SerialGraphSendRequest{}); err == nil {
		t.Fatalf("graphNodeSendData must surface fecbus payload errors")
	}

	errs := validateSerialGraphRuntimeRequest(SerialGraphStartRequest{
		Nodes: []SerialGraphNodeSpec{
			{ID: "", Type: "serial.sender"},
			{ID: "dup", Type: "serial.sender"},
			{ID: "dup", Type: "serial.receiver"},
			{ID: "unknown", Type: "serial.unknown"},
			{ID: "phys-a", Type: "serial.physical", Config: map[string]any{"portName": "COM1"}},
			{ID: "phys-b", Type: "serial.physical", Config: map[string]any{"portName": "COM1"}},
			{ID: "sender", Type: "serial.sender"},
			{ID: "sender2", Type: "serial.sender"},
			{ID: "receiver", Type: "serial.receiver"},
			{ID: "master", Type: "serial.modbus.master"},
		},
		Edges: []SerialGraphEdgeSpec{
			{ID: "", Source: "sender", SourceHandle: "out", Target: "receiver", TargetHandle: "in"},
			{ID: "edge", Source: "sender", SourceHandle: "out", Target: "receiver", TargetHandle: "in"},
			{ID: "edge", Source: "sender", SourceHandle: "out", Target: "master", TargetHandle: "rx"},
			{ID: "missing-source", Source: "ghost", SourceHandle: "out", Target: "receiver", TargetHandle: "in"},
			{ID: "missing-target", Source: "sender", SourceHandle: "out", Target: "ghost", TargetHandle: "in"},
			{ID: "self", Source: "sender", SourceHandle: "out", Target: "sender", TargetHandle: "in"},
			{ID: "bad-out", Source: "sender", SourceHandle: "missing", Target: "receiver", TargetHandle: "in"},
			{ID: "bad-in", Source: "sender", SourceHandle: "out", Target: "receiver", TargetHandle: "missing"},
			{ID: "input-dup", Source: "sender2", SourceHandle: "out", Target: "receiver", TargetHandle: "in"},
			{ID: "fanout", Source: "sender", SourceHandle: "out", Target: "master", TargetHandle: "rx"},
		},
	})
	joined := strings.Join(errs, "\n")
	for _, want := range []string{
		"node id must not be empty",
		"duplicate node id: dup",
		"provider not found: serial.unknown",
		"resource port duplicated: COM1",
		"edge id must not be empty",
		"duplicate edge id: edge",
		"source node not found: ghost",
		"target node not found: ghost",
		"node cannot connect to itself",
		"output port not found: serial.sender.missing",
		"input port not found: serial.receiver.missing",
		"input already connected: receiver.in",
		"fan-out requires a tap node: sender.out",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("validateSerialGraphRuntimeRequest errors %q missing %q", joined, want)
		}
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

func TestSerialGraphRuntimeModbusNodesCaptureProtocolFrames(t *testing.T) {
	svc := NewService(nil)
	defer svc.cleanup()

	graph := startTestGraph(t, svc, SerialGraphStartRequest{
		ID: "graph-modbus-protocol-frames",
		Nodes: []SerialGraphNodeSpec{
			{ID: "master", Type: "serial.modbus.master", Config: map[string]any{"mode": "rtu", "unitIds": "3", "functionCode": 3, "address": 0, "quantity": 2}},
			{ID: "slave", Type: "serial.modbus.slave", Config: map[string]any{"mode": "rtu", "unitIds": "3"}},
			{ID: "response-source", Type: "serial.sender"},
			{ID: "receiver", Type: "serial.receiver"},
		},
		Edges: []SerialGraphEdgeSpec{
			{ID: "edge-master-slave", Source: "master", SourceHandle: "tx", Target: "slave", TargetHandle: "rx"},
			{ID: "edge-slave-receiver", Source: "slave", SourceHandle: "tx", Target: "receiver", TargetHandle: "in"},
			{ID: "edge-response-master", Source: "response-source", SourceHandle: "out", Target: "master", TargetHandle: "rx"},
		},
	})

	if _, err := svc.SendSerialGraphNode(SerialGraphSendRequest{GraphID: graph.ID, NodeID: "master"}); err != nil {
		t.Fatalf("SendSerialGraphNode returned error: %v", err)
	}
	response, err := mb.EncodeFrame(mb.FrameModeRTU, 3, mb.PDU{Function: mb.FunctionReadHoldingRegisters, Data: []byte{4, 0, 0, 0, 0}})
	if err != nil {
		t.Fatalf("EncodeFrame(response): %v", err)
	}
	if _, err := svc.SendSerialGraphNode(SerialGraphSendRequest{
		GraphID: graph.ID,
		NodeID:  "response-source",
		Content: formatGraphFrameBytes(response, "%02x"),
		Mode:    "hex",
	}); err != nil {
		t.Fatalf("SendSerialGraphNode(response-source) returned error: %v", err)
	}

	masterFrames, err := svc.QuerySerialGraphNodeFrames(SerialGraphFrameQuery{GraphID: graph.ID, NodeID: "master", Offset: 0, Limit: 10})
	if err != nil {
		t.Fatalf("QuerySerialGraphNodeFrames(master) returned error: %v", err)
	}
	if masterFrames.Total != 2 || len(masterFrames.Frames) != 2 {
		t.Fatalf("master frame page total=%d len=%d, want request and response frames", masterFrames.Total, len(masterFrames.Frames))
	}
	if masterFrames.Frames[0].Direction != "发送" || masterFrames.Frames[1].Direction != "接收" {
		t.Fatalf("master directions = %q/%q, want send/receive", masterFrames.Frames[0].Direction, masterFrames.Frames[1].Direction)
	}
	for _, want := range []string{"Unit 3", "FC 03", "Read Holding Registers", "Address 0", "Quantity 2", "CRC"} {
		if !strings.Contains(masterFrames.Frames[0].DisplayText, want) {
			t.Fatalf("master request display = %q, want %q", masterFrames.Frames[0].DisplayText, want)
		}
	}
	if strings.Contains(masterFrames.Frames[0].DisplayText, "Offset") {
		t.Fatalf("master request display = %q, want protocol frame text without offset table", masterFrames.Frames[0].DisplayText)
	}
	if !strings.Contains(masterFrames.Frames[1].DisplayText, "Byte Count 4") || !strings.Contains(masterFrames.Frames[1].DisplayText, "Values 0, 0") {
		t.Fatalf("master response display = %q, want decoded response fields", masterFrames.Frames[1].DisplayText)
	}

	slaveFrames, err := svc.QuerySerialGraphNodeFrames(SerialGraphFrameQuery{GraphID: graph.ID, NodeID: "slave", Offset: 0, Limit: 10})
	if err != nil {
		t.Fatalf("QuerySerialGraphNodeFrames(slave) returned error: %v", err)
	}
	if slaveFrames.Total != 2 || len(slaveFrames.Frames) != 2 {
		t.Fatalf("slave frame page total=%d len=%d, want request and response frames", slaveFrames.Total, len(slaveFrames.Frames))
	}
	if slaveFrames.Frames[0].Direction != "接收" || slaveFrames.Frames[1].Direction != "发送" {
		t.Fatalf("slave directions = %q/%q, want receive/send", slaveFrames.Frames[0].Direction, slaveFrames.Frames[1].Direction)
	}
}

func TestSerialGraphRuntimeModbusResponseCanReturnToMaster(t *testing.T) {
	svc := NewService(nil)
	defer svc.cleanup()

	graph := startTestGraph(t, svc, SerialGraphStartRequest{
		ID: "graph-modbus-response-master",
		Nodes: []SerialGraphNodeSpec{
			{ID: "master", Type: "serial.modbus.master", Config: map[string]any{"mode": "rtu", "unitIds": "3", "functionCode": 3, "address": 0, "quantity": 2}},
			{ID: "slave", Type: "serial.modbus.slave", Config: map[string]any{"mode": "rtu", "unitIds": "3"}},
			{ID: "tap", Type: "serial.tap"},
			{ID: "receiver", Type: "serial.receiver"},
		},
		Edges: []SerialGraphEdgeSpec{
			{ID: "edge-master-slave", Source: "master", SourceHandle: "tx", Target: "slave", TargetHandle: "rx"},
			{ID: "edge-slave-tap", Source: "slave", SourceHandle: "tx", Target: "tap", TargetHandle: "in"},
			{ID: "edge-tap-master", Source: "tap", SourceHandle: "out", Target: "master", TargetHandle: "rx"},
			{ID: "edge-tap-receiver", Source: "tap", SourceHandle: "out", Target: "receiver", TargetHandle: "in"},
		},
	})

	if _, err := svc.SendSerialGraphNode(SerialGraphSendRequest{GraphID: graph.ID, NodeID: "master"}); err != nil {
		t.Fatalf("SendSerialGraphNode returned error: %v", err)
	}
	masterFrames, err := svc.QuerySerialGraphNodeFrames(SerialGraphFrameQuery{GraphID: graph.ID, NodeID: "master", Offset: 0, Limit: 10})
	if err != nil {
		t.Fatalf("QuerySerialGraphNodeFrames(master) returned error: %v", err)
	}
	if masterFrames.Total != 2 || len(masterFrames.Frames) != 2 {
		t.Fatalf("master frame page total=%d len=%d, want request and response frames", masterFrames.Total, len(masterFrames.Frames))
	}
	if masterFrames.Frames[0].Direction != "发送" || masterFrames.Frames[1].Direction != "接收" {
		t.Fatalf("master directions = %q/%q, want send/receive", masterFrames.Frames[0].Direction, masterFrames.Frames[1].Direction)
	}
	if !strings.Contains(masterFrames.Frames[1].DisplayText, "Raw ") {
		t.Fatalf("master response display = %q, want raw data", masterFrames.Frames[1].DisplayText)
	}
	assertGraphBufferHasTraffic(t, svc, graph.ID, "receiver")
}

func TestSerialGraphRuntimeModbusInvalidFrameIsMarked(t *testing.T) {
	svc := NewService(nil)
	defer svc.cleanup()

	graph := startTestGraph(t, svc, SerialGraphStartRequest{
		ID: "graph-modbus-invalid-frame",
		Nodes: []SerialGraphNodeSpec{
			{ID: "sender", Type: "serial.sender"},
			{ID: "master", Type: "serial.modbus.master", Config: map[string]any{"mode": "rtu", "unitIds": "3"}},
		},
		Edges: []SerialGraphEdgeSpec{
			{ID: "edge-invalid-master", Source: "sender", SourceHandle: "out", Target: "master", TargetHandle: "rx"},
		},
	})

	if _, err := svc.SendSerialGraphNode(SerialGraphSendRequest{
		GraphID: graph.ID,
		NodeID:  "sender",
		Content: "01 03 00",
		Mode:    "hex",
	}); err != nil {
		t.Fatalf("SendSerialGraphNode returned error: %v", err)
	}
	page, err := svc.QuerySerialGraphNodeFrames(SerialGraphFrameQuery{GraphID: graph.ID, NodeID: "master", Offset: 0, Limit: 10})
	if err != nil {
		t.Fatalf("QuerySerialGraphNodeFrames(master) returned error: %v", err)
	}
	if page.Total != 1 || len(page.Frames) != 1 {
		t.Fatalf("invalid frame page total=%d len=%d, want one frame", page.Total, len(page.Frames))
	}
	frame := page.Frames[0]
	if frame.Error == "" || !strings.Contains(frame.DisplayText, "Invalid frame") || !strings.Contains(frame.DisplayText, "Raw 01 03 00") {
		t.Fatalf("invalid frame = %#v, want error marker and raw data", frame)
	}
}

func TestSerialGraphRuntimeQueryFramesCanReadRecentPage(t *testing.T) {
	svc := NewService(nil)
	defer svc.cleanup()

	graph := startTestGraph(t, svc, SerialGraphStartRequest{
		ID: "graph-recent-frames",
		Nodes: []SerialGraphNodeSpec{
			{ID: "sender", Type: "serial.sender", Config: map[string]any{"mode": "ascii"}},
			{ID: "monitor", Type: "serial.monitor"},
		},
		Edges: []SerialGraphEdgeSpec{
			{ID: "edge-monitor", Source: "sender", SourceHandle: "out", Target: "monitor", TargetHandle: "in"},
		},
	})

	for i := 0; i < 150; i += 1 {
		if _, err := svc.SendSerialGraphNode(SerialGraphSendRequest{
			GraphID: graph.ID,
			NodeID:  "sender",
			Content: "x",
			Mode:    "ascii",
		}); err != nil {
			t.Fatalf("SendSerialGraphNode(%d) returned error: %v", i, err)
		}
	}
	page, err := svc.QuerySerialGraphNodeFrames(SerialGraphFrameQuery{GraphID: graph.ID, NodeID: "monitor", Offset: -120, Limit: 120})
	if err != nil {
		t.Fatalf("QuerySerialGraphNodeFrames(monitor) returned error: %v", err)
	}
	if page.Total != 150 || len(page.Frames) != 120 {
		t.Fatalf("frame page total=%d len=%d, want 150 total and 120 recent frames", page.Total, len(page.Frames))
	}
	if page.Frames[0].Seq != 31 || page.Frames[len(page.Frames)-1].Seq != 150 {
		t.Fatalf("recent frame seq range = %d..%d, want 31..150", page.Frames[0].Seq, page.Frames[len(page.Frames)-1].Seq)
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

func TestSerialGraphRuntimeModbusMasterAutoSendsThroughVirtualPort(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping socat integration test in short mode")
	}
	svc := NewService(nil)
	defer svc.cleanup()
	graph, err := svc.StartSerialGraph(context.Background(), SerialGraphStartRequest{
		ID: "graph-modbus-auto-virtual",
		Nodes: []SerialGraphNodeSpec{
			{ID: "master", Type: "serial.modbus.master", Config: graphModbusMasterConfig(map[string]any{"autoSend": true, "intervalMs": 10, "quantity": 2})},
			{ID: "vport", Type: "serial.virtual", Config: graphVirtualConfig("portweave-graph-modbus-auto-vp")},
			{ID: "slave", Type: "serial.modbus.slave", Config: graphModbusSlaveConfig()},
			{ID: "receiver", Type: "serial.receiver", Config: graphReceiverConfig()},
		},
		Edges: []SerialGraphEdgeSpec{
			{ID: "edge-master-vport", Source: "master", SourceHandle: "tx", Target: "vport", TargetHandle: "tx"},
			{ID: "edge-vport-slave", Source: "vport", SourceHandle: "rx", Target: "slave", TargetHandle: "rx"},
			{ID: "edge-slave-receiver", Source: "slave", SourceHandle: "tx", Target: "receiver", TargetHandle: "in"},
		},
	})
	if err != nil {
		t.Skipf("virtual serial unavailable: %v", err)
	}

	page := waitGraphBuffer(t, svc, graph.ID, "receiver", 9)
	frame, err := mb.DecodeFrame(mb.FrameModeRTU, page.Data[:9])
	if err != nil {
		t.Fatalf("DecodeFrame: %v", err)
	}
	parsed, err := mb.ParseResponse(frame.PDU)
	if err != nil {
		t.Fatalf("ParseResponse: %v", err)
	}
	if frame.UnitID != 1 || parsed.Function != mb.FunctionReadHoldingRegisters || len(parsed.Values) != 2 {
		t.Fatalf("parsed response = %#v unit=%d, want 2 holding registers for unit 1", parsed, frame.UnitID)
	}
	if err := svc.StopSerialGraph(graph.ID); err != nil {
		t.Fatalf("StopSerialGraph returned error: %v", err)
	}
}

func TestSerialGraphRuntimeFecbusMasterAutoSendsThroughVirtualPort(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping socat integration test in short mode")
	}
	svc := NewService(nil)
	defer svc.cleanup()
	graph, err := svc.StartSerialGraph(context.Background(), SerialGraphStartRequest{
		ID: "graph-fecbus-auto-virtual",
		Nodes: []SerialGraphNodeSpec{
			{ID: "master", Type: "serial.fecbus.master", Config: graphFecbusMasterConfig(map[string]any{"autoSend": true, "intervalMs": 10})},
			{ID: "vport", Type: "serial.virtual", Config: graphVirtualConfig("portweave-graph-fecbus-auto-vp")},
			{ID: "slave", Type: "serial.fecbus.slave", Config: graphFecbusSlaveConfig()},
			{ID: "receiver", Type: "serial.receiver", Config: graphReceiverConfig()},
		},
		Edges: []SerialGraphEdgeSpec{
			{ID: "edge-master-vport", Source: "master", SourceHandle: "tx", Target: "vport", TargetHandle: "tx"},
			{ID: "edge-vport-slave", Source: "vport", SourceHandle: "rx", Target: "slave", TargetHandle: "rx"},
			{ID: "edge-slave-receiver", Source: "slave", SourceHandle: "tx", Target: "receiver", TargetHandle: "in"},
		},
	})
	if err != nil {
		t.Skipf("virtual serial unavailable: %v", err)
	}

	page := waitGraphBuffer(t, svc, graph.ID, "receiver", 13)
	frame, err := fb.DecodeFrame(page.Data[:13])
	if err != nil {
		t.Fatalf("DecodeFrame: %v", err)
	}
	if frame.Type != fb.FrameTypeAnswer || frame.TargetAddress != 1 || frame.SourceAddress != 2 || frame.Function() != fb.FunctionStatusAnswer {
		t.Fatalf("fecbus answer = %#v, want status answer from 2 to 1", frame)
	}
	if err := svc.StopSerialGraph(graph.ID); err != nil {
		t.Fatalf("StopSerialGraph returned error: %v", err)
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
	defer func() { _ = conn.Close() }()
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
	portName := "portweave-graph-test-vp"
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
	defer func() { _ = external.Close() }()
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
			{ID: "vport", Type: "serial.virtual", Config: map[string]any{"portName": "portweave-graph-auto-vp", "baudRate": 115200}},
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
			{ID: "vport", Type: "serial.virtual", Config: map[string]any{"portName": "portweave-graph-auto-backpressure-vp", "baudRate": 115200}},
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
				{ID: "vport", Type: "serial.virtual", Config: map[string]any{"portName": "portweave-graph-dup-vp", "baudRate": 115200}},
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
		return fmt.Sprintf("portweave-load-%s-%s-%d", portToken, name, time.Now().UnixNano())
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
			{ID: "bridge", Type: "serial.bridge"},
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
			{ID: "vport", Type: "serial.virtual", Config: graphVirtualConfig(portName("modbus"))},
			{ID: "slave", Type: "serial.modbus.slave", Config: graphModbusSlaveConfig()},
			{ID: "tap", Type: "serial.tap"},
			{ID: "receiver", Type: "serial.receiver", Config: graphReceiverConfig()},
			{ID: "monitor", Type: "serial.monitor", Config: graphMonitorConfig()},
		}
		edges := []SerialGraphEdgeSpec{
			{ID: "edge-master-vport", Source: "master", SourceHandle: "tx", Target: "vport", TargetHandle: "tx"},
			{ID: "edge-vport-slave", Source: "vport", SourceHandle: "rx", Target: "slave", TargetHandle: "rx"},
			{ID: "edge-slave-tap", Source: "slave", SourceHandle: "tx", Target: "tap", TargetHandle: "in"},
			{ID: "edge-tap-master", Source: "tap", SourceHandle: "out", Target: "master", TargetHandle: "rx"},
			{ID: "edge-tap-receiver", Source: "tap", SourceHandle: "out", Target: "receiver", TargetHandle: "in"},
			{ID: "edge-tap-monitor", Source: "tap", SourceHandle: "out", Target: "monitor", TargetHandle: "in"},
		}
		return demoGraphStressCase{
			Request:           request(nodes, edges),
			StatusNodes:       []string{"master", "vport", "slave", "tap", "receiver", "monitor"},
			ProtocolWorkloads: protocolWorkload("master", "receiver"),
			MonitorNodes:      []string{"monitor"},
		}
	case "fecbus-demo":
		nodes := []SerialGraphNodeSpec{
			{ID: "master", Type: "serial.fecbus.master", Config: graphFecbusMasterConfig()},
			{ID: "vport", Type: "serial.virtual", Config: graphVirtualConfig(portName("fecbus"))},
			{ID: "slave", Type: "serial.fecbus.slave", Config: graphFecbusSlaveConfig()},
			{ID: "tap", Type: "serial.tap"},
			{ID: "receiver", Type: "serial.receiver", Config: graphReceiverConfig()},
			{ID: "monitor", Type: "serial.monitor", Config: graphMonitorConfig()},
		}
		edges := []SerialGraphEdgeSpec{
			{ID: "edge-master-vport", Source: "master", SourceHandle: "tx", Target: "vport", TargetHandle: "tx"},
			{ID: "edge-vport-slave", Source: "vport", SourceHandle: "rx", Target: "slave", TargetHandle: "rx"},
			{ID: "edge-slave-tap", Source: "slave", SourceHandle: "tx", Target: "tap", TargetHandle: "in"},
			{ID: "edge-tap-receiver", Source: "tap", SourceHandle: "out", Target: "receiver", TargetHandle: "in"},
			{ID: "edge-tap-monitor", Source: "tap", SourceHandle: "out", Target: "monitor", TargetHandle: "in"},
		}
		return demoGraphStressCase{
			Request:           request(nodes, edges),
			StatusNodes:       []string{"master", "vport", "slave", "tap", "receiver", "monitor"},
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
			{ID: "bridge", Type: "serial.bridge"},
			{ID: "bridge-receiver-a", Type: "serial.receiver", Config: graphReceiverConfig()},
			{ID: "bridge-receiver-b", Type: "serial.receiver", Config: graphReceiverConfig()},
			{ID: "modbus-master", Type: "serial.modbus.master", Config: graphModbusMasterConfig()},
			{ID: "modbus-vport", Type: "serial.virtual", Config: graphVirtualConfig(portName("full-modbus"))},
			{ID: "modbus-slave", Type: "serial.modbus.slave", Config: graphModbusSlaveConfig()},
			{ID: "modbus-tap", Type: "serial.tap"},
			{ID: "modbus-receiver", Type: "serial.receiver", Config: graphReceiverConfig()},
			{ID: "modbus-monitor", Type: "serial.monitor", Config: graphMonitorConfig()},
			{ID: "fecbus-master", Type: "serial.fecbus.master", Config: graphFecbusMasterConfig()},
			{ID: "fecbus-vport", Type: "serial.virtual", Config: graphVirtualConfig(portName("full-fecbus"))},
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
			{ID: "edge-modbus-master-vport", Source: "modbus-master", SourceHandle: "tx", Target: "modbus-vport", TargetHandle: "tx"},
			{ID: "edge-modbus-vport-slave", Source: "modbus-vport", SourceHandle: "rx", Target: "modbus-slave", TargetHandle: "rx"},
			{ID: "edge-modbus-slave-tap", Source: "modbus-slave", SourceHandle: "tx", Target: "modbus-tap", TargetHandle: "in"},
			{ID: "edge-modbus-tap-master", Source: "modbus-tap", SourceHandle: "out", Target: "modbus-master", TargetHandle: "rx"},
			{ID: "edge-modbus-tap-receiver", Source: "modbus-tap", SourceHandle: "out", Target: "modbus-receiver", TargetHandle: "in"},
			{ID: "edge-modbus-tap-monitor", Source: "modbus-tap", SourceHandle: "out", Target: "modbus-monitor", TargetHandle: "in"},
			{ID: "edge-fecbus-master-vport", Source: "fecbus-master", SourceHandle: "tx", Target: "fecbus-vport", TargetHandle: "tx"},
			{ID: "edge-fecbus-vport-slave", Source: "fecbus-vport", SourceHandle: "rx", Target: "fecbus-slave", TargetHandle: "rx"},
			{ID: "edge-fecbus-slave-tap", Source: "fecbus-slave", SourceHandle: "tx", Target: "fecbus-tap", TargetHandle: "in"},
			{ID: "edge-fecbus-tap-receiver", Source: "fecbus-tap", SourceHandle: "out", Target: "fecbus-receiver", TargetHandle: "in"},
		}
		return demoGraphStressCase{
			Request: request(nodes, edges),
			StatusNodes: []string{
				"main-sender", "main-vport", "main-tap", "main-receiver", "main-monitor",
				"bridge-sender-a", "bridge-vport-a", "bridge-sender-b", "bridge-vport-b", "bridge", "bridge-receiver-a", "bridge-receiver-b",
				"modbus-master", "modbus-vport", "modbus-slave", "modbus-tap", "modbus-receiver", "modbus-monitor",
				"fecbus-master", "fecbus-vport", "fecbus-slave", "fecbus-tap", "fecbus-receiver",
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

type graphTCPTestServer struct {
	listener net.Listener
	accepted chan net.Conn
	received chan []byte
	mu       sync.Mutex
	conns    []net.Conn
}

func newGraphTCPTestServer(t *testing.T) *graphTCPTestServer {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen returned error: %v", err)
	}
	return newGraphTCPTestServerOnListener(t, listener)
}

func newGraphTCPTestServerOnListener(t *testing.T, listener net.Listener) *graphTCPTestServer {
	t.Helper()
	server := &graphTCPTestServer{
		listener: listener,
		accepted: make(chan net.Conn, 4),
		received: make(chan []byte, 16),
	}
	t.Cleanup(func() {
		_ = listener.Close()
		server.mu.Lock()
		defer server.mu.Unlock()
		for _, conn := range server.conns {
			_ = conn.Close()
		}
	})
	go server.acceptLoop()
	return server
}

func reserveGraphTCPPort(t *testing.T) (string, int) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen reserve port returned error: %v", err)
	}
	host, portString, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		_ = listener.Close()
		t.Fatalf("SplitHostPort returned error: %v", err)
	}
	port, err := strconv.Atoi(portString)
	if err != nil {
		_ = listener.Close()
		t.Fatalf("Atoi(%q) returned error: %v", portString, err)
	}
	if err := listener.Close(); err != nil {
		t.Fatalf("close reserved listener returned error: %v", err)
	}
	return host, port
}

func (s *graphTCPTestServer) acceptLoop() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return
		}
		s.mu.Lock()
		s.conns = append(s.conns, conn)
		s.mu.Unlock()
		s.accepted <- conn
		go s.readLoop(conn)
	}
}

func (s *graphTCPTestServer) readLoop(conn net.Conn) {
	buf := make([]byte, 1024)
	for {
		n, err := conn.Read(buf)
		if n > 0 {
			data := append([]byte(nil), buf[:n]...)
			s.received <- data
		}
		if err != nil {
			return
		}
	}
}

type graphFailingConn struct {
	mu      sync.Mutex
	closed  bool
	onWrite func()
}

func (c *graphFailingConn) Read([]byte) (int, error) {
	return 0, fmt.Errorf("graph failing conn read")
}

func (c *graphFailingConn) Write([]byte) (int, error) {
	if c.onWrite != nil {
		c.onWrite()
	}
	return 0, fmt.Errorf("graph failing conn write")
}

func (c *graphFailingConn) Close() error {
	c.mu.Lock()
	c.closed = true
	c.mu.Unlock()
	return nil
}

func (c *graphFailingConn) LocalAddr() net.Addr {
	return graphTestAddr("local")
}

func (c *graphFailingConn) RemoteAddr() net.Addr {
	return graphTestAddr("remote")
}

func (c *graphFailingConn) SetDeadline(time.Time) error {
	return nil
}

func (c *graphFailingConn) SetReadDeadline(time.Time) error {
	return nil
}

func (c *graphFailingConn) SetWriteDeadline(time.Time) error {
	return nil
}

func (c *graphFailingConn) isClosed() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.closed
}

type graphBlockingFailingConn struct {
	closed chan struct{}
	once   sync.Once
}

func newGraphBlockingFailingConn() *graphBlockingFailingConn {
	return &graphBlockingFailingConn{closed: make(chan struct{})}
}

func (c *graphBlockingFailingConn) Read([]byte) (int, error) {
	<-c.closed
	return 0, fmt.Errorf("graph blocking failing conn read")
}

func (c *graphBlockingFailingConn) Write([]byte) (int, error) {
	return 0, fmt.Errorf("graph blocking failing conn write")
}

func (c *graphBlockingFailingConn) Close() error {
	c.once.Do(func() { close(c.closed) })
	return nil
}

func (c *graphBlockingFailingConn) LocalAddr() net.Addr {
	return graphTestAddr("local")
}

func (c *graphBlockingFailingConn) RemoteAddr() net.Addr {
	return graphTestAddr("remote")
}

func (c *graphBlockingFailingConn) SetDeadline(time.Time) error {
	return nil
}

func (c *graphBlockingFailingConn) SetReadDeadline(time.Time) error {
	return nil
}

func (c *graphBlockingFailingConn) SetWriteDeadline(time.Time) error {
	return nil
}

type graphTestAddr string

func (a graphTestAddr) Network() string {
	return "tcp"
}

func (a graphTestAddr) String() string {
	return string(a)
}

func (s *graphTCPTestServer) hostPort(t *testing.T) (string, int) {
	t.Helper()
	host, portString, err := net.SplitHostPort(s.listener.Addr().String())
	if err != nil {
		t.Fatalf("SplitHostPort returned error: %v", err)
	}
	port, err := strconv.Atoi(portString)
	if err != nil {
		t.Fatalf("Atoi(%q) returned error: %v", portString, err)
	}
	return host, port
}

func (s *graphTCPTestServer) waitConn(t *testing.T) net.Conn {
	t.Helper()
	select {
	case conn := <-s.accepted:
		return conn
	case <-time.After(2 * time.Second):
		t.Fatalf("remote TCP client did not connect")
		return nil
	}
}

func (s *graphTCPTestServer) waitReceived(t *testing.T) []byte {
	t.Helper()
	select {
	case data := <-s.received:
		return data
	case <-time.After(2 * time.Second):
		t.Fatalf("server did not receive remote TCP write")
		return nil
	}
}

func graphRemoteConfig(host string, port int, overrides ...map[string]any) map[string]any {
	config := map[string]any{
		"protocol":               "raw-tcp",
		"role":                   "client",
		"host":                   host,
		"port":                   port,
		"connectTimeoutMs":       1000,
		"writeTimeoutMs":         1000,
		"reconnect":              true,
		"reconnectIntervalMs":    100,
		"allowStartDisconnected": false,
		"readBufKB":              32,
		"baudRate":               115200,
		"dataBits":               8,
		"stopBits":               "1",
		"parity":                 "none",
		"flowMode":               "none",
	}
	for _, override := range overrides {
		for key, value := range override {
			config[key] = value
		}
	}
	return config
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
	return map[string]any{"displayMode": "hex"}
}

func graphModbusMasterConfig(overrides ...map[string]any) map[string]any {
	config := map[string]any{
		"mode":         "rtu",
		"unitIds":      "1,2",
		"functionCode": int(mb.FunctionReadHoldingRegisters),
		"addressMode":  "zero-based",
		"address":      0,
		"quantity":     125,
	}
	for _, override := range overrides {
		for key, value := range override {
			config[key] = value
		}
	}
	return config
}

func graphModbusSlaveConfig() map[string]any {
	return map[string]any{"mode": "rtu", "unitIds": "1,2"}
}

func graphFecbusMasterConfig(overrides ...map[string]any) map[string]any {
	config := map[string]any{
		"frameType":     int(fb.FrameTypeRequest),
		"sourceAddress": 1,
		"targetAddress": 2,
		"priority":      2,
		"messageNumber": 7,
		"groupNumber":   0,
		"functionCode":  int(fb.FunctionQueryProtocolVersion),
		"dataHex":       "",
	}
	for _, override := range overrides {
		for key, value := range override {
			config[key] = value
		}
	}
	return config
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

func waitGraphNodeStatus(t *testing.T, svc *Service, graphID string, nodeID string, want string) SerialGraphNodeStatus {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	var last SerialGraphNodeStatus
	for time.Now().Before(deadline) {
		info, err := svc.GetSerialGraphStatus(graphID)
		if err == nil {
			last = graphNodeStatus(info, nodeID)
			if last.Status == want {
				return last
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("%s status = %q, want %q (last status: %+v)", nodeID, last.Status, want, last)
	return last
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
