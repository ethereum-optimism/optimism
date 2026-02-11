package rollup

import "github.com/ethereum-optimism/optimism/op-service/eth"

// SuperAuthority is an interface for supernode-level authority operations.
// It is passed to op-node instances during initialization to provide
// supernode-specific functionality and coordination.
type SuperAuthority interface {
	// FullyVerifiedL2Head returns the fully verified L2 head block reference.
	// It returns an empty L2BlockRef if no fully verified head can be determined.
	// Note that the returned block ref may not be local safe and the caller should
	// verify the block's local safety before using it.
	FullyVerifiedL2Head() eth.BlockID
}

// SafeHeadListener is called when the safe head is updated.
// The safe head may advance by more than one block in a single update
// The l1Block specified is the first L1 block that includes sufficient information to derive the new safe head
type SafeHeadListener interface {

	// Enabled reports if this safe head listener is actively using the posted data. This allows the engine queue to
	// optionally skip making calls that may be expensive to prepare.
	// Callbacks may still be made if Enabled returns false but are not guaranteed.
	Enabled() bool

	// SafeHeadUpdated indicates that the safe head has been updated in response to processing batch data
	// The l1Block specified is the first L1 block containing all required batch data to derive newSafeHead
	SafeHeadUpdated(newSafeHead eth.L2BlockRef, l1Block eth.BlockID) error

	// SafeHeadReset indicates that the derivation pipeline reset back to the specified safe head
	// The L1 block that made the new safe head safe is unknown.
	SafeHeadReset(resetSafeHead eth.L2BlockRef) error
}
