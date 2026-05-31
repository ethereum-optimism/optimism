package rollup

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// VerifierHeadSource classifies the origin of a head reported by the SuperAuthority.
type VerifierHeadSource uint8

const (
	// VerifierHeadPreActivation: the registered verifier is inactive at the
	// current local-safe timestamp. Caller uses local-safe / local-finalized.
	VerifierHeadPreActivation VerifierHeadSource = iota
	// VerifierHeadAnchor: the active verifier has no verified-DB entry for this
	// chain yet. Block is zero; Timestamp is the pre-activation cap
	// (`activationTimestamp - 1`); caller resolves the canonical L2 block at
	// that timestamp.
	VerifierHeadAnchor
	// VerifierHeadVerified: Block is the verified tip from the verifier.
	VerifierHeadVerified
)

// Exhaustive — adding a variant requires updating every consumer's switch.
func (s VerifierHeadSource) String() string {
	switch s {
	case VerifierHeadPreActivation:
		return "pre-activation"
	case VerifierHeadAnchor:
		return "anchor"
	case VerifierHeadVerified:
		return "verified"
	default:
		return fmt.Sprintf("unknown(%d)", uint8(s))
	}
}

// VerifierHead is the result of a SuperAuthority head query.
//   - Source == PreActivation: Block and Timestamp are zero; caller uses local.
//   - Source == Verified:      Block is the verified tip; Timestamp is its L2 time.
//   - Source == Anchor:        Block is zero; Timestamp is the pre-activation cap
//     (`activationTimestamp - 1`); caller resolves the canonical L2 block at that
//     timestamp itself.
type VerifierHead struct {
	Block     eth.BlockID
	Timestamp uint64
	Source    VerifierHeadSource
}

// SuperAuthority is the cross-chain attestation surface a supernode exposes to
// op-node: cross-verified safe / finalized head reporting and payload deny-list
// checks. Returned heads are consumed by the engine controller to choose what
// to publish as SafeL2Head / FinalizedHead and whether to apply a payload.
type SuperAuthority interface {
	// FullyVerifiedL2Head returns the cross-verified safe L2 head.
	// `ok=false` signals a transient read failure — caller must hold the
	// previous value (floored at FinalizedHead), never fall back to local-safe.
	FullyVerifiedL2Head(ctx context.Context) (head VerifierHead, ok bool)

	// FinalizedL2Head is the finalized analogue of FullyVerifiedL2Head.
	// Finalized blocks cannot reorg, so the caller may cache the result.
	FinalizedL2Head(ctx context.Context) (head VerifierHead, ok bool)

	// IsDenied reports whether a payload hash is denied at the given block
	// number. Errors are logged but not fatal.
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
