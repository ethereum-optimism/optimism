//go:build unix

package forge

import (
	"os"
	"syscall"
)

// tryLock locks f without waiting. Closing f unlocks it.
func tryLock(f *os.File) bool {
	return syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB) == nil
}
