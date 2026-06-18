package eventbus

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// fakeBridge records Emit calls for assertions without pulling in Wails.
type fakeBridge struct {
	mu     sync.Mutex
	events []emitted
}

type emitted struct {
	name string
	data any
}

func (f *fakeBridge) Emit(name string, data ...any) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	var d any
	if len(data) > 0 {
		d = data[0]
	}
	f.events = append(f.events, emitted{name: name, data: d})
	return true
}

func (f *fakeBridge) snapshot() []emitted {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]emitted, len(f.events))
	copy(out, f.events)
	return out
}

const shortWait = 100 * time.Millisecond

func TestPublishDeliversToInProcessSubscribers(t *testing.T) {
	t.Parallel()
	bus := New()
	got := make(chan string, 1)
	cancel := bus.Subscribe("topic", func(data any) { got <- data.(string) })
	defer cancel()

	bus.Publish("topic", "hello")
	select {
	case v := <-got:
		if v != "hello" {
			t.Fatalf("got %q, want hello", v)
		}
	case <-time.After(shortWait):
		t.Fatalf("subscriber not notified")
	}
}

func TestCancelStopsDelivery(t *testing.T) {
	t.Parallel()
	bus := New()
	called := make(chan any, 4)
	cancel := bus.Subscribe("topic", func(data any) { called <- data })
	cancel()

	bus.Publish("topic", "x")
	select {
	case <-called:
		t.Fatalf("cancelled subscriber must not be called")
	case <-time.After(shortWait):
	}
}

func TestPublishBridgesToFrontend(t *testing.T) {
	t.Parallel()
	bus := New()
	fb := &fakeBridge{}
	bus.BridgeToFrontend(fb)

	got := make(chan any, 1)
	bus.Subscribe("topic", func(data any) { got <- data })

	bus.Publish("topic", "bridged")
	select {
	case <-got:
	case <-time.After(shortWait):
		t.Fatalf("in-process subscriber not notified")
	}
	evs := fb.snapshot()
	if len(evs) != 1 || evs[0].name != "topic" || evs[0].data != "bridged" {
		t.Fatalf("bridge events = %+v, want one bridged topic", evs)
	}
}

func TestPublishWithoutBridgeOnlyInProcess(t *testing.T) {
	t.Parallel()
	bus := New()
	got := make(chan any, 1)
	bus.Subscribe("topic", func(data any) { got <- data })
	// No bridge set: must not panic, only in-process delivery.
	bus.Publish("topic", "solo")
	select {
	case v := <-got:
		if v != "solo" {
			t.Fatalf("got %q, want solo", v)
		}
	case <-time.After(shortWait):
		t.Fatalf("subscriber not notified")
	}
}

func TestHandlerPanicDoesNotAffectOthers(t *testing.T) {
	t.Parallel()
	bus := New()
	other := make(chan any, 1)
	bus.Subscribe("topic", func(data any) { panic("boom") })
	bus.Subscribe("topic", func(data any) { other <- data })

	// Must not panic the caller.
	bus.Publish("topic", "survive")
	select {
	case v := <-other:
		if v != "survive" {
			t.Fatalf("got %q, want survive", v)
		}
	case <-time.After(shortWait):
		t.Fatalf("second subscriber skipped after first panicked")
	}
}

func TestCancelIsIdempotent(t *testing.T) {
	t.Parallel()
	bus := New()
	called := make(chan any, 8)
	cancel := bus.Subscribe("topic", func(data any) { called <- data })
	cancel()
	cancel() // second call must not panic
	cancel() // third call must not panic

	bus.Publish("topic", "x")
	select {
	case <-called:
		t.Fatalf("cancelled subscriber must not be called")
	case <-time.After(shortWait):
	}
}

func TestConcurrentPubSubIsSafe(t *testing.T) {
	t.Parallel()
	bus := New()
	var received atomic.Int64

	// 10 subscribers, each increments a shared counter on every delivery.
	for i := 0; i < 10; i++ {
		cancel := bus.Subscribe("topic", func(data any) {
			received.Add(1)
		})
		defer cancel()
	}

	// 10 concurrent publishers, each Publish fans out to all 10 subscribers.
	var pwg sync.WaitGroup
	for i := 0; i < 10; i++ {
		pwg.Add(1)
		go func() {
			defer pwg.Done()
			bus.Publish("topic", "p")
		}()
	}
	pwg.Wait()

	// Each of 10 publishes notifies 10 subscribers => 100 deliveries expected.
	// Use a short busy-wait bound since delivery is synchronous within Publish.
	deadline := time.Now().Add(shortWait * 10)
	for time.Now().Before(deadline) {
		if received.Load() >= 100 {
			break
		}
		time.Sleep(time.Millisecond)
	}
	if got := received.Load(); got != 100 {
		t.Fatalf("received %d deliveries, want 100", got)
	}
}
