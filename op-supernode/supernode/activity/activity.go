package activity

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// Activity is an open interface to collect pluggable behaviors which satisfy sub-activitiy interfaces.
type Activity interface {
	// Reset is called when a chain container resets to a given timestamp.
	// Activities should clean up any cached state for that chain at or after the timestamp.
	// This is a no-op for activities that don't maintain chain-specific state.
	Reset(chainID eth.ChainID, timestamp uint64)
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

// VerificationActivity is an Activity that can be used to verify the correctness of the Supernode's Chains
type VerificationActivity interface {
	Activity
	Name() string
	CurrentL1() eth.BlockID
	VerifiedAtTimestamp(ts uint64) (bool, error)
}
