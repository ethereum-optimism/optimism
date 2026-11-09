//go:build !unix

package artifacts

// fileLock does not lock where there are no file locks. Processes that pack the same bundle at the
// same time each pack it, the atomic rename keeps one of them, and no bundle is evicted.
type fileLock struct{}

func openFileLock(string) (*fileLock, error) {
	return &fileLock{}, nil
}

func (*fileLock) lock(bool) error {
	return nil
}

func (*fileLock) tryLock() (bool, error) {
	return false, nil
}

func (*fileLock) release() {}
