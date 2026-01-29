package interop

import (
	"errors"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// VerifiedResult represents the verified state at a specific timestamp.
// It contains the L1 head from which the L2 heads were derived,
// and a map of each chain's L2 head at that timestamp.
type VerifiedResult struct {
	Timestamp uint64                      `json:"timestamp"`
	L1Head    eth.BlockID                 `json:"l1Head"`
	L2Heads   map[eth.ChainID]eth.BlockID `json:"l2Heads"`
}

// Result represents the result of interop validation at a specific timestamp given current data.
// it contains all the same information as VerifiedResult, but also contains a list of invalid heads.
type Result struct {
	Timestamp    uint64                      `json:"timestamp"`
	L1Head       eth.BlockID                 `json:"l1Head"`
	L2Heads      map[eth.ChainID]eth.BlockID `json:"l2Heads"`
	InvalidHeads map[eth.ChainID]eth.BlockID `json:"invalidHeads"`
}

func (r *Result) IsValid() bool {
	return len(r.InvalidHeads) == 0
}

func (r *Result) IsEmpty() bool {
	return r.L1Head == (eth.BlockID{}) && len(r.L2Heads) == 0 && len(r.InvalidHeads) == 0
}

var ErrInvalidResult = errors.New("result is invalid")

func (r *Result) ToVerifiedResult() VerifiedResult {
	return VerifiedResult{
		Timestamp: r.Timestamp,
		L1Head:    r.L1Head,
		L2Heads:   r.L2Heads,
	}
}
