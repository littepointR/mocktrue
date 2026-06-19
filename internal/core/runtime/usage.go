package runtime

import (
	goruntime "runtime"
	"time"
)

type processUsage struct {
	cpuTime     time.Duration
	memoryBytes uint64
}

func cpuCount() int {
	count := goruntime.NumCPU()
	if count < 1 {
		return 1
	}
	return count
}
