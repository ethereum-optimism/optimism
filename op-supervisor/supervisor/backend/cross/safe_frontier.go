package cross

import (
	"errors"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db/entrydb"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

var (
	ErrLocalDerivedFrom = errors.New("failed to get local derived from")
	ErrParentBlock      = errors.New("failed to get parent block")
	ErrCrossDerivedFrom = errors.New("failed to get cross derived from")
	ErrOutOfScope       = errors.New("block is out of scope")
)

type SafeFrontierCheckDeps interface {
	ParentBlock(chainID types.ChainID, parentOf eth.BlockID) (parent eth.BlockID, err error)

	CrossDerivedFrom(chainID types.ChainID, derived eth.BlockID) (derivedFrom eth.BlockID, err error)
	LocalDerivedFrom(chainID types.ChainID, derived eth.BlockID) (derivedFrom eth.BlockID, err error)
}

// HazardSafeFrontierChecks verifies all the hazard blocks are either:
//   - already cross-safe.
//   - the first (if not first: local blocks to verify before proceeding)
//     local-safe block, after the cross-safe block.
func HazardSafeFrontierChecks(d SafeFrontierCheckDeps, inL1DerivedFrom eth.BlockID, hazards map[types.ChainIndex]types.BlockSeal) error {
	for hazardChainIndex, hazardBlock := range hazards {
		// TODO(#11105): translate chain index to chain ID
		hazardChainID := types.ChainIDFromUInt64(uint64(hazardChainIndex))
		initDerivedFrom, err := d.CrossDerivedFrom(hazardChainID, hazardBlock.ID())
		if err != nil {
			if errors.Is(err, entrydb.ErrFuture) {
				initDerivedFrom, err = d.LocalDerivedFrom(hazardChainID, hazardBlock.ID())
				if err != nil {
					return fmt.Errorf("%w for chain %s: %v", ErrLocalDerivedFrom, hazardChainID, err)
				}
				// If it doesn't have a parent block, then there is no prior block required to be cross-safe
				if hazardBlock.Number > 0 {
					// Check that parent of hazardBlockID is cross-safe within view
					parent, err := d.ParentBlock(hazardChainID, hazardBlock.ID())
					if err != nil {
						return fmt.Errorf("%w for chain %s: %v", ErrParentBlock, hazardChainID, err)
					}
					initDerivedFrom, err := d.CrossDerivedFrom(hazardChainID, hazardBlock.ID())
					if err != nil {
						return fmt.Errorf("%w for chain %s: %v", ErrCrossDerivedFrom, hazardChainID, err)
					}
					if initDerivedFrom.Number > inL1DerivedFrom.Number {
						return fmt.Errorf("%w: hazard block %s derived from L1 block %s is after scope %s",
							ErrOutOfScope, hazardBlock.ID(), initDerivedFrom, inL1DerivedFrom)
					}
				}
			} else {
				return fmt.Errorf("failed to determine cross-derived of hazard block %s (chain %s): %w", hazardBlock, hazardChainID, err)
			}
		}
		if initDerivedFrom.Number > inL1DerivedFrom.Number {
			return fmt.Errorf("%w: hazard block %s derived from L1 block %s is after scope %s",
				ErrOutOfScope, hazardBlock.ID(), initDerivedFrom, inL1DerivedFrom)
		}
	}
	return nil
}
