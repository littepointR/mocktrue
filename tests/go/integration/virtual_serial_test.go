//go:build (darwin || linux) && integration

package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	goserial "go.bug.st/serial"

	"github.com/littepointR/portweave/internal/core/eventbus"
	"github.com/littepointR/portweave/internal/modules/serial"
	"github.com/littepointR/portweave/tests/automation/helpers"
)

// TestVirtualPairLifecycle 测试虚拟串口对的完整生命周期
func TestVirtualPairLifecycle(t *testing.T) {
	helpers.SkipIfNoSocat(t)
	bus := eventbus.New()
	svc := serial.NewService(bus)
	defer svc.CleanupVirtual()

	ctx := context.Background()

	// 1. 创建虚拟串口对
	pair, err := svc.CreateVirtualPair(ctx, "test-pair-1", "testV0", "testV1")
	if err != nil {
		t.Fatalf("CreateVirtualPair failed: %v", err)
	}
	if pair.ID != "test-pair-1" {
		t.Errorf("expected ID 'test-pair-1', got %q", pair.ID)
	}
	t.Logf("✓ Created pair: %s (%s ↔ %s)", pair.ID, pair.Port1, pair.Port2)

	// 2. 列举虚拟串口对
	pairs := svc.ListVirtualPairs()
	if len(pairs) != 1 {
		t.Errorf("expected 1 pair, got %d", len(pairs))
	}
	t.Logf("✓ Listed %d pair(s)", len(pairs))

	// 3. 测试数据传输
	port1, err := goserial.Open(pair.Port1, &goserial.Mode{BaudRate: 115200})
	if err != nil {
		t.Fatalf("Open port1 failed: %v", err)
	}
	defer port1.Close()

	port2, err := goserial.Open(pair.Port2, &goserial.Mode{BaudRate: 115200})
	if err != nil {
		t.Fatalf("Open port2 failed: %v", err)
	}
	defer port2.Close()

	testData := []byte("Hello Virtual Serial!")
	n, err := port1.Write(testData)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if n != len(testData) {
		t.Errorf("wrote %d bytes, expected %d", n, len(testData))
	}

	// 设置读取超时
	port2.SetReadTimeout(2 * time.Second)
	buf := make([]byte, 256)
	n, err = port2.Read(buf)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	received := string(buf[:n])
	if received != string(testData) {
		t.Errorf("expected %q, got %q", testData, received)
	}
	t.Logf("✓ Data transmitted: %q", received)

	// 4. 删除虚拟串口对
	if err := svc.DeleteVirtualPair("test-pair-1"); err != nil {
		t.Fatalf("DeleteVirtualPair failed: %v", err)
	}

	pairs = svc.ListVirtualPairs()
	if len(pairs) != 0 {
		t.Errorf("expected 0 pairs after delete, got %d", len(pairs))
	}
	t.Logf("✓ Deleted pair successfully")
}

// TestBridgeLifecycle 测试串口桥接的完整生命周期
func TestBridgeLifecycle(t *testing.T) {
	helpers.SkipIfNoSocat(t)
	bus := eventbus.New()
	svc := serial.NewService(bus)
	defer svc.CleanupVirtual()

	ctx := context.Background()

	// 创建两个虚拟串口对
	pair1, err := svc.CreateVirtualPair(ctx, "pair-A", "bridgeA0", "bridgeA1")
	if err != nil {
		t.Fatalf("Create pair1 failed: %v", err)
	}

	pair2, err := svc.CreateVirtualPair(ctx, "pair-B", "bridgeB0", "bridgeB1")
	if err != nil {
		t.Fatalf("Create pair2 failed: %v", err)
	}

	// 等待 socat 完全启动
	time.Sleep(200 * time.Millisecond)

	// 创建桥接：pair1.Port2 ↔ pair2.Port1
	// 数据流：bridgeA0 → bridgeA1 →(bridge)→ bridgeB0 → bridgeB1
	bridge, err := svc.CreateBridge("test-bridge", pair1.Port2, pair2.Port1, 115200)
	if err != nil {
		t.Fatalf("CreateBridge failed: %v", err)
	}
	t.Logf("✓ Created bridge: %s (%s ⇌ %s @ %d bps)",
		bridge.ID, bridge.Port1, bridge.Port2, bridge.BaudRate)

	// 等待桥接 goroutine 启动
	time.Sleep(200 * time.Millisecond)

	// 列举桥接
	bridges := svc.ListBridges()
	if len(bridges) != 1 {
		t.Errorf("expected 1 bridge, got %d", len(bridges))
	}

	// 测试桥接数据传输
	// 写入 bridgeA0 → 应该在 bridgeB1 收到
	portA0, err := goserial.Open(pair1.Port1, &goserial.Mode{BaudRate: 115200})
	if err != nil {
		t.Fatalf("Open A0 failed: %v", err)
	}
	defer portA0.Close()

	portB1, err := goserial.Open(pair2.Port2, &goserial.Mode{BaudRate: 115200})
	if err != nil {
		t.Fatalf("Open B1 failed: %v", err)
	}
	defer portB1.Close()

	portB1.SetReadTimeout(2 * time.Second)

	testData := []byte("Bridged Data!")
	if _, err := portA0.Write(testData); err != nil {
		t.Fatalf("Write to A0 failed: %v", err)
	}

	// 读取 B1 端
	buf := make([]byte, 256)
	totalRead := 0
	deadline := time.Now().Add(3 * time.Second)
	for totalRead < len(testData) && time.Now().Before(deadline) {
		n, err := portB1.Read(buf[totalRead:])
		if err != nil {
			break
		}
		totalRead += n
	}

	received := string(buf[:totalRead])
	if received != string(testData) {
		t.Errorf("bridge transmission failed: expected %q, got %q", testData, received)
	} else {
		t.Logf("✓ Bridge transmitted: %q (A0 → A1 → bridge → B0 → B1)", received)
	}

	// 删除桥接
	if err := svc.DeleteBridge("test-bridge"); err != nil {
		t.Fatalf("DeleteBridge failed: %v", err)
	}

	bridges = svc.ListBridges()
	if len(bridges) != 0 {
		t.Errorf("expected 0 bridges after delete, got %d", len(bridges))
	}
	t.Logf("✓ Deleted bridge successfully")
}

// TestMultipleVirtualPairs 测试创建多个虚拟串口对
func TestMultipleVirtualPairs(t *testing.T) {
	helpers.SkipIfNoSocat(t)
	bus := eventbus.New()
	svc := serial.NewService(bus)
	defer svc.CleanupVirtual()

	ctx := context.Background()

	pairCount := 3
	for i := 0; i < pairCount; i++ {
		id := fmt.Sprintf("multi-pair-%d", i)
		port1 := fmt.Sprintf("multi%dA", i)
		port2 := fmt.Sprintf("multi%dB", i)

		_, err := svc.CreateVirtualPair(ctx, id, port1, port2)
		if err != nil {
			t.Fatalf("Create pair %d failed: %v", i, err)
		}
	}

	pairs := svc.ListVirtualPairs()
	if len(pairs) != pairCount {
		t.Errorf("expected %d pairs, got %d", pairCount, len(pairs))
	}
	t.Logf("✓ Created %d virtual pairs", len(pairs))
}

// TestDuplicatePairIDRejected 测试重复 ID 被拒绝
func TestDuplicatePairIDRejected(t *testing.T) {
	helpers.SkipIfNoSocat(t)
	bus := eventbus.New()
	svc := serial.NewService(bus)
	defer svc.CleanupVirtual()

	ctx := context.Background()

	if _, err := svc.CreateVirtualPair(ctx, "dup-id", "dupA", "dupB"); err != nil {
		t.Fatalf("First create failed: %v", err)
	}

	if _, err := svc.CreateVirtualPair(ctx, "dup-id", "dupC", "dupD"); err == nil {
		t.Error("expected error for duplicate ID, got nil")
	} else {
		t.Logf("✓ Duplicate ID rejected: %v", err)
	}
}

// TestEmptyIDRejected 测试空 ID 被拒绝
func TestEmptyIDRejected(t *testing.T) {
	bus := eventbus.New()
	svc := serial.NewService(bus)
	defer svc.CleanupVirtual()

	ctx := context.Background()

	if _, err := svc.CreateVirtualPair(ctx, "", "a", "b"); err == nil {
		t.Error("expected error for empty ID, got nil")
	} else {
		t.Logf("✓ Empty ID rejected: %v", err)
	}

	if _, err := svc.CreateBridge("", "a", "b", 115200); err == nil {
		t.Error("expected error for empty bridge ID, got nil")
	} else {
		t.Logf("✓ Empty bridge ID rejected: %v", err)
	}
}

// TestSelfBridgeRejected 测试自我桥接被拒绝
func TestSelfBridgeRejected(t *testing.T) {
	helpers.SkipIfNoSocat(t)
	bus := eventbus.New()
	svc := serial.NewService(bus)
	defer svc.CleanupVirtual()

	ctx := context.Background()

	pair, err := svc.CreateVirtualPair(ctx, "self-test", "selfA", "selfB")
	if err != nil {
		t.Fatalf("Create pair failed: %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	if _, err := svc.CreateBridge("self-bridge", pair.Port1, pair.Port1, 115200); err == nil {
		t.Error("expected error for self-bridge, got nil")
	} else {
		t.Logf("✓ Self-bridge rejected: %v", err)
	}
}
