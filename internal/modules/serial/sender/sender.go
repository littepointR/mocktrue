package sender

import (
	"context"
	"sync"
	"time"

	"github.com/suyue/mocktrue/internal/core/errors"
)

// SendFunc is the function signature for sending data to a port.
type SendFunc func(data []byte) (int, error)

// TimedSender sends data at regular intervals.
type TimedSender struct {
	mu       sync.Mutex
	portID   string
	interval time.Duration
	payload  []byte
	sendFunc SendFunc
	running  bool
	cancel   context.CancelFunc
	count    int64
}

// NewTimedSender creates a new timed sender.
func NewTimedSender(portID string, interval time.Duration, payload []byte, sendFunc SendFunc) *TimedSender {
	return &TimedSender{
		portID:   portID,
		interval: interval,
		payload:  payload,
		sendFunc: sendFunc,
	}
}

// Start begins the timed sending. Blocks until stopped or context cancelled.
func (ts *TimedSender) Start(ctx context.Context) error {
	ts.mu.Lock()
	if ts.running {
		ts.mu.Unlock()
		return errors.New(errors.CodeConflict, "timed sender already running")
	}
	ctx, ts.cancel = context.WithCancel(ctx)
	ts.running = true
	ts.count = 0
	ts.mu.Unlock()

	ticker := time.NewTicker(ts.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			ts.mu.Lock()
			ts.running = false
			ts.mu.Unlock()
			return nil
		case <-ticker.C:
			if _, err := ts.sendFunc(ts.payload); err != nil {
				// Log error but continue sending
				continue
			}
			ts.mu.Lock()
			ts.count++
			ts.mu.Unlock()
		}
	}
}

// Stop stops the timed sender.
func (ts *TimedSender) Stop() {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	if ts.cancel != nil {
		ts.cancel()
	}
}

// Count returns the number of sends performed.
func (ts *TimedSender) Count() int64 {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	return ts.count
}

// IsRunning returns whether the sender is active.
func (ts *TimedSender) IsRunning() bool {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	return ts.running
}

// QuickButton represents a preset send button.
type QuickButton struct {
	ID      string `json:"id"`
	Label   string `json:"label"`
	Content string `json:"content"`
	Mode    string `json:"mode"` // "ascii" or "hex"
}

// QuickButtonStore manages quick buttons.
type QuickButtonStore struct {
	mu      sync.RWMutex
	buttons []QuickButton
}

// NewQuickButtonStore creates a new store.
func NewQuickButtonStore() *QuickButtonStore {
	return &QuickButtonStore{
		buttons: make([]QuickButton, 0),
	}
}

// List returns all quick buttons.
func (s *QuickButtonStore) List() []QuickButton {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]QuickButton, len(s.buttons))
	copy(result, s.buttons)
	return result
}

// Add adds a quick button.
func (s *QuickButtonStore) Add(btn QuickButton) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.buttons = append(s.buttons, btn)
}

// Remove removes a quick button by ID.
func (s *QuickButtonStore) Remove(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, btn := range s.buttons {
		if btn.ID == id {
			s.buttons = append(s.buttons[:i], s.buttons[i+1:]...)
			return true
		}
	}
	return false
}

// Update updates a quick button.
func (s *QuickButtonStore) Update(btn QuickButton) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, b := range s.buttons {
		if b.ID == btn.ID {
			s.buttons[i] = btn
			return true
		}
	}
	return false
}
