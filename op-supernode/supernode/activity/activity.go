package activity

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// Activity is an open interface to collect pluggable behaviors which satisfy sub-activitiy interfaces.
type Activity interface {
	Name() string
	// Reset is called when a chain container resets due to an invalidated block.
	// Activities should clean up any cached state for that chain at or after the timestamp.
	// The invalidatedBlock is the block that was is the target of the reset
	// This is a no-op for activities that don't maintain chain-specific state.
	Reset(chainID eth.ChainID, timestamp uint64, invalidatedBlock eth.BlockRef)
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

	// Reset resets the activity's state.
	Reset(chainID eth.ChainID, timestamp uint64, invalidatedBlock eth.BlockRef)

	// CurrentL1 returns the current L1 block ID.
	CurrentL1() eth.BlockID

	// VerifiedAtTimestamp returns true if the activity has verified the data at the given timestamp.
	VerifiedAtTimestamp(ts uint64) (bool, error)

	// LatestVerifiedL2Block returns the latest L2 block which has been verified,
	// along with the timestamp at which it was verified.
	LatestVerifiedL2Block(chainID eth.ChainID) (eth.BlockID, uint64)

	// VerifiedBlockAtL1 returns the verified L2 block and timestamp
	// which guarantees that the verified data at that timestamp
	// originates from or before the supplied L1 block.
	VerifiedBlockAtL1(chainID eth.ChainID, l1Block eth.L1BlockRef) (eth.BlockID, uint64)
}
