package sender

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestTimedSenderStartStop(t *testing.T) {
	t.Parallel()
	var count atomic.Int64
	sendFunc := func(data []byte) (int, error) {
		count.Add(1)
		return len(data), nil
	}

	ts := NewTimedSender("test", 50*time.Millisecond, []byte("hello"), sendFunc)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	go func() { _ = ts.Start(ctx) }()
	time.Sleep(250 * time.Millisecond)

	if ts.Count() < 3 {
		t.Fatalf("Count = %d, want >= 3", ts.Count())
	}
}

func TestTimedSenderStop(t *testing.T) {
	t.Parallel()
	sendFunc := func(data []byte) (int, error) {
		return len(data), nil
	}

	ts := NewTimedSender("test", 50*time.Millisecond, []byte("hello"), sendFunc)

	ctx, cancel := context.WithCancel(context.Background())
	go func() { _ = ts.Start(ctx) }()

	time.Sleep(100 * time.Millisecond)
	cancel()
	time.Sleep(50 * time.Millisecond)

	if ts.IsRunning() {
		t.Fatalf("should not be running after cancel")
	}
}

func TestQuickButtonStore(t *testing.T) {
	t.Parallel()
	store := NewQuickButtonStore()

	// Empty list
	if len(store.List()) != 0 {
		t.Fatalf("initial list should be empty")
	}

	// Add
	store.Add(QuickButton{ID: "1", Label: "Test", Content: "hello", Mode: "ascii"})
	if len(store.List()) != 1 {
		t.Fatalf("list should have 1 item")
	}

	// Update
	store.Update(QuickButton{ID: "1", Label: "Updated", Content: "world", Mode: "hex"})
	btns := store.List()
	if btns[0].Label != "Updated" {
		t.Fatalf("label should be Updated, got %s", btns[0].Label)
	}

	// Remove
	store.Remove("1")
	if len(store.List()) != 0 {
		t.Fatalf("list should be empty after remove")
	}
}
