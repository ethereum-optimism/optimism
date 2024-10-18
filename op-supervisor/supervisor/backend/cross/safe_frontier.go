package cross

import (
	"errors"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/depset"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

type SafeFrontierCheckDeps interface {
	ParentBlock(chainID types.ChainID, parentOf eth.BlockID) (parent eth.BlockID, err error)

	CrossDerivedFrom(chainID types.ChainID, derived eth.BlockID) (derivedFrom eth.BlockID, err error)
	LocalDerivedFrom(chainID types.ChainID, derived eth.BlockID) (derivedFrom eth.BlockID, err error)

	DependencySet() depset.DependencySet
}

// HazardSafeFrontierChecks verifies all the hazard blocks are either:
//   - already cross-safe.
//   - the first (if not first: local blocks to verify before proceeding)
//     local-safe block, after the cross-safe block.
func HazardSafeFrontierChecks(d SafeFrontierCheckDeps, inL1DerivedFrom eth.BlockID, hazards map[types.ChainIndex]types.BlockSeal) error {
	depSet := d.DependencySet()
	for hazardChainIndex, hazardBlock := range hazards {
		hazardChainID, err := depSet.ChainIDFromIndex(hazardChainIndex)
		if err != nil {
			// TODO: translate unknown chain -> conflict error
			return err
		}
		initDerivedFrom, err := d.CrossDerivedFrom(hazardChainID, hazardBlock.ID())
		if err != nil {
			if errors.Is(err, types.ErrFuture) {
				initDerivedFrom, err = d.LocalDerivedFrom(hazardChainID, hazardBlock.ID())
				if err != nil {
					return fmt.Errorf("hazard block %s (chain %d) is not local-safe: %w", hazardBlock, hazardChainID, err)
				}
				// If it doesn't have a parent block, then there is no prior block required to be cross-safe
				if hazardBlock.Number > 0 {
					// Check that parent of hazardBlockID is cross-safe within view
					parent, err := d.ParentBlock(hazardChainID, hazardBlock.ID())
					if err != nil {
						return fmt.Errorf("failed to retrieve parent-block of hazard block %s (chain %s): %w", hazardBlock, hazardChainID, err)
					}
					initDerivedFrom, err := d.CrossDerivedFrom(hazardChainID, parent)
					if err != nil {
						return fmt.Errorf("cannot rely on hazard-block %s (chain %s), parent block %s is not cross-unsafe: %w", hazardBlock, hazardChainID, parent, err)
					}
					if initDerivedFrom.Number > inL1DerivedFrom.Number {
						return fmt.Errorf("local-safe hazard block %s derived from L1 block %s is after scope %s: %w",
							hazardBlock.ID(), initDerivedFrom, inL1DerivedFrom, types.ErrOutOfScope)
					}
				}
			} else {
				return fmt.Errorf("failed to determine cross-derived of hazard block %s (chain %s): %w", hazardBlock, hazardChainID, err)
			}
		} else if initDerivedFrom.Number > inL1DerivedFrom.Number {
			return fmt.Errorf("cross-safe hazard block %s derived from L1 block %s is after scope %s: %w",
				hazardBlock.ID(), initDerivedFrom, inL1DerivedFrom, types.ErrOutOfScope)
		}
	}
	return nil
}
