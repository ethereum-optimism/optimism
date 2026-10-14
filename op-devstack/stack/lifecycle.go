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

// UngracefulLifecycle is implemented by components that can be stopped the way a process crash
// would stop them: no shutdown runs, so state held only in process memory is lost. Writes already
// handed to the OS survive, so this does not model a power loss.
type UngracefulLifecycle interface {
	StopUngracefully()
}
