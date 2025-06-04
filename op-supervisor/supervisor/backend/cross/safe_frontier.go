package cross

import (
	"errors"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

type SafeFrontierCheckDeps interface {
	CandidateCrossSafe(chain eth.ChainID) (candidate types.DerivedBlockRefPair, err error)

	CrossDerivedToSource(chainID eth.ChainID, derived eth.BlockID) (source types.BlockSeal, err error)
}

// HazardSafeFrontierChecks verifies all the hazard blocks are either:
//   - already cross-safe.
//   - the first (if not first: local blocks to verify before proceeding)
//     local-safe block, after the cross-safe block.
func HazardSafeFrontierChecks(d SafeFrontierCheckDeps, inL1Source eth.BlockID, hazards *HazardSet) error {
	for hazardChainID, hazardBlock := range hazards.Entries() {
		initSource, err := d.CrossDerivedToSource(hazardChainID, hazardBlock.ID())
		if err != nil {
			if errors.Is(err, types.ErrFuture) {
				// If not in cross-safe scope, then check if it's the candidate cross-safe block.
				candidate, err := d.CandidateCrossSafe(hazardChainID)
				if err != nil {
					return fmt.Errorf("failed to determine cross-safe candidate block of hazard dependency %s (chain %s): %w", hazardBlock, hazardChainID, err)
				}
				if candidate.Derived.Number == hazardBlock.Number && candidate.Derived.ID() != hazardBlock.ID() {
					return fmt.Errorf("expected block %s (chain %s) does not match candidate local-safe block %s: %w",
						hazardBlock, hazardChainID, candidate.Derived, types.ErrConflict)
				}
				if candidate.Source.Number > inL1Source.Number {
					return fmt.Errorf("local-safe hazard block %s derived from L1 block %s is after scope %s: %w",
						hazardBlock.ID(), initSource, inL1Source, types.ErrOutOfScope)
				}
			} else {
				return fmt.Errorf("failed to determine cross-derived of hazard block %s (chain %s): %w", hazardBlock, hazardChainID, err)
			}
		} else if initSource.Number > inL1Source.Number {
			return fmt.Errorf("cross-safe hazard block %s derived from L1 block %s is after scope %s: %w",
				hazardBlock.ID(), initSource, inL1Source, types.ErrOutOfScope)
		}
	}
	return nil
}
