package forge

import (
	"os"
	"syscall"
)

// peakMemory returns the peak resident memory of an exited child in bytes, or 0 if unknown. Linux
// counts it in KiB. Go starts a child in the memory of this process (CLONE_VM), and Linux's exec
// carries the peak of the memory it replaces into the child's peak, so a child's peak at or below
// this process's own may be this process's.
func peakMemory(state *os.ProcessState) uint64 {
	usage, ok := state.SysUsage().(*syscall.Rusage)
	if !ok || usage.Maxrss <= 0 {
		return 0
	}
	peak := uint64(usage.Maxrss) << 10
	status, _ := os.ReadFile("/proc/self/status")
	if own := parseKiBField(status, "VmHWM:"); own == 0 || peak <= own {
		return 0
	}
	return peak
}
