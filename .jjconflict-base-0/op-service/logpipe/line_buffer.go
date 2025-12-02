package logpipe

import (
	"io"
	"sync"
)

// LogCallback is the function signature for processing a complete log line.
type LogCallback func(line []byte)

// LineBuffer is an io.WriteCloser that buffers data across writes and emits complete lines
// to the provided callback, preserving log entries that may span multiple writes.
// Any trailing partial line is flushed when Close is called.
type LineBuffer struct {
	mu       sync.Mutex
	buf      []byte
	callback LogCallback
}

// NewLineBuffer creates a new LineBuffer that calls the given callback for each complete line.
func NewLineBuffer(callback LogCallback) *LineBuffer {
	return &LineBuffer{callback: callback}
}

// Write appends data, emitting full lines to the callback.
// Empty lines are ignored.
func (lp *LineBuffer) Write(p []byte) (int, error) {
	lp.mu.Lock()
	defer lp.mu.Unlock()

	lp.buf = append(lp.buf, p...)

	start := 0
	for i := 0; i < len(lp.buf); i++ {
		if lp.buf[i] == '\n' {
			line := lp.buf[start:i]
			if len(line) > 0 {
				lp.callback(line)
			}
			start = i + 1
		}
	}
	// Keep any partial trailing line
	if start > 0 {
		lp.buf = append([]byte(nil), lp.buf[start:]...)
	}
	return len(p), nil
}

// Close flushes any buffered partial line.
func (lp *LineBuffer) Close() error {
	lp.mu.Lock()
	defer lp.mu.Unlock()

	if len(lp.buf) > 0 {
		lp.callback(lp.buf)
		lp.buf = nil
	}
	return nil
}

var _ io.WriteCloser = (*LineBuffer)(nil)
