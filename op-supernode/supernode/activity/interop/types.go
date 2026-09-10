package interop

import (
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// VerifiedResult represents the verified state at a specific timestamp.
// It contains the L1 inclusion block from which the L2 heads were included,
// and a map of each chain's L2 head at that timestamp.
type VerifiedResult struct {
	Timestamp   uint64                      `json:"timestamp"`
	L1Inclusion eth.BlockID                 `json:"l1Inclusion"`
	L2Heads     map[eth.ChainID]eth.BlockID `json:"l2Heads"`
}

// InvalidHead pairs a block identifier with the output preimage fields needed
// for optimistic root computation in the superroot API. The full OutputV0 can
// be reconstructed on demand via OutputV0() since BlockHash is already in BlockID.
type InvalidHead struct {
	eth.BlockID
	StateRoot                eth.Bytes32 `json:"stateRoot"`
	MessagePasserStorageRoot eth.Bytes32 `json:"messagePasserStorageRoot"`
}

// Result represents the result of interop validation at a specific timestamp given current data.
// it contains all the same information as VerifiedResult, but also contains a list of invalid heads.
//
// A chain's block at the timestamp has three possible verdicts, not two: verified, invalid, or
// NOT YET DECIDABLE. The third exists because the first two are not exhaustive —
//
//	absence of data is not evidence of a conflict.
//
// A verifier that reads an initiating message it does not have yet learns nothing about whether
// that message exists, only that it cannot see it from here, now. Treating that as "invalid" is
// how a live-edge verifier forks itself off a perfectly valid chain: it deny-lists the executing
// block, rewinds, re-derives the same block, and deny-lists it again, while the rest of the
// network — which had the initiating message — advances past it.
//
// A proof-carried chain makes this routine rather than exotic: its initiating messages become
// visible only when a proof batch lands on L1, minutes after the blocks they belong to exist, so
// a driven chain's block can legitimately reference a message this verifier has not received yet.
// The distinction is not specific to one kind of chain though — any verifier whose view of a peer
// lags for any reason meets the same seam — which is why it lives here in the shared judge and not
// behind a chain-kind branch.
type Result struct {
	Timestamp    uint64                      `json:"timestamp"`
	L1Inclusion  eth.BlockID                 `json:"l1Inclusion"`
	L2Heads      map[eth.ChainID]eth.BlockID `json:"l2Heads"`
	InvalidHeads map[eth.ChainID]InvalidHead `json:"invalidHeads"`
	// NotReady holds the chains whose block at this timestamp could not be decided yet,
	// because an executing message in it references an initiating block the source chain's
	// LogsDB has not reached. It is never persisted: a round holding one of these waits, and
	// only advance/invalidate transitions reach the WAL.
	NotReady map[eth.ChainID]eth.BlockID `json:"notReady,omitempty"`
}

// PendingTransition is the generic write-ahead-log entry for an effectful
// interop decision. Recovery and steady-state both use the same apply path.
//
// Phase 2 keeps this intentionally small:
// - advance/invalidate carry their Result directly
// - rewind carries the accepted frontier to rewind from
// Later phases can expand this into a richer explicit transition plan.
//
// InvalidationParentPayloads is populated only for DecisionInvalidate transitions. It carries
// the canonical parent payload (height-1) for each invalidated chain, captured at build time
// and used at apply time to drive the rewind. Storing the full payload means a crash mid-rewind
// can be recovered without consulting the live EL — the canonical block may already have been
// pruned by the EL once it left the canonical chain.
type PendingTransition struct {
	Decision                   Decision                                      `json:"decision"`
	Result                     *Result                                       `json:"result,omitempty"`
	Rewind                     *RewindPlan                                   `json:"rewind,omitempty"`
	InvalidationParentPayloads map[eth.ChainID]*eth.ExecutionPayloadEnvelope `json:"invalidationParentPayloads,omitempty"`
}

// RewindPlan is the explicit rewind transition persisted in the WAL.
// It captures the target verified frontier and engine reset decision so recovery
// can apply the same rewind path without recomputing it from live state.
//
// TargetPayloads carries the canonical payload at each chain's rewind target, captured at
// build time. The apply path uses it to drive the engine rewind without consulting the live
// EL — see the InvalidationParentPayloads comment on PendingTransition for the rationale.
type RewindPlan struct {
	RewindAtOrAfter  uint64                                        `json:"rewindAtOrAfter"`
	ResetAllChainsTo *uint64                                       `json:"resetAllChainsTo,omitempty"`
	TargetHeads      map[eth.ChainID]eth.BlockID                   `json:"targetHeads,omitempty"`
	TargetPayloads   map[eth.ChainID]*eth.ExecutionPayloadEnvelope `json:"targetPayloads,omitempty"`
}

func (r *Result) IsValid() bool {
	return len(r.InvalidHeads) == 0
}

// IsReady reports whether every chain's block at this timestamp got a verdict. A result can be
// valid (nothing is known to be invalid) and still not ready (something is not yet knowable);
// only a result that is both may advance the verified frontier.
func (r *Result) IsReady() bool {
	return len(r.NotReady) == 0
}

func (r *Result) IsEmpty() bool {
	return r.L1Inclusion == (eth.BlockID{}) && len(r.L2Heads) == 0 &&
		len(r.InvalidHeads) == 0 && len(r.NotReady) == 0
}

func (r *Result) ToVerifiedResult() VerifiedResult {
	return VerifiedResult{
		Timestamp:   r.Timestamp,
		L1Inclusion: r.L1Inclusion,
		L2Heads:     r.L2Heads,
	}
}
