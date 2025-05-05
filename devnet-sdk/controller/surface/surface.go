package surface

import "context"

type ControlSurface interface {
}

type ServiceLifecycleSurface interface {
	StartService(context.Context, string) error
	StopService(context.Context, string) error
}
