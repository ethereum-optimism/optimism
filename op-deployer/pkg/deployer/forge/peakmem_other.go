//go:build !linux && !darwin

package forge

import "os"

// peakMemory returns 0: it knows the peak memory of a process only on Linux and macOS.
func peakMemory(*os.ProcessState) uint64 {
	return 0
}
