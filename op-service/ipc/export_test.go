package ipc

import "time"

// SetReadyTimeout overrides how long Spawn waits for readiness and returns a func restoring it.
func SetReadyTimeout(d time.Duration) (restore func()) {
	prev := readyTimeout
	readyTimeout = d
	return func() { readyTimeout = prev }
}
