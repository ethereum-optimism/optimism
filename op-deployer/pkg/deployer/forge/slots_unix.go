//go:build unix

package forge

import (
	"os"
	"syscall"
)

// tryLock takes the lock on f without waiting, and reports whether it got it. Closing f frees it.
func tryLock(f *os.File) bool {
	return syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB) == nil
}
