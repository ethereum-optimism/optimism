package forge

import (
	"os"
	"syscall"
)

// peakMemory returns the peak resident memory of an exited child in bytes, or 0 if unknown. macOS
// counts it in bytes.
func peakMemory(state *os.ProcessState) uint64 {
	usage, ok := state.SysUsage().(*syscall.Rusage)
	if !ok || usage.Maxrss <= 0 {
		return 0
	}
	return uint64(usage.Maxrss)
}
