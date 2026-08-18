package dsl

import "time"

const (
	// DefaultPollInterval is the pause between attempts of retrying checks.
	DefaultPollInterval = 2 * time.Second
	// DefaultTimeout bounds individual RPC reads.
	DefaultTimeout = 30 * time.Second
)
