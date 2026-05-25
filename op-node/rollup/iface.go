package rollup

import (
	"context"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// CanonicalChain is the minimal probe used by SuperAuthority.CanonicalDeniedHeight
// to ask "what is the canonical block at this L2 height?". Implementations are
// expected to return the EL's view of the canonical chain (e.g. via
// eth_getBlockByNumber).
type CanonicalChain interface {
	L2BlockRefByNumber(ctx context.Context, number uint64) (eth.L2BlockRef, error)
}

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
	// CanonicalDeniedHeight returns the lowest L2 block height with a deny-list
	// entry whose payload hash matches the canonical hash at that height,
	// walking the deny list from highest to lowest. Iteration stops at the
	// first height where no denied entry is canonical (assuming monotone
	// canonicality: once a denied block has been reorged out, lower entries
	// that are parents of the still-canonical chain are also reorged out).
	//
	// Returns (height, true, nil) when a canonical denied entry is found.
	// Returns (0, false, nil) when the deny list is empty or no entries are canonical.
	//
	// Used at reset time to cap the safe head to (height - 1) so that derivation
	// re-derives the denied block, hits the IsDenied check, and emits
	// deposits-only attributes that replace the block via consolidation.
	CanonicalDeniedHeight(ctx context.Context, canonical CanonicalChain) (uint64, bool, error)
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
