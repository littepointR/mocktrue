// Package helpers provides shared utilities for integration and E2E tests.
package helpers

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/littepointR/mocktrue/internal/core/eventbus"
	"github.com/littepointR/mocktrue/internal/modules/serial"
)

// TestEnv 提供测试环境的完整生命周期管理
type TestEnv struct {
	T       *testing.T
	Bus     *eventbus.EventBus
	Service *serial.Service
	mu      sync.Mutex
	cleanup []func()
}

// NewTestEnv 创建测试环境
func NewTestEnv(t *testing.T) *TestEnv {
	t.Helper()
	bus := eventbus.New()
	svc := serial.NewService(bus)

	env := &TestEnv{
		T:       t,
		Bus:     bus,
		Service: svc,
	}

	t.Cleanup(env.Cleanup)
	return env
}

// AddCleanup 添加清理函数
func (e *TestEnv) AddCleanup(fn func()) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.cleanup = append(e.cleanup, fn)
}

// Cleanup 清理所有资源
func (e *TestEnv) Cleanup() {
	e.mu.Lock()
	defer e.mu.Unlock()
	for i := len(e.cleanup) - 1; i >= 0; i-- {
		e.cleanup[i]()
	}
	e.Service.CleanupVirtual()
}

// CreateVirtualPair 创建虚拟串口对（自动清理）
func (e *TestEnv) CreateVirtualPair(id, port1, port2 string) (*serial.VirtualPairInfo, error) {
	pair, err := e.Service.CreateVirtualPair(context.Background(), id, port1, port2)
	if err != nil {
		return nil, err
	}
	e.AddCleanup(func() {
		_ = e.Service.DeleteVirtualPair(id)
	})
	// 等待 socat 完全启动
	time.Sleep(150 * time.Millisecond)
	return pair, nil
}

// CreateBridge 创建桥接（自动清理）
func (e *TestEnv) CreateBridge(id, port1, port2 string, baudRate int) (*serial.BridgeInfo, error) {
	bridge, err := e.Service.CreateBridge(id, port1, port2, baudRate)
	if err != nil {
		return nil, err
	}
	e.AddCleanup(func() {
		_ = e.Service.DeleteBridge(id)
	})
	// 等待 bridge goroutine 启动
	time.Sleep(150 * time.Millisecond)
	return bridge, nil
}

// AssertEventually 在超时时间内反复检查条件，直到满足或超时
func AssertEventually(t *testing.T, msg string, timeout time.Duration, check func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if check() {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("Timeout (%v): %s", timeout, msg)
}

// SocatAvailable 检查 socat 是否可用
func SocatAvailable() bool {
	_, err := exec.LookPath("socat")
	return err == nil
}

// SkipIfNoSocat 如果 socat 不可用则跳过测试
func SkipIfNoSocat(t *testing.T) {
	t.Helper()
	if !SocatAvailable() {
		t.Skip("socat not available")
	}
}

// AppExecutable 返回 mocktrue 可执行文件路径
func AppExecutable() string {
	wd, _ := os.Getwd()
	// 从 tests/go/integration 或类似位置回到项目根
	for i := 0; i < 5; i++ {
		bin := filepath.Join(wd, "bin", "mocktrue")
		if _, err := os.Stat(bin); err == nil {
			return bin
		}
		wd = filepath.Dir(wd)
	}
	return ""
}

// BuildPortName 生成唯一的端口名（避免并发测试冲突）
func BuildPortName(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
}
