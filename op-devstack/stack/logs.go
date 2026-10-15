package stack

import "github.com/ethereum-optimism/optimism/op-service/logpipe"

// LogObservable is implemented by components whose process output is parsed into log entries.
type LogObservable interface {
	// ObserveLogs calls observer for every log entry the component emits, across restarts, until
	// the returned function is called; a delivery in flight at that moment may still complete.
	ObserveLogs(observer func(logpipe.LogEntry)) (stop func())
}
