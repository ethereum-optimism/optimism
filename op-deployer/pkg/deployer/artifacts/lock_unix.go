//go:build unix

package artifacts

import (
	"errors"
	"fmt"
	"os"
	"syscall"
	"time"
)

// fileLock is a lock on a file. The system frees it when its process exits.
type fileLock struct {
	f *os.File
}

// openFileLock opens the lock of the file at path, which it creates.
func openFileLock(path string) (*fileLock, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, fmt.Errorf("failed to open the lock %s: %w", path, err)
	}
	return &fileLock{f: f}, nil
}

// lock waits up to timeout for the lock, exclusive or shared.
func (l *fileLock) lock(exclusive bool, timeout time.Duration) error {
	how := syscall.LOCK_SH
	if exclusive {
		how = syscall.LOCK_EX
	}
	deadline := time.Now().Add(timeout)
	for {
		err := syscall.Flock(int(l.f.Fd()), how|syscall.LOCK_NB)
		if err == nil {
			return nil
		}
		if !errors.Is(err, syscall.EWOULDBLOCK) {
			return fmt.Errorf("failed to lock %s: %w", l.f.Name(), err)
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("another process held the lock %s for more than %s", l.f.Name(), timeout)
		}
		time.Sleep(lockPollInterval)
	}
}

// tryLock takes the lock exclusive without waiting, and returns false when another holder has it.
func (l *fileLock) tryLock() (bool, error) {
	err := syscall.Flock(int(l.f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if errors.Is(err, syscall.EWOULDBLOCK) {
		return false, nil
	}
	return err == nil, err
}

// release frees the lock.
func (l *fileLock) release() {
	_ = l.f.Close()
}
