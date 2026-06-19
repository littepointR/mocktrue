package runtime

import (
	"context"
	"sync"
	"time"
)

// Metrics describes current resource usage for the running MockTrue process.
type Metrics struct {
	CPUPercent  float64
	MemoryBytes uint64
}

// Service exposes application runtime metrics to the frontend.
type Service struct {
	mu       sync.Mutex
	lastCPU  time.Duration
	lastWall time.Time
}

// NewService constructs a runtime metrics service.
func NewService() *Service {
	return &Service{}
}

func (s *Service) ServiceName() string { return "runtime" }

// Snapshot returns current process CPU and memory usage.
func (s *Service) Snapshot(ctx context.Context) (*Metrics, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	usage, err := readProcessUsage()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	cpuPercent := 0.0

	s.mu.Lock()
	if !s.lastWall.IsZero() {
		cpuDelta := usage.cpuTime - s.lastCPU
		wallDelta := now.Sub(s.lastWall)
		if cpuDelta > 0 && wallDelta > 0 {
			cpuPercent = float64(cpuDelta) / float64(wallDelta) / float64(cpuCount()) * 100
		}
	}
	s.lastCPU = usage.cpuTime
	s.lastWall = now
	s.mu.Unlock()

	if cpuPercent < 0 {
		cpuPercent = 0
	}

	return &Metrics{
		CPUPercent:  cpuPercent,
		MemoryBytes: usage.memoryBytes,
	}, nil
}
