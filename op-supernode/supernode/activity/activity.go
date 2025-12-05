package activity

import "context"

// Activity is an open interface to collect pluggable behaviors which satisfy sub-activitiy interfaces.
type Activity interface {
}

// RunnableActivity is an Activity that can be started and stopped independently.
// The Supernode calls start through a goroutine and calls stop when the application is shutting down.
type RunnableActivity interface {
	Activity
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

// RPCActivity is an Activity that can be exposed to the RPC server.
// Any methods exposed through the RPC server are mounted under the activity namespace.
type RPCActivity interface {
	Activity
	RPCNamespace() string
	RPCService() interface{}
}
