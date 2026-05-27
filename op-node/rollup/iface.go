package rollup

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// VerifierHeadSource classifies the origin of a head reported by the SuperAuthority.
type VerifierHeadSource uint8

const (
	// VerifierHeadPreActivation signals that every registered verifier is still
	// inactive at the current local-safe timestamp. The caller should use its
	// local-safe / local-finalized head — pre-activation content is verified by
	// consensus alone.
	VerifierHeadPreActivation VerifierHeadSource = iota
	// VerifierHeadAnchor signals that at least one active verifier has no
	// verified-DB entry that includes this chain yet (the chain hasn't joined
	// the verifier's depset, or the verifier just started). The Block field
	// carries the per-(chain, verifier) activation-anchor block — the L2 block
	// at timestamp `verifier.ActivationTimestamp() - 1`.
	VerifierHeadAnchor
	// VerifierHeadVerified signals the head is the verifier's verified tip for
	// this chain. The Block field carries that tip.
	VerifierHeadVerified
)

// String implements fmt.Stringer. The exhaustive switch is the structural gate
// that forces new VerifierHeadSource variants to update every consumer.
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

// VerifierHeadStatus classifies whether a SuperAuthority call produced a usable
// answer or signalled a transient read failure.
type VerifierHeadStatus uint8

const (
	// VerifierHeadOk signals the returned VerifierHead is valid and should be
	// honored. The caller still bounds the result by its own local-safe head
	// and validates EL canonicality.
	VerifierHeadOk VerifierHeadStatus = iota
	// VerifierHeadHoldPrevious signals a transient verifier read failure. The
	// caller must not advance the head and must not fall back to local-safe;
	// floor at FinalizedHead instead.
	VerifierHeadHoldPrevious
)

// String implements fmt.Stringer.
func (s VerifierHeadStatus) String() string {
	switch s {
	case VerifierHeadOk:
		return "ok"
	case VerifierHeadHoldPrevious:
		return "hold-previous"
	default:
		return fmt.Sprintf("unknown(%d)", uint8(s))
	}
}

// VerifierHead is the tri-state head reported by the SuperAuthority.
// Block is zero when Source == VerifierHeadPreActivation.
type VerifierHead struct {
	Block  eth.BlockID
	Source VerifierHeadSource
}

// SuperAuthority provides cross-verified safe / finalized head reporting and
// payload deny-list checks from a supernode. When running inside a supernode,
// this is what the engine controller consults to decide what to publish as
// SafeL2Head / FinalizedHead and whether to apply a payload.
type SuperAuthority interface {
	// FullyVerifiedL2Head returns the cross-verified safe L2 head, as a tri-state.
	//
	// On VerifierHeadOk:
	//   - Source == VerifierHeadPreActivation: caller uses local-safe.
	//   - Source == VerifierHeadAnchor: Block is the activation-anchor block,
	//     contributed by at least one active verifier with no entry for this chain.
	//   - Source == VerifierHeadVerified: Block is the oldest verified tip across
	//     active verifiers.
	// On VerifierHeadHoldPrevious: a verifier read failed; the caller must hold
	// the previous value and floor at FinalizedHead — never advance, never fall
	// back to local-safe.
	FullyVerifiedL2Head() (VerifierHead, VerifierHeadStatus)

	// FinalizedL2Head returns the cross-verified finalized L2 head, as a tri-state
	// with the same semantics as FullyVerifiedL2Head. The finalized head is
	// caller-cacheable because finalized blocks cannot reorg.
	FinalizedL2Head() (VerifierHead, VerifierHeadStatus)

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
