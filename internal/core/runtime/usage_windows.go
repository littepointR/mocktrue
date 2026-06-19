//go:build windows

package runtime

import (
	goruntime "runtime"
)

func readProcessUsage() (processUsage, error) {
	var mem goruntime.MemStats
	goruntime.ReadMemStats(&mem)
	return processUsage{
		cpuTime:     0,
		memoryBytes: mem.Sys,
	}, nil
}
