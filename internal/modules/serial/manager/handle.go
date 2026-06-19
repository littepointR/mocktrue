package manager

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"

	"github.com/suyue/mocktrue/internal/core/eventbus"
	"github.com/suyue/mocktrue/internal/modules/serial/port"
)

// Handle wraps a single open serial port with a background read loop.
type Handle struct {
	id      string
	config  port.SerialConfig
	port    port.Port
	bus     *eventbus.EventBus
	cancel  context.CancelFunc
	stopped atomic.Bool

	mu      sync.RWMutex
	rxBytes int64
	txBytes int64
}

func newHandle(id string, config port.SerialConfig, p port.Port, bus *eventbus.EventBus) *Handle {
	return &Handle{
		id:     id,
		config: config,
		port:   p,
		bus:    bus,
	}
}

// Status returns an immutable snapshot of the handle's current state.
func (h *Handle) Status() *HandleStatus {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return &HandleStatus{
		ID:      h.id,
		Config:  h.config,
		IsOpen:  !h.stopped.Load(),
		RxBytes: h.rxBytes,
		TxBytes: h.txBytes,
	}
}

// start launches the background read loop.
func (h *Handle) start(parent context.Context) {
	_ = parent
	ctx, cancel := context.WithCancel(context.Background())
	h.cancel = cancel
	go h.readLoop(ctx)
}

// stop cancels the read loop and closes the port.
func (h *Handle) stop() {
	if h.stopped.Swap(true) {
		return // already stopped
	}
	if h.cancel != nil {
		h.cancel()
	}
	_ = h.port.Close()
}

// write sends bytes to the underlying serial port and updates TX stats.
func (h *Handle) write(data []byte) (int, error) {
	n, err := h.port.Write(data)
	if n > 0 {
		h.mu.Lock()
		h.txBytes += int64(n)
		h.mu.Unlock()
	}
	return n, err
}

func (h *Handle) resetCounters() {
	h.mu.Lock()
	h.rxBytes = 0
	h.txBytes = 0
	h.mu.Unlock()
}

// readLoop reads bytes from the port and emits DataEvents. It runs until
// the context is cancelled or the port is closed.
func (h *Handle) readLoop(ctx context.Context) {
	buf := make([]byte, 4096)
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		n, err := h.port.Read(buf)
		if err != nil {
			if h.stopped.Load() {
				return // expected: port was closed
			}
			slog.Error("serial read error", slog.String("id", h.id), slog.Any("err", err))
			return
		}
		if n > 0 {
			// Copy bytes so the caller can safely hold the slice
			data := make([]byte, n)
			copy(data, buf[:n])

			h.mu.Lock()
			h.rxBytes += int64(n)
			h.mu.Unlock()

			h.bus.Publish("serial:data", DataEvent{
				PortID: h.id,
				Data:   data,
			})
		}
	}
}
