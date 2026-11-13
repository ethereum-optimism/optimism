//go:build !linux

package forge

// memoryLimit returns 0: it knows the memory of the process only on Linux.
func memoryLimit() uint64 {
	return 0
}
