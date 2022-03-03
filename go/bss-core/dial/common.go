package dial

import "time"

const (
	// DefaultTimeout is default duration the service will wait on startup to
	// make a connection to either the L1 or L2 backends.
	DefaultTimeout = 5 * time.Second
)
