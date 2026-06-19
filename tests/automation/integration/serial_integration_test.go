//go:build integration

// Package integration 提供后端的真实集成测试。
// 使用 socat 创建虚拟串口对，验证端到端数据流。
//
// 运行: go test -tags integration -v ./tests/automation/integration/...
package integration

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	goserial "go.bug.st/serial"

	"github.com/suyue/mocktrue/internal/modules/serial"
	"github.com/suyue/mocktrue/internal/modules/serial/manager"
	"github.com/suyue/mocktrue/internal/modules/serial/port"
	"github.com/suyue/mocktrue/tests/automation/helpers"
)

// TestSerialPortLifecycle 测试串口完整生命周期
func TestSerialPortLifecycle(t *testing.T) {
	helpers.SkipIfNoSocat(t)
	env := helpers.NewTestEnv(t)

	// 1. 创建虚拟串口对
	pair, err := env.CreateVirtualPair("lifecycle", "lifeA", "lifeB")
	if err != nil {
		t.Fatalf("CreateVirtualPair: %v", err)
	}
	t.Logf("✓ 虚拟串口对创建: %s ↔ %s", pair.Port1, pair.Port2)

	// 2. 通过 Service.OpenPort 打开端口
	status, err := env.Service.OpenPort(context.Background(), manager.OpenRequest{
		Config: port.SerialConfig{
			PortName: pair.Port1,
			BaudRate: 115200,
			DataBits: 8,
			StopBits: "1",
			Parity:   "none",
			FlowMode: "none",
		},
	})
	if err != nil {
		t.Fatalf("OpenPort: %v", err)
	}
	t.Logf("✓ 端口打开: ID=%s", status.ID)

	// 3. 验证已在列表中
	list := env.Service.ListPorts()
	if len(list) != 1 {
		t.Errorf("expected 1 open port, got %d", len(list))
	}

	// 4. 通过另一端写入数据，验证接收
	otherEnd, err := goserial.Open(pair.Port2, &goserial.Mode{BaudRate: 115200})
	if err != nil {
		t.Fatalf("Open other end: %v", err)
	}
	defer otherEnd.Close()

	testData := []byte("Hello from lifecycle test!")

	// 订阅 data 事件
	receivedCount := atomic.Int64{}
	cancel := env.Bus.Subscribe("serial:data", func(payload any) {
		receivedCount.Add(1)
	})
	defer cancel()

	if _, err := otherEnd.Write(testData); err != nil {
		t.Fatalf("Write: %v", err)
	}

	// 等待事件
	helpers.AssertEventually(t, "等待 serial:data 事件", 2*time.Second, func() bool {
		return receivedCount.Load() > 0
	})
	t.Logf("✓ 接收到 %d 个 data 事件", receivedCount.Load())

	// 5. QueryPage 验证数据存储
	time.Sleep(200 * time.Millisecond)
	snap, err := env.Service.QueryPage(status.ID, 0, 256)
	if err != nil {
		t.Fatalf("QueryPage: %v", err)
	}
	if snap.Total < int64(len(testData)) {
		t.Errorf("buffer total = %d, expected >= %d", snap.Total, len(testData))
	}
	t.Logf("✓ Buffer 总字节数: %d", snap.Total)

	// 6. Send 写入另一端并更新 TX 统计
	otherEnd.SetReadTimeout(2 * time.Second)
	sent, err := env.Service.Send(serial.SendRequest{
		PortID:  status.ID,
		Content: "service send",
		Mode:    "ascii",
	})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if sent != len("service send") {
		t.Fatalf("Send wrote %d bytes, want %d", sent, len("service send"))
	}
	readBuf := make([]byte, 64)
	n, err := otherEnd.Read(readBuf)
	if err != nil {
		t.Fatalf("Read sent data: %v", err)
	}
	if string(readBuf[:n]) != "service send" {
		t.Fatalf("sent data = %q, want %q", readBuf[:n], "service send")
	}
	list = env.Service.ListPorts()
	if len(list) != 1 || list[0].TxBytes < int64(sent) {
		t.Fatalf("TxBytes not updated: %+v", list)
	}
	t.Logf("✓ Send 写入并更新 TX 统计: %d 字节", sent)

	// 7. HEX Send 解码后写入
	sent, err = env.Service.Send(serial.SendRequest{
		PortID:  status.ID,
		Content: "aa 55 01",
		Mode:    "hex",
	})
	if err != nil {
		t.Fatalf("Send hex: %v", err)
	}
	if sent != 3 {
		t.Fatalf("HEX Send wrote %d bytes, want 3", sent)
	}
	n, err = otherEnd.Read(readBuf)
	if err != nil {
		t.Fatalf("Read hex sent data: %v", err)
	}
	if got := readBuf[:n]; string(got) != string([]byte{0xaa, 0x55, 0x01}) {
		t.Fatalf("hex sent data = % x, want aa 55 01", got)
	}
	t.Logf("✓ HEX Send 解码成功")

	// 8. 关闭端口
	if err := env.Service.ClosePort(status.ID); err != nil {
		t.Fatalf("ClosePort: %v", err)
	}

	list = env.Service.ListPorts()
	if len(list) != 0 {
		t.Errorf("expected 0 open ports after close, got %d", len(list))
	}
	if _, err := env.Service.QueryPage(status.ID, 0, 1); err == nil {
		t.Fatalf("QueryPage after close must fail")
	}
	t.Logf("✓ 端口已关闭")
}

// TestVirtualPairDataTransfer 测试虚拟串口对数据传输
func TestVirtualPairDataTransfer(t *testing.T) {
	helpers.SkipIfNoSocat(t)
	env := helpers.NewTestEnv(t)

	pair, err := env.CreateVirtualPair("transfer", "trA", "trB")
	if err != nil {
		t.Fatalf("CreateVirtualPair: %v", err)
	}

	port1, err := goserial.Open(pair.Port1, &goserial.Mode{BaudRate: 115200})
	if err != nil {
		t.Fatalf("Open port1: %v", err)
	}
	defer port1.Close()

	port2, err := goserial.Open(pair.Port2, &goserial.Mode{BaudRate: 115200})
	if err != nil {
		t.Fatalf("Open port2: %v", err)
	}
	defer port2.Close()
	port2.SetReadTimeout(2 * time.Second)

	// 双向测试
	tests := []struct {
		name    string
		data    string
		fromOne bool
	}{
		{"port1→port2 ASCII", "Hello!", true},
		{"port2→port1 ASCII", "World!", false},
		{"port1→port2 二进制", string([]byte{0x00, 0x01, 0xFF, 0xAA}), true},
		{"长数据", string(make([]byte, 1024)), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sender, receiver goserial.Port
			if tt.fromOne {
				sender, receiver = port1, port2
			} else {
				sender, receiver = port2, port1
			}
			receiver.SetReadTimeout(2 * time.Second)

			data := []byte(tt.data)
			if _, err := sender.Write(data); err != nil {
				t.Fatalf("Write: %v", err)
			}

			buf := make([]byte, len(data))
			total := 0
			deadline := time.Now().Add(2 * time.Second)
			for total < len(data) && time.Now().Before(deadline) {
				n, err := receiver.Read(buf[total:])
				if err != nil {
					t.Fatalf("Read: %v", err)
				}
				total += n
			}

			if string(buf[:total]) != tt.data {
				t.Errorf("数据不匹配")
			} else {
				t.Logf("✓ 传输 %d 字节成功", len(data))
			}
		})
	}
}

// TestBridgeConnectsTwoPairs 测试桥接连接两对虚拟串口
func TestBridgeConnectsTwoPairs(t *testing.T) {
	helpers.SkipIfNoSocat(t)
	env := helpers.NewTestEnv(t)

	// 创建两对虚拟串口
	pairA, err := env.CreateVirtualPair("pairA", "ttyA0", "ttyA1")
	if err != nil {
		t.Fatalf("Create pairA: %v", err)
	}
	pairB, err := env.CreateVirtualPair("pairB", "ttyB0", "ttyB1")
	if err != nil {
		t.Fatalf("Create pairB: %v", err)
	}

	// 桥接 A1 ↔ B0
	bridge, err := env.CreateBridge("bridge1", pairA.Port2, pairB.Port1, 115200)
	if err != nil {
		t.Fatalf("CreateBridge: %v", err)
	}
	t.Logf("✓ 桥接创建: %s ⇌ %s", bridge.Port1, bridge.Port2)

	// 数据流: A0 → A1 →(bridge)→ B0 → B1
	a0, err := goserial.Open(pairA.Port1, &goserial.Mode{BaudRate: 115200})
	if err != nil {
		t.Fatalf("Open A0: %v", err)
	}
	defer a0.Close()

	b1, err := goserial.Open(pairB.Port2, &goserial.Mode{BaudRate: 115200})
	if err != nil {
		t.Fatalf("Open B1: %v", err)
	}
	defer b1.Close()
	b1.SetReadTimeout(3 * time.Second)

	testData := []byte("Bridge transfer test!")
	if _, err := a0.Write(testData); err != nil {
		t.Fatalf("Write A0: %v", err)
	}

	buf := make([]byte, len(testData))
	total := 0
	deadline := time.Now().Add(3 * time.Second)
	for total < len(testData) && time.Now().Before(deadline) {
		n, err := b1.Read(buf[total:])
		if err != nil {
			break
		}
		total += n
	}

	if string(buf[:total]) != string(testData) {
		t.Errorf("桥接传输失败: 期望 %q, 实际 %q", testData, buf[:total])
	} else {
		t.Logf("✓ 桥接传输成功: A0→A1→bridge→B0→B1 = %q", testData)
	}

	// 反向测试
	b1.SetReadTimeout(0)
	a0.SetReadTimeout(3 * time.Second)
	reverseData := []byte("Reverse direction!")
	if _, err := b1.Write(reverseData); err != nil {
		t.Fatalf("Write B1: %v", err)
	}

	buf = make([]byte, len(reverseData))
	total = 0
	deadline = time.Now().Add(3 * time.Second)
	for total < len(reverseData) && time.Now().Before(deadline) {
		n, err := a0.Read(buf[total:])
		if err != nil {
			break
		}
		total += n
	}

	if string(buf[:total]) != string(reverseData) {
		t.Errorf("反向桥接失败: 期望 %q, 实际 %q", reverseData, buf[:total])
	} else {
		t.Logf("✓ 反向桥接成功: B1→B0→bridge→A1→A0 = %q", reverseData)
	}
}

// TestMultipleVirtualPairs 测试同时创建多个虚拟串口对
func TestMultipleVirtualPairs(t *testing.T) {
	helpers.SkipIfNoSocat(t)
	env := helpers.NewTestEnv(t)

	const count = 5
	for i := 0; i < count; i++ {
		id := fmt.Sprintf("multi-%d", i)
		_, err := env.CreateVirtualPair(id,
			fmt.Sprintf("multiA%d", i),
			fmt.Sprintf("multiB%d", i))
		if err != nil {
			t.Fatalf("Create pair %d: %v", i, err)
		}
	}

	pairs := env.Service.ListVirtualPairs()
	if len(pairs) != count {
		t.Errorf("expected %d pairs, got %d", count, len(pairs))
	}
	t.Logf("✓ 成功创建 %d 个虚拟串口对", count)
}

// TestVirtualPortsAppearInEnumeration 测试用户创建的虚拟串口出现在前端可用端口列表中。
func TestVirtualPortsAppearInEnumeration(t *testing.T) {
	helpers.SkipIfNoSocat(t)
	env := helpers.NewTestEnv(t)

	vport, err := env.Service.CreateVirtualPort(context.Background(), "enumerate", "enumA")
	if err != nil {
		t.Fatalf("CreateVirtualPort: %v", err)
	}
	t.Cleanup(func() { _ = env.Service.DeleteVirtualPort("enumerate") })

	ports, err := env.Service.EnumeratePorts(context.Background())
	if err != nil {
		t.Fatalf("EnumeratePorts: %v", err)
	}
	var names []string
	for _, p := range ports {
		names = append(names, p.Name)
	}
	joined := strings.Join(names, "\n")
	if !strings.Contains(joined, vport.Port) {
		t.Fatalf("virtual port missing from enumeration: %v", names)
	}
	if strings.Contains(joined, "enumA-peer") {
		t.Fatalf("hidden virtual peer leaked into enumeration: %v", names)
	}
	t.Logf("✓ 虚拟端口出现在枚举列表: %s", vport.Port)
}

// TestMultipleSerialPortsParallel 测试多串口并行收发
func TestMultipleSerialPortsParallel(t *testing.T) {
	helpers.SkipIfNoSocat(t)
	env := helpers.NewTestEnv(t)

	pair1, _ := env.CreateVirtualPair("para1", "paraA0", "paraA1")
	pair2, _ := env.CreateVirtualPair("para2", "paraB0", "paraB1")

	// 通过 Service 打开两个端口
	s1, err := env.Service.OpenPort(context.Background(), manager.OpenRequest{
		Config: port.SerialConfig{PortName: pair1.Port1, BaudRate: 115200},
	})
	if err != nil {
		t.Fatalf("OpenPort 1: %v", err)
	}

	s2, err := env.Service.OpenPort(context.Background(), manager.OpenRequest{
		Config: port.SerialConfig{PortName: pair2.Port1, BaudRate: 115200},
	})
	if err != nil {
		t.Fatalf("OpenPort 2: %v", err)
	}

	// 验证两个端口都打开
	list := env.Service.ListPorts()
	if len(list) != 2 {
		t.Errorf("expected 2 open ports, got %d", len(list))
	}

	// 双方写入数据
	other1, _ := goserial.Open(pair1.Port2, &goserial.Mode{BaudRate: 115200})
	defer other1.Close()
	other2, _ := goserial.Open(pair2.Port2, &goserial.Mode{BaudRate: 115200})
	defer other2.Close()

	other1.Write([]byte("data for port1"))
	other2.Write([]byte("data for port2"))

	time.Sleep(300 * time.Millisecond)

	// 验证两个 buffer 各自接收了数据
	snap1, _ := env.Service.QueryPage(s1.ID, 0, 256)
	snap2, _ := env.Service.QueryPage(s2.ID, 0, 256)

	if snap1.Total == 0 || snap2.Total == 0 {
		t.Errorf("buffer empty: snap1=%d, snap2=%d", snap1.Total, snap2.Total)
	}
	t.Logf("✓ 端口1 buffer: %d 字节, 端口2 buffer: %d 字节", snap1.Total, snap2.Total)
}

// TestErrorHandling 测试错误处理
func TestErrorHandling(t *testing.T) {
	env := helpers.NewTestEnv(t)
	ctx := context.Background()

	tests := []struct {
		name string
		fn   func() error
	}{
		{
			"空 ID 创建虚拟对",
			func() error {
				_, err := env.Service.CreateVirtualPair(ctx, "", "a", "b")
				return err
			},
		},
		{
			"空端口名创建虚拟对",
			func() error {
				_, err := env.Service.CreateVirtualPair(ctx, "id1", "", "")
				return err
			},
		},
		{
			"删除不存在的虚拟对",
			func() error {
				return env.Service.DeleteVirtualPair("nonexistent")
			},
		},
		{
			"自我桥接",
			func() error {
				_, err := env.Service.CreateBridge("self", "/tmp/x", "/tmp/x", 115200)
				return err
			},
		},
		{
			"删除不存在的桥接",
			func() error {
				return env.Service.DeleteBridge("nonexistent")
			},
		},
		{
			"发送空内容",
			func() error {
				_, err := env.Service.Send(serial.SendRequest{PortID: "missing", Content: "", Mode: "ascii"})
				return err
			},
		},
		{
			"发送非法 HEX",
			func() error {
				_, err := env.Service.Send(serial.SendRequest{PortID: "missing", Content: "not-hex", Mode: "hex"})
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.fn(); err == nil {
				t.Errorf("期望错误但收到 nil")
			} else {
				t.Logf("✓ 正确返回错误: %v", err)
			}
		})
	}
}
