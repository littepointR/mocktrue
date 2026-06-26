package porttest

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sort"
	"sync"
	"time"

	"github.com/littepointR/portweave/internal/modules/serial/port"
)

// MemoryBackend is a deterministic in-memory implementation of port.Backend for
// tests. Add linked endpoints with AddPair, then open each endpoint by name.
type MemoryBackend struct {
	mu        sync.Mutex
	endpoints map[string]*memoryEndpoint
}

// NewMemoryBackend creates an empty memory serial backend.
func NewMemoryBackend() *MemoryBackend {
	return &MemoryBackend{endpoints: make(map[string]*memoryEndpoint)}
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

	b.endpoints[a] = &memoryEndpoint{name: a, peerName: c, data: make(chan byte, 4096)}
	b.endpoints[c] = &memoryEndpoint{name: c, peerName: a, data: make(chan byte, 4096)}
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

// Open returns a fresh handle for a registered memory endpoint.
func (b *MemoryBackend) Open(cfg port.SerialConfig) (port.Port, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	endpoint := b.endpoints[cfg.PortName]
	if endpoint == nil {
		return nil, errors.New("unknown memory serial port")
	}
	if endpoint.active != nil && !endpoint.active.isClosed() {
		return nil, fmt.Errorf("memory serial port already open: %s", cfg.PortName)
	}

	p := newMemoryPort(b, endpoint)
	endpoint.active = p
	return p, nil
}

type memoryEndpoint struct {
	name     string
	peerName string
	data     chan byte
	active   *memoryPort
}

type memoryPort struct {
	backend  *MemoryBackend
	endpoint *memoryEndpoint
	closed   chan struct{}
	once     sync.Once
	mu       sync.RWMutex
	timeout  time.Duration
}

func newMemoryPort(backend *MemoryBackend, endpoint *memoryEndpoint) *memoryPort {
	return &memoryPort{
		backend:  backend,
		endpoint: endpoint,
		closed:   make(chan struct{}),
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
		case b := <-p.endpoint.data:
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
		case b := <-p.endpoint.data:
			return b, nil
		case <-timer.C:
			return 0, errors.New("memory serial read timeout")
		}
	}

	select {
	case <-p.closed:
		return 0, io.ErrClosedPipe
	case b := <-p.endpoint.data:
		return b, nil
	}
}

func (p *memoryPort) Write(buf []byte) (int, error) {
	for i, b := range buf {
		peer, err := p.peerEndpoint()
		if err != nil {
			return i, err
		}
		select {
		case <-p.closed:
			return i, io.ErrClosedPipe
		case peer.data <- b:
		}
	}
	return len(buf), nil
}

func (p *memoryPort) peerEndpoint() (*memoryEndpoint, error) {
	p.backend.mu.Lock()
	defer p.backend.mu.Unlock()

	if p.isClosed() {
		return nil, io.ErrClosedPipe
	}
	peerEndpoint := p.backend.endpoints[p.endpoint.peerName]
	if peerEndpoint == nil {
		return nil, io.ErrClosedPipe
	}
	return peerEndpoint, nil
}

func (p *memoryPort) SetReadTimeout(timeout time.Duration) error {
	p.mu.Lock()
	p.timeout = timeout
	p.mu.Unlock()
	return nil
}

func (p *memoryPort) Close() error {
	p.once.Do(func() {
		close(p.closed)
		p.backend.mu.Lock()
		if p.endpoint.active == p {
			p.endpoint.active = nil
		}
		p.backend.mu.Unlock()
	})
	return nil
}

func (p *memoryPort) isClosed() bool {
	select {
	case <-p.closed:
		return true
	default:
		return false
	}
}
