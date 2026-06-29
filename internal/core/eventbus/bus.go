package eventbus

import (
	"log/slog"
	"sync"
	"sync/atomic"
)

// Bridge is the minimal surface EventBus needs to forward events to the
// Wails frontend. *application.App satisfies it via app.Event.Emit.
type Bridge interface {
	Emit(name string, data ...any) bool
}

// EventBus is an in-process pub/sub bus that optionally forwards every
// Publish to the frontend through a Bridge.
type EventBus struct {
	mu     sync.RWMutex
	subs   map[string][]handler
	bridge Bridge
	logger *slog.Logger
}

type handler struct {
	id int64
	fn func(data any)
}

var nextHandlerID atomic.Int64

// New constructs an EventBus with no bridge and a discard logger.
func New() *EventBus {
	return &EventBus{
		subs:   make(map[string][]handler),
		logger: slog.Default(),
	}
}

// BridgeToFrontend sets the bridge used to forward Publish calls to the
// frontend. Passing nil disables bridging.
func (b *EventBus) BridgeToFrontend(br Bridge) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.bridge = br
}

// Subscribe registers an in-process handler for name and returns a cancel
// function. The cancel function is idempotent.
func (b *EventBus) Subscribe(name string, fn func(data any)) (cancel func()) {
	b.mu.Lock()
	id := nextHandlerID.Add(1)
	h := handler{id: id, fn: fn}
	b.subs[name] = append(b.subs[name], h)
	b.mu.Unlock()

	var once sync.Once
	return func() {
		once.Do(func() {
			b.mu.Lock()
			defer b.mu.Unlock()
			subs := b.subs[name]
			for i, s := range subs {
				if s.id == id {
					b.subs[name] = append(subs[:i], subs[i+1:]...)
					break
				}
			}
		})
	}
}

// Publish notifies all in-process subscribers, then forwards to the frontend
// bridge if one is set. A panicking handler is recovered so it cannot break
// other subscribers or the caller.
func (b *EventBus) Publish(name string, data any) {
	b.mu.RLock()
	// Copy handlers under the read lock so iteration is safe.
	handlers := make([]handler, len(b.subs[name]))
	copy(handlers, b.subs[name])
	bridge := b.bridge
	logger := b.logger
	b.mu.RUnlock()

	for _, h := range handlers {
		func() {
			defer func() {
				if r := recover(); r != nil {
					logger.Error("eventbus handler panicked",
						slog.String("topic", name),
						slog.Any("panic", r))
				}
			}()
			h.fn(data)
		}()
	}

	if bridge != nil {
		bridge.Emit(name, data)
	}
}
