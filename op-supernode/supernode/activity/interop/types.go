package interop

import (
	"sort"

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

// ChainObservation captures everything observed for a single chain at a
// verified timestamp: the L2 head, the L1 block at which that head became safe
// (the per-chain L1 inclusion), and the OutputV0 preimage fields needed to
// reconstruct the optimistic output without querying live engine state.
//
// BlockHash is intentionally omitted — it is L2Head.Hash. Storing it again
// would duplicate the canonical block identifier.
type ChainObservation struct {
	L2Head                   eth.BlockID `json:"l2Head"`
	L1Head                   eth.BlockID `json:"l1Head"`
	StateRoot                eth.Bytes32 `json:"stateRoot"`
	MessagePasserStorageRoot eth.Bytes32 `json:"messagePasserStorageRoot"`
}

// OutputV0 reconstructs the full OutputV0 preimage for this chain.
func (o ChainObservation) OutputV0() eth.OutputV0 {
	return eth.OutputV0{
		StateRoot:                o.StateRoot,
		MessagePasserStorageRoot: o.MessagePasserStorageRoot,
		BlockHash:                o.L2Head.Hash,
	}
}

// OutputRoot returns the per-chain output root hash.
func (o ChainObservation) OutputRoot() eth.Bytes32 {
	v0 := o.OutputV0()
	return eth.OutputRoot(&v0)
}

// VerifiedRecord is the durable verifier snapshot at a timestamp.
// It stores only source facts; aggregates and RPC views are derived on demand.
//
// Per-chain observations snapshot the optimistic data that produced this
// verified record, so superroot_atTimestamp can serve OptimisticAtTimestamp
// from the durable record without re-reading live SafeDB or engine state.
// Without this snapshot, a chain rewind between verification and serving
// could drop derivable optimistic blocks from the response.
type VerifiedRecord struct {
	Timestamp uint64                            `json:"timestamp"`
	Chains    map[eth.ChainID]ChainObservation  `json:"chains"`
	CurrentL1 eth.BlockID                       `json:"currentL1"`
}

// L2Heads returns the per-chain L2 head map for compatibility with consumers
// that work with the legacy VerifiedResult shape.
func (r VerifiedRecord) L2Heads() map[eth.ChainID]eth.BlockID {
	if r.Chains == nil {
		return nil
	}
	out := make(map[eth.ChainID]eth.BlockID, len(r.Chains))
	for id, obs := range r.Chains {
		out[id] = obs.L2Head
	}
	return out
}

// L1Inclusion returns the aggregate L1 block at which this superroot is
// derivable: the maximum per-chain L1 head. Computed on demand from Chains.
func (r VerifiedRecord) L1Inclusion() eth.BlockID {
	var max eth.BlockID
	for _, obs := range r.Chains {
		if obs.L1Head.Number > max.Number {
			max = obs.L1Head
		}
	}
	return max
}

// SortedChainIDs returns the chain IDs sorted ascending — the canonical order
// used by SuperV1.
func (r VerifiedRecord) SortedChainIDs() []eth.ChainID {
	ids := make([]eth.ChainID, 0, len(r.Chains))
	for id := range r.Chains {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i].Cmp(ids[j]) < 0 })
	return ids
}

func (r VerifiedRecord) ToVerifiedResult() VerifiedResult {
	return VerifiedResult{
		Timestamp:   r.Timestamp,
		L1Inclusion: r.L1Inclusion(),
		L2Heads:     r.L2Heads(),
	}
}

func (r VerifiedRecord) ResponseData() *eth.SuperRootResponseData {
	chainOutputs := make([]eth.ChainIDAndOutput, 0, len(r.Chains))
	for _, id := range r.SortedChainIDs() {
		chainOutputs = append(chainOutputs, eth.ChainIDAndOutput{
			ChainID: id,
			Output:  r.Chains[id].OutputRoot(),
		})
	}
	super := eth.NewSuperV1(r.Timestamp, chainOutputs...)
	return &eth.SuperRootResponseData{
		VerifiedRequiredL1: r.L1Inclusion(),
		Super:              super,
		SuperRoot:          eth.SuperRoot(super),
	}
}

// InvalidHead pairs a block identifier with the output preimage fields needed
// for optimistic root computation in the superroot API. The full OutputV0 can
// be reconstructed on demand via OutputV0() since BlockHash is already in BlockID.
//
// L1Inclusion is the per-chain L1 block at which the invalid optimistic block
// became safe; persisted alongside the deny record so OptimisticAtTimestamp can
// be served from durable state.
type InvalidHead struct {
	eth.BlockID
	StateRoot                eth.Bytes32 `json:"stateRoot"`
	MessagePasserStorageRoot eth.Bytes32 `json:"messagePasserStorageRoot"`
	L1Inclusion              eth.BlockID `json:"l1Inclusion"`
}

// Result represents the result of interop validation at a specific timestamp given current data.
// it contains all the same information as VerifiedResult, but also contains a list of invalid heads.
//
// L1Heads is the per-chain L1 inclusion snapshot captured atomically with
// L2Heads in observeRound. It is plumbed through verification so that
// VerifiedRecord and InvalidHead can record the per-chain L1 inclusion at the
// observation moment.
type Result struct {
	Timestamp    uint64                      `json:"timestamp"`
	L1Inclusion  eth.BlockID                 `json:"l1Inclusion"`
	L2Heads      map[eth.ChainID]eth.BlockID `json:"l2Heads"`
	L1Heads      map[eth.ChainID]eth.BlockID `json:"l1Heads"`
	InvalidHeads map[eth.ChainID]InvalidHead `json:"invalidHeads"`
}

// PendingTransition is the generic write-ahead-log entry for an effectful
// interop decision. Recovery and steady-state both use the same apply path.
//
// DecisionAdvance uses Verified, DecisionInvalidate uses Result, and DecisionRewind uses Rewind.
type PendingTransition struct {
	Decision Decision        `json:"decision"`
	Result   *Result         `json:"result,omitempty"`
	Verified *VerifiedRecord `json:"verified,omitempty"`
	Rewind   *RewindPlan     `json:"rewind,omitempty"`
}

// RewindPlan is the explicit rewind transition persisted in the WAL.
// It captures the target verified frontier and engine reset decision so recovery
// can apply the same rewind path without recomputing it from live state.
type RewindPlan struct {
	RewindAtOrAfter  uint64                      `json:"rewindAtOrAfter"`
	ResetAllChainsTo *uint64                     `json:"resetAllChainsTo,omitempty"`
	TargetHeads      map[eth.ChainID]eth.BlockID `json:"targetHeads,omitempty"`
}

func (r *Result) IsValid() bool {
	return len(r.InvalidHeads) == 0
}

func (r *Result) IsEmpty() bool {
	return r.L1Inclusion == (eth.BlockID{}) && len(r.L2Heads) == 0 && len(r.InvalidHeads) == 0
}

func (r *Result) ToVerifiedResult() VerifiedResult {
	return VerifiedResult{
		Timestamp:   r.Timestamp,
		L1Inclusion: r.L1Inclusion,
		L2Heads:     r.L2Heads,
	}
}
