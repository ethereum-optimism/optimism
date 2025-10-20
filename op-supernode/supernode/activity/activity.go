package activity

import "context"

type Activity interface {
}

// Activity is a background task that can be started and stopped independently
// from chain containers. Activities may operate across multiple chains.
type RunnableActivity interface {
	Activity
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

type RPCActivity interface {
	Activity
	RPCNamespace() string
	RPCService() interface{}
}
