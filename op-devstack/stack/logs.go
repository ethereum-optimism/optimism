package stack

import "github.com/ethereum-optimism/optimism/op-service/logpipe"

// LogRecorder is implemented by components whose process log a test can watch.
type LogRecorder interface {
	// RecordLogLines returns a record of every log entry the component writes from now on, across
	// restarts, whose message contains substring.
	RecordLogLines(substring string) *logpipe.LogRecord
}
