package artifacts

import (
	"sync"
)

var (
	globalTracker     *TempDirTracker
	globalTrackerOnce sync.Once
)

// getGlobalTracker returns the global tracker, initializing it on first use
func getGlobalTracker() *TempDirTracker {
	globalTrackerOnce.Do(func() {
		globalTracker = NewTempDirTracker()
	})
	return globalTracker
}

// RegisterForCleanup registers a directory for cleanup
func RegisterForCleanup(dirPath string) {
	tracker := getGlobalTracker()
	tracker.Add(dirPath)
}

// CleanupAll performs cleanup of all registered temporary directories
func CleanupAll() error {
	tracker := getGlobalTracker()
	return tracker.Cleanup()
}

// GetCleanupDirs returns a copy of currently registered cleanup directories
func GetCleanupDirs() []string {
	tracker := getGlobalTracker()
	return tracker.GetTempDirs()
}
