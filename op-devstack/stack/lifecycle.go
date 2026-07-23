package stack

import "context"

type Lifecycle interface {
	Start()
	Stop()
}

type ControlledLifecycle interface {
	StartControlled(ctx context.Context) error
	StopControlled(ctx context.Context) error
	Running() bool
}
