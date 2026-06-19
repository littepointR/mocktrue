//go:build darwin || linux

package runtime

import (
	"syscall"
	"time"
)

func readProcessUsage() (processUsage, error) {
	var rusage syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &rusage); err != nil {
		return processUsage{}, err
	}

	memoryBytes := uint64(rusage.Maxrss)
	if memoryBytes > 0 && isLinuxMaxrssKilobytes() {
		memoryBytes *= 1024
	}

	return processUsage{
		cpuTime:     timevalDuration(rusage.Utime) + timevalDuration(rusage.Stime),
		memoryBytes: memoryBytes,
	}, nil
}

func timevalDuration(tv syscall.Timeval) time.Duration {
	return time.Duration(tv.Sec)*time.Second + time.Duration(tv.Usec)*time.Microsecond
}
