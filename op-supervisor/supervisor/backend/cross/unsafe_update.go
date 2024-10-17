package cross

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

type CrossUnsafeDeps interface {
	CrossUnsafe(chainID types.ChainID) (types.BlockSeal, error)

	//CycleCheckDeps
}

func CrossUnsafeUpdate(chainID types.ChainID, d CrossUnsafeDeps) error {
	// fetch cross-head
	crossSafe, err := d.CrossUnsafe(chainID)
	if err != nil {
		// TODO handle genesis case
	}

	// open block N+1
	candidate, _, execMsgs, err := d.OpenBlock(chainID, crossSafe.Number+1)
	if err != nil {
		return fmt.Errorf("failed to open block %d: %w", crossSafe.Number+1, err)
	}

	hazards, err := CrossUnsafeHazards(d, chainID, candidate, execMsgs)
	if err != nil {
		// TODO
	}
	if err := HazardUnsafeFrontierChecks(d, hazards); err != nil {
		// TODO
	}
	//if err := HazardCycleChecks(d, candidate.Timestamp, hazards); err != nil {
	//// TODO
	//}
	// TODO promote the candidate block to cross-unsafe
	return nil
}
