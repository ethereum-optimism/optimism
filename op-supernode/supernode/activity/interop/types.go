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

// Result represents the result of interop validation at a specific timestamp given current data.
// it contains all the same information as VerifiedResult, but also contains a list of invalid heads.
type Result struct {
	Timestamp    uint64                      `json:"timestamp"`
	L1Inclusion  eth.BlockID                 `json:"l1Inclusion"`
	L2Heads      map[eth.ChainID]eth.BlockID `json:"l2Heads"`
	InvalidHeads map[eth.ChainID]eth.BlockID `json:"invalidHeads"`
}

// PendingTransition is the generic write-ahead-log entry for an effectful
// interop decision. Recovery and steady-state both use the same apply path.
//
// Phase 2 keeps this intentionally small:
// - advance/invalidate carry their Result directly
// - rewind carries the accepted timestamp to rewind from
// Later phases can expand this into a richer explicit transition plan.
type PendingTransition struct {
	Decision Decision    `json:"decision"`
	Result   *Result     `json:"result,omitempty"`
	Rewind   *RewindPlan `json:"rewind,omitempty"`
}

// RewindPlan is the explicit rewind transition persisted in the WAL.
// It captures the target verified frontier and per-chain logsDB target heads so
// recovery can apply the same rewind path without recomputing it from live
// state.
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
