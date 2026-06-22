package porttest

import (
	"context"
	"errors"
	"io"
	"sort"
	"sync"
	"time"

	"github.com/littepointR/mocktrue/internal/modules/serial/port"
)

// MemoryBackend is a deterministic in-memory implementation of port.Backend for
// tests. Add linked endpoints with AddPair, then open each endpoint by name.
type MemoryBackend struct {
	mu        sync.Mutex
	endpoints map[string]*memoryPort
}

// NewMemoryBackend creates an empty memory serial backend.
func NewMemoryBackend() *MemoryBackend {
	return &MemoryBackend{endpoints: make(map[string]*memoryPort)}
}

// DefaultSerialConfig returns a standard 115200-8N1 config for tests.
func DefaultSerialConfig(portName string) port.SerialConfig {
	return port.SerialConfig{
		PortName: portName,
		BaudRate: 115200,
		DataBits: 8,
		StopBits: "1",
		Parity:   "none",
		FlowMode: "none",
	}
}

// AddPair registers two linked in-memory serial endpoints.
func (b *MemoryBackend) AddPair(a, c string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	left := newMemoryPort(a)
	right := newMemoryPort(c)
	left.peer = right
	right.peer = left
	b.endpoints[a] = left
	b.endpoints[c] = right
}

// Enumerate returns registered memory endpoints sorted by name.
func (b *MemoryBackend) Enumerate(ctx context.Context) ([]port.PortInfo, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	names := make([]string, 0, len(b.endpoints))
	for name := range b.endpoints {
		names = append(names, name)
	}
	sort.Strings(names)

	ports := make([]port.PortInfo, 0, len(names))
	for _, name := range names {
		ports = append(ports, port.PortInfo{Name: name, FriendlyName: name})
	}
	return ports, nil
}

// Open returns a registered memory endpoint.
func (b *MemoryBackend) Open(cfg port.SerialConfig) (port.Port, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	endpoint := b.endpoints[cfg.PortName]
	if endpoint == nil {
		return nil, errors.New("unknown memory serial port")
	}
	return endpoint, nil
}

type memoryPort struct {
	name    string
	peer    *memoryPort
	data    chan byte
	closed  chan struct{}
	once    sync.Once
	mu      sync.RWMutex
	timeout time.Duration
}

func newMemoryPort(name string) *memoryPort {
	return &memoryPort{
		name:   name,
		data:   make(chan byte, 4096),
		closed: make(chan struct{}),
	}
}

func (p *memoryPort) Read(buf []byte) (int, error) {
	if len(buf) == 0 {
		return 0, nil
	}

	first, err := p.readByte()
	if err != nil {
		return 0, err
	}
	buf[0] = first
	n := 1

	for n < len(buf) {
		select {
		case b := <-p.data:
			buf[n] = b
			n++
		default:
			return n, nil
		}
	}
	return n, nil
}

func (p *memoryPort) readByte() (byte, error) {
	p.mu.RLock()
	timeout := p.timeout
	p.mu.RUnlock()

	if timeout > 0 {
		timer := time.NewTimer(timeout)
		defer timer.Stop()
		select {
		case <-p.closed:
			return 0, io.ErrClosedPipe
		case b := <-p.data:
			return b, nil
		case <-timer.C:
			return 0, errors.New("memory serial read timeout")
		}
	}

	select {
	case <-p.closed:
		return 0, io.ErrClosedPipe
	case b := <-p.data:
		return b, nil
	}
}

func (p *memoryPort) Write(buf []byte) (int, error) {
	if p.peer == nil {
		return 0, io.ErrClosedPipe
	}
	for i, b := range buf {
		select {
		case <-p.closed:
			return i, io.ErrClosedPipe
		case <-p.peer.closed:
			return i, io.ErrClosedPipe
		case p.peer.data <- b:
		}
	}
	return len(buf), nil
}

func (p *memoryPort) SetReadTimeout(timeout time.Duration) error {
	p.mu.Lock()
	p.timeout = timeout
	p.mu.Unlock()
	return nil
}

func (p *memoryPort) Close() error {
	p.once.Do(func() { close(p.closed) })
	return nil
}
