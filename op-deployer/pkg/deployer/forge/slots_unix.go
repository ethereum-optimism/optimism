//go:build unix

package forge

import (
	"errors"
	"os"
	"syscall"
)

// tryLock locks f without waiting, and returns false when another holder has it. Any other error is
// returned: a cache directory on NFS without a lock daemon would otherwise look busy forever.
// Closing f unlocks it.
func tryLock(f *os.File) (bool, error) {
	err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if errors.Is(err, syscall.EWOULDBLOCK) {
		return false, nil
	}
	return err == nil, err
}
