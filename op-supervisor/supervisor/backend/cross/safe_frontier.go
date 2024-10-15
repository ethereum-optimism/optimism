package cross

import (
	"errors"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db/entrydb"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
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

				}
				// If it doesn't have a parent block, then there is no prior block required to be cross-safe
				if hazardBlock.Number > 0 {
					// Check that parent of hazardBlockID is cross-safe within view
					parent, err := d.ParentBlock(hazardChainID, hazardBlock.ID())
					if err != nil {

					}
					initDerivedFrom, err := d.CrossDerivedFrom(hazardChainID, hazardBlock.ID())
					if err != nil {

					}
					if initDerivedFrom.Number > inL1DerivedFrom.Number {
						// TODO outside of scope (if cross is not in scope, local is not going to be either)
					}
				}

			} else {
				return fmt.Errorf("failed to determine cross-derived of hazard block %s (chain %s): %w", hazardBlock, hazardChainID, err)
			}
		}
		if initDerivedFrom.Number > inL1DerivedFrom.Number {
			// TODO outside of scope (if cross is not in scope, local is not going to be either)
		}
	}
	return nil
}
