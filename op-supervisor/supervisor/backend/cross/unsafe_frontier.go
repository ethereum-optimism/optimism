package cross

import (
	"errors"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db/entrydb"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

type UnsafeFrontierCheckDeps interface {
	ParentBlock(chainID types.ChainID, parentOf eth.BlockID) (parent eth.BlockID, err error)

	IsCrossUnsafe(chainID types.ChainID, block eth.BlockID) error
	IsLocalUnsafe(chainID types.ChainID, block eth.BlockID) error
}

// HazardUnsafeFrontierChecks verifies all the hazard blocks are either:
//   - already cross-unsafe.
//   - the first (if not first: local blocks to verify before proceeding)
//     local-unsafe block, after the cross-unsafe block.
func HazardUnsafeFrontierChecks(d UnsafeFrontierCheckDeps, inL1DerivedFrom eth.BlockID, hazards map[types.ChainIndex]types.BlockSeal) error {
	for hazardChainIndex, hazardBlock := range hazards {
		hazardChainID, err := types.ChainIDFromIndex(hazardChainIndex)
		if err != nil {
			return err
		}
		err = d.IsCrossUnsafe(hazardChainID, hazardBlock.ID())
		if err != nil {
			if errors.Is(err, entrydb.ErrFuture) {
				err = d.IsLocalUnsafe(hazardChainID, hazardBlock.ID())
				if err != nil {

				}
				// If it doesn't have a parent block, then there is no prior block required to be cross-safe
				if hazardBlock.Number > 0 {
					// Check that parent of hazardBlockID is cross-safe within view
					parent, err := d.ParentBlock(hazardChainID, hazardBlock.ID())
					if err != nil {

					}
					if err := d.IsCrossUnsafe(hazardChainID, hazardBlock.ID()); err != nil {

					}
				}
			} else {
				return fmt.Errorf("failed to determine cross-derived of hazard block %s (chain %s): %w", hazardBlock, hazardChainID, err)
			}
		}
	}
	return nil
}
