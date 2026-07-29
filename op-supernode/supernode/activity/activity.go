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

	// IsActiveAt reports whether this verification activity is responsible for
	// verifying L2 content at the given timestamp. Activities that are scheduled
	// (e.g. an interop activation in the future) return false for timestamps
	// strictly before their activation point, signaling that callers may treat
	// data at or before that timestamp as safe without consulting this activity.
	IsActiveAt(ts uint64) bool

	// LatestVerifiedL2Block returns the latest verified L2 block.
	// (block, ts, nil) — verified tip at ts.
	// (empty, ts, nil) — no verified entry; ts is the pre-activation cap the
	//   caller should resolve to a canonical L2 block. ts==0 means no cap.
	// (empty, 0, err) — verifier is transiently unavailable.
	LatestVerifiedL2Block(chainID eth.ChainID) (eth.BlockID, uint64, error)

	// VerifiedBlockAtL1 returns the latest verified L2 block whose data was
	// derived from or before the supplied L1 block. Return shape matches
	// LatestVerifiedL2Block: an empty BlockID with non-zero ts is a cap, not a
	// failure.
	VerifiedBlockAtL1(chainID eth.ChainID, l1Block eth.L1BlockRef) (eth.BlockID, uint64, error)
}
