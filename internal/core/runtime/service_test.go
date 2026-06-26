package runtime

import (
	"context"
	"testing"
	"time"
)

func TestServiceName(t *testing.T) {
	svc := NewService()
	if got := svc.ServiceName(); got != "runtime" {
		t.Fatalf("ServiceName() = %q, want runtime", got)
	}
}

func TestServiceSnapshotReturnsProcessMetrics(t *testing.T) {
	svc := NewService()

	first, err := svc.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("Snapshot first call: %v", err)
	}
	if first.MemoryBytes <= 0 {
		t.Fatalf("MemoryBytes = %d, want > 0", first.MemoryBytes)
	}
	if first.CPUPercent < 0 {
		t.Fatalf("CPUPercent = %f, want >= 0", first.CPUPercent)
	}

	time.Sleep(10 * time.Millisecond)

	second, err := svc.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("Snapshot second call: %v", err)
	}
	if second.MemoryBytes <= 0 {
		t.Fatalf("MemoryBytes = %d, want > 0", second.MemoryBytes)
	}
	if second.CPUPercent < 0 {
		t.Fatalf("CPUPercent = %f, want >= 0", second.CPUPercent)
	}
}
