package rollup

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// SuperAuthority provides payload validation functionality from a supernode.
// When running inside a supernode, this allows the engine controller to check
// if payloads are denied before applying them, enabling coordinated block invalidation.
type SuperAuthority interface {
	// FullyVerifiedL2Head returns the fully verified L2 head block reference.
	// The second return value indicates whether the caller should fall back to local-safe.
	// If useLocalSafe is true, the BlockID return value should be ignored and local-safe used instead.
	// If useLocalSafe is false, the BlockID is the cross-verified safe head.
	FullyVerifiedL2Head() (head eth.BlockID, useLocalSafe bool)
	// FinalizedL2Head returns the finalized L2 head block reference.
	// The second return value indicates whether the caller should fall back to local-finalized.
	// If useLocalFinalized is true, the BlockID return value should be ignored and local-finalized used instead.
	// If useLocalFinalized is false, the BlockID is the cross-verified finalized head.
	FinalizedL2Head() (head eth.BlockID, useLocalFinalized bool)
	// IsDenied checks if a payload hash is denied at the given block number.
	// Returns true if the payload should not be applied.
	// The error indicates if the check could not be performed (should be logged but not fatal).
	IsDenied(blockNumber uint64, payloadHash common.Hash) (bool, error)
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
