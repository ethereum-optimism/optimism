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

// UngracefulLifecycle is implemented by components that can be stopped the way a crash or a
// power loss would stop them: without a shutdown, so whatever they held only in memory is lost.
type UngracefulLifecycle interface {
	StopUngracefully()
}
