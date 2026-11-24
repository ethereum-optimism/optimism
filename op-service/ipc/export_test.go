package ipc

import "time"

// StderrTailLines is how many of the child's last stderr lines a Spawn error quotes.
const StderrTailLines = stderrTailLines

// SetReadyTimeout overrides how long Spawn waits for readiness and returns a func restoring it.
func SetReadyTimeout(d time.Duration) (restore func()) {
	prev := readyTimeout
	readyTimeout = d
	return func() { readyTimeout = prev }
}
