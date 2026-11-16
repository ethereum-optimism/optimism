//go:build linux || darwin

package forge

import (
	"os"
	"runtime"
	"syscall"
)

// peakMemory returns the peak resident memory of an exited process in bytes, or 0 if unknown.
func peakMemory(state *os.ProcessState) uint64 {
	usage, ok := state.SysUsage().(*syscall.Rusage)
	if !ok || usage.Maxrss <= 0 {
		return 0
	}
	// Linux counts it in KiB, macOS in bytes.
	if runtime.GOOS == "darwin" {
		return uint64(usage.Maxrss)
	}
	return uint64(usage.Maxrss) << 10
}
