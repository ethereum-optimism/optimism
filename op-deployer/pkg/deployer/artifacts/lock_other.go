//go:build !unix

package artifacts

import (
	"fmt"
	"os"
	"time"
)

// fileLock does not lock where there are no file locks. Processes that pack the same bundle at the
// same time each pack it, the atomic rename keeps one of them, and no bundle is evicted.
type fileLock struct{}

func openFileLock(string) (*fileLock, error) {
	return &fileLock{}, nil
}

func (*fileLock) lock(bool, time.Duration) error {
	return nil
}

func (*fileLock) tryLock() (bool, error) {
	return false, nil
}

func (*fileLock) release() {}

// checkPrivateDir checks that dir is a directory and not a link: there are no owners to check.
func checkPrivateDir(dir string) error {
	info, err := os.Lstat(dir)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", dir)
	}
	return nil
}
