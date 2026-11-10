//go:build !linux

package forge

// memoryLimit returns 0: only Linux containers limit the memory of op-deployer.
func memoryLimit() uint64 {
	return 0
}
