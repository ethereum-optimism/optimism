package artifacts

import (
	"os"
	"os/signal"
	"sync"
	"syscall"
)

var (
	globalTracker     *TempDirTracker
	globalTrackerOnce sync.Once
	globalCleanupDone chan struct{}
)

// getGlobalTracker returns the global tracker, initializing it on first use
func getGlobalTracker() *TempDirTracker {
	globalTrackerOnce.Do(func() {
		globalTracker = NewTempDirTracker()
		globalCleanupDone = make(chan struct{})

		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

		go func() {
			defer close(globalCleanupDone)
			<-sigChan
			if err := globalTracker.Cleanup(); err != nil {
				// We can't log here as the process is shutting down
			}
		}()
	})
	return globalTracker
}

// RegisterForCleanup registers a directory for automatic cleanup on process exit
func RegisterForCleanup(dirPath string) {
	tracker := getGlobalTracker()
	tracker.Add(dirPath)
}

// WaitForCleanup waits for cleanup to complete (useful for testing)
func WaitForCleanup() {
	if globalCleanupDone != nil {
		<-globalCleanupDone
	}
}
