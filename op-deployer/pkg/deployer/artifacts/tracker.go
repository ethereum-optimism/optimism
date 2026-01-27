package artifacts

import (
	"os"
	"sync"
)

type TempDirTracker struct {
	mu       sync.Mutex
	tempDirs []string
}

// NewTempDirTracker creates a new tracker for temporary directories.
func NewTempDirTracker() *TempDirTracker {
	return &TempDirTracker{
		tempDirs: make([]string, 0),
	}
}

// Add registers a temporary directory for cleanup.
func (t *TempDirTracker) Add(dirPath string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.tempDirs = append(t.tempDirs, dirPath)
}

// Cleanup removes all tracked temporary directories.
func (t *TempDirTracker) Cleanup() error {
	t.mu.Lock()
	dirs := make([]string, len(t.tempDirs))
	copy(dirs, t.tempDirs)
	t.mu.Unlock()

	var lastErr error
	for _, dir := range dirs {
		if err := os.RemoveAll(dir); err != nil && !os.IsNotExist(err) {
			lastErr = err
		}
	}

	t.mu.Lock()
	t.tempDirs = nil
	t.mu.Unlock()

	return lastErr
}

// GetTempDirs returns a copy of the tracked directories (for testing/debugging).
func (t *TempDirTracker) GetTempDirs() []string {
	t.mu.Lock()
	defer t.mu.Unlock()
	dirs := make([]string, len(t.tempDirs))
	copy(dirs, t.tempDirs)
	return dirs
}
