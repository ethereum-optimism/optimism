//go:build !windows

package preimage

import "syscall"

func setNonblock(fd int) {
	// Unix only
	_ = syscall.SetNonblock(fd, true)
}
